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
the DynamicResources Allocator attempts to allocate devices to ResourceClaims. Since the allocator
has a best-fit algorithm, the order in which devices are evaluated can affect both which devices
are allocated to a claim and how long it will take the allocator to find a set of devices that meet
all the criteria.

## Motivation

The DRA allocator is responsible for allocating devices to ResourceClaims. It does this by searching
through the available devices on a node, which are provided through the ResourceSlice API. It goes
through the devices and attempts to find a set of devices that satisfies all the criteria defined
in the ResourceClaim. As soon as it finds a set of devices that works, it stops the search and
returns the solution.

If the ResourceClaim asks for a GPU with at least 1Gi of memory and the ResourceSlices contains
devices with memory ranging from 1Gi to 80Gi, the order in which devices are evaluated are
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
  be evaluated.

### Non-Goals

* Provide a way to order the way the allocator evaluates devices across drivers.
* Full implementation of scoring.

## Proposal

### Risks and Mitigations

* The need for this feature is partly a result of not having scoring, and as a result,
  the scheduler only does a first-fit search for devices. With scoring, the order of the
  devices across resource pools and ResourceSlices doesn't matter since the allocator
  would evaluate all sets of devices to choose the one with the best fit. So this
  feature will be less relevant once we get scoring, but we don't
  currently have any timeline for when that might happen.

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
    // evaluate devices from ResourceSlices when allocating devices
    // for ResourceClaims. A higher value means the allocator will
    // evaluate devices from the ResourceSlice before devices from
    // ResourceSlices with a lower value. Devices from ResourceSlices
    // with the same value can be evaluated in any order.
    //
    // This field is optional and ResourceSlices without a value
    // specified is assumed to have a priority of zero. This means
    // that a negative value means that devices in the ResourceSlice
    // will be evaluated after devices in ResourceSlices without a
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
    // evaluate devices from resource pools when allocating devices
    // for ResourceClaims. A higher value means the allocator will
    // evaluate devices from the resource pool before devices from
    // resource pools with a lower value. Devices from resource pools
    // with the same value can be evaluated in any order. The ordering
    // only applies to resource pools from the same driver. DRA does
    // not provide a way to specify the ordering between resource pools
    // from different drivers.
    //
    // This field is optional and resource pool without a value
    // specified is assumed to have a priority of zero. This means
    // that a negative value means that devices in the resource pool
    // will be evaluated after devices in resource pools without a
    // value specified.
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

Start of v1.36 development cycle (2026-01-14):

- `k8s.io/dynamic-resource-allocation/structured`: 33.3%
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: 80.2%

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

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

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
