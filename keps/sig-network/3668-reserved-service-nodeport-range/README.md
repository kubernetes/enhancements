# KEP-3668: Reserve Nodeport Ranges For Dynamic And Static Port Allocation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Story 1](#story-1)
  - [Story 2](#story-2)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Current Services NodePort allocation model](#current-services-nodeport-allocation-model)
  - [Proposed Services NodePorts allocation model](#proposed-services-nodeports-allocation-model)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Nodeport Service can expose a Service outside the cluster, and allow external applications access to an set of Pods. NodePort has several ports that are widespread in the cluster and allow to load-balance traffic from the external. And the port number can be assigned:

- dynamically, the cluster will pick one within the configured service node port range.
- statically, the user will set one port within the configured service node port range.

Currently, there is no possibility, before creating a NodePort Service with a static port, to know if the port has been chosen already by any other NodePort using dynamic allocation.

The NodePort Service port range can be logically subdivided to avoid the risk of conflict between NodePort that use static and dynamic port allocation. The idea is mostly the same with [Reserve Service IP Ranges (KEP-3070)](https://github.com/kubernetes/enhancements/tree/master/keps/sig-network/3070-reserved-service-ip-range#release-signoff-checklist) did.

## Motivation

There are many situations that users need to rely on well-known predefined NodePort ports. 

The usage scenario is similar in nature to the pre-assigned Cluster IP addresses described in KEP 3070: in some deployments, IPs or ports need to be hard-coded in some specific components of the cluster. Hard-coding of NodePort typically occurs in private cloud deployments with complex hyper-converged architectures, where we cannot rely on some of the underlying infrastructure, or even need to use the services running on Kubernetes as a base service. As a concrete example, imagine we need to deploy a set of applications in an environment without DNS servers and external LBs, this set of applications includes several K8S-based deployments, but others are non-K8S-based which deploied on baremetal. At this point, CoreDNS is expected to be the only DNS server in the environment, and both need to provide resolution services not only within K8S, but also outside the K8S cluster, where a fixed IP and NodePort is necessary.

During cluster initialization, some services created earlier may take up the CluserIP and NodePort ports that we expect to assign later, the ClusterIP problem is solved in and KEP 3070, while the NodePort problem may still occur.


### Goals

- allow to create a Service with a static NodePort with less risk of port conflict.


### Non-Goals

- changing the allocators implementation (bitmaps, ...).
- add new configuration options or flags.

## Proposal

### Story 1

"As a Kubernetes administrator, I would like to be able to assign a fixed port to my DNS Service to provide access to services outside the cluster and reduce the possibility of conflicts with existing NodePort Services within the cluster."

### Story 2
"As a Kubernetes developer I want to allocate safely fixed ports to some NodePort Services so I can automate some configuration parameters of my cluster"

### Risks and Mitigations
There is no risk. Every NodePort Service should be able to get an port from the NodePort range as long as there are free ports.

This KEP only subdivides the NodePort range, and prioritize one range over other for dynamically port allocation, without adding limitations to the number of ports assigned.

## Design Details

### Current Services NodePort allocation model

nodePort is an additional port that needs to be bound to all nodes for the service, if a port is specified manually, is in-range, and is not in use,
it will be allocated to the nodePort; otherwise creation of the service will fail.
The Service NodePort range is defined in the apiserver with the following flag:

```
--service-node-port-range <a string in the form 'N1-N2'>     Default: 30000-32767
A port range to reserve for services with NodePort visibility. This must not overlap with the ephemeral port range on nodes. Example: '30000-32767'. Inclusive at both ends of the range.
```

In a multi master environment, multiple requests can arrive at same time at different apiservers, the allocation logic must
guarantee that there are no duplicate port assigned. To solve this consensus problem in a distributed system, allocators are
implemented using bitmaps that are serialized and stored in an ["opaque" API object](https://github.com/kubernetes/kubernetes/blob/6a16d7d31aeb0f95c4ae513311bfecef9492f30e/pkg/apis/core/types.go#L5714).

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

The [apiserver Service registry contains allocators backed by those bitmaps](p[kg/registry/core/rest/storage_core.go](https://github.com/kubernetes/kubernetes/blob/2ac6a4121f5b2a94acc88d62c07d8ed1cd34ed63/pkg/registry/core/rest/storage_core.go#L96-L103)) in order to assign these ports.

```go
type LegacyRESTStorage struct {
	ServiceClusterIPAllocator          rangeallocation.RangeRegistry
	SecondaryServiceClusterIPAllocator rangeallocation.RangeRegistry
	ServiceNodePortAllocator           rangeallocation.RangeRegistry
}

...
func (c LegacyRESTStorageProvider) NewLegacyRESTStorage(apiResourceConfigSource serverstorage.APIResourceConfigSource, restOptionsGetter generic.RESTOptionsGetter) (LegacyRESTStorage, genericapiserver.APIGroupInfo, error) {
  ...
	var serviceNodePortRegistry rangeallocation.RangeRegistry
	serviceNodePortAllocator, err := portallocator.New(c.ServiceNodePortRange, func(max int, rangeSpec string) (allocator.Interface, error) {
		mem := allocator.NewAllocationMap(max, rangeSpec)
		// TODO etcdallocator package to return a storage interface via the storageFactory
		etcd, err := serviceallocator.NewEtcd(mem, "/ranges/servicenodeports", serviceStorageConfig.ForResource(api.Resource("servicenodeportallocations")))
		if err != nil {
			return nil, err
		}
		serviceNodePortRegistry = etcd
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

### Proposed Services NodePorts allocation model

The strategy proposaled here is mentioned and implemented in (KEP 3070)[https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/3070-reserved-service-ip-range/README.md#proposed-services-clusterips-allocation-model] first, with some fine-tuning. The following is a specific explanation of the strategy.

the strategy following formula `min(max($min, node-range-size/$step), $max)`, described as never less than $min or more than $max, with a graduated step function between them, with $min = 16, $max = 128 and $step = 32. If node-range-size < $min the formula doesn't apply and the strategy will not change, both dynamic and static ports will be allocated from the whole range with equal probability.

- lower band, used preferably for static port assignment.
- upper band, used preferably for dynamic port assignment.
  
Dynamically port assignment will use the upper band by default, once this has been exhausted it will use the lower range. This will allow users to define static allocations on the lower band of the predefined range with a low risk of collision, keeping the implementation backwards compatible.


Example 1 Default service nodeport range:

- Service Node Port Range: 30000-32767
- Range Size: 32767 - 30000 = 2767
- Band Offset: min(max(16,2767/32),128) = min(86,128) = 86
- Static band start: 30000
- Static band ends: 30086

           ┌─────────────┬─────────────────────────────────────────────┐
           │   static    │                    dynamic                  │
           └─────────────┴─────────────────────────────────────────────┘

           ◄────────────► ◄────────────────────────────────────────────►
          30000        30086                                          32767


Example 2 Large service nodeport range:

- Service Node Port Range: 20000-32767
- Range Size: 32767 - 30000 = 12767
- Band Offset: min(max(16,12767/32),128) = min(398,128) = 128
- Static band start: 20000
- Static band ends: 20128

           ┌─────────────┬─────────────────────────────────────────────┐
           │   static    │                    dynamic                  │
           └─────────────┴─────────────────────────────────────────────┘

           ◄────────────► ◄────────────────────────────────────────────►
          20000        20128                                          32767


Example 3 Small service nodeport range:

- Service Node Port Range: 32567-32767
- Range Size: 32767 - 32567 = 200
- Band Offset: min(max(16,200/32),128) = min(max(16, 6),128) = 16
- Static band start: 32567
- Static band ends: 32583

           ┌─────────────┬─────────────────────────────────────────────┐
           │   static    │                    dynamic                  │
           └─────────────┴─────────────────────────────────────────────┘

           ◄────────────► ◄────────────────────────────────────────────►
          32567        32583                                          32767



### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates
None

##### Unit tests
- `kubernetes/pkg/registry/core/service/portallocator`: `12/10/2022` - `83.1`

##### Integration tests

This feature doesn't modify the cluster behavior, only the order on which dynamic port are assigned to NodePort Services, there is no need for e2e or integration tests, unit tests are enough.

##### e2e tests

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
  - Feature gate name: ServiceNodePortStaticSubrange
  - Components depending on the feature gate: kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

Dynamic allocated NodePort will not be initially chosen from the first X values of the Service Node Port range, where X is obtained from the formula
described in [Proposed Services NodePorts allocation model](#proposed-services-nodeports-allocation-model)
This change is transparent to the user, since the port is and was already random, it doesn't matter from which range the port is obtained.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing, this only affects new requests and how these new request obtain the dynamically assigned ports.


###### Are there any tests for feature enablement/disablement?

No need to add new test, current tests should keep working as today.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Current Services will not be affected, it doesn't matter in which range their port are allocated.
This only affects the creation of new NodePort Services without an port specified, there is no risk of impact on running workloads.

###### What specific metrics should inform a rollback?

There is currently no metric available for the NodePort allocation. so 4 metric should be added.

  - kube_apiserver_nodeport_allocator_allocated_ports
  - kube_apiserver_nodeport_allocator_available_ports
  - kube_apiserver_nodeport_allocator_allocation_total
  - kube_apiserver_nodeport_allocator_allocation_errors_total

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Since it is a purely in-memory feature the upgrade or downgrande doesn't have any impact.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?
no

### Monitoring Requirements

Following allocator metrics have a new label to each metric containing information about the type of allocation requested (dynamic or static):

  - kube_apiserver_nodeport_allocator_allocation_total
  - kube_apiserver_nodeport_allocator_allocation_errors_total

The other allocator the metrics can not use the additional label because, when an port is released, there is not information on how the port was allocated.

  - kube_apiserver_nodeport_allocator_allocated_ports
  - kube_apiserver_nodeport_allocator_available_ports

###### How can an operator determine if the feature is in use by workloads?

An operator will only observe that the change in behavior described for dynamically assigned NodePort for Services.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [X] Other (treat as last resort)
  - Details: Create services without setting the nodePort and observe that all the services ports allocated belong the upper band of the range, and once the upper band is exhausted it keep assigning ports on the lower band until the whole range is exhausted.

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

This proposal adds 4 more metrics, and two of them is used to observe this feature. I think there is no any missing metrics should be added in feature.

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A
###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

## Drawbacks

## Alternatives
