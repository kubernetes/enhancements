# KEP-4188: New Kubelet gRPC API with endpoint returning local Pods readiness information

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [User Stories](#user-stories)
- [Proposal](#proposal)
  - [What kind of API to chose?](#what-kind-of-api-to-chose)
  - [Can we integrate with PodResource API?](#can-we-integrate-with-podresource-api)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Control Plane availability issue](#control-plane-availability-issue)
    - [Kubelet restarts issue](#kubelet-restarts-issue)
- [Design Details](#design-details)
  - [Proposed API](#proposed-api)
      - [Unit tests](#unit-tests)
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
- [Future work](#future-work)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
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

Proposal to add a new Kubelet gRPC API with endpoint returning local Pods readiness information.
Serving that information by Kubelet within a Node will increase reliability and reduce load to the Kubernetes API Server and traffic outside the node. A connectivity issue between Node and Control Plane should not impact workloads which depend on Pods readiness statuses.

## Motivation

Kubelet is responsible for running Health Checks (probes) and communicating the
results via Pod status. All that information is stored in cache and reported to
Kube-API. Right now pod's readiness information is tightly coupled with the Kubernetes API
Server. When a workload wants to know the actual state of Pods running on the
Node, it needs to fetch it from Kube-API. This causes some issues:

* Reliability - for various reasons Kube-API might not be available
  (connectivity issue, control plane updates)
* Scalability - adding new watchers to kube-API is a scalability concern. By
  exposing the endpoint that will serve the Pods readiness status directly from the Kubelet
  cache we can use  it on node workloads and avoid additional dependencies on the
  Kubernetes API Server.

| Impact | Description|
| ------- | ------------ |
| + | Reliability - for various reasons kube-API might not be available but this doesn’t mean that local workloads are not accessible and on node system workloads should have the most recent data about pod's readiness even when kube-API is unreachable. |
| + | Scalability - Reduce the load on kube-API by reducing the number of watchers and using Kubelet to fetch local Pods readiness. Fetching only Pods limited to one node is costly operation for kube-API. |
| + | Safety - Read-only API will not add security risks. |
| + | Reduce resource consumption by workload. Using Kube-API we can fetch objects like PodSpec or PodStatus, for some on-node workload this might be unnecessary, with this API workload can reduce the resource consumption. |
| - | This API will add load to Kubelet (mitigation: API will be rate limited) |
| - | Kubelet does not support RBAC authorization for gRPC. (mitigation: This API is designed to be accessible for all workloads running on the node without the authorization. The unix socket will be used for the connection and all exposed data will be carefully reviewed.) |
| - | Limiting the scope of the API to the readiness information because we are not introducing the RBAC for this API. |

### Goals

The goal of this API is to expose Pod readiness information directly from the
source - Kubelet, independent of Control Plane availability. This would remove the need for node-local
components to request this node-local information from
the Kubernetes API Server.

Kubelet already has a podresources endpoint
([2403-pod-resources-allocatable-resources](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2403-pod-resources-allocatable-resources))
which returns information about Pod’s containers and Devices. This API does not
contain information about pod readiness status.

Kubelet is responsible for computing Pod status and stores it in a local cache.
We want to create a new gRPC API that will expose pod conditions that are computed by
Kubelet and return the most recent data even when kube-API is not reachable.
This API is open for future modification if needed but exposed data via this API should be limited
to pod's readiness information.

### Non-Goals

Exposing pods detailed data that are not related to pod's readiness.

### User Stories
* Some on node system workloads want to reduce Control Plane dependency and
  introduce locality for Pod’s readiness to improve reliability and scalability.
* Custom monitoring tools may want to have local visibility into the readiness
  of Pods running on the same Node.
* Some on node system workloads interested in Pod readiness want to reduce
  resource consumption.

## Proposal

We are proposing to create a new Kubelet API that will return pod's readiness information.

* The API will return data about both Static and Regular Pods.
* The API will not return partial data. If Kubelet does not know actual information
about workloads then gRPC FAILED_PRECONDITION (9) error code will be returned.
* The API should return the most recent information about Pods computed by the
  Kubelet even when those data were not reported or accepted by kube-API.
* The API will be read-only and accessible for on-node workloads (we will use a
  unix socket for the connection) with authorization limited to unix standard permissions.
* The API will be versioned.

### What kind of API to chose?

Our goal is to expose the API locally on the node and without Control Plane
dependency. That means that we want to avoid authorization which will be done via
kube-API. To fulfill this requirement we decided to use unix socket to access
the API. On top of unix socket we can create REST or gRPC API.

|   | gRPC | REST |
| - | ---- | -----|
| Support for unix socket | yes | yes |
| Can reuse existing API | possibly PodResource API. See [here](#can-we-integrate-with-podresource-api) | no, the existing REST API is served on ReadOnlyPort (shutdown on various cloud providers due to security reasons), AuthenticatedPort (authorization is done by kube-API). |
| Can reuse existing libraries (client-go, helpers in documentation) | no | yes|
| Support for streaming | yes | yes|
| Strongly typed | yes | no |
| Versions support | yes | yes |

Given the above table, we believe that a gRPC API is the best path forward.

### Can we integrate with PodResource API?

|   | PodResource API | Status API |
| - | --------------- | ------------- |
| What information is returned with API | Container resources (devices, cpu_ids, memory, dynamic resources) | for Static and Regular Pods pod readiness Conditions and UID  |
| Who is the owner of the data returned by API | Kubelet, those data are not reported to Control Plane | static Pods - Kubelet, regular Pods - Kubelet, data are reported to kube-API |
| What is the alternative for the API | reading the state files resource manager use | for regular Pods - fetching pod's data with kube-API, for static Pods - none |
| Control Plane dependency | no | no |
| Authorization | none, using unix socket to limit the clients to local workloads | none, using unix socket to limit the clients to local workloads |
| Access | Read-Only | Read-Only|
| Clients type / use cases | Device Plugins, CSI, Topology aware scheduling | CNI, monitoring and observability tools |

The PodResource API includes an entirely unrelated set of information that is
unlikely to be of use to the set of clients that would benefit from
understanding Pod readiness. We propose creating a new API for this purpose.

### Risks and Mitigations

This API is read-only, which removes a large class of risks.

| Risk                                                      | Impact        | Mitigation |
| --------------------------------------------------------- | ------------- | ---------- |
| Too many requests to the API impacting the Kubelet performances | High          | Rate limiting the API. |
| Misuse of the API  | High |  This API is Read-only. We will expose only a small portion of the pod's information related to the pod readiness. Exposed data does not contain sensitive information that could be used in a malicious way. |
| Kubelet restart [issue](https://github.com/kubernetes/kubernetes/issues/100277) | High | This API should serve only complete information about workloads readiness. If Kubelet is in the init phase and not all pod’s readiness information is known, then the API should report the error. |
| Unauthorized access to the API | Medium | This API is designed to be accessed by all on-node workloads. Authorization will be provided by unix standard permissions to the socket file. |
| Exposing the API to all workloads on the node | Medium | Exposed data via the API is limited to readiness information only. |
| Kube-API is down or unreachable | Low | Kube-API availability should not impact this API. When the control plane is down or unreachable but Kubelet is working properly this API should return most recent data about local Pods readiness that are computed by Kubelet even if those data were not reported or accepted by Kube-API. |

#### Control Plane availability issue

The Kubernetes Control Plane is the source of truth and guarantees system
integrity. However, already running Pods should not be dependent on Control
Plane availability or connectivity issues. When a Node is healthy but has
problems accessing the Kubernetes API, it should not result in node-local
components losing track of the health of other node-local components. For
example, networking components may unnecessarily route to unhealthy endpoints
if they're relying on stale information from the API Server.

The new API should return Pods readiness that is computed by Kubelet even if
they are not reported back to kube-API but reflect the current Pod’s state
locally.

It is worth noting that a node may lose connectivity to the Control Plane even
if the Control Plane is healthy. In that case, the Node itself would eventually
be marked as unhealthy. We believe it is still acceptable for node-local
components to get node-local Pod readiness information as this would not have
any impact on other Nodes in the cluster.

#### Kubelet restarts issue

There is a Kubelet restart
[issue](https://github.com/kubernetes/kubernetes/issues/100277) that could
impact this API. After restart Kubelet marks all Pods as NotReady and then
starts the init loop checking the actual pod's status. Kubelet should return
the pod's readiness information only after the initial loop is finished
otherwise the API should return gRPC error `FAILED_PRECONDITION`. This
API needs to be reliable and returned data should reflect the actual state of
the Pods. We don't want this API to return partial data.

## Design Details

### Proposed API

We propose to add new gPRC API `status` in Kubelet, listening on a unix socket
at `/var/lib/Kubelet/status/Kubelet.sock`. The endpoint will be versioned. The
gRPC Service will expose 3 methods serving local Pods statuses data:

```protobuf
service PodStatus {
    // ListPodStatus returns a of List of PodStatus
    rpc ListPodStatus(PodStatusListRequest) returns (PodStatusListResponse) {}
    // WatchPodStatus returns a stream of List of PodStatus
    // Whenever a pod state change api returns the new list
    rpc WatchPodStatus(PodStatusWatchRequest) returns (stream PodStatusWatchResponse) {}
    // GetPodStatus returns a PodStatus for given pod's UID
    rpc GetPodStatus(PodStatusGetRequest) returns (PodStatusGetResponse) {}
}

// PodCondition aligns with v1.PodCondition.
message PodCondition {
    PodConditionType Type = 1;
    ConditionStatus Status = 2;
    Timestamp LastProbeTime = 3;
    Timestamp LastTransitionTime = 4;
    string Reason = 5;
    string Message = 6;
}

// PodConditionType aligns with v1.PodConditionType
enum PodConditionType {
    ContainersReady = 0;
    Initialized = 1;
    Ready = 2;
    PodScheduled = 3;
    DisruptionTarget = 4;
}

// ConditionStatus aligns with v1.ConditionStatus
enum ConditionStatus {
    True = 0;
    False = 1;
    Unknown = 2;
}

// PodStatus returns a Pod details and list of status Conditions with deletion info.
message PodStatus {
    string podUID = 1;
    string podNamespace = 2;
    string podName = 3;
    bool static = 4;
    repeated PodCondition conditions = 5;
    Timestamp DeletionTimestamp = 3;
}

// PodStatusResponse returns a stream of List of PodStatus.
// Whenever a Pod state changes it will return the new list.
message PodStatusListResponse {
    // PodStatus includes the Readiness information of Pods.
    // In the future it may be extended to include additional information.
    repeated PodStatus Pods = 1;
}

// PodStatusGetRequest contains Pods UID
message PodStatusGetRequest {
    string podUID = 1;
}
```

##### Unit tests


##### e2e tests

  - Feature gate enable/disable tests.
  - Test API work when Kubelet restarts.
  - Test API work when Control Plane is unreachable.
  - Test API to work with kubelet Standalone mode.

### Graduation Criteria

#### Alpha

- [ ] Feature implemented behind a feature flag.
- [ ] e2e tests completed and enabled.
- [ ] sig-auth input on proposed lack of authorization for this API.

#### Beta
- [ ] Fix Kubelet restart [issue](https://github.com/kubernetes/kubernetes/issues/100277)

#### GA

- [ ] No major bugs related to usage of API in 3 months cycle.

### Upgrade / Downgrade Strategy

With gRPC the version is part of the service name.
Old versions and new versions should always be served and listened by the Kubelet.

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire
### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodReadinessAPI` is a new feature gate to
  enable / disable PodReadiness API.
  - Components depending on the feature gate: Kubelet

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, through feature gates.

###### What happens if we reenable the feature if it was previously rolled back?

The API becomes available again. The API is stateless, so no recovery is needed, clients can just consume the data.

###### Are there any tests for feature enablement/disablement?

yes, test will demonstrate that when the feature gate is disabled, the API returns the appropriate error code.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Kubelet may fail to start. The new API may cause the Kubelet to crash.

###### What specific metrics should inform a rollback?

`pod_readiness _endpoint_errors_get` - but only with feature gate `PodReadinessAPI` enabled.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?


###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Look at the `pod_readiness_endpoint_requests_list`, `pod_readiness_endpoint_requests_get`, `pod_readiness_endpoint_requests_watch` metric exposed by the Kubelet.

###### How can someone using this feature know that it is working for their instance?
Log entry indicating that API is ready to received the traffic will be added.

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name:  `pod_status_endpoint_requests_total`, `pod_status_endpoint_requests_list` and `pod_status_endpoint_requests_get`.
  - Components exposing the metric: Kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

As part of this feature enhancement, per-API-endpoint resources metrics are being added; `pod_status_endpoint_requests_get`, `pod_status_endpoint_requests_list` add `pod_status_endpoint_errors_get`.

### Dependencies

N/A

###### Does this feature depend on any specific services running in the cluster?

N/A

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

Yes.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Yes, the API needs to create new objects based on data stored in Kubelet cache. This might increase CPU and memory consumption. Kubelet stores only local Pods data and we are coping only small amount of pod.Status object, so memory increase will not be significant.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A.

###### What are other known failure modes?

The Kubelet might be in init phase when client call the API. The API should return well-known error message.

###### What steps should be taken if SLOs are not being met to determine the problem?

The API should be disabled using the feature gate.

## Implementation History

- 2023-09-05: KEP created

## Drawbacks

## Alternatives

## Future work

This API is open to future extension but added information should be limited to pod's readiness information.
