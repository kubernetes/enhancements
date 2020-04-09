---
title: graduate-nodelocaldns-to-beta
authors:
  - "@prameshj"
owning-sig: sig-network
reviewers:
  - "@bowei"
  - "@thockin"
  - "@johnbelamaric"
approvers:
  - "@bowei"
  - "@thockin"
creation-date: 2019-04-24
last-updated: 2020-09-23
status: implemented
see-also:
  - "/keps/sig-network/0030-nodelocal-dns-cache.md"
---

# Graduate NodeLocal DNSCache to beta

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Alternatives](#alternatives)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

[NodeLocal DNSCache](https://github.com/kubernetes/kubernetes/blob/master/cluster/addons/dns/nodelocaldns/README.md) is an addon that runs a dnsCache pod as a daemonset to improve clusterDNS performance and reliability. The feature has been in Alpha since 1.13 release. This document lays out the plan to promote it to beta.

## Motivation

NodeLocal DNSCache has been in Alpha for the past 2 releases and users have deployed it to fix DNS performance issues.
- [Example 1](https://github.com/kubernetes/kubernetes/issues/56903#issuecomment-485353223)
- [Example 2](https://github.com/kubernetes/kubernetes/issues/45363#issuecomment-443019910)

Based on the feedback so far, we feel the feature is ready to be graduated to beta.

### Goals

Graduate NodeLocal DNSCache to beta.

## Proposal

N.B. Although CoreDNS is now the default DNS server on Kubernetes clusters, this document still uses the name kube-dns since the service name is still the same.

Based on the initial feedback for NodeLocal DNSCache feature, HA seems to be the common ask. 

The current implementation introduces a single point of failure, since all pods on a node rely on the node-local-dns pod that is running on the same node, for DNS requests.
Any dedicated node-agent brings with it the issue of single-point of failure. Example: kube-proxy or any other CNI pod has a similar issue. These are in the control plane and they do leave behind some state upon exit that is sufficient for services to partially work. In that way, the scenario is a little different from the a node-cache pod going down, since the latter in in the data path. But it is not all that different since kube-proxy if down for long enough, will cause a drift from current configured state resulting in datapath failures.

Here are some failure modes for the node-cache pod:

1) Pod Evicted - We create this daemonset with `priorityClassName: system-node-critical` setting to greatly reduce the likelihood of eviction.
2) Config error - node-local-dns restarts repeatedly due to incorrect config. This will be resolved only when the config error has been fixed. There will be DNS downtime until then, even though the kube-dns pods might be available.
3) OOMKilled - node-local-dns gets OOMKilled due to its own memory usage or some other component using up all memory resources on the node. There is a chance this will cause other disruptions on the node in addition to DNS downtime though. 
4) Upgrades to node-local-dns daemonset - There will be DNS downtime when node-local-dns pods shut down, until the new pods are up and running.

We are proposing a solution that will help in all these cases. For beta, we will start providing enablement for HA, full implementation will be a GA criterion.
 
The proposal here is to use an additional listen IP for node-local-dns pod. The node-local-dns pod listens on the 169.254.20.10 IP address today. We will extend node-local-dns to listen on the kube-dns service IP as well. Requests to kube-dns service IP will be handled by node-local-dns pod when it is up. If it is unavailable, the requests will go to kube-dns endpoints instead. The determination of whether node-local-dns service is available will be done by an external component - This could be a new daemonset or new functionality in an existing daemonset that manages networking.

### Risks and Mitigations

* The proposed HA solution will not work in IPVS mode of kube-proxy. This is because skipping IPVS translation rules is not possible using an iptables NOTRACK rule. So, if the service IP is used by pods for DNS resolution, requests will always hit the IPVS load-balancing rules and reach the kube-dns endpoints.
One way to get the desired behavior is to change the selectors for the kube-dns service, so that we set empty endpoints when we want the node-local-dns pods to be used. However, this approach has not been tested yet. Also, this would change it across the board, instead of for a single node as the need may be.

* If the pod performing the checks and flipping the DNS server gets evicted, we could still end up with DNS downtime.

* Adding a watcher introduces more resource consumption to support NodeLocal DNSCache feature. This can be mitigated by combining this logic into an existing daemonset. Also, it will be possible to run node-local-dns without this additional component, without the HA benefit.

## Design Details

In this new design, node-local-dns pod creates a dummy interface with 2 IP addresses - the link local IP address 169.254.20.10(This happens already today) and the kube-dns service IP address. 
A new service spec is added to the node-local-dns yaml, this is almost identical to the kube-dns service spec. 

