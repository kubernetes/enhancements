# KEP-0752 EndpointSlices

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [EndpointSlice API](#endpointslice-api)
  - [Mapping](#mapping)
  - [Endpoint Port](#endpoint-port)
  - [Topology (Per Endpoint)](#topology-per-endpoint)
  - [Converting Topology in EndpointSlice GA](#converting-topology-in-endpointslice-ga)
  - [Zone (Per Endpoint)](#zone-per-endpoint)
  - [EndpointSlice Naming](#endpointslice-naming)
- [Estimation](#estimation)
- [Sample Case 1: 20,000 endpoints, 5,000 nodes](#sample-case-1-20000-endpoints-5000-nodes)
  - [Service Creation/Deletion](#service-creationdeletion)
  - [Single Endpoint Update](#single-endpoint-update)
  - [Rolling Update](#rolling-update)
- [Sample Case 2: 20 endpoints, 10 nodes](#sample-case-2-20-endpoints-10-nodes)
  - [Service Creation/Deletion](#service-creationdeletion-1)
  - [Single Endpoint Update](#single-endpoint-update-1)
  - [Rolling Update](#rolling-update-1)
- [Implementation](#implementation)
  - [Requirements](#requirements)
  - [EndpointSlice Controller](#endpointslice-controller)
  - [Additional EndpointSlice Controllers](#additional-endpointslice-controllers)
    - [Workflows](#workflows)
  - [Kube-Proxy](#kube-proxy)
  - [Endpoints Controller (classic)](#endpoints-controller-classic)
  - [EndpointSliceMirroring Controller](#endpointslicemirroring-controller)
    - [Mirroring Details](#mirroring-details)
    - [Handling Endpoints Events](#handling-endpoints-events)
    - [Controller Start Up](#controller-start-up)
    - [Corresponding Bug Fix for Endpoints and EndpointSlice Controller](#corresponding-bug-fix-for-endpoints-and-endpointslice-controller)
    - [Limitations](#limitations)
    - [Testing Plan](#testing-plan)
      - [Unit Tests](#unit-tests)
      - [E2E Tests](#e2e-tests)
- [Roll Out Plan](#roll-out-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha -&gt; Beta](#alpha---beta)
  - [Beta -&gt; GA](#beta---ga)
  - [Splitting IP address type for better dual stack support](#splitting-ip-address-type-for-better-dual-stack-support)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
- [FAQ](#faq)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP was converted from the [original proposal doc][original-doc]. The
current  [Core/V1 Endpoints API][v1-endpoints-api] comes with severe
performance/scalability drawbacks affecting multiple components in the
control-plane (apiserver, etcd, endpoints-controller, kube-proxy). This doc
proposes a new EndpointSlice API aiming to replace Core/V1 Endpoints API for
most internal consumers, including kube-proxy. The new EndpointSlice API aims to
address existing problems as well as leaving room for future extension.


## Motivation

In the current Endpoints API, one object instance contains all the individual
endpoints of a service. Whenever a single pod in a service is
added/updated/deleted, the whole Endpoints object (even when the other endpoints
didn't change) is re-computed, written to storage (etcd) and sent to all
watchers (e.g. kube-proxy). This leads to 2 major problems:

- Storing multiple megabytes of endpoints puts strain on multiple parts of the
  system due to not having a paging system and a monolithic watch/storage
  design. [The max number of endpoints is bounded by the K8s storage layer
  (etcd)][max-object-size]], which has a hard limit on the size of a single
  object (1.5MB by default). That means attempts to write an object larger than
  the limit will be rejected. Additionally, there is a similar limitation in the
  watch path in Kubernetes apiserver. For a K8s service, if its Endpoints object
  is too large, endpoint updates will not be propagated to kube-proxy(s), and
  thus iptables/ipvs won’t be reprogrammed.
- [Performance degradation in large k8s deployments.][perf-degrade] Not being
  able to efficiently read/update individual endpoint changes can lead to (e.g
  during rolling upgrade of a service) endpoints operations that are quadratic
  in the number of its elements. If one consider watches in the picture (there's
  one from each kube-proxy), the situation becomes even worse as the quadratic
  traffic gets multiplied further with number of watches (usually equal to
  #nodes in the cluster).

The new EndpointSlice API aims to address existing problems as well as leaving
room for future extension.


### Goals

- Support tens of thousands of backend endpoints in a single service on cluster
  with thousands of nodes.
- Move the API towards a general-purpose backend discovery API.
- Leave room for foreseeable extension:
  - Support multiple IPs per pod
  - More endpoint states than Ready/NotReady
  - Dynamic endpoint subsetting

### Non-Goals
- Change functionality provided by K8s V1 Service API.
- Provide better load balancing for K8s service backends.

## Proposal

### EndpointSlice API
The following new EndpointSlice API will be added to the `Discovery` API group.

```golang
type EndpointSlice struct {
    metav1.TypeMeta `json:",inline"`
    // OwnerReferences should be set when the object is derived from a k8s
    // object.
    // The object labels may include the following keys:
    // * kubernetes.io/service-name: the label value indicates the name of the
    //   service from which the EndpointSlice is derived. EndpointSlices which
    //   are not associated with a Service should not use this key.
    // * endpointslice.kubernetes.io/managed-by: the label value represents a
    //   unique name for the controller or application that manages this
    //   EndpointSlice.
    // +optional
    metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
    // addressType specifies the type of address carried by this EndpointSlice.
    // All addresses in this slice must be the same type.
    // Default is IP
    // +optional
    AddressType *AddressType `json:"addressType" protobuf:"bytes,4,rep,name=addressType"`
    // endpoints is a list of unique endpoints in this slice. Each slice may
    // include a maximum of 1000 endpoints.
    // +listType=atomic
    Endpoints []Endpoint `json:"endpoints" protobuf:"bytes,2,rep,name=endpoints"`
    // ports specifies the list of network ports exposed by each endpoint in
    // this slice. Each port must have a unique name. When ports is empty, it
    // indicates that there are no defined ports. When a port is defined with a
    // nil port value, it indicates "all ports". Each slice may include a
    // maximum of 100 ports.
    // +optional
    // +listType=atomic
    Ports []EndpointPort `json:"ports" protobuf:"bytes,3,rep,name=ports"`
}

// AddressType represents the type of address referred to by an endpoint.
type AddressType string

const (
    // AddressTypeIP represents an IP Address.
    // This address type has been deprecated and has been replaced by the IPv4
    // and IPv6 adddress types. New resources with this address type will be
    // considered invalid. This will be fully removed in 1.18.
    // +deprecated
    AddressTypeIP = AddressType("IP")
    // AddressTypeIPv4 represents an IPv4 Address.
    AddressTypeIPv4 = AddressType(corev1.IPv4Protocol)
    // AddressTypeIPv6 represents an IPv6 Address.
    AddressTypeIPv6 = AddressType(corev1.IPv6Protocol)
    // AddressTypeFQDN represents a FQDN.
    AddressTypeFQDN = AddressType("FQDN")
)

// Endpoint represents a single logical "backend" implementing a service.
type Endpoint struct {
    // addresses of this endpoint. The contents of this field are interpreted
    // according to the corresponding EndpointSlice addressType field. This
    // allows for cases like dual-stack (IPv4 and IPv6) networking. Consumers
    // (e.g. kube-proxy) must handle different types of addresses in the context
    // of their own capabilities. This must contain at least one address but no
    // more than 100.
    // +listType=set
    Addresses []string `json:"addresses" protobuf:"bytes,1,rep,name=addresses"`
    // conditions contains information about the current status of the endpoint.
    Conditions EndpointConditions `json:"conditions,omitempty" protobuf:"bytes,2,opt,name=conditions"`
    // hostname of this endpoint. This field may be used by consumers of
    // endpoints to distinguish endpoints from each other (e.g. in DNS names).
    // Multiple endpoints which use the same hostname should be considered
    // fungible (e.g. multiple A values in DNS). Must pass DNS Label (RFC 1123)
    // validation.
    // +optional
    Hostname *string `json:"hostname,omitempty" protobuf:"bytes,3,opt,name=hostname"`
    // targetRef is a reference to a Kubernetes object that represents this
    // endpoint.
    // +optional
    TargetRef *v1.ObjectReference `json:"targetRef,omitempty" protobuf:"bytes,4,opt,name=targetRef"`

    // deprecatedTopology contains topology information part of the v1beta1
    // API. This field is deprecated, and will be removed when the v1beta1
    // API is removed (no sooner than kubernetes v1.24).  While this field can
    // hold values, it is not writable through the v1 API, and any attempts to
    // write to it will be silently ignored. Topology information can be found
    // in the zone and nodeName fields instead.
    // +optional
    DeprecatedTopology map[string]string `json:"deprecatedTopology,omitempty" protobuf:"bytes,5,opt,name=deprecatedTopology"`

    // nodeName represents the name of the Node hosting this endpoint. This can
    // be used to determine endpoints local to a Node. This field can be enabled
    // with the EndpointSliceNodeName feature gate.
    // +optional
    NodeName *string `json:"nodeName,omitempty" protobuf:"bytes,6,opt,name=nodeName"`
    // zone is the name of the Zone this endpoint exists in.
    // +optional
    Zone *string `json:"zone,omitempty" protobuf:"bytes,7,opt,name=zone"`
}

// EndpointConditions represents the current condition of an endpoint.
type EndpointConditions struct {
    // ready indicates that this endpoint is prepared to receive traffic,
    // according to whatever system is managing the endpoint. A nil value
    // indicates an unknown state. In most cases consumers should interpret this
    // unknown state as ready.
    // +optional
    Ready *bool `json:"ready,omitempty" protobuf:"bytes,1,name=ready"`
}

// EndpointPort represents a Port used by an EndpointSlice
type EndpointPort struct {
    // The name of this port. All ports in an EndpointSlice must have a unique
    // name. If the EndpointSlice is dervied from a Kubernetes service, this
    // corresponds to the Service.ports[].name.
    // Name must either be an empty string or pass IANA_SVC_NAME validation:
    // * must be no more than 15 characters long
    // * may contain only [-a-z0-9]
    // * must contain at least one letter [a-z]
    // * it must not start or end with a hyphen, nor contain adjacent hyphens
    // Default is empty string.
    Name *string `json:"name,omitempty" protobuf:"bytes,1,name=name"`
    // The IP protocol for this port.
    // Must be UDP, TCP, or SCTP.
    // Default is TCP.
    Protocol *v1.Protocol `json:"protocol,omitempty" protobuf:"bytes,2,name=protocol"`
    // The application protocol for this port.
    // +optional
    AppProtocol *string `json:"appProtocol,omitempty" protobuf:"bytes,3,name=appProtocol"`
    // The port number of the endpoint.
    // If this is not specified, ports are not restricted and must be
    // interpreted in the context of the specific consumer.
    Port *int32 `json:"port,omitempty" protobuf:"bytes,4,opt,name=port"`
}
```

### Mapping
- 1 Service maps to N EndpointSlice objects.
- Each EndpointSlice contains at most 100 endpoints by default
  (MaxEndpointsPerSlice: configurable via controller manager flag).
- If a EndpointSlice is derived from K8s:
  - The following label is added to identify corresponding service:
    - Key: kubernetes.io/service
    - Value: ${service name}
  - For EndpointSlice instances that are not derived from kubernetes Services,
    the above label must not be applied.
  - The OwnerReferences of the EndpointSlice instances will be set to the
    corresponding service.
- For backend pods with non-uniform named ports (e.g. a service port targets a
  named port. Backend pods have different port number with the same port name),
  this would amplify the number of EndpointSlice object depending on the number
  of backend groups with same ports.
- EndpointSlice will be covered by resource quota. This is to limit the max
  number of EndpointSlice objects in one namespace. This would provide
  protection for k8s apiserver. For instance, a malicious user would not be able
  to DOS k8s API by creating services selecting all pods.

### Endpoint Port
Ports apply to all endpoints contained in the EndpointSlice. The endpoint port
number becomes optional in the EndpointSlice API while the port number field in
core/v1 Endpoints API is required. This allows the API to support services with
no port remapping or all port services.

### Topology (Per Endpoint)
**DEPRECATED** in v1.20 and will not be part of the will not be part of the
discovery/v1 API in v1.21. This field will remain in v1beta1 and a conversion
functionality will exist to convert to and from v1.

A new topology field (string to string map) is added to each endpoint. It can
contain arbitrary topology information associated with the endpoint. If the
EndpointSlice instance is derived from K8s service, the topology may contain
following well known key:
- **kubernetes.io/hostname**: the value indicates the hostname of the node where
  the endpoint is located. This should match the corresponding node label.
- **topology.kubernetes.io/zone**: the value indicates the zone where the
  endpoint is located. This should match the corresponding node label.
- **topology.kubernetes.io/region**: the value indicates the region where the
  endpoint is located. This should match the corresponding node label.

If the Kubernetes Service has topological keys specified, the corresponding node
labels will be copied to endpoint topology.

### Converting Topology in EndpointSlice GA

For beta resources that have set the topology field, the string to string map
will be converted into the `deprecatedTopology` field the v1 resource. This
field will be read-only in the v1 API.

The only key in the topology field that will not be preserved is
**topology.kubernetes.io/zone** and instead its value will be converted
into the [zone field](#zone-per-endpoint) on the Endpoint.

### Zone (Per Endpoint)
A new `zone` field (string) is added to each endpoint that will indicate the
zone where the endpoint is located. For resources before `v1` where the
topology field is populated, if **topology.kubernetes.io/zone** exists, the
value of that key will be converted to this new `zone` field. When converting
back to a beta resource, the value will be added back to the endpoint's map.

### EndpointSlice Naming
Use `generateName` with the Service name as prefix:
```
${service name}-${random}
```

## Estimation
This section provides comparisons between Endpoints API and EndpointSlice API
under 3 scenarios:
- Service Creation/Deletion
- Single Endpoint Update
- Rolling Update

```
Number of Backend Pod: P
Number of Node: N
Number of Endpoint Per EndpointSlice:B
```

## Sample Case 1: 20,000 endpoints, 5,000 nodes

### Service Creation/Deletion


|                          | Endpoints             | 100 Endpoints per EndpointSlice | 1 Endpoint per EndpointSlice |
|--------------------------|-----------------------|---------------------------------|------------------------------|
| # of writes              | O(1)                  | O(P/B)                          | O(P)                         |
|                          | 1                     | 200                             | 20000                        |
| Size of API object       | O(P)                  | O(B)                            | O(1)                         |
|                          | 20k * const = ~2.0 MB | 100 * const = ~10 KB            | < ~1KB                       |
| # of watchers per object | O(N)                  | O(N)                            | O(N)                         |
|                          | 5000                  | 5000                            | 5000                         |
| # of total watch event   | O(N)                  | O(NP/B)                         | O(NP)                        |
|                          | 5000                  | 5000 * 200 = 1,000,000          | 5000 * 20000 = 100,000,000   |
| Total Bytes Transmitted  | O(PN)                 | O(PN)                           | O(PN)                        |
|                          | 2.0MB * 5000 = 10GB   | 10KB * 5000 * 200 = 10GB        | ~10GB                        |

### Single Endpoint Update

|                          | Endpoints             | 100 Endpoints per EndpointSlice | 1 Endpoint per EndpointSlice |
|--------------------------|-----------------------|---------------------------------|------------------------------|
| # of writes              | O(1)                  | O(1)                            | O(1)                         |
|                          | 1                     | 1                               | 1                            |
| Size of API object       | O(P)                  | O(B)                            | O(1)                         |
|                          | 20k * const = ~2.0 MB | 100 * const = ~10 KB            | < ~1KB                       |
| # of watchers per object | O(N)                  | O(N)                            | O(N)                         |
|                          | 5000                  | 5000                            | 5000                         |
| # of total watch event   | O(N)                  | O(N)                            | O(N)                         |
|                          | 5000                  | 5000                            | 5000                         |
| Total Bytes Transmitted  | O(PN)                 | O(BN)                           | O(N)                         |
|                          | ~2.0MB * 5000 = 10GB  | ~10k * 5000 = 50MB              | ~1KB * 5000 = ~5MB           |


### Rolling Update

|                          | Endpoints                   | 100 Endpoints per EndpointSlice | 1 Endpoint per EndpointSlice |
|--------------------------|-----------------------------|---------------------------------|------------------------------|
| # of writes              | O(P)                        | O(P)                            | O(P)                         |
|                          | 20k                         | 20k                             | 20k                          |
| Size of API object       | O(P)                        | O(B)                            | O(1)                         |
|                          | 20k * const = ~2.0 MB       | 100 * const = ~10 KB            | < ~1KB                       |
| # of watchers per object | O(N)                        | O(N)                            | O(N)                         |
|                          | 5000                        | 5000                            | 5000                         |
| # of total watch event   | O(NP)                       | O(NP)                           | O(NP)                        |
|                          | 5000 * 20k                  | 5000 * 20k                      | 5000 * 20k                   |
| Total Bytes Transmitted  | O(P^2N)                     | O(NPB)                          | O(NP)                        |
|                          | 2.0MB * 5000 * 20k = 200 TB | 10KB * 5000 * 20k = 1 TB        | ~1KB * 5000 * 20k = ~100 GB  |


## Sample Case 2: 20 endpoints, 10 nodes

### Service Creation/Deletion

|                          | Endpoints             | 100 Endpoints per EndpointSlice | 1 Endpoint per EndpointSlice |
|--------------------------|-----------------------|---------------------------------|------------------------------|
| # of writes              | O(1)                  | O(P/B)                          | O(P)                         |
|                          | 1                     | 1                               | 20                           |
| Size of API object       | O(P)                  | O(B)                            | O(1)                         |
|                          | ~1KB                  | ~1KB                            | ~1KB                         |
| # of watchers per object | O(N)                  | O(N)                            | O(N)                         |
|                          | 10                    | 10                              | 10                           |
| # of total watch event   | O(N)                  | O(NP/B)                         | O(NP)                        |
|                          | 1 * 10 = 10           | 1 * 10 = 10                     | 10 * 20 = 200                |
| Total Bytes Transmitted  | O(PN)                 | O(PN)                           | O(PN)                        |
|                          | ~1KB * 10 = 10KB      | ~1KB * 10 = 10KB                | ~1KB * 200 = 200KB           |

### Single Endpoint Update

|                          | Endpoints             | 100 Endpoints per EndpointSlice | 1 Endpoint per EndpointSlice |
|--------------------------|-----------------------|---------------------------------|------------------------------|
| # of writes              | O(1)                  | O(1)                            | O(1)                         |
|                          | 1                     | 1                               | 1                            |
| Size of API object       | O(P)                  | O(B)                            | O(1)                         |
|                          | ~1KB                  | ~1KB                            | ~1KB                         |
| # of watchers per object | O(N)                  | O(N)                            | O(N)                         |
|                          | 10                    | 10                              | 10                           |
| # of total watch event   | O(N)                  | O(N)                            | O(N)                         |
|                          | 1                     | 1                               | 1                            |
| Total Bytes Transmitted  | O(PN)                 | O(BN)                           | O(N)                         |
|                          | ~1KB * 10 = 10KB      | ~1KB * 10 = 10KB                | ~1KB * 10 = 10KB             |


### Rolling Update

|                          | Endpoints                   | 100 Endpoints per EndpointSlice | 1 Endpoint per EndpointSlice |
|--------------------------|-----------------------------|---------------------------------|------------------------------|
| # of writes              | O(P)                        | O(P)                            | O(P)                         |
|                          | 20                          | 20                              | 20                           |
| Size of API object       | O(P)                        | O(B)                            | O(1)                         |
|                          | ~1KB                        | ~1KB                            | ~1KB                         |
| # of watchers per object | O(N)                        | O(N)                            | O(N)                         |
|                          | 10                          | 10                              | 10                           |
| # of total watch event   | O(NP)                       | O(NP)                           | O(NP)                        |
|                          | 10 * 20                     | 10 * 20                         | 10 * 20                      |
| Total Bytes Transmitted  | O(P^2N)                     | O(NPB)                          | O(NP)                        |
|                          | ~1KB * 10 * 20 = 200KB      | ~1KB * 10 * 20 = 200KB          | ~1KB * 10 * 20 = 200KB       |


## Implementation

### Requirements

- **Persistence (Minimal Churn of Endpoints)**

Upon service endpoint changes, the # of object writes and disruption to ongoing connections should be minimal.

- **Handling Restarts & Failures**

The producer/consumer of EndpointSlice must be able to handle restarts and recreate state from scratch with minimal change to existing state.


### EndpointSlice Controller

A new EndpointSlice Controller will be added to `kube-controller-manager`. It will manage the lifecycle EndpointSlice instances derived from services.
```
Watch: Service, Pod, Node ==> Manage: EndpointSlice
```

### Additional EndpointSlice Controllers

Since EndpointSlices were meant to be highly extensible, it's important to
ensure that they can be managed by other controllers without being deleted or
modified by the primary EndpointSlice Controller. To achieve that, we propose
adding a `endpointslice.kubernetes.io/managed-by` label. The EndpointSlice
controller will set a value of `endpointslice-controller` on each EndpointSlice
it manages. It will not modify any EndpointSlices without that label value.

In the alpha release of EndpointSlices in 1.16, this label did not exist and all
EndpointSlices associated with a Service that had a selector specified were
managed by the EndpointSlice Controller. Upgrading clusters that had the
`EndpointSlice` feature gate enabled in 1.16 will require cleaning up any older
EndpointSlices without this label.

#### Workflows
On Service Create/Update/Delete:
- `syncService(svc)`

On Pod Create/Update/Delete:
- Reverse lookup relevant services
- For each relevant service,
  - `syncService(svc)`

`syncService(svc)`:
- Look up selected backend pods
- Look up existing EndpointSlices for the service `svc`.
- Calculate difference between wanted state and current state.
- Perform reconciliation with minimized changes.

### Kube-Proxy

Kube-Proxy will support consuming EndpointSlice resources as an alternative to
Endpoints resources. This will be enabled with the `EndpointSliceProxying`
feature gate.

This will require minimal changes to proxier implementations, instead updating
the shared code that transforms Endpoints into a generic data structure to also
transform EndpointSlices into that same data structure.

### Endpoints Controller (classic)

The existing Endpoints controller will continue to run. After EndpointSlices
become GA, the Endpoints controller may limit the number of endpoints it stores
per resource.

* Kubernetes 1.21: If the number of endpoints in one Endpoints object exceeds
  1000, set `endpoints.kubernetes.io/over-capacity` label to "warning".
* Kubernetes 1.22: Limit the number of endpoints in one Endpoints object to 1000
  and set the `endpoints.kubernetes.io/over-capacity` label to "truncated" when
  truncation occurs.

### EndpointSliceMirroring Controller

In some cases, custom Endpoints resources are created by applications. To ensure
that these applications will not need to concurrently write to both Endpoints
and EndpointSlice resources, a new EndpointSliceMirroring controller will be
used to mirror custom Endpoints resources to corresponding EndpointSlices.

* A new `endpointslice.kubernetes.io/skip-mirror` label will be introduced.
  When this label is set to "true" on an Endpoints resource, the
  EndpointSliceMirroring controller will not mirror this resource to an
  EndpointSlice.
* The Endpoints controller will set this new label to "true" for all Endpoints
  resources it manages.
* The EndpointSliceMirroring controller will watch for Endpoints without that
  label set and create equivalent EndpointSlice resources.

#### Mirroring Details
The EndpointSliceMirroring controller will mirror events on matching Endpoints
to corresponding EndpointSlices. Individual Endpoints resources may translate
into multiple EndpointSlices. This will occur if an Endpoints resource has
multiple subsets or includes endpoints with multiple IP families (IPv4 and
IPv6). Each mirrored EndpointSlice resource will:
* Be tied to the corresponding Endpoints resource with an OwnerReference for
  automatic garbage collection.
* Have the `kubernetes.io/service-name` label set to the name of the Endpoints
  resource for easy lookup.
* Have the `endpointslice.kubernetes.io/managed-by` label set to
  `endpointslicemirroring-controller.k8s.io` to ensure that the EndpointSlice
  controller does not modify these EndpointSlices.
* Mirror all labels from the Endpoints resource and all endpoints and ports
  from the corresponding subset.

#### Handling Endpoints Events
The EndpointSliceMirroring controller will have similar logic to the existing
EndpointSlice controller. There may be opportunities to share some code between
the implementations.
* **Create**: The controller will create equivalent EndpointSlices, grouped by
  unique port and IP family combinations.
* **Update**: The controller will compare the existing EndpointSlices mirrored
  for this Endpoints resource with the desired set. Similar to the EndpointSlice
  controller, it will group endpoints by unique port and IP family combinations
  and attempt to minimize changes to EndpointSlices by recycling resources
  wherever possible.
* **Delete**: The controller will ensure that associated EndpointSlices are
  removed. Although the OwnerReference should ensure that associated
  EndpointSlices are automatically removed if the Endpoints resource is deleted,
  manual cleanup will be required if only the skip label is changed.

The controller will use a label filtered watch, so changing the skip label will
naturally appear as create or delete events, even if the Endpoints resource is
only modified.

#### Controller Start Up
The EndpointSliceMirroring controller will not be started until the Endpoints
controller indicates that it has completed the first full sync. This will ensure
that it does not mirror Endpoints that are already managed by the Endpoints
controller (and therefore would also have EndpointSlices managed by the
EndpointSlice controller).

When the controller starts up, it will be responsible for cleaning up any excess
EndpointSlices that it manages that are no longer desired. This will be
determined by listing EndpointSlices the controller is managing with the
`endpointslice.kubernetes.io/managed-by` label and comparing those with the
EndpointSlices desired for any Endpoints with the
`endpointslice.kubernetes.io/skip-mirror` label set to "true".

#### Corresponding Bug Fix for Endpoints and EndpointSlice Controller
The Endpoints and EndpointSlice controller currently do not delete Endpoints or
EndpointSlices when a selector is removed from a Service. This bug will be fixed
before the EndpointsMirroring controller is added.

#### Limitations
To simplify implementation and align with the planned limitation on the size of
Endpoints, the EndpointSliceMirroring controller will limit the number of
endpoints mirrored to EndpointSlice resources to 1000 per Endpoints resource.

#### Testing Plan
The following will need to be covered as part of the testing plan:

##### Unit Tests
* Stale EndpointSlices managed by EndpointSliceMirroring controller are cleaned
  up by the controller when it starts up.
* Only Endpoints with the skip label set to "true" will be mirrored by the
  EndpointSliceMirroring controller.
* Endpoints transitioning between values of the skip label should result in
  corresponding EndpointSlices being created or deleted.
* Mirrored EndpointSlices will:
  * Include the appropriate labels, endpoints, and ports of the corresponding
    Endpoints resource.
  * Refer to the original Endpoints resource with an OwnerReference and a
    `kubernetes.io/service-name` label.
  * Include a `endpointslice.kubernetes.io/managed-by` label set to
    `endpointslicemirroring-controller.k8s.io`.
* Endpoints with dual stack Endpoints should be represented with multiple
  EndpointSlices, at least 1 per IP family.
* Endpoints with multiple subsets should be represented with multiple
  EndpointSlices, at least 1 per subset.
* Endpoints with multiple subsets and IP families should be represented with
  multiple EndpointSlices, at least 1 per subset and IP family.
* Endpoints with more than 1000 endpoints should result in exactly 1000
  endpoints being mirrored to corresponding EndpointSlices.
* EndpointSlices with a different `endpointslice.kubernetes.io/managed-by` will
  not be modified by the EndpointSlice mirroring controller.
* When a selector is added to a Service with a preexisting Endpoints resource:
  * The Endpoints controller will modify the existing Endpoints resource to add
    the `endpointslice.kubernetes.io/skip-mirror` label.
  * The EndpointSliceMirroring controller will delete any EndpointSlices it has
    mirrored for that Endpoints resource.

##### E2E Tests
* Custom Endpoints are mirrored through create, update, and delete process.
* Endpoints created by the Endpoints controller include the skip label.
* EndpointSlices are created with the appropriate labels, endpoints, ports, and
  Service references (label and owner ref).

## Roll Out Plan

**Kubernetes 1.16: Alpha Release**
* Initial EndpointSlice alpha release. No EndpointSlice functionality enabled by
  default.

**Kubernetes 1.17: Beta API**
* EndpointSlice API graduates to beta, no EndpointSlice functionality is enabled
  by default.
* API additions include `endpointslice.kubernetes.io/managed-by` label to enable
  EndpointSlices managed by multiple controllers.

**Kubernetes 1.18: Controller Enabled by Default**
* EndpointSlice controller is now enabled by default.
* EndpointSlice functionality for kube-proxy is now guarded by a new alpha
  `EndpointSliceProxying` feature gate.

**Kubernetes 1.19: Custom Endpoints Mirrroring, Kube-Proxy on Linux uses EndpointSlices by Default**
* Endpoints not managed by the Endpoints controller will be automatically
  mirrored with new EndpointsMirroring controller.
* `EndpointSliceProxying` feature gate will graduate to beta on Linux:
  * Kube-Proxy on Linux will use EndpointSlices by default.
  * Kube-Proxy on Windows will support EndpointSlices in an alpha state.

**Kubernetes 1.20: Add NodeName, Deprecate Topology**
* Given the updated plans for topology aware routing, the topology field is no
  longer useful, it will be deprecated.
* A new NodeName field will be added, guarded by the EndpointSliceNodeName
  feature gate.

**Kubernetes 1.21: GA API, Kube-Proxy on Windows uses EndpointSlices by
Default**
* The EndpointSlice API will graduate to v1.
* The topology field will be removed.
* A new Zone field will be added to provide per endpoint topology information.
* `EndpointSliceNodeName` feature gate will graduate to beta.
* `WindowsEndpointSliceProxying` feature gate will graduate to beta on Windows:
  * Kube-Proxy on Windows will use EndpointSlices by default.
* A new `endpoints.kubernetes.io/over-capacity` label will be set to "warning"
  on Endpoints resources exceeding 1000 endpoints.

**Kubernetes 1.22: Kube-Proxy GA**
* The `EndpointSliceProxying` feature gate guarding EndpointSlice integration
  with kube-proxy will graduate to GA on both Linux and Windows.
* Endpoints resources will be limited to 1000 endpoints. The
  `endpoints.kubernetes.io/over-capacity` label will continue to be set to
  "truncated" in these cases.

## Graduation Criteria

### Alpha -> Beta

- Kube-proxy switch to consume EndpointSlice API. (Already done in Alpha)
- Verify performance/scalability via testing. (Scale tested to 50k endpoints in
  4k node cluster)
- Get performance fixes identified in scale testing merged.
- Split `IP` address type into new `IPv4` and `IPv6` address types.
- Implement dual-stack EndpointSlice support in kube-proxy.
- Implement e2e tests that ensure both Endpoints and EndpointSlices are tested.
- Add support for `endpointslice.kubernetes.io/managed-by` label.
- Add FQDN addressType.
- Add support for optional appProtocol field on `EndpointPort`.

### Beta -> GA

- EndpointSlice API has been stable at beta for at least 2 minor releases.
- EndpointSlice Controller has been enabled by default for 2 minor releases.
- EndpointSlices has been used by default for at least 1 minor release.
- Implementation has been scaled tested beyond 100k endpoints.

### Splitting IP address type for better dual stack support

Although the initial vision for the `IP` address type was to be inclusive of
both IPv4 and IPv6 addresses, that ended up complicating workflows in consumers
like kube-proxy. In that case, and we anticipate many more, the consumer is only
interested in a specific IP family for an endpoint. Both Endpoints and Services
have moved toward using different resources per IP family. It only makes sense
to mirror that behavior with EndpointSlices.

With that in mind, the proposed changes for beta will involve the following:

1. Add 2 additional address types: `IPv4` and `IPv6`.
2. Update the EndpointSlice controller to only create EndpointSlices with these
address types.
3. Deprecate the `IP` address type, making it invalid for new EndpointSlices in
1.17 before becoming fully invalid in 1.18.

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - *EndpointSlice*: Enables this feature for kube-apiserver (self referential
      EndpointSlice management) and kube-controller-manager (EndpointSlice and
      EndpointSlice mirroring controllers).
    - *EndpointSliceProxying*: Enables this feature for kube-proxy - it will use
      EndpointSlices as the source of endpoint data instead of Endpoints.

* **Does enabling the feature change any default behavior?**
  No. There is potential for noticeable bugs, most likely if there are any bugs
  with the EndpointSlice implementation of kube-proxy. These bugs could result
  in proxy rules being programmed with inaccurate or stale data.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes.

* **What happens if we reenable the feature if it was previously rolled back?**
  Reenabling the `EndpointSlice` feature gate could result in the controller
  cleaning up or updating stale EndpointSlices. Reenabling the
  `EndpointSliceProxying` feature gate should not result in any changes to API
  resources though it may result in reprogramming iptables/ipvs rules. This
  would be especially noticeable if EndpointSlices and Endpoints were out of
  sync.

* **Are there any tests for feature enablement/disablement?**
  There are unit tests covering kube-proxy with and without the EndpointSlice
  functionality enabled.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  The biggest risk comes from enabling EndpointSlice functionality as part of an
  upgrade. The controller needs to be enabled before kube-proxy to ensure it has
  sufficient time to create EndpointSlices for all endpoints in the cluster. If
  kube-proxy is started before then and configured to read from EndpointSlices,
  only the endpoints that have been represented with EndpointSlices at that
  point in time will be accessible from that Node.

* **What specific metrics should inform a rollback?**
  It's unlikely that the EndpointSlice controller or API would need to be rolled
  back as they are additive features. A rollback might be necessary if enabling
  the EndpointSlice feature gate on kube-controller-manager or kube-apiserver
  significantly increased the CPU or memory utilization.

  For kube-proxy, a rollback may be necessary if a significant increase in
  network_programming_duration_seconds is observed. Alternatively, if
  sync_proxy_rules_endpoint_changes_pending is consistently higher than usual,
  it may indicate that kube-proxy has fallen behind and a rollback should be
  considered.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  This has been manually tested on a variety of e2e clusters by manually
  enabling or disabling the feature gate and/or API for api-server,
  controller-manager, and kube-proxy. These tests showed that transitioning
  between config states on these components works well for this feature.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  No.

### Monitoring requirements

* **How can an operator determine if the feature is in use by workloads?**
  There are a variety of metrics that indicate the quantity and efficiency of
  EndpointSlices in the cluster. These include:
  - endpoint_slice_controller/endpoints_added_per_sync
  - endpoint_slice_controller/endpoints_removed_per_sync
  - endpoint_slice_controller/endpoints_desired
  - endpoint_slice_controller/num_endpoint_slices
  - endpoint_slice_controller/desired_endpoint_slices
  - endpoint_slice_controller/changes

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [x] Metrics
    - Metric name: [network_programming_duration_seconds][]
    - Components exposing the metric: This requires cooperation between the
      EndpointSlice controller and kube-proxy. The metric itself is exposed by
      kube-proxy. This indicates how long it takes for proxy rules to be updated
      after a Kubernetes resource changes.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  <!-- TODO(robscott): Revisit this suggestion -->
  The network_programming_duration_seconds metric should be under 60 at least
  95% of the time.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  Since this is a networking feature, it would be helpful to have more metrics
  covering things like error rates. Unfortunately request errors can be caused
  by a wide variety of things so it would likely be a very noisy metric.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  Enabling the EndpointSliceProxying feature gate on kube-proxy is dependent
  on the EndpointSlice feature gate being enabled on kube-controller-manager
  and kube-apiserver. This feature does not depend on any additional services.

### Scalability


* **Will enabling / using this feature result in any new API calls?**
  Enabling the EndpointSlice controller will result in significant numbers of
  write (create, update, delete) requests to the EndpointSlice API. The
  throughput will depend on the number of EndpointSlices required and the
  frequency with which they'll be updated.

  Each EndpointSlice can store up to 100 endpoints by default. If most Services
  in a cluster have less than 100 endpoints, the load from the EndpointSlice
  controller will be similar to the Endpoints controller. If many Services have
  more than 100 endpoints, more EndpointSlices will be required and therefore
  additional write requests will be sent to the API.

  The frequency that EndpointSlices are updated depends on how frequently
  Services or their endpoints (typically Pods) are updated. Frequent Pod
  replacement will result in frequent EndpointSlice updates - similar to the
  behavior of the Endpoints controller.

  Enabling the EndpointSliceProxying feature gate for kube-proxy will result in
  changes to read API calls. Instead of watching Endpoints resources, kube-proxy
  will watch EndpointSlices. This should result in significant decreases in
  bandwidth for large clusters with large Services.

* **Will enabling / using this feature result in introducing new API types?**
  - API type: EndpointSlice
  - Supported number of objects per cluster: ~10,000
  - Supported number of objects per namespace: ~500

  EndpointSlices have been scale tested up to ~10k total slices in a cluster and
  ~500 total slices in a Namespace. In practice, these numbers can likely be
  safely exceeded. As a general rule, larger Services place more load on the
  controller than more smaller Services with the same total number of endpoints.
  More testing should be done to determine more precise numbers here.

* **Will enabling / using this feature result in any new calls to cloud
  provider?**
  No.

* **Will enabling / using this feature result in increasing size or count
  of the existing API objects?**
  No.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
  No.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Enabling the EndpointSlice controller will result in slight increases of CPU
  and RAM for kube-controller-manager. This will increase with the number of
  EndpointSlices and frequency of updates. Enabling the EndpointSlice
  functionality in kube-proxy will result in slight decreases to CPU and RAM
  utilization, becoming increasingly significant at scale.

### Troubleshooting
* **How does this feature react if the API server and/or etcd is unavailable?**
  This would inherit the behavior of kube-controller-manager and kube-proxy.
  There is nothing in this feature that would make these components more or less
  resilient.

* **What are other known failure modes?**
  - EndpointSlice controller is not creating EndpointSlices.
    - Detection: If endpoint_slice_controller/num_endpoint_slices is 0 or
      completely missing.
    - Diagnostics: The EndpointSlice controller will have helpful log messages
      in the kube-controller manager logs.
    - Testing: This would be covered by any kube-proxy tests covering an empty
      state.

* **What steps should be taken if SLOs are not being met to determine the problem?**
  Review the EndpointSlice controller and kube-proxy logs for any error
  messages.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
[network_programming_duration_seconds]: https://github.com/kubernetes/community/blob/master/sig-scalability/slos/network_programming_latency.md

## Implementation History

- Alpha: 1.16
- Beta (disabled by default): 1.17
- Beta (controller enabled by default): 1.18

## Alternatives

1. increase the etcd size limits
2. endpoints controller batches / rate limits changes
3. apiserver batches / rate-limits watch notifications
4. apimachinery to support object level pagination

## FAQ

- #### Why not pursue the alternatives?

In order to fulfill the goal of this proposal, without redesigning the Core/V1
Endpoints API, all items listed in the alternatives section are required. Item
#1 increase maximum endpoints limitation by increasing the object size limit.
This may bring other performance/scalability implications. Item #2 and #3 can
reduce transmission overhead but sacrificed endpoint update latency. Item #4 can
further reduce transmission overhead, however it is a big change to the existing
API machinery.

In summary, each of the items can only achieve incremental gain to some extent.
Compared to this proposal, the combined effort would be equal or more while
achieving less performance improvements.

In addition, the EndpointSlice API is capable to express endpoint subsetting,
which is the natural next step for improving k8s service endpoint scalability.


- #### Why only include up to 100 endpoints in one EndpointSlice object? Why not 1 endpoint? Why not 1000 endpoints?

Based on the data collected from user clusters, vast majority (> 99%) of the k8s
services have less than 100 endpoints. For small services, EndpointSlice API
will make no difference. If the MaxEndpointsPerSlice is too small (e.g. 1
endpoint per EndpointSlice), controller loses capability to batch updates, hence
causing worse write amplification on service creation/deletion and scale
up/down. Etcd write RPS is significant limiting factor.

- #### Why do we have a condition struct for each endpoint?

The current Endpoints API only includes a boolean state (Ready vs. NotReady) on
individual endpoint. However, according to pod life cycle, there are more states
(e.g. Graceful Termination, ContainerReary). In order to represent additional
states other than Ready/NotReady,  a status structure is included for each
endpoint. More condition types can be added in the future without compatibility
disruptions. As more conditions are added, different consumer (e.g. different
kube-proxy implementations) will have the option to evaluate the additional
conditions.

- #### Why not use a CRD?

**1. Protobuf is more efficient**
Currently CRDs don't support protobuf. In our testing, a protobuf watch is
approximately 5x faster than a JSON watch. We used pprof to profile 2 versions
of kube-proxy using EndpointSlices and running on 2 different nodes in a 150
node cluster as it scaled up to 15k endpoints. Over the 15 minute window,
kube-proxy with JSON used 17% more CPU time, with the difference in
`StreamWatcher.receive` accounting for all of that. With protobuf enabled, that
function took 1/5th the time of the JSON implementation.

**2. Validation is too complex**
Validation of addresses relies on addressType, something that would be
difficult, maybe impossible, to recreate with OpenAPI validations. Additionally,
there are a number of validations currently in use that are able to reuse the
same validations used elsewhere for Services or Endpoints such as
`IsDNS1123Label` and `IsValidIP`. Although these could be recreated with OpenAPI
validations, the error messages would not be as helpful and we would lose the
consistency in messaging from the related resources.

**3. EndpointSlices are required for the API Server to be accessible**
In an interesting race condition, the API Server needs to be available before
much can happen. With both Endpoints and EndpointSlices, that means that it
needs to manage the references to itself by creating these resources on startup.
Since this makes EndpointSlice core enough to be a dependency of API Server, it
would have to exist in every cluster. If EndpointSlices were a CRD, the CRD
would also have to be installed by the API Server, making a CRD a core
dependency of Kubernetes. Additionally, any components like kube-proxy that
depend on EndpointSlices would break if the CRD hadn't been installed before
they started up.


[original-doc]: https://docs.google.com/document/d/1sLJfolOeEVzK5oOviRmtHOHmke8qtteljQPaDUEukxY/edit#
[v1-endpoints-api]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#endpoints-v1-core
[max-object-size]: https://github.com/kubernetes/kubernetes/issues/73324
[perf-degrade]: https://github.com/kubernetes/community/blob/master/sig-scalability/blogs/k8s-services-scalability-issues.md#endpoints-traffic-is-quadratic-in-the-number-of-endpoints
