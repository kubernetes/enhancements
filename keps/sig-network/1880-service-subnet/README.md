# KEP-1880: Kubernetes Services ClusterIP and NodePort Allocators API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Future Goals](#future-goals)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
    - [Story 5](#story-5)
    - [Story 6](#story-6)
    - [Story 7](#story-7)
- [Proposal](#proposal)
  - [Current Services ClusterIPs and NodePort allocation model](#current-services-clusterips-and-nodeport-allocation-model)
  - [Proposed Allocation model](#proposed-allocation-model)
    - [IP Addresses](#ip-addresses)
  - [NodePort](#nodeport)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
  - [Phase I](#phase-i)
  - [Phase 2](#phase-2)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
    - [Alternative 1](#alternative-1)
    - [Alternative 2](#alternative-2)
    - [Alternative 3](#alternative-3)
    - [Alternative 4](#alternative-4)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP solves current Kubernetes Services issues and constraintes caused by current allocation model,
offering a public API for Services resources allocation.

## Motivation

Current Services allocation implementation cause different issues that are not easily solvable, like leaking
ClusterIPs https://github.com/kubernetes/kubernetes/issues/87603.

It also imposes some constraints and limitations on the system:

* the size of the Service Cluster CIDR, for IPv4 the prefix size is limited to /12, however,
for IPv6 it is limited to /112 or fewer.This restriction is causing issues for IPv6 users, since /64 is the standard and minimum recommended prefix length.

* There is no possibility to reserve a range of Service IP addresses 
https://github.com/kubernetes/kubernetes/issues/95570

* The Service IP Range is not resizable, it's configured by a flag in each apiserver, that complicates still
more the possibility of migrate or resize it in a multi-control-plane environment.

* The Service IP range is not exposed, so it is not possible for other components in the cluster to consume it.

* The NodePort range is fixed and doesn't support multiple port-ranges
https://github.com/openshift/enhancements/pull/396

* It complicates the Service REST API, the model doesn't scale well, as we could see during the dual-stack implementations, so any expansions to N service-subnets is very difficult.


### Goals

Implement an API for ClusterIP and NodePort Services allocation that:
https://github.com/kubernetes/enhancements/pull/1881#issuecomment-757003362

> * scales with the number of reservations, not with the space of possible reservations auditing does not require a lock of any sort
> * local interactions are easily checkable, you don't need to be aware of everything else that has been allocated conflicts should be rare and easy to retry (i.e., trying to reserve an object that another apiserver just reserved)
> * the number of extra allocation objects is tunable -- this can be very lightweight for small clusters


### Non-Goals

Any change unrelated to Services, ClusterIP and NodePort, specifically any generalization, like an IPAM API.

### Future Goals

* Multiple Service Subnets per cluster
* Service Subnet migration

### User Stories

#### Story 1

As a Kubernetes admin that wants to use IPv6, I want to be able to follow the IPv6 address
planning best practices, being able to assign /64 prefixes to my end subnets.

#### Story 2

As a Kubernetes admin I want to be able to resize my current Service subnet with zero downtime, or
requiring complex operations.

#### Story 3

As a Kubernetes user I want to reserve specific IP addresses from the Service subnet.

#### Story 4

As a Kubernetes user I want to dynamically change the node-port range so I can allocate more node-ports
in case my range haa been exhausted.

#### Story 5

As a Kubernetes developer I want to be able to know the current Service IP range and the IP adddresses allocated.

#### Story 6

As a Kubernetes developer working on multicluster I want to be able to create multiple Service Subnets with
different characteristics.

#### Story 7

As a Kubernetes admin using a dual-stack cluster I want to be able to change the primary IP family of the cluster without any disruption.

## Proposal

### Current Services ClusterIPs and NodePort allocation model

[Kubernetes Services](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#service-v1-core) need to be able to dynamically allocate resources from a predefined range/pool for some fields:

* ClusterIPs

ClusterIP is the IP address of the service and is usually assigned randomly. If an address is specified manually, is in-range, and is not in use, it will be allocated to the service; otherwise creation of the service will fail. 
The Service ClusterIP range is defined in the apiserver with the following flag:

```
--service-cluster-ip-range string
A CIDR notation IP range from which to assign service cluster IPs. This must not overlap with any IP ranges assigned to nodes or pods.
```
And in the controller-manager:
```
--service-cluster-ip-range string
CIDR Range for Services in cluster. Requires --allocate-node-cidrs to be true
```

* NodePort

Some Service Types (LoadBalancer, NodePort) allocates a special port on each node to expose services in a port from a range specified by --service-node-port-range flag. Each node proxies that port (the same port number on every Node) into your Service.
The Service NodePort range is defined in the apiserver with the following flag:

```
--service-node-port-range portRange     Default: 30000-32767
A port range to reserve for services with NodePort visibility. Example: '30000-32767'. Inclusive at both ends of the range.
```

These allocation requirements are one of the principal reasons of the [complicated Service REST implementation](https://github.com/kubernetes/kubernetes/pull/96684):

> Service has an "outer" and "inner" REST handler. They use the same strategy, but the outer calls into the inner, which causes a lot of complexity in the code (including an open-coded partial reimplementation of a date-unknown snapshot of the generic REST code) and results in Prepare and Validate hooks being called twice.

The allocators are implemented using bitmaps that are serialized and stored in an ["opaque" API object](https://github.com/kubernetes/kubernetes/blob/b246220/pkg/apis/core/types.go#L5311).

```go
// RangeAllocation is an opaque API object (not exposed to end users) that can be persisted to record
// the global allocation state of the cluster. The schema of Range and Data generic, in that Range
// should be a string representation of the inputs to a range (for instance, for IP allocation it
// might be a CIDR) and Data is an opaque blob understood by an allocator which is typically a
// binary range.  Consumers should use annotations to record additional information (schema version,
// data encoding hints). A range allocation should *ALWAYS* be recreatable at any time by observation
// of the cluster, thus the object is less strongly typed than most.
type RangeAllocation struct {
	metav1.TypeMeta
	// +optional
	metav1.ObjectMeta
	// A string representing a unique label for a range of resources, such as a CIDR "10.0.0.0/8" or
	// port range "10000-30000". Range is not strongly schema'd here. The Range is expected to define
	// a start and end unless there is an implicit end.
	Range string
	// A byte array representing the serialized state of a range allocation. Additional clarifiers on
	// the type or format of data should be represented with annotations. For IP allocations, this is
	// represented as a bit array starting at the base IP of the CIDR in Range, with each bit representing
	// a single allocated address (the fifth bit on CIDR 10.0.0.0/8 is 10.0.0.4).
	Data []byte
}
```

To avoid leaking resources, they also need to implement to repair loops that run in the `controlplane` instance
that keep the bitmaps in sync with the current Services allocated IPs.

### Proposed Allocation model
https://github.com/kubernetes/enhancements/pull/1881#issuecomment-757003362

* define a new Object (i.e. IPRange and NodePortRange) that define the range used for allocation.
The value of the range is the one passed as parameters in the apiserver flags for backwards compatibility.
```
<<[UNRESOLVED range modification]>>

Since the Range objects will be created by the bootstrap logic of the apiserver, the first apiserver to start will define the Range for the whole cluster.
The Range objects can not be deleted, but can be updated meanwhile the objects allocated referencing it are withing the new range. Per example, in a cluster with a Service Range 10.0.0.0/24, it will be always possible to update the service to a larger subnet 10.0.0.0/16. However, it will not be possible to use a smaller subnet, i.e. 10.0.0.0/28, if there are IPs allocated out if that range, i.e. 10.0.0.244.

<<[/UNRESOLVED]>>


```

* define another object (ie. IPAddress and NodePort) that represent the allocated resources, the name of
the object will be the port or the IP, so we can guarantee the consistency.

* the allocated object must reference the object used to define the range allocation, and validated against it (i.e. IP within range)

```
<<[UNRESOLVED allocation mode]>>

* and allocated object can have different status: allocated, reserved, free, ...

* on service creation, if the NodePort or ClusterIP field is set:
1. check that the object doesn't exist in the apiserver and Create a new one (owned by the Service)
2. check if the object exists and the Status is Free and change the status.

* on service creation, if the NodePort or ClusterIP field is not set, allocate a random one from the defined range in the corresponding Object (IPRange or NodePortRange)

There can be different model to allocate a random resource:

1. Same as today, the apiserver can keep an index with current status and start with a random offset to
find a free resources.

2. A controller create a pool of free resource Objects and the apiserver picks one of them, this adds 
more complexity to the model, but will improve the time to allocate a new free resources since is just
popping an object from a list.

<<[/UNRESOLVED]>>
```

## Design Details


### Cluster IPs

#### IP Ranges

Define two new objects, a new IPRange Object that defines the range used to allocate Services Cluster IPs

```go
// IPRange defines a range of IPs using CIDR format (192.168.0.0/24 or 2001:db2::0/64).
type IPRange struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPRangeSpec   `json:"spec,omitempty"`
	Status IPRangeStatus `json:"status,omitempty"`
}

// IPRangeSpec describe how the IPRange's specification looks like.
type IPRangeSpec struct {
	// Range of IPs in CIDR format (192.168.0.0/24 or 2001:db2::0/64).
	Range string `json:"range"`
	// Primary indicates if this is the primary allocator to be used by the
	// apiserver to allocate IP addresses.
	// NOTE this can simplify the Service strategy logic so we don't have to infer
	// the primary allocator, it also may allow to switch between primary families in
	// a cluster, but this looks like a loooong shot.
	// +optional
	Primary bool `json:"primary,omitempty"`
}

// IPRangeStatus defines the observed state of IPRange.
type IPRangeStatus struct {
	// Free represent the number of IP addresses that are not allocated in the Range.
	// +optional
	Free int64 `json:"free,omitempty"`
}
```

Reserve well-known-names for the IPRanges objects, that represent current allocators service-subnet-ranges:

```go
// default allocator well-known-names
const (
	DefaultIPv4Allocator = "allocator.ipv4.k8s.io"
	DefaultIPv6Allocator = "allocator.ipv6.k8s.io"
)
```

On the controlplane controller, add a new function similar to the existing ones used for [bootraping the default kubernetes services and namespaces](https://github.com/kubernetes/kubernetes/blob/ff34bd246b26bef622d35de68f3c143055e3aac7/pkg/controlplane/controller.go#L246-L251), to create the IPRange from the vale defined in the corresponding flag with the service-iprange-subnet


```go
	if utilfeature.DefaultFeatureGate.Enabled(features.ClusterIPAllocatorAPI) {
		if err := createAllocatorIfNeeded(c.AllocatorClient, c.ServiceClusterIPRange); err != nil {
			return err
		}
		if utilfeature.DefaultFeatureGate.Enabled(features.IPv6DualStack) && c.SecondaryServiceClusterIPRange.IP != nil {
			if err := createAllocatorIfNeeded(c.AllocatorClient, c.SecondaryServiceClusterIPRange); err != nil {
				return err
			}
		}
	}
```

This way we guarantee that the Range Objects has been created before the Services.

```
<<[UNRESOLVED IPRange API Operations]>>

* IPRange objects can not be deleted

* an update on the Range field can only be allowed if there are no IP addresses allocated out of the new range.

* a controller will watch IPAddress operations to update the IPRange.Status.Free value with the number
of IP addresses free in the range

<<[/UNRESOLVED]>>
```

#### IP Addresses

New IP Address Object that represents a ClusterIP in the cluster

```go
// IPAddress represents an IP used by Kubernetes associated to an IPRange.
// The name of the object is the IP address decimal number, because colons
// are not allowed and IPv6 addresses have different text representations.
// xref: https://tools.ietf.org/html/rfc4291
type IPAddress struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IPAddressSpec   `json:"spec,omitempty"`
	Status IPAddressStatus `json:"status,omitempty"`
}

// IPRangeRef contains information that points to the IPRange being used
type IPRangeRef struct {
	// APIGroup is the group for the resource being referenced
	APIGroup string
	// Kind is the type of resource being referenced
	Kind string
	// Name is the name of resource being referenced
	Name string
}

// IPAddressSpec describe the attributes in an IP Address,
type IPAddressSpec struct {
	// Address is the text representation of the IP Address.
	Address string `json:"address"`
	// IPRangeRef references the IPRange associated to this IP Address.
	IPRangeRef IPRangeRef `json:"ipRangeRef,omitempty"`
}

// IPAddressStatus defines the observed state of IPAddress.
type IPAddressStatus struct {
	State IPAddressState `json:"state,omitempty"`
}

// IPAddressState defines the state of the IP address
type IPAddressState string

// These are the valid statuses of IPAddresses.
const (
	// IPPending means the IP has been allocated by the system but the object associated
	// (typically Services ClusterIPs) has not been persisted yet.
	IPPending IPAddressState = "Pending"
	// IPAllocated means the IP has been persisted with the object associated.
  IPAllocated IPAddressState = "Allocated"
  // IPReserved means the IP has been reserved and can not be allocated to any Service.
	IPReserved IPAddressState = "Reserved"
	// IPFree means that IP has not been allocated neither persisted.
	IPFree IPAddressState = "Free"
)
```

The name of the IPAddress has to be unique, so we can use it to provide consistency and avoid duplicate allocations.
However, Kubernetes objects doesn't allow names with `:`, used by IPv6 Addresses. In addition, [IPv6 addresses
can have different text representations](https://tools.ietf.org/html/rfc4291).

To guarantee the name uniqueness, we can use the decimal representation of the IP address, since IP can be a 32 bits or a 128 bits number. To facilitate this operations we introduce two new utils functions:

```go
// IPToDecimal converts an IP to a string with its decimal representation
func IPToDecimal(ip net.IP) string {
	if ip == nil {
		return ""
	}
	i := new(big.Int)
	// compare the address with its decimal representation
	if ip.To4() != nil {
		i.SetBytes(ip.To4())
	} else if ip.To16() != nil {
		i.SetBytes(ip.To16())
	}
	return i.String()
}

// DecimalToIP converts a string with the decimal representation of an IP address and returns the IP
func DecimalToIP(ipDecimal string) net.IP {
	if len(ipDecimal) == 0 {
		return nil
	}
	// convert decimal representation to BigInt
	i, ok := new(big.Int).SetString(ipDecimal, 10)
	// if we can not convert the string to a number
	// or the number has more than 16 bytes it is not an IP
	if !ok || len(i.Bytes()) > 16 {
		return nil
	}
	bufLen := net.IPv4len
	if len(i.Bytes()) > net.IPv4len {
		bufLen = net.IPv6len
	}
	// convert BigInt to IP address, truncate to IP Len bytes
	r := append(make([]byte, bufLen), i.Bytes()...)
	return net.IP(r[len(r)-bufLen:])
}
```


```
<<[UNRESOLVED IPAddress API Operations]>>

* IPAddress objects created must reference the IPRange object, and validated against it. 
The `address` field will contain the IP address representation, if no one is provided it will be defaulted useing the value obtained from the name. If the `address` is set it must match the `name`.

* IPAddress status updates depend on the allocation model chosen, it may allowed to change after creation,

QUESTION garbage collection, cluster and namespaced scoped objects. Which will be the best scope for these new objects? If we can make that IPAddresses are owned by the Services that allocate them, the garbage collector when deleting a Service will take care of the release????

<<[/UNRESOLVED]>>
```

### NodePort

TBD


### Test Plan

* Unit tests

TBD

* Integration Tests

TBD

* E2E tests 


### Graduation Criteria

The change must be transparent to the users, so there should not be any performance degradation
or Services breaking changes. 

### Upgrade / Downgrade Strategy

The service-subnet and node-port range will keep using the flags for the initial configuration, it can
happen that the service subnet or node-port range are modified after the initial configuration, in that
case they will use the same value as configured by the flags.

### Version Skew Strategy

That apisever control-plane will keep using repair loops to sync current service IP allocated with
the corresponding allocator, that guarantee that the skeweved apiservers work because they have the
same source of truth.

```
<<[UNRESOLVED detect apiservers with different version ]>>

Any update on the IPRange allocated object will not be allowed if there are apiservers without the
same version
QUESTION Is this even possible?

<<[/UNRESOLVED]>>
```

## Implementation History

### Phase I

The first phase consists in introduce the new API objects as a dropin replacement of current bitmap allocators.
This can be done just [implementing the `Allocator.Interface`](https://github.com/kubernetes/kubernetes/blob/master/pkg/registry/core/service/ipallocator/allocator.go) using the new API objects:

```go
// Interface manages the allocation of IP addresses out of a range. Interface
// should be threadsafe.
type Interface interface {
	Allocate(net.IP) error
	AllocateNext() (net.IP, error)
	Release(net.IP) error
	ForEach(func(net.IP))
	CIDR() net.IPNet

	// For testing
	Has(ip net.IP) bool
}
```

There is actually a working prototype with this approach in https://github.com/kubernetes/kubernetes/pull/98280

### Phase 2

The next phase consists in simplifying the Serice REST API leveraging the new API changes introduced in Phase I.

This buils on top of Tim Hockin changes:

https://github.com/kubernetes/kubernetes/pull/96684
https://github.com/kubernetes/kubernetes/pull/95666


With the new hooks and the new API fields, we can use API machinery tools and primitives to deal
with the resource allocation and simplify the Services REST Strategy.


```
<<[UNRESOLVED allocated resources deletion ]>>

Allocated resources can be owned by the services, so the Garbage Collector guarantees the release of the
allocated resources with the 

<<[/UNRESOLVED]>>
```

## Drawbacks

Compared to a bitmap this design takes more space per reservation,
which is superior only at some level of sparseness

## Alternatives

The inital goal was to remove the service subnet restriction for IPv6, where only /112 subnets
or fewer weren't allowed due to the bitma restrictions.
However, since the main problem was the allocation model for Services, the scope has been redefined
to change the allocation model and solve the real problem instead one of its consequences:

Several alternatives were proposed in the original PR:

#### Alternative 1

https://github.com/kubernetes/enhancements/pull/1881#issuecomment-672090253

#### Alternative 2

https://github.com/kubernetes/enhancements/pull/1881#issuecomment-672135245


#### Alternative 3

https://github.com/kubernetes/enhancements/pull/1881#issuecomment-672764247


#### Alternative 4

https://github.com/kubernetes/enhancements/pull/1881#issuecomment-755737620