```
apiVersion: v1
kind: Service
metadata:
  name: node-local-upstream
  namespace: kube-system
  labels:
    k8s-app: kube-dns
    kubernetes.io/cluster-service: "true"
    addonmanager.kubernetes.io/mode: Reconcile
    kubernetes.io/name: "NodeLocalUpstream"
spec:
  selector:
    k8s-app: kube-dns
  ports:
  - name: dns
    port: 53
    protocol: UDP
  - name: dns-tcp
    port: 53
```

It has the same selectors as kube-dns, so this service IP will be mapped to the same kube-dns endpoints.
This new service is required for node-local-dns pod to talk to kube-dns endpoints. This will be the IP address used by node-local-dns in case of cache misses in the cluster.local domain.
This service spec does not reserve a specific clusterIP, let's assume the assigned IP is 10.0.0.50 (it can be different on each setup). Let's assume the kube-dns IP is 10.0.0.10.
By default, kube-proxy will install rules so that packets targeting 10.0.0.10 are DNAT'ed to one of the kube-dns endpoints. A similar rule will be installed for 10.0.0.50 as well. However, we need packets to 10.0.0.10 to be sent to the local interface that node-local-dns pod is listening on. This is possible by using the NOTRACK action in iptables.

As mentioned in the previous [KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/0030-nodelocal-dns-cache.md#proposal), node-local-dns pod installs iptables rule with NOTRACK action so that connections to and from the node-local-dns IP on port 53 can avoid being tracked via CONNTRACK. The purpose was to prevent usage of conntrack entries for DNS requests. Another benefit is that this avoids additional NAT table rules from being applied on the packet. So, as long as we have a rule 
`-d 10.0.0.10 --dport 53 -j NOTRACK`,
the NAT table rules that reroute the packet to kube-dns endpoints will not be applied. We also need a filter table rule to make sure the packet isn't dropped.
The request packet can now be locally consumed by node-local-dns pod. The node-local-dns pod will use the new service IP - 10.0.0.50, as its Upstream Nameserver, which will still map to the endpoints.
So, we can use the NOTRACK rule as a switch to flip between using node-local-dns and kube-dns endpoints.

Here is a diagram to explain this flow:


![ ](nodelocal-HA.png  "NodeLocalDNS HA-design")


As summarized above, we will use the iptables NOTRACK rule to implement the "USE LOCAL?" condition in the diagram.

The benefits of this approach are:

1) node-local-dns can be used on existing clusters without any kubelet change. Pods continue to use kube-dns service IP in their /etc/resolv.conf and we transparently switch the backend to the new cache.

2) We are able to, somewhat elegantly, failover to kube-dns endpoints.

3) Disabling node-local-dns does not require any kubelet change either.

We still need some component to dynamically determine when to use node-local-dns and when to flip to kube-dns endpoints. This logic can be separated out into an independent container/pod whose function is to query for dns records on 169.254.20.10:53 and follow some threshold to either install or remove the NOTRACK rules. This can be a new Daemonset or combined into an existing Daemonset that is in HostNetwork mode and manages iptables rules in some way - for instance a CNI Daemonset. This component will handle adding all iptables rules needed for node-local-dns.

The caveat of this approach is that it only works in the iptables implementation of kube-proxy. 
Another observation is that the upstream dns server IP used by node-local-dns will differ from one setup to another since it is a dynamically allocated service IP.  This doesn't appear to be a major concern.

### Test Plan

* We are running all the existing DNS tests with NodeLocal DNSCache enabled:
  - [kube-dns-performance-nodecache](https://k8s-testgrid.appspot.com/sig-network-gce#gce-kubedns-performance-nodecache)
  - [coredns-performance-nodecache](https://k8s-testgrid.appspot.com/sig-network-gce#gce-coredns-performance-nodecache)
  - [kube-dns-nodecache](https://k8s-testgrid.appspot.com/sig-network-gce#gci-gce-kube-dns-nodecache)


### Graduation Criteria

In order to graduate to beta, we need:

* Lock down the node-local-dns configmap so that Corefile cannot be modified directly.

* Enablement of HA for NodeLocal DNSCache. With this support, the iptables rules management can be separated out to a different component. node-local-dns pod will accept multiple listen IP addresses as well.

### Alternatives

One suggestion for HA that has come up is to list multiple nameservers in the client pods' /etc/resolv.conf - both node-local-dns IP as well as kube-dns service IP.
This is not recommended because the behavior is inconsistent depending on the client library. glibc 2.16+ and musl implementations send queries in parallel to both nameservers, so if we use both kube-dns IP as well as the link-local IP used by NodeLocal DNSCache, we could make the DNS query explosion problem worse. More queries means more conntrack entries and more DNATs.
This workaround could be viable for client implementations that do round-robin.

Running 2 daemonsets of node-local-dns using the same listenIP - 169.254.20.10 via SO_REUSEPORT option. Upgrades will be done one daemonset at a time.

## Implementation History

NodeLocal DNSCache was introduced in Kubernetes 1.13
