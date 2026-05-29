# KEP-6128: LoadBalancer Resource

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Explicit internal load balancer](#story-1-explicit-internal-load-balancer)
    - [Story 2: Observing load balancer provisioning state](#story-2-observing-load-balancer-provisioning-state)
    - [Story 3: In-cluster Gateway API integration](#story-3-in-cluster-gateway-api-integration)
    - [Story 4: Multi-cloud portability](#story-4-multi-cloud-portability)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Definition](#api-definition)
  - [Status](#status)
  - [Conditions](#conditions)
    - [Accepted](#accepted)
    - [Programmed](#programmed)
    - [Degraded](#degraded)
  - [Binding Model](#binding-model)
  - [Validation](#validation)
  - [Interaction with Gateway API](#interaction-with-gateway-api)
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
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Add fields directly to Service (KEP-4631 approach)](#add-fields-directly-to-service-kep-4631-approach)
  - [Use Gateway API for all load balancer configuration](#use-gateway-api-for-all-load-balancer-configuration)
  - [Use CRDs instead of a built-in type](#use-crds-instead-of-a-built-in-type)
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
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes a new `LoadBalancer` resource in the
`networking.k8s.io/v1alpha1` API group that decouples load balancer
configuration and status from the `Service` resource.

Today, load balancer behavior in Kubernetes is configured through a
combination of `Service.Spec` fields (e.g., `sessionAffinity`,
`loadBalancerSourceRanges`) and cloud-provider-specific annotations
(e.g., internal vs external routing, IP allocation, health check
intervals). Load balancer status is limited to the IP/hostname list in
`Service.Status.LoadBalancer`, with no structured way for cloud
providers to communicate provisioning state, feature support, or
degraded behavior.

The new `LoadBalancer` resource provides:

- **Standardized configuration** for common load balancer features
  currently handled through provider-specific annotations
  (routability, IP allocation, health checks).

- **Structured status** with conditions aligned to Gateway API
  semantics (`Accepted`, `Programmed`, `Degraded`).

- **Feature reporting** so cloud providers can advertise supported
  capabilities, enabling e2e tests and users to determine provider
  support without hard-coded skip rules.

- **Gateway API integration point**. In-cluster Gateway
  implementations need load balancers to route and forward traffic
  from external clients to the Gateway proxy. The `LoadBalancer`
  resource provides a structured way to configure and observe this
  routing/forwarding layer without Gateway API having to redefine
  load balancer semantics.

The resource is optional and additive. Existing Services of type
`LoadBalancer` continue to work unchanged. When a `LoadBalancer`
resource exists with the same name as a Service in the same namespace,
the cloud controller manager uses it as the source of truth, falling
back to `Service.Spec` for unset fields.

## Motivation

The current model for configuring load balancers in Kubernetes has
several problems:

1. **Configuration is scattered across annotations.** Core load
   balancer behaviors like internal vs external routing, IP allocation
   strategy, and health check configuration are handled through
   cloud-provider-specific annotations. Every cloud provider has
   invented its own annotation names for the same concepts (e.g.,
   `service.beta.kubernetes.io/aws-load-balancer-internal`,
   `networking.gke.io/internal-load-balancer`,
   `service.beta.kubernetes.io/azure-load-balancer-internal`). This
   makes multi-cloud deployments and tooling unnecessarily difficult.

2. **Status reporting is inadequate.** `Service.Status.LoadBalancer`
   only contains a list of ingress IPs/hostnames. There is no
   structured way for a cloud provider to communicate whether a load
   balancer is being provisioned, has failed, is serving traffic, or
   is operating in a degraded mode. This was the primary motivation
   behind [KEP-4631], which attempted to add conditions directly to
   the Service resource but was never implemented.

3. **Service is overloaded.** The `Service` resource serves multiple
   purposes: service discovery, internal cluster routing, and
   external load balancer management. Mixing load balancer
   infrastructure concerns with service abstraction concerns makes
   the API harder to reason about and evolve independently.

4. **In-cluster Gateway API implementations need load balancer
   infrastructure.** Gateway API implementations that run their
   proxy inside the cluster need load balancers to route and forward
   traffic from external clients to the proxy pods. Today, this is
   done through `Service` of type `LoadBalancer` with cloud-specific
   annotations. A standalone `LoadBalancer` resource provides a
   structured way to configure and observe this routing/forwarding
   layer. (Gateway API implementations that run externally and manage
   their own load balancer infrastructure are not affected.)

5. **E2e tests cannot determine provider capabilities.** As
   identified in [KEP-4631] and [kubernetes/kubernetes#123714], load
   balancer e2e tests have historically hard-coded provider-specific
   skip rules. With in-tree cloud providers removed, there is no
   scalable way for tests to determine which features a cloud provider
   supports.

[KEP-4631]: https://github.com/kubernetes/enhancements/pull/4632
[kubernetes/kubernetes#123714]: https://github.com/kubernetes/kubernetes/issues/123714

### Goals

- Define a new namespaced `LoadBalancer` resource in the
  `networking.k8s.io/v1alpha1` API group with explicit fields for
  common load balancer configuration.

- Provide structured status reporting with conditions (`Accepted`,
  `Programmed`, `Degraded`) aligned with Gateway API conventions.

- Mirror load balancer addresses (IPs/hostnames) in
  `LoadBalancer.Status` so users can observe load balancer state from
  a single resource.

- Provide a feature reporting mechanism so cloud providers can
  advertise supported capabilities.

- Maintain full backward compatibility: existing `Service` of type
  `LoadBalancer` without a corresponding `LoadBalancer` resource must
  continue to work exactly as today.

- Provide a clear integration point for in-cluster Gateway API
  implementations to configure and observe the load balancer
  routing/forwarding layer without redefining these semantics.

### Non-Goals

- Replacing or deprecating `Service` of type `LoadBalancer`. The
  `Service` resource remains the primary way to expose workloads.

- Requiring all cloud providers to support the new resource. Cloud
  controller managers opt in to `LoadBalancer` resource support.

- Moving service-routing concerns (`externalTrafficPolicy`,
  `internalTrafficPolicy`, `ipFamilies`, `ipFamilyPolicy`,
  `allocateLoadBalancerNodePorts`) to the `LoadBalancer` resource.
  These are service behavior, not load balancer infrastructure
  concerns.

- Defining how cloud providers should migrate from annotation-based
  configuration to the new resource. Migration strategy is left to
  individual cloud providers.

- Defining a mechanism for cloud-provider-specific extension fields
  beyond the common set. Cloud providers may continue to use
  annotations for provider-specific features.

## Proposal

### User Stories

#### Story 1: Explicit internal load balancer

A platform team wants to create an internal load balancer without
relying on provider-specific annotations. They create a `LoadBalancer`
resource with `routability: Internal` alongside their Service:

```yaml
apiVersion: networking.k8s.io/v1alpha1
kind: LoadBalancer
metadata:
  name: my-service
  namespace: default
spec:
  routability: Internal
---
apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: default
spec:
  type: LoadBalancer
  ports:
    - port: 80
      targetPort: 8080
  selector:
    app: my-app
```

#### Story 2: Observing load balancer provisioning state

A user creates a `Service` of type `LoadBalancer` and a corresponding
`LoadBalancer` resource. Instead of repeatedly checking whether
`Service.Status.LoadBalancer.Ingress` is populated, they can watch
the `LoadBalancer` resource conditions:

```
$ kubectl get loadbalancer my-service -o jsonpath='{.status.conditions}'
[
  {"type": "Accepted", "status": "True", ...},
  {"type": "Programmed", "status": "True", ...}
]
```

#### Story 3: In-cluster Gateway API integration

An in-cluster Gateway API implementation creates a `LoadBalancer`
resource to configure the routing/forwarding infrastructure that
delivers external traffic to the Gateway proxy pods, instead of
relying on cloud-specific annotations on a Service. The Gateway
controller watches `LoadBalancer` status to determine when the
infrastructure is ready to receive traffic.

#### Story 4: Multi-cloud portability

An organization deploying the same application across AWS, GCP, and
Azure uses a `LoadBalancer` resource with `routability: Internal` and
`loadBalancerSourceRanges` instead of maintaining three different sets
of cloud-specific annotations per environment.

### Notes/Constraints/Caveats

- The `LoadBalancer` resource is bound to a `Service` by name within
  the same namespace. This follows the same pattern as `Service` and
  `Endpoints`.

- Cloud controller managers are not required to support the
  `LoadBalancer` resource. If a cloud controller manager does not
  recognize the resource, the `Service` continues to work as it does
  today.

- When a `LoadBalancer` resource exists, it serves as the source of
  truth for the fields it defines. Fallback semantics depend on the
  field type:
  - **Pointer fields** (`Routability`, `SessionAffinity`,
    `HealthCheck`, `ServiceRef`): `nil` means "no opinion, fall back
    to Service." A non-nil value is the source of truth.
  - **Slice fields** (`SourceRanges`): `nil` means "no opinion, fall
    back to Service." An explicit empty slice (`[]`) means "the
    LoadBalancer explicitly sets no value" and does **not** fall back
    to the Service. For `SourceRanges`, `nil` falls back to
    `Service.Spec.LoadBalancerSourceRanges`, while `[]` means "no
    source range restrictions."

### Risks and Mitigations

**Risk: Cloud provider adoption.** If cloud providers do not adopt
the new resource, users gain no benefit.

*Mitigation:* The resource is designed to be optional and additive.
Cloud providers can adopt it incrementally. The structured status
reporting alone provides enough value to motivate adoption, as it
solves the e2e testing problem identified in KEP-4631.

**Risk: Configuration drift between Service and LoadBalancer.** Users
may set conflicting values on the Service and LoadBalancer resource.

*Mitigation:* The precedence rule is clear: `LoadBalancer` resource
fields take precedence when set, with fallback to `Service.Spec`
fields when unset. Cloud controller managers should set a condition or
event when they detect conflicting configuration.

**Risk: RBAC misconfiguration allows users to overwrite status.**
The `LoadBalancer.Status` should only be written by the cloud
controller manager, not by end users.

*Mitigation:* RBAC policies must ensure that only the cloud
controller manager's service account has `update` and `patch`
permissions on the `loadbalancers/status` subresource. Default
ClusterRoles should be configured accordingly.

**Risk: Lateral configuration override via spec create permissions.**
A user or controller with `create` permissions on `loadbalancers` in
a namespace can create a `LoadBalancer` resource that silently alters
the behavior of any `Service` of type `LoadBalancer` in that namespace
(e.g., changing routability or source ranges) without the Service
owner's knowledge.

<<[UNRESOLVED rbac-binding-security]>>

The name-match binding model means that any principal with `create`
permissions on `loadbalancers` can affect any Service in the same
namespace. Options to mitigate:

1. **Restrict `loadbalancers` creation** to the same set of
   principals that can create Services. This is the simplest approach
   but may be too broad.

2. **Require an explicit opt-in annotation on the Service** (e.g.,
   `networking.k8s.io/loadbalancer-resource: "true"`) before the
   cloud controller manager considers a matching `LoadBalancer`
   resource. This prevents unsolicited overrides.

3. **Use an explicit `serviceRef` field** on the `LoadBalancer`
   resource (see Binding Model section) which could enable
   admission-time validation of cross-resource permissions.

4. **Rely on namespace-level trust boundaries.** Kubernetes
   namespaces are already trust boundaries — principals with write
   access to a namespace can already modify Services, Endpoints, and
   other resources that affect networking behavior.

<<[/UNRESOLVED]>>

## Design Details

### API Definition

The `LoadBalancer` resource is defined in the
`networking.k8s.io/v1alpha1` API group:

```golang
// LoadBalancer defines the desired configuration and reports the
// status of a load balancer associated with a Service of type
// LoadBalancer. The LoadBalancer must have the same name and
// namespace as the Service it is associated with.
type LoadBalancer struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   LoadBalancerSpec   `json:"spec,omitempty"`
    Status LoadBalancerStatus `json:"status,omitempty"`
}

type LoadBalancerSpec struct {
    // Routability indicates whether the load balancer should be
    // reachable from the public internet (External) or only from
    // within the network (Internal). If unset, the cloud controller
    // manager falls back to the Service annotations or its own
    // default behavior.
    // +optional
    Routability *LoadBalancerRoutability `json:"routability,omitempty"`

    // SessionAffinity defines the session affinity type and
    // configuration for the load balancer. If nil, the cloud
    // controller manager falls back to Service.Spec.SessionAffinity
    // and Service.Spec.SessionAffinityConfig.
    // +optional
    SessionAffinity *LoadBalancerSessionAffinity `json:"sessionAffinity,omitempty"`

    // SourceRanges restricts traffic to the load balancer to the
    // specified client IP ranges. If nil, the cloud controller
    // manager falls back to Service.Spec.LoadBalancerSourceRanges.
    // An explicit empty list means "no source range restrictions"
    // and does not fall back to the Service.
    // +optional
    SourceRanges []CIDR `json:"sourceRanges,omitempty"`

    // HealthCheck configures how the cloud load balancer performs
    // health checks against backends. These are cloud infrastructure
    // health checks, independent of Pod readiness probes.
    // If nil, the cloud controller manager uses its default health
    // check configuration.
    // +optional
    HealthCheck *HealthCheckConfig `json:"healthCheck,omitempty"`

    // ServiceRef is an optional explicit reference to the Service
    // this LoadBalancer is associated with. If unset, the
    // LoadBalancer binds to a Service with the same name in the
    // same namespace. When set, the name-match binding is not used
    // and the referenced Service is used instead.
    // +optional
    ServiceRef *ServiceReference `json:"serviceRef,omitempty"`
}

// CIDR is a string type that represents a network in CIDR notation
// (e.g., "203.0.113.0/24", "2001:db8::/32"). Values must be valid
// CIDR notation and are validated at admission time.
// +kubebuilder:validation:Format=cidr
type CIDR string

// ServiceReference identifies a Service in the same namespace.
type ServiceReference struct {
    // Name is the name of the Service.
    Name string `json:"name"`
}

// LoadBalancerSessionAffinity defines session affinity mode and
// configuration for a load balancer.
type LoadBalancerSessionAffinity struct {
    // Type selects the session affinity mode.
    // Valid values are "None" and "ClientIP".
    // +enum
    Type corev1.ServiceAffinity `json:"type"`

    // Config holds additional session affinity configuration.
    // Applicable only when Type is "ClientIP".
    // +optional
    Config *corev1.SessionAffinityConfig `json:"config,omitempty"`
}

// HealthCheckConfig configures how the cloud load balancer
// performs health checks against backends. These are
// infrastructure-level health checks performed by the load
// balancer itself, independent of Kubernetes readiness probes.
type HealthCheckConfig struct {
    // Interval is the time between consecutive health checks.
    // +optional
    Interval *metav1.Duration `json:"interval,omitempty"`

    // Path is the HTTP path used for health checks. If set, the
    // load balancer performs HTTP health checks against this path
    // on each backend. If unset, the load balancer uses TCP or
    // its default health check mechanism.
    // This is particularly useful for Gateway API integration,
    // where the load balancer may need to health check a proxy
    // rather than individual pods.
    // +optional
    Path *string `json:"path,omitempty"`

    // Port is the port used for health checks. If unset, the
    // load balancer uses the first service port or its default.
    // +optional
    Port *int32 `json:"port,omitempty"`

    // HealthyThreshold is the number of consecutive successful
    // health checks required before a backend is considered
    // healthy.
    // +optional
    HealthyThreshold *int32 `json:"healthyThreshold,omitempty"`

    // UnhealthyThreshold is the number of consecutive failed
    // health checks required before a backend is considered
    // unhealthy.
    // +optional
    UnhealthyThreshold *int32 `json:"unhealthyThreshold,omitempty"`
}

// LoadBalancerRoutability defines whether a load balancer is
// externally or internally routable.
// +enum
type LoadBalancerRoutability string

const (
    // LoadBalancerRoutabilityExternal indicates the load balancer
    // is reachable from the public internet.
    LoadBalancerRoutabilityExternal LoadBalancerRoutability = "External"

    // LoadBalancerRoutabilityInternal indicates the load balancer
    // is only reachable from within the network.
    LoadBalancerRoutabilityInternal LoadBalancerRoutability = "Internal"
)
```

<<[UNRESOLVED ip-allocation]>>

IP address allocation is a common load balancer feature supported
across cloud providers (e.g., AWS EIP allocations, GKE IP address
references). The shape of this field needs further discussion, as
different providers model this differently:

- AWS: `service.beta.kubernetes.io/aws-load-balancer-eip-allocations`
  (references to Elastic IP allocation IDs)
- GKE: `networking.gke.io/load-balancer-ip-addresses` (references
  to GCP address resources)
- Azure: Public IP resource IDs

A potential approach is a string slice of provider-interpreted
references:

```golang
    // IPAddressAllocations specifies pre-allocated IP address
    // resources to use for the load balancer. The format of these
    // references is cloud-provider-specific.
    // +optional
    IPAddressAllocations []string `json:"ipAddressAllocations,omitempty"`
```

<<[/UNRESOLVED]>>

<<[UNRESOLVED subnet-selection]>>

Subnet selection is common but highly implementation-specific. Some
providers use subnet IDs, others use tags or names. This may be
better left to annotations or a future KEP.

<<[/UNRESOLVED]>>


<<[UNRESOLVED loadbalancer-class]>>

`Service.Spec.LoadBalancerClass` selects which cloud controller
manages the load balancer. If a `LoadBalancer` resource also specifies
a class, and it conflicts with the Service's class, the behavior is
undefined. Options:

1. The `LoadBalancer` resource's class takes precedence.
2. The class must match if both are set, otherwise the cloud
   controller sets an error condition.
3. The class field only lives on the `LoadBalancer` resource and is
   removed from Service in a future version.

<<[/UNRESOLVED]>>

### Status

```golang
type LoadBalancerStatus struct {
    // Conditions describe the current state of the LoadBalancer.
    // Known condition types are "Accepted", "Programmed", and
    // "Degraded".
    // +optional
    // +listType=map
    // +listMapKey=type
    Conditions []metav1.Condition `json:"conditions,omitempty"`

    // Addresses contains the IP addresses and/or hostnames assigned
    // to the load balancer. These addresses are also reported in
    // Service.Status.LoadBalancer.Ingress for backward compatibility.
    // +optional
    // +listType=map
    // +listMapKey=ip
    Addresses []LoadBalancerAddress `json:"addresses,omitempty"`

    // SupportedFeatures indicates the set of features that the cloud
    // provider's load balancer implementation supports. This allows
    // users and e2e tests to determine capabilities without
    // provider-specific knowledge.
    // +optional
    // +listType=set
    SupportedFeatures []LoadBalancerFeature `json:"supportedFeatures,omitempty"`
}

// LoadBalancerAddress represents an address assigned to the load
// balancer. This struct mirrors the fields of
// corev1.LoadBalancerIngress to maintain consistency with
// Service.Status.LoadBalancer.Ingress.
type LoadBalancerAddress struct {
    // IP is the IP address assigned to the load balancer.
    // +optional
    IP string `json:"ip,omitempty"`

    // Hostname is the hostname assigned to the load balancer.
    // +optional
    Hostname string `json:"hostname,omitempty"`

    // IPMode specifies how the load balancer IP behaves.
    // "VIP" means the IP is a virtual IP that is directly
    // routable. "Proxy" means the IP belongs to a proxy that
    // forwards traffic to the backends.
    // +optional
    IPMode *corev1.LoadBalancerIPMode `json:"ipMode,omitempty"`

    // Ports is the list of ports exposed on this address.
    // +optional
    // +listType=atomic
    Ports []corev1.PortStatus `json:"ports,omitempty"`
}

// LoadBalancerFeature identifies a load balancer feature that a
// cloud provider may or may not support.
type LoadBalancerFeature string

const (
    LoadBalancerFeatureSessionAffinity LoadBalancerFeature = "SessionAffinity"
    LoadBalancerFeatureSourceRanges    LoadBalancerFeature = "SourceRanges"
    LoadBalancerFeatureInternalRouting LoadBalancerFeature = "InternalRouting"
    LoadBalancerFeatureIPv6            LoadBalancerFeature = "IPv6"
    LoadBalancerFeatureUDP             LoadBalancerFeature = "UDP"
    LoadBalancerFeatureSCTP            LoadBalancerFeature = "SCTP"
    LoadBalancerFeatureMixedProtocol   LoadBalancerFeature = "MixedProtocol"
)
```

<<[UNRESOLVED feature-reporting]>>

The `SupportedFeatures` field on `LoadBalancer.Status` reports
per-instance feature support. An alternative is a cluster-scoped
mechanism (similar to `StorageClass` or `IngressClass`, or Gateway
API's `SupportedFeatures` on `GatewayClass`) where a cloud provider
advertises its general capabilities once, rather than per-instance.

Arguments for per-instance:
- Features may genuinely vary per-instance (e.g., internal LBs may
  support different features than external ones).
- Simpler to implement — no additional resource type needed.

Arguments for per-class (cluster-scoped):
- Avoids duplicating the same feature list across every
  `LoadBalancer` instance (wasteful at scale).
- Consistent with Gateway API's `GatewayClass` pattern.
- Semantically correct: features are a property of the
  provider/class, not individual instances.

A hybrid approach is also possible: per-class features for general
capabilities, with per-instance overrides for configuration-dependent
support.

<<[/UNRESOLVED]>>

<<[UNRESOLVED service-status-interaction]>>

When a `LoadBalancer` resource exists alongside a Service, the
relationship between `LoadBalancer.Status.Addresses` and
`Service.Status.LoadBalancer.Ingress` must be clearly defined:

1. **The cloud controller manager MUST continue populating
   `Service.Status.LoadBalancer.Ingress`** regardless of whether a
   `LoadBalancer` resource exists. kube-proxy, DNS controllers,
   ingress controllers, and other tools read Service status. Changing
   this behavior would break backward compatibility.

2. **`LoadBalancer.Status.Addresses` is a mirror**, populated by the
   cloud controller manager at the same time as
   `Service.Status.LoadBalancer.Ingress`. The cloud controller
   manager is the source of truth for both.

3. **If the two diverge** (e.g., due to a bug or race condition),
   `Service.Status.LoadBalancer.Ingress` takes precedence for traffic
   routing purposes, since kube-proxy and other data-plane components
   do not watch the `LoadBalancer` resource.

This effectively means `LoadBalancer.Status.Addresses` is a
convenience copy for users and tooling that want to observe the full
load balancer state from a single resource.

<<[/UNRESOLVED]>>

### Conditions

The `LoadBalancer` resource uses conditions aligned with Gateway API
semantics:

#### Accepted

The `Accepted` condition indicates that the cloud controller manager
has seen the `LoadBalancer` resource and acknowledges responsibility
for it.

- `Status=True, Reason=Accepted`: The controller has accepted the
  resource and will attempt to provision a load balancer.
- `Status=False, Reason=Unsupported`: The controller cannot provision
  a load balancer for this configuration (explained in `Message`).
- `Status=False, Reason=Conflict`: The `LoadBalancer` resource
  conflicts with the corresponding Service (e.g., mismatched
  `loadBalancerClass`).
- `Status=False, Reason=InvalidService`: The matching Service does
  not exist, is not of type `LoadBalancer`, or the `serviceRef`
  points to a non-existent Service.
- `Status=False, Reason=ServiceNotFound`: The `serviceRef` references
  a Service that does not exist.

If no `Accepted` condition appears within a reasonable time after
creation, the user should assume no cloud controller manager is
handling this resource.

#### Programmed

The `Programmed` condition indicates that the load balancer
infrastructure has been configured in the data plane. Following
Gateway API semantics, `Programmed=True` means the load balancer
configuration has been accepted and programmed into the underlying
infrastructure, but traffic may not yet be flowing due to
propagation delays (e.g., DNS propagation for hostname-based load
balancers, or cloud infrastructure eventual consistency).

- `Status=True, Reason=Programmed`: The load balancer is configured
  in the data plane. Addresses in `.Status.Addresses` are allocated
  but may have a brief propagation delay before traffic flows.
- `Status=False, Reason=Provisioning`: The controller is in the
  process of provisioning the load balancer.
- `Status=False, Reason=Infrastructure`: Provisioning failed due to
  an infrastructure problem (explained in `Message`).

When `Programmed` transitions to `True`, the load balancer addresses
must be populated in `.Status.Addresses`.

#### Degraded

The `Degraded` condition indicates that the load balancer is
provisioned but is not fully implementing all requested semantics.

- `Status=True, Reason=SessionAffinityNotSupported`: Session affinity
  was requested but is not supported by this load balancer.
- `Status=True, Reason=SourceRangesNotSupported`: Source range
  filtering was requested but is not supported.
- `Status=True, Reason=Multiple`: The load balancer is degraded for
  multiple reasons (detailed in `Message`).
- `Status=Unknown` (or unset): The controller does not have enough
  information to determine whether the load balancer is degraded.

<<[UNRESOLVED degraded-condition]>>

The `Degraded` condition serves the same purpose as
`LoadBalancerDegraded` from KEP-4631. An alternative is to remove
this condition and rely solely on the feature reporting mechanism
(`SupportedFeatures`) combined with `Accepted: False` for hard
failures. The `Degraded` condition provides a more immediate signal
that something is "working but not as requested."

<<[/UNRESOLVED]>>

### Binding Model

By default, a `LoadBalancer` resource is bound to a `Service` by
matching name and namespace. This follows the same convention as
`Service` and `Endpoints`.

Alternatively, the `LoadBalancer` resource can use the optional
`spec.serviceRef` field to explicitly reference a Service by name.
When `serviceRef` is set, name-match binding is not used. This
enables scenarios where the `LoadBalancer` and `Service` have
different names (e.g., when a Gateway controller manages the
lifecycle) and provides a clearer binding intent.

- The `LoadBalancer` resource can be created before, after, or at
  the same time as the corresponding Service.
- If a `LoadBalancer` resource exists but no corresponding `Service`
  of type `LoadBalancer` exists (either by name-match or
  `serviceRef`), the cloud controller manager should set the
  `Accepted` condition to `False` with `Reason=InvalidService` until
  a matching Service appears.
- If a `Service` of type `LoadBalancer` exists without a
  corresponding `LoadBalancer` resource, the cloud controller manager
  behaves exactly as it does today.

<<[UNRESOLVED ownership-model]>>

The ownership model between `LoadBalancer` and `Service` needs
further discussion:

1. **Service owns LoadBalancer**: The `LoadBalancer` resource has an
   owner reference to the Service. Deleting the Service deletes the
   LoadBalancer. Deleting the LoadBalancer reverts to Service-only
   behavior.

2. **LoadBalancer owns Service**: The `LoadBalancer` resource is the
   primary object. This is unusual in Kubernetes and likely not the
   right model.

3. **No ownership**: Both resources are independent. The cloud
   controller manager observes both and reconciles. Deleting either
   requires explicit cleanup by the user or controller.

4. **Cloud provider decides**: The ownership model is left to the
   cloud controller manager implementation. This provides flexibility
   but may lead to inconsistent behavior across providers.

This decision affects deletion semantics: what happens when a user
deletes the `LoadBalancer` resource while the Service still exists.

<<[/UNRESOLVED]>>

<<[UNRESOLVED immutability]>>

Whether any `LoadBalancer.Spec` fields should be immutable after
creation is unclear. Candidates for immutability might include
`routability` (switching between internal and external may require
tearing down and reprovisioning the entire LB). However, making
fields mutable and letting the cloud controller reconcile is
simpler and more consistent with Kubernetes conventions.

<<[/UNRESOLVED]>>

<<[UNRESOLVED dual-stack]>>

Dual-stack Services receive both IPv4 and IPv6 addresses. The
`LoadBalancer` resource does not currently model IP family
preferences or requirements. Questions:

1. Should the `LoadBalancer` resource have an `ipFamilies` or
   `ipFamilyPolicy` field, or is this purely a Service concern?

2. If a cloud provider can only provision a single-stack load
   balancer for a dual-stack Service, should this be reflected as
   `Degraded=True` on the `LoadBalancer` resource?

3. Should `LoadBalancer.Status.Addresses` include family information
   to distinguish IPv4 and IPv6 addresses?

For alpha, dual-stack behavior is inherited from the Service and
the `LoadBalancer` resource does not add any IP-family-specific
fields. This may need to change in beta based on feedback.

<<[/UNRESOLVED]>>

### Validation

**Admission-time validation** (kube-apiserver):

- `Spec.SourceRanges` entries must be valid CIDR notation.
- `Spec.Routability`, if set, must be one of `External` or `Internal`.
- `Spec.SessionAffinity.Type`, if set, must be one of `None` or
  `ClientIP`.
- `Spec.HealthCheck` fields must be non-negative where applicable.
- `Spec.ServiceRef.Name`, if set, must be a valid DNS subdomain name.

**Reconciliation-time validation** (cloud controller manager):

- If a `LoadBalancer` resource exists but the matching Service is not
  of type `LoadBalancer` (or does not exist), the cloud controller
  manager sets `Accepted=False` with `Reason=InvalidService` and
  `Message` explaining that the referenced Service must be of type
  `LoadBalancer`.

- If a `LoadBalancer` resource uses `serviceRef` to reference a
  Service that does not exist, the cloud controller manager sets
  `Accepted=False` with `Reason=ServiceNotFound`.

- If the `LoadBalancer` resource specifies a `loadBalancerClass` that
  conflicts with the Service's class, the cloud controller manager
  sets `Accepted=False` with `Reason=Conflict`.

Reconciliation-time validation is appropriate here because the
`LoadBalancer` resource may be created before the corresponding
Service, and admission validation cannot enforce cross-resource
constraints.

### Interaction with Gateway API

In-cluster Gateway API implementations that need load balancers to
route and forward external traffic to their proxy pods can create a
`LoadBalancer` resource instead of configuring a Service with
cloud-specific annotations. This allows the Gateway controller to:

- Specify load balancer configuration (routability, source ranges)
  in a structured, portable way.
- Observe load balancer provisioning state through conditions,
  without polling `Service.Status.LoadBalancer.Ingress`.
- Avoid redefining load balancer routing/forwarding semantics within
  the Gateway API itself.

The `LoadBalancer` resource handles the routing/forwarding layer
(VIPs, ports, delivering traffic to backends) while the Gateway
handles proxying semantics. Gateway API implementations that run
externally and manage their own load balancer infrastructure are
not affected by this KEP.

### Test Plan

[x] I/we understand the owners of the involved components may require
updates to existing tests to make this code solid enough prior to
committing the changes necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- Validation and defaulting of the `LoadBalancer` API type.
- Conversion tests for field presence/absence with the feature gate
  enabled and disabled.
- RBAC tests ensuring `/status` subresource is only writable by
  controllers.

- `k8s.io/apiserver`: `<date>` - `<test coverage>`
- `k8s.io/cloud-provider`: `<date>` - `<test coverage>`

##### Integration tests

- Creating a `LoadBalancer` resource with and without a corresponding
  Service verifies the binding behavior.
- Creating a `LoadBalancer` with conflicting fields from the Service
  verifies the precedence behavior.
- Feature gate enable/disable tests for the new API type.

##### e2e tests

- Cloud provider implementations (cloud-provider-kind at minimum for
  alpha) should demonstrate:
  - Setting `Accepted` and `Programmed` conditions on the
    `LoadBalancer` resource.
  - Populating `.Status.Addresses` when the load balancer is
    provisioned.
  - Falling back to Service.Spec when `LoadBalancer` fields are unset.
  - Using `LoadBalancer.Spec.Routability` to provision internal vs
    external load balancers.
  - Reporting `SupportedFeatures` in status.

### Graduation Criteria

#### Alpha

- `LoadBalancer` API type implemented behind the `LoadBalancerResource`
  feature gate in `kube-apiserver`.
- Helpers for the new API implemented in `k8s.io/cloud-provider`.
- At least one cloud provider implementation (cloud-provider-kind)
  supporting the new resource with conditions and status.
- Initial e2e tests demonstrating the lifecycle.
- Documentation describing the new resource and its relationship
  to Services.

#### Beta

- At least two additional cloud provider implementations support the
  new resource.
- Feedback from alpha users has been addressed.
- All open questions in this KEP have been resolved.
- E2e tests covering all spec fields and status conditions.
- Documentation updated with migration guidance for cloud providers.

#### GA

- At least two cloud providers from different categories (public
  cloud and non-public-cloud) fully support the resource.
- At least one Gateway API implementation uses the `LoadBalancer`
  resource for load balancer routing/forwarding infrastructure.
- Allowing time for feedback.
- Conformance tests covering the core `LoadBalancer` resource
  behavior.

### Upgrade / Downgrade Strategy

**Upgrade:** When the `LoadBalancerResource` feature gate is enabled,
the `LoadBalancer` API type becomes available. Existing `Service`
resources of type `LoadBalancer` are unaffected. Cloud controller
managers that support the new resource will start watching for
`LoadBalancer` resources and reconciling them alongside Services.

**Downgrade:** When the feature gate is disabled, the `LoadBalancer`
API type is no longer served. Existing `LoadBalancer` resources remain
in etcd but are inaccessible. Services continue to function as they
did before the feature was enabled. Cloud controller managers should
handle the absence of the API gracefully (e.g., via discovery checks).

### Version Skew Strategy

The `LoadBalancer` resource introduces a new API type, so version
skew concerns are limited:

- **API server skew**: In HA clusters during a rolling upgrade, some
  API servers may serve the `LoadBalancer` type while others do not.
  Cloud controller managers should use API discovery to confirm the
  type is available before watching it.

- **Cloud controller manager skew**: An older cloud controller
  manager that does not know about the `LoadBalancer` resource will
  simply ignore it. The Service continues to function as today.

- **No node-level impact**: The `LoadBalancer` resource is a control
  plane concept. Kubelets and kube-proxy are unaffected.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `LoadBalancerResource`
  - Components depending on the feature gate:
    - `kube-apiserver`: Serves the new `LoadBalancer` API type.
  - Cloud controller managers (out-of-tree) opt in to the new
    resource via API discovery, not via a feature gate. If the
    `LoadBalancer` API type is present in the API server, the CCM
    may choose to watch and reconcile it.

###### Does enabling the feature change any default behavior?

No. The `LoadBalancer` resource is optional. Existing Services of
type `LoadBalancer` behave identically whether or not the feature
gate is enabled. New behavior only occurs when a user explicitly
creates a `LoadBalancer` resource.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate stops serving the `LoadBalancer` API
type. Existing Services continue to function. Cloud controller
managers that supported the resource will fall back to Service-only
behavior.

Existing `LoadBalancer` objects remain in etcd but are inaccessible.
They can be cleaned up by re-enabling the feature gate and deleting
them, or by etcd maintenance.

###### What happens if we reenable the feature if it was previously rolled back?

Previously created `LoadBalancer` resources in etcd become accessible
again. Cloud controller managers will reconcile them as if they were
newly created.

###### Are there any tests for feature enablement/disablement?

Unit tests will verify that the `LoadBalancer` API type is only served
when the feature gate is enabled, and that cloud controller managers
handle the absence of the API type gracefully.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout cannot impact already running workloads because the feature
is purely additive. Existing Services of type `LoadBalancer` are
unaffected.

A rollback (disabling the feature gate) is safe for existing
workloads. The only impact is that users who created `LoadBalancer`
resources will lose the additional configuration and status reporting
those resources provided. Load balancers themselves continue to
function via the Service resource.

###### What specific metrics should inform a rollback?

- Errors in cloud-controller-manager logs related to `LoadBalancer`
  resource reconciliation.
- Unexpected changes to Services of type `LoadBalancer` that do not
  have a corresponding `LoadBalancer` resource.
- API server errors related to the new API type.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. This feature is purely additive.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

By checking for the existence of `LoadBalancer` resources in the
cluster:

```
kubectl get loadbalancers --all-namespaces
```

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Condition name: `Accepted`, `Programmed`
  - Other field: `.status.addresses`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The time for a cloud controller manager to set the `Accepted`
condition after a `LoadBalancer` resource is created should be
comparable to other controller reconciliation times (within 30
seconds under normal conditions).

The time to reach `Programmed=True` depends on the cloud provider's
infrastructure provisioning time and is not bounded by this KEP.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name: `loadbalancer_resource_reconcile_duration_seconds`
  - Aggregation method: histogram
  - Components exposing the metric: cloud-controller-manager
- [ ] Other (treat as last resort)
  - Details: `LoadBalancer.Status.Conditions` provide direct
    observability of the reconciliation state.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A metric tracking the count of `LoadBalancer` resources by condition
status (e.g., how many are in `Programmed=True` vs
`Programmed=False`) would be useful for cluster-level observability
but is not required for alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- Cloud controller manager
  - Usage description: Reconciles `LoadBalancer` resources and
    provisions load balancer infrastructure.
  - Impact of its outage on the feature: `LoadBalancer` resources
    will not be reconciled. Conditions will not be updated. Existing
    load balancers continue to function.
  - Impact of its degraded performance or high-error rates on the
    feature: Delayed reconciliation of `LoadBalancer` resources.

### Scalability

###### Will enabling / using this feature result in any new API calls?

- The cloud controller manager will LIST/WATCH `LoadBalancer`
  resources.
- The cloud controller manager will PATCH `LoadBalancer` status
  when reconciling.
- Estimated throughput: proportional to the number of Services of
  type `LoadBalancer` (typically low, tens to hundreds per cluster).

###### Will enabling / using this feature result in introducing new API types?

- API type: `LoadBalancer` (networking.k8s.io/v1alpha1)
- Supported number of objects per cluster: same as Services of type
  LoadBalancer (typically tens to hundreds)
- Supported number of objects per namespace: no specific limit beyond
  the general namespace object limit

###### Will enabling / using this feature result in any new calls to the cloud provider?

No. The cloud controller manager already provisions load balancers
for Services. The `LoadBalancer` resource provides additional
configuration input but does not trigger additional provisioning
calls beyond what the Service already triggers.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. The `LoadBalancer` resource is a new object type, not an addition
to existing objects. Existing Service objects are not modified.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Minimal increase in cloud-controller-manager memory for watching the
additional resource type. Minimal increase in etcd storage for
`LoadBalancer` objects. Both are proportional to the number of
`LoadBalancer` resources, which is bounded by the number of
LoadBalancer Services.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This is a control plane feature with no node-level impact.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The cloud controller manager cannot watch or update `LoadBalancer`
resources. Existing load balancers continue to function. Once the
API server recovers, the controller reconciles any pending changes.

###### What are other known failure modes?

- Cloud controller manager does not support the `LoadBalancer`
  resource:
  - Detection: `LoadBalancer` resources have no conditions set.
  - Mitigations: Services continue to function as today. Users
    should check cloud controller manager version and documentation.
  - Diagnostics: Absence of conditions on the `LoadBalancer` resource
    indicates the controller is not watching this type.
  - Testing: E2e tests verify that Services work without a
    `LoadBalancer` resource.

- Conflicting configuration between Service and LoadBalancer:
  - Detection: Cloud controller manager sets a condition or event
    indicating the conflict.
  - Mitigations: User resolves the conflict by updating either the
    Service or the LoadBalancer resource.
  - Diagnostics: Condition message describes the specific conflict.
  - Testing: Integration tests cover conflict scenarios.

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check cloud-controller-manager logs for reconciliation errors.
2. Verify the `LoadBalancer` resource conditions for error reasons.
3. Check that the feature gate is enabled on both the API server and
   cloud controller manager.
4. Verify RBAC permissions for the cloud controller manager service
   account.

## Implementation History

- 2026-05-29: Initial KEP draft created.

## Drawbacks

- **API surface expansion.** A new resource type increases the API
  surface. The resource is optional and additive, so existing
  workflows are unaffected, but users and tooling need to learn
  about it.

- **Split configuration.** Load balancer configuration split across
  two resources (Service and LoadBalancer) could confuse users.
  Documentation and tooling (e.g., `kubectl describe service` showing
  the associated `LoadBalancer` resource) can mitigate this.

- **Cloud provider adoption burden.** Cloud providers need to update
  their controllers. The resource is optional, and structured status
  reporting provides a strong incentive for adoption.

## Alternatives

### Add fields directly to Service (KEP-4631 approach)

[KEP-4631] proposed adding conditions and a
`RequiredLoadBalancerFeatures` field directly to the Service resource.
This approach was simpler but had drawbacks:

- It further overloaded the already complex Service resource.
- Reviewer feedback noted that adding per-feature fields "begs why
  we don't do this for EVERYTHING" (@thockin).
- Users had to specify requirements in two places (Service spec and
  RequiredLoadBalancerFeatures), which was considered "blech"
  (@thockin).
- It did not address the annotation sprawl problem for common
  configuration like internal routing.

The standalone `LoadBalancer` resource provides a cleaner separation
of concerns while addressing the same status and feature reporting
needs.

### Use Gateway API for all load balancer configuration

Gateway API's `Gateway` resource already models infrastructure with
conditions and status. However:

- Not all users are on Gateway API. `Service` of type `LoadBalancer`
  remains the primary mechanism for external load balancing.
- Gateway API focuses on proxying semantics. Load balancer
  routing/forwarding infrastructure is a separate concern that
  in-cluster Gateway implementations need but shouldn't have to
  define.
- The `LoadBalancer` resource complements Gateway API by providing
  the routing/forwarding layer that in-cluster implementations can
  delegate to.

### Use CRDs instead of a built-in type

Cloud providers could define their own CRDs for load balancer
configuration. However:

- This leads to the same fragmentation problem as annotations.
- A built-in type provides a standard API that all providers can
  converge on.
- E2e tests need a standard API to test against.

## Infrastructure Needed (Optional)

None.
