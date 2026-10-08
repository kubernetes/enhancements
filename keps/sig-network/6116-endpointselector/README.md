# KEP-6116: EndpointSelector

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Service EndpointSelector Projection](#service-endpointselector-projection)
  - [Controller-Managed Creation](#controller-managed-creation)
  - [User-Managed Creation](#user-managed-creation)
  - [User Stories](#user-stories)
    - [Story 1: InferencePool Implementation Simplification](#story-1-inferencepool-implementation-simplification)
    - [Story 2: Controller-Managed Endpoints Without Redundant Pod Watching](#story-2-controller-managed-endpoints-without-redundant-pod-watching)
    - [Story 3: Referenceable Backend Selection for an Existing Service](#story-3-referenceable-backend-selection-for-an-existing-service)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Security](#security)
    - [Control Plane Load](#control-plane-load)
    - [Orphaned Resources](#orphaned-resources)
    - [API Confusion](#api-confusion)
- [Design Details](#design-details)
  - [Current State](#current-state)
  - [Proposed Implementation](#proposed-implementation)
  - [API Definition](#api-definition)
  - [EndpointSlice-Controller Changes](#endpointslice-controller-changes)
  - [New Service EndpointSelector Controller](#new-service-endpointselector-controller)
  - [Metadata Propagation](#metadata-propagation)
  - [Service Projection Mapping](#service-projection-mapping)
  - [Edge Cases and Deferred Design Decisions](#edge-cases-and-deferred-design-decisions)
  - [Controller-Managed Conventions](#controller-managed-conventions)
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
    - [API server or etcd unavailable](#api-server-or-etcd-unavailable)
    - [Explicit EndpointSelector does not update EndpointSlices after a pod readiness change](#explicit-endpointselector-does-not-update-endpointslices-after-a-pod-readiness-change)
    - [Service-derived EndpointSelector projection is absent or stale](#service-derived-endpointselector-projection-is-absent-or-stale)
    - [Service-derived EndpointSelector projection name collision](#service-derived-endpointselector-projection-name-collision)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Extend Service.spec.selector to Support matchExpressions](#extend-servicespecselector-to-support-matchexpressions)
  - [Manual EndpointSlice Management](#manual-endpointslice-management)
  - [Shadow Service (Headless Service to Generate EndpointSlices)](#shadow-service-headless-service-to-generate-endpointslices)
  - [Broader Service Decomposition](#broader-service-decomposition)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
  SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for
    [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806)
    must be hit by
    [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to
  mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

`EndpointSlices` provide a scalable way to track (typically in-cluster) network
endpoints in Kubernetes, providing addressing, health, and topology information
to consumers. The practical interface for managing the lifecycle of
`EndpointSlices` at scale is the `Service` resource, which includes additional
functionality (ClusterIP/VIP, DNS hostname, kube-proxy load balancing) that is
unwanted in many use cases. The `EndpointSelector` resource allows users and
controllers to declare a pod selector and a set of ports; the
`endpointslice-controller` manages the corresponding `EndpointSlices`.

## Motivation

Today, a user or controller that wants a set of `EndpointSlices` for a set of
pods matching a label selector has two options:

1. Create a `Service`.
2. Create and update `EndpointSlices` manually as pods matching that selector
   spin up and down.

Option 2 tends to be avoided when possible: a single pod status change (for
example, Running, Terminating, or Ready) can trigger writes to every
`EndpointSlice` for the affected workload across a large cluster. The Kubernetes
control plane already performs this work for `EndpointSlices` originating from a
`Service`; the same mechanism should be available to resources that do not need
`Service` semantics. For lack of a scoped API, many controllers settle for
option 1, creating a `Service` whenever they need `EndpointSlices`. The
following resources from across the Kubernetes ecosystem demonstrate
label-selection and endpoint-management logic being implemented independently:

- Istio `ServiceEntry`: `spec.workloadSelector` selects Kubernetes pods and VM
  workloads as service endpoints.
  - [API](https://github.com/istio/api/blob/master/networking/v1alpha3/service_entry.proto)
  - [Documentation](https://istio.io/latest/docs/reference/config/networking/service-entry/)
- Cilium `CiliumLocalRedirectPolicy`:
  `spec.redirectBackend.localEndpointSelector` selects node-local backend pods
  for redirected traffic.
  - [API](https://github.com/cilium/cilium/blob/main/pkg/k8s/apis/cilium.io/v2/clrp_types.go)
  - [Documentation](https://docs.cilium.io/en/stable/network/kubernetes/local-redirect-policy/)
- Gateway API [`XBackend` (experimental)][gep-4894] defines
  `EndpointSelector` with a pod `LabelSelector` for backend endpoints.

These examples are evidence of duplicated workload-selection logic, not
proposed direct migrations. Their additional behavior—Cilium's node-local
redirect handling and Istio's VM endpoint handling—remains outside this KEP's
scope.

Furthermore, `Service.spec.selector` is an equality-based map and an empty
selector represents manually managed endpoints. A dedicated resource can use a
`metav1.LabelSelector` and support set-based `matchExpressions` without changing
those Service semantics. This has been a longstanding community request (see
kubernetes/kubernetes#48528 and kubernetes/kubernetes#62795).

### Goals

- Allow users and controllers to create `EndpointSlices` for a set of pods
  matching a label selector without creating a `Service`.
- Extend `endpointslice-controller` to watch `EndpointSelector` objects in
  addition to `Service` objects.
- Create a referenceable `EndpointSelector` projection for each `Service` with
  a pod selector, without changing the Service EndpointSlice reconciliation
  path.
- Support `matchLabels` and `matchExpressions` for pod selection.

### Non-Goals

- Changing `Service` as an EndpointSlice source.
- Extending `endpointslice-controller` to read arbitrary resources with pod
  selectors.
- Selecting DRA `ResourceClaims` or DRA-allocated secondary interfaces as
  endpoints.
- Defining Multi-Network Service behavior or semantics.

## Proposal

This KEP introduces `EndpointSelector`[^1], a namespace-scoped resource that
allows users and controllers to obtain a managed set of `EndpointSlices` for a
pod label selector without creating a `Service`. `Service` and
`EndpointSelector` are independent inputs to `endpointslice-controller`: each
has its own EndpointSlice reconciliation and ownership model.

The `EndpointSelector` spec is limited to backend selection and EndpointSlice
port metadata: `selector`, target ports, IP families, and application protocol.
It contains no Service frontend or policy configuration, such as ClusterIP, DNS,
topology hints, or readiness overrides.

[^1]: Alternative names considered: `EndpointGroup`, `EndpointPool`

`EndpointSelector` objects are created in one of three ways:

### Service EndpointSelector Projection

`service-endpointselector-controller` creates an `EndpointSelector` projection
for each `Service` with a pod selector. The projection exposes the Service's
backend selector, target ports, and IP families for controllers that want to
reference that selection independently of the Service API.

The projection has the same namespace and name as its Service, giving consumers
a stable reference. This name is reserved only by the projection controller's
ownership check: a pre-existing `EndpointSelector` with that name is a
projection only when its controller owner reference identifies that exact
Service UID. A user-managed object with the same name is never adopted,
modified, or deleted.

The projection is not an input to Service EndpointSlice reconciliation. The
existing Service path continues to create and own Service EndpointSlices; the
projection cannot delay, alter, or duplicate them.

### Controller-Managed Creation

A third-party controller creates an `EndpointSelector` in response to a
higher-level resource (for example, an `InferencePool`). The controller is
responsible for its own lifecycle management. Setting `ownerReferences` to the
parent resource enables automatic garbage collection when the parent is
deleted. Using `generateName` instead of `name` avoids naming conflicts when
multiple controllers target the same workload.

### User-Managed Creation

A user creates an `EndpointSelector` directly (for example, via `kubectl`) and
manages its lifecycle explicitly. No `ownerReference` is required. The
resource is referenced by name from higher-level objects (for example, a
`Backend` resource in Gateway API). This model suits cases where the
`EndpointSelector` outlives any single parent or where multiple consumers share
the same set of endpoints. A cross-namespace reference does not inherently
grant the referencing resource permission to read the `EndpointSelector`, the
pods it selects, or the `EndpointSlices` it produces; appropriate RBAC rules
must be in place.

### User Stories

#### Story 1: InferencePool Implementation Simplification

As a platform operator running AI inference workloads, I create an
`InferencePool` targeting my model-serving pods by label. The `InferencePool`
controller creates a corresponding `EndpointSelector`, which the
`endpointslice-controller` uses to produce `EndpointSlices` reflecting live pod
readiness. My gateway's endpoint picker consumes those slices directly —
without any kube-proxy round-robin in front of it — which is what makes
per-request model routing viable.

#### Story 2: Controller-Managed Endpoints Without Redundant Pod Watching

As a controller author, my CRD needs to track the endpoints of the pods it
manages. Rather than watching pods, tracking readiness transitions, and managing
`EndpointSlice` packing — logic the `endpointslice-controller` already owns —
I create an `EndpointSelector` with `ownerReferences` pointing to my resource
and let the `endpointslice-controller` handle the rest. My controller only
needs to create and delete the `EndpointSelector`; the endpoint lifecycle is
not my problem.

#### Story 3: Referenceable Backend Selection for an Existing Service

As an app developer, I have an existing `Service` and need another controller
to reference the same backend selection. I reference the Service-derived
`EndpointSelector` from a [Gateway API `XBackend`][gep-4894], while declaring
connection settings such as TLS and protocol on that Backend. The Service keeps
its own routing and EndpointSlice lifecycle; the projection gives the Backend a
stable backend-selection contract without adding those settings to Service.

[gep-4894]: https://gateway-api.sigs.k8s.io/geps/gep-4894/

### Notes/Constraints/Caveats

`EndpointSelector` and `Service` overlap in backend selection but are sibling
APIs. The EndpointSlice controller reconciles Service EndpointSlices from
Service and EndpointSelector EndpointSlices from explicitly created
EndpointSelectors. A Service-derived projection is referenceable selection
metadata; it is not substituted into either reconciliation path.

An `EndpointSelector` is loosely coupled to its consumers. It has no awareness
of which higher-level resources reference it, just as `EndpointSlices` have no
awareness of which consumers watch them. This mirrors the existing
`Service` → `EndpointSlice` relationship and scales for the same reasons: a
single `EndpointSelector` can be referenced by multiple resources, and those
resources can come and go without coordinating through the `EndpointSelector`
itself.

### Risks and Mitigations

#### Security

`EndpointSelector` is namespace-scoped. The `endpointslice-controller` only
selects pods within the same namespace as the `EndpointSelector`, enforced at
reconciliation time the same way it is for `Service`-owned slices. RBAC for
creating `EndpointSelectors` follows the same model as `Services`: namespace
admins can create them; cluster-level restrictions apply through standard
mechanisms. `NetworkPolicy` continues to apply to the selected pods regardless
of whether their `EndpointSlices` originated from a `Service` or an
`EndpointSelector`.

A controller that needs managed Pod endpoints can be granted permission to
create `EndpointSelectors` without direct `EndpointSlice` write permission. The
controller is thereby limited to publishing same-namespace, Pod-derived
endpoints rather than arbitrary addresses, allowing administrators to reduce
the direct `EndpointSlice` write access described in
[kubernetes/kubernetes#103675](https://github.com/kubernetes/kubernetes/issues/103675).

#### Control Plane Load

The per-object reconciliation cost of an explicitly created `EndpointSelector`
is equivalent to a `Service` with the same selector. The Service projection adds
one object write per selector-based Service and a controller cache, but it does
not add EndpointSlice reconciliation or alter the Service endpoint path.

#### Orphaned Resources

Controller-managed `EndpointSelectors` that lack `ownerReferences` will persist
after the owning resource is deleted, along with the `EndpointSlices` they
produced. Operators can identify orphaned resources by inspecting
`ownerReferences` or by using controller-specific labels applied by the
managing controller.

#### API Confusion

Introducing a resource that partially overlaps with `Service` risks confusion
about which to use when. `Service` remains the API for a frontend, DNS, and
kube-proxy semantics. `EndpointSelector` is the API for managed backend
selection where those Service semantics are unwanted or actively harmful.

## Design Details

### Current State

Today, the `endpointslice-controller` reads changes to `Service` and `Pod`
objects to manage Service-owned `EndpointSlices`. When a `Service` with a pod
selector is created or updated, the controller creates or updates
`EndpointSlices` to reflect matching pods. A `Service` with a nil pod selector
or type `ExternalName` is ignored. When a pod changes status, the controller
updates the relevant EndpointSlices; when a Service is deleted, it garbage
collects the slices it owns.

### Proposed Implementation

When the `EndpointSelector` feature gate is enabled, the
`endpointslice-controller` adds an `EndpointSelector` watch while retaining its
existing `Service` watch. Service and EndpointSelector events enqueue their own
reconciliation keys and produce only their respective EndpointSlices. A
Service-derived projection is excluded from EndpointSelector EndpointSlice
reconciliation, preventing duplicate slices.

The `service-endpointselector-controller` watches selector-based Services and
maintains their projections asynchronously. Services without a pod selector do
not get a projection. A failed projection write is retried independently and
does not affect Service EndpointSlice reconciliation.

### API Definition

```go
// EndpointSelector is a namespace-scoped resource in discovery.k8s.io/v1alpha1 that
// declares a pod label selector and a set of ports. The
// endpointslice-controller manages the corresponding EndpointSlices.
type EndpointSelector struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec EndpointSelectorSpec `json:"spec,omitempty"`
  // Status is intentionally omitted. A status design is left for a follow-on
  // KEP once consumption patterns across controller types are established.
}

type EndpointSelectorSpec struct {
  // Selector selects the pods whose addresses are tracked by this resource. It
  // must not be empty. Both matchLabels and matchExpressions are supported.
  // The selector is mutable; changing it retargets the managed EndpointSlices
  // in the same way changing Service.spec.selector does today.
  Selector metav1.LabelSelector `json:"selector"`

  // IPFamilies specifies the IP families for which EndpointSlices should be
  // produced. Defaults to all address families present in matching pod
  // addresses, producing one EndpointSlice addressType per family found (IPv4,
  // IPv6, or both in a dual-stack cluster). Set this field to restrict output
  // to a specific family.
  // +optional
  IPFamilies []corev1.IPFamily `json:"ipFamilies,omitempty"`

  // Ports defines the endpoint target ports and protocols exposed on the
  // selected pods. Omitting this field produces EndpointSlices with an empty
  // ports list.
  // +optional
  Ports []EndpointSelectorPort `json:"ports,omitempty"`
}

// EndpointSelectorPort defines a single port exposed by the selected pods.
type EndpointSelectorPort struct {
  // Name is a human-readable identifier for this port. Must match the
  // corresponding container port name if one exists.
  // +optional
  Name string `json:"name,omitempty"`

  // Protocol is the IP protocol for this port (TCP, UDP, or SCTP).
  // Defaults to TCP.
  // +optional
  Protocol corev1.Protocol `json:"protocol,omitempty"`

  // TargetPort is the target port number or name. Numeric values are used directly;
  // string values are resolved per-pod by the endpointslice-controller by
  // matching against pod.spec.containers[].ports[].name.
  TargetPort intstr.IntOrString `json:"targetPort"`

  // AppProtocol is the application protocol for this port. The
  // endpointslice-controller copies it to the corresponding EndpointSlice port.
  // +optional
  AppProtocol *string `json:"appProtocol,omitempty"`
}

```

A manually created `EndpointSelector`:

```yaml
apiVersion: discovery.k8s.io/v1alpha1
kind: EndpointSelector
metadata:
  name: my-inference-pool-endpoints
  namespace: default
spec:
  selector:
    matchLabels:
      app: my-model-server
  ports:
    - name: grpc
      targetPort: 8080
      protocol: TCP
      appProtocol: kubernetes.io/grpc
---
apiVersion: gateway.networking.x-k8s.io/v1alpha1
kind: XBackend
metadata:
  name: my-backend
  namespace: default
spec:
  type: EndpointSelector
  port: 80
  endpointSelector:
    selectorRef:
      name: my-inference-pool-endpoints
      namespace: default
```

A controller-managed `EndpointSelector` created by an `InferencePool`
controller with garbage-collection metadata:

```yaml
apiVersion: discovery.k8s.io/v1alpha1
kind: EndpointSelector
metadata:
  generateName: my-inference-pool-
  namespace: default
  ownerReferences:
    - apiVersion: inference.networking.k8s.io/v1
      kind: InferencePool
      name: my-inference-pool
      uid: "<uid>"
      controller: true
      blockOwnerDeletion: true
spec:
  selector:
    matchLabels:
      app: my-model-server
  ports:
    - name: grpc
      targetPort: 8080
      protocol: TCP
```

### EndpointSlice-Controller Changes

The `endpointslice-controller` retains its Service reconciliation and adds an
independent EndpointSelector reconciliation path. The two paths share internal
pod discovery and EndpointSlice packing logic where appropriate, but neither
resource is translated into the other for EndpointSlice production.

Service-owned EndpointSlices retain their current owner references, labels, and
metadata propagation. EndpointSlices produced for an explicitly created
EndpointSelector are owned by that EndpointSelector and carry the
`kubernetes.io/endpoint-selector-name` label. The controller does not infer
EndpointSlice ownership from an EndpointSelector's owner reference.

The EndpointSelector informer and workers are independent from the existing
Service informer and workers. EndpointSelector cache sync must not be a
prerequisite for starting Service reconciliation. If kube-controller-manager has
the feature gate enabled but kube-apiserver does not serve the resource, the
EndpointSelector path logs and retries its discovery and informer setup while
the Service path continues normally.

### New Service EndpointSelector Controller

A new `service-endpointselector-controller` in `pkg/controller` creates and
deletes an `EndpointSelector` for each `Service` with a pod selector. The
projection is owned by its Service and deleted when the Service is deleted. The
controller is enabled by the `EndpointSelector` feature gate in
`kube-controller-manager`.

The controller reconciles projections using plain `Create`/`Update`/`Delete`
API calls. It owns projection fields derived from Service and restores those
fields after direct edits. Projection writes are asynchronous and independent
from the Service EndpointSlice controller.

Before updating or deleting a same-name EndpointSelector, the controller must
verify that its controller owner reference identifies the Service by name and
UID. If the object has no such owner reference, it is a name collision: the
controller emits a warning Event on the Service and leaves the object unchanged.
It must not adopt, overwrite, or garbage collect that object. The projection is
unavailable until the collision is removed, but Service EndpointSlices continue
to reconcile directly from the Service.

### Metadata Propagation

Service reconciliation continues to copy Service labels to Service-owned
EndpointSlices. For an explicitly created EndpointSelector, non-reserved labels
on the EndpointSelector are copied to its EndpointSlices. The controller owns
`kubernetes.io/endpoint-selector-name` and other reserved EndpointSlice labels.
This KEP does not introduce general annotation propagation.

For each EndpointSelector port, the controller copies `appProtocol` to the
corresponding EndpointSlice port. This preserves application-protocol metadata
for EndpointSlice consumers of explicitly created EndpointSelectors.

### Service Projection Mapping

The projection makes backend-selection data referenceable while leaving Service
EndpointSlice behavior on its existing path.

| Service input | Projection behavior |
| :--- | :--- |
| `Service.spec.selector == nil` or `Service.type == ExternalName` | No projection is created. |
| Non-nil `Service.spec.selector` | The projection uses equivalent `matchLabels`. |
| `Service.spec.selector` updated | The projection updates; Service EndpointSlices continue to reconcile directly from Service. |
| `Service.spec.ports[].targetPort` | The projection copies the target port verbatim. |
| `Service.spec.ports[].appProtocol` | The projection copies the corresponding application protocol. |
| `Service.spec.ipFamilies` | The projection copies the requested IP families. |
| Projection identity | The projection has the Service's namespace and name and a controller owner reference to the Service UID. |
| Same-name user-managed `EndpointSelector` | The controller reports a collision and leaves the object unchanged; no projection is created. |
| `Service` deletion | The projection is garbage collected; Service EndpointSlices follow their existing lifecycle. |

### Edge Cases and Deferred Design Decisions

An `EndpointSelector` with an empty selector is invalid. Empty
`metav1.LabelSelector` normally means "match everything", but that is too easy
to create accidentally for an API that directly publishes pod endpoints.
Validation rejects a selector with no `matchLabels` and no `matchExpressions`.
For `Service`, this KEP preserves existing behavior: a nil selector opts the
`Service` out of derived `EndpointSelector` creation.

`spec.selector` uses `metav1.LabelSelector` as a value type (not a pointer),
which produces a non-nullable field in OpenAPI and makes the
required-selector constraint clearer. Both `matchLabels` and
`matchExpressions` are supported. The value-type decision should be confirmed
with SIG Network before moving to implementable, as a pointer type would change
validation behavior.

`spec.selector` is mutable for explicitly created `EndpointSelector` objects.
Changing it retargets their managed `EndpointSlices`. For Service-derived
projections, the service-endpointselector-controller restores selector edits
from the Service.

Set-based selection (`matchExpressions`) for `Service` users is a separate
concern. `Service.spec.selector` remains `map[string]string`; how `Service`
users opt into set-based selection (for example, via a future
`Service.spec.selectorRef`) is deferred to a follow-on KEP and is not part of
this proposal.

`EndpointSelectorPort.targetPort` is typed as `intstr.IntOrString`. The
endpointslice-controller resolves named target ports per Pod when generating
EndpointSlices by matching `pod.spec.containers[].ports[].name`.

`EndpointSelectorPort.appProtocol` has the same semantics as
`EndpointSlicePort.appProtocol`; the endpointslice-controller copies it to
generated EndpointSlices without interpreting it.

### Controller-Managed Conventions

The `service-endpointselector-controller` uses the Service's namespace and name
for its projection. It recognizes a projection only by a matching controller
owner reference, including the Service UID; it never adopts a same-name object
that lacks that owner reference.

For third-party controllers that create `EndpointSelector` objects, suggested
practices include:

- Setting `ownerReferences` to the parent resource so that Kubernetes garbage
  collection removes the `EndpointSelector` when the parent is deleted.
- Using `generateName` rather than `name` to avoid naming conflicts when
  multiple controllers may target the same workload.

### Test Plan

[X] I/we understand the owners of the involved components may require updates
to existing tests to make this code solid enough prior to committing the
changes necessary to implement this enhancement.

##### Prerequisite testing updates

Existing endpointslice-controller tests cover Service reconciliation. New tests
must preserve that coverage and add EndpointSelector reconciliation coverage
without changing the Service test contract.

##### Unit tests

The following packages will be modified or created for Alpha. Coverage
percentages will be filled in before the release is targeted.

- `k8s.io/kubernetes/pkg/controller/endpointslice`: `<date>` - `<coverage>`
- `k8s.io/kubernetes/pkg/controller/serviceendpointselector` (new):
  `<date>` - `<coverage>`
- `k8s.io/kubernetes/pkg/apis/discovery/validation`: `<date>` - `<coverage>`
- `k8s.io/kubernetes/pkg/registry/discovery/endpointselector` (new):
  `<date>` - `<coverage>`

| Test description | Expected result |
| :--- | :--- |
| `EndpointSelector` created with `matchLabels` selector | `EndpointSlices` generated for matching pods |
| `EndpointSelector` created with empty selector | Validation rejects at admission |
| `EndpointSelector` created with `matchExpressions` selector | `EndpointSlices` generated for pods matching the expression |
| Manually created `EndpointSelector` selector updated | `EndpointSlices` retargeted to the newly matching pods |
| Pod with labels matching an `EndpointSelector` added | Address appears in `EndpointSlice` |
| Pod with labels matching an `EndpointSelector` deleted | Address removed from `EndpointSlice` |
| Pod transitions from not-ready to ready | Endpoint `ready` and `serving` conditions updated to true |
| Pod transitions from ready to not-ready | Endpoint `ready` and `serving` conditions updated to false |
| `EndpointSelector` deleted | Owned `EndpointSlices` garbage collected |
| `EndpointSelector` with non-reserved labels | Generated `EndpointSlices` carry those labels |
| `EndpointSelector` with reserved EndpointSlice labels | Generated `EndpointSlices` use controller-owned reserved label values |
| `EndpointSelector` with `appProtocol` | Generated `EndpointSlice` ports carry the same `appProtocol` |
| Explicitly created `EndpointSelector` with an owner reference | `EndpointSlice` ownerRef points to `EndpointSelector` |
| `EndpointSlice` produced from an `EndpointSelector` | Carries `kubernetes.io/endpoint-selector-name` label |
| `Service` with pod selector created | `service-endpointselector-controller` creates a corresponding `EndpointSelector` |
| `Service` with nil pod selector created | No `EndpointSelector` created |
| `Service` selector updated | Corresponding `EndpointSelector` updated to match |
| User edits projection fields derived from `Service` | `service-endpointselector-controller` restores the Service-derived values |
| User creates same-name `EndpointSelector` without the Service owner reference | Controller emits a collision Event and does not modify or delete the user-managed object |
| `Service` with named `targetPort` created | Projection keeps the target port value unchanged |
| `Service` with `appProtocol` created | Projection keeps the corresponding application protocol |
| `Service` with pod selector deleted | Corresponding `EndpointSelector` deleted |
| Feature gate disabled, `EndpointSelector` object submitted | API server rejects |
| Feature gate disabled, controllers running | `EndpointSelector` objects not reconciled |

##### Integration tests

- `EndpointSelector` created → matching pods reflected in `EndpointSlice`
  within controller sync period.
- Pod readiness transitions (ready → not-ready → ready) reflected in
  `EndpointSlice` conditions within the expected latency bound.
- `EndpointSelector` deleted → all owned `EndpointSlices` garbage collected.
- `Service` with pod selector created → `service-endpointselector-controller`
  creates a corresponding projection without creating duplicate EndpointSlices.
- Direct edits to projection fields derived from Service are reconciled back to
  the Service-derived values.
- `Service` deleted → corresponding projection deleted; Service EndpointSlices
  follow their existing lifecycle.
- Feature gate toggled off → API server rejects new `EndpointSelector` objects;
  existing `Service`-backed `EndpointSlices` remain current.
- kube-controller-manager gate enabled while kube-apiserver does not serve
  `EndpointSelector` → Service-owned EndpointSlices continue to reflect Pod
  readiness; the EndpointSelector path remains unavailable and retries without
  blocking Service workers.
- Feature gate toggled off for an explicit EndpointSelector consumer → its
  EndpointSlices remain but stop updating; the test verifies the documented
  stale-endpoint behavior and recovery after re-enablement.
- A user-managed EndpointSelector collides with a Service projection name → the
  object is unchanged, the Service receives a collision Event, and Service
  EndpointSlices remain current.
- Feature gate toggled off then on → controllers resume reconciliation; drifted
  `EndpointSlices` return to sync without manual intervention.

Links will be added once test files are created in `kubernetes/kubernetes`:
[integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=EndpointSelector),
[triage search](https://storage.googleapis.com/k8s-triage/index.html?test=EndpointSelector)

##### e2e tests

- `EndpointSelector` created in a live cluster → `EndpointSlices` exist and
  reflect matching pods by name, IP, and endpoint readiness conditions.
- `Service` with a pod selector auto-creates a corresponding `EndpointSelector`
  projection without creating duplicate EndpointSlices.
- Controller-managed `EndpointSelector` with an `ownerReference` is garbage
  collected when the owning resource is deleted.

Links will be added once test files are created in `kubernetes/kubernetes`:
[SIG Network](https://testgrid.k8s.io/sig-network?include-filter-by-regex=EndpointSelector),
[triage search](https://storage.googleapis.com/k8s-triage/index.html?test=EndpointSelector)

### Graduation Criteria

#### Alpha

- `EndpointSelector` API type in `discovery.k8s.io/v1alpha1` defined,
  registered, and validated (including `matchExpressions` and empty selector
  rejection).
- `endpointslice-controller` reconciles `EndpointSelector` objects while
  retaining Service reconciliation.
- `service-endpointselector-controller` creates and deletes an
  `EndpointSelector` for each `Service` with a pod selector.
- Open questions from the Design Details resolved before moving to
  implementable: projection naming, projection field ownership, orphaned
  resource enforcement, and name validation.
- Unit and integration tests covering: `EndpointSelector` creation →
  `EndpointSlice` generation; pod readiness condition transitions;
  `EndpointSelector` deletion → `EndpointSlice` garbage collection;
  EndpointSelector-to-EndpointSlice label propagation; Service projection
  mapping; projection edits reconciled back to Service-derived values; feature
  gate off → API server rejects `EndpointSelector` objects.
- Basic e2e tests enabled (not required to be in Testgrid for Alpha).

#### Beta

- Feedback gathered from Alpha adopters.
- Open questions from Design Details resolved or explicitly deferred with
  written justification.
- All known Alpha issues and gaps resolved.
- Monitoring requirements defined and implemented (metrics in
  `kube-controller-manager` exposing `EndpointSelector` reconciliation
  activity).
- Downgrade tests and scalability benchmarks complete.
- All tests in Testgrid and linked in this KEP.

#### GA

- Sustained real-world adoption across multiple independent consumers (at
  minimum: `InferencePool` and one Gateway API implementation).
- `InferencePool` shadow-`Service` workaround pattern officially deprecated
  in documentation.
- All Beta feedback resolved.
- Conformance tests added — `EndpointSelector` behavior is not optional.
- Minimum two-release window since Beta.

### Upgrade / Downgrade Strategy

**Enabling the feature gate.** The EndpointSelector API and controller are
enabled together. The EndpointSlice controller retains Service reconciliation
throughout the rollout. Once the EndpointSelector API is available,
explicitly-created EndpointSelectors begin reconciliation and the
service-endpointselector-controller begins creating Service projections.

**Disabling the feature gate.** The API server stops accepting new
`EndpointSelector` objects and the controllers stop reconciling them. Existing
`EndpointSelector` objects remain in etcd but are ignored. Service EndpointSlice
reconciliation continues unchanged. EndpointSlices owned by explicitly created
EndpointSelectors stop being updated until the gate is re-enabled.

**Re-enabling after rollback.** The controllers resume reconciliation. The
`service-endpointselector-controller` syncs all `Services` and creates any
missing projections. The endpointslice-controller reconciles explicitly created
EndpointSelectors, bringing their drifted EndpointSlices back into sync without
manual intervention.

### Version Skew Strategy

Kubernetes requires `kube-apiserver` to be upgraded before
`kube-controller-manager`, which means a new `kube-controller-manager` will
always run against an apiserver that is at least its version. Under the standard
upgrade order, if the `EndpointSelector` feature gate is enabled by default on
a given release, both the API and the controller can assume the API is
available. During the off-by-default stage, the administrator is responsible
for enabling the feature gate on both `kube-controller-manager` and
`kube-apiserver`.

**n-1 controller-manager (old controller, new API server).** The old
`kube-controller-manager` does not know about `EndpointSelector`. It continues
to watch `Service` objects and produce `EndpointSlices` from them as today.
`EndpointSelector` objects may exist on the new API server if the gate is
enabled there, but nothing reconciles them until the controller-manager is
upgraded.

**kubelet and kube-proxy** are not involved in `EndpointSelector`
reconciliation. They consume `EndpointSlices` regardless of whether those
slices were produced by a `Service` or an `EndpointSelector`, and are not
affected by version skew in either direction.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `EndpointSelector`
  - Components that will depend on the feature gate:
    - `kube-apiserver` (to accept EndpointSelector objects)
    - `kube-controller-manager` (to reconcile EndpointSelector objects)

###### Does enabling the feature change any default behavior?

Yes. In addition to making the `EndpointSelector` API available, enabling the
feature will cause `service-endpointselector-controller` to create a
non-authoritative `EndpointSelector` projection for each `Service` with a pod
selector. These projections will let other controllers reference a Service's
backend selection without using `Service` as their contract.

The existing `endpointslice-controller` will continue to reconcile Service-owned
`EndpointSlices` directly from `Service`; Service-derived `EndpointSelector`
objects will not produce another set of `EndpointSlices`. Consequently, enabling
the feature will not change Service endpoint availability, EndpointSlice
contents, or the Service-to-EndpointSlice reconciliation path.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, but only after consumers that depend on EndpointSlices from explicitly
created `EndpointSelectors` have been migrated to another endpoint source or
been stopped. Setting the `EndpointSelector` feature gate to false will stop the
API server from accepting new EndpointSelector objects and will stop the
controllers from reconciling them. Existing EndpointSelector objects and their
EndpointSlices will remain in etcd but will not be acted upon. Their
EndpointSlices can therefore retain stale Pod IPs, and consumers may route to
deleted or replaced Pods.

Service-owned EndpointSlices will remain current because their controller path
will continue to reconcile directly from Service. Service-derived projections
may remain stale while the gate is disabled, but will not affect Service endpoint
availability.

###### What happens if we reenable the feature if it was previously rolled back?

The controllers will resume reconciliation. The
`service-endpointselector-controller` will sync all Services and create or
update missing and stale projections. The `endpointslice-controller` will
reconcile explicitly created EndpointSelector objects, bringing their drifted
EndpointSlices back into sync. No manual intervention will be required.

###### Are there any tests for feature enablement/disablement?

Alpha integration tests will cover:

- API server rejection of `EndpointSelector` objects when the gate is disabled.
- `endpointslice-controller` skipping reconciliation of `EndpointSelector`
  objects when the gate is disabled.
- Service-owned `EndpointSlices` will continue to reflect Pod readiness changes
  while the gate is enabled, disabled, and re-enabled.
- Service-derived projections will be created without producing duplicate
  `EndpointSlices`.
- `EndpointSlices` written for explicitly created `EndpointSelector` objects
  will become stale (not deleted) when the gate is disabled, and return to sync
  when re-enabled. The test will verify that rollback requires direct consumers
  to be migrated or stopped first.
- kube-controller-manager with the gate enabled and kube-apiserver with the
  gate disabled: Service reconciliation will start and remain healthy while the
  EndpointSelector informer is unavailable.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Full analysis will be completed at Beta. Expected failure modes include:

- Skewed rollout: in HA clusters where some API servers have the gate enabled
  and others do not, EndpointSelector requests that reach an old apiserver
  will return 404. The EndpointSelector path will retry discovery and informer
  setup, but its cache sync will not gate Service workers. Service-owned
  EndpointSlices will remain current because they will continue to be reconciled
  directly from Service. Explicit EndpointSelector consumers will be unavailable
  until the resource is
  served reliably by the API server fleet.
- Rollback with direct consumers: disabling the gate leaves EndpointSlices from
  explicitly created EndpointSelectors present but stale. Consumers may route
  to deleted or replaced Pod IPs until the gate is re-enabled. Operators will
  migrate those consumers to another endpoint source or stop them before
  rollback; Service consumers will be unaffected.
- Service projection writes may be rejected by quota, admission, or RBAC. The
  service-endpointselector-controller will retry the write; until it succeeds,
  the derived `EndpointSelector` will be absent or stale for consumers that
  reference it. This will not interrupt EndpointSlice updates for the Service.
- Object volume burst: clusters with many `Services` will generate a large number of
  `EndpointSelector` create calls when the gate is first enabled. The
  `service-endpointselector-controller` will be the sole writer and will be
  rate-limited by the standard client-go work queue, but the burst may still be
  visible in API server metrics.
- Mid-reconciliation restart: a `kube-controller-manager` restart during the
  initial projection sync may leave a window where some Services do not yet have
  a corresponding EndpointSelector. The controller will be idempotent: it will
  create missing projections on the next sync, and Service-owned EndpointSlices
  will continue to be reconciled directly.

###### What specific metrics should inform a rollback?

Signals to monitor will include:

- A relative increase in error results from
  `endpoint_slice_controller_syncs` for EndpointSelector reconciliation.
- Increase in `EndpointSlice` churn (endpoints added or removed per sync).
- API server error rate for `EndpointSelector` operations.
- Pod readiness transitions not reflected in `EndpointSlices` within the
  expected latency bound.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual upgrade → downgrade → upgrade testing will be documented before
targeting Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No APIs, fields, flags, or features will be deprecated at Alpha.

At GA, documentation for the shadow-`Service` pattern — creating a headless
`Service` solely to generate `EndpointSlices` — will be updated to recommend
`EndpointSelector` for that backend-selection use case.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

`EndpointSelector` objects will be listed directly with
`kubectl get endpointselectors --all-namespaces`. The existing API server
metric `apiserver_storage_objects{resource="endpointselectors.discovery.k8s.io"}`
will report the number of stored EndpointSelector objects once the API is
implemented.

###### How can someone using this feature know that it is working for their instance?

- [ ] Other
  - Verify that `EndpointSlice` objects exist and reflect live Pod readiness:
    `kubectl get endpointslices -l kubernetes.io/service-name=<name>` for
    Services, or
    `kubectl get endpointslices -l kubernetes.io/endpoint-selector-name=<name>`
    for an explicitly created `EndpointSelector`. For a Service-derived
    projection, verify that the corresponding `EndpointSelector` exists and
    its selector and target ports match the Service.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

`EndpointSelector`-managed `EndpointSlices` will be expected to reflect Pod
readiness changes within the same latency bounds as Service-managed slices. The
existing [EndpointSlice SLO][eps-slo] (Pod readiness → slice updated within 1s
for small clusters) will apply.

[eps-slo]: https://git.k8s.io/community/sig-scalability/slos/slos.md

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `endpoint_slice_controller_syncs`
  - Aggregation method: Counter, partitioned by reconciliation result.
  - Component exposing the metric: `kube-controller-manager`
  - Detail: A relative increase in `result="error"` will indicate failed
    EndpointSlice reconciliation for explicitly created EndpointSelectors.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Before Beta, evaluate whether `endpoint_slice_controller_syncs` needs a source
label to distinguish explicitly created EndpointSelectors from Services. Any
such label must preserve the existing metric's stability guarantees.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- `kube-apiserver`: will need the `EndpointSelector` feature gate enabled to
  accept `EndpointSelector` objects.
- `kube-controller-manager`: will need to run with the gate enabled for the
  `endpointslice-controller` and `service-endpointselector-controller` to
  reconcile `EndpointSelector` objects.

No external services or cloud provider APIs will be required.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. The implementation will introduce:

- New watch: `endpointslice-controller` will add a watch on `EndpointSelector`
  objects.
- New writes: `service-endpointselector-controller` will create and delete
  `EndpointSelector` objects in proportion to `Service` count and churn.
  In steady state, `Service` updates will trigger a re-sync but will typically
  produce no write if the `EndpointSelector` is already current.
- `EndpointSlice` create/update/delete rate will be unchanged for the `Service`
  projection path — the same slices will be produced directly from the same
  Service and pod events.

Throughput estimates relative to existing `EndpointSlice` load will be
benchmarked before Beta.

###### Will enabling / using this feature result in introducing new API types?

Yes: `EndpointSelector` (`discovery.k8s.io/v1alpha1`) will be namespace-scoped.

For each selector-based Service, one EndpointSelector projection will be
created. Explicit testing targets will be defined and validated as part of Beta.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

New `EndpointSelector` objects will be written to etcd, each roughly the same
size as a `Service` object. One projection will be created per selector-based
Service.

The total `EndpointSlice` count will not increase for the Service projection
path — the same slices will continue to be produced directly from `Service`.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The Service Pod readiness → EndpointSlice update path will remain unchanged.
Explicit EndpointSelectors will use the EndpointSlice controller directly, with
the same Pod-readiness-to-EndpointSlice SLO as Services. Benchmarking before
Beta will validate the additional controller and informer load.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

`kube-controller-manager` will gain a new informer cache for
`EndpointSelector` objects. Memory overhead will be proportional to the number
of EndpointSelector objects — the same order of magnitude as the existing
Service informer cache for Service projections.

CPU overhead from the `service-endpointselector-controller` will be
proportional to Service churn rate. Writes will be gated on diffs, so
steady-state cost is expected to be low. Formal benchmarks will be completed
before Beta.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No new node-resource exhaustion is expected. Service EndpointSlices will remain
functionally equivalent from `kube-proxy`'s perspective. EndpointSlices created
for direct EndpointSelector consumers will not require new node-level resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

#### API server or etcd unavailable

- **Detection:** API server and controller error metrics will increase;
  controller logs will show failed list, watch, or write operations.
- **Mitigations:** Restore API server or etcd availability. Existing
  EndpointSlices will continue serving traffic until controllers recover and catch
  up.
- **Diagnostics:** Check API server and etcd health, then inspect
  kube-controller-manager logs for EndpointSlice and EndpointSelector
  reconciliation errors.
- **Testing:** Integration tests will verify that controllers resume reconciliation
  after transient API server errors.

###### What are other known failure modes?

#### Explicit EndpointSelector does not update EndpointSlices after a pod readiness change

- **Detection:** `endpoint_slice_controller_syncs{result="error"}` will increase,
  or the EndpointSlices selected by
  `kubernetes.io/endpoint-selector-name=<name>` do not reflect Pod readiness.
- **Mitigations:** Verify the feature gate on kube-apiserver and
  kube-controller-manager, correct invalid EndpointSelector configuration, and
  restore controller access to the API server. Restart kube-controller-manager
  only after the underlying condition is corrected.
- **Diagnostics:** Inspect kube-controller-manager logs for the
  EndpointSelector's namespace and name; compare its selector and target ports
  with the selected Pods and generated EndpointSlices.
- **Testing:** Integration tests will cover Pod readiness transitions, invalid
  selectors and ports, and feature-gate disablement and re-enablement.

#### Service-derived EndpointSelector projection is absent or stale

- **Detection:** The Service has no corresponding EndpointSelector, or the
  projection's selector or target ports differ from the Service. API server
  write errors and service-endpointselector-controller logs identify rejected
  projection writes.
- **Mitigations:** Correct quota, admission, or RBAC policy that rejects the
  projection, then allow the controller's normal retry to create or update it.
  Service-owned EndpointSlices will continue to update directly from the Service
  while the projection is unavailable.
- **Diagnostics:** Compare the Service with its projected EndpointSelector and
  inspect controller logs for the Service namespace and name.
- **Testing:** Integration tests will cover projection creation, updates, rejected
  writes, controller restart, and the absence of duplicate EndpointSlices.

#### Service-derived EndpointSelector projection name collision

- **Detection:** The Service has a warning Event stating that its projection
  name is occupied, and the same-name EndpointSelector lacks a controller owner
  reference to that Service UID.
- **Mitigations:** Rename or delete the user-managed EndpointSelector, then let
  the projection controller retry. Service-owned EndpointSlices will continue to
  update while the projection is unavailable.
- **Diagnostics:** Inspect the EndpointSelector's controller owner reference
  and compare its namespace, name, and UID with the Service.
- **Testing:** Integration tests will verify that the controller never modifies or
  deletes a same-name object without the matching Service owner reference.

###### What steps should be taken if SLOs are not being met to determine the problem?

Full runbook will be required at Beta. In Alpha, operators will check `kube-controller-manager` logs
at verbosity level 4 or higher for `endpointslice-controller` sync errors, and
verify the feature gate is enabled on both `kube-apiserver` and
`kube-controller-manager`.

## Implementation History

- 2026-05-26: KEP issue filed
  ([kubernetes/enhancements#6116](https://github.com/kubernetes/enhancements/issues/6116))

## Drawbacks

- `EndpointSelector` partially overlaps with `Service`. Users may be unsure
  which to use, especially when `Service` semantics are mostly — but not
  entirely — unwanted.
- Creating one EndpointSelector projection per selector-based Service increases
  `kube-controller-manager` memory and API server object count proportionally.

## Alternatives

### Extend Service.spec.selector to Support matchExpressions

`Service.spec.selector` is typed as `map[string]string` rather than
`metav1.LabelSelector`. Adding `matchExpressions` support requires introducing
a new field (the existing field cannot change type), which creates a semantic
ambiguity: an empty `selector` currently means "selectorless/manual mode."
A second selector field makes the interaction between the two undefined for old
clients. Tim Hockin closed kubernetes/kubernetes#48528 as low-urgency and
high-cost in 2023 for this reason.

### Manual EndpointSlice Management

Controllers write `EndpointSlices` directly without a `Service`. This is
option 2 from the Motivation section. It forces every controller to
re-implement pod-watching and `EndpointSlice` packing logic, and a single pod
readiness change can trigger writes to every `EndpointSlice` for the affected
workload — a scalability problem that grows with cluster size.

### Shadow Service (Headless Service to Generate EndpointSlices)

Controllers create a headless `Service` solely to trigger
`endpointslice-controller` to produce `EndpointSlices`. This is the current
`InferencePool` workaround. It brings unwanted DNS entries, requires
`Service`-create RBAC for controllers that should not need it, and is actively
harmful for cases like `InferencePool` where a VIP would cause kube-proxy to
intercept traffic before the endpoint picker can act.

### Broader Service Decomposition

Tim Hockin has noted (kubernetes/kubernetes#48528) that `Service` could be
decomposed into composable primitives more broadly. This KEP seeks to be one
step in that direction but does not attempt to solve the entire problem in one
go. A broader vision is discussed in [this slide deck][decompose-svc-slides]
which was [presented at a SIG Network meeting][decompose-svc-recording].

[decompose-svc-slides]: https://docs.google.com/presentation/d/1h_2WYyvIbyyCIMN61FInAfFtaJk_TrYpDXKoHXnUfy8/edit?slide=id.p#slide=id.p
[decompose-svc-recording]: https://youtu.be/OmD_fKasCNA?si=xpcOhcPgUd7_mbQw&t=1083

## Infrastructure Needed

None for Alpha. If the shared controller library is extracted to a staging
repository in a future release, that subproject will be noted here.
