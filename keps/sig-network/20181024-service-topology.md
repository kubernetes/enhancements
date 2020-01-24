---
title: Topology-aware service routing
authors:
  - "@m1093782566"
  - "@andrewsykim"
owning-sig: sig-network
reviewers:
  - "@thockin"
  - "@johnbelamaric"
approvers:
  - "@thockin"
creation-date: 2018-10-24
last-updated: 2019-10-17
status: implementable
---

# Topology-aware service routing

## Table of Contents

<!-- toc -->
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-goals](#non-goals)
  - [User cases](#user-cases)
  - [Background](#background)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Service API changes](#service-api-changes)
  - [Intersection with Local External Traffic Policy](#intersection-with-local-external-traffic-policy)
  - [Service Topology Scalability](#service-topology-scalability)
    - [New PodLocator resource](#new-podlocator-resource)
    - [New PodLocator controller](#new-podlocator-controller)
  - [EndpointSlice controller changes](#endpointslice-controller-changes)
  - [Kube-proxy changes](#kube-proxy-changes)
  - [DNS server changes (in beta stage)](#dns-server-changes-in-beta-stage)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta](#alpha---beta)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
<!-- /toc -->

## Motivation

Figure out a generic way to implement the "local service" route, say "topology aware routing of service".

Locality is defined by user, it can be any topology-related thing. "Local" means the "same topology level", e.g. same node, same rack, same failure zone, same failure region, same cloud provider etc. Two nodes are considered "local" if they have the same value for a particular label, called the "topology key".

### Goals

A generic way to support topology aware routing of services in arbitrary topological domains, e.g. node, rack, zone, region, etc. by node labels.

### Non-goals

* Scheduler spreading to implement this sort of topology guarantee
* Dynamic Availability
* Health-checking
* Capacity-based or load-based spillover

### User cases

* Logging agents such as fluentd. Deploy fluentd as DaemonSet and applications only need to communicate with the fluentd in the same node.
* For a sharded service that keeps per-node local information in each shard.
* Authenticating proxies such as [aws-es-proxy](https://github.com/kopeio/aws-es-proxy).
* In container identity wg, being able to give daemonset pods a unique identity per host is on the 2018 plan, and ensuring local pods can communicate to local node services securely is a key goal there. -- from @smarterclayton
* Regional data costs in multi-AZ setup - for instance, in AWS, with a multi-AZ setup, half of the traffic will switch AZ, incurring regional data Transfer costs, whereas if something was local, it wouldn't hit the network.
* Performance benefit (node local/rack local) is lower latency/higher bandwidth.

### Background

It's a pain point for multi-zone clusters deployment since cross-zone network traffic being charged, while in-zone is not. In addition, cross-node traffic may carry sensitive metadata from other nodes. Therefore, users always prefer the service backends that close to them, e.g. same zone, rack and host etc. for security, performance and cost concerns.

Kubernetes scheduler can constraining a pod to only be able to run on particular nodes/zones. However, Kubernetes service proxy just randomly picks an available backend for service routing and this one can be very far from the user, so we need a topology-aware service routing solution in Kubernetes. Basically, to find the nearest service backend. In other words, allowing people to configure if ALWAY reach a to local service backend. In this way, they can reduce network latency, improve security, save money and so on. However, because topology is arbitrary, zone, region, rack, generator, whatever, who knows? We should allow arbitrary locality.

`ExternalTrafficPolicy` was added in v1.4, but only for NodePort and external LB traffic. NodeName was added to `EndpointAddress` to allow kube-proxy to filter local endpoints for various future purposes.

Based on our experience of advanced routing setup and recent demo of enabling this feature in Kubernetes, this document would like to introduce a more generic way to support arbitrary service topology.

## Proposal

This proposal builds off of earlier requests to [use local pods only for kube-proxy loadbalancing](https://github.com/kubernetes/kubernetes/issues/7433) and [node-local service proposal](https://github.com/kubernetes/kubernetes/pull/28637). But, this document proposes that not only the particular "node-local" user case should be taken care, but also a more generic way should be figured out.

Locality is an "user-defined" thing. When we set topology key "hostname" for service, we expect node carries different node labels on the key "hostname".

Users can control the level of topology. For example, if someone run logging agent as a daemonset, they can set the "hard" topology requirement for same-host. If "hard" is not met, then just return "service not available".

And if someone set a "soft" topology requirement for same-host, say they "preferred" same-host endpoints and can accept other hosts when for some reasons local service's backend is not available on some host.

If multiple endpoints satisfy the "hard" or "soft" topology requirement, we will randomly pick one by default.

Routing decision is expected to be implemented by kube-proxy and kube-dns/coredns for headless service.


## Design Details

### Service API changes

Users need a way to declare what service is local and the definition of local backends for the particular service.

In this proposal, we give the service owner a chance to configure the service locality things. A new property would be introduced to `ServiceSpec`, say `topologyKeys` - it's a string slice and should be optional.

```go
type ServiceSpec struct {
  // topologyKeys is a preference-order list of topology keys.  If backends exist for
  // index [0], they will always be chosen; only if no backends exist for index [0] will backends for index [1] be considered.
  // If this field is specified and all indices have no backends, the service has no backends, and connections will fail.  We say these requirements are hard.
  // In order to express soft requirement, we may give a special node label key "*" as it means "match all nodes".
  TopologyKeys []string `json:"topologyKeys" protobuf:"bytes,1,opt,name=topologyKeys"`
}
```

An example of `Service` with topology keys:

```
kind: Service
metadata:
  name: service-local
spec:
  topologyKeys: ["kubernetes.io/hostname", "topology.kubernetes.io/zone"]
```


In our example above, we will firstly try to find the backends in the same host. If no backends match, we will then try the same zone. If finally we can't find any backends in the same host or same zone, then we say the service has no satisfied backends and connections will fail.

If we configure topologyKeys as `["kubernetes.io/hostname", "*"]`, we just do the effort to find the backends in the same host and will not fail the connection if no matched backends found.

### Intersection with Local External Traffic Policy

Topology Aware Services (# of topologyKeys > 0) and Services using `externalTrafficPolicy=Local` will be mutually exclusive. This will be enforced by API validation where a Service with `externalTrafficPolicy=Local` cannot specify any topology keys and vice versa.

Before Service Topology goes GA, kube-proxy should be able to provide feature parity with `externalTrafficPolicy=Local` when the chosen topology is `kubernetes.io/hostname`. The correctness and feasibility of the `kubernetes.io/hostname` topology as a sufficient replacement for `externalTrafficPolicy=Local` will be evaluated during the alpha phase.

### Service Topology Scalability

The alpha release of Service Topology will have a simple implementation where kube-proxy and any other consumers of `topologyKeys` will determine the locality of backends
by filtering Endpoints and Nodes in-memory from client-go's cache informer/lister. The alpha implementation of Service Topology will not account for scalabilty, however,
the beta release of Service Topology will address scalability concerns by introducing a `PodLocator` resource. Note that the design and implementation details around `PodLocator`
may vary based on user feedback in the alpha release.

#### New PodLocator resource

As `EndpointAddress` already contains the `nodeName` field, we can build a service that will preemptively map Pods to their topologies. From there, all interested components (at least kube-proxy, kube-dns, and coredns) can watch the `PodLocator` object and do necessary mappings internally. Given that we don't know which labels are topology labels, we are going to copy all node labels to `PodLocator`.

```
// PodLocator represents information about where a pod exists in arbitrary space.  This is useful for things like
// being able to reverse-map pod IPs to topology labels, without needing to watch all Pods or all Nodes.
type PodLocator struct {
    metav1.TypeMeta
    // +optional
    metav1.ObjectMeta

    // NOTE: Fields in this resource must be relatively small and relatively low-churn.

    IPs []PodIPInfo // being added for dual-stack support
    NodeName string
    NodeLabels map[string]string
}
```

In order to reference PodLocator back to a Pod easily, PodLocator namespace and name would be 1:1 with Pod namespace and name. In other words, PodLocator is a lightweight object which stores Pod location/topology information.

**NOTE**: whether `PodLocator` will be a core API resource or a CRD will be determined after the alpha release of Service Topology.

#### New PodLocator controller

A new PodLocator controller will watch and cache all Pods and Nodes. Then pre-cook pod name to {pod IPs, node name, node labels} mapping.

When a Pod is added, PodLocator controller will created a new PodLocator object whose namespace and name are 1:1 with Pod namespace and name. Then it will populate the Pod's IP(s), node name and labels into the new object.

When a Pod is updated, PodLocator controller will first check if IPs or Spec.NodeName are changed. If changed, PodLocator controller will update the corresponding PodLocator object accordingly, otherwise will ignore this change.

When a Pod is deleted, PodLocator controller will delete the corresponding PodLocator object.

When a Node is updated, PodLocator controller will first check if its labels are changed. If changed, will update all the PodLocators whose corresponding Pods running on it.

When a Node is deleted, PodLocator controller will reset the NodeName and NodeLabels of all the PodLocators whose corresponding Pods running on it.

### EndpointSlice controller changes

The EndpointSlice controller will be updated to set values corresponding to the
TopologyKeys field on Services. All topology values will be derived from labels
on the Node a Pod is running on.

### Kube-proxy changes

Kube-proxy will respect topology keys for each service, so kube-proxy on different nodes may create different proxy rules.

Kube-proxy will watch its own node and will find the endpoints that are in the same topological domain as the node if `service.TopologyKeys` is not empty.

For the beta release, kube-proxy will watch the PodLocator apart from Service and Endpoints. For each Endpoints object, kube-proxy will find the original Pod via EndpointAddress.TargetRef, therefore will get PodLocator object and its topology information. Kube-proxy will only create proxy rules for endpoints that are in the same topological domain as the node running kube-proxy.

### DNS server changes (in beta stage)

We should consider this kind of topology support for headless service in coredns and kube-dns. As the DNS servers will respect topology keys for each headless service, different clients/pods on different nodes may get different dns response.

In order to handle headless services, the DNS server needs to know the node corresponding to the client IP address in the DNS request - i.e, it needs to map PodIP -> Node. Kubernetes DNS servers(include kube-dns and CoreDNS) will watch PodLocator object. When a client/pod request a headless service domain to DNS server, dns server will retrieve the node labels of both client and the backend Pods via PodLocator. DNS server will only select the IPs of backend Pods which are in the same topological domain with client Pod, and then write A record.

### Test Plan

* Unit tests to check API validation of topology keys.
* Unit tests to check API validation of incompatibility with
  externalTrafficPolicy.
* Unit tests for topology endpoint filtering function, including at least:
  * No topology constraints (no keys)
  * A single hard constraint (single specific key)
  * A single `"*"` constraint.
  * Multiple hard constraints (no final `"*"` key).
  * Multiple constraints plus a final `"*"` key.
* Each of the above sets of keys must have tests with varying topologies
  available.
* E2E tests with hard constraint and soft constraints using Endpoints.
* E2E tests with hard constraint and soft constraints using EndpointSlice.

### Graduation Criteria

#### Alpha

- topologyKeys field is added to the Service API
- kube-proxy implements service topology in-memory only enabled by `ServiceTopology` feature gate.
- Unit tests

#### Alpha -> Beta

- scalability concerns are addressed by introducing the `PodLocator` API.
- kube-proxy is updated to watch `PodLocator` for topology filtering of endpoints.
- coredns watches `PodLocator` objets and performs topology-aware headless services.
- E2E tests exist for service topology.

##### Beta -> GA Graduation

TODO: after beta
