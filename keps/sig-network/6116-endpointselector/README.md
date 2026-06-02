# KEP-6116: EndpointSelector

<!--
Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

<!-- toc -->
- [KEP-6116: EndpointSelector](#kep-6116-endpointselector)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Manual Creation](#manual-creation)
    - [Controller-Managed Creation](#controller-managed-creation)
    - [User Stories](#user-stories)
      - [Story 1: InferencePool Implementation Simplification](#story-1-inferencepool-implementation-simplification)
      - [Story 2: Controller-Managed Endpoints Without Redundant Pod Watching](#story-2-controller-managed-endpoints-without-redundant-pod-watching)
      - [Story 3: Client Settings Configuration for an Existing Service](#story-3-client-settings-configuration-for-an-existing-service)
    - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
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
    - [Service Compatibility Matrix](#service-compatibility-matrix)
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
          - [How can this feature be enabled / disabled in a live cluster?](#how-can-this-feature-be-enabled--disabled-in-a-live-cluster)
          - [Does enabling the feature change any default behavior?](#does-enabling-the-feature-change-any-default-behavior)
          - [Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?](#can-the-feature-be-disabled-once-it-has-been-enabled-ie-can-we-roll-back-the-enablement)
          - [What happens if we reenable the feature if it was previously rolled back?](#what-happens-if-we-reenable-the-feature-if-it-was-previously-rolled-back)
          - [Are there any tests for feature enablement/disablement?](#are-there-any-tests-for-feature-enablementdisablement)
    - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
          - [How can a rollout or rollback fail? Can it impact already running workloads?](#how-can-a-rollout-or-rollback-fail-can-it-impact-already-running-workloads)
          - [What specific metrics should inform a rollback?](#what-specific-metrics-should-inform-a-rollback)
          - [Were upgrade and rollback tested? Was the upgrade-\>downgrade-\>upgrade path tested?](#were-upgrade-and-rollback-tested-was-the-upgrade-downgrade-upgrade-path-tested)
          - [Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?](#is-the-rollout-accompanied-by-any-deprecations-andor-removals-of-features-apis-fields-of-api-types-flags-etc)
    - [Monitoring Requirements](#monitoring-requirements)
          - [How can an operator determine if the feature is in use by workloads?](#how-can-an-operator-determine-if-the-feature-is-in-use-by-workloads)
          - [How can someone using this feature know that it is working for their instance?](#how-can-someone-using-this-feature-know-that-it-is-working-for-their-instance)
          - [What are the reasonable SLOs (Service Level Objectives) for the enhancement?](#what-are-the-reasonable-slos-service-level-objectives-for-the-enhancement)
          - [What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?](#what-are-the-slis-service-level-indicators-an-operator-can-use-to-determine-the-health-of-the-service)
          - [Are there any missing metrics that would be useful to have to improve observability of this feature?](#are-there-any-missing-metrics-that-would-be-useful-to-have-to-improve-observability-of-this-feature)
    - [Dependencies](#dependencies)
          - [Does this feature depend on any specific services running in the cluster?](#does-this-feature-depend-on-any-specific-services-running-in-the-cluster)
    - [Scalability](#scalability)
          - [Will enabling / using this feature result in any new API calls?](#will-enabling--using-this-feature-result-in-any-new-api-calls)
          - [Will enabling / using this feature result in introducing new API types?](#will-enabling--using-this-feature-result-in-introducing-new-api-types)
          - [Will enabling / using this feature result in any new calls to the cloud provider?](#will-enabling--using-this-feature-result-in-any-new-calls-to-the-cloud-provider)
          - [Will enabling / using this feature result in increasing size or count of the existing API objects?](#will-enabling--using-this-feature-result-in-increasing-size-or-count-of-the-existing-api-objects)
          - [Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?](#will-enabling--using-this-feature-result-in-increasing-time-taken-by-any-operations-covered-by-existing-slisslos)
          - [Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?](#will-enabling--using-this-feature-result-in-non-negligible-increase-of-resource-usage-cpu-ram-disk-io--in-any-components)
          - [Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?](#can-enabling--using-this-feature-result-in-resource-exhaustion-of-some-node-resources-pids-sockets-inodes-etc)
    - [Troubleshooting](#troubleshooting)
          - [How does this feature react if the API server and/or etcd is unavailable?](#how-does-this-feature-react-if-the-api-server-andor-etcd-is-unavailable)
          - [What are other known failure modes?](#what-are-other-known-failure-modes)
          - [What steps should be taken if SLOs are not being met to determine the problem?](#what-steps-should-be-taken-if-slos-are-not-being-met-to-determine-the-problem)
  - [Implementation History](#implementation-history)
  - [Drawbacks](#drawbacks)
  - [Alternatives](#alternatives)
    - [Extend Service.spec.selector to Support matchExpressions](#extend-servicespecselector-to-support-matchexpressions)
    - [Manual EndpointSlice Management](#manual-endpointslice-management)
    - [Shadow Service (Headless Service to Generate EndpointSlices)](#shadow-service-headless-service-to-generate-endpointslices)
    - [Broader Service Decomposition](#broader-service-decomposition)
  - [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->

`EndpointSlices` provide a scalable way to track (typically in-cluster) network
endpoints in Kubernetes, providing addressing, health, and topology information
to consumers. The practical interface for managing the lifecycle of
`EndpointSlices` at scale is the `Service` resource, which includes additional
functionality (ClusterIP/VIP, DNS hostname, kube-proxy load balancing) that is
unwanted in many use cases. The `EndpointSelector` resource allows users and
controllers to declare a pod selector and a set of ports; the
`endpointslice-controller` manages the corresponding `EndpointSlices`.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP. Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

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
following resources from across the Kubernetes ecosystem each implement their
own version of label-selection-based endpoint management:

- Istio `ServiceEntry`: `spec.workloadSelector` selects Kubernetes pods and VM
  workloads as service endpoints.
  - API: https://github.com/istio/api/blob/master/networking/v1alpha3/service_entry.proto
  - Docs: https://istio.io/latest/docs/reference/config/networking/service-entry/
- Cilium `CiliumLocalRedirectPolicy`: `spec.redirectBackend.localEndpointSelector`
  selects node-local backend pods for redirected traffic.
  - API: https://github.com/cilium/cilium/blob/main/pkg/k8s/apis/cilium.io/v2/clrp_types.go
  - Docs: https://docs.cilium.io/en/stable/network/kubernetes/local-redirect-policy/
- Kubernetes SIG Network Gateway API (proposed): GEP-4488 `Backend` defines
  `EndpointSelector` with a pod `LabelSelector` for backend endpoints.
  - GEP: https://github.com/kubernetes-sigs/gateway-api/blob/main/geps/gep-4488/index.md

Furthermore, the highly coupled nature of `Service` has made it difficult to
extend its label selection functionality to support more complex selector
semantics (for example, set-based `matchExpressions`) without breaking
backwards compatibility with existing `Service` objects that use an empty
selector to indicate "manual" mode. This has been a longstanding community
request (see kubernetes/kubernetes#48528 and kubernetes/kubernetes#62795) that
has been deferred due to the complexity of adding new selector fields to
`Service` without breaking existing objects.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Allow users and controllers to create `EndpointSlices` for a set of pods
  matching a label selector without creating a `Service`.
- Extend `endpointslice-controller` to watch `EndpointSelector` objects instead
  of `Service` objects when the feature gate is enabled.
- Create a new controller that creates an `EndpointSelector` for each `Service`
  with a pod selector to maintain backwards compatibility.
- Support equality-based pod selection initially (`matchLabels`), matching the
  practical selector expressiveness of `Service.spec.selector` without carrying
  forward `Service`'s nil-selector opt-out semantics.

### Non-Goals

- Replacing `Service` altogether.
- Extending `endpointslice-controller` to read arbitrary resources with pod
  selectors.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?
The "Design Details" section below is for the real nitty-gritty.
-->

This KEP introduces `EndpointSelector`[^1], a namespace-scoped resource that allows
users and controllers to obtain a managed set of `EndpointSlices` for a pod
label selector without creating a `Service`. The `endpointslice-controller`
watches `EndpointSelector` objects and manages their corresponding
`EndpointSlices` using the same reconciliation logic it applies to
`Service`-owned slices today — tracking pod readiness, topology, and slice
packing.

The `EndpointSelector` spec exposes the `Service` fields that affect endpoint
selection and EndpointSlice presentation: `selector`, endpoint `ports`,
`publishNotReadyAddresses`, and `hints.trafficDistribution`. `ports` describe
the ports on selected endpoints — equivalent to `Service` target ports — not
frontend `Service` ports. Fields tied to `Service`'s frontend role (ClusterIP,
DNS hostname, load balancing policy, etc.) are not part of `EndpointSelector`.

[^1]: Alternative names considered: `EndpointGroup`, `EndpointPool`

`EndpointSelector` objects are created in one of two ways:

### Manual Creation

A user or controller creates an `EndpointSelector` directly and manages its
lifecycle. No `ownerReference` is set. The resource is referenced by name from
higher-level objects (for example, a `Backend` resource in Gateway API). This
is the right model when the `EndpointSelector` outlives any single parent, or
when a user wants to track a set of pod endpoints without introducing a
`Service`.

### Controller-Managed Creation

A controller creates an `EndpointSelector` in response to a higher-level
resource (for example, an `InferencePool`). The controller sets
`ownerReferences` for garbage collection, uses `generateName` instead of
`name` to avoid naming conflicts when multiple controllers target the same
workload, and adds an `app.kubernetes.io/managed-by` label. Users do not
interact with the `EndpointSelector` directly; it is an implementation detail
of the parent resource's lifecycle.

To maintain backwards compatibility, a new controller will be added to
`pkg/controller` that creates an `EndpointSelector` for each `Service` with a
pod selector. This controller is enabled by the same feature gate as the
`endpointslice-controller` changes and ensures that existing `Service` objects
get a corresponding `EndpointSelector` without user action. The
`endpointslice-controller` reconciles the `EndpointSelector` instead of the
`Service`, but ownership of the resulting `EndpointSlices` is unchanged — they
still point to the `Service` — to avoid breaking any tooling that filters by
owner kind.

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1: InferencePool Implementation Simplification

As a platform operator running AI inference workloads, I create an `InferencePool`
targeting my model-serving pods by label. The `InferencePool` controller creates
a corresponding `EndpointSelector`, which the `endpointslice-controller` uses to
produce `EndpointSlices` reflecting live pod readiness. My gateway's endpoint
picker consumes those slices directly — without any kube-proxy round-robin in
front of it — which is what makes per-request model routing viable.

#### Story 2: Controller-Managed Endpoints Without Redundant Pod Watching

As a controller author, my CRD needs to track the endpoints of the pods it
manages. Rather than watching pods, tracking readiness transitions, and managing
`EndpointSlice` packing — logic the `endpointslice-controller` already owns — I
create an `EndpointSelector` with `ownerReferences` pointing to my resource and
let the `endpointslice-controller` handle the rest. My controller only needs to
create and delete the `EndpointSelector`; the endpoint lifecycle is not my
problem.

#### Story 3: Client Settings Configuration for an Existing Service

As an app developer, I have an existing `Service` and I want to configure client
settings like TLS certificates, the MCP protocol, or other connection parameters
so my gateway can connect to it. Rather than creating new infrastructure, I
create a [Gateway API `Backend`][gep-4488] of `type: EndpointSelector` and set
its `selectorRef` to the `EndpointSelector` the `Service` controller
automatically created for my `Service`. I declare the TLS configuration and
protocol on the `Backend`; the gateway uses those settings when routing to my
existing pods. The `Service` routing and endpoint selection behavior are
unchanged, though `EndpointSlices` gain the new
`kubernetes.io/endpoint-selector-name` label once the feature gate is
enabled.

[gep-4488]: https://gateway-api.sigs.k8s.io/geps/gep-4488/

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

`EndpointSelector` and `Service` overlap in the endpoint-selection piece but are
not in conflict. Once this feature is enabled, the `Service` controller creates
an `EndpointSelector` for each `Service` with a pod selector, and the
`endpointslice-controller` drives reconciliation from that object. Users and
tooling that interact with `Services` and their `EndpointSlices` observe the
same functional behavior.

An `EndpointSelector` is loosely coupled to its consumers. It has no awareness
of which higher-level resources reference it, just as `EndpointSlices` have no
awareness of which consumers watch them. This mirrors the existing
`Service` → `EndpointSlice` relationship and scales for the same reasons: a
single `EndpointSelector` can be referenced by multiple resources, and those
resources can come and go without coordinating through the `EndpointSelector`
itself.

The Gateway API [GEP-4731] introduces `XEndpointSelector` in the experimental
channel (`gateway.networking.x-k8s.io/v1alpha1`) as a stopgap while this KEP
matures. The Gateway API community has stated that `XEndpointSelector` will not
progress to the standard channel. All Gateway API object reference fields that
point to an `XEndpointSelector` MUST accept pluggable groups — not hardcoded to
`gateway.networking.k8s.io` — to allow a clean migration to the core API once
this KEP reaches GA.

`EndpointSelector` distinguishes between fields that control which pods are
selected, fields that control EndpointSlice conditions, and fields that describe
how consumers route to them. Pod readiness does not by itself determine whether
a matching pod appears in an `EndpointSlice`; it determines the endpoint
conditions written for that pod. `publishNotReadyAddresses` is a first-class
spec field because it affects the `ready` condition on generated endpoints,
mirroring `Service.spec.publishNotReadyAddresses`. Traffic distribution metadata
belongs in `spec.hints` to keep the API boundary explicit. In Alpha,
`spec.hints` exposes only `trafficDistribution`, mirroring
`Service.spec.trafficDistribution`. Vendor-specific extension fields in `hints`
are explicitly deferred past Alpha.

[GEP-4731]: https://github.com/kubernetes-sigs/gateway-api/pull/4731

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?
How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Security

`EndpointSelector` is namespace-scoped. The `endpointslice-controller` only
selects pods within the same namespace as the `EndpointSelector`, enforced at
reconciliation time the same way it is for `Service`-owned slices. RBAC for
creating `EndpointSelectors` follows the same model as `Services`: namespace
admins can create them; cluster-level restrictions apply through standard
mechanisms. `NetworkPolicy` continues to apply to the selected pods regardless
of whether their `EndpointSlices` originated from a `Service` or an
`EndpointSelector`.

#### Control Plane Load

The per-object reconciliation cost of an `EndpointSelector` is equivalent to a
`Service` with the same selector. For clusters where the `Service` controller
auto-creates an `EndpointSelector` per `Service`, the `EndpointSlice` count and
reconciliation frequency do not change — the same slices are produced, now
driven by an `EndpointSelector` rather than directly by the `Service`. The net
additional load is a new watch in the `endpointslice-controller` and apiserver
writes to create the `EndpointSelector` objects for `Services`.

#### Orphaned Resources

Controller-managed `EndpointSelectors` that lack `ownerReferences` will persist
after the owning resource is deleted, along with the `EndpointSlices` they
produced. The convention that controller-managed `EndpointSelectors` MUST set
`ownerReferences` is enforced by documentation and, potentially, conformance.
The `app.kubernetes.io/managed-by` label allows operators to identify and audit
controller-managed `EndpointSelectors` in a cluster.

#### API Confusion

Introducing a resource that partially overlaps with `Service` risks confusion
about which to use when. This is mitigated by making the relationship explicit:
`Service` creates an `EndpointSelector` on behalf of users; `EndpointSelector`
is the right choice only when `Service`'s additional semantics — ClusterIP, DNS,
kube-proxy routing — are unwanted or actively harmful.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Current State

Today, the `endpointslice-controller` reads changes to `Service` and `Pod`
objects to manage `EndpointSlices`. When a `Service` with a pod selector is
created or updated, the controller creates or updates `EndpointSlices` to
reflect the set of pods matching that selector. A `Service` with a nil pod
selector or type `ExternalName` is ignored and produces no `EndpointSlice`; the
controller preserves this nil selector opt-out behavior when creating derived
`EndpointSelector` objects. When a pod changes status (for example, becomes
ready), the controller updates the relevant `EndpointSlices` accordingly. When
the `Service` is deleted, the controller garbage collects the owned
`EndpointSlices`.

### Proposed Implementation

When the `EndpointSelector` feature gate is enabled, the
`endpointslice-controller` watches `EndpointSelector` objects instead of
`Service`. This is a purposeful structural refactor to prevent duplication bugs
and ensure that the same reconciliation logic applies to both `Service`-owned
and `EndpointSelector`-owned slices. If the `EndpointSelector` API is not yet
available on the API server when the gate is enabled — detectable via informer
registration failure — the controller falls back to `Service`-watching until
the API becomes accessible, then transitions automatically. A new controller (the
`service-endpointselector-controller`) in `kube-controller-manager` watches
`Service` objects and creates an `EndpointSelector` for each `Service` with a
pod selector, ensuring backwards compatibility. The `Service` remains the
source of truth; the `EndpointSelector` is a derived resource managed by the
`Service` controller. Users and tooling that interact with `Services` and their
`EndpointSlices` observe the same functional behavior. The
`service-endpointselector-controller` is an implementation detail of the
`Service` → `EndpointSelector` compatibility path with no guarantees of
stability between releases.

### API Definition

```go
// EndpointSelector is a namespace-scoped resource in discovery.k8s.io/v1alpha1 that
// declares a pod label selector and a set of ports. The
// endpointslice-controller manages the corresponding EndpointSlices.
type EndpointSelector struct {
  metav1.TypeMeta   `json:",inline"`
  metav1.ObjectMeta `json:"metadata,omitempty"`

  Spec EndpointSelectorSpec `json:"spec,omitempty"`
  // Status is intentionally omitted in Alpha. EndpointSelector may be created
  // by multiple controller types; status design is deferred until use cases
  // across those controllers are better understood.
}

type EndpointSelectorSpec struct {
  // Selector selects the pods whose addresses are tracked by this resource. It
  // must not be empty. In Alpha, pkg/apis/discovery/validation rejects non-empty
  // matchExpressions; only matchLabels is accepted. matchExpressions support is
  // deferred to a follow-up. The selector is mutable; changing it retargets the
  // managed EndpointSlices in the same way changing Service.spec.selector does
  // today.
  Selector metav1.LabelSelector `json:"selector"`

  // IPFamilies specifies the IP families for which EndpointSlices should be
  // produced. Defaults to all address families present in matching pod
  // addresses, producing one EndpointSlice addressType per family found (IPv4,
  // IPv6, or both in a dual-stack cluster). Set this field to restrict output
  // to a specific family. Mirrors Service.spec.ipFamilies.
  // +optional
  IPFamilies []corev1.IPFamily `json:"ipFamilies,omitempty"`

  // Ports defines the endpoint port numbers and protocols exposed on the
  // selected pods. These are target ports, not Service frontend ports. Omitting
  // this field causes the controller to include all container ports from
  // matching pods.
  // +optional
  Ports []EndpointSelectorPort `json:"ports,omitempty"`

  // PublishNotReadyAddresses controls how the ready condition is set for pods
  // that are not yet ready. Mirrors the identically-named field on Service.spec.
  // +optional
  PublishNotReadyAddresses bool `json:"publishNotReadyAddresses,omitempty"`

  // Hints provides topology and routing metadata that the
  // endpointslice-controller propagates into EndpointSlice hints.
  // +optional
  Hints *EndpointSelectorHints `json:"hints,omitempty"`
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

  // Port is the target port number. Whether this field should become an
  // IntOrString to allow named-port references (analogous to
  // Service.spec.ports[].targetPort) is an open question — see Edge Cases.
  Port int32 `json:"port"`

  // AppProtocol is the application-level protocol hint for this port,
  // following the same conventions as Service.spec.ports[].appProtocol.
  // +optional
  AppProtocol *string `json:"appProtocol,omitempty"`
}

// EndpointSelectorHints carries routing metadata that the
// endpointslice-controller propagates into EndpointSlice hints. Fields here
// are written by users or high-level controllers — not by consumers.
// TODO: I'm somewhat worried about re-using the term "hints" with different
// semantics than per-endpoint topology hints. Open to suggestions on naming
type EndpointSelectorHints struct {
  // TrafficDistribution expresses a preference for how traffic is routed to
  // endpoints (for example, "PreferClose"). Mirrors the identically-named
  // field on Service.spec.
  // +optional
  TrafficDistribution *string `json:"trafficDistribution,omitempty"`
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
      port: 8080
      protocol: TCP
      appProtocol: kubernetes.io/grpc
  hints:
    trafficDistribution: PreferClose
```

A controller-managed `EndpointSelector` created by an `InferencePool`
controller with garbage-collection metadata:

```yaml
apiVersion: discovery.k8s.io/v1alpha1
kind: EndpointSelector
metadata:
  generateName: my-inference-pool-
  namespace: default
  labels:
    app.kubernetes.io/managed-by: inferencepool-controller
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
      port: 8080
      protocol: TCP
```

### EndpointSlice-Controller Changes

The `endpointslice-controller` today reconciles `EndpointSlices` from `Service`
objects. When the `EndpointSelector` feature gate is enabled, the controller is
refactored to drive reconciliation from `EndpointSelector` objects instead. The
`service-endpointselector-controller` (described below) ensures every `Service`
with a pod selector has a corresponding `EndpointSelector`, so existing
`Service`-owned `EndpointSlices` continue to be produced without user action.

To reduce fracturing in the `endpointslice-controller`, a new interface
`EndpointSliceSource` will be introduced that abstracts the commonalities between
`Service` and `EndpointSelector` as sources of `EndpointSlices`. This interface
will be introduced to the controller first, before `EndpointSelector`, to ensure
the necessary information from source objects is correctly identified.

As mentioned above, Gateway API [GEP-4731] introduces `XEndpointSelector` in
the experimental channel as a means for experimentation and feedback until this
KEP reaches GA. This resource will never be promoted to the standard channel;
however, it may be desirable for the core endpointslice package to be able to
consume `XEndpointSelector` as an additional `EndpointSliceSource` to allow
Gateway API implementations to avoid re-implementing pod-watching logic. The core
`endpointslice-controller` will not create `EndpointSlices` for `XEndpointSelector`
objects directly; rather, the controller will expose the necessary interfaces and
abstractions to allow a Gateway API implementation to consume `XEndpointSelector`
objects in a separate controller that reuses the core pod-watching and slice
management logic. This additional mode will be removed once `EndpointSelector`
reaches GA.

Today, `EndpointSlice` objects produced for a `Service` carry a
`kubernetes.io/service-name` label to link each slice to its Service. This
convention will remain unchanged for the compatibility path; in other words,
`EndpointSlices` produced for a `Service`-owned `EndpointSelector` will maintain
the `kubernetes.io/service-name` label and ownership reference to the `Service`.
The `endpointslice-controller` determines label and owner assignment from the
`EndpointSelector`'s `ownerReferences`: if it points to a `Service`, the
resulting `EndpointSlices` carry the `kubernetes.io/service-name` label and an
ownerRef to that `Service`; if it has no owner or an owner of a different kind,
the `EndpointSlices` carry no `kubernetes.io/service-name` label and their
ownerRef points to the `EndpointSelector` itself. Pointing the ownerRef to the
`EndpointSelector` rather than to the higher-level resource (for example, an
`InferencePool`) allows garbage collection without requiring the controller to
understand the higher-level resource's API.

A new label `kubernetes.io/endpoint-selector-name` will be added to all
`EndpointSlices` to link them to their owning `EndpointSelector`. The existing
`kubernetes.io/service-name` label only links to the `Service` and is not
sufficient for slices owned by an `EndpointSelector` without a `Service` owner.
The new label will be added for all `EndpointSlices` regardless of whether they
are owned by a `Service` or an `EndpointSelector`, to allow tooling to identify
the owning `EndpointSelector` for any given slice. `EndpointSelector` names must
be safe to use as Kubernetes label values, just as `Service` names are safe to
use in the existing `kubernetes.io/service-name` label. Alpha validation will
therefore use the same name constraints as `Service` unless a different
label-safe linking mechanism is selected before implementation.

### New Service EndpointSelector Controller

A new `service-endpointselector-controller` in `pkg/controller` creates and
deletes an `EndpointSelector` for each `Service` with a pod selector. The
`Service` remains the source of truth; the `EndpointSelector` is derived from
it and deleted when the `Service` is deleted. The controller is enabled by the
`EndpointSelector` feature gate in `kube-controller-manager`.

The controller should reconcile Service-owned `EndpointSelector` objects using
Server-Side Apply for field-ownership-preserving updates. During Alpha, the
controller owns all fields and labels derived from `Service` and overwrites
direct user edits to those values. Labels and annotations not owned by the
controller may be preserved, but Alpha does not guarantee any user-customizable
fields on a Service-owned `EndpointSelector`.

### Metadata Propagation

Today, the `endpointslice-controller` copies non-reserved `Service` labels to
the `EndpointSlices` it creates. When this feature gate is enabled,
`EndpointSelector` becomes the source object for `EndpointSlices`, so the
`endpointslice-controller` copies non-reserved labels from the
`EndpointSelector` to generated `EndpointSlices`. The controller continues to own
reserved EndpointSlice labels (the full list must be confirmed against the
`endpointslice-controller` source before this KEP moves to implementable)
including at minimum `kubernetes.io/service-name`,
`endpointslice.kubernetes.io/managed-by`, and
`kubernetes.io/endpoint-selector-name`.

For the Service compatibility path, the `service-endpointselector-controller`
copies non-reserved `Service` labels to the derived `EndpointSelector`. The
label propagation chain is therefore:
`Service` → Service-owned `EndpointSelector` → Service-owned `EndpointSlices`.
This preserves the existing observable behavior for `EndpointSlices` generated
for `Services`, while keeping the `endpointslice-controller` responsible for
copying metadata from the source object it reconciles.

Directly created `EndpointSelector` objects use the same rule:
non-reserved labels on the `EndpointSelector` are propagated to generated
`EndpointSlices`. This KEP does not introduce general annotation propagation in
Alpha; annotations that affect EndpointSlice behavior, such as topology-related
configuration, are handled as explicit spec fields or implementation-specific
controller inputs rather than copied wholesale.

### Service Compatibility Matrix

For the compatibility path, existing `Service` behavior is preserved. The
derived `EndpointSelector` is an implementation detail that lets the
`endpointslice-controller` reconcile from a smaller source object without
changing the observable `Service` → `EndpointSlice` contract.

| Existing `Service` behavior | `EndpointSelector` compatibility behavior |
| :--- | :--- |
| `Service.spec.selector == nil` | No `EndpointSelector` is created; the `Service` remains selectorless/manual. |
| `Service.type == ExternalName` | No `EndpointSelector` is created and no `EndpointSlices` are produced by this controller. |
| Non-nil `Service.spec.selector` with equality-based labels | The `service-endpointselector-controller` creates or updates a Service-owned `EndpointSelector` with equivalent `matchLabels`. |
| `Service.spec.selector` updated | The Service-owned `EndpointSelector` is updated; the `endpointslice-controller` reconciles additions and removals. |
| `Service.spec.ports[]` | Service frontend ports are not exposed on `EndpointSelector`. The compatibility path preserves existing EndpointSlice port output, including target-port resolution, protocol, port name, and `appProtocol`. |
| Named `Service.spec.ports[].targetPort` | The compatibility path preserves existing per-pod named-port resolution. This may be represented in the internal `EndpointSliceSource` adapter rather than as a direct user-facing `EndpointSelectorPort` field. |
| Headless `Service` with no ports | Existing empty EndpointSlice port-list behavior is preserved. Direct `EndpointSelector` objects can also omit `spec.ports`. |
| `publishNotReadyAddresses` | Matching pods are represented in EndpointSlices and endpoint conditions mirror current Service behavior; setting this field affects the generated `ready` condition for not-ready pods. |
| Pod readiness, termination, node, zone, and hostname changes | Generated endpoint conditions and metadata remain functionally equivalent to pre-gate EndpointSlices. |
| Service labels copied to EndpointSlices | Existing Service label propagation remains unchanged: non-reserved Service labels are copied to the derived `EndpointSelector`, then copied by the `endpointslice-controller` to Service-owned EndpointSlices. |
| `kubernetes.io/service-name` label and `Service` ownerRef | Preserved for Service-backed EndpointSlices so existing consumers and tooling continue to work. |
| `Service.spec.trafficDistribution` and topology hint behavior | Existing hint behavior is preserved. The Service-derived source supplies the information needed to produce the same EndpointSlice hints as before the gate. |
| Service IP family/address-type behavior | Existing EndpointSlice address-type selection is preserved for Service-backed slices: the `service-endpointselector-controller` copies `Service.spec.ipFamilies` to the derived `EndpointSelector.spec.ipFamilies`. Direct `EndpointSelector` objects use `spec.ipFamilies` to control which address families are produced; if unset, the controller generates one `EndpointSlice` per address family present in matching pod addresses. |
| `Service` deletion | The derived `EndpointSelector` is deleted, and Service-owned `EndpointSlices` are garbage collected as they are today. |

### Edge Cases and Deferred Design Decisions

An `EndpointSelector` with an empty selector is invalid. Empty
`metav1.LabelSelector` normally means "match everything", but that is too easy
to create accidentally for an API that directly publishes pod endpoints. Alpha
validation rejects a selector with no `matchLabels` and no `matchExpressions`.
For `Service`, this KEP preserves existing behavior: a nil selector opts the
`Service` out of derived `EndpointSelector` creation.

Alpha uses a value `metav1.LabelSelector` (not a pointer) as the API shape
even though `matchExpressions` are rejected initially. This keeps the API
aligned with the future goal of set-based selection without changing the field
type later. A value type produces a non-nullable field in OpenAPI, which makes
the required-selector constraint clearer than a pointer would. This decision
should be confirmed with SIG Network before moving to implementable, as a
pointer or a narrower Alpha-only type would change validation behavior.

`spec.selector` is mutable for manually created `EndpointSelector` objects.
Changing it retargets the managed `EndpointSlices`, matching the mutability of
`Service.spec.selector`. For Service-owned `EndpointSelector` objects, direct
user edits to the selector are not supported in Alpha; the
`service-endpointselector-controller` restores the selector derived from the
owning `Service`.

Supporting `matchExpressions` for `EndpointSelector` does not require changing
`Service.spec.selector`. Directly created `EndpointSelector` objects can allow
`matchExpressions` in a later release once validation and scale testing are
complete. How `Service` users opt into set-based selection is deferred. Options
include allowing carefully scoped user ownership of selected fields on the
Service-owned `EndpointSelector`, or adding a `Service.spec.selectorRef` that
points at a pre-created `EndpointSelector`. This KEP does not add
`selectorRef` in Alpha.

**Open question — named port representation in `EndpointSelectorPort.Port`.**
The `endpointslice-controller` currently resolves named `targetPort` values
(for example, `"http"`) per-pod by reading the matching container port name
from `pod.spec.containers[].ports`. This resolution happens inside the
controller, which already watches pods. For the Service compatibility path,
the `service-endpointselector-controller` creates `EndpointSelector` objects,
and named-port resolution must happen somewhere. Two approaches are under
consideration:

- *Option A*: Keep `Port` as `int32` and resolve named ports inside the
  `service-endpointselector-controller` before writing the derived
  `EndpointSelector`. The `EndpointSelector` always carries resolved numeric
  ports. Users of direct `EndpointSelector` objects must supply numeric ports;
  named-port resolution is not available without a `Service`.
- *Option B* (preferred): Change `Port` to `intstr.IntOrString`, mirroring
  `Service.spec.ports[].targetPort`. The `endpointslice-controller` resolves
  named port strings per-pod when generating `EndpointSlices`. Direct
  `EndpointSelector` objects can reference named ports just as `Service` can.

Option B aligns the API more closely with `Service.spec.ports[].targetPort`
semantics, is more useful for direct `EndpointSelector` consumers, and keeps
named-port resolution where it belongs: in the controller that watches pods.
Resolve before moving to implementable; changing this field type after Alpha
would be a breaking API change.

### Controller-Managed Conventions

Third-party controllers that create `EndpointSelectors` programmatically MUST:

- Set `ownerReferences` pointing to the managing resource for garbage
  collection. Without `ownerReferences`, the `EndpointSelector` is treated as
  user-managed, and the `endpointslice-controller` will not garbage collect it
  or its `EndpointSlices` when the owning resource is deleted.
- Use `generateName` instead of `name` to avoid naming conflicts when multiple
  controllers target the same workload.
- Set `app.kubernetes.io/managed-by` to identify the managing controller.

The `service-endpointselector-controller` uses a deterministic name derived
from the `Service` name to allow lookup by service name.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

Existing `endpointslice-controller` unit tests are tightly coupled to `Service`
as the reconciliation source. Before `EndpointSelector` support is added, those
tests must be refactored to work against the new `EndpointSliceSource`
interface, so that new tests can cover both sources without duplicating test
infrastructure.

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
-->

The following packages will be modified or created for Alpha. Coverage
percentages will be filled in before the release is targeted.

- `k8s.io/kubernetes/pkg/controller/endpointslice`: `<date>` - `<coverage>`
- `k8s.io/kubernetes/pkg/controller/serviceendpointselector` (new): `<date>` - `<coverage>`
- `k8s.io/kubernetes/pkg/apis/discovery/validation`: `<date>` - `<coverage>`
- `k8s.io/kubernetes/pkg/registry/discovery/endpointselector` (new): `<date>` - `<coverage>`

| Test description | Expected result |
| :--- | :--- |
| `EndpointSelector` created with `matchLabels` selector | `EndpointSlices` generated for matching pods |
| `EndpointSelector` created with empty selector | Validation rejects at admission |
| `EndpointSelector` created with non-empty `matchExpressions` | Validation rejects at admission |
| Manually created `EndpointSelector` selector updated | `EndpointSlices` retargeted to the newly matching pods |
| Pod with labels matching an `EndpointSelector` added | Address appears in `EndpointSlice` |
| Pod with labels matching an `EndpointSelector` deleted | Address removed from `EndpointSlice` |
| Pod transitions from not-ready to ready | Endpoint `ready` and `serving` conditions updated to true |
| Pod transitions from ready to not-ready | Endpoint `ready` and `serving` conditions updated to false |
| `EndpointSelector` with `publishNotReadyAddresses: true` | Not-ready pod endpoint has `ready: true` while `serving` reflects pod readiness |
| `EndpointSelector` deleted | Owned `EndpointSlices` garbage collected |
| `EndpointSelector` with non-reserved labels | Generated `EndpointSlices` carry those labels |
| `EndpointSelector` with reserved EndpointSlice labels | Generated `EndpointSlices` use controller-owned reserved label values |
| `EndpointSelector` with `ownerReference` pointing to a `Service` | `EndpointSlice` carries `kubernetes.io/service-name` label and `Service` ownerRef |
| `EndpointSelector` with `ownerReference` pointing to non-`Service` | `EndpointSlice` carries no `kubernetes.io/service-name`; ownerRef points to `EndpointSelector` |
| Any `EndpointSlice` produced by `endpointslice-controller` | Carries `kubernetes.io/endpoint-selector-name` label |
| `Service` with pod selector created | `service-endpointselector-controller` creates a corresponding `EndpointSelector` |
| `Service` with non-reserved labels created | Derived `EndpointSelector` carries those labels; generated `EndpointSlices` preserve existing Service label propagation behavior |
| `Service` with nil pod selector created | No `EndpointSelector` created |
| `Service` selector updated | Corresponding `EndpointSelector` updated to match |
| User edits Service-owned `EndpointSelector` fields derived from `Service` | `service-endpointselector-controller` restores the Service-derived values |
| `Service` with named `targetPort` created | Service-backed `EndpointSlices` preserve existing named-port resolution |
| Service-backed compatibility matrix scenarios | `EndpointSlices` remain functionally equivalent to pre-gate output |
| `Service` with pod selector deleted | Corresponding `EndpointSelector` deleted |
| `spec.hints.trafficDistribution` set | Value propagated into `EndpointSlice` hints |
| Feature gate disabled, `EndpointSelector` object submitted | API server rejects |
| Feature gate disabled, controllers running | `EndpointSelector` objects not reconciled |

##### Integration tests

<!--
Integration tests run against a real API server and controller. The following
scenarios are required for Alpha.
-->

- `EndpointSelector` created → matching pods reflected in `EndpointSlice`
  within controller sync period.
- Pod readiness transitions (ready → not-ready → ready) reflected in
  `EndpointSlice` conditions within the expected latency bound.
- `EndpointSelector` deleted → all owned `EndpointSlices` garbage collected.
- `Service` with pod selector created → `service-endpointselector-controller`
  creates a corresponding `EndpointSelector`; `endpointslice-controller`
  produces `EndpointSlices` functionally equivalent to the pre-gate output.
- Service labels are copied to the derived `EndpointSelector`, and
  non-reserved `EndpointSelector` labels are copied to generated
  `EndpointSlices`.
- Direct edits to Service-owned `EndpointSelector` fields derived from
  `Service` are reconciled back to the Service-derived values.
- `Service` deleted → corresponding `EndpointSelector` deleted →
  `EndpointSlices` garbage collected.
- Feature gate toggled off → API server rejects new `EndpointSelector` objects;
  existing `Service`-backed `EndpointSlices` remain current.
- Feature gate toggled off then on → controllers resume reconciliation; drifted
  `EndpointSlices` return to sync without manual intervention.

Links will be added once test files are created in `kubernetes/kubernetes`:
[integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=EndpointSelector),
[triage search](https://storage.googleapis.com/k8s-triage/index.html?test=EndpointSelector)

##### e2e tests

<!--
End-to-end tests are not required to be in Testgrid for Alpha.
-->

- `EndpointSelector` created in a live cluster → `EndpointSlices` exist and
  reflect matching pods by name, IP, and endpoint readiness conditions.
- `Service` with a pod selector auto-creates a corresponding `EndpointSelector`
  (compatibility path); existing `EndpointSlices` for that `Service` are
  functionally unchanged.
- Controller-managed `EndpointSelector` with an `ownerReference` is garbage
  collected when the owning resource is deleted.
- `publishNotReadyAddresses: true` causes not-ready pod endpoints to have
  `ready: true` in a live cluster while `serving` continues to reflect actual
  pod readiness.

Links will be added once test files are created in `kubernetes/kubernetes`:
[SIG Network](https://testgrid.k8s.io/sig-network?include-filter-by-regex=EndpointSelector),
[triage search](https://storage.googleapis.com/k8s-triage/index.html?test=EndpointSelector)

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*
-->

#### Alpha

- `EndpointSelector` API type in `discovery.k8s.io/v1alpha1` defined,
  registered, and validated (including `matchExpressions` and empty selector
  rejection).
- `endpointslice-controller` refactored to reconcile from `EndpointSelector`
  objects when the `EndpointSelector` feature gate is enabled.
- `service-endpointselector-controller` creates and deletes an
  `EndpointSelector` for each `Service` with a pod selector.
- Open questions from the Design Details resolved before moving to
  implementable: ownership chain, structural seam, migration switchover,
  orphaned resource enforcement, name validation, and Service-owned field
  ownership strategy.
- Unit and integration tests covering: `EndpointSelector` creation →
  `EndpointSlice` generation; pod readiness condition transitions;
  `EndpointSelector` deletion → `EndpointSlice` garbage collection;
  EndpointSelector-to-EndpointSlice label propagation; Service compatibility
  matrix scenarios; Service-owned `EndpointSelector` direct edits reconciled
  back to Service-derived values; feature gate off → API server rejects
  `EndpointSelector` objects.
- Basic e2e tests enabled (not required to be in Testgrid for Alpha).

#### Beta

- Feedback gathered from Alpha adopters and from [GEP-4731] experimental
  implementations.
- Open questions from Design Details resolved or explicitly deferred with
  written justification.
- `matchExpressions` support added to `spec.selector` (addresses
  kubernetes/kubernetes#48528 and kubernetes/kubernetes#62795) — or
  explicitly deferred with justification.
- Path for `Service` users to opt into set-based endpoint selection resolved
  (for example, controlled edits to Service-owned `EndpointSelectors` or a
  future `Service.spec.selectorRef`) — or explicitly deferred with
  justification.
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

**Enabling the feature gate.** The `EndpointSelector` feature gate may be
enabled on `kube-controller-manager` while `kube-apiserver` nodes are still
being upgraded. If the `EndpointSelector` API is not yet available on all
apiserver nodes, the `endpointslice-controller` detects this via informer
registration errors and falls back to `Service`-watching until the API is
available fleet-wide (see [Version Skew Strategy](#version-skew-strategy)).
`EndpointSlices` remain current throughout the upgrade window. As apiservers
are upgraded, the `service-endpointselector-controller` creates an
`EndpointSelector` for each `Service` with a pod selector and the
`endpointslice-controller` transitions to `EndpointSelector`-watching
automatically.

**Disabling the feature gate.** The API server stops accepting new
`EndpointSelector` objects and the controllers stop reconciling them. Existing
`EndpointSelector` objects remain in etcd but are ignored. The
`endpointslice-controller` returns to reconciling `EndpointSlices` directly
from `Service` objects, so `Service`-backed slices remain current.
`EndpointSlices` owned by manually created `EndpointSelector` objects stop
being updated until the gate is re-enabled.

**Re-enabling after rollback.** The controllers resume reconciliation. The
`service-endpointselector-controller` syncs all `Services` and creates any
missing `EndpointSelector` objects. The `endpointslice-controller` reconciles
all `EndpointSelector` objects, bringing any drifted `EndpointSlices` back into
sync without manual intervention.

### Version Skew Strategy

Kubernetes requires `kube-apiserver` to be upgraded before
`kube-controller-manager`, which is the standard control-plane upgrade order.

The `endpointslice-controller` detects `EndpointSelector` API availability via
informer registration. When the gate is enabled, the controller attempts to
register a watch on `discovery.k8s.io/v1alpha1 EndpointSelector`. If the API
server does not yet have the resource registered, the watch returns a
resource-not-found error. The controller detects this, falls back to
`Service`-watching mode, and retries registration periodically. Once the API
becomes available and registration succeeds, the controller transitions to
`EndpointSelector`-watching automatically. `EndpointSlices` remain current
throughout — during the fallback window they are produced from `Service` objects;
after the transition they are produced from `EndpointSelector` objects.

**n-1 controller-manager (old controller, new API server).** The old
`kube-controller-manager` does not know about `EndpointSelector`. It continues
to watch `Service` objects and produce `EndpointSlices` from them as today.
`EndpointSelector` objects may exist on the new API server if the gate is enabled
there, but nothing reconciles them until the controller-manager is upgraded.

**n+1 controller-manager (new controller, old API server).** The new
`endpointslice-controller` attempts to register an `EndpointSelector` informer;
the old apiserver returns a resource-not-found error, and the controller falls
back to `Service`-watching. `EndpointSlices` continue to be produced from
`Service` objects without interruption. The `service-endpointselector-controller`
similarly retries `EndpointSelector` CREATEs against the old apiserver. As each
apiserver node is upgraded and the `EndpointSelector` API becomes available, the
informer registration succeeds and the controller transitions to
`EndpointSelector`-watching automatically.

**kubelet and kube-proxy** are not involved in `EndpointSelector`
reconciliation. They consume `EndpointSlices` regardless of whether those slices
were produced by a `Service` or an `EndpointSelector`, and are not affected by
version skew in either direction.

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `EndpointSelector`
  - Components depending on the feature gate:
    - `kube-apiserver` (to accept EndpointSelector objects)
    - `kube-controller-manager` (to reconcile EndpointSelector objects)

###### Does enabling the feature change any default behavior?

For new clusters, no existing behavior changes. `EndpointSelector` is a new
resource type; nothing creates `EndpointSelector` objects unless a user or
controller explicitly does so.

For upgraded clusters, the `service-endpointselector-controller` begins
creating an `EndpointSelector` for each existing `Service` with a pod
selector. This is additive — `EndpointSlices` continue to be produced and
served to `kube-proxy` and other consumers without interruption.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Setting the `EndpointSelector` feature gate to false stops the API server
from accepting new `EndpointSelector` objects and stops the controllers from
reconciling them. Existing `EndpointSelector` objects remain in etcd but are
not acted upon. The `endpointslice-controller` reverts to reconciling
`EndpointSlices` directly from `Service` objects, so `Service`-backed slices
remain current. `EndpointSlices` owned by manually created `EndpointSelector`
objects stop being updated until the gate is re-enabled.

###### What happens if we reenable the feature if it was previously rolled back?

The controllers resume reconciliation. The `service-endpointselector-controller`
syncs all `Services` and creates any missing `EndpointSelector` objects. The
`endpointslice-controller` reconciles all `EndpointSelector` objects, bringing
any drifted `EndpointSlices` back into sync. No manual intervention is
required.

###### Are there any tests for feature enablement/disablement?

Yes. Alpha integration tests cover:

- API server rejecting `EndpointSelector` objects when the gate is disabled.
- `endpointslice-controller` skipping reconciliation of `EndpointSelector`
  objects when the gate is disabled.
- `EndpointSlices` written while the gate was enabled drifting gracefully
  (not deleted) when the gate is disabled, and returning to sync when
  re-enabled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

Full analysis required at Beta. Known candidates:

- Skewed rollout: in HA clusters where some API servers have the gate enabled
  and others do not, `EndpointSelector` creates that reach an old apiserver
  return 404 and are retried. The `endpointslice-controller` detects the
  missing API via informer registration errors and falls back to
  `Service`-watching for that window. `EndpointSlices` remain current; no
  workloads are impacted during the upgrade window.
- Object volume burst: clusters with many `Services` generate a large number of
  `EndpointSelector` create calls when the gate is first enabled. The
  `service-endpointselector-controller` is the sole writer and is rate-limited
  by the standard client-go work queue, but the burst may still be visible in
  API server metrics.
- Mid-reconciliation restart: a `kube-controller-manager` restart during the
  initial migration sync leaves a window where some `Services` do not yet have
  a corresponding `EndpointSelector`. This is safe because the controller is
  idempotent — the missing objects are created on the next sync.

###### What specific metrics should inform a rollback?

Specific metric names will be defined at Beta. Signals to monitor:

- Spike in `endpointslice-controller` sync error rate.
- Increase in `EndpointSlice` churn (endpoints added or removed per sync).
- API server error rate for `EndpointSelector` operations.
- Pod readiness transitions not reflected in `EndpointSlices` within the
  expected latency bound.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Manual upgrade → downgrade → upgrade testing will be documented before
targeting Beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No APIs, fields, flags, or features are deprecated at Alpha.

At GA, documentation for the shadow-`Service` pattern — creating a headless
`Service` solely to generate `EndpointSlices` — will be updated to recommend
`EndpointSelector` instead.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

`EndpointSelector` objects can be listed directly:
`kubectl get endpointselectors --all-namespaces`. A dedicated metric will be
added to `kube-controller-manager` before Beta.

###### How can someone using this feature know that it is working for their instance?

- [ ] Other
  - Verify that `EndpointSlice` objects exist and reflect live pod readiness:
    `kubectl get endpointslices -l kubernetes.io/service-name=<name>` for
    `Service`-backed selectors, or
    `kubectl get endpointslices -l kubernetes.io/endpoint-selector-name=<name>`
    to filter by `EndpointSelector` directly.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

`EndpointSelector`-managed `EndpointSlices` should reflect pod readiness
changes within the same latency bounds as `Service`-managed slices. The
existing [EndpointSlice SLO][eps-slo] (pod readiness → slice updated within
1s for small clusters) applies.

[eps-slo]: https://git.k8s.io/community/sig-scalability/slos/slos.md

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Existing `endpointslice-controller` metrics extended with an `owner_type`
    label distinguishing `Service`-owned from `EndpointSelector`-owned slices
    (for example,
    `endpoint_slice_controller_syncs_total{owner_type="EndpointSelector"}`).
    Exact metric names will be defined before Beta.
  - Component exposing the metric: `kube-controller-manager`

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Before Beta, evaluate whether existing `endpointslice-controller` metrics
(`endpoints_added_per_sync`, `endpoints_removed_per_sync`, `syncs_total`) can
be extended with an `owner_type` label rather than introducing new metric
families. This keeps the observability surface small and avoids breaking
existing dashboards.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

- `kube-apiserver`: must have the `EndpointSelector` feature gate enabled to
  accept `EndpointSelector` objects.
- `kube-controller-manager`: must be running and have the gate enabled for the
  `endpointslice-controller` and `service-endpointselector-controller` to
  reconcile `EndpointSelector` objects.

No external services or cloud provider APIs are required.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.
For beta, this section is required.
-->

###### Will enabling / using this feature result in any new API calls?

Yes.

- New watch: `endpointslice-controller` adds a watch on `EndpointSelector`
  objects.
- New writes: `service-endpointselector-controller` creates and deletes
  `EndpointSelector` objects in proportion to `Service` count and churn.
  In steady state, `Service` updates trigger a re-sync but typically produce
  no write if the `EndpointSelector` is already current.
- `EndpointSlice` create/update/delete rate is unchanged for the `Service`
  compatibility path — the same slices are produced from the same pod events.

Throughput estimates relative to existing `EndpointSlice` load will be
benchmarked before Beta.

###### Will enabling / using this feature result in introducing new API types?

Yes: `EndpointSelector` (`discovery.k8s.io/v1alpha1`), namespace-scoped.

For the `Service` compatibility path, the number of `EndpointSelector` objects
scales with the number of `Services` that have a pod selector. Clusters that
run O(10^3–10^4) `Services` today will have a roughly equivalent number of
`EndpointSelector` objects. Third-party controllers (for example, `InferencePool`)
add additional objects beyond this baseline.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

New `EndpointSelector` objects are written to etcd, each roughly the same
size as a `Service` object. For the `Service` compatibility path, one
`EndpointSelector` is created per `Service` with a pod selector.

The total `EndpointSlice` count does not increase for the compatibility
path — the same slices are produced from the same pods, now driven through an
`EndpointSelector` rather than directly from `Service`.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The pod readiness → `EndpointSlice` update path gains one additional level of
indirection (Service → EndpointSelector → EndpointSlice in the reconciler). No
additional API server round-trips are needed since `EndpointSelector` objects
are already in the controller's informer cache. The impact on end-to-end
latency is expected to be negligible, but must be confirmed with benchmarks
before Beta.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

`kube-controller-manager` gains a new informer cache for `EndpointSelector`
objects. Memory overhead is proportional to the number of `EndpointSelector`
objects — the same order of magnitude as the existing `Service` informer cache
for the compatibility path.

CPU overhead from the `service-endpointselector-controller` is proportional
to `Service` churn rate. Writes are gated on diffs, so steady-state cost is
low. Formal benchmarks will be completed before Beta.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. Service-backed `EndpointSlices` remain functionally equivalent from
`kube-proxy`'s perspective. `EndpointSlices` created for direct
`EndpointSelector` consumers do not require new node-level resources.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

Same degradation model as `Service`/`EndpointSlice` today: the controllers
cannot reconcile while the API server is unavailable, and `EndpointSlices` may
become stale as pods change readiness. Existing `EndpointSlices` continue to
serve traffic until the API server recovers and the controllers catch up. There
are no additional failure modes introduced by this feature.

###### What are other known failure modes?

Full documentation required at Beta. Candidates:

- **Controller-managed `EndpointSelector` not garbage collected**: the owning
  resource is deleted but the `EndpointSelector` persists. `ownerReference`
  garbage collection handles this automatically; the
  `app.kubernetes.io/managed-by` label enables manual audit.
- **`EndpointSlices` not updated after pod readiness change**: detectable via
  `endpointslice-controller` sync error metrics. Mitigation: verify feature
  gate status; restart `kube-controller-manager`.

###### What steps should be taken if SLOs are not being met to determine the problem?

Full runbook required at Beta. In Alpha, check `kube-controller-manager` logs
at verbosity level 4 or higher for `endpointslice-controller` sync errors, and
verify the feature gate is enabled on both `kube-apiserver` and
`kube-controller-manager`.

## Implementation History

- 2026-05-26: KEP issue filed ([kubernetes/enhancements#6116](https://github.com/kubernetes/enhancements/issues/6116))

## Drawbacks

- `EndpointSelector` partially overlaps with `Service`. Users may be unsure
  which to use, especially when `Service` semantics are mostly — but not
  entirely — unwanted.
- Auto-creating one `EndpointSelector` per `Service` (the compatibility path)
  roughly doubles the number of objects the selector-reconciliation machinery
  tracks, increasing `kube-controller-manager` memory and API server load
  proportionally.
- The [GEP-4731] experimental track may converge faster than the KEP process.
  If Gateway API implementations standardize on `XEndpointSelector` before this
  KEP reaches GA, migrating them to the upstream API becomes a coordination
  problem across multiple projects.

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
decomposed into composable primitives more broadly. This KEP seeks to be
one step in that direction but does not attempt to solve the entire problem
in one go. A broader vision is discussed in [this slide deck][decompose-svc-slides]
which was [presented at a SIG Network meeting][decompose-svc-recording].

[decompose-svc-slides]: https://docs.google.com/presentation/d/1h_2WYyvIbyyCIMN61FInAfFtaJk_TrYpDXKoHXnUfy8/edit?slide=id.p#slide=id.p
[decompose-svc-recording]: https://youtu.be/OmD_fKasCNA?si=xpcOhcPgUd7_mbQw&t=1083

## Infrastructure Needed (Optional)

None for Alpha. If the shared controller library is extracted to a staging
repository in a future release, that subproject will be noted here.
