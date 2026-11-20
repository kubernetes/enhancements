# KEP-6508: Pod Resource Hints for Autoscalers

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [The churn loop](#the-churn-loop)
  - [Why each node autoscaler integrating each pod autoscaler does not scale](#why-each-node-autoscaler-integrating-each-pod-autoscaler-does-not-scale)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Hints at a glance](#hints-at-a-glance)
  - [User Stories](#user-stories)
    - [Story 1: Startup boost then in-place shrink](#story-1-startup-boost-then-in-place-shrink)
    - [Story 2: Recommendation applied only on recreate](#story-2-recommendation-applied-only-on-recreate)
    - [Story 3: A third-party pod autoscaler](#story-3-a-third-party-pod-autoscaler)
    - [Story 4: Capacity buffers and DaemonSet overhead](#story-4-capacity-buffers-and-daemonset-overhead)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [The resourcehints subresource](#the-resourcehints-subresource)
  - [Validation](#validation)
  - [Writer contract](#writer-contract)
  - [Consumer contract](#consumer-contract)
  - [Combining hints from more than one manager](#combining-hints-from-more-than-one-manager)
  - [Notes from the prototypes](#notes-from-the-prototypes)
  - [How VPA would write hints](#how-vpa-would-write-hints)
  - [How Karpenter would read hints](#how-karpenter-would-read-hints)
  - [How Cluster Autoscaler would read hints](#how-cluster-autoscaler-would-read-hints)
  - [Workload-level hints](#workload-level-hints)
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
  - [Node autoscalers read each pod autoscaler's CRD](#node-autoscalers-read-each-pod-autoscalers-crd)
  - [Aggregated &quot;pod recommendation&quot; API](#aggregated-pod-recommendation-api)
  - [Pod annotation](#pod-annotation)
  - [Shared CRD](#shared-crd)
  - [Dry-run-only pod create](#dry-run-only-pod-create)
  - [Replacement PodTemplate owned by the workload controller](#replacement-podtemplate-owned-by-the-workload-controller)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation (e.g. additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes)

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Pod autoscalers (VPA, and the many in-house or vendor ones) change how big a
pod is by changing its resource requests. In this KEP "pod autoscaler" means
this kind only, not horizontal autoscalers such as HPA. Node autoscalers
(Cluster Autoscaler, Karpenter) decide how many nodes to run and which pods to
move. Today the two only meet through the pod's
current `spec.containers[].resources`. That is the right number for a pod that
stays where it is, but often the wrong number for the pod that replaces it
after an eviction: the pod was boosted at start and shrunk in place, or the
recommendation only applies when the pod is recreated.

This KEP adds a small, autoscaler-neutral `status.resourceHints` list to Pod,
written through a new `pods/resourcehints` subresource. A pod autoscaler
publishes one thing the pod spec does not say: `replacement`, the requests a
replacement for this pod would be admitted with right now, including any
startup boost.

Node autoscalers read it from the Pod objects they already watch when they
simulate moving a pod. They no longer need to import, watch, and reimplement
any specific pod autoscaler's CRD and admission logic.

## Motivation

### The churn loop

The motivating failure is a loop between a pod autoscaler and a node
autoscaler that each act correctly on the information they have.

1. A Deployment's template requests 3 CPU because the app is CPU hungry at
   start. A pod autoscaler boosts at start and shrinks the pod in place to
   500m once it is Ready.
2. The node autoscaler sees a 500m pod on a 4 CPU node and consolidates: it
   picks a 2 CPU node and evicts the pod.
3. The replacement is admitted at 3 CPU, does not fit, and goes Pending.
4. The node autoscaler launches a 4 CPU node. Go to 1.

The loop repeats every few minutes, bounded only by node launch time and the
pod autoscaler's shrink delay. The same divergence between "how big the
pod is now" and "how big its replacement will be" appears whenever:

- a startup boost is active or has ended (VPA AEP-7862),
- a recommendation is only applied at admission (VPA `Initial` and `Recreate`
  modes, or the observation window in AEP-9936),
- the autoscaler grows a pod in place after it starts,
- a workload is switched back to recommend-only, so running pods keep the
  autoscaler's size and new ones start at the template.

### Why each node autoscaler integrating each pod autoscaler does not scale

[kubernetes/autoscaler#9898] asks for "a common mechanism that recommendation
systems (VPA, custom controllers) can implement and node autoscalers can
consume, enabling a node autoscaler to ask 'if this pod were recreated now, what
resources would it receive?' without replicating internal logic of the
recommender."

Karpenter shipped an interim answer ([karpenter#3134], [karpenter#3235]): it
polls VPA objects, resolves pod owners to VPA `targetRef`s, and replicates VPA
admission logic (controlled resources, mode, min/max clamping, startup boost).
The Karpenter RFC lists the cons itself: it couples Karpenter to VPA's CRD
schema, only works for VPA, and replicates webhook logic. Maintainers said they
would deprecate it once a neutral interface exists. Cluster Autoscaler has no
VPA integration at all. Any vendor pod autoscaler needs a new source in
each node autoscaler, with its own RBAC, owner informers, CRD-absent handling,
and hydration rules.

As @DerekFrank put it on [KEP-5972]: "What we need is a way to know how a pod
will come back if we get rid of it, which can then accommodate in place resizing
and dynamic containers. We can simplify lots of the system if we build one API
for that instead of taking it on piecemeal."

### Goals

- Define one vendor-neutral place where any pod autoscaler publishes the
  replacement size that node autoscalers need, and that node autoscalers read
  without knowing which pod autoscaler wrote them.
- Fix the churn loop for startup boost and recreate-only recommendations, in
  both Cluster Autoscaler and Karpenter.
- Let a pod autoscaler publish a hint without being granted `pods/status`
  or `pods` update.
- Cover per-container requests, native sidecars, and pod-level resources.
- Fail open: no hints means today's behavior, with no startup hydration
  dependency on any optional component.

### Non-Goals

- Changing how kube-scheduler or kubelet admit, place, or size pods. Hints are
  advisory and never enforced.
- Replacing the pod autoscaler's own API (VPA objects stay as they are).
- Predicting every possible admission mutation (sidecar injection, secrets).
  See [dry-run create](#dry-run-only-pod-create) in Alternatives.
- Limits. Node autoscalers pack on requests.
- Horizontal scaling hints.
- Hints about pods that stay on their node. A running pod's footprint is
  already on the Pod as `max(spec, allocated)`, and an in-place resize updates
  `spec` before kubelet actuates it, so a pending grow is already counted. Grows rejected at
  admission because they exceed the node's allocatable ([kubernetes#136043])
  are left to the pod autoscaler or to scheduler preemption for resize
  ([KEP-5836]).
- Recommendation ranges, startup timing, and pressure (PSI) signals. No node
  autoscaler decision needs them to fix the churn loop.
- Workload-level hints in alpha. See [Workload-level hints](#workload-level-hints).

## Proposal

### Hints at a glance

When a node autoscaler consolidates, it needs two numbers:

| Question | Applies to | Source |
|---|---|---|
| How big will the evicted pod be when it comes back? | Every pod being moved, to an existing or a new node | **New:** `replacement` hint |
| How much room do the pods already on the destination take? | Pods that stay where they are | Already on the Pod: `max(spec, allocated)` |

| Hint | Question it answers | Main writer | Main consumers |
|---|---|---|---|
| `replacement` | If this pod is evicted now, what will its replacement request? | Pod autoscaler (from its admission logic) | Node autoscaler consolidation, scale-down, drift simulation; capacity buffers; DaemonSet overhead |

Each manager owns one entry in the list, keyed by `manager`.

### User Stories

#### Story 1: Startup boost then in-place shrink

My Java service needs 3 CPU for about 90 seconds at start and 500m afterwards.
My pod autoscaler boosts it at admission and shrinks it in place. While
the boost is active the pod's requests already equal its replacement's, so it
publishes nothing (W2). After the shrink it publishes
`replacement: {app: {cpu: 3}}`. My node autoscaler simulates moving the pod at 3 CPU,
sees it will not fit on the smaller node, and leaves it. The loop stops without
the node autoscaler knowing which pod autoscaler I use.

#### Story 2: Recommendation applied only on recreate

My VPA is in `Recreate` mode and its target for my pod is 2 CPU. The running
pod still requests 500m. VPA publishes `replacement: {app: {cpu: 2}}`. When the
node autoscaler considers draining the node, it plans for 2 CPU, so it does not
consolidate onto a node that cannot hold the replacement. The opposite also
works. If the target is smaller than the current request, the node autoscaler
can consolidate a node it would otherwise keep.

#### Story 3: A third-party pod autoscaler

I run an in-house right-sizer. I implement the writer contract and grant it
`pods/resourcehints`. Both Cluster Autoscaler and Karpenter honor it with no
code or configuration changes in either.

#### Story 4: Capacity buffers and DaemonSet overhead

A `CapacityBuffer` with a `scalableRef` to a VPA-managed Deployment sizes its
placeholder pods from the workload's live pods. With hints, it uses
`replacement` and reserves room for what a new replica will actually request.
The same rule makes node autoscalers size DaemonSet overhead on new nodes from
a DaemonSet pod's `replacement` instead of the template ([karpenter#2346]).

### Notes/Constraints/Caveats

- Hints are about the replacement, not about this pod's allocation.
  kube-scheduler and kubelet ignore them. A node autoscaler that ignores them
  keeps today's behavior.
- `replacement` is per container and per resource. It is merged over the
  replacement's spec, not substituted for it. Containers and resources the
  writer does not manage (injected sidecars, LimitRange defaults) are omitted
  and keep their current values.
- `replacement` describes a pod from the same owner created now. Pods in a
  rolling update can have different templates. Hints are per pod, so each pod's
  hint reflects its own template.
- Pod-level resources ([KEP-2837]) are covered with `podRequests`. When a pod
  sets `spec.resources`, consumers apply `podRequests` the same way they apply
  `spec.resources` today.
- Pod status is reset on create, so a hint can never be set at admission. The
  writer sets it after the pod exists. A fresh pod has no hint until then,
  which is safe because a fresh pod's spec already reflects admission.

### Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Write amplification, one patch per pod per recommendation change (the main objection in [kubernetes/autoscaler#9898]) | Writers omit `replacement` when it equals the pod's current effective requests, and update only when values move beyond the writer's own actuation threshold, or when the writer resizes or evicts. A stable workload writes nothing. In-place writers already patch each pod they resize, so the hint adds at most one small patch per resize. In recreate-only modes (VPA `Initial` and `Recreate`), running pods are not resized, so a recommendation change rewrites the hint on every pod of the workload, and each write is a full Pod watch event. [Workload-level hints](#workload-level-hints) reduce this to one write per workload at beta. API Priority and Fairness applies to the subresource. |
| Stale hint after the writer dies or stops managing a pod | `updateTime` on every entry. Consumers MAY ignore entries older than a configured age. Writers MUST remove their entry when they stop managing a pod. A stale hint that is too large only makes the node autoscaler more conservative. A stale hint that is too small can still cause the churn loop, which is why W3 asks writers to publish increases promptly. |
| Malicious or buggy writer inflates hints to block consolidation | Writing requires `pods/resourcehints`, which is separate from `pods`, `pods/status` and `pods/resize`. Validation caps entry count and size. Hints cannot change scheduling or allocation. The worst outcome is a node autoscaler keeping or adding nodes, which is visible through the `manager` field. |
| Two pod autoscalers disagree | Consumers take the per-resource max of `replacement` across managers. This is the safe direction for churn. Running two pod autoscalers on one pod stays unsupported, and the conflict is visible on the Pod. |
| Kubelet or other `pods/status` writers clobber hints | The `pods/status` strategy preserves `status.resourceHints`, like it preserves other fields owned by other subresources. Integration tests cover kubelet-style full status updates. |
| Hint larger than anything the node autoscaler can provision | Consumers treat the pod as not movable, since its replacement could not be scheduled anywhere, so consolidation of its node is blocked rather than made unsafe. They emit an event naming the manager. |

## Design Details

### API

New fields in `k8s.io/api/core/v1`. All are behind the `PodResourceHints`
feature gate.

```go
type PodStatus struct {
	// ...existing fields...

	// ResourceHints are sizing hints published by pod autoscalers and
	// other managers for consumers such as node autoscalers. Each entry is
	// owned by one manager. Hints are advisory. kubelet and kube-scheduler
	// ignore them.
	// Written only through the pods/resourcehints subresource.
	// +featureGate=PodResourceHints
	// +optional
	// +listType=map
	// +listMapKey=manager
	// +k8s:maxItems=4
	ResourceHints []PodResourceHint `json:"resourceHints,omitempty" protobuf:"bytes,NN,rep,name=resourceHints"`
}

// PodResourceHint is one manager's sizing hints for a pod.
type PodResourceHint struct {
	// Manager identifies the writer as a domain-prefixed path, for example
	// "autoscaling.k8s.io/vpa-updater".
	// +required
	Manager string `json:"manager"`

	// UpdateTime is when the manager last computed this entry.
	// +required
	UpdateTime metav1.Time `json:"updateTime"`

	// Replacement is what a replacement of this pod would request if it were
	// created now, including any startup boost. It is merged over the
	// replacement's spec per container and per resource.
	// +optional
	Replacement *ResourceHintValues `json:"replacement,omitempty"`
}

// ResourceHintValues are requests for some of a pod's containers and,
// for pods with pod-level resources, for the pod.
type ResourceHintValues struct {
	// Containers lists per-container requests. Names refer to entries in
	// spec.containers or spec.initContainers (including native sidecars).
	// Containers and resources not listed keep their spec values.
	// +optional
	// +listType=map
	// +listMapKey=name
	Containers []ContainerResourceHint `json:"containers,omitempty"`

	// PodRequests are pod-level requests, used when spec.resources is set.
	// +optional
	PodRequests ResourceList `json:"podRequests,omitempty"`
}

type ContainerResourceHint struct {
	// +required
	Name string `json:"name"`
	// +optional
	Requests ResourceList `json:"requests,omitempty"`
}
```

Example, a startup-boosted pod after its in-place shrink to 500m:

```yaml
status:
  resourceHints:
  - manager: autoscaling.k8s.io/vpa-updater
    updateTime: "2026-10-16T18:02:11Z"
    replacement:
      containers:
      - name: app
        requests: {cpu: "3"}
```

<<[UNRESOLVED status vs spec ]>>
`status` is proposed because hints are derived, observed information that
kubelet does not act on, and because they must survive without the user's
manifest knowing about them. The cost is that `pods/resize` cannot carry a hint
in the same request, so in-place writers make two calls. A `spec` field written
through `pods/resize` would make them atomic, at the price of putting
non-desired state in spec.
<<[/UNRESOLVED]>>

### The resourcehints subresource

`pods/resourcehints` supports `get`, `update` and `patch` (including
server-side apply). Its strategy copies only `status.resourceHints` from the
request onto the stored object and ignores everything else, like `pods/resize`
does for container resources. The `pods/status` strategy and the main `pods`
strategy preserve the stored `status.resourceHints`, so kubelet status updates
and user edits never drop hints.

Writers use server-side apply with their own field manager and `force: true`
on their own list entry. Each manager only touches its own entry.

RBAC for a pod autoscaler:

```yaml
- apiGroups: [""]
  resources: ["pods/resourcehints"]
  verbs: ["get", "patch", "update"]
```

Node autoscalers need nothing new. They already `list` and `watch` pods.

When the feature gate is disabled, the subresource is not served, and the
field is dropped on write unless it is already set on the stored object, per
the usual disabled-field rules.

### Validation

- At most 4 entries. `manager` is a valid domain-prefixed path, at most 63 characters in the name segment and unique within the list.
- Container names in hints must exist in `spec.containers` or `spec.initContainers`. Ephemeral containers are not allowed.
- Quantities are non-negative and resource names are valid standard or extended resource names. Hugepages are allowed.
- `podRequests` requires `spec.resources` to be set.
- `updateTime` is not in the future by more than 5 minutes.
- No cross-check against limits. The writer owns that, since a replacement may get different limits.

### Writer contract

A manager (a pod autoscaler, or any controller that changes pod requests
at admission) that publishes hints:

1. **W1** sets `replacement` to what its own admission path would produce for a new pod from the same owner right now, including startup boost, and including only the containers and resources it controls. VPA can call the same function its webhook uses.
2. **W2** omits `replacement` when it equals the pod's current effective requests within the writer's actuation threshold. Stable pods carry no hint.
3. **W3** rewrites an entry only when a value changes beyond that threshold, or when the writer resizes or evicts the pod. It must not refresh `updateTime` on a timer alone. It SHOULD update promptly when `replacement` increases, because a hint that is too small can still produce the churn loop, while a hint that is too large only makes the node autoscaler more conservative.
4. **W4** removes its entry when it stops managing the pod.

### Consumer contract

A node autoscaler that reads hints. `hinted(pod, field)` is the pod's requests
with `field` merged per container and per resource, combining managers as in
the next section.

1. **C1 Moving a pod.** When simulating eviction of a running pod (consolidation, scale-down, drift, expiration, node repair), size the pod as `hinted(pod, replacement)`. Without `replacement`, keep today's behavior (`max(spec, allocated)`). The result can be smaller than today, which unlocks consolidation when a recreate-only recommendation is smaller. C1 also applies wherever the consumer re-injects pods from nodes it is draining, since those pods come back as replacements.
2. **C2 Pending pods.** Do not apply `replacement` to pending pods, since admission already sized them.
3. **C3 Pods that stay.** Size pods that are not being moved as today, `max(spec, allocated)`. Hints do not change a node's utilization or headroom.
4. **C4 Re-evaluation.** Re-evaluate consolidation when a pod's hints change. Node autoscalers already watch pods, so this is a predicate change, not a new watch.
5. **C5 Fail open.** Missing, invalid, or stale hints mean today's behavior. Consumers must not block on hint availability.
6. **C6 Observability.** Expose which manager's hint changed a decision (event or metric label), so operators can trace a blocked consolidation.

### Combining hints from more than one manager

Take the per-container, per-resource max of `replacement` across managers.

Max is the churn-safe direction. It can cost money when two managers disagree,
which surfaces the misconfiguration instead of hiding it.

### Notes from the prototypes

Prototypes exist for kube-apiserver, VPA (writer), Karpenter and Cluster
Autoscaler (consumers), each as one commit on top of the upstream commit noted:

- kube-apiserver (API, subresource, validation):
  [jigisha620/kubernetes@pod-resource-hints-prototype](https://github.com/jigisha620/kubernetes/tree/pod-resource-hints-prototype), on v1.37.1.
- VPA updater (writer):
  [jigisha620/autoscaler@pod-resource-hints-prototype](https://github.com/jigisha620/autoscaler/tree/pod-resource-hints-prototype), on `97af4eb811ef`.
- Karpenter (consumer):
  [jigisha620/karpenter@pod-resource-hints-prototype](https://github.com/jigisha620/karpenter/tree/pod-resource-hints-prototype), on `6eac93bac87e`.
- Cluster Autoscaler (consumer):
  [jigisha620/cluster-autoscaler@pod-resource-hints-prototype](https://github.com/jigisha620/cluster-autoscaler/tree/pod-resource-hints-prototype), on `c75655b2217b`.

In kwok e2e runs of the startup-boost churn loop, Karpenter without hints made
13 disruptions and created 15 nodes in 6 minutes. With a `replacement` hint it
made 0 disruptions on 1 node in 15 minutes, while still evaluating
consolidation. Cluster Autoscaler with a `replacement` hint removed no nodes
and kept the pod on its node.

The prototypes surfaced these points, now reflected above or to be settled
during API review:

- The subresource must be declared for declarative validation
  (`+k8s:supportsSubresource="/resourcehints"` on `Pod`). Without it every
  write fails with an internal error.
- Typed clients built on an older `k8s.io/api` silently drop
  `status.resourceHints` on decode. A writer that does read-modify-write with
  an old client erases hints. Writers MUST use server-side apply on the
  subresource. The `pods/status` strategy preserves hints, so older kubelets
  and status writers are safe.
- On kwok and in tests, the apiserver only accepts a resize when the node
  reports `containerStatuses[].resources`. Consumer e2e tests need a fake
  kubelet step for that.
- Karpenter does not re-evaluate consolidation when a pod is resized in
  place. C4 fixes this for hint and request changes alike.
- No consumer needed new RBAC, a new watch, or owner resolution. Karpenter
  dropped its VPA module dependency entirely.

### How VPA would write hints

This section is informative. The design is owned by VPA maintainers.

- The **updater** already iterates pods with their matching VPA and holds
  `pods` patch and `pods/resize` RBAC. It computes `replacement` with the
  admission controller's `GetContainersResourcesForPod`, including startup
  boost. It writes after each resize or eviction decision, and when the
  recommendation crosses the existing update priority threshold. This replaces
  the per-pod `vpaCpuStartupBoost` bookkeeping for consumers. VPA can keep the annotation for itself.
- The **recommender** does not need pod RBAC.

### How Karpenter would read hints

Informative. Karpenter already merges per-container predictions over pod
requests in `pkg/state/prediction` and applies them to pods being moved in
`Scheduler.updateCachedPodData`. A hint source plugs into the same seam with no
owner resolution, no CRD, no extra RBAC, and no hydration gate:

- C1 at `updateCachedPodData` for pods in `deletingPodUIDs`, and in
  `buildDaemonOverheadGroups` for new-node DaemonSet overhead.
- C4 by marking the cluster unconsolidated when a pod's hints change, and
  resetting `consolidateAfter` on hint and resize changes.

The VPA source can then be deprecated, as the RFC already plans.

### How Cluster Autoscaler would read hints

Informative. CAS has no pod autoscaler integration today.

- C1 in `simulator.findPlaceFor` when copying a pod for removal simulation, the
  same place `ClearTPURequests` rewrites requests today.
- Capacity buffers with a `scalableRef` size placeholder pods from
  `replacement`.
- Open for CAS maintainers: CAS picks scale-down candidates by utilization
  before it simulates moves, so a node whose pods are large by spec but small
  by `replacement` is never simulated and C1 never runs. Whether candidate
  selection should also consider `replacement` is a CAS design decision.
  Karpenter's consolidation simulates every candidate, so it does not hit this.

### Workload-level hints

Per-pod hints are the alpha deliverable because every node autoscaler already
watches pods, every workload kind (including CRD workloads) produces pods, and
pods in a rolling update can legitimately differ.

Workload-level hints would cut writes from N per workload to one, and would
let a node autoscaler size replicas that do not exist yet (scale from zero,
capacity buffers). Two carriers fit:

1. **Workload API** ([KEP-4671]). Pods already reference a scheduling group
   through `spec.schedulingGroup`. A `resourceHints` list on the Workload's pod
   group template status would apply to every pod in that group, with per-pod
   hints taking precedence. This is blocked on `GenericWorkload` being on by
   default and on Deployment integration ([KEP-6276]).
2. **Scale-like polymorphic subresource.** A `resourcehints` subresource on
   workload kinds, discovered like `/scale`, so CRD workloads can opt in.

Both reuse `PodResourceHint` unchanged. The decision is deferred to beta, after
alpha shows real write volumes.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- `pkg/apis/core/validation`: every validation rule above, including container
  name resolution for init and sidecar containers, and `podRequests` without
  `spec.resources`.
- `pkg/registry/core/pod`: the `resourcehints` strategy only changes
  `status.resourceHints`. `pods/status` and main pod updates preserve it.
  Disabled-gate field dropping.
- `pkg/api/pod`: `DropDisabledPodFields` for the new field.

Coverage targets are set when the implementation PR lands.

##### Integration tests

- `test/integration/pods`: write hints through the subresource with SSA from
  two managers. Both entries survive a kubelet-style `pods/status` update and a
  `pods/resize` call.
- RBAC: a user with only `pods/resourcehints` can write hints, but cannot
  change status or spec. A user with only `pods/status` cannot change hints.
- Feature gate on, write, gate off: the field is preserved on unrelated
  updates and the subresource is not served.

##### e2e tests

- `test/e2e/node` or `test/e2e/apimachinery`: create a pod, publish hints,
  resize the pod in place, and check hints are unchanged and readable. Alpha
  feature tag.

Out of tree, each consumer project carries its own e2e, using kind with kwok
nodes and a hint writer:

- Karpenter: the churn loop from [Motivation](#the-churn-loop) reproduces
  without hints and stops with hints, with the VPA source disabled.
- Cluster Autoscaler: scale-down simulation honors `replacement`.
- VPA: the updater publishes `replacement` for a startup-boosted pod and for a
  recreate-only recommendation.

### Graduation Criteria

#### Alpha

- Feature gate `PodResourceHints`, default off.
- API, subresource, validation, strategy, unit and integration tests.
- At least one writer prototype (VPA updater) and both consumer prototypes
  (Cluster Autoscaler, Karpenter) demonstrating the churn loop fix.

#### Beta

- VPA writes hints in a released version. Both Cluster Autoscaler and
  Karpenter consume them in released versions.
- Write volume measured on a 5000-node, 150k-pod scalability run with VPA
  managing a large share of pods. Decide on workload-level hints.
- UNRESOLVED block settled (status vs spec).
- e2e tests for all subresource operations.

#### GA

- Two releases at beta with no API changes.
- At least two independent writers and both node autoscalers in production use.
- Conformance tests for the subresource.

#### Deprecation

N/A. This KEP adds a new field and subresource and does not deprecate any
existing flag or API.

### Upgrade / Downgrade Strategy

Upgrade: nothing changes until a writer starts publishing. Downgrade to a
version without the field drops hints from storage on the next write of the
object. Consumers fall back to today's behavior (C5).

### Version Skew Strategy

Only kube-apiserver is involved. kubelet does not read or write hints. Kubelets
older than the apiserver send full status updates without the field, and the
`pods/status` strategy preserves stored hints. Node autoscalers built against an
older `k8s.io/api` ignore the field.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodResourceHints`
  - Components depending on the feature gate: kube-apiserver
- [x] Other
  - Writers and consumers each have their own opt-in flag.

###### Does enabling the feature change any default behavior?

No. Nothing writes hints until a writer is configured to.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Stored hints stay until the next write of the pod, but writers can no
longer reach the subresource and consumers fall back to today's behavior.

###### What happens if we reenable the feature if it was previously rolled back?

Hints that were preserved become visible again. Writers rewrite stale entries
on their next change.

###### Are there any tests for feature enablement/disablement?

Yes, integration tests toggle the gate around writes (see Test Plan).

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It cannot affect running workloads directly, since kubelet and scheduler ignore
hints. A buggy writer can make a node autoscaler keep or add nodes. Disable
the writer or delete its entries.

###### What specific metrics should inform a rollback?

`apiserver_request_total{resource="pods",subresource="resourcehints"}` growing
beyond expectations, and `apiserver_request_duration_seconds` for pods. On the
consumer side, node count growth correlated with hints.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Planned for beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. Karpenter's VPA prediction source can be deprecated in Karpenter, which is
out of tree.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

`kubectl get pods -A -o jsonpath='{..resourceHints[*].manager}'`, and the
subresource request metric.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: `status.resourceHints`
- [x] Events
  - Node autoscalers emit events naming the manager whose hint blocked or
    changed a disruption decision (C6).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Existing pod write latency SLOs apply to the subresource.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `apiserver_request_duration_seconds{resource="pods",subresource="resourcehints"}`
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Consumer-side metrics are owned by each node autoscaler.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. Hints are useful only with at least one writer and one consumer, both
optional.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. Writers patch `pods/resourcehints`. Under W2 and W3 a writer patches a pod
only when a hint changes materially, or when it resizes or evicts that pod.
For in-place writers that is at most one extra patch per resize. In
recreate-only modes (VPA `Initial` and `Recreate`), running pods are not
resized, so each recommendation change beyond the threshold rewrites the hint
on every pod of the workload: N writes where VPA writes one VPA object today,
each delivered as a full Pod watch event to every pod watcher. Workload-level
hints remove this at beta. Consumers make no new calls.

###### Will enabling / using this feature result in introducing new API types?

No new kinds. New fields on Pod and one subresource.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Pods with hints grow by about 200 to 600 bytes per manager entry, capped by
validation (4 entries).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No measurable change expected. Watch event volume for pods grows with hint
writes, and is measured at beta.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Slightly larger pod objects in etcd and in every pod informer cache.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Writers cannot publish, and consumers keep their last observed pods. No new
failure mode.

###### What are other known failure modes?

- A stale or inflated hint blocks consolidation.
  - Detection: node autoscaler events naming the manager, and `updateTime`.
  - Mitigations: delete the manager's entry, or fix or disable the writer.
  - Diagnostics: node autoscaler logs at the disruption decision.
  - Testing: consumer e2e.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check the subresource request rate by user agent and throttle or disable the
noisy writer.

## Implementation History

- 2026-06-29: [kubernetes/autoscaler#9898] opened.
- 2026-07-09: Karpenter in-place pod vertical scaling RFC merged ([karpenter#3037]).
- 2026-10-13: Karpenter VPA prediction source merged ([karpenter#3235]).
- 2026-10-16: This KEP drafted.
- 2026-10-17: Prototypes of kube-apiserver, VPA (writer), Karpenter and Cluster Autoscaler (consumers). kwok e2e shows the startup-boost churn loop stopping in both node autoscalers.
- 2026-11-11: Scope reduced to the `replacement` hint after reviewing the
  discussion in [karpenter#3037], [kubernetes/autoscaler#9898] and SIG
  Autoscaling Slack. Dry-run drawbacks verified on a test cluster.

## Drawbacks

- Pod status gains a field that neither kubelet nor scheduler uses. Its only
  purpose is coordination between out-of-tree controllers.
- Per-pod writes, even when limited, scale with pod count rather than workload
  count.
- Writers must reimplement nothing, but they must compute `replacement`
  honestly. A writer that cannot predict its own admission output gains little.

## Alternatives

### Node autoscalers read each pod autoscaler's CRD

What Karpenter does today with VPA. It works, but couples every node
autoscaler to every pod autoscaler's schema and webhook logic, needs
per-CRD RBAC and owner resolution, and has to handle CRD-absent and hydration
cases. It cannot see VPA's boosted value, which is not in VPA status.

### Aggregated "pod recommendation" API

Modeled on `metrics.k8s.io`, favored by VPA maintainers in
[kubernetes/autoscaler#9898]. Only one implementer per cluster, a hard
operational commitment for managed offerings, and a synchronous call on the
node autoscaler's critical path for every simulated pod.

### Pod annotation

For example `autoscaling.x-k8s.io/size-on-eviction`. No schema, no validation,
and it requires `pods` patch, which also allows changing labels and other
annotations. It can serve as an out-of-tree stopgap before this KEP's alpha,
with the same JSON shape as `PodResourceHint`, so consumers can migrate by
changing only where they read from.

### Shared CRD

For example `PodEvictionPrediction` in `autoscaling.x-k8s.io`. Avoids core API
changes, but every consumer needs another watch, owner resolution, and a
dependency on whoever installs the CRD. Since the data is about a pod and every
consumer already watches pods, a pod field is cheaper for everyone.

### Dry-run-only pod create

A node autoscaler submits a dry-run create (`dryRun=All`) of the pod's
replacement, built from the owner's pod template, and reads the admitted
object. Proposed in [karpenter#3037], with a dry-run-only permission suggested
there and in [kubernetes/autoscaler#9898] so the node autoscaler does not need
`create pods`.

Dry-run is the most complete answer for the replacement's size. It runs every
mutating webhook, so it also sees sidecar injection, RuntimeClass overhead,
and labels or tolerations that affect scheduling, and it needs no change from
any pod autoscaler. Cost on the consolidation hot path is manageable: a
background cache keyed by owner, with a fresh dry-run before eviction, keeps
calls to one per owner per refresh.

Its drawbacks:

1. **RBAC.** RBAC does not distinguish dry-run from real creates, so today the
   node autoscaler needs `create pods` cluster-wide. Fixing this needs a
   dry-run-only permission in k/k (see [kubernetes#95449]). Dry-run responses
   can also expose values that webhooks inject into the pod, such as secrets.
2. **ResourceQuota.** Admission counts the dry-run pod in addition to the
   running pod it would replace. In a namespace at its quota the dry-run is
   rejected even though the real replacement fits once eviction frees the old
   pod's usage. The rejection returns only an error, not the mutated pod.
   Multi-tenant clusters often run namespaces close to quota. Reproduced on
   Kubernetes v1.33 with one running 50m pod under a
   `pods: 1, requests.cpu: 100m` quota: a same-size replacement was rejected
   with `exceeded quota: ... requested: pods=1, used: pods=1, limited: pods=1`,
   and with the pod limit raised, a 60m replacement was rejected with
   `requested: requests.cpu=60m, used: requests.cpu=50m, limited:
   requests.cpu=100m`. Both would fit after eviction. A fix needs either quota
   skipped for prediction-only requests, which hides replacements that really
   exceed quota, or "replacing pod X" semantics where quota first subtracts the
   named pod's usage.
3. **Requester identity.** The real replacement is created by the workload
   controller (for example
   `system:serviceaccount:kube-system:replicaset-controller`), the dry-run by
   the node autoscaler. Webhooks and policies that match on the requesting user
   can mutate or reject the two differently. How common such policies are is
   unknown. A dry-run-only endpoint could evaluate the request as the owning
   controller.

Webhooks also receive `dryRun: true` and may legitimately skip work on
dry-run. Mutations that differ on dry-run are allowed but expected to be rare.

The following are not drawbacks:

- **Unavailable fail-closed webhooks.** If a `failurePolicy: Fail` webhook is
  down, the real replacement cannot be created either, so a failed dry-run
  correctly tells the node autoscaler not to evict.
- **Rebuilding the replacement.** The node autoscaler dry-runs the owner's pod
  template, not the running pod, which admission has already mutated. Owners
  without a discoverable template (custom controllers that build pods in code)
  fall back to today's behavior.

Dry-run is complementary to this KEP. It also answers the replacement's shape
(sidecars, labels, tolerations), which hints do not cover. Pod resource hints
are computed by the writer without creating a pod, so they do not hit quota or
requester-identity divergence and need no new permission model.

### Replacement PodTemplate owned by the workload controller

Proposed by @x13n. Predicts the full spec, but requires every admission webhook
to mutate templates as well as pods, and a signal when admission logic changes.

## Infrastructure Needed (Optional)

None.

[kubernetes/autoscaler#9898]: https://github.com/kubernetes/autoscaler/issues/9898
[kubernetes#136043]: https://github.com/kubernetes/kubernetes/pull/136043
[karpenter#2346]: https://github.com/kubernetes-sigs/karpenter/issues/2346
[karpenter#3037]: https://github.com/kubernetes-sigs/karpenter/pull/3037
[karpenter#3134]: https://github.com/kubernetes-sigs/karpenter/pull/3134
[karpenter#3235]: https://github.com/kubernetes-sigs/karpenter/pull/3235
[KEP-2837]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2837-pod-level-resource-spec
[KEP-4671]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling
[KEP-5836]: https://github.com/kubernetes/enhancements/issues/5836
[KEP-5972]: https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/5972-dynamic-containers
[KEP-6276]: https://github.com/kubernetes/enhancements/issues/6276

[kubernetes#95449]: https://github.com/kubernetes/kubernetes/issues/95449
