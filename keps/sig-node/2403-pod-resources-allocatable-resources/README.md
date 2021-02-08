# Extend kubelet pod resource assignment endpoint to return allocatable resources

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Node Feature Discovery](#node-feature-discovery)
    - [Topology aware scheduling](#topology-aware-scheduling)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Proposed API](#proposed-api)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha to Beta Graduation](#alpha-to-beta-graduation)
    - [Beta to G.A Graduation](#beta-to-ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Add a new endpoint](#add-a-new-endpoint)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements](https://github.com/kubernetes/enhancements/issues/2403)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [X] Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- ~~ [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io] ~~
- [X] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This document presents an addition to the kubelet pod resources endpoint (pod resources API) which allows third party consumers to learn about the
compute device allocation, thus, alongside the existing pod resources API endpoint, properly evaluate the node capacity.

## Motivation

### Goals

* Enable node monitoring agents to know the allocatable compute resources on a node, thus properly calculate the node compute resource utilization.

## Proposal

### User Stories

#### Node Feature Discovery

Enable the Node Feature Discovery to [expose hardware topology information](https://github.com/kubernetes-sigs/node-feature-discovery/issues/333).

#### Topology aware scheduling

This interface can be used to track down allocated resources with information about the NUMA topology of the worker node in general way.
This interface can be used to the available resources on the worker node. The kubelet is the best source of information because it manages concrete resources assignment. The information can then be used in NUMA aware scheduling.
Combining the information reported by the `List` API, which pertains the current allocation, with the information reported by the `GetAllocatableResources` API, monitoring agent can reliably report the compute device
utilization and availability.


### Risks and Mitigations

This API is read-only, which removes a large class of risks. The aspects that we consider below are as follows:
- What are the risks associated with the API service itself?
- What are the risks associated with the data itself?

| Risk                                                      | Impact        | Mitigation |
| --------------------------------------------------------- | ------------- | ---------- |
| Too many requests risk impacting the kubelet performances | High          | Implement rate limiting and or passive caching, follow best practices for gRPC resource management. |
| Improper access to the data | Low | Server is listening on a root owned unix socket. This can be limited with proper pod security policies. |


## Design Details

### Proposed API

We propose to extend the existing pod resources gRPC service of the Kubelet, listening on a unix socket at `/var/lib/kubelet/pod-resources/kubelet.sock`.

The GRPC Service will expose an additional endpoint:
- 'GetAllocatableResources`, which returns a single AllocatableResourcesResponse, enabling monitor applications to query for the allocatable set of resources available on the node.
This endpoint will return error if the corresponding feature gate is disabled.

The extended interface is shown in proto below:
```protobuf
// PodResources is a service provided by the kubelet that provides information about the
// node resources consumed by pods and containers on the node
service PodResources {
    rpc List(ListPodResourcesRequest) returns (ListPodResourcesResponse) {}
    rpc GetAllocatableResources(AllocatableResourcesRequest) returns (AllocatableResourcesResponse) {}
}

message AllocatableResourcesRequest {}

// AvailableResourcesResponses contains informations about all the devices known by the kubelet
message AllocatableResourcesResponse {
    repeated ContainerDevices devices = 1;
    repeated int64 cpu_ids = 2;
}

// ListPodResourcesRequest is the request made to the PodResources service
message ListPodResourcesRequest {}

// ListPodResourcesResponse is the response returned by List function
message ListPodResourcesResponse {
    repeated PodResources pod_resources = 1;
}

// PodResources contains information about the node resources assigned to a pod
message PodResources {
    string name = 1;
    string namespace = 2;
    repeated ContainerResources containers = 3;
}

// ContainerResources contains information about the resources assigned to a container
message ContainerResources {
    string name = 1;
    repeated ContainerDevices devices = 2;
    repeated int64 cpu_ids = 3;
}

// Topology describes hardware topology of the resource
message TopologyInfo {
	repeated NUMANode nodes = 1;
}

// NUMA representation of NUMA node
message NUMANode {
	int64 ID = 1;
}

// ContainerDevices contains information about the devices assigned to a container
message ContainerDevices {
    string resource_name = 1;
    repeated string device_ids = 2;
    TopologyInfo topology = 3;
}
```


### Test Plan

The implementation PR adds a suite of E2E tests which cover both the existing `List` endpoint already implemented in the podresources API and
the new proposed `GetAllocatableResources` API.

### Graduation Criteria

#### Alpha
- [X] Implement the new service API.
- [X] Ensure proper e2e node tests are in place.

#### Alpha to Beta Graduation
- [X] The new API is consumed by other public software components (e.g. NFD).
- [X] No major bugs reported in the previous cycle.

#### Beta to G.A Graduation
- [X] Allowing time for feedback (1 year).
- [X] Risks have been addressed.

### Upgrade / Downgrade Strategy

With gRPC the version is part of the service name.
Old versions and new versions should always be served and listened by the kubelet.

To a cluster admin upgrading to the newest API version, means upgrading Kubernetes to a newer version as well as upgrading the monitoring component.

To a vendor changes in the API should always be backwards compatible.

### Version Skew Strategy

Kubelet will always be backwards compatible, so going forward existing plugins are not expected to break.

## Production Readiness Review Questionnaire
### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`).
    - Feature gate name: `KubeletPodResourcesGetAllocatable`.
    - Components depending on the feature gate: kubelet, 3rd party consumers.

* **Does enabling the feature change any default behavior?** No
* **Can the feature be disabled once it has been enabled (i.e. can we rollback the enablement)?** Yes, through feature gates.
* **What happens if we reenable the feature if it was previously rolled back?** The API becomes available again. The API is stateless, so no recovery is needed, clients can just consume the data.
* **Are there any tests for feature enablement/disablement?** A e2e test will demonstrate that when the feature gate is disabled, the API returns the appropriate error code.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?** Kubelet may fail to start. The new API may report inconsistent data, or may cause the kubelet to crash.
* **What specific metrics should inform a rollback?** `pod_resources_endpoint_errors_get_allocatable` - but only with feature gate enabled. Otherwise the API will always return a known error, giving a false negative signal.
* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?** Not Applicable.
* **Is the rollout accompanied by any deprecations and/or removals of features,  APIs, fields of API types, flags, etc.?** No.

### Monitoring requirements
* **How can an operator determine if the feature is in use by workloads?**
  - Look at the `pod_resources_endpoint_requests_get_allocatable` metric exposed by the kubelet.
  - Clients are connected to the podresources unix socket, for example  bychecking which containers mount the podresources socket path.
* **What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?**
  - [X] Metrics
    - Metric name: `pod_resources_endpoint_requests_total`, `pod_resources_endpoint_requests_list`, `pod_resources_endpoint_requests_get_allocatable`, `pod_resources_endpoint_errors_list`, `pod_resources_endpoint_errors_get_allocatable`
    - Components exposing the metric: kubelet

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?** N/A.
* **Are there any missing metrics that would be useful to have to improve observability if this feature?** As part of this feature enhancement, per-API-endpoint resources metrics are being added; to observe this feature the `pod_resources_endpoint_requests_get_allocatable` metric should be used. We will also add error counting metrics to improve the observability of the API.


### Dependencies

* **Does this feature depend on any specific services running in the cluster?** Not applicable.

### Scalability

* **Will enabling / using this feature result in any new API calls?** No.
* **Will enabling / using this feature result in introducing new API types?** No.
* **Will enabling / using this feature result in any new calls to cloud provider?** No.
* **Will enabling / using this feature result in increasing size or count of the existing API objects?** No.
* **Will enabling / using this feature result in increasing time taken by any operations covered by [existing SLIs/SLOs][]?** No. Feature is out of existing any paths in kubelet.
* **Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?** DDOSing the API can lead to resource exhaustion. It is planned to be addressed as part of G.A.
Feature only collects data when requests comes in, data is then garbage collected. Data collected is proportional to the number of pods on the node.

### Troubleshooting

* **How does this feature react if the API server and/or etcd is unavailable?**: No effect.
* **What are other known failure modes?** feature gate disabled: the API will always return a well-known error. In normal operation, the API is expected to never return error and always return a valid response, because it utilizes internal kubelet data which is always available. Bugs may lead to the API to return unexpected errors, or to return inconsistent data. Consumers of the API should treat unexpected errors as bugs of this API.
* **What steps should be taken if SLOs are not being met to determine the problem?** N/A

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2021-02-02: KEP extracted from [previous iteration](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2043-pod-resource-concrete-assigments)
- 2021-02-04: KEP polished, added feature gate, clarified the graduation criteria.
- 2021-02-08: KEP updated adding per-specific-endpoint metrics to the podresources API and clarifying failure modes.

## Alternatives

### Add a new endpoint
* Pros:
  * No changes to existing APIs
* Cons:
  * Requires the client to consume two APIs
  * This work nicely fits in the boundaries and purpose of the podresources API
  * The changes proposed in this KEP are very low-risk and backward compatible
