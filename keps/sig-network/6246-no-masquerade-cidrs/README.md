# KEP-6246: Resolving Source IP Ambiguity for Node-to-Pod Traffic in Kube-Proxy

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [The Datapath and Masquerading Problem](#the-datapath-and-masquerading-problem)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [1. The `no-masquerade-cidrs` Option](#1-the-no-masquerade-cidrs-option)
  - [2. Kube-Proxy SNAT to Primary Node IP](#2-kube-proxy-snat-to-primary-node-ip)
  - [3. Clearer Recommended Behaviors for CNIs](#3-clearer-recommended-behaviors-for-cnis)
<!-- /toc -->

## Summary

This KEP addresses the lack of portability and expected behavior regarding the source IP for node-to-pod traffic in Kubernetes, particularly when traversing services via `kube-proxy`. Based on recent community discussions (e.g., kubernetes/kubernetes#138360), it introduces three structural solutions:
1. A datapath bypass in `kube-proxy` (`no-masquerade-cidrs`) to avoid unnecessary SNAT.
2. A configurable option to force `kube-proxy` to masquerade to the primary node IP instead of the bridge IP.
3. Establishing clearer recommended guidelines for CNI implementations to natively assign the node's primary IP for node-to-pod routing.

## Motivation

As Dan Winship noted in kubernetes/kubernetes#133024, Kubernetes makes no hard guarantees about what the source IP of node-to-pod traffic will be. For example, when pods run on a bridge network, traffic from the node to the pod frequently takes the private IP of the bridge as its source, rather than the node's primary public/internal IP. This breaks the assumption that users can portably write `NetworkPolicy` rules matching the node's primary IP to allow/deny node-to-pod traffic.

Some CNIs (such as Cilium) have already solved this for direct node-to-pod traffic by setting up routing rules so that traffic organically gets the node's primary IP. However, this fix does not naturally extend to traffic traversing `kube-proxy` (node-to-service-to-pod). Because `kube-proxy` unconditionally masquerades this traffic, the packet ends up bypassing the native routing rules set up by the CNI, resulting in an inconsistent source IP.

### The Datapath and Masquerading Problem

From a datapath perspective, unconditionally masquerading traffic is suboptimal. If `kube-proxy` knows it can simply DNAT the packet to a Pod IP and let it continue, the packet will hit the exact same CNI routing rules that direct node-to-pod traffic hits. This organically ensures the correct source IP is used without relying on an explicit NAT translation in iptables/nftables. 

While adding a `no-masquerade-cidrs` configuration requires explicit setup from administrators (impacting usability slightly), it provides a much cleaner, more structural datapath architecture.

Additionally, for situations where masquerading *is* required, allowing `kube-proxy` to explicitly SNAT to the node's primary InternalIP (rather than falling back to whatever IP the interface provides) is a necessary piece of the puzzle.

### Goals

- Implement a `no-masquerade-cidrs` configuration to allow `kube-proxy` to skip MASQUERADE for specific destination CIDRs.
- Implement an option (or behavior) for `kube-proxy` to explicitly SNAT to the node's primary IP instead of the bridge IP.
- Formalize a *recommended behavior* for CNIs to guarantee that node-to-pod traffic uses the primary node IP as the source IP, noting that bridge IP usage is generally an anti-pattern that wasn't previously strictly discouraged.

### Non-Goals

- Deprecate existing generic flags like `cluster-cidr` or change their current behavior.
- Tighten conformance requirements to strictly fail CNIs that use the bridge IP, as this would break significant existing usage. We only seek to clarify the recommended path.

## Proposal

### 1. The `no-masquerade-cidrs` Option

We propose extending the `KubeProxyConfiguration` type:

```go
type KubeProxyConfiguration struct {
    ...
    // NoMasqueradeCIDRs is an explicit list of CIDRs to which traffic should not be masqueraded.
    NoMasqueradeCIDRs []string `json:"noMasqueradeCIDRs,omitempty"`
}
```

When generating routing rules (iptables or nftables), `kube-proxy` will insert a `RETURN` or equivalent skip mechanism in the post-routing masquerade chain. If a packet's destination matches any of the `no-masquerade-cidrs`, `kube-proxy` will skip SNAT entirely. The packet will be DNATed to the Pod IP and routed natively. If the CNI (e.g., Cilium) sets up proper source-IP routing for Pod IPs, the packet correctly inherits the node's primary IP natively.

### 2. Kube-Proxy SNAT to Primary Node IP

To address cases where masquerading cannot be bypassed, `kube-proxy` will be enhanced to optionally configure SNAT directly to the node's primary `InternalIP`. This prevents the kernel from defaulting the source IP to a bridge interface IP. This was explored in #138360 but requires formalization as part of the holistic source IP strategy.

### 3. Clearer Recommended Behaviors for CNIs

We will update the Kubernetes networking documentation to establish a clear "Recommended Behavior" for node-to-pod communication. The recommendation will state that CNI plugins should ensure the node's primary IP is used as the source IP for traffic originating from the node to local or remote pods. This bridges the gap for `NetworkPolicy` portability without tightening rigid conformance rules retroactively.
