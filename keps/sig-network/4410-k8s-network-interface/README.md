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
markedly different approaches. This fragmentation and issues with the API have
negatively impacted adoption in sectors such as telecommunications. Our goal is
to transform Kubernetes networking by making networks and their components
actual resources within the Kubernetes API. This will allow for the development
of shared functionalities and their integration into the API. We anticipate that
this new approach will enhance support for areas that are currently struggling,
facilitate the development and promotion of common features, and better define
and accommodate advanced functionalities and potential areas for expansion.

### Goals

1. Design a cool looking t-shirt
2. Provide Kubernetes APIs for the creation, configuration and management of networks (e.g. `Pod` networks)
3. Provide documentation, examples, troubleshooting and FAQ's for KNI.
4. Establish feature parity with current CNI [ADD, DEL]
5. Handle support levels like Gateway API (e.g. "core" and "extended")
6. Handle implementation-specific use cases through extension points
7. Decouple the Pod and Node Network setup
8. Simplify/enable triggering garbage collection to ensure no resources are left behind
9. Provide the ability to identify the IP address family without parsing the value (such as a field)
10. Provide as much backwards-compatibility with CNI as is feasible
11. Guarantee the network is setup and in a healthy state before containers are started (ephemeral, init, regular)
12. If feasible, provide API awareness of Pod network namespaces (e.g. interface names)
13. Provide support for Kata and other virtualized runtimes
14. Provide a reference implementation

### Non-Goals

1. Any changes to the kube-scheduler 
2. Any specific implementation other than the reference implementation. However we should ensure the KNI-API is flexible enough to support

## Proposal

The proposal of this KEP is to design and implement the KNI-API and make necessary changes to the CRI-API and container runtimes. The scope should be kept to a minimum and we should target feature parity. 

### User Stories

#### Story 1

As a cluster operator, I need the ability to determine my network(s) is ready so that my pods come up with a working network.

#### Story 2

As a cluster operator, I need the ability to determine what networks are available on my node so that upstream components can ensure the pod is scheduled on the appropriate node. 

#### Story 3

As a Kubernetes developer, I need the ability to have extension points for pod network setup, teardown and update so that I can support future Kubernetes networking features with either reducing the changes to core kubernetes or eliminating them

### Notes/Constraints/Caveats

Changes to the pod specification will require hard evidence. 

The specifics of "Network Readiness" is an implementation detail. We need to provide this RPC to the user. 

We should consider the trade offs to using a Native K8s Network object or CRD's.
Using a native object would allow passing a slice of network type to AttachNetwork

Since the network runtime can be run separated from the container runtime, you can package everything into a pod and not need to have binaries on disk. This allows the CNI plugins to be isolated in the pod and the pod will never need to mount /opt/cni/bin or /etc/cni/net.d. This offers a potentially more ability to control execution. Keep in mind CNI is the implementation however when this is used chaining is still available. 
