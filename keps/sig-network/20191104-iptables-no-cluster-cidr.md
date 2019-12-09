---
title: Remove knowledge of pod cluster CIDR from iptables rules
authors:
  - "@satyasm"
owning-sig: sig-network
participating-sigs:
reviewers:
  - "@thockin"
  - "@caseydavenport"
  - "@mikespreitzer"
  - "@aojea"
  - "@fasaxc"
  - "@squeed"
  - "@bowei"
  - "@dcbw"
  - "@darwinship"
approvers:
  - "@thockin"
editor: TBD
creation-date: 2019-11-04
last-updated: 2019-11-27
status: implementable
see-also:
replaces:
superseded-by:
---

# Removing Knowledge of pod cluster CIDR from iptables rules

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [iptables - masquerade off cluster traffic to services by node IP](#iptables---masquerade-off-cluster-traffic-to-services-by-node-ip)
  - [iptables - redirecting pod traffic to external loadbalancer VIP to cluster IP](#iptables---redirecting-pod-traffic-to-external-loadbalancer-vip-to-cluster-ip)
  - [iptables - accepting traffic after first packet, after being accepted by kubernetes rules](#iptables---accepting-traffic-after-first-packet-after-being-accepted-by-kubernetes-rules)
  - [ipvs - masquerade off cluster traffic to services by node IP](#ipvs---masquerade-off-cluster-traffic-to-services-by-node-ip)
  - [ipvs - accepting traffic after first packet, after being accepted by kubernetes rules](#ipvs---accepting-traffic-after-first-packet-after-being-accepted-by-kubernetes-rules)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
  - [Multiple cluster CIDR rules](#multiple-cluster-cidr-rules)
  - [ip-masq-agent like behavior](#ip-masq-agent-like-behavior)
<!-- /toc -->

## Summary

The iptables implementation of kube-proxy today references the cluster CIDR for pods in three places for the following reasons.

   1. [Masquerade off cluster traffic to services by node IP](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/iptables/proxier.go#L887-L894)
   2. [Redirecting pods traffic to external loadbalancer VIP to cluster IP](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/iptables/proxier.go#L1248-L1260)
   3. [Accepting traffic after first packet, after being accepted by kubernetes rules](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/iptables/proxier.go#L1389-L1411)

In addition, the ipvs implementation also references it in two places for similar purposes

   1. [Masquerade off cluster traffic to services by node IP](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/ipvs/proxier.go#L1558-L1563)
   2. [Accepting traffic after first packet, after being accepted by kubernetes](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/ipvs/proxier.go#L1635-L1654)

This enhancement proposes ways to achieve similar goals without tracking the pod cluster CIDR to do so.

## Motivation

The idea that makes kubernetes networking model unique and powerful is the concept of each pod having its own IP, 
with all the pod IPs being natively routable within the cluster. The service chains in iptable rules depend on this 
capability by assuming that they can treat all the endpoints of a cluster as being equivalent and load balance service 
traffic across all the endpoints, by just translating destination to the pod IP address.

While this is powerful, it also means pod IP addresses are in many cases the constraining resource for cluster creation
and scale. It would be valuable for implementations to have different strategies for managing pod IP addresses that can
adapt to different environment needs.

Some examples of use cases:

   * Creating a cluster out of many disjoint ranges instead of a single range.
   * Expanding a cluster with more disjoint ranges after initial creation.

Not having to depend on the cluster pod CIDR for routing service traffic would effectively de-couple pod IP management
and allocation strategies from service management and routing. Which in turn would mean that it would be far cheaper 
to evolve the IP allocation schemes while sharing the same service implementation, thus significantly lowering the bar
for adoption of alternate schemes.

Alternate implementations that don’t use iptables could also adopt this same reasoning to not have to track the cluster
CIDR for routing cluster traffic.

### Goals

   * Not having to depend on the cluster pod CIDR for iptable rules and cluster traffic routing.

### Non-Goals

   * Providing alternate models of IP allocation schemes for pod CIDR.
   * Enhancing current allocators to handle disjoint ranges.
   * Enhancing current allocators to add additional ranges after cluster creation.
   * Changing current assumptions around having a single pod CIDR per node.

## Proposal

As stated above, the goal is to re-implement the functionality called out in the summary, but in a 
way that does not depend on a pod cluster CIDR. The essence of the proposal is that for the 
first two cases in iptables implementation and first case in ipvs, we can replace the `-s proxier.clusterCIDR` with 
some notion of node local pod traffic.

The core logic in these cases is “how to determine” cluster originated traffic from non-cluster originated ones. 
The proposal is that tracking pod traffic generated from within the node is sufficient to determine cluster originated 
traffic. For the first two use cases in iptables and first use case in ipvs, we provide alternatives to using 
proxier.clusterCIDR in one of the following ways to determine cluster originated traffic

   1. `-s node.podCIDR` (where node podCIDR is used for allocating pod IPs within the node)
   2. `--in-interface prefix+` (where all pod interfaces start with same prefix,
      or where all pod traffic appears to come from a single bridge or other interface)
   3. `-m physdev --physdev-is-in` (for kubenet if we don’t want to depend on node podCIDR)

Note the above are equivalent definitions, when considering only pod traffic originating from within the node.

For the last use case, in iptables and ipvs, the proposal is to drop the reference to the cluster CIDR.

The reasoning behind why this works are as follows.

### iptables - masquerade off cluster traffic to services by node IP

The [rule here currently](
  https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/iptables/proxier.go#L887-L894
) looks as follows

```go
// This masquerades off-cluster traffic to a service VIP.  The idea
// is that you can establish a static route for your Service range,
// routing to any node, and that node will bridge into the Service
// for you.  Since that might bounce off-node, we masquerade here.
// If/when we support "Local" policy for VIPs, we should update this.
writeLine(proxier.natRules, append(args, "! -s", proxier.clusterCIDR, "-j", string(KubeMarkMasqChain))...)
```

The logic is that if the source IP is not part of the cluster CIDR range,
then it must have originated from outside the cluster. Hence we add a rule to masquerade by
the node IP so that we can send traffic to any pod within the cluster.

One key insight when thinking about this data path though is the fact that the iptable rules run
at _every_ node boundary. So when a pod sends a traffic to a service IP, it gets translated to
one of the pod IPs _before_ it leaves the node at the node boundary. So it's highly unlikely to 
receive traffic at a node, whose destination is the service cluster IP, that is initiated by pods
within the cluster, but not scheduled within that node.

Going by the above reasoning, if we receive traffic destined to a service whose source is not within the node 
generated pod traffic, we can say with very high confidence that the traffic originated from outside the cluster. 
So we can rewrite the rule in terms of just the pod identity within the node (node CIDR, interface prefix or bridge).
This would be the simplest change with respect to re-writing the rule without any assumptions on how pod 
networking is setup.

### iptables - redirecting pod traffic to external loadbalancer VIP to cluster IP

The [rule here currently](
  https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/iptables/proxier.go#L1248-L1260
) looks as follows

```go
// First rule in the chain redirects all pod -> external VIP traffic to the
// Service's ClusterIP instead. This happens whether or not we have local
// endpoints; only if clusterCIDR is specified
if len(proxier.clusterCIDR) > 0 {
  args = append(args[:0],
    "-A", string(svcXlbChain),
    "-m", "comment", "--comment",
    `"Redirect pods trying to reach external loadbalancer VIP to clusterIP"`,
    "-s", proxier.clusterCIDR,
    "-j", string(svcChain),
  )
  writeLine(proxier.natRules, args...)
}
```

The logic here is that if the source IP is part of cluster CIDR and we detect that is being
sent to a load balancer IP for a service, we short circuit it by jumping directly to the
service chain instead of having the packet go out of the cluster, get routed back and then
translated to one of the backends.

Given that iptable rules are applied at the node boundary before any traffic from pods within
that node leave the node, the same arguments above apply here for replacing the cluster CIDR
with a representation of pod's nodeCIDR or it's interfaces.

### iptables - accepting traffic after first packet, after being accepted by kubernetes rules

The [rule here currently](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/iptables/proxier.go#L1389-L1411)
looks as follows

```go
// The following rules can only be set if clusterCIDR has been defined.
if len(proxier.clusterCIDR) != 0 {
  // The following two rules ensure the traffic after the initial packet
  // accepted by the "kubernetes forwarding rules" rule above will be
  // accepted, to be as specific as possible the traffic must be sourced
  // or destined to the clusterCIDR (to/from a pod).
  writeLine(proxier.filterRules,
    "-A", string(kubeForwardChain),
    "-s", proxier.clusterCIDR,
    "-m", "comment", "--comment", `"kubernetes forwarding conntrack pod source rule"`,
    "-m", "conntrack",
    "--ctstate", "RELATED,ESTABLISHED",
    "-j", "ACCEPT",
  )
  writeLine(proxier.filterRules,
    "-A", string(kubeForwardChain),
    "-m", "comment", "--comment", `"kubernetes forwarding conntrack pod destination rule"`,
    "-d", proxier.clusterCIDR,
    "-m", "conntrack",
    "--ctstate", "RELATED,ESTABLISHED",
    "-j", "ACCEPT",
  )
}
```

The interesting part of this rule that it already matches conntrack state to "RELATED,ESTABLISHED", 
which means that it does not apply to the initial packet, but after the connection has been setup and accepted.

In this case, dropping the `-d proxier.clusterCIDR` rule should have minimal impact on it behavior.
We would just be saying that if any connection is already established or related, just accept it.

In addition, since this rule is written after the rule to drop packets marked by `KUBE-MARK-DROP`,
by the time we reach this rule, packets marked to dropped by kubernetes would already have been dropped.
So it should not break any kubernetes specific logic.

Unfortunately in this case, it's not possible replace the cluster CIDR rule with local CIDR as
the traffic could be getting forwarded through this node to another node.

### ipvs - masquerade off cluster traffic to services by node IP

The [rule here currently](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/ipvs/proxier.go#L1558-L1563)
looks as follows.

```go
// This masquerades off-cluster traffic to a service VIP.  The idea
// is that you can establish a static route for your Service range,
// routing to any node, and that node will bridge into the Service
// for you.  Since that might bounce off-node, we masquerade here.
// If/when we support "Local" policy for VIPs, we should update this.
writeLine(proxier.natRules, append(args, "dst,dst", "! -s", proxier.clusterCIDR, "-j", string(KubeMarkMasqChain))...)
```

By the same logic used in the first case for iptables, we can replace references to clusterCIDR with equivalent
node specific pod identification (node.podCIDR, interface prefix or bridge) to determine whether the traffic originated
from within the cluster or not.

### ipvs - accepting traffic after first packet, after being accepted by kubernetes rules

The [rule here currently](https://github.com/kubernetes/kubernetes/blob/master/pkg/proxy/ipvs/proxier.go#L1635-L1654)
looks as follows

```go
// The following two rules ensure the traffic after the initial packet
// accepted by the "kubernetes forwarding rules" rule above will be
// accepted, to be as specific as possible the traffic must be sourced
// or destined to the clusterCIDR (to/from a pod).
writeLine(proxier.filterRules,
	"-A", string(KubeForwardChain),
	"-s", proxier.clusterCIDR,
	"-m", "comment", "--comment", `"kubernetes forwarding conntrack pod source rule"`,
	"-m", "conntrack",
	"--ctstate", "RELATED,ESTABLISHED",
	"-j", "ACCEPT",
)
writeLine(proxier.filterRules,
	"-A", string(KubeForwardChain),
	"-m", "comment", "--comment", `"kubernetes forwarding conntrack pod destination rule"`,
	"-d", proxier.clusterCIDR,
	"-m", "conntrack",
	"--ctstate", "RELATED,ESTABLISHED",
	"-j", "ACCEPT",
)
```
Again, applying similar logic to the last rule for iptables, the proposal here is to simplify this drop reference to
the proxy.clusterCIDR and just match on the connection state.

### Risks and Mitigations

The biggest risk we have is that we are expanding the scope of the last rule to potentially include non-kubernetes
traffic. This is considered mostly safe as it does not break any of the intended drop behavior. Plus once the initial
connection has been accepted, assuming nodes are used for kubernetes workloads, it's highly unlikely that we would
need to not accept it later.

## Design Details

The idea of ‘determine cluster originated traffic’ would be captured in a new go interface type within kube-proxy,
with different implementations of the interface. The kube-proxy implementation itself would just call method on
interface to get the match criteria to write in the rule.

The new behavior can be opted-in using two flags. The first to determine the mode to use for detection, and the
other (optionally) being the value to use in that mode. This separation of mode and value has the nice property
that if we default the mode to "cluster-cidr", then the current `--cluster-cidr` flag can be used as is to get
the current behaviour. So upgrades with no changes retain current behavior.

```
--detect-local={cluster-cidr | node-cidr | pod-interface-prefix  | bridge}

  the mode to use for detection local traffic. The default is cluster-cidr (current behavior)

--cluster-cidr="cidr[,cidr,..]"

  the current --cluster-cidr flag. It will be enhanced to read a comma separated list of CIDRs so that more
  than one can be specified if necessary. kube-proxy considers traffic as local if source is one
  of the CIDR values. This is only used if `--detect-local=cluster-cidr` .

--node-cidr[="cidr[,cidr,..]"]

  (optional) list of node CIDRs as a comma separated list. kube-proxy considers traffic as local if source is one
  of the CIDR values. If value is not specified, or flag is omitted,  defaults to node.podCIDR property on the node.
  This is only used if `--detect-local=node-cidr` .

--pod-interface-prefix="prefix[,prefix,..]"

  kube-proxy considers traffic as local if originating from an interface which matches one of given
  prefixes. string argument is a comma separated list of interface prefix names, without the ending '+'.
  This is only used if `--detect-local=pod-interface-prefix`
```

Given that we are handling a list of rules, the jump to `KUBE-MARK-MARQ` will be implemented with a
jump to a new chain `KUBE-MASQ-IF-NOT-LOCAL` which will then either return or jump to `KUBE-MARK-MASQ`
as appriate. For example:

```
-A WHEREVER -blah -blah -blah -j MARK-MASQ-IF-NOT-LOCAL

-A MARK-MASQ-IF-NOT-LOCAL -s 10.0.1.0/24 -j RETURN
-A MARK-MASQ-IF-NOT-LOCAL -s 10.0.3.0/24 -j RETURN
-A MARK-MASQ-IF-NOT-LOCAL -s 10.0.5.0/24 -j RETURN
-A MARK-MASQ-IF-NOT-LOCAL -j KUBE-MARK-MASQ
```

Future changes to detection of local traffic (say using things like mark etc) can be done by adding more options
to the `--detect-local` mode flag with any appropriate additional flags.

### Graduation Criteria

These additional flags will go through alpha, beta etc graduation as for any feature.

## Implementation History

2019-11-04 - Creation of the KEP
2019-11-27 - Revision with Implementation Details

## Drawbacks [optional]

The main caveat in this KEP is the relaxation of the accept rule for "ESTABLISHED,RELATED" packets. The other two rules
have equivalent implementations, as long as we continue to guarantee that pod traffic is routed at the node boundary
on _every_ and _all_ nodes that makes up the kubernetes cluster. This would not work if that assumption were to change.

## Alternatives [optional]

### Multiple cluster CIDR rules
One alternative to consider is to explicitly track a list of cluster CIDRs in the ip table rules. If we 
want to do this, we might want to consider making the cluster CIDR a first class resource, which we want to avoid.

Instead in most cases, where the interface prefix is mostly fixed or we are using the `node.spec.podCIDR` attribute,
changes to the cluster CIDR does not need any change to the kube-proxy arguments or a restart, which we believe 
is of benefit when managing clusters.

### ip-masq-agent like behavior
The other alternative is to have kube-proxy never track it and instead use something like 
[ip-masq-agent](https://kubernetes.io/docs/tasks/administer-cluster/ip-masq-agent/) to track what we masquerade
or not. In this case, it assumes more knowledge from the users, but it does provide for a single place to update
these cidrs using existing tooling.
