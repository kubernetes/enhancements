---
title: Topology-aware service routing
authors:
  - "@m1093782566"
  - "@imroc"
owning-sig: sig-network
reviewers:
  - "@thockin"
  - "@johnbelamaric"
approvers:
  - "@thockin"
creation-date: 2018-10-24
last-updated: 2019-07-22
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
  - [Service API changes](#service-api-changes)
  - [Topology Infomation](#topology-infomation)
  - [Kube-proxy changes](#kube-proxy-changes)
  - [DNS server changes (in beta stage)](#dns-server-changes-in-beta-stage)
  - [Graduation Criteria](#graduation-criteria)
    -  [Alpha](#alpha)
    -  [Alpha -&gt; Beta](#alpha---beta)
    -  [Beta -&gt; GA Graduation](#beta---ga-graduation)
  
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

Users can control the level of topology. For example, if someone run logging agent as a daemonset, he can set the "hard" topology requirement for same-host. If "hard" is not met, then just return "service not available". 

And if someone set a "soft" topology requirement for same-host, say he "preferred" same-host endpoints and can accept other hosts when for some reasons local service's backend is not available on some host.

If multiple endpoints satisfy the "hard" or "soft" topology requirement, we will randomly pick one by default. 

Routing decision is expected to be implemented by kube-proxy and kube-dns/coredns for headless service.


### Service API changes

Users need a way to declare what service is local and the definition of local backends for the particular service.

In this proposal, we give the service owner a chance to configure the service locality things. A new property would be introduced to `ServiceSpec`, say `topologyKeys` - it's a string slice and should be optional.

```go
type ServiceSpec struct {
  // topologyKeys is a preference-order list of topology keys.  If backends exist for
  // index [0], they will always be chosen; only if no backends exist for index [0] will backends for index [1] be considered.
  // If this field is specified and all indices have no backends, the service has no backends, and connections will fail.  We say these requirements are hard.
  // In order to experss soft requirement, we may give a special node label key "" as it means "match all nodes".
  TopologyKeys []string `json:"topologyKeys" protobuf:"bytes,1,opt,name=topologyKeys"`
}
```

An example of `Service` with topology keys:

```
kind: Service
metadata:
  name: service-local
spec:
  topologyKeys: ["kubernetes.io/hostname", "failure-domain.beta.kubernetes.io/zone", "failure-domain.beta.kubernetes.io/region"]
```


In our example above, we will firstly try to find the backends in the same host. If no backends match, we will then try the lucky of same zone. If finally we can't find any backends in the same host or same zone, then we say the service has no satisfied backends and connections will fail.

If we configure topologyKeys as `["kubernetes.io/hostname", ""]`, we just do the effort to find the backends in the same host and will not fail the connection if no matched backends found.

### Topology Infomation

The alpha release of Service Topology will have a simple implementation where kube-proxy and any other consumers of `topologyKeys` will determine the locality of backends
by filtering Endpoints and Nodes in-memory from client-go's cache informer/lister, which means kube-proxy will watch node resource if this feature is enabled. The beta release of Service Topology will consuming  `EndpointSlice` which introduced in this [Proposal](https://github.com/freehan/enhancements/blob/master/keps/sig-network/20190603-EndpointSlice-API.md) (It is still at a very early stage when writing this proposal), and do not watch node anymore.


### Kube-proxy changes

Kube-proxy will respect topology keys for each service, so kube-proxy on different nodes may create different proxy rules.

Kube-proxy will watch its own node and will find the endpoints that are in the same topological domain as the node if `service.TopologyKeys` is not empty.

Kube-proxy will only create proxy rules for endpoints that are in the same topological domain as the node running kube-proxy. For Alpha, kube-proxy will watch Node apart from Service and Endpoints to get topology infomation in Alpha, kube-proxy will find the Node where the Endpoint is located via EndpointAddress.NodeName, therefore will get its topology information by Node's labels. For Beta, kube-proxy will watch EndpointSlice instead of Node which already contains Endpoint's topology infomation.
 

### DNS server changes (in beta stage)

We should consider this kind of topology support for headless service in coredns and kube-dns. As the DNS servers will respect topology keys for each headless service, different clients/pods on different nodes may get different dns response.

In order to handle headless services, the DNS server needs to know the node corresponding to the client IP address in the DNS request - i.e, it needs to map PodIP -> Node. Kubernetes DNS servers(include kube-dns and CoreDNS) will watch EndpointSlice object. When a client/pod request a headless service domain to DNS server, dns server will retrieve the node labels of both client and the backend Pods via EndpointSlice. DNS server will only select the IPs of backend Pods which are in the same topological domain with client Pod, and then write A record.


### Graduation Criteria

#### Alpha

- topologyKeys field is added to the Service API
- kube-proxy implements service topology in-memory by watching Node only enabled by `TopoloyAwareServiceRouting` feature gate.

#### Alpha -> Beta

- kube-proxy is updated to watch `EndpointSlice` for topology filtering of endpoints.
- coredns/kube-dns watches `EndpointSlice` objets and performs topology-aware headless services.
- E2E tests exist for service topology.

##### Beta -> GA Graduation

TODO: after beta
