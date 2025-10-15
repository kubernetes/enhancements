# KEP-4188: New Kubelet gRPC API with endpoint returning local Pods information

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [What kind of API to chose?](#what-kind-of-api-to-chose)
  - [Can we integrate with PodResource API?](#can-we-integrate-with-podresource-api)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Control Plane availability issue](#control-plane-availability-issue)
    - [Kubelet restarts issue](#kubelet-restarts-issue)
- [Design Details](#design-details)
  - [Pod State Selection](#pod-state-selection)
  - [Proposed API](#proposed-api)
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

This KEP proposes a new Kubelet gRPC API with an endpoint that returns local Pod information,
including the full PodSpec and status.
Serving this information from Kubelet on a node increases reliability and reduces load on
the Kubernetes API Server and network traffic outside the node. Connectivity issues between the node and
control plane should not impact workloads that depend on Pod state information.
The full PodSpec may contain sensitive information, so access to this API is secured by
restricting the UNIX socket to local admin users only.

## Motivation

Kubelet is responsible for running Health Checks (probes) and communicating the
results via Pod status. All that information is stored in cache and reported to
Kube-API. Right now pod's state information is tightly coupled with the Kubernetes API
Server. When a workload wants to know the actual state of Pods running on the
Node, it needs to fetch it from Kube-API. This causes some issues:

* Reliability - for various reasons Kube-API might not be available
  (connectivity issue, control plane updates)
* Scalability - adding new watchers to kube-API is a scalability concern. By
  exposing the endpoint that will serve the Pods state directly from the Kubelet
  cache we can use it on node workloads and avoid additional dependencies on the
  Kubernetes API Server.
* Flexibility - consumers may need more than just readiness, such as phase, IPs,
  resource usage, or labels/annotations, or the full pod spec. Supporting field selection
  enables lean, efficient queries, but the API can also provide the full pod object when needed.
  Because the full pod spec may contain secrets or other sensitive data, access must be tightly controlled.

| Impact | Description|
| ------- | ------------ |
| + | Reliability - for various reasons kube-API might not be available but this doesn’t mean that local workloads are not accessible and on node system workloads should have the most recent data about pod's state even when kube-API is unreachable. |
| + | Scalability - Reduce the load on kube-API by reducing the number of watchers and using Kubelet to fetch local Pods information. Fetching only Pods limited to one node is costly operation for kube-API. |
| + | Safety - Read-only API will not add security risks. |
| + | Reduce resource consumption by workload. Using Kube-API we can fetch objects like PodSpec or PodStatus, for some on-node workload this might be unnecessary, with this API workload can reduce the resource consumption by requesting only the fields they need. The API can also provide the full pod spec for advanced use cases, but this is restricted to authorized local users. |
| - | This API will add load to Kubelet (mitigation: API will be rate limited) |
| - | Kubelet does not support RBAC authorization for gRPC. (mitigation: This API is designed to be accessible only to local admin users. The unix socket will be secured with file permissions to restrict access to privileged users, and all exposed data will be carefully reviewed.) |

### Goals

- Expand Kubelet API Scope: Provide a general pod information API, equivalent to the apiserver.
- Support an optional fieldmask-based filtering for lean responses.
- Maintain strict local-only, read-only access via UNIX socket and file permissions.

The goal of this API is to expose comprehensive Pod information, including the full PodSpec and status, directly from the
source - Kubelet, independent of Control Plane availability. This would remove the need for node-local
components to request this node-local information from
the Kubernetes API Server.

The API should allow consumers to request only the fields they need, using protobuf fieldmasks, to enable efficient
and lean data transfer. For advanced use cases, the full pod spec can be returned, but only to authorized local admin
users due to the potential for sensitive information.

Kubelet already has a podresources endpoint
([2403-pod-resources-allocatable-resources](https://github.com/kubernetes/enhancements/tree/master/keps/sig-node/2403-pod-resources-allocatable-resources))
which returns information about Pod’s containers and Devices. This API does not
contain information about pod state.

Kubelet is responsible for computing Pod status and stores it in a local cache.
We want to create a new gRPC API that will expose pod information that is computed by
Kubelet and return the most recent data even when kube-API is not reachable.
This API is open for future modification if needed but exposed data via this API should be limited
to pod's information relevant for node-local consumers.

### Non-Goals

- Providing write access to pod information via this API.
- Exposing cluster-wide information beyond the scope of specific, node-local pods.

## Proposal

We are proposing to create a new Kubelet API that will return pod information, including the full PodSpec and status, not just readiness.

* The API will return data about both Static and Regular Pods.
* The API will not return partial data. If Kubelet does not know actual information
about workloads then gRPC FAILED_PRECONDITION (9) error code will be returned.
* The API should return the most recent information about Pods computed by the
  Kubelet even when those data were not reported or accepted by kube-API.
* The API will be read-only and accessible for on-node workloads via a
  unix socket, with access restricted to local admin users (e.g., root or a specific admin group) using file permissions.
* The API will be versioned.
* The API will support protobuf fieldmasks to allow clients to request only the fields they need from the pod information,
* but can also return the full pod spec for authorized users.

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

|   | PodResource API | Status API                                                                    |
| - | --------------- |-------------------------------------------------------------------------------|
| What information is returned with API | Container resources (devices, cpu_ids, memory, dynamic resources) | Pod spec and status (full or eventually filtered) for Static and Regular Pods |
| Who is the owner of the data returned by API | Kubelet, those data are not reported to Control Plane | static Pods - Kubelet, regular Pods - Kubelet, data are reported to kube-API  |
| What is the alternative for the API | reading the state files resource manager use | for regular Pods - fetching pod's data with kube-API, for static Pods - none  |
| Control Plane dependency | no | no                                                                            |
| Authorization | none, using unix socket to limit the clients to local workloads | none, using unix socket to limit the clients to local workloads               |
| Access | Read-Only | Read-Only                                                                     |
| Clients type / use cases | Device Plugins, CSI, Topology aware scheduling | CNI, monitoring and observability tools                                       |

The PodResource API includes an entirely unrelated set of information that is
unlikely to be of use to the set of clients that would benefit from
the full PodSpec and status. We propose creating a new API for this purpose.

### User Stories
* Some on-node system workloads want to reduce Control Plane dependency and
  introduce locality for Pod’s state to improve reliability and scalability.
  * checking Pod readiness for GKE CNI plugins to correctly program networking rules.
  * checking system Pods health for [node problem detector](https://github.com/kubernetes/node-problem-detector/issues/1112).
* Custom monitoring tools may want to have local information on Pods running on
  the same Node, faster and more reliable than fetching data from kube-API.
  * [Inspektor Gadget](https://github.com/inspektor-gadget/inspektor-gadget) and Kubescape's [node-agent](https://github.com/kubescape/node-agent) are using a Pod informer and some delays
    may be introduced between Pod state change and informer cache update.
* Some on-node system workloads interested in Pod state want to reduce
  resource consumption by requesting only the fields they need.

### Risks and Mitigations

This API is read-only, which removes a large class of risks.

| Risk                                                      | Impact        | Mitigation                                                                                                                                                                                                                                                                          |
| --------------------------------------------------------- | ------------- |-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Too many requests to the API impacting the Kubelet performances | High          | Rate limiting the API.                                                                                                                                                                                                                                                              |
| Misuse of the API  | High | This API is Read-only. We will expose only a subset of the pod's information relevant for node-local consumers. Exposed data does not contain sensitive information that could be used in a malicious way.                                                                          |
| Kubelet restart [issue](https://github.com/kubernetes/kubernetes/issues/100277) | High | This API should serve only complete information about workloads. If Kubelet is in the init phase and not all pod’s information is known, then the API should report an error.                                                                                                       |
| Unauthorized access to the API | Medium | This API is designed to be accessible only to privileged node workloads. Authorization will be provided by unix standard permissions to the socket file.                                                                                                                                       |
| Exposing the API to all workloads on the node | Medium | The unix socket will be secured with file permissions to restrict access to local admin users only. Exposed data via the API may contain sensitive information (e.g., environment variables, secrets references) present in the PodSpec, so access is tightly controlled.           |
| Kube-API is down or unreachable | Low | Kube-API availability should not impact this API. When the control plane is down or unreachable but Kubelet is working properly this API should return most recent data about local Pods that are computed by Kubelet even if those data were not reported or accepted by Kube-API. |

#### Control Plane availability issue

The Kubernetes Control Plane is the source of truth and guarantees system
integrity. However, already running Pods should not be dependent on Control
Plane availability or connectivity issues. When a Node is healthy but has
problems accessing the Kubernetes API, it should not result in node-local
components losing track of the health of other node-local components. For
example, networking components may unnecessarily route to unhealthy endpoints
if they're relying on stale information from the API Server.

The new API should return the full PodSpec and status computed by Kubelet even if
they are not yet reported back to kube-API but reflect the current Pod’s state
locally.

It is worth noting that a node may lose connectivity to the Control Plane even
if the Control Plane is healthy. In that case, the Node itself would eventually
be marked as unhealthy. We believe it is still acceptable for node-local
components to get node-local Pod information as this would not have
any impact on other Nodes in the cluster.

#### Kubelet restarts issue

There is a Kubelet restart
[issue](https://github.com/kubernetes/kubernetes/issues/100277) that could
impact this API. After restart Kubelet marks all Pods as NotReady and then
starts the init loop checking the actual pod's status. Kubelet should return
the pod's information only after the initial loop is finished
otherwise the API should return gRPC error `FAILED_PRECONDITION`. This
API needs to be reliable and returned data should reflect the actual state of
the Pods. We don't want this API to return partial data.

## Design Details

### Pod State Selection

Kubelet maintains multiple representations of pod state:
- The state derived from the Container Runtime Interface (CRI), reflecting the actual status of containers on the node.
- The state that is sent to the API server, which may be subject to additional processing or delays.
- The state received from the API server, reflecting the control plane's view.

It is important to define which state this API will serve. For the purposes of this API, the intent is to serve the most
up-to-date and accurate pod state as known locally by the Kubelet, typically the state based on the CRI and Kubelet's internal
reconciliation, rather than the potentially stale state received from the API server. This ensures consumers receive the
freshest possible information about pods running on the node.

### Proposed API

We propose to add new gRPC API `pods` in Kubelet, listening on a unix socket
at `/var/lib/kubelet/pods/kubelet.sock`. The endpoint will not be versioned. The
gRPC Service will expose 3 methods serving local Pods data:

```protobuf
import "google/protobuf/field_mask.proto";
import "k8s.io/api/core/v1/generated.proto";

service Pods {
    // ListPods returns a list of v1.Pod, filtered by field mask.
    rpc ListPods(PodListRequest) returns (PodListResponse) {}
    // WatchPods returns a stream of list of PodInfo, filtered by field mask.
    // Whenever a pod state changes, api returns the new list.
    rpc WatchPods(PodWatchRequest) returns (stream PodWatchResponse) {}
    // GetPod returns a PodInfo for given pod's UID, filtered by field mask.
    rpc GetPod(PodGetRequest) returns (PodGetResponse) {}
}

// PodListRequest allows specifying a field mask.
message PodListRequest {
    // Optional field mask in the gRPC metadata, to specify which pod fields to return.
}

// PodListResponse returns a list of v1.Pod.
message PodListResponse {
    repeated v1.Pod pods = 1;
}

// PodWatchRequest allows specifying a field mask.
message PodWatchRequest {
    // Optional field mask in the gRPC metadata, to specify which pod fields to return.
}

// PodWatchResponse returns a v1.Pod, as a stream.
message PodWatchResponse {
    v1.Pod pod = 1;
}

// PodGetRequest contains Pod UID and optional field mask.
message PodGetRequest {
    string podUID = 1;
    // Optional field mask in the gRPC metadata, to specify which pod fields to return.
}

// PodGetResponse returns a v1.Pod.
message PodGetResponse {
    v1.Pod pod = 1;
}

// ...other request/response messages as needed...
```

The v1.Pod message is the latest core Pod API based on the version of Kubelet, from the file `k8s.io/api/core/v1/generated.proto`.
Which means if there is a version skew between the API server and Kubelet, they may be serving different versions of the Pod API.

The use of `google.protobuf.FieldMask` allows clients to specify which fields of the v1.Pod message they are interested in,
enabling lean and efficient responses. Only authorized local admin users can use this API due to the potential sensitivity of the full PodSpec.

### Test Plan

##### Prerequisite testing updates

##### Unit tests

- Test parsing and validation of protobuf FieldMasks.
- Test filtering of Pod objects based on requested FieldMasks.

##### Integration tests

- End-to-end tests to verify that the API works as expected with different FieldMask combinations and client requests.
- Test API work when Kubelet restarts.
- Test API work when Control Plane is unreachable.
- Test API to work with kubelet Standalone mode.
- Scalability tests to ensure the API performs well under load.

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
  - Feature gate name: `PodInfoAPI` is a new feature gate to
  enable / disable PodInfo API.
  - Components depending on the feature gate: Kubelet

###### Does enabling the feature change any default behavior?

No, but the API will only be accessible to local admin users due to the sensitive nature of the full PodSpec.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, through feature gates and by restricting/removing access to the UNIX socket.

###### What happens if we reenable the feature if it was previously rolled back?

The API becomes available again. The API is stateless, so no recovery is needed, clients can just consume the data.

###### Are there any tests for feature enablement/disablement?

Yes, test will demonstrate that when the feature gate is disabled, the API returns the appropriate error code.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Kubelet may fail to start. The new API may cause the Kubelet to crash.

###### What specific metrics should inform a rollback?

`pod_info_endpoint_errors_get` - but only with feature gate `PodInfoAPI` enabled.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Look at the `pod_info_endpoint_requests_list`, `pod_info_endpoint_requests_get`, `pod_info_endpoint_requests_watch` metric exposed by the Kubelet.

###### How can someone using this feature know that it is working for their instance?

Access the API - it should return the Pod information.

Log entry indicating that API is ready to receive the traffic will be added.

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
  - Metric name:  `pod_info_endpoint_requests_total`, `pod_info_endpoint_requests_list` and `pod_info_endpoint_requests_get`.
  - Components exposing the metric: Kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

As part of this feature enhancement, per-API-endpoint resources metrics are being added; `pod_info_endpoint_requests_get`, `pod_info_endpoint_requests_list` add `pod_info_endpoint_errors_get`.

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

Yes, the API needs to create new objects based on data stored in Kubelet cache. This might increase CPU and memory consumption. Kubelet stores only local Pods data, and we can filter which information to get, so memory increase will not be significant.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A.

###### What are other known failure modes?

The Kubelet might be in init phase when client call the API. The API should return well-known error message. Unauthorized access attempts will be prevented by UNIX socket file permissions.

###### What steps should be taken if SLOs are not being met to determine the problem?

The API should be disabled using the feature gate.

## Implementation History

- 2023-09-05: KEP created
- 2025-09-30: KEP updated with new API and goals

## Drawbacks

## Alternatives

## Future work

This API is open to future extension but added information should be limited to pod's information relevant for node-local consumers. The use of field masks allows for future extensibility while maintaining efficient message sizes. The security model may be revisited if finer-grained access control is needed in the future.
