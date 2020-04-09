---
title: EndpointSlice API 
authors:
  - "@freehan"
owning-sig: sig-network
reviewers:
  - "@bowei"
  - "@thockin"
  - "@wojtek-t"
  - "@johnbelamaric"
approvers:
  - "@bowei"
  - "@thockin"
creation-date: 2019-06-01
last-updated: 2019-06-01
status: implementable
see-also:
  - "https://docs.google.com/document/d/1sLJfolOeEVzK5oOviRmtHOHmke8qtteljQPaDUEukxY/edit#"
---
# EndpointSlice API 

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goal](#goal)
  - [Non-Goal](#non-goal)
- [Proposal](#proposal)
  - [EndpointSlice API](#endpointslice-api-1)
  - [Mapping](#mapping)
  - [EndpointMeta (Per EndpointSlice)](#endpointmeta-per-endpointslice)
  - [Topology (Per Endpoint)](#topology-per-endpoint)
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
  - [Endpoint Controller (classic)](#endpoint-controller-classic)
- [Roll Out Plan](#roll-out-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Splitting IP address type for better dual stack support](#splitting-ip-address-type-for-better-dual-stack-support)
- [Alternatives](#alternatives)
- [FAQ](#faq)
<!-- /toc -->

## Summary 

This KEP was converted from the [original proposal doc][original-doc]. The current  [Core/V1 Endpoints API][v1-endpoints-api] comes with severe performance/scalability drawbacks affecting multiple components in the control-plane (apiserver, etcd, endpoints-controller, kube-proxy). 
This doc proposes a new EndpointSlice API aiming to replace Core/V1 Endpoints API for most internal consumers, including kube-proxy.
The new EndpointSlice API aims to address existing problems as well as leaving room for future extension.


## Motivation

In the current Endpoints API, one object instance contains all the individual endpoints of a service. Whenever a single pod in a service is added/updated/deleted, the whole Endpoints object (even when the other endpoints didn't change) is re-computed, written to storage (etcd) and sent to all watchers (e.g. kube-proxy). This leads to 2 major problems:

- Storing multiple megabytes of endpoints puts strain on multiple parts of the system due to not having a paging system and a monolithic watch/storage design. [The max number of endpoints is bounded by the K8s storage layer (etcd)][max-object-size]], which has a hard limit on the size of a single object (1.5MB by default). That means attempts to write an object larger than the limit will be rejected. Additionally, there is a similar limitation in the watch path in Kubernetes apiserver. For a K8s service, if its Endpoints object is too large, endpoint updates will not be propagated to kube-proxy(s), and thus iptables/ipvs won’t be reprogrammed.
- [Performance degradation in large k8s deployments.][perf-degrade] Not being able to efficiently read/update individual endpoint changes can lead to (e.g during rolling upgrade of a service) endpoints operations that are quadratic in the number of its elements. If one consider watches in the picture (there's one from each kube-proxy), the situation becomes even worse as the quadratic traffic gets multiplied further with number of watches (usually equal to #nodes in the cluster).

The new EndpointSlice API aims to address existing problems as well as leaving room for future extension.


### Goal

- Support tens of thousands of backend endpoints in a single service on cluster with thousands of nodes.
- Move the API towards a general-purpose backend discovery API.
- Leave room for foreseeable extension:
  - Support multiple IPs per pod
  - More endpoint states than Ready/NotReady
  - Dynamic endpoint subsetting
  
### Non-Goal
- Change functionality provided by K8s V1 Service API.
- Provide better load balancing for K8s service backends.

## Proposal

### EndpointSlice API
The following new EndpointSlice API will be added to the `Discovery` API group.

```
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
    // topology contains arbitrary topology information associated with the
    // endpoint. These key/value pairs must conform with the label format.
    // https://kubernetes.io/docs/concepts/overview/working-with-objects/labels
    // Topology may include a maximum of 16 key/value pairs. For endpoints
    // backed by Kubernetes Pods, This may include, but is not limited to the
    // following well known keys:
    // * kubernetes.io/hostname: the value indicates the hostname of the node
    //   where the endpoint is located. This should match the corresponding
    //   node label.
    // * topology.kubernetes.io/zone: the value indicates the zone where the
    //   endpoint is located. This should match the corresponding node label.
    // * topology.kubernetes.io/region: the value indicates the region where the
    //   endpoint is located. This should match the corresponding node label.
    // +optional
    Topology map[string]string `json:"topology,omitempty" protobuf:"bytes,5,opt,name=topology"`
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
- Each EndpointSlice contains at most 100 endpoints by default (MaxEndpointsPerSlice: configurable via controller manager flag).
- If a EndpointSlice is derived from K8s:
  - The following label is added to identify corresponding service:  
    - Key: kubernetes.io/service
    - Value: ${service name}
  - For EndpointSlice instances that are not derived from kubernetes Services, the above label must not be applied.
  - The OwnerReferences of the EndpointSlice instances will be set to the corresponding service.
- For backend pods with non-uniform named ports (e.g. a service port targets a named port. Backend pods have different port number with the same port name), this would amplify the number of EndpointSlice object depending on the number of backend groups with same ports.
- EndpointSlice will be covered by resource quota. This is to limit the max number of EndpointSlice objects in one namespace. This would provide protection for k8s apiserver. For instance, a malicious user would not be able to DOS k8s API by creating services selecting all pods.

### EndpointMeta (Per EndpointSlice)
EndpointMeta contains metadata applying to all endpoints contained in the EndpointSlice.
- **Endpoint Port**: The endpoint port number becomes optional in the EndpointSlice API while the port number field in core/v1 Endpoints API is required. This allows the API to support services with no port remapping or all port services.   

### Topology (Per Endpoint)
A new topology field (string to string map) is added to each endpoint. It can contain arbitrary topology information associated with the endpoint. If the EndpointSlice instance is derived from K8s service, the topology may contain following well known key:
- **kubernetes.io/hostname**: the value indicates the hostname of the node where the endpoint is located. This should match the corresponding node label.
- **topology.kubernetes.io/zone**: the value indicates the zone where the endpoint is located. This should match the corresponding node label.
- **topology.kubernetes.io/region**: the value indicates the region where the endpoint is located. This should match the corresponding node label.

If the k8s service has topological keys specified, the corresponding node labels will be copied to endpoint topology. 

### EndpointSlice Naming
Use `generateName` with service name as prefix:
```
${service name}-${random}
```

## Estimation
This section provides comparisons between Endpoints API and EndpointSlice API under 3 scenarios:
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
managed by the EndpointSlice Controller. To add support for this label in 1.17,
a temporary `endpointslice.kubernetes.io/managed-by-setup` annotation on the
Service will be used to provide a seamless upgrade. In 1.17, the EndpointSlice
controller will claim each EndpointSlice without the corresponding label and
annotation set, setting those values to claim ownership. In 1.18, the annotation
on the Service can safely be removed.

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

Kube-proxy will be modified to consume EndpointSlice instances besides Endpoints resource. A flag will be added to kube-proxy to toggle the mode. 

```
Watch: Service, EndpointSlice ==> Manage: iptables, ipvs, etc
```
- Merge multiple EndpointSlice into an aggregated list.
- Reuse the existing processing logic 

### Endpoint Controller (classic)

In order to ensure backward compatibility for external consumer of the core/v1 Endpoints API, the existing K8s endpoint controller will keep running until the API is EOL. The following limitations will apply:
- Starting from EndpointSlice beta: If # of endpoints in one Endpoints object exceed 1000, generate a warning event to the object. 
- Starting from EndpointSlice GA: Only include up to 1000 endpoints in one Endpoints Object and throw events.

## Roll Out Plan

| K8s Version | State | OSS Controllers                                                            | Internal Consumer (Kube-proxy) |
|-------------|-------|----------------------------------------------------------------------------|--------------------------------|
| 1.16        | Alpha | EndpointSliceController (Alpha) EndpointController (GA with normal event)  | Endpoints                      |
| 1.17        | Beta  | EndpointSliceController (Beta)  EndpointController (GA with warning event) | EndpointSlice                  |
| 1.19+       | GA    | EndpointSliceController (GA)    EndpointController (GA with limitation)    | EndpointSlice                  |


## Graduation Criteria

In order to graduate to beta, we will:

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

## Alternatives

1. increase the etcd size limits
2. endpoints controller batches / rate limits changes
3. apiserver batches / rate-limits watch notifications
4. apimachinery to support object level pagination

## FAQ

- #### Why not pursue the alternatives?

In order to fulfill the goal of this proposal, without redesigning the Core/V1 Endpoints API, all items listed in the alternatives section are required. Item #1 increase maximum endpoints limitation by increasing the object size limit. This may bring other performance/scalability implications. Item #2 and #3 can reduce transmission overhead but sacrificed endpoint update latency. Item #4 can further reduce transmission overhead, however it is a big change to the existing API machinery. 

In summary, each of the items can only achieve incremental gain to some extent. Compared to this proposal, the combined effort would be equal or more while achieving less performance improvements. 

In addition, the EndpointSlice API is capable to express endpoint subsetting, which is the natural next step for improving k8s service endpoint scalability.      

- #### Why only include up to 100 endpoints in one EndpointSlice object? Why not 1 endpoint? Why not 1000 endpoints?

Based on the data collected from user clusters, vast majority (> 99%) of the k8s services have less than 100 endpoints. For small services, EndpointSlice API will make no difference. If the MaxEndpointsPerSlice is too small (e.g. 1 endpoint per EndpointSlice), controller loses capability to batch updates, hence causing worse write amplification on service creation/deletion and scale up/down. Etcd write RPS is significant limiting factor.

- #### Why do we have a condition struct for each endpoint? 

The current Endpoints API only includes a boolean state (Ready vs. NotReady) on individual endpoint. However, according to pod life cycle, there are more states (e.g. Graceful Termination, ContainerReary). In order to represent additional states other than Ready/NotReady,  a status structure is included for each endpoint. More condition types can be added in the future without compatibility disruptions. As more conditions are added, different consumer (e.g. different kube-proxy implementations) will have the option to evaluate the additional conditions. 

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
