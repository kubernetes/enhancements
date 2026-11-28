# KEP-6313: Network-Isolated Pod Sandboxes

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: DRA-provisioned accelerator fabric](#story-1-dra-provisioned-accelerator-fabric)
    - [Story 2: Disconnected batch job](#story-2-disconnected-batch-job)
    - [Story 3: VM sandboxes and device passthrough](#story-3-vm-sandboxes-and-device-passthrough)
    - [Story 4: Local-only multi-container pods](#story-4-local-only-multi-container-pods)
    - [Story 5: Agentic orchestrators](#story-5-agentic-orchestrators)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Validation](#validation)
  - [Updates from clients that do not know the field](#updates-from-clients-that-do-not-know-the-field)
  - [Disabled feature gate](#disabled-feature-gate)
  - [CRI Integration](#cri-integration)
  - [Kubelet Behavior](#kubelet-behavior)
  - [Control Plane Behavior](#control-plane-behavior)
  - [Interaction with Core Networking APIs](#interaction-with-core-networking-apis)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) [Enhancement issue](https://github.com/kubernetes/enhancements/issues/6313) in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP adds **network-isolated Pods**: Pods that opt out of the default pod
network. An isolated pod gets a network namespace with only a loopback
interface and is not reachable from the cluster unless other network
connectivity is added to it somehow.

## Motivation

Kubernetes was built on the early assumption that every Pod is attached to a
single flat cluster network and is routable from every other Pod. This
assumption has always generated friction with workloads that need advanced
routing, multi-tenancy or meshes. Those workloads get around the "default
network" with out-of-band mechanisms: per-node runtime configuration, custom
runtime classes, chained plugins, or annotations.

SIG Network has already tried to solve this inside the project. The
[Multi-Network effort (KEP-3698)] concluded it was too complex: problems like
network probes or DNS across multiple networks had no clear solution, portability
depended on very specific (often private) plugin implementations, and the
user stories were fragmented — some wanted multiple NICs, others VPC-like
networks — each backed by a relatively small number of users. A follow-up
proposal to simplify the low level API ([KEP-4410]) required offloading the
network to another component that will make it incompatible with existing
deployments and hard to roll out.

What has worked is composing on top of stable core primitives instead of
extending the core network model:

- [Kubernetes Network Drivers][knd-paper] defined a new architecture, based
  on DRA, to provide a native mechanism to plumb additional network devices.
  [DRANET] is a project that demonstrated the successful use of this
  architecture to solve the challenges of the new AI/ML workloads and RDMA
  networks.
- [KEP-4817] (GA) added a standard way to report network data in DRA. All
  drivers report their network devices with a common language, and
  integrators can use that data to build network integrations out of core
  (for example, their own EndpointSlice controllers), keeping core stable.
- [NRI] showed how network applications can integrate directly with the
  runtime. This removes the limitations of the CNI model, which composes
  plugins only through serial Unix-style chaining and file system based configuration
  that breaks the separation of concerns (see the [service mesh CNI discussion]).

This friction is more critical these days. New workloads, like the agentic
ones, require strict isolation, and their density and scale make the traditional
out-of-band mechanisms obsolete. At the same time, DRA network drivers, VM sandboxes
that provide their own networking (e.g. Kata/Firecracker/... with VFIO passthrough),
disconnected jobs, ... have been waiting for a solution for years.

One of the missing pieces is a core primitive to NOT attach the default pod
network. Decomposing the Pod network attachment solves this problem for both
maintainers and integrators: integrators get an isolated sandbox they can
build on with the mechanisms above, and the project stops carrying the toil
of workarounds and requirements that the core network model was never meant
to express.

[Multi-Network effort (KEP-3698)]: https://github.com/kubernetes/enhancements/issues/3698
[KEP-4410]: https://github.com/kubernetes/enhancements/issues/4410
[knd-paper]: https://ieeexplore.ieee.org/document/11146291/
[DRANET]: https://github.com/google/dranet
[KEP-4817]: https://github.com/kubernetes/enhancements/issues/4817
[NRI]: https://github.com/containerd/nri
[service mesh CNI discussion]: https://github.com/kubernetes/kubernetes/issues/130594

### Goals

- Allow users to declare that a Pod must not be attached to the default pod
  network.
- Leave `hostNetwork` and every existing pod unchanged: the new field is
  optional and never defaulted, and clients and controllers that do not know
  it keep working.
- Fully define the semantics of isolated pods and fail predictably: reject
  configurations that cannot work without the pod network (network probes,
  host ports) at admission time instead of failing at runtime, and never
  silently run a pod that requested isolation attached to the pod network
  (e.g., on version-skewed nodes that do not support this feature).
- Keep isolated pods out of service discovery: Services, Endpoints and
  EndpointSlices.
- Define the contract between the kubelet and the container runtimes for
  isolated pods.

### Non-Goals

- Designing a multi-network or secondary-interface attachment framework.
  `podNetwork: Isolated` is intentionally about the absence of automatic attachment;
  it provides a clean surface for delegation (e.g., DRA drivers), but how
  interfaces are attached afterwards and its lifecycle is out of scope.
- Adding other networks to the API. Today `status.podIPs`, Services,
  Endpoints and EndpointSlices, NetworkPolicy and cluster DNS only work with
  the default pod network, and this KEP does not change that (see
  [Interaction with Core Networking APIs](#interaction-with-core-networking-apis)).
  New `podNetwork` values for other networks, or changes to those APIs
  to support other networks, need a future KEP.
- Isolating anything other than networking. Storage, IPC, PID, devices and
  host access are governed by existing fields.
- Deprecating, replacing or mirroring `hostNetwork`. It is a GA v1 field and
  remains the only way to request the host network; `podNetwork` may not be
  set together with it.
- Changing how kube-proxy, NetworkPolicy or DNS work for non-isolated pods.
- Changing the defaults of `dnsPolicy` or `enableServiceLinks`, or the
  service environment variables, for isolated pods. They are configured with
  their own fields, exactly as for any other pod.
- Skipping service account token mounting for isolated pods. Tokens may still
  be consumed by non-network means.

## Proposal

Add a new optional enum field `podNetwork` to `PodSpec` with the values
`Default` and `Isolated`. The field is never defaulted: when it is unset, a
pod behaves as if it was `Default`, which is what every existing
non-`hostNetwork` pod does today. The field must be unset when
`hostNetwork: true`. `Isolated` is the new behavior:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: network-isolated-example
spec:
  podNetwork: Isolated # new field; unset behaves as Default
  containers:
  - name: worker
    image: registry.k8s.io/e2e-test-images/agnhost:2.53
    command: ["sleep", "3600"]
```

Semantics of `podNetwork: Isolated`:

1. Configurations that need to reach the pod from the node (`hostPort`, and
   `httpGet`, `tcpSocket` and `grpc` probes and lifecycle handlers) are
   rejected by validation (see [Validation](#validation)).
2. The kubelet uses a new CRI field to tell the runtime to create the sandbox
   network namespace with only the loopback interface and to skip the
   default-pod-network plumbing (e.g., runtimes that normally invoke CNI
   plugins to attach the sandbox to the default pod network would skip that
   step).
3. The pod runs with empty `status.podIP`/`status.podIPs` and becomes `Ready`
   without waiting for an IP. `status.hostIP` is still reported. The API
   server rejects a status update that reports a pod IP for an isolated
   pod. The kubelet starts the pod even when the runtime reports that the
   pod network is not ready, as it does for `hostNetwork` pods.
4. The endpoints and endpointslice controllers never select the pod into
   Endpoints/EndpointSlices, even if a Service selector matches it.
5. DNS configuration and service environment variables are generated as for
   any other pod: `dnsPolicy` defaults to `ClusterFirst` and
   `enableServiceLinks` to `true`. They identify resolvers and Service IPs that
   the pod can only reach when an external integration (a DRA driver, VM
   networking) provides connectivity by other means. A pod that wants no
   network dependency sets `dnsPolicy: None` with a `dnsConfig` (for
   example, a resolver on loopback) and `enableServiceLinks: false`.
6. The kubelet maps the pod hostname to `127.0.0.1` and `::1` in the
   managed `/etc/hosts`, so `localhost` and `$(hostname)` resolve inside
   the sandbox. `hostAliases` are written as usual.
7. If the kubelet or the container runtime does not support isolated
   sandboxes, the kubelet rejects the pod during node-level admission (after
   scheduling, the pod is marked `Failed` with reason `PodFeatureUnsupported`)
   instead of silently attaching it to the default pod network (**fail
   closed**). This is the existing [Node Declared Features (KEP-5328)]
   admission check; see [Kubelet Behavior](#kubelet-behavior).
8. NetworkPolicy does not apply to these pods, as with `hostNetwork` pods
   today: implementations enforce policy on pod network attachments and pod
   IPs, and an isolated pod has neither. Interfaces attached later by
   delegated mechanisms are outside the core network model and therefore
   outside NetworkPolicy scope. The full contract is defined in
   [Interaction with Core Networking APIs](#interaction-with-core-networking-apis).

`podNetwork: Default` and an unset field do not change any behavior: the pod
gets its own network namespace and is connected to the default pod network.
`hostNetwork: true` pods are not affected either; they cannot set the field.

### User Stories

#### Story 1: DRA-provisioned accelerator fabric

As an operator of an AI training cluster, I allocate RDMA/accelerator NICs to
pods through DRA drivers. I want the pod sandbox to start empty: no
interfaces, routes or iptables rules from the default pod network plugin. The DRA
driver is then the only owner of the pod connectivity. The default attachment
is pure overhead for these pods: they communicate exclusively over the fabric
interfaces, the default interface and its routes can conflict with the routing
configured by the driver, and every sandbox pays IP allocation and network
plumbing latency for a network it never uses. Today I have to coordinate
teardown and priority with the CNI configuration on every node.

#### Story 2: Disconnected batch job

As a security engineer, I run jobs that process sensitive data mounted from
volumes, and I must prove to auditors that the workload has no network path
in or out. A deny-all NetworkPolicy depends on the network plugin enforcing
it, and the pod still gets a pod IP. With `podNetwork: Isolated` there is
nothing to enforce: the only interface is loopback. The guarantee is about
provisioning, not a runtime invariant: the sandbox starts with only loopback
and `podNetwork` is immutable on Pods, so changing the pod's connectivity afterwards
requires a privileged node-level actor (the workload itself cannot attach
interfaces without host-level privileges). Defending against privileged
actors is out of scope.

#### Story 3: VM sandboxes and device passthrough

As a confidential-computing platform owner using VM sandboxes, networking
is provided inside the VM (e.g., via VFIO passthrough of a physical function).
The node-level network plumbing is useless and potentially harmful. I want
the runtime to skip it entirely, driven by a core API field rather
than a runtime-specific annotation. Unlike Story 2, isolation is not the
goal here: the platform itself provides the networking, and the empty
sandbox is simply the correct starting state.

#### Story 4: Local-only multi-container pods

As a CI system author, I run pods whose containers only talk over `localhost`
(hermetic builds and tests). I want to guarantee that no external network
dependency can leak into the build, and I do not want these pods to consume
cluster IP space or appear in service discovery.

#### Story 5: Agentic orchestrators

As an operator of an agent orchestrator (e.g. Substrate), I run untrusted,
often machine-generated code in sandboxed pods at very high density and
churn. Each agent must start with zero network access by default; the
orchestrator then grants connectivity explicitly through its own channels
(a local proxy over `localhost`, or interfaces injected by a driver). I
cannot depend on NetworkPolicy enforcement for this guarantee, and at this
scale the per-pod network plumbing and IPAM are too much overhead: the pods never
use the default pod network, but they still consume IP space, endpoints
processing, and setup/teardown time on every sandbox.

### Notes/Constraints/Caveats

**Naming.** This was extensively discussed during the initial reviews
([kubernetes/kubernetes#141604]):

- "Hermetic" was the original working name of the feature. It is not used in
  the API, the KEP title or this document: reviewers pointed out that
  "hermetic" implies storage, host access or compute isolation that this KEP
  does not provide.
- "CNI" never appears in the API. CNI is an implementation detail of Linux
  container runtimes; it does not exist in the Kubernetes API vocabulary.
- `podNetwork` is the name of the field. Kubernetes has one network that
  every pod is connected to unless it opts out, and all the core networking
  APIs (Services, NetworkPolicy, `status.podIPs`, ...) only work with that
  network. The API calls it "the pod network"; this document also calls it
  "the default pod network". The field says whether the pod is attached to
  it.
- `Default` is the current behavior of a non-`hostNetwork` pod: the pod gets
  its own network namespace and is attached to the pod network.
- `Isolated` describes the sandbox when it is created: a network namespace
  with only loopback, not attached to the pod network. It is not a
  permanent prohibition: interfaces can still be attached afterwards by
  delegated mechanisms (see [Non-Goals](#non-goals)).
- The field must be unset for `hostNetwork` pods: it describes the pod's own
  network namespace, and a `hostNetwork` pod has none (see
  [Validation](#validation)).

**The pod still has a network namespace.** The pod keeps loopback
(`127.0.0.1`, `::1`) and the containers in the pod communicate over
`localhost`. This is why names like `noNetwork` were rejected: they are
technically inaccurate.

**Kubernetes API access.** An isolated pod cannot necessarily reach the
kube-apiserver (or anything else). The documentation in [services-networking concepts] assumes
that every pod can reach the API and other pods. Those documents must be
updated to describe `podNetwork: Isolated` as an explicit, opt-in exception.
Pods that do not set the field keep all the documented guarantees.

**Downward API.** `status.podIP`/`status.podIPs` resolve to empty values for
isolated pods. `status.hostIP` remains populated.

**Secondary networks attached through CNI delegation.** Some projects (for
example [Multus], and [KubeVirt] through it) attach additional interfaces by
intercepting the CNI invocation that the runtime performs for the default pod
network. That invocation is an implementation detail of the runtime, and it
does not happen for `podNetwork: Isolated` sandboxes, so today those projects
are not invoked for isolated pods. The semantics of `Isolated` are "not attached
to the default pod network", not "no secondary interfaces": a runtime that
knows how to attach secondary interfaces without attaching the default pod
network is free to do so. Pods that do not set `podNetwork: Isolated` are
not affected in any way, and existing integrations keep working unchanged
for them. For isolated pods, the paths that do not depend on the
default-pod-network CNI invocation are DRA network drivers ([DRANET]) and
runtime hooks ([NRI], OCI hooks through CDI); [aojea/network-device-plugin]
demonstrates the pattern. Not breaking these downstream consumers and
providing them a compatibility path to isolated pods is a
[Beta](#beta) graduation requirement.

[Multus]: https://github.com/k8snetworkplumbingwg/multus-cni
[KubeVirt]: https://github.com/kubevirt/kubevirt
[aojea/network-device-plugin]: https://github.com/aojea/network-device-plugin

[kubernetes/kubernetes#141604]: https://github.com/kubernetes/kubernetes/pull/141604
[services-networking concepts]: https://kubernetes.io/docs/concepts/services-networking/

### Risks and Mitigations

**Fail-open on version skew (security risk).** An old runtime or kubelet
that does not understand the field could silently attach an isolated pod to
the default pod network. The feature uses
[Node Declared Features (KEP-5328)] to keep isolated pods away from them:

- A node declares `NetworkIsolatedPods` only when its kubelet knows the
  field, has the gate enabled, and its runtime reports the capability via
  CRI `RuntimeFeatures`. A kubelet that predates the field never declares
  it.
- The scheduler never places an isolated pod on a node that does not
  declare the feature. This covers every pod that the scheduler binds,
  including DaemonSet pods.
- A kubelet that knows the field **fails closed**: if `podNetwork` is
  `Isolated` and the node does not declare the feature, the kubelet rejects
  the pod at admission with reason `PodFeatureUnsupported`. If a runtime
  reports the capability but attaches the sandbox anyway, the kubelet stops
  the sandbox and starts no container in it (see
  [Kubelet Behavior](#kubelet-behavior)).
- A pod that is created with `spec.nodeName` set does not go through the
  scheduler, and a kubelet that predates the field cannot fail closed. The
  kube-apiserver must reject placing an isolated pod on a node that does
  not declare the feature. The alpha implementation (v1.38) does not have
  this check. This is a gap, not a design choice: the need for the check
  was identified after the implementation merged, and there is no reason to
  allow these pods. Adding the check is a [Beta](#beta) requirement.
- Static pods do not go through the kube-apiserver. A static pod that sets
  `podNetwork: Isolated` on a kubelet that predates the field runs attached
  to the pod network. This needs write access to the node's manifests and is
  a documented residual risk.

See [Version Skew Strategy](#version-skew-strategy).

[Node Declared Features (KEP-5328)]: https://github.com/kubernetes/enhancements/issues/5328

**Ecosystem assumptions that every pod has an IP.** Controllers, meshes and
operators that read `status.podIP` will see an empty value. For pods whose
sandbox is still being created this is already true today, but a `Ready` pod
with no IP is a new observable state, and components may mishandle it —
anywhere from logging errors to crashing (e.g., code that assumes
`net.ParseIP(pod.Status.PodIP)` returns non-nil for every `Ready`,
non-`hostNetwork` pod). A pod network implementation with that assumption
could be crashed by any user allowed to create pods. Mitigations: the
feature is opt-in behind a feature gate; before beta the e2e tests must
pass on the 4 most common pod network implementations (Calico,
OVN-Kubernetes, Kindnet and Cilium) with `podNetwork: Isolated` pods in the
cluster; a conformance test checks this new state on every implementation;
and the new state is documented for integrators. Operators can additionally restrict who may set the field with
admission policy (e.g., ValidatingAdmissionPolicy). No Pod Security
Admission change is proposed: the field removes network access rather than
granting privilege. The endpoints controllers already ignore pods without
IPs.

**Interaction with `hostNetwork`.** `podNetwork` and `hostNetwork` are
separate fields that say different things: `hostNetwork` says that
the pod uses the node's network namespace, and `podNetwork` says whether the
pod's own network namespace is attached to the pod network. Validation
rejects setting both. `hostNetwork` keeps its meaning for every reader, so
Pod Security Admission (which reads `hostNetwork` for the `baseline` host
namespaces check) and policy webhooks that inspect `hostNetwork` keep
working without changes. The cost is one cross-field rule that a client
that does not know `podNetwork` can hit when it turns `hostNetwork` on for a
workload template that has the field stored. The API server handles that
case on update (see
[Updates from clients that do not know the field](#updates-from-clients-that-do-not-know-the-field)).

**Omitting the field on update does not clear it.** To protect isolated
workloads from clients that do not know the field, an update that leaves
`podNetwork` out keeps the stored value. This is also true for clients that
know the field: removing `podNetwork: Isolated` from a manifest and applying
it again (client-side or server-side apply) leaves the workload isolated.
Isolation is turned off by setting `podNetwork: Default`. Integration tests
cover this behavior, and the user-facing documentation must call it out.

**Scope creep towards multi-network.** An enum makes it easy to propose
new values (`bridge`-like modes, named networks), and once the API names
the default pod network someone will want to name the other ones. Not in this
KEP: all the core networking APIs only work with the default pod network, and
that stays the same. Any new value or API change needs its own KEP and
cannot change the guarantees for pods connected to the default pod network.

## Design Details

### API Changes

New enum type and `PodSpec` field:

```go
// PodNetwork selects whether a pod is attached to the pod network.
// +enum
// +k8s:enum
type PodNetwork string

const (
	// PodNetworkDefault attaches the pod to the pod network (the network that
	// Kubernetes connects every pod to unless the pod opts out). The container
	// runtime performs its configured network plumbing and the pod is assigned
	// pod IPs. This is the historical behavior (for non-hostNetwork pods).
	PodNetworkDefault PodNetwork = "Default"

	// PodNetworkIsolated does not attach the pod to the pod network. The pod gets
	// an isolated network namespace with only the loopback interface
	// configured. The pod's podIPs are left unset and the pod is never
	// selected into Services.
	PodNetworkIsolated PodNetwork = "Isolated"
)
```

```go
type PodSpec struct {
	// ...

	// hostNetwork indicates that the pod should use the host's network namespace.
	// ...
	// When true, podNetwork may not be set: the pod has no network namespace
	// of its own.
	// +optional
	HostNetwork bool `json:"hostNetwork,omitempty" protobuf:"varint,11,opt,name=hostNetwork"`

	// ...

	// podNetwork selects whether a non-hostNetwork pod is attached to the pod network.
	// "Default" attaches the pod to the pod network. "Isolated" gives
	// the pod an isolated network namespace with only the loopback interface
	// configured, not attached to the pod network.
	// This field must be unset if hostNetwork is true. For non-hostNetwork
	// pods, this behaves as if it was set to "Default" if it is not set.
	// If a pod is "Isolated", hostPort and network-based probes and lifecycle
	// handlers are not allowed.
	// This field is immutable on Pods.
	// +featureGate=NetworkIsolatedPods
	// +optional
	// +k8s:optional
	PodNetwork *PodNetwork `json:"podNetwork,omitempty" protobuf:"bytes,45,opt,name=podNetwork,casttype=PodNetwork"`
}
```

**Why an enum and not a boolean.** The first proposal was `hermetic *bool`
(see [Alternatives](#alternatives)). It got a lot of pushback. The
[API conventions] warn against booleans: they "trend towards a small set of
mutually exclusive options" and rarely stay binary. With an enum, a new
option later is just a new value.

`hostNetwork` stays a separate field. It is a GA v1 field and cannot be
removed, and `podNetwork` only describes pods that have their own network
namespace, so the two fields do not overlap: validation rejects a
`hostNetwork: true` pod that sets `podNetwork`, with any value (see
[Validation](#validation)).

[API conventions]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#primitive-types

### Validation

The validation applies to Pods and to all workload templates
([initial implementation][validation-commit]):

- Values other than `Default` and `Isolated` are rejected.
- The field may not be set, with any value, when `hostNetwork` is true. It
  describes the pod's own network namespace, and a `hostNetwork` pod has
  none.
- The field may only be set when the `NetworkIsolatedPods` gate is enabled,
  or when the existing object already sets it (see
  [Disabled feature gate](#disabled-feature-gate)).
- For `Isolated` pods, in every container:
  - `httpGet`, `tcpSocket` and `grpc` are forbidden in `livenessProbe`,
    `readinessProbe` and `startupProbe`. The kubelet runs these probes from
    the node, and nothing in an isolated pod is reachable from there,
    whatever host the probe points to;
  - `httpGet` and `tcpSocket` are forbidden in the `postStart` and `preStop`
    lifecycle handlers, for the same reason;
  - `ports[*].hostPort` is forbidden: there is no pod IP to forward to.
- Only `exec` probes and `exec` or `sleep` lifecycle handlers pass
  validation for isolated pods. If a future mechanism lets the kubelet probe
  isolated pods, this validation can be relaxed. A reflection-based unit
  test (`TestIsolatedPodHandlerCoverage`) walks the probe and handler types,
  so that adding a new handler without deciding whether it works for
  isolated pods fails CI.
- `podNetwork` cannot be changed after the pod is created, like the other
  fields that define the sandbox; the existing pod update validation
  enforces it. The field is mutable in workload templates.
- `ValidatePodStatusUpdate` rejects a non-empty `status.podIP`/`status.podIPs`
  when `spec.podNetwork` is `Isolated`, whatever the state of the feature
  gate, turning the empty-IP guarantee into an
  API invariant instead of a kubelet behavior. Status writes are normally
  node-owned and guarded only by RBAC and NodeRestriction; this narrow
  spec-consistency check also makes a fail-open node loud: its status
  updates are rejected instead of silently publishing an IP. The kubelet
  patches only `status`, so the check applies to kubelets that do not know
  the field too. The consequence on such a node is that the pod's status is
  never updated while its sandbox is running (it stays `Pending` in the API
  with the kubelet logging `Failed to update status for pod`); the kubelet
  only reads sandbox IPs from a `SANDBOX_READY` sandbox, so once the pod is
  deleted and the sandbox stopped, the final status update succeeds and the
  pod is removed normally.

[validation-commit]: https://github.com/kubernetes/kubernetes/commit/77704f35e8b72f7495edf137315ee6af17f7d152

### Updates from clients that do not know the field

A client built before the field existed sends a full update (PUT) without
it, and its patches carry the stored value through without knowing about
it. Without help from the API server, such a client would:

- fail the immutability check when it updates anything on a Pod that sets
  the field, for example a label;
- silently attach an isolated workload to the pod network when it replaces
  the workload object;
- get a validation error about a field it does not know when it turns
  `hostNetwork` on for a template that has the field stored.

The API server adjusts the field on update, before validation, for Pods and
for every resource that embeds a `PodTemplateSpec` (PodTemplate,
ReplicationController, ReplicaSet, Deployment, StatefulSet, DaemonSet, Job
and CronJob) ([initial implementation][update-commit]). The rules, for an
object that has the field stored, are:

- An update that omits the field keeps the stored value.
- An update that turns `hostNetwork` on without changing `podNetwork` drops
  the stored value, and the result is a valid host network template. A
  PATCH carries the stored value through, so the API server cannot tell
  this client apart from one that knows the field and sends the same value
  again while it turns `hostNetwork` on. That client gets a valid host
  network template too.
- Every other update goes to validation unchanged. Validation rejects a
  client that sets the field on an object that has `hostNetwork: true`,
  that sets both in one request on an object that did not have the field,
  or that changes the value while it turns `hostNetwork` on.

Consequences for clients that know the field:

- Isolation is turned off by setting `podNetwork: Default`. Omitting the
  field keeps `Isolated`. This includes server-side apply: a manager that
  applied `Isolated` and then applies a configuration without the field
  still gets `Isolated`, because an omitted field on update cannot be told
  apart from a client that never knew it.
- Once an object stores the field, the field only becomes unset again when
  `hostNetwork` is turned on.
- On Pods the field is immutable, so the first rule only lets clients that
  do not know the field send full updates of pods that set it.

[update-commit]: https://github.com/kubernetes/kubernetes/commit/a11c33b1923fa89fbf941eff4a562b0762effc4b

### Disabled feature gate

`podNetwork` does not follow the usual rule of dropping a disabled field
from new objects: dropping `Isolated` silently attaches a pod that asked
for isolation to the pod network. With the `NetworkIsolatedPods` gate
disabled on the kube-apiserver:

- `Isolated` on an object that does not already set the field is rejected
  by validation (`spec.podNetwork: Forbidden: the NetworkIsolatedPods
  feature gate is disabled`).
- `Default` means the same as unset. `dropDisabledFields` drops it when the
  existing object does not already set the field, so creating a Pod or a
  workload with `Default` succeeds and stores no field.
- Objects that already set the field keep it on update (also when the
  client omits it, see the previous section), stay valid, and are still
  fully validated.
- A workload template stored with `Isolated` keeps the value, but the pods
  that its controller creates are new objects, and they are rejected. The
  workload stops creating pods instead of creating pods that are attached
  to the pod network. To move the workload back to the pod network, set
  `podNetwork: Default` in the template: this is accepted because the object
  already sets the field, and the pods created from it have the field
  dropped.
- `hostNetwork` pods and pods that do not set the field are not affected.

### CRI Integration

A new typed field in `PodSandboxConfig` (`k8s.io/cri-api`) tells the runtime
whether to attach the sandbox network namespace to the default pod network.
CRI already expresses the host network through
`NamespaceOption.network = NODE`; the new field only describes sandboxes that
get their own network namespace (`NamespaceOption.network = POD`), so it has
no `HOST` value:

```proto
// PodNetwork selects whether a sandbox network namespace is attached to the
// pod network.
// See https://kep.k8s.io/6313 for more details.
enum PodNetwork {
    // POD_NETWORK_DEFAULT attaches the sandbox network namespace to the pod
    // network. The runtime performs its configured network plumbing (e.g.,
    // invokes CNI plugins on Linux). This is the default and matches
    // historical behavior.
    POD_NETWORK_DEFAULT = 0;
    // POD_NETWORK_ISOLATED creates the sandbox network namespace with only the
    // loopback interface configured (up) and does not attach it to the pod
    // network:
    // - The runtime MUST NOT perform its pod network plumbing when the
    //   sandbox is created or torn down (e.g., MUST NOT invoke CNI ADD or
    //   DEL on Linux).
    // - PodSandboxStatus.network MUST NOT report any IP: the field MAY be
    //   omitted, or set with an empty ip and no additional_ips.
    // - The sandbox resolver configuration (/etc/resolv.conf) MUST be
    //   generated from PodSandboxConfig.dns_config exactly as given, even
    //   when it is empty. The runtime MUST NOT fall back to the host's
    //   configuration.
    // Sandboxes with this value only run on nodes whose runtime reports
    // RuntimeFeatures.pod_network_isolated; a runtime that does not support it
    // never receives it from a kubelet.
    POD_NETWORK_ISOLATED = 1;
}

message PodSandboxConfig {
    // ... existing fields ...

    // pod_network selects whether the sandbox network namespace is attached
    // to the pod network. It only applies when NamespaceOption.network is
    // POD; for NODE (host network) sandboxes there is no sandbox network
    // namespace and runtimes MUST ignore this field. Kubelets that predate
    // the field leave it unset (POD_NETWORK_DEFAULT).
    // Feature gate: NetworkIsolatedPods
    // See https://kep.k8s.io/6313 for more details.
    PodNetwork pod_network = 10;
}
```

Proto3 cannot distinguish an unset enum from its zero value, so
`POD_NETWORK_DEFAULT = 0` is what an older kubelet sends for every sandbox,
including host-network ones; this is why the field only applies to
sandboxes with their own network namespace. The values carry the
`POD_NETWORK_` prefix because proto enum values share the package scope
(existing enums use the same convention: `PROPAGATION_*`, `SANDBOX_*`,
`CONTAINER_*`).

Capability discovery, so the kubelet can fail closed:

```proto
message RuntimeFeatures {
    // ... existing fields ...

    // pod_network_isolated is set to true if the runtime supports creating
    // sandboxes with PodNetwork POD_NETWORK_ISOLATED for every
    // runtime handler it serves. The capability is runtime-wide because the
    // default pod network plumbing it skips is done by the runtime, not by
    // the handler.
    // See https://kep.k8s.io/6313 for more details.
    bool pod_network_isolated = 6;
}
```

Runtime obligations for `POD_NETWORK_ISOLATED` sandboxes:

- create the netns according to `NamespaceOption` as usual;
- bring up `lo` (`127.0.0.1/8`, `::1/128`), the equivalent of
  `ip link set lo up`;
- no pod network plumbing on setup or teardown (no CNI `ADD` or `DEL` on
  Linux);
- report no IPs in `PodSandboxStatus.network`: omit the field, or set it
  with an empty `ip` and no `additional_ips`.

The kubelet only sends `POD_NETWORK_ISOLATED` to a runtime that reports
`pod_network_isolated`.

No released container runtime reports the capability yet. The
[aojea/containerd `cniless` branch] is the reference implementation of the
original POC and predates the `pod_network` field; it must be updated to
the merged CRI API. Upstream
containerd and CRI-O implementations are a beta graduation requirement.

[aojea/containerd `cniless` branch]: https://github.com/aojea/containerd/tree/cniless

### Kubelet Behavior

- Admission is fail-closed, through [Node Declared Features (KEP-5328)].
  The kubelet registers a `NetworkIsolatedPods` declared feature. Its
  `Requirements` are the `NetworkIsolatedPods` feature gate and the
  `pod_network_isolated` runtime capability, its `Discover` returns true
  only when the gate is enabled and the runtime reports the capability in
  `RuntimeFeatures`, and its `InferForScheduling` returns true for pods with
  `podNetwork: Isolated`. This is the same shape as the existing
  `UserNamespacesHostNetworkSupport` feature, which is gated on the
  `user_namespaces_host_network` runtime capability. The framework's
  existing kubelet admit handler rejects a pod that requires a feature the
  node does not declare with reason `PodFeatureUnsupported` (recorded as the
  pod's `status.reason`, a warning event and
  `kubelet_admission_rejections_total{reason="PodFeatureUnsupported"}`), so
  a `podNetwork: Isolated` pod is rejected when the gate is off or the
  runtime lacks the capability, with no kubelet code beyond the feature
  registration. Attaching an intentionally isolated pod to the default pod
  network is worse than not running it.
- `generatePodSandboxConfig` sets `pod_network: POD_NETWORK_ISOLATED`, and
  `PodSandboxChanged` does not recreate the sandbox because the IP is
  missing.
- If the runtime reports IPs for the sandbox of an isolated pod, the kubelet
  stops the sandbox and fails the pod sync before it starts any container:
  the pod asked for isolation. The failure is reported with a
  `FailedCreatePodSandBox` warning event and counted in
  `kubelet_started_pods_errors_total`. This covers a runtime that reports
  the capability and attaches the sandbox anyway.
- The kubelet does not wait for the pod network to start an isolated pod.
  When the runtime reports that the network is not ready
  (`NetworkNotReady`), the kubelet still starts `hostNetwork` pods and
  isolated pods.
- `status.podIP`/`status.podIPs` stay empty. Readiness is computed from
  container state only. `status.hostIP` is reported as usual.
- The managed `/etc/hosts` is still mounted, with the pod hostname (and
  `hostname.subdomain` if set) mapped to `127.0.0.1` and `::1` to avoid
  breaking applications that depend on that.
- The `kubelet_first_network_pod_start_sli_duration_seconds` metric ignores
  isolated pods, as it ignores `hostNetwork` pods: it measures the first pod
  that needs the pod network.

### Control Plane Behavior

- The kube-apiserver changes are the validation, the update handling and
  the disabled-gate handling described in the previous sections.
- No kube-controller-manager change is needed. `ShouldPodBeInEndpoints`
  (in `k8s.io/endpointslice/util`, shared by the Endpoints and EndpointSlice
  controllers) already returns `false` for pods without `status.podIPs`, and
  isolated pods never have them (an API invariant, see
  [Validation](#validation)), so they are never published even if a Service
  selector matches. The e2e tests check this for both controllers.
- The `NetworkIsolatedPods` declared feature (see
  [Kubelet Behavior](#kubelet-behavior)) is published in
  `node.status.declaredFeatures`, so the scheduler keeps isolated pods away
  from nodes without a supporting kubelet and runtime. This KEP only
  registers a new feature in that existing mechanism: it adds no scheduler
  code and no scheduler feature gate. The `NodeDeclaredFeatures` framework
  gate (GA and locked on since v1.37) is what enables the scheduler plugin
  that matches pod requirements against `node.status.declaredFeatures` and
  the kubelet admit handler. The kubelet admission remains the final
  guarantee for pods that bypass the scheduler, on kubelets that know the
  field.
- Missing from the alpha implementation, required for beta: the
  kube-apiserver rejects placing a `podNetwork: Isolated` pod on a node that
  does not declare `NetworkIsolatedPods`, when the pod is created with
  `spec.nodeName` set or is bound through the `pods/binding` subresource.
  This covers the pods that bypass the scheduler on kubelets that predate
  the field (see [Version Skew Strategy](#version-skew-strategy)). The need
  for this check was identified after the alpha implementation merged. The
  `NodeDeclaredFeatureValidator` admission plugin of KEP-5328 is the place
  for it; today it only validates pod updates, so extending it
  needs the agreement of SIG Node.

### Interaction with Core Networking APIs

With `podNetwork: Isolated`, "not connected to the default pod network"
becomes a supported state, so this section spells out how the core
networking APIs behave for those pods. All the core networking APIs work
only with the host network and the default pod network. The rules below
describe the current behavior; this KEP does not change any of them for
existing pods:

- **`status.podIPs`** only ever contains the pod's IPs on the pod's default
  network (pod-network IPs for pods attached to the pod network, node IPs
  for `hostNetwork` pods). Interfaces attached to isolated pods by
  delegated mechanisms are never reported there: publishing those addresses
  is the responsibility of the integration that attaches them (for example
  through the standardized DRA network device data of [KEP-4817]). Isolated
  pods therefore always have an empty `status.podIPs`, enforced by status
  validation (see [Validation](#validation)).
- **`hostPort`** forwards from a host address to `status.podIPs` and nothing
  else. With no pod IPs there is nothing to forward to, which is why it is
  rejected by validation for isolated pods.
- **Service virtual IPs.** Pods that are not attached to the default pod
  network are not *required* to be able to reach Service IPs, but
  they are *allowed* to: an implementation or integration may provide
  reachability by other means (a connect-time eBPF proxier, a
  driver-injected interface with the appropriate routes).
- **Cluster DNS** follows the same pattern for *access*: not required,
  allowed. With the default `dnsPolicy` (`ClusterFirst`), the resolver
  configuration of an isolated pod points to the cluster DNS Service. It works
  wherever something provides a path to the resolvers, and lookups fail
  otherwise; a pod that does not want that sets `dnsPolicy: None` with its
  own `dnsConfig`. The *contents* of
  cluster DNS are derived from `status.podIPs` (through Endpoints and
  EndpointSlices), so isolated pods have no DNS records.
- **Service selection.** Services select backends through `status.podIPs`:
  the Endpoints and EndpointSlice controllers never publish isolated pods
  (`ShouldPodBeInEndpoints`), even when a selector matches.
- **NetworkPolicy** `podSelector`/`namespaceSelector` select traffic to and
  from `status.podIPs` over the default pod network. Isolated pods have
  neither an attachment nor pod IPs, so NetworkPolicy does not apply to
  them, as with host-network pods today.
- **EndpointSlice API.** In `addressType: IPv4`/`IPv6` slices managed by the
  core controller, a `targetRef` to a Pod implies the addresses are that
  pod's `status.podIPs`, reachable over the default pod network. Slices
  created by third-party controllers (such as the out-of-core integrations
  described in the Motivation) may carry addresses from other networks; that
  is existing behavior and unchanged by this KEP.

Changing any of these APIs to work with other networks, or adding
`podNetwork` values for other networks, is out of scope. It needs its
own KEP.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None identified.

##### Unit tests

Coverage added with the alpha implementation:

- `pkg/apis/core/validation`: `TestValidatePodNetwork` (both values ×
  `hostNetwork` × probes × lifecycle handlers × `hostPort` × `dnsPolicy`,
  feature gate on and off), `TestIsolatedPodHandlerCoverage` (the
  probe/handler completeness test) and `TestAllowPodNetwork`.
- `pkg/api/pod`: `TestDropPodNetworkDefault` (`dropDisabledFields` with the
  gate disabled) and `TestSyncPodNetworkOnUpdate` (updates from clients that
  do not know the field, for Pods and templates).
- `test/declarative_validation/core/pod`: unsupported enum values.
- `pkg/kubelet/kuberuntime`: sandbox config generation
  (`TestGeneratePodSandboxConfigPodNetwork`), `PodSandboxChanged` for a
  sandbox without an IP, and `TestSyncPodNetworkIsolated` with a runtime
  that honors `pod_network` and with one that ignores it.
- `staging/src/k8s.io/component-helpers/nodedeclaredfeatures/features/networkisolatedpods`:
  `Requirements`, `Discover`, `InferForScheduling` and `InferForUpdate` of
  the `NetworkIsolatedPods` declared feature (gate on/off, runtime
  capability on/off).
- `pkg/kubelet`: managed hosts file content for isolated pods, and pod sync
  while the pod network is not ready.
- `pkg/kubelet/status`: readiness generation without pod IPs.

- `pkg/apis/core/validation`: `2026-09-02` - `84.9%`
- `pkg/apis/core/v1`: `2026-09-02` - `81.2%`
- `pkg/kubelet/kuberuntime`: `2026-09-02` - `68.3%`

##### Integration tests

`test/integration/pods/pod_network_test.go`:

- `TestPodNetworkCreate`: the field is not defaulted (unset stays unset,
  with and without `hostNetwork`); `Isolated` keeps the standard
  `dnsPolicy` and `enableServiceLinks` defaults and explicit values;
  `hostNetwork` with either value, network probes, `hostPort` and
  unsupported values are rejected.
- `TestPodNetworkUpdate`: immutability on Pods; a full update from a client
  that does not know the field keeps the stored value; a status update with
  a pod IP is rejected and one with only host IPs is accepted.
- `TestPodNetworkWorkloadTemplates`: Deployment, ReplicaSet, StatefulSet,
  DaemonSet, ReplicationController, PodTemplate, Job and CronJob templates;
  the rules of
  [Updates from clients that do not know the field](#updates-from-clients-that-do-not-know-the-field),
  including `hostNetwork` turned on for an `Isolated` template by PATCH and
  by full update; server-side apply.
- `TestPodNetworkGateDisabled`: `Isolated` is rejected and `Default` is
  dropped on create; stored objects keep the field and are still validated;
  a stored `Isolated` template is moved back to the pod network with
  `Default`.

- [TestPodNetwork*](https://github.com/kubernetes/kubernetes/blob/master/test/integration/pods/pod_network_test.go): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=TestPodNetwork), [triage](https://storage.googleapis.com/k8s-triage/index.html?test=TestPodNetwork)

##### e2e tests

- `test/e2e/network/isolated.go` (cluster e2e, `Network Isolated Pods`,
  labels `Feature:NetworkIsolatedPodsSupport` and
  `FeatureGate:NetworkIsolatedPods`, requires a runtime with support): the
  pod runs and becomes `Ready` with empty `status.podIP` and a reported
  `status.hostIP`; only `lo` exists in the sandbox; the pod accepts
  connections on loopback; port forwarding works; the hostname and
  `hostAliases` resolve via `/etc/hosts`; the downward API `status.podIP` is
  empty; exec probes work; the pod is not selected into the EndpointSlices
  or the Endpoints of a matching Service, while a regular pod with the same
  labels is; isolated pods run in Deployments, StatefulSets and Jobs.
- `test/e2e_node/network_isolated_test.go` (node e2e): with a runtime with
  support, sandbox creation without a pod IP and with only loopback,
  `/etc/resolv.conf` generated exactly from `dnsConfig` with
  `dnsPolicy: None`, exec probes, and kubelet restart idempotency (no
  sandbox recreation due to the missing IP). With any runtime, fail-closed
  admission (`PodFeatureUnsupported`) against a runtime without support and
  with the kubelet gate disabled.
- No CI job selects `Feature:NetworkIsolatedPodsSupport` yet: the tests that
  need runtime support do not run until a job uses a container runtime that
  reports `pod_network_isolated`. A periodic job will be added under
  sig-network.

- [Network Isolated Pods](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/network/isolated.go) and [node e2e](https://github.com/kubernetes/kubernetes/blob/master/test/e2e_node/network_isolated_test.go): links to [SIG Network testgrid](https://testgrid.k8s.io/sig-network-misc?include-filter-by-regex=Network.Isolated.Pods) and [triage](https://storage.googleapis.com/k8s-triage/index.html?test=Network%20Isolated%20Pods) to be added once the job exists.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `NetworkIsolatedPods` feature gate (default off)
  in kube-apiserver and kubelet.
- new Pod field `podNetwork` enum, validation and update handling complete.
- CRI `PodSandboxConfig.pod_network` and `RuntimeFeatures.pod_network_isolated`
  merged in `cri-api`.
- `NetworkIsolatedPods` registered in [Node Declared Features (KEP-5328)],
  which gives both the scheduler filter and the kubelet fail-closed
  admission.
- Unit and integration tests listed above; e2e tests merged, to run in CI
  when a runtime implementation is available.

#### Beta

- The kube-apiserver rejects placing a `podNetwork: Isolated` pod on a node
  that does not declare `NetworkIsolatedPods` (pod creation with
  `spec.nodeName` set, and `pods/binding`), with integration tests. See
  [Control Plane Behavior](#control-plane-behavior).
- At least one released upstream runtime (containerd and/or CRI-O) supports
  `PodNetwork POD_NETWORK_ISOLATED` and reports the capability.
- The e2e tests pass on clusters with the 4 most common pod network
  implementations: Calico, OVN-Kubernetes, Kindnet and Cilium.
  `podNetwork: Isolated` pods work as described here and other pods are not
  affected: no crashes, no error log spam, NetworkPolicy keeps working.
- Downstream consumers that attach secondary networks through CNI
  delegation (Multus, and KubeVirt through it) are not broken: pods that do
  not set `podNetwork: Isolated` keep working unchanged with those projects
  installed, verified by e2e, and there is a documented and tested
  compatibility path (NRI or CDI hooks, or DRA) for them to attach
  interfaces to `podNetwork: Isolated` pods.
- Upgrade, downgrade and feature gate disablement tested, including what
  happens to running isolated pods when the gate is disabled on the
  kubelet.
- Scheduling based on node declared features validated in heterogeneous
  clusters (mixed node versions and runtimes).
- Feedback gathered from DRA driver authors and early adopters.
- E2e tests running in Testgrid and linked in this KEP; no flakes for the
  gate-enabled jobs.
- Documentation updated, including the [services-networking concepts] pages
  that currently assume universal pod connectivity.

#### GA

- Container runtime support on containerd and CRI-O.
- Real-world usage of `podNetwork: Isolated` by at least two distinct classes of
  adopters (e.g., a DRA networking driver and a batch/security platform).
- At least two releases after beta to allow user feedback and bug reports.
- Tests are promoted to conformance, so we guarantee that the default pod
  network behavior does not change for pods attached to the pod network and
  for `hostNetwork` pods, and that the new `podNetwork: Isolated` behavior
  is implemented across all clusters.
- All issues and gaps identified during beta resolved.

#### Deprecation

Not applicable

### Upgrade / Downgrade Strategy

- **Upgrade.** Enabling the feature requires enabling the `NetworkIsolatedPods`
  gate on kube-apiserver and kubelets, plus a
  runtime version that reports the capability. Existing workloads require no
  changes: the field is not defaulted, so objects that do not set it are not
  modified and behavior is identical.
- **Downgrade / gate disablement on the kube-apiserver.** New uses of
  `Isolated` are rejected instead of dropped, stored objects keep the field,
  and workloads with an `Isolated` template stop creating pods until the
  gate is enabled again or the template is set to `Default` (see
  [Disabled feature gate](#disabled-feature-gate)).
- **Downgrade / gate disablement on a kubelet.** A kubelet with the gate
  disabled (including older kubelets that already know the field) does not
  declare the node feature and refuses to admit `podNetwork: Isolated` pods
  instead of running them attached to the network. Disabling the gate needs
  a kubelet restart, and the kubelet runs admission again for the pods it
  already has (see [Node Declared Features (KEP-5328)]). Isolated pods that
  were running on the node are therefore rejected too and transition to
  `Failed` (`PodFeatureUnsupported`); they are not left running. This KEP
  has no test for this path yet; adding one is a [Beta](#beta) requirement.
  Drain isolated pods from a node before the gate is disabled on its
  kubelet.

### Version Skew Strategy

- **kube-apiserver (HA, mixed versions).** During a rolling control-plane
  upgrade, an n-1 apiserver does not know the field: it drops it on write,
  or rejects the request when the client asks for strict field validation
  (the `kubectl` default). This is the standard skew behavior for a new
  field; users should not rely on the field until all apiservers are
  upgraded and the gate is enabled everywhere. A v1.38 apiserver knows the
  field but does not have the check for pods that bypass the scheduler
  (see the old kubelet case below); with its default configuration (gate
  off) it rejects new `Isolated` pods.
- **Old clients.** A client built before the field exists drops it on a full
  update (PUT). The API server keeps the stored value, so the update does
  not fail the immutability check on Pods and does not attach an isolated
  workload to the pod network. An old client that turns `hostNetwork` on
  for a template that has the field stored gets a host network template
  (see
  [Updates from clients that do not know the field](#updates-from-clients-that-do-not-know-the-field)).
  A client that deletes an object and creates it again loses the field, as
  with any field it does not know.
- **kube-controller-manager (any version).** The controller manager has no
  code for this feature. Isolated pods never have IPs, and pods without IPs
  are already excluded from Endpoints and EndpointSlices by existing
  controller logic.
- **kube-scheduler (n-1).** The scheduler infers the requirement from the
  pod spec with the `NetworkIsolatedPods` feature registered in v1.38, so a
  v1.38 scheduler already filters nodes for isolated pods.
- **Old kubelet (field unknown).** The most important skew: a kubelet that
  predates the field (v1.37 and older) ignores it and would run the pod
  attached to the default pod network (fail-open). With a v1.39 control
  plane these are the v1.36 and v1.37 kubelets; a v1.38 kubelet knows the
  field and fails closed. [Node Declared Features (KEP-5328)], GA
  since v1.37, exists for this case: a feature that a node does not declare
  is a feature that the node does not have.
  1. Old kubelets never declare `NetworkIsolatedPods`, so the scheduler
     never places `podNetwork: Isolated` pods on them. This includes
     DaemonSet pods, which the scheduler binds.
  2. Kubelets that know the field fail closed for the pods that reach them
     without the scheduler (`spec.nodeName`, static pods).
  3. The kube-apiserver rejects placing an isolated pod on a
     node that does not declare the feature, when the pod is created with
     `spec.nodeName` set or is bound through `pods/binding` (see
     [Control Plane Behavior](#control-plane-behavior)). This closes the
     path that reaches an old kubelet without the scheduler. The alpha
     implementation (v1.38) does not have this check; adding it is a beta
     requirement. Until then, do not set `spec.nodeName` on isolated pods
     for nodes that do not declare the feature.
  4. If an old kubelet runs an isolated pod anyway, the API server rejects
     its status updates because they carry a pod IP (see
     [Validation](#validation)), so the problem is visible.

  The residual risk is a static pod that sets `podNetwork: Isolated` in the
  manifests of a kubelet that predates the field. It never goes through the
  kube-apiserver, and it needs write access to the node's file system. This
  is documented.
- **New kubelet + old runtime.** The runtime does not report
  `pod_network_isolated`, the node does not declare the feature, and the
  kubelet rejects the pod at admission (fail-closed, `PodFeatureUnsupported`).
- **New kubelet + runtime that reports the capability but ignores the
  field.** The kubelet finds IPs on the sandbox, stops it and starts no
  container; the API server also rejects a status with pod IPs.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `NetworkIsolatedPods`
  - Components depending on the feature gate: kube-apiserver, kubelet

###### Does enabling the feature change any default behavior?

No. The field is optional and never defaulted, so pods and workloads that do
not set it are not modified. Only pods that explicitly set
`podNetwork: Isolated` behave differently, and that value cannot be set
without the gate.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, and it fails closed. Disabling the gate on the apiserver rejects new
uses of `podNetwork: Isolated`; the value is not dropped, because that would
attach the pod to the pod network. `Default` is dropped from new objects.
Stored objects keep the field and remain valid. Workloads whose template is
stored with `Isolated` cannot create new pods while the gate is disabled;
they are moved back to the pod network by setting `podNetwork: Default` in
the template. Kubelets with the gate disabled refuse to admit
`podNetwork: Isolated` pods; this includes isolated pods that were running
when the kubelet restarted with the gate disabled, which transition to
`Failed` (see
[Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)). No
existing (non-isolated) workload is affected in any way by enabling or
disabling the gate.

###### What happens if we reenable the feature if it was previously rolled back?

`Isolated` can be set on new objects again. Stored objects that retained
`podNetwork` values resume full semantics: workloads with an `Isolated`
template create pods again. There is no state to reconcile beyond the field
itself.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests cover `dropDisabledFields` and validation with the gate on
and off, including objects that already set the field. The integration test
`TestPodNetworkGateDisabled` creates a Pod and a Deployment with the gate
enabled, disables it, and checks creates, updates and status updates against
a real apiserver. A node e2e test checks that a kubelet with the gate
disabled rejects a `podNetwork: Isolated` pod. Running isolated pods on a
kubelet that restarts with the gate disabled are not tested yet.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Running workloads are not touched: the feature only affects pods explicitly
setting `podNetwork: Isolated`, and sandboxes are never reconfigured in place.
Failure modes during rollout are limited to isolated pods themselves:
an isolated pod that reaches a node whose kubelet or runtime lacks support
gets a
`PodFeatureUnsupported` admission rejection (visible, fail-closed) on
kubelets that know the field. A kubelet that predates the field runs the pod
wired to the default pod network, but only a pod that bypasses the scheduler
can reach it (see [Version Skew Strategy](#version-skew-strategy)). Mixed-version HA
control planes may intermittently drop the field on write until all
apiservers are upgraded.

A rollback (gate disabled) impacts isolated workloads only, and it fails
closed: their controllers cannot create new pods, and isolated pods on a
kubelet that restarts with the gate disabled are rejected
(see [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)).

###### What specific metrics should inform a rollback?

- `kubelet_admission_rejections_total{reason="PodFeatureUnsupported"}`
  (unschedulable/failing isolated pods).
- apiserver validation rejections of status updates that attempt to set
  `status.podIP` on `podNetwork: Isolated` pods (indicates a fail-open node
  that must be upgraded or cordoned).
- `kubelet_started_pods_errors_total` together with `FailedCreatePodSandBox`
  events for isolated pods (a runtime that reports the capability and
  attaches the sandbox anyway).
- Standard rollout signals: `kubelet_pod_start_duration_seconds`,
  apiserver validation error rates on pod writes.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet; it will be tested before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Query pods with `spec.podNetwork=Isolated` (e.g., via kube-state-metrics or a
field query). On nodes, `PodFeatureUnsupported` admission rejections (events
and `kubelet_admission_rejections_total`) indicate attempted use on
unsupported nodes.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: the pod is `Running`/`Ready` with empty `status.podIP` and
    `status.podIPs`; `kubectl exec` into the pod shows only the loopback
    interface. Status updates attempting to report a pod IP for an
    isolated pod are rejected by validation and indicate a fail-open node.
- [x] Events
  - Event Reason: `PodFeatureUnsupported` when the node cannot honor the
    isolation request (fail closed); the message names the missing
    `NetworkIsolatedPods` feature.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Isolated pod sandbox creation should be at least as fast and reliable as
regular pods on the same node (it strictly removes the network plumbing step
from the startup path). No regression in startup latency SLOs for
non-isolated pods.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_pod_start_duration_seconds`,
    `kubelet_started_pods_errors_total`,
    `kubelet_admission_rejections_total{reason="PodFeatureUnsupported"}`.
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. `kubelet_admission_rejections_total` already exists with a `reason`
label and `PodFeatureUnsupported` is already in its allowlist.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- Container runtime with CRI `PodNetwork POD_NETWORK_ISOLATED`
  support (containerd/CRI-O version supporting this KEP).

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, marginally. `PodSpec`, and every `PodTemplateSpec` embedded in workload
objects, gains one optional enum field. The field is not defaulted, so only
the objects that set it grow, by a few bytes (the field name and a short
string value). No new objects are created.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Isolated pod startup removes the network plumbing step entirely so it will improve pod startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Resource usage decreases using isolated pods (no CNI execution,
no IPAM, no endpoints processing).

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. Isolated pods consume fewer node resources than equivalent
cluster-networked pods (no veth pairs, no IPs, ...).

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No change: isolated pods behave like any other pod. They do not need the
pod network to be ready on the node to start.

###### What are other known failure modes?

- Isolated pod stuck rejected on a node.
  - Detection: pod `status.reason` and a warning event with reason
    `PodFeatureUnsupported`;
    `kubelet_admission_rejections_total{reason="PodFeatureUnsupported"}`.
  - Mitigations: upgrade the node's runtime/kubelet or reschedule the pod to
    a supporting node; this is the intended fail-closed behavior.
  - Diagnostics: the event message lists the unsatisfied feature
    (`NetworkIsolatedPods`); `node.status.declaredFeatures` shows what the
    node supports; the kubelet log names the runtime and the missing
    `RuntimeFeatures.pod_network_isolated` capability.
  - Testing: covered by node e2e against a runtime without support.
- Isolated pod not started because the runtime attached its sandbox to the
  pod network.
  - Detection: `FailedCreatePodSandBox` warning events with the message
    `runtime attached the sandbox of a pod with podNetwork "Isolated" to the
    pod network`; `kubelet_started_pods_errors_total` increases; the pod
    stays `Pending`.
  - Mitigations: fix or upgrade the container runtime: it reports
    `pod_network_isolated` and does not honor `pod_network`.
  - Diagnostics: the event and the kubelet log list the IPs that the
    runtime reported.
  - Testing: unit test with a runtime that ignores the field
    (`TestSyncPodNetworkIsolated`).
- Workload with an `Isolated` template does not create pods after the gate
  was disabled on the kube-apiserver.
  - Detection: the workload controller reports pod creation failures with
    `spec.podNetwork: Forbidden: the NetworkIsolatedPods feature gate is
    disabled`.
  - Mitigations: enable the gate again, or set `podNetwork: Default` in the
    template to move the workload back to the pod network.
  - Diagnostics: events of the workload object; kube-apiserver feature
    gates.
  - Testing: `TestPodNetworkGateDisabled` covers the rejected create and the
    move back with `Default`.
- Isolated pod attached to the default pod network (fail-open) on a
  kubelet that predates the field. Only a pod that bypasses the scheduler
  can reach such a kubelet.
  - Detection: the kubelet's status updates are rejected by apiserver
    validation (non-empty `status.podIP` for a `podNetwork: Isolated` pod);
    the pod stays `Pending` in the API while running on the node, and the
    kubelet logs `Failed to update status for pod`.
  - Mitigations: delete the pod and upgrade the node; do not set
    `spec.nodeName` on isolated pods for nodes that do not declare
    `NetworkIsolatedPods` (the kube-apiserver check required for beta
    rejects them).
  - Diagnostics: comparison of pod spec vs. status; node kubelet version;
    `node.status.declaredFeatures`.
  - Testing: node declared features (from alpha) keep the scheduler away
    from these nodes, and integration tests cover the kube-apiserver check
    (beta); the fail-open path itself is not testable on the old kubelet.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect kubelet admission events/metrics for rejection reasons, verify
runtime capability via the node's CRI status, and check for version skew
between apiserver, kubelet and runtime.

## Implementation History

- 2026-08-26: Proposal socialized on the SIG Network [mailing list] and in a
  [design doc], gathering initial feedback.
- 2026-08-26: POC published as [kubernetes/kubernetes#141604] (API bool +
  validation/defaulting, CRI plumbing, kubelet behavior, endpoints exclusion,
  e2e/node tests, patched containerd reference implementation).
- 2026-09-02: KEP created as `provisional`, adopting the `networkMode` enum
  design based on POC and design-doc review feedback.
- 2026-09-09: Field renamed to `defaultNetwork` with values `Pod`, `Host` and
  `None` based on KEP review feedback.
- 2026-09-26: KEP marked `implementable` targeting alpha in v1.38. CRI field
  renamed to `default_network`, scoped to sandboxes with their own network
  namespace; kube-controller-manager dropped from the feature gate;
  compatibility with CNI-delegation based secondary networks added as a
  beta criterion.
- 2026-09-27: Fail-closed kubelet admission expressed as a Node Declared
  Feature (`PodFeatureUnsupported`, existing metric) instead of a bespoke
  check; `hostNetwork: true` takes precedence over the defaulted `Pod` value
  so old clients patching workload templates keep working; CRI enum values
  prefixed to avoid the `NamespaceMode.POD` clash.
- 2026-11-16: Alpha implementation merged for v1.38
  ([kubernetes/kubernetes#143280]). The design changed during the API
  review of the implementation: the field is `podNetwork` with the values
  `Default` and `Isolated`; it is not defaulted and does not mirror
  `hostNetwork` (it may not be set on `hostNetwork` pods); `dnsPolicy` and
  `enableServiceLinks` keep their standard defaults; the feature gate is
  `NetworkIsolatedPods`; the CRI field and capability are `pod_network` and
  `pod_network_isolated`; an update that omits the field keeps the stored
  value; a disabled gate rejects new uses of `Isolated` instead of dropping
  the field.
- 2026-11-17: KEP updated to match the alpha implementation. Beta moved
  from v1.41 to v1.39 and stable from v1.43 to v1.41: kubelets that predate
  the field are handled with
  Node Declared Features instead of waiting three releases, as suggested in
  the discussions at KubeCon. A kube-apiserver check for pods that bypass
  the scheduler is added as a beta requirement.

[kubernetes/kubernetes#143280]: https://github.com/kubernetes/kubernetes/pull/143280

[mailing list]: https://groups.google.com/a/kubernetes.io/g/sig-network/c/dWqf4h4Dz8s/m/6MNo-2w2EgAJ
[design doc]: https://docs.google.com/document/d/1vVZ2zazJnDaj3z0Ahi6fknGJo_SIYXoteQZ3Q3BGYBI/

## Drawbacks

- For opted-in pods only, it breaks the long-standing documented model where
  every pod has an IP and can reach every other pod and the API server. The
  [services-networking concepts] documentation must be updated. Tooling that
  assumes every pod has an IP will see empty values (the same state as a
  sandbox that has not started yet).
- Container runtimes must implement and test a third sandbox networking mode.

## Alternatives

**A boolean field (`hermetic: *bool`).** The first design, implemented in
the POC ([kubernetes/kubernetes#141604]). It got a lot of pushback: a
boolean cannot grow to more options, and "hermetic" promises more than
network isolation. The enum can get new values later (see
[API Changes](#api-changes)).

**`defaultNetwork: Pod/Host/None`, mirroring `hostNetwork`.** The version of
this KEP that was first merged. One enum described the three cases. API
defaulting set it on every Pod and workload template (`Pod`, or `Host` when
`hostNetwork: true`) and kept it in sync with `hostNetwork` in both
directions. It was replaced during the API review of the implementation.
The trade-offs:

- With the gate enabled, defaulting changes every Pod and every workload
  template, including the ones that do not use the feature.
- Two fields say the same thing and must be kept in sync on every write.
  That version of the KEP already had a case with no good answer: a client
  that does not know the field and patches `hostNetwork: false` into a
  template stored with `Host` does not see its change.
- `podNetwork` is optional and never defaulted. Objects that do not use the
  feature are not modified, `hostNetwork` remains the only way to request
  the host network, and the only cross-field rule is that the two fields
  cannot be set together.

**`None`-specific defaults for `dnsPolicy` and `enableServiceLinks`.** The
same version defaulted `dnsPolicy` to `None` (allowed without nameservers)
and `enableServiceLinks` to `false` for isolated pods, and made
`enableServiceLinks` also govern the `KUBERNETES_*` variables for them. The
implementation does not do this. `podNetwork` only decides whether the
sandbox is attached to the pod network; DNS and service links are
configured with their own fields, with the same defaults and validation as
for any other pod. A default that depends on the value of another field
makes the same manifest mean different things, and integrations that
provide connectivity by other means had to override those defaults again.

**Alternative field names.** Considered and rejected:

| Candidate | Assessment |
| --- | --- |
| `hermetic: true` | Implies isolation beyond networking (storage, host access, compute). Not used in the API, the KEP title or this document; see [Notes/Constraints/Caveats](#notesconstraintscaveats). |
| `loopbackOnly: true` | Too mechanical; describes the Linux namespace state rather than workload intent, and is unclear about DNS/service links/probes. |
| `isolated: true` | Overloaded as a field name: confused with compute isolation (CPU pinning, NUMA), security sandboxes (gVisor/Kata) and NetworkPolicy namespace isolation. As a value of `podNetwork`, the field name gives the missing context. |
| `airgapped: true` | Misleading: commonly refers to offline clusters/infrastructure, not individual sandboxes. |
| `disconnected: true` | Sounds like a transient operational state (edge node offline), not an intentional configuration. |
| `noNetwork: true` | Technically inaccurate: the pod does have a network namespace and a loopback interface. |
| `cniDisabled: true` | Leaks a runtime implementation detail; CNI is not Kubernetes API vocabulary and does not apply to Windows or VM runtimes. |
| `podNetwork: true/false` | A second boolean that cannot be true at the same time as `hostNetwork`; needs validation for that combination and cannot grow to more options. |
| `defaultNetwork: Pod/Host/None` | The version of this KEP that was first merged; see above. |
| `networkMode: Default/Host/None` | An earlier version of this KEP. "mode" does not say what the pod is or is not connected to; `podNetwork` names the network, the same one all the core networking APIs work with. |

`podNetwork: Isolated` avoids the problems above: the field says which
network the pod is or is not attached to, the value describes the state of
the sandbox, and `hostNetwork` is left alone.

**Pod annotation (e.g., `io.kubernetes.cri.sandbox.hermetic`).** Annotations
are not versioned, not validated, and fail open: an unaware kubelet or
runtime silently ignores them, and the dependent fields (probes, DNS, service
links) cannot be validated.

**Multi-network attachment API.** The multi-network effort models an explicit
list of networks that a pod attaches to; an empty list could mean "no
networks". That design is much larger and has unresolved conformance
questions (Services, NetworkPolicy, probes and DNS across networks). This KEP
only covers the absence of automatic attachment — a binary intent — and does
not preclude any future multi-network work.

**Deny-all NetworkPolicy.** It depends on a plugin that enforces the policy.
The pod still gets an IP and node CIDR space, still attaches interfaces,
still publishes endpoints, and there is no guarantee about the sandbox
contents.

**RuntimeClass / per-node runtime configuration that skips CNI.** It is not
declarative per pod and it is invisible to the control plane: validation,
endpoints and status handling remain wrong. It is not portable across
runtimes and it requires dedicated node pools.
