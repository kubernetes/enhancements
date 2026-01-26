# KEP-5773: ResourceSlice Priority

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Implementation Details](#implementation-details)
  - [Validation](#validation)
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
  - [Lexicograph ordering of ResourceSlices and resource pools based on names](#lexicograph-ordering-of-resourceslices-and-resource-pools-based-on-names)
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

This KEP proposes a way to let Dynamic Resource Allocation (DRA) drivers control the order in which
the scheduler attempts to allocate devices to ResourceClaims. Since the allocator
has a first-fit algorithm, the order in which devices are considered can affect both which devices
are allocated to a claim and how long it will take the allocator to find a set of devices that meet
all the criteria.

## Motivation

The DRA allocator is responsible for allocating devices to ResourceClaims. It does this by searching
through the available devices on a node, which are provided through the ResourceSlice API. It goes
through the devices and attempts to find a set of devices that satisfies all the criteria defined
in the ResourceClaim. As soon as it finds a set of devices that works, it stops the search and
returns the solution.

If the ResourceClaim asks for a GPU with at least 1Gi of memory and the ResourceSlices contains
devices with memory ranging from 1Gi to 80Gi, the order in which devices are considered are
important. If the allocator attempts a GPU with 80Gi of memory first, it will find that it
satisfies the ResourceClaim and it will be returned as the solution. But this is not a very
efficient solution since allocating a GPU with just the right amount of memory would be
better. By letting the driver control the order of devices in the allocator, it can publish
the devices in an order of increasing memory, leading to a higher likelihood that a device
with a better fit will be chosen.

ResourceSlices might also include network-atteched devices, that needs to be attached to the
node after a pod has been scheduled. This can lead to slower startup time, so typically it
will be better to allocate local devices rather than network-attached devices if possible.
Since ResourceSlices with network-attached devices will typically be published by a
controller rather than by a driver running on a node, they will be published as part of a
different resource pool than node-local devices.

### Goals

* Give DRA drivers a way to signal to the allocator the order in which devices should
  be considered.

### Non-Goals

* Provide a way to order the way the allocator considers devices across drivers.
* Full implementation of scoring.

## Proposal

### Risks and Mitigations

* The need for this feature is partly a result of not having scoring, and as a result,
  the scheduler only does a first-fit search for devices. With scoring, the order of the
  devices across resource pools and ResourceSlices doesn't matter since the allocator
  would be responsible for determining which sets of devices that needs to be considered
  to know that we have at least close to a best-fit solution. Note that this doesn't necessarily
  mean all possible sets of devices will have to be considered as that might be too
  expensive. So this feature might be less relevant when we implement scoring, but we don't
  currently have any timeline for when - or if - that might happen.

* This will require that the order of ResourceSlices and resource pools are sorted
  in the allocator. This means additional work in the allocator so it might
  negatively impact the scheduling throughput for DRA workloads.

## Design Details

The proposal is to add two new fields to the ResourceSlice API, one for specifying
the priority of a ResourceSlice within a resource pool and another one to specify
the priority of resource pool relative to other resource pools published by the
same driver.

### API Changes

```go
type ResourceSliceSpec struct {
    ...

    // Priority specifies the order in which the allocator should
    // consider devices from ResourceSlices when allocating devices
    // for ResourceClaims. A higher value means the allocator will
    // consider devices from the ResourceSlice before devices from
    // ResourceSlices with a lower value. Devices from ResourceSlices
    // with the same value can be considered in any order.
    //
    // This field is optional and ResourceSlices without a value
    // specified is assumed to have a priority of zero. This means
    // that devices in a ResourceSlice with a negative value
    // will be considered after devices in ResourceSlices without a
    // value specified.
    //
    // +optional
    Priority *int64
}
```

```go
type ResourcePool struct {
    ...

    // Priority specifies the order in which the allocator should
    // consider devices from resource pools when allocating devices
    // for ResourceClaims. A higher value means the allocator will
    // consider devices from the resource pool before devices from
    // resource pools with a lower value. Devices from resource pools
    // with the same value can be considered in any order. The ordering
    // only applies to resource pools from the same driver. DRA does
    // not provide a way to specify the ordering between resource pools
    // from different drivers.
    //
    // This field is optional and resource pool without a value
    // specified is assumed to have a priority of zero. This means
    // that devices in a resource pool with a negative value
    // will be considered after devices in resource pools without a
    // value specified.
    //
    // For pools with the same priority, whether explicit or implicit,
    // pools without binding conditions are considered first.
    //
    // +optional
    Priority *int64
}
```

### Implementation Details

The priority fields will be used in the `GatherPools` function in
the [allocator package](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/dynamic-resource-allocation/structured/internal/experimental/pools_experimental.go#L58).
The function will be updated to sort the ResourceSlices within each
resource pool based on the `spec.priority` field and the resource pools
within each driver will be sorted based on the `spec.pool.priority` field.

### Validation

The `spec.pool.priority` field will be set on all ResourceSlices within the
same pool, which means that there is a chance that different ResourceSlices
might have different values for the priority. For a resource pool to be valid,
all ResourceSlices in the poolwith the same value for `spec.pool.generation`
must also have the same value for `spec.pool.priority`. If this is not true,
the resource pool will be considered invalid and devices will not be allocated
from the pool.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

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

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

<!--
Generated with:
go test -cover ./pkg/scheduler/framework/plugins/dynamicresources/...  ./staging/src/k8s.io/dynamic-resource-allocation/structured ./staging/src/k8s.io/dynamic-resource-allocation/structured/internal/experimental | sed -e 's/.*\(k8s.io[a-z/-]*\).*coverage: \(.*\) of statements/- `\1`: \2/' | sort
-->

Start of v1.36 development cycle (2026-01-23):

- `k8s.io/dynamic-resource-allocation/structured`: 33.3%
- `k8s.io/dynamic-resource-allocation/structured/internal/experimental`: 93.1%
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: 80.0%

##### Integration tests

Existing integration tests to control the worst case size of ResourceSlice objects
will be updated.

Impact on performance will be checked with the sched_perf tests for DRA

##### e2e tests

The e2e test suite will be updated to cover this feature.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from alpha
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

#### GA

- 3 examples of real-world usage
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

### Upgrade / Downgrade Strategy

Priorities will be ignored when downgrading to a release without support for it or when
disabling the feature. The effect is as if no priorities were set.

If drivers try to publish ResourceSlices with priorities in clusters where the
feature is not enabled, they will get an error. The recommended reaction is to
log and fail, which indicates to admins that they to update or reconfigure the
driver.

### Version Skew Strategy

During version skew where the apiserver supports the feature and the scheduler
doesn't, priorities can be set without encountering errors or
warnings, but they won't have any effect.

## Production Readiness Review Questionnaire


### Feature Enablement and Rollback


###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: DRAResourceSlicePriority
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler

###### Does enabling the feature change any default behavior?

No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The behavior of scheduling changes when it was in use.
Running applications are not affected.

###### What happens if we reenable the feature if it was previously rolled back?

Priorities will again be considered when allocating devices to new or restarting
workloads.

###### Are there any tests for feature enablement/disablement?

This will be covered through unit tests for the apiserver and scheduler.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It will not impact already existing workloads, since behavior will remain unchanged
for workloads that don't explicitly opt-in to the feature by setting the new fields
added by this feature. Also, the fields are only considered during scheduling, so even
when the fields are used they will not impact running workloads, unless it has to be
rescheduled.

###### What specific metrics should inform a rollback?

One indicator are unexpected restarts of the cluster control plane components (kube-scheduler, apiserver).

If the scheduler_pending_pods metric in the kube-scheduler suddenly increases of remains constant, it
can suggest that pods are no longer gettings scheduled which might be due to a problem with the DRA
scheduler plugin.

In all cases further analysis of logs and pod events is needed to determine whether errors are related to
this feature.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be done manually before transition to beta by bringing up a KinD cluster with kubeadm and
changing the feature gate for individual components.

Roundtripping of API types is covered by unit tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No

### Monitoring Requirements


###### How can an operator determine if the feature is in use by workloads?

There will be `ResourceSlices` in the cluster with at least one of:

* the `spec.priority` field set.
* the `spec.pool.priority` field set.

This means that there are drivers running in the cluster that are publishing
`ResourceSlices` using the feature.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] API .status
  - Condition name: 
  - Other field: `.status.allocation.devices.results.device` references a device from `ResourceSlice` or resource pool
    and it is the expected device based on availability of devices in the `ResourceSlice` and resource pool.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

As for normal pod scheduling of pods using ResourceClaims, there is no SLO for scheduling with partitionable devices.

This feature allows drivers to specify the order in which the scheduler considers devices, which can impact the
scheduling latency.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: resourceclaim_controller_resource_claims
  - Metric name: resourceclaim_controller_allocated_resource_claims
  - Metric name: workqueue with name="resource_claim"
  - Metric name: scheduler_pending_pods

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

This feature depends on the DRA structured parameters feature being enabled, and on DRA
drivers being deployed. There are no requirements beyond those already needed for DRA
structured parameters. Core DRA is locked to on in 1.36, but it can still be disabled
through emulation.

### Scalability


###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No. This KEP proposes extensions to an existing type, but not a new type itself.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, it adds new two new fields to the `ResourceSlice` type, so it will increase the
max size of `ResourceSlice` objects. But the two fields are both of type `int64`, so
adding these fields will not meaningfully impact the worst case size for `ResourceSlice`
objects. However, the integration tests computing the worst case size for `ResourceSlice`
objects will be updated.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No, as scheduling of pods using ResourceClaims are not covered by any SLO.

The time required to schedule a pod with at least one ResourceClaim can be impacted by this change.

It might increase due to:
* The `ResourceSlices` within a resource pool will have to be sorted if at least one
  `ResourceSlice` in the pool specifies the priority. This is additional work.
* The resource pools published by the same driver will have to be sorted if at least
  one resource pools specifies the priority. This is additional work.
* When the driver specifies the priority, the scheduler might have to do *more* work
  to find a set of devices that meets the requirements specified by a ResourceClaim.

It might decrease due to:
* When the driver specifies the priority, the scheduler might have to do *less* work
  to find a set of devices that meets the requirements specified by a ResourceClaim.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

### Troubleshooting

The troubleshooting section in https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters#troubleshooting
still applies.

###### How does this feature react if the API server and/or etcd is unavailable?

See https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters#how-does-this-feature-react-if-the-api-server-andor-etcd-is-unavailable

###### What are other known failure modes?

See https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/4381-dra-structured-parameters#what-are-other-known-failure-modes.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A since this feature does not come with an SLO.

## Implementation History

- 1.36: first KEP revision and implementation

## Drawbacks

It will have limited usefulness once scoring is implemented for DRA, but it is
a complicated feature and we don't have any design or timeline for when that
will happen.

## Alternatives

### Lexicograph ordering of ResourceSlices and resource pools based on names

Rather than introducing new fields for ordering, the allocator could sort
resource pools per driver and ResourceSlices within a resource pool based
on the names for resource pools and ResourceSlices. This could be implemented
without any changes to the API and the ResourceSlice controller could be
updated to manage the naming for driver authors.

However, this solution comes with some drawbacks:
* It will change the ordering for existing workloads. DRA would still allocate
  devices that meets the criteria specified by ResourceClaims, but users might
  see a different set of devices allocated than before and the performance might
  be impacted (could be better or worse).
* All resource pools and ResourceSlices will be impacted, since the allocator
  doesn't have a way of knowing whether a reordering is desirable or not. With
  specific fields the allocator will not make any changes to the ordering unless
  the driver requests it.
* It means that the names of the resources comes with semantics, which is subtle
  and not intuitive.
