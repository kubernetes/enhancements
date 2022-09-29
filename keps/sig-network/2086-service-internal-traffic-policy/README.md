# KEP-2086: Service Internal Traffic Policy

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
    - [API Enablement Rollback](#api-enablement-rollback)
    - [Proxy Enablement Rollback](#proxy-enablement-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [EndpointSlice Subsetting](#endpointslice-subsetting)
  - [Bool Field For Node Local](#bool-field-for-node-local)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [ ] Production readiness review approved
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

Add a new field `spec.internalTrafficPolicy` to Service that allows node-local and topology-aware routing for Service traffic.

## Motivation

Internal traffic routed to a Service has always been randomly distributed to all endpoints.
This KEP proposes a new API in Service to address use-cases such as node-local and topology aware routing
for internal Service traffic.

### Goals

* Allow internal Service traffic to be routed to node-local or topology-aware endpoints.
* Default behavior for internal Service traffic should not change.

### Non-Goals

* Topology aware routing for zone/region topologies -- while this field enables this feature, this KEP only covers node-local routing.
  See the Topology Aware Hints KEP for more details.

## Proposal

Introduce a new field in Service `spec.internalTrafficPolicy`. The field will have 2 codified values:
1. Cluster (default): route to all cluster-wide endpoints (or use topology aware subsetting if enabled).
2. Local: only route to node-local endpoints, drop otherwise.

A feature gate `ServiceInternalTrafficPolicy` will also be introduced this feature.
The `internalTrafficPolicy` field cannot be set on Service during the alpha stage unless the feature gate is enabled.
During the Beta stage, the feature gate will be on by default.

The `internalTrafficPolicy` field will not apply for headless Services or Services of type `ExternalName`.

### User Stories (Optional)

#### Story 1

As a platform owner, I want to create a Service that always directs traffic to a logging daemon or metrics agent on the same node.
Traffic should never bounce to a daemon on another node since the logs would then report an incorrect log source.

### Risks and Mitigations

* When the `Local` policy is set, it is the user's responsibility to ensure node-local endpoints are ready, otherwise traffic will be dropped.

## Design Details

Proposed addition to core v1 API:
```go
type ServiceInternalTrafficPolicyType string

const (
	ServiceTrafficPolicyTypeCluster     ServiceTrafficPolicyType = "Cluster"
	ServiceTrafficPolicyTypeLocal       ServiceTrafficPolicyType = "Local"
)

// ServiceSpec describes the attributes that a user creates on a service.
type ServiceSpec struct {
	...
	...

	// InternalTrafficPolicy specifies if the cluster internal traffic
	// should be routed to all endpoints or node-local endpoints only.
	// "Cluster" routes internal traffic to a Service to all endpoints.
	// "Local" routes traffic to node-local endpoints only, traffic is
	// dropped if no node-local endpoints are ready.
	// The default value is "Cluster".
	// +featureGate=ServiceInternalTrafficPolicy
	// +optional
	InternalTrafficPolicy *ServiceInternalTrafficPolicyType `json:"internalTrafficPolicy,omitempty" protobuf:"bytes,22,opt,name=internalTrafficPolicy"`
}
```

This field will be independent from externalTrafficPolicy. In other words, internalTrafficPolicy only applies to traffic originating from internal sources.

Proposed changes to kube-proxy:
* when `internalTrafficPolicy=Cluster`, default to existing behavior today.
* when `internalTrafficPolicy=Local`, route to endpoints in EndpointSlice that maches the local node's topology, drop traffic if none exist.

Overlap with topology-aware routing:

| ExternalTrafficPolicy | InternalTrafficPolicy | Topology | External Result | Internal Result |
| - | - | - | - | - |
| - | - | Auto | Topology | Topology |
| Local | - | Auto | Local | Topology |
| Local | Local | Auto | Local | Local |

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `pkg/registry/core/service`: `v1.22` - `API strategy tests for feature enablement and upgrade safety (dropping disabled fields)`
- `pkg/apis/core/v1`: `v1.22` - `API defauting tests`
- `pkg/proxy/iptables`: `v1.22` - `iptables rules tests + feature enablement`
- `pkg/proxy/ipvs`: `v1.22` - `ipvs rules tests + feature enablement`
- `pkg/proxy`: `v1.22` - `generic kube-proxy Service tests`

NOTE: search [ServiceInternalTrafficPolicy](https://github.com/kubernetes/kubernetes/search?q=ServiceInternalTrafficPolicy) in the Kubernetes repo for references to existing tests.

##### Integration tests

- `Test_ExternalNameServiceStopsDefaultingInternalTrafficPolicy`: https://github.com/kubernetes/kubernetes/blob/61b983a66b92142e454c655eb2add866c9b327b0/test/integration/service/service_test.go#L34
- `Test_ExternalNameServiceDropsInternalTrafficPolicy`: https://github.com/kubernetes/kubernetes/blob/61b983a66b92142e454c655eb2add866c9b327b0/test/integration/service/service_test.go#L78
- `Test_ConvertingToExternalNameServiceDropsInternalTrafficPolicy`: https://github.com/kubernetes/kubernetes/blob/61b983a66b92142e454c655eb2add866c9b327b0/test/integration/service/service_test.go#L125

##### e2e tests

- `should respect internalTrafficPolicy=Local Pod to Pod`: https://github.com/kubernetes/kubernetes/blob/4bc1398c0834a63370952702eef24d5e74c736f6/test/e2e/network/service.go#L2520
- `should respect internalTrafficPolicy=Local Pod (hostNetwork: true) to Pod`: https://github.com/kubernetes/kubernetes/blob/4bc1398c0834a63370952702eef24d5e74c736f6/test/e2e/network/service.go#L2598
- `should respect internalTrafficPolicy=Local Pod and Node, to Pod (hostNetwork: true)`: https://github.com/kubernetes/kubernetes/blob/4bc1398c0834a63370952702eef24d5e74c736f6/test/e2e/network/service.go#L2678

### Graduation Criteria

Alpha:
* feature gate `ServiceInternalTrafficPolicy` _must_ be enabled for apiserver to accept values for `spec.internalTrafficPolicy`. Otherwise field is dropped.
* kube-proxy handles traffic routing for 2 initial internal traffic policies `Cluster`, and `Local`.
* Unit tests as defined in "Test Plan" section above. E2E tests are nice to have but not required for Alpha.

Beta:
* integration tests exercising API behavior for `spec.internalTrafficPolicy` field of Service.
* e2e tests exercising kube-proxy routing when `internalTrafficPolicy` is `Local`.
* feature gate `ServiceInternalTrafficPolicy` is enabled by default.
* consensus on how internalTrafficPolicy overlaps with topology-aware routing.

GA:
* metrics for total number of Services that have no endpoints (kubeproxy/sync_proxy_rules_no_endpoints_total) with additional labels for internal/external and local/cluster policies.
* Fix a bug where internalTrafficPolicy=Local would force externalTrafficPolicy=Local (https://github.com/kubernetes/kubernetes/pull/106497).
* Sufficient integration/e2e tests (many were already added for Beta, but we'll want to revisit tests based on changes that landed during Beta).

### Upgrade / Downgrade Strategy

* The `internalTrafficPolicy` field will be off by default during the alpha stage but can handle any existing Services that has the field already set.
This ensures n-1 apiservers can handle the new field on downgrade.
* On upgrade, if the feature gate is enabled there should be no changes in the behavior since the default value for `internalTrafficPolicy` is `Cluster`.

### Version Skew Strategy

Since this feature will be alpha for at least 1 release, an n-1 kube-proxy should handle enablement of this feature if a new apiserver enabled it.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: `ServiceInternalTrafficPolicy`
    - Components depending on the feature gate: kube-apiserver, kube-proxy

* **Does enabling the feature change any default behavior?**

No, enabling the feature does not change any default behavior since the default value of `internalTrafficPolicy` is `Cluster`.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

Yes, the feature gate can be disabled, but Service resource that have set the new field will persist that field unless unset by the user.

* **What happens if we reenable the feature if it was previously rolled back?**

New Services should be able to set the `internalTrafficPolicy` field. Existing Services that have the field set will begin to apply the policy again.

* **Are there any tests for feature enablement/disablement?**

There will be unit tests to verify that apiserver will drop the field when the `ServiceInternalTrafficPolicy` feature gate is disabled.

Tests added so far:
* https://github.com/kubernetes/kubernetes/blob/0038bcfad495a0458372867a77c8ca646f361c40pkg/registry/core/service/strategy_test.go#L368-L390
* https://github.com/kubernetes/kubernetes/blob/0038bcfad495a0458372867a77c8ca646f361c40/pkg/registry/core/service/storage/storage_test.go#L682-L684
* https://github.com/kubernetes/kubernetes/blob/0038bcfad495a0458372867a77c8ca646f361c40/test/integration/service/service_test.go

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

Rollout should have minimal impact because the default value of `internalTrafficPolicy` is `Cluster`, which is the default behavior today.

* **What specific metrics should inform a rollback?**

Metrics representing Services being black-holed will be added. This metric can inform rollback.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

#### API Enablement Rollback

API Rollback was manually validated using the following steps:

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

Check that Services set `spec.internalTrafficPolicy`:
```
$ kubectl -n kube-system get svc kube-dns -o yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  creationTimestamp: "2022-09-27T14:03:46Z"
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: CoreDNS
  name: kube-dns
  namespace: kube-system
  resourceVersion: "219"
  uid: 9b455b45-ae9f-43d1-98ca-f275b805ab95
spec:
  clusterIP: 10.96.0.10
  clusterIPs:
  - 10.96.0.10
  internalTrafficPolicy: Cluster
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - name: dns
    port: 53
    protocol: UDP
    targetPort: 53
  - name: dns-tcp
    port: 53
    protocol: TCP
    targetPort: 53
  - name: metrics
    port: 9153
    protocol: TCP
    targetPort: 9153
  selector:
    k8s-app: kube-dns
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}
```

Turn off the `ServiceInternalTrafficPolicy` feature gate in `kube-apiserver`:
```
$ docker exec -ti kind-control-plane bash
$ vim /etc/kubernetes/manifests/kube-apiserver.yaml
# append --feature-gates=ServiceInternalTrafficPolicy=false to kube-apiserver flags
```

Validate that the Service still preserves the field. This is expected for backwards compatibility reasons:
```
$ kubectl -n kube-system get svc kube-dns -o yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    prometheus.io/port: "9153"
    prometheus.io/scrape: "true"
  creationTimestamp: "2022-09-27T14:03:46Z"
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    kubernetes.io/name: CoreDNS
  name: kube-dns
  namespace: kube-system
  resourceVersion: "219"
  uid: 9b455b45-ae9f-43d1-98ca-f275b805ab95
spec:
  clusterIP: 10.96.0.10
  clusterIPs:
  - 10.96.0.10
  internalTrafficPolicy: Cluster
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - name: dns
    port: 53
    protocol: UDP
    targetPort: 53
  - name: dns-tcp
    port: 53
    protocol: TCP
    targetPort: 53
  - name: metrics
    port: 9153
    protocol: TCP
    targetPort: 9153
  selector:
    k8s-app: kube-dns
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}
```

Validate that new Services do not have the field anymore:
```
$ cat service.yaml
---
apiVersion: v1
kind: Service
metadata:
  name: nginx
  labels:
    app: nginx
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    protocol: TCP
$ kubectl apply -f service.yaml
service/nginx created
$ kubectl get svc nginx -o yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"v1","kind":"Service","metadata":{"annotations":{},"labels":{"app":"nginx"},"name":"nginx","namespace":"default"},"spec":{"ports":[{"port":80,"protocol":"TCP"}],"selector":{"app":"nginx"}}}
  creationTimestamp: "2022-09-27T14:10:51Z"
  labels:
    app: nginx
  name: nginx
  namespace: default
  resourceVersion: "867"
  uid: e1bf394a-3759-4534-ac44-8cb8e44c1971
spec:
  clusterIP: 10.96.55.182
  clusterIPs:
  - 10.96.55.182
  ipFamilies:
  - IPv4
  ipFamilyPolicy: SingleStack
  ports:
  - port: 80
    protocol: TCP
    targetPort: 80
  selector:
    app: nginx
  sessionAffinity: None
  type: ClusterIP
status:
  loadBalancer: {}
```

#### Proxy Enablement Rollback

Rolling back kube-proxy behavior was tested manually with the following steps:

Create a Kind cluster with 2 worker nodes:
```
$ cat kind.yaml
# three node (two workers) cluster config
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker

$ kind create cluster --config kind.yaml
Creating cluster "kind" ...
 ✓ Ensuring node image (kindest/node:v1.25.0) 🖼
 ✓ Preparing nodes 📦 📦 📦
 ✓ Writing configuration 📜
 ✓ Starting control-plane 🕹️
 ✓ Installing CNI 🔌
 ✓ Installing StorageClass 💾
 ✓ Joining worker nodes 🚜
Set kubectl context to "kind-kind"
You can now use your cluster with:

kubectl cluster-info --context kind-kind
```

Create a webserver Deployment and Service with `internalTrafficPolicy=Local`:
```
$ cat agnhost-webserver.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agnhost-server
  labels:
    app: agnhost-server
spec:
  replicas: 3
  selector:
    matchLabels:
      app: agnhost-server
  template:
    metadata:
      labels:
        app: agnhost-server
    spec:
      containers:
      - name: agnhost
        image: registry.k8s.io/e2e-test-images/agnhost:2.40
        args:
        - serve-hostname
        - --port=80
        ports:
        - containerPort: 80

---
apiVersion: v1
kind: Service
metadata:
  name: agnhost-server
  labels:
    app: agnhost-server
spec:
  internalTrafficPolicy: Local
  selector:
    app: agnhost-server
  ports:
  - port: 80
    protocol: TCP
```

Exec into one node and verify that kube-proxy only proxies to the local nodes.
```kind-worker2
$ kubectl get svc agnhost-server
NAME             TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
agnhost-server   ClusterIP   10.96.226.141   <none>        80/TCP    7m56s
$ kubectl get po -o wide
NAME                              READY   STATUS    RESTARTS   AGE     IP           NODE           NOMINATED NODE   READINESS GATES
agnhost-server-7d4db667fc-h5m5p   1/1     Running   0          6m2s    10.244.2.3   kind-worker    <none>           <none>
agnhost-server-7d4db667fc-nrdg4   1/1     Running   0          5m59s   10.244.2.4   kind-worker    <none>           <none>
agnhost-server-7d4db667fc-tpbpk   1/1     Running   0          6m      10.244.1.4   kind-worker2   <none>           <none>

$ docker exec -ti kind-worker2 bash
root@kind-worker2:/# iptables-save
...
...
...
-A KUBE-SVL-VPG43MSD43N5Z7KJ ! -s 10.244.0.0/16 -d 10.96.226.141/32 -p tcp -m comment --comment "default/agnhost-server cluster IP" -m tcp --dport 80 -j KUBE-MARK-MASQ
-A KUBE-SVL-VPG43MSD43N5Z7KJ -m comment --comment "default/agnhost-server -> 10.244.1.4:80" -j KUBE-SEP-MNGWCV7VA4JSYYKU
COMMIT
# Completed on Tue Sep 27 14:50:07 2022
```

Turn off the `ServiceInternalTrafficPolicy` feature gate in kube-proxy by editing the kube-proxy ConfigMap and restarting kube-proxy:
```
$ kubectl -n kube-system edit cm kube-proxy
# set featureGates["ServiceInternalTrafficPolicy"]=false in config.conf
configmap/kube-proxy edited

$ kubectl -n kube-system delete po -l k8s-app=kube-proxy
pod "kube-proxy-d6nf2" deleted
pod "kube-proxy-f89g4" deleted
pod "kube-proxy-lrk9l" deleted
```

Exec into the same node as before and check that kube-proxy now proxies to all endpoints even though
the Service has `internalTrafficPolicy` set to `Local`:
```
$ docker exec -ti kind-worker2 bash
root@kind-worker2:/# iptables-save
...
...
...
-A KUBE-SVC-VPG43MSD43N5Z7KJ ! -s 10.244.0.0/16 -d 10.96.226.141/32 -p tcp -m comment --comment "default/agnhost-server cluster IP" -m tcp --dport 80 -j KUBE-MARK-MASQ
-A KUBE-SVC-VPG43MSD43N5Z7KJ -m comment --comment "default/agnhost-server -> 10.244.1.4:80" -m statistic --mode random --probability 0.33333333349 -j KUBE-SEP-MNGWCV7VA4JSYYKU
-A KUBE-SVC-VPG43MSD43N5Z7KJ -m comment --comment "default/agnhost-server -> 10.244.2.3:80" -m statistic --mode random --probability 0.50000000000 -j KUBE-SEP-EMZSNZ2TWKTA6UM4
-A KUBE-SVC-VPG43MSD43N5Z7KJ -m comment --comment "default/agnhost-server -> 10.244.2.4:80" -j KUBE-SEP-KTGIEIH27TIG3JO7
COMMIT
# Completed on Tue Sep 27 14:55:04 2022
```

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

* Check Service to see if `internalTrafficPolicy` is set to `Local`.
* A per-node "blackhole" metric will be added to kube-proxy which represent Services that are being intentionally dropped (internalTrafficPolicy=Local and no endpoints). The metric will be named `kubeproxy/sync_proxy_rules_no_endpoints_total` (subject to rename).

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**

They can check the `kubeproxy/sync_proxy_rules_no_endpoints_total` metric when internalTrafficPolicy=Local and there are no endpoints.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

This will depend on Service topology and whether `internalTrafficPolicy=Local` is being used.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

A new metric will be added to represent Services that are being "blackholed" (internalTrafficPolicy=Local and no endpoints).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

No.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

No, since this is a user-defined field in Service. No extra calls will be required
from EndpointSlice as well since topology information is already stored there.

* **Will enabling / using this feature result in introducing new API types?**

No API types are introduced, only a new field in Service.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

No

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**

This feature will (negligibly) increase the size of Service by adding a single field.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

This feature may slightly increase kube-proxy's sync time for iptable / IPVS rules,
since node topology must be calculated, but this is likely negligible given we
already have many checks like this for `externalTrafficPolicy: Local`.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

Any increase in CPU usage by kube-proxy to calculate node-local topology will likely
be offset by reduced iptable rules it needs to sync when using the `Local`
traffic policy.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

Services will not be able to update their internal traffic policy.

* **What are other known failure modes?**

A Service `internalTrafficPolicy` is set to `Local` but there are no node-local endpoints.

* **What steps should be taken if SLOs are not being met to determine the problem?**

* check Service for internal traffic policy
* check EndpointSlice to ensure nodeName is set correctly
* check iptables/ipvs rules on kube-proxy

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

2020-10-09: KEP approved as implementable in "alpha" stage.
2021-03-08: alpha implementation merged for v1.21
2021-05-12: KEP approved as implementable in "beta" stage.

## Drawbacks

Added complexity in the Service API and in kube-proxy to address node-local routing.
This also pushes some responsibility on application owners to ensure pods are scheduled
to work with node-local routing.

## Alternatives

### EndpointSlice Subsetting

EndpointSlice subsetting per node can address the node-local use-case, but this would not be very scalable
for large clusters since that would require an EndpointSlice resource per node.

### Bool Field For Node Local

Instead of `trafficPolicy` field with codified values, a bool field can be used to enable node-local routing.
While this is simpler, it is not expressive enough for the `PreferLocal` use-case where traffic should ideally go
to a local endpoint, but be routed somewhere else otherwise.

