# KEP-3070: Reserve Service IP Ranges For Dynamic and Static IP Allocation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Current Services ClusterIPs allocation model](#current-services-clusterips-allocation-model)
  - [Proposed Services ClusterIPs allocation model](#proposed-services-clusterips-allocation-model)
  - [Test Plan](#test-plan)
      - [Unit tests](#unit-tests)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Kubernetes Services are an abstract way to expose an application running on a
set of Pods. Services has a ClusterIP that is virtual and allows to load-balance
traffic across the different Pods. This ClusterIP can be assigned:

- dynamically, the cluster will pick one within the configured Service IP range.
- statically, the user will set one IP within the configured Service IP range.

Currently, there is no possibility, before creating a Service with a static Cluster IP,
to know if the IP has been chosen already by any other Service using dynamic allocation.

The Service IP range can be logically subdivided to avoid the risk of conflict between
Services that use static and dynamic IP allocation.

## Motivation

There are situation that users need to rely on well-known predefined IP addresses.

The best examples are the `kubernetes.default` and DNS Service addresses. For the former, the first
IP in the Service IP range is used to create this special service during the apiserver bootstrap process.
For the later, the IP is hardcoded in the kubelet flag `--cluster-dns`. For historical reasons, and by
convention, the Service IP for DNS to use is the 10th IP from the range, however, until the DNS
Service has been created, this IP can be taken by chance by any Service created with a dynamically
allocated IP.

There are also other use cases like the one explained in
https://github.com/kubernetes/kubernetes/issues/95570

### Goals

- allow to create a Service with a static IPs with less risk of IP conflict.

### Non-Goals

- changing the allocators implementation (bitmaps, ...).
- add new configuration options or flags.

## Proposal

### User Stories (Optional)

#### Story 1

"As a Kubernetes admin I want to be able to assign to my DNS Service the 10th IP of the Service IP range
with a low risk of conflicting with another Service that obtains its IP address dynamically"

#### Story 2

"As a Kubernetes developer I want to allocate safely fixed IPs to some Services so I can automate some
configuration parameters of my cluster"

### Risks and Mitigations

There is no risk. Every Service should be able to get an IP from the Service IP range as long as there are free IPs.

This KEP only subdivides the Service IP range, and prioritize one range over other for dynamically
IP allocation, without adding limitations to the number of IPs assigned.

## Design Details

### Current Services ClusterIPs allocation model

ClusterIP is the IP address of the service, if an address is specified manually, is in-range, and is not in use,
it will be allocated to the service; otherwise creation of the service will fail.
The Service ClusterIP range is defined in the apiserver with the following flag:

```
--service-cluster-ip-range string
A CIDR notation IP range from which to assign service cluster IPs. This must not overlap with any IP ranges assigned to nodes or pods.
```

In a multi master environment, multiple requests can arrive at same time at different apiservers, the allocation logic must
guarantee that there are no duplicate IP assigned. To solve this consensus problem in a distributed system, allocators are
implemented using bitmaps that are serialized and stored in an ["opaque" API object](https://github.com/kubernetes/kubernetes/blob/b246220/pkg/apis/core/types.go#L5311).

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

The [apiserver Service registry contains allocators backed by those bitmaps](p[kg/registry/core/rest/storage_core.go](https://github.com/kubernetes/kubernetes/blob/2ac6a4121f5b2a94acc88d62c07d8ed1cd34ed63/pkg/registry/core/rest/storage_core.go#L96-L103)) in order to assign these IPs.

```go
type LegacyRESTStorage struct {
	ServiceClusterIPAllocator          rangeallocation.RangeRegistry
	SecondaryServiceClusterIPAllocator rangeallocation.RangeRegistry
	ServiceNodePortAllocator           rangeallocation.RangeRegistry
}

...
func (c LegacyRESTStorageProvider) NewLegacyRESTStorage(restOptionsGetter generic.RESTOptionsGetter) (LegacyRESTStorage, genericapiserver.APIGroupInfo, error) {
  ...
	serviceClusterIPAllocator, err := ipallocator.New(&serviceClusterIPRange, func(max int, rangeSpec string) (allocator.Interface, error) {
		mem := allocator.NewAllocationMap(max, rangeSpec)
		// TODO etcdallocator package to return a storage interface via the storageFactory
		etcd, err := serviceallocator.NewEtcd(mem, "/ranges/serviceips", serviceStorageConfig.ForResource(api.Resource("serviceipallocations")))
		if err != nil {
			return nil, err
		}
		serviceClusterIPRegistry = etcd
		return etcd, nil
	})
```

The bitmaps implement a [strategy for allocating a random free value if none is specified](https://github.com/kubernetes/kubernetes/blob/2ac6a4121f5b2a94acc88d62c07d8ed1cd34ed63/pkg/registry/core/service/allocator/bitmap.go#L186-L205):

```go
// randomScanStrategy chooses a random address from the provided big.Int, and then
// scans forward looking for the next available address (it will wrap the range if
// necessary).
type randomScanStrategy struct {
	rand *rand.Rand
}

func (rss randomScanStrategy) AllocateBit(allocated *big.Int, max, count int) (int, bool) {
	if count >= max {
		return 0, false
	}
	offset := rss.rand.Intn(max)
	for i := 0; i < max; i++ {
		at := (offset + i) % max
		if allocated.Bit(at) == 0 {
			return at, true
		}
	}
	return 0, false
}
```

### Proposed Services ClusterIPs allocation model

The proposal is to implement a new strategy on the Service IP allocation bitmap that logically subdivide the range
in two bands based on the following formula `min(max($min, cidrSize/$step), $max)`, described as ~never less than $min or more than $max, with a graduated step function between them~, with $min = 16, $max = 256 and $step = 16. If cidrSize < $min the formula doesn't apply and the strategy will not change, both dynamic and static IPs will be allocated from the whole range with equal probability.

- lower band, used preferably for static ip assignment.
- upper band, used preferably for dynamic IP assignment.

Dynamically IP assignment will use the upper band by default, once this has been exhausted it will use the
lower range. This will allow users to define static allocations on the lower band of the predefined
range with a low risk of collision, keeping the implementation backwards compatible.

Example 1 Large Subnet:

- Service IP CIDR: 192.168.0.0/16
- Range Size: 2^16 - 2 = 65534
- Band Offset: min(max(16,65536/16),256) = min(4096,256) = 256
- Static band start: 192.168.0.1
- Static band ends: 192.168.1.0

           ┌─────────────┬─────────────────────────────────────────────┐
           │   static    │                    dynamic                  │
           └─────────────┴─────────────────────────────────────────────┘

           ◄────────────► ◄────────────────────────────────────────────►
    192.168.0.1     192.168.1.0                                 192.168.255.254

Example 2:

- Service IP CIDR: 192.168.0.0/22
- Range Size: 2^10 - 2 = 1022
- Band Offset: min(max(16,1024/16),256) = min(max(16,64),256) = 64
- Static band start: 192.168.0.1
- Static band ends: 192.168.0.64

           ┌─────────────┬─────────────────────────────────────────────┐
           │   static    │                    dynamic                  │
           └─────────────┴─────────────────────────────────────────────┘

           ◄────────────► ◄────────────────────────────────────────────►
    192.168.0.1     192.168.0.64                                 192.168.3.254

Example 3 Small Subnet:

- Service IP CIDR: 192.168.0.0/26
- Range Size: 2^6 - 2 = 62
- Band Offset: min(max(16,64/16),256) = min(max(16,4),256) = 16
- Static band start: 192.168.0.1
- Static band ends: 192.168.0.16

           ┌─────────────┬─────────────────────────────────────────────┐
           │   static    │                    dynamic                  │
           └─────────────┴─────────────────────────────────────────────┘

           ◄────────────► ◄────────────────────────────────────────────►
    192.168.0.1     192.168.0.16                                 192.168.0.62


```go
// randomScanReservedStrategy choose a random address from the provided big.Int and then scans
// forward looking for the next available address. The big.Int range is subdivided so it will try
// to allocate first from the reserved block for dynamic allocated addresses (it will wrap around
// back to the start of the dynamic subrange if necessary). If there is no free address it will
// try to allocate one from the reserved block for static allocated addresses too.
type randomScanReservedStrategy struct {
	rand     *rand.Rand
	reserved int
}
```

### Test Plan

This feature doesn't modify the cluster behavior, only the order on which dynamic IP are assigned to Services,
there is no need for e2e or integration tests, unit tests are enough.


[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Unit tests


- pkg/registry/core/service/allocator/bitmap_test.go - 84.2
- pkg/registry/core/service/ipallocator/allocator_test.go - 87

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag

#### Beta

- Gather feedback from developers and community.
- No issues reported.

#### GA

- No issues reported during two releases.

### Upgrade / Downgrade Strategy

The feature only changes the allocation strategy, the bitmaps and the underlay logic remain the same,
guaranteeing full compatibility and no risk rolling back the feature.

### Version Skew Strategy

Same as with upgrade, there is no change in the bitmaps, keeping the feature 100% compatible.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: ServiceIPStaticSubrange
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - The feature require to restart the apiserver to use the new allocation strategy for Services, however,
  it will not require downtime in multi-control-plane environments.
    
###### Does enabling the feature change any default behavior?

Dynamic allocated IPs will not be initially chosen from the first X values of the Service IP range, where X is obtained from the formula
described in [Proposed Services ClusterIPs allocation model](#proposed-services-clusterips-allocation-model)
This change is transparent to the user, since the IP is and was already random, it doesn't matter from which range the IP is obtained.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing, this only affects new requests and how these new request obtain the dynamically assigned IPs.

###### Are there any tests for feature enablement/disablement?

No need to add new test, current tests should keep working as today.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Current Services will not be affected, it doesn't matter in which range their IPs are allocated.
This only affects the creation of new Services without an IP specified, there is no risk of impact on running workloads.

###### What specific metrics should inform a rollback?

The allocation logic already has the following metrics, [some metrics have been extended with a
new label to contain information about the allocation scope requested](#monitoring-requirements):

  - allocated_ips
  - available_ips
  - allocation_total
  - allocation_errors_total

The increase of the errors metrics or the trend of the allocated_ips and available_ips doesn't change and new services are created or deleted.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Since it is a purely in-memory feature the upgrade or downgrande doesn't have any impact.
###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

### Monitoring Requirements

Following allocator metrics have been expanded with a new label to each metric containing information about the
type of allocation requested (dynamic or static):

  - allocation_total
  - allocation_errors_total

The other allocator the metrics can not use the additional label because, when an ip is released, there is not information on how the ip was allocated.

  - allocated_ips
  - available_ips

###### How can an operator determine if the feature is in use by workloads?

An operator will only observe that the change in behavior described for dynamically assigned ClusterIP for Services.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [X] Other (treat as last resort)
  - Details:
  - Create services without setting the ClusterIP and observe that all the services IPs allocated belong the upper band of the range,
    and once the upper band is exhausted it keep assigning IPs on the lower band until the whole range is exhausted.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A
###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?


- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

We could have a metric exposing the configuration parameters of the allocator, but that will collide
and is incompatible with a new KEP that expand the Service allocators to make the dynamically configurable.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No
### Scalability

###### Will enabling / using this feature result in any new API calls?

No
###### Will enabling / using this feature result in introducing new API types?

No
###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No
###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No
###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No
### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A
###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History


## Drawbacks

- v1.24 - Initial implementation https://github.com/kubernetes/kubernetes/pull/106792
- v1.25 - Beta
## Alternatives

- Replace current allocation implementation based on bitmaps https://github.com/kubernetes/enhancements/pull/1881
- Modify the allocation logic in the IP allocator, instead of the bitmap allocator.

The problem is that the IP allocator uses the bitmap allocator as backend. The bitmap allocator is the one that guarantees the consistency using etcd. 
Also, the IP allocator doesn't implement a strategy to acquire random IPs, this is done the bitmap allocator.
Implementing the logic in the IP allocator will not only be racy, it will also have an impact on performance, because to assign a dynamic IP, it will have to request a random IP to the
bitmap allocator and discard and retry until it find one in the desired range.
