# IPv4/IPv6 Dual-stack

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Implementation Plan](#implementation-plan)
  - [Awareness of Multiple IPs per Pod](#awareness-of-multiple-ips-per-pod)
  - [Required changes to Container Runtime Interface (CRI)](#required-changes-to-container-runtime-interface-cri)
    - [Versioned API Change: PodStatus v1 core](#versioned-api-change-podstatus-v1-core)
      - [Default Pod IP Selection](#default-pod-ip-selection)
    - [PodStatus Internal Representation](#podstatus-internal-representation)
    - [Maintaining Compatible Interworking between Old and New Clients](#maintaining-compatible-interworking-between-old-and-new-clients)
      - [V1 to Core (Internal) Conversion](#v1-to-core-internal-conversion)
      - [Core (Internal) to V1 Conversion](#core-internal-to-v1-conversion)
  - [Awareness of Multiple NodeCIDRs per Node](#awareness-of-multiple-nodecidrs-per-node)
    - [kubelet Startup Configuration for Dual-Stack Pod CIDRs](#kubelet-startup-configuration-for-dual-stack-pod-cidrs)
    - [kube-proxy Startup Configuration for Dual-Stack Pod CIDRs](#kube-proxy-startup-configuration-for-dual-stack-pod-cidrs)
    - ['kubectl get pods -o wide' Command Display for Dual-Stack Pod Addresses](#kubectl-get-pods--o-wide-command-display-for-dual-stack-pod-addresses)
    - ['kubectl describe pod ...' Command Display for Dual-Stack Pod Addresses](#kubectl-describe-pod--command-display-for-dual-stack-pod-addresses)
  - [Container Networking Interface (CNI) Plugin Considerations](#container-networking-interface-cni-plugin-considerations)
  - [Services](#services)
    - [Type NodePort, LoadBalancer, ClusterIP](#type-nodeport-loadbalancer-clusterip)
      - [Service Object Mutability Rules](#service-object-mutability-rules)
      - [Impact on pre-existing Services](#impact-on-pre-existing-services)
      - [Creating a New Single-Stack Service](#creating-a-new-single-stack-service)
      - [Creating a New Dual-Stack Service](#creating-a-new-dual-stack-service)
      - [NodePort Allocations](#nodeport-allocations)
    - [Type Headless services](#type-headless-services)
    - [Type ExternalName](#type-externalname)
  - [Endpoints](#endpoints)
  - [kube-proxy Operation](#kube-proxy-operation)
    - [kube-proxy Startup Configuration Changes](#kube-proxy-startup-configuration-changes)
      - [Multiple cluster CIDRs configuration](#multiple-cluster-cidrs-configuration)
  - [CoreDNS Operation](#coredns-operation)
  - [Ingress Controller Operation](#ingress-controller-operation)
    - [GCE Ingress Controller: Out-of-Scope, Testing Deferred For Now](#gce-ingress-controller-out-of-scope-testing-deferred-for-now)
    - [NGINX Ingress Controller - Dual-Stack Support for Bare Metal Clusters](#nginx-ingress-controller---dual-stack-support-for-bare-metal-clusters)
  - [Load Balancer Operation](#load-balancer-operation)
  - [Cloud Provider Plugins Considerations](#cloud-provider-plugins-considerations)
    - [Multiple cluster CIDRs configuration](#multiple-cluster-cidrs-configuration-1)
  - [Container Environment Variables](#container-environment-variables)
  - [Kubeadm Support](#kubeadm-support)
    - [Kubeadm Configuration Options](#kubeadm-configuration-options)
    - [Kubeadm-Generated Manifests](#kubeadm-generated-manifests)
  - [vendor/github.com/spf13/pflag](#vendorgithubcomspf13pflag)
  - [End-to-End Test Support](#end-to-end-test-support)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [Dual-stack at the Edge](#dual-stack-at-the-edge)
  - [Variation: Single-Stack Service CIDRs](#variation-single-stack-service-cidrs)
    - [Changes Required](#changes-required)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
<!-- /toc -->

## Summary

This proposal adds IPv4/IPv6 dual-stack functionality to Kubernetes clusters.
This includes the following concepts:
- Awareness of multiple IPv4/IPv6 address assignments per pod
- Awareness of multiple IPv4/IPv6 address assignments per service
- Native IPv4-to-IPv4 in parallel with IPv6-to-IPv6 communications to, from,
  and within a cluster

## Motivation

The adoption of IPv6 has increased in recent years, and customers are
requesting IPv6 support in Kubernetes clusters. To this end, the support of
IPv6-only clusters was added as an alpha feature in Kubernetes Version 1.9.
Clusters can now be run in either IPv4-only, IPv6-only, or in a
"single-pod-IP-aware" dual-stack configuration. This "single-pod-IP-aware"
dual-stack support is limited by the following restrictions:
- Some CNI network plugins are capable of assigning dual-stack addresses on a
  pod, but Kubernetes is aware of only one address per pod.
- Kubernetes system pods (api server, controller manager, etc.) can have only
  one IP address per pod, and system pod addresses are either all IPv4 or all
  IPv6.
- Endpoints for services are either all IPv4 or all IPv6 within a cluster.
- Service IPs are either all IPv4 or all IPv6 within a cluster.

For scenarios that require legacy IPv4-only clients or services (either
internal or external to the cluster), the above restrictions mean that complex
and expensive IPv4/IPv6 transition mechanisms (e.g. NAT64/DNS64, stateless
NAT46, or SIIT/MAP) will need to be implemented in the data center networking.

One path to adding transition mechanisms would be to modify Kubernetes to
provide support for IPv4 and IPv6 communications in parallel, for both pods and
services, throughout the cluster (a.k.a. "full" dual-stack).

A second, simpler alternative, which is a variation to the "full" dual-stack
model, would be to provide dual-stack addresses for pods and nodes, but
restrict service IPs to be single-family (i.e. allocated from a single service
CIDR). In this case, service IPs in a cluster would be either all IPv4 or all
IPv6, as they are now. Please refer to the "Variation: Single-Stack Service
CIDRs" section under "Alternatives" below).

This proposal aims to add "full" dual-stack support to Kubernetes clusters,
providing native IPv4-to-IPv4 communication and native IPv6-to-IPv6
communication to, from and within a Kubernetes cluster.

### Goals

- Pod Connectivity: IPv4-to-IPv4 and IPv6-to-IPv6 access between pods
- Access to External Servers: IPv4-to-IPv4 and IPv6-to-IPv6 access from pods to
  external servers
- NGINX Ingress Controller Access: Access from IPv4 and/or IPv6 external
  clients to Kubernetes services via the Kubernetes NGINX Ingress Controller.
- Dual-stack support for Kubernetes service IPs, NodePorts, and ExternalIPs
- Functionality tested with the Bridge CNI plugin, PTP CNI plugin, and
  Host-Local IPAM plugins as references
- Maintain backwards-compatible support for IPv4-only and IPv6-only clusters

### Non-Goals

- Cross-family connectivity: IPv4-to-IPv6 and IPv6-to-IPv4 connectivity is
  considered outside of the scope of this proposal. (As a possible future
  enhancement, the Kubernetes NGINX ingress controller could be modified to
  load balance to both IPv4 and IPv6 addresses for each endpoint. With such a
  change, it's possible that an external IPv4 client could access a Kubernetes
  service via an IPv6 pod address, and vice versa).
- CNI network plugins: Some plugins other than the Bridge, PTP, and Host-Local
  IPAM plugins may support Kubernetes dual-stack, but the development and
  testing of dual-stack support for these other plugins is considered outside
  of the scope of this proposal.
- Multiple IPs within a single family: Code changes will be done in a way to
  facilitate future expansion to more general multiple-IPs-per-pod and
  multiple-IPs-per-node support. However, this initial release will impose
  "dual-stack-centric" IP address limits as follows:
  - Pod addresses: 1 IPv4 address and 1 IPv6 addresses per pod maximum
  - Service addresses: 1 IPv4 address and 1 IPv6 addresses per service maximum
- External load balancers: Load-balancer providers will be expected to update
  their implementations.
- Dual-stack support for Kubernetes orchestration tools other than kubeadm
  (e.g. miniKube, KubeSpray, etc.) are considered outside of the scope of this
  proposal. Communication about how to enable dual-stack functionality will be
  documented appropriately in-order so that aformentioned tools may choose to
  enable it for use.

## Proposal

In order to support dual-stack in Kubernetes clusters, Kubernetes needs to have
awareness of and support dual-stack addresses for pods and nodes. Here is a
summary of the proposal (details follow in subsequent sections):

- Kubernetes needs to be made aware of multiple IPs per pod (limited to one
  IPv4 and one IPv6 address per pod maximum).
- Link Local Addresses (LLAs) on a pod will remain implicit (Kubernetes will
  not display nor track these addresses).
- Service Cluster IP Range `--service-cluster-ip-range=` will support the
  configuration of one IPv4 and one IPV6 address block).
- Service IPs will be allocated from one or both IP families as requested by
  the Service `spec` OR from the first configured address range if the service
  expresses nothing about IP families.
- Backend pods for a service can be dual-stack.
- Endpoints (not EndpointSlice) addresses will match the first address family
  allocated to the Service (eg. An IPv6 Service IP will only have IPv6
  Endpoints)
- EndpointSlices will support endpoints of both IP families.
- Kube-proxy iptables mode needs to drive iptables and ip6tables in parallel.
  Support includes:
  - Service IPs: IPv4 and IPv6 family support
  - NodePort: Support listening on both IPv4 and IPv6 addresses
  - ExternalIPs: Can be IPv4 or IPv6
- Kube-proxy IPVS mode will support dual-stack functionality similar to
  kube-proxy iptables mode as described above. IPVS kube-router support for
  dual-stack, on the other hand, is considered outside of the scope of this
  proposal.
- The pod status API changes will leave room for per-IP metadata for future
  Kubernetes enhancements.  The metadata and enhancements themselves are out of
  scope.
- Kubectl commands and output displays will need to be modified for dual-stack.
- Kubeadm support will need to be added to enable spin-up of dual-stack
  clusters. Kubeadm support is required for implementing dual-stack continuous
  integration (CI) tests.
- New e2e test cases will need to be added to test parallel IPv4/IPv6
  connectivity between pods, nodes, and services.

### Implementation Plan

Given the scope of this enhancement, it has been suggested that we break the
implementation into discrete phases that may be spread out over the span of
many release cycles. The phases are as follows:

Phase 1 (Kubernetes 1.16)
- API type modifications
   - Kubernetes types
   - CRI types
- dual-stack pod networking (multi-IP pod)
- kubenet multi-family support

Phase 2 (Kubernetes 1.16)
- Multi-family services including kube-proxy
- Working with a CNI provider to enable dual-stack support
- Update component flags to support multiple `--bind-address`

Phase 3 (Planned Kubernetes 1.17)
- Kube-proxy iptables support for dual-stack
- Downward API support for Pod.Status.PodIPs
- Test e2e CNI plugin support for dual-stack
- Pod hostfile support for IPv6 interface address
- Kind support for dual-stack
- e2e testing
- EndpointSlice API support in kube-proxy for dual-stack (IPVS and iptables)
- Dual-stack support for kubeadm
- Expand container runtime support (containerd, CRI-O)

Phase 4 and beyond
- Adapting to feedback from earlier phases
- External dependencies, eg. cloud-provider, CNI, CRI, CoreDNS etc...

### Awareness of Multiple IPs per Pod

Since Kubernetes Version 1.9, Kubernetes users have had the capability to use
dual-stack-capable CNI network plugins (e.g. Bridge + Host Local, Calico,
etc.), using the [0.3.1 version of the CNI Networking Plugin
API](https://github.com/containernetworking/cni/blob/spec-v0.3.1/SPEC.md), to
configure multiple IPv4/IPv6 addresses on pods. However, Kubernetes currently
captures and uses only IP address from the pod's main interface.

This proposal aims to extend the Kubernetes Pod Status and the Pod Network
Status API so that Kubernetes can track and make use of one or many IPv4
addresses and one or many IPv6 address assignment per pod.

CNI provides list of IPs and their versions. Kubernetes currently just chooses
to ignore this and use a single IP. This means this struct will need to follow
the pod.Pod IP style of primary IP as is, and another slice of IPs, having
Pod.IPs[0] == Pod.IP which will look like the following:

```
    type PodNetworkStatus struct {
      metav1.TypeMeta `json:",inline"`

      // IP is the primary IPv4/IPv6 address of the pod. Among other things it is the address that -
      //   - kube expects to be reachable across the cluster
      //   - service endpoints are constructed with
      //   - will be reported in the PodStatus.PodIP field (will override the IP reported by docker)
      IP net.IP `json:"ip" description:"Primary IP address of the pod"`
            IPs  []net.IP `json:"ips" description:"list of ips"`
    }
```

CNI as of today does not provide additional metadata to the IP. So this
Properties field - speced in this KEP - will be empty until CNI spec includes
properties. Although in theory we can copy the interface name, gateway etc.
into this Properties map.

For health/liveness/readiness probe support, the default behavior will not
change and [the default PodIP](#Default Pod IP Selection) will be used in case
that no host will be passed as a parameter to the probe. This will simplify the
logic in favor of avoiding false positives or negatives on the health checks,
with the disadvantage that it will not be able to use secondary IPs.
This means that applications must always listen in the Pod default IP in
order to use healthchecks.

### Required changes to Container Runtime Interface (CRI)

The PLEG loop + status manager of kubelet makes an extensive use of
PodSandboxStatus call to wire up PodIP to api server, as a patch call to Pod
Resources. The problem with this is the response message wraps
PodSandboxNetworkStatus struct and this struct only carries one IP. This will
require a change similar to the change described above. We will work with the
CRI team to coordinate this change.

```
    type PodSandboxNetworkStatus struct {
      // IP address of the PodSandbox.
      Ip string `protobuf:"bytes,1,opt,name=ip,proto3" json:"ip,omitempty"`
            Ips []string `protobuf:"bytes,1,opt,name=ip,proto3" json:"ip,omitempty"`
    }
```

The Kubelet must respect the the order of the IPs returned by the runtime.

#### Versioned API Change: PodStatus v1 core

In order to maintain backwards compatibility for the core V1 API, this proposal
retains the existing (singular) "PodIP" field in the core V1 version of the
[PodStatus V1 core
API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#podstatus-v1-core),
and adds a new array of structures that store pod IPs along with associated
metadata for that IP. The metadata for each IP (refer to the "Properties" map
below) will not be used by the dual-stack feature, but is added as a
placeholder for future enhancements, e.g. to allow CNI network plugins to
indicate to which physical network that an IP is associated. Retaining the
existing "PodIP" field for backwards compatibility is in accordance with the
[Kubernetes API change
quidelines](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md).

```
    // Default IP address allocated to the pod. Routable at least within the
    // cluster. Empty if not yet allocated.
    PodIP string `json:"podIP,omitempty" protobuf:"bytes,6,opt,name=podIP"`

    // IP address information for entries in the (plural) PodIPs slice.
    // Each entry includes:
    //    IP: An IP address allocated to the pod. Routable at least within
    //        the cluster.
    type PodIP struct {
        IP string
    }

    // IP addresses allocated to the pod. This list
    // is inclusive, i.e. it includes the default IP address stored in the
    // "PodIP" field, and this default IP address must be recorded in the
    // 0th entry (PodIPs[0]) of the slice. The list is empty if no IPs have
    // been allocated yet.
    PodIPs []PodIP `json:"podIPs,omitempty" protobuf:"bytes,6,opt,name=podIPs"`
```

##### Default Pod IP Selection

Older servers and clients that were built before the introduction of full
dual-stack will only be aware of and make use of the original, singular PodIP
field above. It is therefore considered to be the default IP address for the
pod. When the PodIP and PodIPs fields are populated, the PodIPs[0] field must
match the (default) PodIP entry. If a pod has both IPv4 and IPv6 addresses
allocated, then the IP address chosen as the default IP address will match the
IP family of the cluster's configured service CIDR. For example, if the service
CIDR is IPv4, then the IPv4 address will be used as the default address.

#### PodStatus Internal Representation

The PodStatus internal representation will be modified to use a slice of
PodIP structs rather than a singular IP ("PodIP"):

```
    // IP address information. Each entry includes:
    //    IP: An IP address allocated to the pod. Routable at least within
    //        the cluster.
    // Empty if no IPs have been allocated yet.
    type PodIP struct {
        IP string
    }

    // IP addresses allocated to the pod with associated metadata.
    PodIPs []PodIP `json:"podIPs,omitempty" protobuf:"bytes,6,opt,name=podIPs"`
```

This internal representation should eventually become part of a versioned API
(after a period of deprecation for the singular "PodIP" field).

#### Maintaining Compatible Interworking between Old and New Clients

Any Kubernetes API change needs to consider consistent interworking between a
possible mix of clients that are running old vs. new versions of the API. In
this particular case, however, there is only ever one writer of the PodStatus
object, and it is the API server itself. Therefore, the API server does not
have an absolute requirement to implement any safeguards and/or fixups between
the singular PodIP and the plural PodIPs fields as described in the guidelines
for pluralizing singular API fields that is included in the [Kubernetes API
change
quidelines](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md).

However, as a defensive coding measure and for future-proofing, the following
API version translation logic will be implemented for the PodIP/PodIPs fields:

##### V1 to Core (Internal) Conversion

- If only V1 PodIP is provided:
  - Copy V1 PodIP to core PodIPs[0]
- Else if only V1 PodIPs[] is provided: # Undetermined as to whether this can actually happen (@thockin)
  - Copy V1 PodIPs[] to core PodIPs[]
- Else if both V1 PodIP and V1 PodIPs[] are provided:
  - Verify that V1 PodIP matches V1 PodIPs[0]
  - Copy V1 PodIPs[] to core PodIPs[]
- Delete any duplicates in core PodIPs[]

##### Core (Internal) to V1 Conversion

  - Copy core PodIPs[0] to V1 PodIP
  - Copy core PodIPs[] to V1 PodIPs[]

### Awareness of Multiple NodeCIDRs per Node

As with PodIP, corresponding changes will need to be made to NodeCIDR. These
changes are essentially the same as the aformentioned PodIP changes which
create the pularalization of NodeCIDRs to a slice rather than a singular and
making those changes across the internal representation and v1 with associated
conversations.

#### kubelet Startup Configuration for Dual-Stack Pod CIDRs

The existing "--pod-cidr" option for the [kubelet startup
configuration](https://kubernetes.io/docs/reference/command-line-tools-reference/kubelet/)
will be modified to support multiple IP CIDRs in a comma-separated list (rather
than a single IP string), i.e.:

```
  --pod-cidr  ipNetSlice   [IP CIDRs, comma separated list of CIDRs, Default: []]
```

Only one CIDR or two CIDRs, one of each IP family, will be used; other cases
will be invalid.

#### kube-proxy Startup Configuration for Dual-Stack Pod CIDRs

The existing "cluster-cidr" option for the [kube-proxy startup
configuration](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/)
will be modified to support multiple cluster CIDRs in a comma-separated list
(rather than a single IP string), i.e:

```
  --cluster-cidr  ipNetSlice   [IP CIDRs, comma separated list of CIDRs, Default: []]
```

Only one CIDR or two CIDRs, one of each IP family, will be used; other cases
will be invalid.

#### 'kubectl get pods -o wide' Command Display for Dual-Stack Pod Addresses

The output for the 'kubectl get pods -o wide' command will not need
modification and will only display the primary pod IP address as determined by
the first IP address block configured via the `--cluster-cidr=` on
kube-controller-manager. eg. The following is expected output for a cluster is
configured with an IPv4 address block as the first configured via the
`--cluster-cidr=` on kube-controller-manager:

```
       kube-master# kubectl get pods -o wide
       NAME               READY     STATUS    RESTARTS   AGE       IP                          NODE
       nginx-controller   1/1       Running   0          20m       10.244.2.7                  n01
       kube-master#
```

For comparison, here expected output for a cluster is configured with an IPv6
address block as the first configured via the `--cluster-cidr=` on
kube-controller-manager:

```
       kube-master# kubectl get pods -o wide
       NAME               READY     STATUS    RESTARTS   AGE       IP                          NODE
       nginx-controller   1/1       Running   0          20m       fd00:db8:1::2               n01
       kube-master#
```

#### 'kubectl describe pod ...' Command Display for Dual-Stack Pod Addresses

The output for the 'kubectl describe pod ...' command will need to be modified
to display a list of IPs for each pod, e.g.:

```
       kube-master# kubectl describe pod nginx-controller
       .
       .
       .
        IP:           10.244.2.7
        IPs:
          IP:           10.244.2.7
          IP:           fd00:200::7
       .
       .
       .
```

### Container Networking Interface (CNI) Plugin Considerations

This feature requires the use of the [CNI Networking Plugin API version
0.3.1](https://github.com/containernetworking/cni/blob/spec-v0.3.1/SPEC.md) or
later. The dual-stack feature requires no changes to this API.

The versions of CNI plugin binaries that must be used for proper dual-stack
functionality (and IPv6 functionality in general) depend upon the version of
the container runtime that is used in the cluster nodes (see [CNI issue
#531](https://github.com/containernetworking/cni/issues/531) and [CNI plugins
PR #113](https://github.com/containernetworking/plugins/pull/113)):

### Services

Services depend on `--service-cluster-ip-range` for `ClusterIP` assignment. A
dual-stack cluster can have this flag configured as either
`--services-cluster-ip-range=<cidr>,<cidr>` or
`--services-cluster-ip-range=<cidr>`. The following rules apply:

1. The maximum allowed number of CIDRs in this flag is two. Attempting to use
   more than two CIDRs will generate failures during startup.
2. When configured with two CIDRs then the values are expected to have
   different IP families. e.g
   `--service-cluster-ip-range=<ipv4-cidr>,<ipv6-cidr>` or
   `--service-cluster-ip-range=<ipv6-cidr>,<ipv4-cidr>`. Attempting to
   configure a cluster with two CIDRs of the same family will generate failures
   during startup.
3. It is possible to have a "partially-dual-stack" cluster, with dual-stack pod
   networking but only a single service CIDR. This means that pods in the
   cluster will be able to send and receive traffic on both IP families, but
   services which use the `clusterIP` or `ClusterIPs` field will only be able
   to use the IP family of this flag.
4. When upgrading a cluster to dual-stack, the first element of the
   `--service-cluster-ip-range` flag MUST NOT change IP families.  As an
   example, an IPv4 cluster can safely be upgraded to use
   `--service-cluster-ip-range=<ipv4-cidr>,<ipv6-cidr>`, but can not safely be
   changed to `--service-cluster-ip-range=<ipv6-cidr>,<ipv4-cidr>`.
5. The first CIDR entry of `--service-cluster-ip-range` is considered the
   cluster's default service address family.
6. Attempting to allocate `ClusterIP` or `ClusterIPs` from an IP address family
   that is not configured on the cluster will result in an error.

> The below discussion assumes that `EndpointSlice and EndpointSlice
> Controller`  will allow dual-stack endpoints, the existing Endpoints
> controller will remain single-stack. That means enabling dual-stack services
> by having more than one entry in `--service-cluster-ip-range` without
> enabling `EndpointSlice` feature gate will result in validation failures
> during startup.

#### Type NodePort, LoadBalancer, ClusterIP

In a dual-stack cluster, Services can be:
1. Single-Stack: service IP are allocated (or reserved, if provided by user)
   from the first or the second CIDR (if configured) entry as configured using
   `--service-cluster-ip-range` flag, in this scenario services will have a
   single allocated `ClusterIP` (also set in `ClusterIPs`).
2. Dual-Stack (optional): service IP is allocated (or reserved, if provided by
   user) from both the first and the second CIDR (if configured) entries. In
   this scenario services will have one or two assigned `ClusterIPs`. If the
   cluster is not configured for dual-stack then  the resulting service will be
   created as a single-stack and `ClusterIP/ClusterIPs` will be assigned from a
   single family.
3. Dual-Stack (required): if the cluster is not configured for dual-stack then
   service creation will fail. If the cluster is configured for dual-stack the
   resulting service will carry two `ClusterIPs`.

The above is achieved using the following changes to Service api.
1. Add a `spec.ipFamilyPolicy` optional field with `SingleStack`, `PreferDualStack`,
   and `RequireDualStack` as possible values.
2. Add a `spec.ipFamilies[]` optional field with possible values of
   `nil`, `["IPv4"]`, `["IPv6"]`, `["IPv4", "IPv6"]`, or `["IPv6", "IPv4"]`.
   This field identifies the desired IP families of a service. The apiserver
   will reject the service creation request if the specified IP family is not
   supported by the cluster. Further dependency and validation on this field is
   described below.
3. Add a `spec.clusterIPs[]` field that follows the same rules as
   `status.PodIPs/status.PodIP`. Maximum entries in this array is two entries
   of two different IP address families. Semantically `spec.clusterIP` will be
   referred as the `Primary Service IP`.

> We expect most users to specify as little as possible when creating a
> Service, but they may specify exact IP families or even IP addresses.
> Services are expected to be internally consistent whenever values for the
> optional fields are provided. For example, a service which requests
>  `spec.ipFamilies : ["IPv4"]` and `spec.clutserIPs: ["<ipv6>" ]` is
>  considered invalid because `ipFamilies[0]` does not match `ClusterIPs[0]`.

The rules that govern these fields are described in the below sections

##### Service Object Mutability Rules

The existing mutability rules will remain as is. Services can not
change their `clusterIP` (aka `clusterIPs[0]`) - this is considered as
`Service Primary ClusterIP`, Services can still change `type`s as before (e.g.
from `ClusterIP` to `ExternalName`). The following are additional immutability
rules

1. Services can change from `Single-Stack` to `Dual-Stack (required)` or `Dual
   Stack (optional)` as described below.
2. Services can also change `Dual-Stack` to `Single-Stack`.
3. Services can not change the primary service IP `spec.clusterIP` or
   `spec.clusterIPs[0]` because this will break existing mutability rules. This
   also apply even changing a service from `Single-Stack` to `Dual-Stack` or
   the other way around.

##### Impact on pre-existing Services

Services which existed before this feature will not be changed, and when read
through an updated apiserver will appear as single-stack Services, with
`spec.ipFamilies` set to the cluster's default service address family. The above applies
on all types of services except

1. Headless services with no selector: they will be defaulted to `spec.ipFamilyPolicy: PreferDualStack`
and `spec.IPFamilies[ 'IPv4', 'IPv6']`.
2. `ExternalName` services: they will be defaulted to `nil` to all the new fields

##### Creating a New Single-Stack Service

Users who want to create a new single-stack Service can create it using one of
the following methods (in increasing specificity):
1. Do not set `spec.ipFamilyPolicy`, `spec.ipFamilies`, or `spec.clusterIPs`
   fields. The apiserver will default `spec.ipFamilyPolicy` to `SingleStack`, will
   assign `spec.ipFamilies` to the cluster's default IP family, and will
   allocate a `clusterIP` of that same family.
2. Set `spec.ipFamilyPolicy:` to `SingleStack` and do not set `spec.ipFamilies`, or
   `spec.clusterIPs`.  This is the same as case 1, above
3. Either do not set `spec.ipFamilyPolicy` or set it to `SingleStack` and
   `spec.ipFamilies` to a one-element list containing a valid IP family value.
   The apiserver will default `spec.ipFamilyPolicy: SingleStack`. And it will try to
   allocate a `clusterIP` of the specified family.  It will fail the
   request if the requested IP family is not configured on the cluster.
4. Either do not set `spec.ipFamilyPolicy` or set it to `SingleStack`, either do not
   set `spec.ipFamilies` or set to to a valid IP family value, and set
   `spec.clusterIPs` to a valid IP address.  Assuming that all specified values
   are internally consistent, the apiserver will default `spec.ipFamilyPolicy:
   SingleStack` and `spec.ipFamilies: ['family-of-clusterIPs[0]']`. It will fail the
   request if the IP family is not configured on the cluster, if the IP is out
   of CIDR range, or if the IP is already allocated to a different service.

The below are examples that follow the above rules.

Given a cluster that is configured with --service-cluster-ip-range=<ipv4-cidr>
or --service-cluster-ip-range=<ipv4-cidr>,<ipv6-cidr>:

This input:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
```

...produces this result:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
  ipFamilyPolicy: SingleStack # defaulted
  ipFamilies:                 # defaulted
    - IPv4
  clusterIP: 1.2.3.4          # allocated
  clusterIPs:                 # allocated
    - 1.2.3.4
```

If that cluster were configured for IPv6-only (single-stack) or IPv6 first
(dual-stack) it would have resulted in:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
  ipFamilyPolicy: SingleStack # defaulted
  ipFamilies:                 # defaulted
    - IPv6
  clusterIP: 2001::1          # allocated
  clusterIPs:                 # allocated
    - 2001::1
 ```

A user can be more prescriptive about which IP family they want.  This input:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
  ipFamilies: # set by user
    - IPv6
```

...produces this result, assuming the cluster has IPv6 available:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
  ipFamilyPolicy: SingleStack # defaulted
  ipFamilies:                 # set by user
    - IPv6
  clusterIP: 2001::1          # allocated
  clusterIPs:
    - 2001::1
```

> It is assumed that any service created with `spec.ipFamilyPolicy: nil` is
> single-stack. Users can change this behavior using admission control web
> hooks, if they want to default services to dual-stack.

##### Creating a New Dual-Stack Service

Users can create-dual stack services according to the following methods (in
increasing specificity):
- If the user *prefers* dual stack (if available, service creation will not fail if
  the cluster is not configured for dual stack) then they can do one of the
  following:
  1. Set `spec.ipFamilyPolicy` to `PreferDualStack` and do not set `spec.ipFamilies` or
     `spec.clusterIPs`. The apiserver will set `spec.ipFamilies` according to
     how the cluster is configured, and will allocate IPs according to those
     families.
  2. Set `spec.ipFamilyPolicy` to `PreferDualStack`, set `spec.ipFamilies` to one IP
     family, and do not set `spec.clusterIPs`. The apiserver will set
     `spec.ipFamilies` to the requested family if it is available (otherwise it
     will fail) and the alternate family if dual-stack is configured, and will
     allocate IPs according to those families.
  3. Set `spec.ipFamilyPolicy` to `PreferDualStack`, either do not set `spec.ipFamilies`
     or set it to one IP family, and set `spec.clusterIPs` to one IP.
     The apiserver will default `spec.ipFamilies` to the family of
     `clusterIPs[0]` and the alternate family if dual-stack is configured, will
     reserve the specified IP if possible, and will allocate a second IP of the
     alternate family if dual-stack is configured.
- If the user *requires* dual-stack service (service creation will fail if cluster
  is not configured for dual-stack) then they can do one of the following:
  1. set `spec.ipFamilyPolicy` to `RequireDualStack` and apiserver will default
     `spec.ipFamilies` to ip families configured on cluster and will allocate IPs
    accordingly.
  2. Either don't set `spec.ipFamilyPolicy` or set it to `RequireDualStack`, and set
     `spec.ipFamilies` to the two IP families in any order.
  3. Either don't set `spec.ipFamilyPolicy` or set it to `RequireDualStack`, and set
     `spec.clusterIPs` to any two IPs of different families, in any order.

The below are sample services that demonstrate the above rules.

Given a cluster configured for dual-stack as `<ipv4-cidr>,<ipv6-cidr>`, this
input:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
  ipFamilyPolicy: RequireDualStack # set by user
  ipFamilies:                      # set by user
    - IPv4
    - IPv6
```

...will produce this result:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
  ipFamilyPolicy: RequireDualStack # set by user
  ipFamilies:                      # set by user
    - IPv4
    - IPv6
  clusterIP: 1.2.3.4               # allocated
  clusterIPs:                      # allocated
    - 1.2.3.4
    - 2001::1
```

If the cluster had not been configured for dual-stack, this would have failed.
The user could have specified `ipFamilies` in the alternate order, and the
allocated `clusterIPs` would have switched, too.

This input:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
  ipFamilyPolicy: PreferDualStack # set by user
```

...would produce the following result on a single-stack IPv6 cluster:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
  ipFamilyPolicy: PreferDualStack # set by user
  ipFamilies:                      # defaulted
    - IPv6                         # note: single entry
  clusterIP: 2001::1               # allocated
  clusterIPs:                      # allocated
    - 2001::1                      # note: single entry
```

On an IPv6-IPv4  dual-stack cluster it would have produced:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-service
spec:
  type: ClusterIP
  selector:
    app: MyApp
  ports:
    - protocol: TCP
      port: 80
      targetPort: 9376
  ipFamilyPolicy: PreferDualStack # set by user
  ipFamilies:                      # defaulted
    - IPv6                         # note: two entries
    - IPv4
  clusterIP: 2001::1               # allocated
  clusterIPs:                      # allocated
    - 2001::1                      # note: two entries
    - 1.2.3.4
```

##### NodePort Allocations

NodePort reservations will apply across both families. That means a
reservation of `NodePort: 12345` will happen on both families, irrespective
of the IP family of the service. A single service which is dual-stack may use
the same NodePort on both families.  `kube-proxy` will ensure that traffic is
routed according to the families assigned for services.

#### Type Headless services

1. For Headless with selector, the value of `spec.ipFamilies` will drive the
   endpoint selection. Endpoints will use the value of `ipFamiles[0]`.
   EndpointSlices will use endpoints with all IP families of the Service. This
   value is expected to match cluster configuration. i.e. creating  service with
   a family not configured on server will generate an error.
2. For Headless without selector the values of `spec.ipFamilyPolicy` and
   `spec.ipFamilies` are semantically validated for correctness but are not
   validated against cluster configuration. This means a headless service without
   selectors can have IP families that are not configured on server.

> headless services with no selectors that are not providing values for `ipFamilies`
  and/or `ipfamilyPolicy` apiserver will default to `IPFamilyPolicy: PreferDualStack`
  and `IPFamilies: ["IPv4", "IPv6"]`. If the service have `SingleStack` set to
  `IPFamilyPolicy` the following will be set `IPFamilies: [$cluster_default_family]`

#### Type ExternalName

For ExternalName service the value of `spec.ipFamilyPolicy` and
`spec.ipFamilies` is expected to be `nil`. Any other value for these fields
will cause validation errors. The previous rule does not apply on conversion. apiserver
will automatically set `spec.ipFamilyPolicy` and `spec.ipFamilies` to nil when converting
`ClusterIP` service to `ExternalName` service.

### Endpoints

The current [Kubernetes Endpoints
API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#endpoints-v1-core)
(i.e. before the addition of the dual-stack feature), supports only a single IP
address per endpoint. With the addition of the dual-stack feature, pods serving
as backends for Kubernetes services may now have both IPv4 and IPv6 addresses.
This presents a design choice of how to represent such dual-stack endpoints in
the Endpoints API. Two choices worth considering would be:
- 2 single-family endpoints per backend pod: Make no change to the [Kubernetes
  Endpoints
  API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#endpoints-v1-core).
  Treat each IPv4/IPv6 address as separate, distinct endpoints, and include
  each address in the comma-separated list of addresses in an 'Endpoints' API
  object.
- 1 dual-stack endpoint per backend pod: Modify the [Kubernetes Endpoints
  API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#endpoints-v1-core)
  so that each endpoint can be associated with a pair of IPv4/IPv6 addresses.

Given a phased approach, the 2 single-family endpoint approach represents the
least disruptive change. Services will select only the endpoints that match the
`spec.ipFamily` defined in the Service.

For example, for a Service named `my-service` that has the `spec.ipFamily` set
to IPv4 would only have endpoints only from the IPv4 address family.

```bash
$ kubectl get ep my-service
NAME         ENDPOINTS                                         AGE
my-service   10.244.0.6:9376,10.244.2.7:9376,10.244.2.8:9376   84s
```

Likewise, for a Service named `my-service-v6` that has the `spec.ipFamily` set
to IPv6 would only have endpoints from the IPv6 address family.

```bash
$ kubectl get ep my-service-v6
NAME            ENDPOINTS                                              AGE
my-service-v6   [fd00:200::7]:9376,[fd00:200::8]:9376,[fd00::6]:9376   3m28s
```

### kube-proxy Operation

Kube-proxy will be modified to drive iptables and ip6tables in parallel. This
will require the implementation of a second "proxier" interface in the
Kube-Proxy server in order to modify and track changes to both tables. This is
required in order to allow exposing services via both IPv4 and IPv6, e.g. using
Kubernetes:
  - NodePort
  - ExternalIPs

#### kube-proxy Startup Configuration Changes

##### Multiple cluster CIDRs configuration
The existing "--cluster-cidr" option for the [kube-proxy startup
configuration](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/)
will be modified to support multiple IP CIDRs in a comma-separated list (rather
than a single IP CIDR).  A new [kube-proxy
configuration](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-proxy/)
argument will be added to allow a user to specify multiple cluster CIDRs.

```
  --cluster-cidr  ipNetSlice   (IP CIDRs, in a comma separated list, Default: [])
```

Only one CIDR or two CIDRs, one of each IP family, will be used; other cases
will be invalid.

### CoreDNS Operation

CoreDNS will need to make changes in order to support the plural form of
endpoint addresses. Some other considerations of CoreDNS support for
dual-stack:

- Because service IPs will remain single-family, pods will continue to access
  the CoreDNS server via a single service IP. In other words, the nameserver
  entries in a pod's /etc/resolv.conf will typically be a single IPv4 or single
  IPv6 address, depending upon the IP family of the cluster's service CIDR.
- Non-headless Kubernetes services: CoreDNS will resolve these services to
  either an IPv4 entry (A record) or an IPv6 entry (AAAA record), depending
  upon the IP family of the cluster's service CIDR.
- Headless Kubernetes services: CoreDNS will resolve these services to either
  an IPv4 entry (A record), an IPv6 entry (AAAA record), or both, depending on
  the service's `ipFamily`.

### Ingress Controller Operation

The [Kubernetes ingress
feature](https://kubernetes.io/docs/concepts/services-networking/ingress/)
relies on the use of an ingress controller. The two "reference" ingress
controllers that are considered here are the [GCE ingress
controller](https://github.com/kubernetes/ingress-gce/blob/master/README.md#glbc)
and the [NGINX ingress
controller](https://github.com/kubernetes/ingress-nginx/blob/master/README.md#nginx-ingress-controller).

#### GCE Ingress Controller: Out-of-Scope, Testing Deferred For Now

It is not clear whether the [GCE ingress
controller](https://github.com/kubernetes/ingress-gce/blob/master/README.md#glbc)
supports external, dual-stack access. Testing of dual-stack access to
Kubernetes services via a GCE ingress controller is considered out-of-scope
until after the initial implementation of dual-stack support for Kubernetes.

#### NGINX Ingress Controller - Dual-Stack Support for Bare Metal Clusters

The [NGINX ingress
controller](https://github.com/kubernetes/ingress-nginx/blob/master/README.md#nginx-ingress-controller)
should provide dual-stack external access to Kubernetes services that are
hosted on baremetal clusters, with little or no changes.

- Dual-stack external access to NGINX ingress controllers is not supported with
  GCE/GKE or AWS cloud platforms.
- NGINX ingress controller needs to be run on a pod with dual-stack external
  access.
- On the load balancer (internal) side of the NGINX ingress controller, the
  controller will load balance to backend service pods on a per
  dual-stack-endpoint basis, rather than load balancing on a per-address basis.
  For example, if a given backend pod has both an IPv4 and an IPv6 address, the
  ingress controller will treat the IPv4 and IPv6 address endpoints as a single
  load-balance target. Support of dual-stack endpoints may require upstream
  changes to the NGINX ingress controller.
- Ingress access can cross IP families. For example, an incoming L7 request
  that is received via IPv4 can be load balanced to an IPv6 endpoint address in
  the cluster, and vice versa.

### Load Balancer Operation

External load balancers that rely on Kubernetes services for load balancing
functionality may implement dual-stack support, but are not required to.  Some
implementations will need to instantiate two distinct LBs (e.g. in a cloud).
If the cloud provider load balancer maps directly to the pod IPs then a
dual-stack load balancer could be used. Additional information may need to be
provided to the cloud provider to configure dual-stack. See Implementation Plan
for further details.

### Cloud Provider Plugins Considerations

The [Cloud
Providers](https://kubernetes.io/docs/concepts/cluster-administration/cloud-providers/)
may have individual requirements for dual-stack in addition to below.

#### Multiple cluster CIDRs configuration

The existing "--cluster-cidr" option for the
[cloud-controller-manager](https://kubernetes.io/docs/reference/command-line-tools-reference/cloud-controller-manager/)
will be modified to support multiple IP CIDRs in a comma-separated list (rather
than a single IP CIDR).

```
  --cluster-cidr  ipNetSlice   (IP CIDRs, in a comma separated list, Default: [])
```

Only one CIDR or two CIDRs, one of each IP family, will be used; other cases
will be invalid.

The cloud CIDR allocator will be updated to support allocating from multiple
CIDRs. The route controller will be updated to create routes for multiple
CIDRs.

### Container Environment Variables

The [container environmental
variables](https://kubernetes.io/docs/concepts/containers/container-environment/#container-environment)
will support dual-stack for pod information (downward API) but not for service
information.

The Downward API
[status.podIP](https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/#capabilities-of-the-downward-api)
will preserve the existing single IP address, and will be set to the default IP
for each pod. A new environmental variable named status.podIPs will contain a
space-separated list of IP addresses. The new pod API will have a slice of
structures for the additional IP addresses. Kubelet will translate the pod
structures and return podIPs as a comma-delimited string.

Here is an example of how to define a pluralized `MY_POD_IPS` environmental
variable in a pod definition yaml file:

```
  - name: MY_POD_IPS
    valueFrom:
      fieldRef:
        fieldPath: status.podIPs
```

This definition will cause an environmental variable setting in the pod similar
to the following:

```
MY_POD_IPS=fd00:10:20:0:3::3,10.20.3.3
```

### Kubeadm Support

Dual-stack support will need to be added to kubeadm both for dual-stack
development purposes, and for use in dual-stack continuous integration tests.

- The Kubeadm config options and config file will support dual-stack options
  for apiserver-advertise-address, and podSubnet.

#### Kubeadm Configuration Options

The kubeadm configuration options for advertiseAddress and podSubnet will need
to be changed to handle a comma-separated list of CIDRs:

```
    api:
      advertiseAddress: "fd00:90::2,10.90.0.2" [Multiple IP CIDRs, comma separated list of CIDRs]
    networking:
      podSubnet: "fd00:10:20::/72,10.20.0.0/16" [Multiple IP CIDRs, comma separated list of CIDRs]
```

#### Kubeadm-Generated Manifests

Kubeadm will need to generate dual-stack CIDRs for the
--service-cluster-ip-range command line argument in kube-apiserver.yaml:

```
    spec:
      containers:
      - command:
        - kube-apiserver
        - --service-cluster-ip-range=fd00:1234::/110,10.96.0.0/12
```

Kubeadm will also need to generate dual-stack CIDRs for the --cluster-cidr
argument in kube-apiserver.yaml:

```
    spec:
      containers:
      - command:
        - kube-controller-manager
        - --cluster-cidr=fd00:10:20::/72,10.20.0.0/16
```

### vendor/github.com/spf13/pflag

This dual-stack proposal will introduce a new IPNetSlice object to spf13.pflag
to allow parsing of comma separated CIDRs. Refer to
[https://github.com/spf13/pflag/pull/170](https://github.com/spf13/pflag/pull/170)

### End-to-End Test Support

End-to-End tests will be updated for dual-stack. The dual-stack e2e tests will
utilized kubernetes-sigs/kind along with supported cloud providers. The e2e
test-suite for dual-stack is running
[here](https://testgrid.k8s.io/sig-network-dualstack-azure-e2e#dualstack-azure-e2e).
Once dual-stack support is added to kind, corresponding dual-stack e2e tests
will be run on kind similar to
[this](https://testgrid.k8s.io/sig-release-master-blocking#kind-ipv6-master-parallel).

The E2E test suite that will be run for dual-stack will be based upon the
[IPv6-only test suite](https://github.com/CiscoSystems/kube-v6-test) as a
baseline. New versions of the network connectivity test cases that are listed
below will need to be created so that both IPv4 and IPv6 connectivity to and
from a pod can be tested within the same test case. A new dual-stack test flag
will be created to control when the dual-stack tests are run versus single
stack versions of the tests:

```
[It] should function for node-pod communication: udp [Conformance]
[It] should function for node-pod communication: http [Conformance]
[It] should function for intra-pod communication: http [Conformance]
[It] should function for intra-pod communication: udp [Conformance]
```
Most service test cases do not need to be updated as the service remains single
stack.

For the test that checks pod internet connectivity, the IPv4 and IPv6 tests can
be run individually, with the same initial configurations.

```
[It] should provide Internet connection for containers
```

### User Stories
\<TBD\>

### Risks and Mitigations
\<TBD\>


## Implementation History
Refer to the [Implementation Plan](#implementation-plan)

## Alternatives

### Dual-stack at the Edge

Instead of modifying Kubernetes to provide dual-stack functionality within the
cluster, one alternative is to run a cluster in IPv6-only mode, and instantiate
IPv4-to-IPv6 translation mechanisms at the edge of the cluster. Such an
approach can be called "Dual-stack at the Edge". Since the translation
mechanisms are mostly external to the cluster, very little changes (or
integration) would be required to the Kubernetes cluster itself. (This may be
quicker for Kubernetes users to implement than waiting for the changes proposed
in this proposal to be implemented).

For example, a cluster administrator could configure a Kubernetes cluster in
IPv6-only mode, and then instantiate the following external to the cluster:
- Stateful NAT64 and DNS64 servers: These would handle connections from IPv6
  pods to external IPv4-only servers. The NAT64/DNS64 servers would be in the
  data center, but functionally external to the cluster. (Although one
  variation to consider would be to implement the DNS64 server inside the
  cluster as a CoreDNS plugin.)
- Dual-stack ingress controllers (e.g. Nginx): The ingress controller would
  need dual-stack access on the external side, but would load balance to
  IPv6-only endpoints inside the cluster.
- Stateless NAT46 servers: For access from IPv4-only, external clients to
  Kubernetes pods, or to exposed services (e.g. via NodePort or ExternalIPs).
  This may require some static configuration for IPv4-to-IPv6 mappings.

### Variation: Single-Stack Service CIDRs

As a variation to the "full" Dual-Stack approach, we could consider a simpler
form which adds dual-stack to Pods but not Services.  The main advantage of
this model would be lower implementation complexity.  This approach was
explored and it was determined that it was surprising to users, and the
complexity was not significantly less.

#### Changes Required
- [controller-manager startup
  configuration](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-controller-manager/):
  The "--service-cluster-ip-range" startup argument would need to be modified
  to accept a comma-separated list of CIDRs.
- [kube-apiserver startup
  configuration](https://kubernetes.io/docs/reference/command-line-tools-reference/kube-apiserver/):
  The "--service-cluster-ip-range" would need to be modified to accept a
  comma-separated list of CIDRs.
- [Service V1 core
  API](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.10/#service-v1-core):
  This versioned API object would need to be modified to support multiple
  cluster IPs for each service. This would require, for example, the addition
  of an "ExtraClusterIPs" slice of strings, and the designation of one of the
  cluster IPs as the default cluster IP for a given service (similar to changes
  described above for the PodStatus v1 core API).
- The service allocator: This would need to be modified to allocate a service
  IP from each service CIDR for each service that is created.
- 'kubectl get service' command: The display output for this command would need
  to be modified to return multiple service IPs for each service.
- CoreDNS may need to be modified to loop through both (IPv4 and IPv6) service
  IPs for each given Kubernetes service, and advertise both IPs as A and AAAA
  records accordingly in DNS responses.

## Test Plan

* Test-grid e2e tests - https://testgrid.k8s.io/sig-network-dualstack-azure-e2e
* e2e tests
  * [sig-network] [Feature:IPv6DualStackAlphaFeature] [LinuxOnly] should create
    pod, add IPv6 and IPv4 IP to pod IPs
  * [sig-network] [Feature:IPv6DualStackAlphaFeature] [LinuxOnly] should have
    IPv4 and IPv6 internal node IP
  * [sig-network] [Feature:IPv6DualStackAlphaFeature] [LinuxOnly] should have
    IPv4 and IPv6 node podCIDRs
  * [sig-network] [Feature:IPv6DualStackAlphaFeature] [LinuxOnly] should be
    able to reach pod on IPv4 and IPv6 IP
    [Feature:IPv6DualStackAlphaFeature:Phase2]
  * [sig-network] [Feature:IPv6DualStackAlphaFeature] [LinuxOnly] should create
    service with cluster IP from primary service range
    [Feature:IPv6DualStackAlphaFeature:Phase2]
  * [sig-network] [Feature:IPv6DualStackAlphaFeature] [LinuxOnly] should create
    service with IPv4 cluster IP [Feature:IPv6DualStackAlphaFeature:Phase2]
  * [sig-network] [Feature:IPv6DualStackAlphaFeature] [LinuxOnly] should create
    service with IPv6 cluster IP [Feature:IPv6DualStackAlphaFeature:Phase2]
* e2e tests should cover the following:
  * multi-IP, same family
  * multi-IP, dual-stack
  * single IP, IPv4
  * single IP, IPv6

## Graduation Criteria

This capability will move to beta when the following criteria have been met.

* Kubernetes types finalized
* CRI types finalized
* Pods to support multi-IPs
* Nodes to support multi-CIDRs
* Service resource supports pods with multi-IP
* Kubenet to support multi-IPs

This capability will move to stable when the following criteria have been met.

* Support of at least one CNI plugin to provide multi-IP
* e2e test successfully running on two platforms
* testing ingress controller infrastructure with updated dual-stack services
