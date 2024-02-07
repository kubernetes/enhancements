# KEP-4410: Kubernetes Networking reImagined

> **NOTE**: for the initial PR we've removed a lot of the templated text and
> aimed to keep this first iteration small and easier to consume. We are only
> focusing on the "What" and "Why" (e.g. motivation, goals, user stories) for
> this iteration so that we can build consensus on those first before we add
> any of the "How".

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
<!-- /toc -->

## Summary

This proposal is to design and implement the KNI [Kubernetes Networking Interface] or better known as Kubernetes Networking reImagined. KNI will create a Network resource and provide an API that will provide network status, availability, how to attach a pod to a network, detach the pod from the network and update a pods network.  

## Motivation

Kubernetes networking has traditionally been challenging to understand for users
interacting with the Kubernetes API, and there has been considerable flexibility
in how Container Network Interfaces (CNIs) set up networking within clusters.
This has resulted in a scenario where things like pod networking (including pod
to pod networking) is opaque to users, with different implementations taking
markedly different approaches. This fragmentation has spread networking across
all layers of the stack which include k8s components like kube-proxy, netpol agents,
container runtime with CNI plugins and low level runtimes like kata and issues
with the API have negatively impacted adoption in sectors such as telecommunications.
Our goal is to transform Kubernetes networking by making networks and their components
actual resources within the Kubernetes API. This will allow for the development
of shared functionalities and their integration into the API. We anticipate that
this new approach will enhance support for areas that are currently struggling,
facilitate the development and promotion of common features, and better define
and accommodate advanced functionalities and potential areas for expansion.

### Goals

- Design a cool looking t-shirt
- Provide Kubernetes APIs for the creation, configuration and management of interfaces
- Provide documentation, examples, troubleshooting and FAQ's for KNI.
- KNI should provide the API's required to establish feature parity with current CNI [ADD, DEL]
- Handle support levels like Gateway API (e.g. "core" and "extended")
- Handle implementation-specific use cases through extension points
- Decouple the Pod and Node Network setup
- Provide garbage collection to ensure no resources created during pod setup such as Linux bridges, ebpf programs, 
allocated IP addresses are left behind after pod deletion
- Improve the current IP handling for pods (PodIP) to be handle multiple IP addresses and 
a field to identify the IP address family (IPV4 vs IPV6)
- Provide backwards compatibility for the existing CNI approach and migration a path to fully adopt KNI
- Guarantee the network is setup and in a healthy state before containers are started (ephemeral, init, regular)
- If feasible, provide API awareness of Pod network namespaces (e.g. interface names)
- Provide a uniform approach for network setup/teardown for both virtualized (kata) and non-virtualized (runc) 
runtimes including kubevirt. This could eliminate the high and low level runtimes from the networking path
- Provide a reference implementation of the KNI network runtime
- Provide the ability to have all the dependencies packaged in the container image (no more CNI binaries in the host file system)
..- No more downloading CNI binaries via initContainers/Mounting /etc/cni/net.d or /opt/cni/bin
- Provide the ability to use native k8s resources for configuration such as a ConfigMap's instead of configuration files in host file system
- Provide an API to indicate network readiness for the node (no more files on disk)
- Eliminate the need to exec binaries and replace with gRPC
- Make troubleshooting easier by having logs accessible via kubectl logs
- Improve network pod startup time
- Provide the ability to prevent additional scheduling of pods if IPAM is out of IP addresses without evicting running pods

### Non-Goals

1. Any changes to the kube-scheduler 
2. Any specific implementation other than the reference implementation. However we should ensure the KNI-API is flexible enough to support

## Proposal

The proposal of this KEP is to design and implement the KNI-API and make necessary changes to the CRI-API and container runtimes. The scope should be kept to a minimum and we should target feature parity. 

### User Stories

We are constantly adding these user stories, please join the community sync to discuss. 

#### Story 1

As a cluster operator, I need the ability to determine my network(s) is ready so that my pods come up with a working network.

#### Story 2

As a cluster operator, I need the ability to determine what networks are available on my node so that upstream components can ensure the pod is scheduled on the appropriate node. 

#### Story 3

As a Kubernetes developer, I need the ability to have extension points for pod network setup, teardown and update so that I can support future Kubernetes networking features with either reducing the changes to core kubernetes or eliminating them

#### Story 4

As a tool which manages eBPF programs on a Kubernetes cluster (bpfman,
inspektorgadget), I would like to be able to see the network interfaces of a
`Pod` via the Kubernetes API so that I can attach TC/XDP network programs to
those interfaces based on knowing the Pod name.

### Notes/Constraints/Caveats

Additional Information/Diagrams: https://docs.google.com/document/d/1Gz7iNtJNMI-zKJhaOcI3aflPCx3etJ01JMxzbtvruKk/edit?usp=sharing

Changes to the pod specification will require hard evidence. 

The specifics of "Network Readiness" is an implementation detail. We need to provide this RPC to the user. 

We should consider the trade offs to using a Native K8s Network object or CRD's.
Using a native object would allow passing a slice of network type to AttachNetwork

Since the network runtime can be run separated from the container runtime, you can package everything into a pod and not need to have binaries on disk. This allows the CNI plugins to be isolated in the pod and the pod will never need to mount /opt/cni/bin or /etc/cni/net.d. This offers a potentially more ability to control execution. Keep in mind CNI is the implementation however when this is used chaining is still available. 
