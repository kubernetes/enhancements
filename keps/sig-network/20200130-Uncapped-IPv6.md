---
title: Uncapped IPv6
authors:
  - "@aojea"
owning-sig: sig-network
participating-sigs:
  - sig-scalability
  - sig-api-machinery
reviewers:
  - "@thockin"
  - "@bowei"
  - "@liggitt"
  - "@smarterclayton"
  - "@wojtek-t"
approvers:
  - "@thockin"
  - "@liggitt"
  - "@wojtek-t"
editor: TBD
creation-date: 2020-01-30
last-updated: 2020-01-30
status: provisional
see-also:
  - "/keps/sig-network/20190714-IPv6-beta-proposal.md"
  - "https://github.com/kubernetes/kubernetes/issues/44918"
  - "https://github.com/kubernetes/kubernetes/pull/86620"
  - "https://github.com/kubernetes/kubernetes/pull/79993"

---

# Remove IPv6 Allocation Restrictions

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Roaring bitmaps](#roaring-bitmaps)
  - [Services IP Allocator](#services-ip-allocator)
  - [Node IPAM Controller and Range Allocator](#node-ipam-controller-and-range-allocator)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

Kubernetes impose some restrictions on the number of Service IPs and CIDR Range sizes when using IPv6.
This is done to avoid scalability issues caused by IPv6, we should take into account that IPv4 has an
address namespace of 2^32 addresses vs 2^128 with IPv6.

## Motivation

One of the main motivation to restric allocations when using IPv6 is caused because Kubernetes is using internally bitmaps (or bitsets) to store the data.
Bitmaps are fast data structures, but they can use too much memory and not efficiently. Per example, in sparse scenarios like with the ip allocator,
if we allocate one IP at the position 1,000,000 we have to use over 100kB only for that IP.

The IPv6 allocation restrictions can be removed or relaxed if we use compressed bitmaps, it also will reduce the size of the data needed to store in etcd.

### Goals

Replace current bitmap structures that are causing IPv6 restrictions by [roaring bitmaps](http://roaringbitmap.org/about/).

### Non-Goals

Replace other bitmap structures that are not causing IPv6 restrictions.

Move out of the core REST or refactor the port and ip allocators.

## Proposal

Kubernetes has the following IPv6 restrictions:

- Maximum subnet mask size in the Node IPAM Controller Range Allocator, given by the difference between the cluster-cidr-mask size and the node-cidr-mask size.

https://github.com/kubernetes/kubernetes/blob/bd1195c28e22bbdcb95d718a31186581dc0a5f5f/pkg/controller/nodeipam/ipam/cidrset/cidr_set.go#L42-L51

```go
// The subnet mask size cannot be greater than 16 more than the cluster mask size
// TODO: https://github.com/kubernetes/kubernetes/issues/44918
// clusterSubnetMaxDiff limited to 16 due to the uncompressed bitmap
// Due to this limitation the subnet mask for IPv6 cluster cidr needs to be >= 48
// as default mask size for IPv6 is 64.
clusterSubnetMaxDiff = 16
// halfIPv6Len is the half of the IPv6 length
halfIPv6Len = net.IPv6len / 2
```

- Maximum subnet size (/12 for IPv4 and /112 for IPv6) for the Service Cluster CIDR in the kube-apiserver:

https://github.com/kubernetes/kubernetes/blob/bd1195c28e22bbdcb95d718a31186581dc0a5f5f/cmd/kube-apiserver/app/options/validation.go#L48-L54

```go
// Complete() expected to have set Primary* and Secondary*
// primary CIDR validation
var ones, bits = options.PrimaryServiceClusterIPRange.Mask.Size()
if bits-ones > 20 {
  errs = append(errs, errors.New("specified --service-cluster-ip-range is too large"))
}
```

- Maximum number of IPs (65536 for IPv6) for the Service Cluster subnet in the service IP allocator:

https://github.com/kubernetes/kubernetes/blob/5e31799701123c50025567b8534e1a62dbc0e9f6/pkg/registry/core/service/ipallocator/allocator.go#L275-L283

```go
  // For IPv6, the max size will be limited to 65536
  // This is due to the allocator keeping track of all the
  // allocated IP's in a bitmap. This will keep the size of
  // the bitmap to 64k.
  if bits == 128 && (bits-ones) >= 16 {
    return int64(1) << uint(16)
  } else {
    return int64(1) << uint(bits-ones)
  }
```

### Roaring bitmaps

Roaring bitmaps are compressed bitmaps which tend to outperform conventional compressed bitmaps such as WAH, EWAH or Concise. In some instances, they can be hundreds of times faster and they often offer significantly better compression.

There is a [roaring bitmap go version](https://github.com/RoaringBitmap/roaring) that will be imported into the Kubernetes project with its dependencies.

- github.com/RoaringBitmap/roaring
- github.com/glycerine/go-unsnap-stream
- github.com/glycerine/goconvey
- github.com/golang/snappy
- github.com/mschoch/smat
- github.com/philhofer/fwd
- github.com/tinylib/msgp
- github.com/willf/bitset

Roaring bitmap are implemented using an int32 as index, limiting the size of the compressed bitmap to 2^32.

### Services IP Allocator

The service registry use allocators for ports and services ips that use bitmaps that are stored in etcd.

The bitmap used for portallocator is out of the scope of this KEP because the range is relatively small (0-65535)
and doesn't depend on the IP family.

A new allocation map using roaring bitmaps, with a contiguous scan strategy for allocation to avoid sparse scenarios,
will be added for the Service IP allocator:

pkg/registry/core/service/allocator/bitmap.go

```go
// NewRoaringAllocationMap creates an allocation bitmap using the contiguous scan strategy.
func NewRoaringAllocationMap(max int, rangeSpec string) *RoaringBitmap {
  a := RoaringBitmap{
    allocated: roaring.NewBitmap(),
    count:     0,
    max:       max,
    rangeSpec: rangeSpec,
    }
  return &a
}
```

and will be used for the Service IP allocator:

pkg/registry/core/service/ipallocator/allocator.go

```go
// Helper that wraps NewAllocatorCIDRRange, for creating a range backed by an in-memory store.
func NewCIDRRange(cidr *net.IPNet) (*Range, error) {
  return NewAllocatorCIDRRange(cidr, func(max int, rangeSpec string) (allocator.Interface, error) {
    return allocator.NewRoaringAllocationMap(max, rangeSpec), nil
    })
}
```

There other limitations that are woth considering:

Etcd request size limit restrict the size of the bitmap
https://github.com/etcd-io/etcd/blob/master/Documentation/dev-guide/limit.md#request-size-limit

> etcd is designed to handle small key value pairs typical for metadata. Larger > requests will work, but may increase the latency of other requests. By default, the maximum size of any request is 1.5 MiB. This limit is configurable through --max-request-bytes flag for etcd server.

The funcion ForEach() iterates over all the elements of the bitmap, the time between resyncs will impose a limit on the size of the bitmap.

```go
// ForEach calls the provided function for each allocated bit.  The
// AllocationBitmap may not be modified while this loop is running.
func (r *AllocationBitmap) ForEach(fn func(int)) {
```

The utils/net RangeSize() funcion is limited to MaxInt64 too

```go
// RangeSize returns the size of a range in valid addresses.
// returns the size of the subnet (or math.MaxInt64 if the range size would overflow int64)
func RangeSize(subnet *net.IPNet) int64 {
	ones, bits := subnet.Mask.Size()
	if bits == 32 && (bits-ones) >= 31 || bits == 128 && (bits-ones) >= 127 {
		return 0
	}
	// this checks that we are not overflowing an int64
	if bits-ones >= 63 {
		return math.MaxInt64
	}
	return int64(1) << uint(bits-ones)
}
```

https://gitlab.cncf.ci/kubernetes/kubernetes/commit/8725f720831adbdfbd70819c7134111a0967b6a6?view=parallel

### Node IPAM Controller and Range Allocator

We can add new Range allocator based on Roaring Bitmaps
--cidr-allocator-type="RangeAllocator"
https://github.com/kubernetes/kubernetes/blob/ded6ee953c68f8333ee6291e0bcb7e58604fac00/pkg/controller/nodeipam/ipam/doc.go#L17-L29

The Node IPAM controller manages the assigned CIDR ranges to each node.
It uses a CidrSet to manage a set of CIDR ranges from which blocks of IPs can be allocated from.
A new CidrSet struct using roaring bitmaps will be added:

```go
// CidrSet manages a set of CIDR ranges from which blocks of IPs can
// be allocated from.
// It allocates the IP subnet addresses in a bitmap as an offset of the ClusterCIDR IP subnet
// The masks of the ClusterCIDR and NodeCIDRs are known in advance
type CidrSet struct {
  sync.Mutex
  clusterCIDR   *net.IPNet
  subnetMask    net.IPMask
  base          *big.Int
  offset        *big.Int
  maxCIDRs      int
  nextCandidate int
  used          *roaring.Bitmap
}
```

### User Stories

#### Story 1

As a Kubernetes user I want to be able to choose any subnet mask without restrictions for my Nodes or Service IPs.

#### Story 2

As a Kubernetes developer I want to reduce the size of memory used for Service IPs and CIDRs Ranges allocation
without any significant penalty on performance.

### Implementation Details/Notes/Constraints

PR with a prototype here:

https://github.com/kubernetes/kubernetes/pull/86620

Introduce a new feature gate Roaring?? that allow to use the new bitmap structures.

### Risks and Mitigations

The roaring bitmaps have a limited range of 2^32 integer that seems big enough.
This implies that user won't be able to use all the ranges configured.

The [range limitation can be removed if needed](https://github.com/RoaringBitmap/CRoaring/issues/1#issuecomment-168721352), but the effort doesn't seem worth the value and the data stored can be too big.

Other option is to relax current limitations to the new limit given by the roaring bitmaps.

## Design Details

### Test Plan

* Benchmark tests comparing performance, memory and CPU usage
* Scalability tests (TBD)
* Upgrade tests

### Graduation Criteria

Alpha -->  Beta
 
- TBD

Beta --> GA

- TBD

### Upgrade / Downgrade Strategy

- The ipallocator will need a new prefix in the etcd registry to avoid conflicts with previous
or older bitmaps versions.
Once enabled the new bitmap storage, the repair loop will recreate the right entries in the new storage.

- The ipam controller store the status on the node spec fields

### Version Skew Strategy

- The ipallocator will need a new prefix in the etcd registry to avoid conflicts with previous
or older bitmaps versions.
Once enabled the new bitmap storage, the repair loop will recreate the right entries in the new storage.

- The ipam controller store the status on the node spec fields

## Drawbacks

Current limits are not very restrictive and the lack of IPv6 deployments doesn't allow to evaluate how really
useful will be this change.

## Alternatives

There are other compressed bitmap implementations or data structures that can be used.
