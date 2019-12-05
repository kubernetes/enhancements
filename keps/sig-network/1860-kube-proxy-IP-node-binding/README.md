# KEP-1860 Make Kubernetes aware of the LoadBalancer behaviour

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes


## Summary

The different kube-proxy implementations (at least ipvs and iptables), as of today, are binding the External IPs of the LoadBalancer Service to each node. (iptables are creating some rules to redirect packets directly to the service and ipvs is binding the IP to one interface on the node). This feature exists because:

1. To redirect pod traffic that is intended for LoadBalancer IP directly to the service, bypassing the loadbalancer for path optimization.
2. Some LBs are sending packet with the destination IP equal to the LB IP, so the IP needs to be routed directly to the right service (otherwise looping back and forth)

This enhancement proposes a way to make these features configurable, giving the cloud provider, a way to disable the behaviour of kube-proxy.
The best way would be to set a default behaviour at the cloud controller level.

## Motivation

There are numerous problems with the current behaviour:

1. Some cloud providers (Scaleway, Tencent Cloud, ...) are using the LB's external IP (or a private IP) as source IP when sending packets to the cluster. This is a problem in the ipvs mode of kube-proxy since the IP is bounded to an interface and healthchecks from the LB are never coming back.
2. Some cloud providers (DigitalOcean, Scaleway, ...) have features at the LB level (TLS termination, PROXY protocol, ...). Bypassing the LB means missing these features when the packet arrives at the service (leading to protocols errors).

So, giving options to these cloud providers to disable the actual behaviour would be very valuable.

Currently there is some hacky workaround that set the `Hostname` on the `Service` in order to bypass kube-proxy binding (AWS and DigitalOcean for instance), but as stated, it's a bit hacky. This KEP should also give a solution to this hack.

### Goals

* Expose an option for the different controllers managing Load Balancer (e.g. the Cloud Controller Manager), to choose the actual behaviour of kube-proxy for all services.

### Non-Goals

* Deprecate the default behaviour of kube-proxy
* Change the default behaviour of kube-proxy

## Proposal

The solution would be to add a new field in the `loadBalancer` field of a `Service`'s `status`, like `ingressMode`. This new field will be used by kube-proxy in order to not bind the Load Balancer's External IP to the node (in both IPVS and iptables mode). The value `VIP` would be the default one (if not set for instance), keeping the current behaviour. The value `Proxy` would be used in order to disable the the shortcut.

Since the `EnsureLoadBalancer` returns a `LoadBalancerStatus`, the Cloud Controller Manager can optionally set the `ingressMode` field before returning the status.

### User Stories

#### Story 1

User 1 is a Managed Kubernetes user. There cluster is running on a cloud provider where the LB's behaviour matches the `VIP` `ingressMode`.
Nothing changes for them since the cloud provider manages the Cloud Controller Manager.

#### Story 2

User 2 is a Managed Kubernetes user. There cluster is running on a cloud provider where the LB's behaviour matches the `Proxy` `ingressMode`.
Almost nothing changes for them since the cloud provider manages the Cloud Controller Manager.
The only difference is that pods using the load balancer IP may observe a higher response time since the datapath will go through the load balancer. 

#### Story 3

User 3 is a developer working on a cloud provider Kubernetes offering. 
On the next version of `k8s.io/cloud-provider`, the cloud provider can optionally set the `ingressMode` field as part of `LoadBalancerStatus`, and reflect the cloud provider load balancer behaviour.

### Risks and Mitigations

1. The first risk is when using the `Proxy` `ingressMode` for pod using the load balancer IP. In this case the packets will not bypass the load balancer anymore.
However if a user wants to to keep using the in cluster datapath, he can still use the ClusterIP of the service.

## Drawbacks

1. Performance hit, see the risk 1. above.

## Alternatives

A viable alternative may be to use a flag directly on `kube-proxy`.
When running on a cloud provider managed Kubernetes this solution is viable since the cloud provider will be able to set the right value on `kube-proxy`.
When running a self hosted cluster, the user needs to be aware of how the cloud's load balancers works and need to set the flag on `kube-proxy` accordingly.
