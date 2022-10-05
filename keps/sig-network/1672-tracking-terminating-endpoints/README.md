# KEP-1672: Tracking Terminating Endpoints in the EndpointSlice API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
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
<!-- /toc -->

## Release Signoff Checklist

- [X] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] KEP approvers have approved the KEP status as `implementable`
- [X] Design details are appropriately documented
- [X] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] Graduation criteria is in place
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [X] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, terminating endpoints are considered "not ready" regardless of their actual readiness.
Before any work is done in improving how terminating endpoints are handled, there must be a way
to track whether an endpoint is terminating without having to watch the associated pods. This
KEP proposes a means to track the terminating state of an endpoint via the EndpointSlice API.
This would enable consumers of the API to make smarter decisions when it comes to handling
terminating endpoints (see KEP-1669 as an example).

## Motivation

### Goals

* Provide a mechanism to track whether an endpoint is terminating by only watching the EndpointSlice API.

### Non-Goals

* Consumption of the new API field is out of scope for this KEP but future KEPs will leverage
the work done here to improve graceful terminination of pods in certain scenarios (see issue [85643](https://github.com/kubernetes/kubernetes/issues/85643))

## Proposal

This KEP proposes to keep "terminating" pods in the set of endpoints in EndpointSlice with
additions to the API to indicate whether a given endpoint is terminating or not. If consumers
of the API (e.g. kube-proxy) are required to treat terminating endpoints differently, they
may do so by checking this condition.

The criteria for a ready endpoint (pod phase + readiness probe) will not change based on the
terminating state of pods, but consumers of the API may choose to prefer endpoints that are both ready and not terminating.

### User Stories (optional)

#### Story 1

A consumer of the EndpointSlice API (e.g. kube-proxy) may want to know which endpoints are
terminating without having to watch Pods directly for scalability reasons.

One example would be the IPVS proxier which should set the weight of an endpoint to 0
during termination and finally remove the real server when the endpoint is removed.
Without knowing when a pod is done terminating, the IPVS proxy makes a best-effort guess
at when the pod is terminated by looking at the connection tracking table.

### Notes/Constraints/Caveats (optional)

### Risks and Mitigations

Tracking the terminating state of endpoints poses some scalability concerns as each
terminating endpoint adds additional writes to the API. Today, a terminating pod
results in 1 write in Endpoints (removing the endpoint). With the proposed changes,
each terminating endpoint could result in at least 2 writes (ready -> terminating -> removed)
and possibly more depending on how many times readiness changes during termination.

## Design Details

To track whether an endpoint is terminating, a `terminating` and `serving` field would be added as part of
the `EndpointCondition` type in the EndpointSlice API.

```go
// EndpointConditions represents the current condition of an endpoint.
type EndpointConditions struct {
    // ready indicates that this endpoint is prepared to receive traffic,
    // according to whatever system is managing the endpoint. A nil value
    // indicates an unknown state. In most cases consumers should interpret this
    // unknown state as ready. For compatibility reasons, ready should never be
    // "true" for terminating endpoints.
    // +optional
    Ready *bool `json:"ready,omitempty" protobuf:"bytes,1,name=ready"`

    // serving is identical to ready except that it is set regardless of the
    // terminating state of endpoints. This condition should be set to true for
    // a ready endpoint that is terminating. If nil, consumers should defer to
    // the ready condition. This field can be enabled with the
    // EndpointSliceTerminatingCondition feature gate.
    // +optional
    Serving *bool `json:"serving,omitempty" protobuf:"bytes,2,name=serving"`

    // terminating indicates that this endpoint is terminating. A nil value
    // indicates an unknown state. Consumers should interpret this unknown state
    // to mean that the endpoint is not terminating. This field can be enabled
    // with the EndpointSliceTerminatingCondition feature gate.
    // +optional
    Terminating *bool `json:"terminating,omitempty" protobuf:"bytes,3,name=terminating"`
}
```

NOTE: A nil value for `Terminating` indicates that the endpoint is not terminating.

Updates to endpointslice controller:
* include pods with a deletion timestamp in endpointslice
* any pod with a deletion timestamp will have condition.terminating = true
* any terminating pod must have condition.ready = false.
* the new `serving` condition is set based on pod readiness regardless of terminating state.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `pkg/registry/discovery/endpointslice` (check tests that set the `EndpointSliceTerminatingCondition` feature gate)
    - API strategy unit tests to validate that terminating condition field cannot be set when feature gate is disabled.
    - API strategy unit tests to validate terminating condition is preserved if existing EndpointSlice has it set.
- `pkg/controller/endpointslice` (check tests that set the `EndpointSliceTerminatingCondition` feature gate)
    - endpointslice controller unit tests will validate pods with a deletion timestamp are included with condition.teriminating=true
    - endpointslice controller unit tests will validate that the ready condition can change for terminating endpoints
    - endpointslice controller unit tests will validate that terminating condition is not set when feature gate is disabled.

##### Integration tests

- `TestEndpointSliceTerminating`: https://github.com/kubernetes/kubernetes/blob/61b983a66b92142e454c655eb2add866c9b327b0/test/integration/endpointslice/endpointsliceterminating_test.go#L44

##### e2e tests

N/A - integation tests were sufficient

### Graduation Criteria

#### Alpha

* EndpointSlice API includes `Terminating` and `Serving` condition.
* `Terminating` and `Serving` condition can only be set if feature gate `EndpointSliceTerminatingCondition` is enabled.
* Unit tests in endpointslice controller and API validation/strategy.

#### Beta

* Integration API tests exercising the `terminating` and `serving` conditions.
* `EndpointSliceTerminatingCondition` is enabled by default.
* Consensus on scalability implications resulting from additional EndpointSlice writes with approval from sig-scalability.

#### GA

* E2E tests validating that terminating pods are properly reflected in EndpointSlice API.
* Ensure there are no performance/scalability regressions when enabling additional endpointslice writes for terminating endpoints.
  * This will be validated by running the existing scalability test suites where pods handle SIGTERM from kubelet before terminating.
* All necessary metrics are in place to provide adequate observability and monitoring for this feature.

### Upgrade / Downgrade Strategy

Since this is an addition to the EndpointSlice API, the upgrade/downgrade strategy will follow that
of the [EndpointSlice API work](/keps/sig-network/20190603-endpointslices/README.md).

### Version Skew Strategy

Since this is an addition to the EndpointSlice API, the version skew strategy will follow that
of the [EndpointSlice API work](/keps/sig-network/20190603-endpointslices/README.md).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: EndpointSliceTerminatingCondition
  - Components depending on the feature gate: kube-apiserver and kube-controller-manager

###### Does enabling the feature change any default behavior?

Yes, terminating endpoints are now included as part of EndpointSlice API. The `ready` condition of an endpoint will always be `false` to ensure consumers do not send traffic to terminating endpoints unless the new conditions `serving` and `terminating` are checked.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. On rollback, terminating endpoints will no longer be included in EndpointSlice and the `terminating` and `serving` conditions will not be set.

###### What happens if we reenable the feature if it was previously rolled back?

EndpointSlice will continue to have the `terminating` and `serving` condition set and terminating endpoints will be added to the endpointslice in it's next sync.

###### Are there any tests for feature enablement/disablement?

Yes, there will be strategy API unit tests validating if the new API field is allowed based on the feature gate.

These tests can be found here:
- https://github.com/kubernetes/kubernetes/blob/master/test/integration/endpointslice/endpointsliceterminating_test.go#L44
- https://github.com/kubernetes/kubernetes/blob/master/pkg/registry/discovery/endpointslice/strategy_test.go#L42-L137

### Rollout, Upgrade and Rollback Planning

###### How can a rollout fail? Can it impact already running workloads?

If there are consumers of EndpointSlice that do not check the `ready` condition, then they may unexpectedly start sending traffic to terminating endpoints.
It is assumed that almost all consumers of EndpointSlice check the `ready` condition prior to allowing traffic to a pod.

###### What specific metrics should inform a rollback?

EndpointSlice controller supports the following metrics that would be relevant for this feature:
- endpoint_slice_controller_endpoints_added_per_sync
- endpoint_slice_controller_endpoints_removed_per_sync
- endpoint_slice_controller_changes
- endpoint_slice_controller_endpointslices_changed_per_sync
- endpoint_slice_controller_syncs

The following metrics can be used to see if the introduction of this change resulted in a significantly
large number of traffic to the apiserver.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Rollback was manually validated using the following steps:

Create a kind cluster:
```
$ kind create cluster
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.25.0) 🖼
 ✓ Preparing nodes 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
Set kubectl context to "kind-kind"
You can now use your cluster with:

kubectl cluster-info --context kind-kind

Thanks for using kind! 😊
```

Check an EndpointSlice object to verify that the EndpointSliceTerminatingCondition feature gate is enabled.
Note that endpoints should have 3 conditions, `ready`, `serving` and `terminating`:
```
$ kubectl -n kube-system get endpointslice
NAME             ADDRESSTYPE   PORTS        ENDPOINTS               AGE
kube-dns-zp8h5   IPv4          9153,53,53   10.244.0.2,10.244.0.4   2m20s
$ kubectl -n kube-system get endpointslice kube-dns-zp8h5 -oyaml
addressType: IPv4
apiVersion: discovery.k8s.io/v1
endpoints:
- addresses:
  - 10.244.0.2
  conditions:
    ready: true
    serving: true
    terminating: false
  nodeName: kind-control-plane
  targetRef:
    kind: Pod
    name: coredns-565d847f94-lgsrp
    namespace: kube-system
    uid: cefa189a-66e0-4da3-8341-5c4e9f11407b
- addresses:
  - 10.244.0.4
  conditions:
    ready: true
    serving: true
    terminating: false
  nodeName: kind-control-plane
  targetRef:
    kind: Pod
    name: coredns-565d847f94-ptfln
    namespace: kube-system
    uid: d9003b65-2316-4d76-96f5-34d7570e6fcb
kind: EndpointSlice
metadata:
...
...
```

Turn off the `EndpointSliceTerminatingCondition` feature gate in `kube-apiserver` and `kube-controller-manager` (this is effectively
the state of the feature gate when it was Alpha).
```
$ docker ps
CONTAINER ID   IMAGE                  COMMAND                  CREATED         STATUS         PORTS                       NAMES
501fafe18dbd   kindest/node:v1.25.0   "/usr/local/bin/entr…"   4 minutes ago   Up 4 minutes   127.0.0.1:36795->6443/tcp   kind-control-plane
$ docker exec -ti kind-control-plane bash
$ vim /etc/kubernetes/manifests/kube-apiserver.yaml
# append --feature-gates=EndpointSliceTerminatingCondition=false to kube-apiserver flags
$ vim /etc/kubernetes/manifests/kube-controller-manager.yaml
# append --feature-gates=EndpointSliceTerminatingCondition=false to kube-controller-manager flags
```

Once `kube-apiserver` and `kube-controller-manager` restarts with the flag disabled, check that endpoints have the
`serving` and `terminating` conditions preserved and only dropped on the next update.
```
# preserved initially
$ kubectl -n kube-system get endpointslice kube-dns-zp8h5 -oyaml
addressType: IPv4
apiVersion: discovery.k8s.io/v1
endpoints:
- addresses:
  - 10.244.0.2
  conditions:
    ready: true
    serving: true
    terminating: false
  nodeName: kind-control-plane
  targetRef:
    kind: Pod
    name: coredns-565d847f94-lgsrp
    namespace: kube-system
    uid: cefa189a-66e0-4da3-8341-5c4e9f11407b
- addresses:
  - 10.244.0.4
  conditions:
    ready: true
    serving: true
    terminating: false
  nodeName: kind-control-plane
  targetRef:
    kind: Pod
    name: coredns-565d847f94-ptfln
    namespace: kube-system
    uid: d9003b65-2316-4d76-96f5-34d7570e6fcb
kind: EndpointSlice
metadata:
...

# trigger an update to endpointslice
$ kubectl -n kube-system delete po -l k8s-app=kube-dns
pod "coredns-565d847f94-lgsrp" deleted
pod "coredns-565d847f94-ptfln" deleted

# verify that serving/terminating conditions are now dropped
$ kubectl -n kube-system get endpointslice kube-dns-zp8h5 -oyaml
addressType: IPv4
apiVersion: discovery.k8s.io/v1
endpoints:
- addresses:
  - 10.244.0.6
  conditions:
    ready: true
  nodeName: kind-control-plane
  targetRef:
    kind: Pod
    name: coredns-565d847f94-jhtxk
    namespace: kube-system
    uid: b9a45145-fd3a-4f03-8243-59b0a0789bbf
- addresses:
  - 10.244.0.5
  conditions:
    ready: true
  nodeName: kind-control-plane
  targetRef:
    kind: Pod
    name: coredns-565d847f94-r6n9s
    namespace: kube-system
    uid: e20ce163-cf2b-4251-bcc8-352dcaf135c9
kind: EndpointSlice
metadata:
...
...
```

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The condition will always be set for terminating pods but consumers may choose to ignore them. It is up to consumers of the API to provide metrics
on how the new conditions are being used.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The existing SLI can be used to determine the health of this feature:

```
Latency of programming in-cluster load balancing mechanism (e.g. iptables), measured from when service spec or list of its Ready pods change to when it is reflected in load balancing mechanism, measured as 99th percentile over last 5 minutes aggregated across all programmers
```

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

It's hard to gauge an exact number here, because the existing SLI does not have a target SLO yet.
However, we should assume that the addition of the `serving` and `terminating` conditions do not
significantly impact the latency of kube-proxy syncing load balancer rules.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Adapting the existing endpoint slice controller metrics to also include endpoint conditions
as a label could be useful since a user can distinguish if the endpoint churn is happening due
to the addition of terminating endpoints or for another reason.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

None aside from the existing core Kubernetes components, specifically kube-apiserver and kube-controller-manager.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, there will be more writes to EndpointSlice when:
* a pod starts termination
* a pod's readiness changes during termination

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, it will increase the size of EndpointSlice by adding two boolean fields for each endpoint.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The networking programming latency SLO might be impacted due to additional writes to EndpointSlice.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

More writes to EndpointSlice could result in more resource usage from etcd disk IO and network bandwidth for all watchers.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

EndpointSlice conditions will get stale.

###### What are other known failure modes?

* Consumers of EndpointSlice that do not not check the `ready` condition may unexpectedly use terminating endpoints.

###### What steps should be taken if SLOs are not being met to determine the problem?

* Disable the feature gate
* Check if consumers of EndpointSlice are using the serving or termianting condition
* Check etcd disk usage

## Implementation History

- [x] 2020-04-23: KEP accepted as implementable for v1.19
- [x] 2020-07-01: initial PR with alpha imlementation merged for v1.20
- [x] 2020-05-12: KEP accepted as implementable for v1.22

## Drawbacks

There are some scalability draw backs as tracking terminating endpoints requires at least 1 additional write per endpoint.

