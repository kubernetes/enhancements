---
title: Network Proxy
authors:
  - "@cheftako"
  - "@anfernee"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-networking
  - sig-cloud-provider
reviewers:
  - TBD
  - "@lavalamp"
  - "@deads2k"
  - "@bowei"
  - "@andrewsykim"
approvers:
  - TBD
editor: "@calebamiles"
creation-date: 2019-02-25
last-updated: 2019-02-26
status: provisional
see-also:
  - "https://goo.gl/qiARUK - Network Proxy design proposal"
  - "https://goo.gl/ipwDkX - Explicit API server to node communications"
replaces:
superseded-by:
---

# Network Proxy

## Table of Contents

- [Network Proxy](#network-proxy)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Definitions](#definitions)
  - [Proposal](#proposal)
    - [Network Context](#network-context)
    - [Proxy gRPC definition](#proxy-grpc-definition)
    - [Connectivity Proxy](#connectivity-proxy)
    - [Direct Connection](#direct-connection)
    - [Kubernetes API Server Outbound Requests](#kubernetes-api-server-outbound-requests)
    - [Testing the Solution](#testing-the-solution)
    - [Security](#security)
    - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [User Stories](#user-stories)
    - [Combined Master and Cluster Network](#combined-master-and-cluster-network)
    - [Master and Unstrusted Cluster Network](#master-and-unstrusted-cluster-network)
    - [Master and Cluster Networks which are not IP Routable](#master-and-cluster-networks-which-are-not-ip-routable)
    - [Better Monitoring](#better-monitoring)
  - [Design Details](#design-details)
    - [Risks and Mitigations](#risks-and-mitigations)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
      - [Examples](#examples)
        - [Alpha -> Beta Graduation](#alpha---beta-graduation)
        - [Beta -> GA Graduation](#beta---ga-graduation)
        - [Removing a deprecated flag](#removing-a-deprecated-flag)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
  - [Implementation History](#implementation-history)
  - [Drawbacks [optional]](#drawbacks-optional)
  - [Alternatives [optional]](#alternatives-optional)
  - [Infrastructure Needed [optional]](#infrastructure-needed-optional)

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

We will build an extensible system which controls network traffic from the Kube API Server. 
We will add a network proxy system. The KAS can be configured to send traffic
(or not) to one or more of the proxies. Users can drop in custom proxies if the
default behavior is insufficient.

## Motivation

Kubernetes has outgrown the [SSH tunnels](https://github.com/kubernetes/kubernetes/issues/54076). 
They complicate KAS code and only one cloud provider implemented them.
After a year of deprecation time, they will be removed in an upcoming release.

In retrospect, having an explicit level of indirection that separates user-initiated network traffic from API 
server-initiated traffic is a useful concept.
Cloud providers want to control how API server to pod, node and service network traffic is implemented.
Cloud providers may choose to run their API server (control network) and the cluster nodes (cluster network) 
on isolated networks. The control and cluster networks may have overlapping IP addresses. 
There for they require a non IP routing layer (SSH tunnel are an example).
Adding this layer enables metadata audit logging. It allows validation of outgoing API server connections.
Structuring the API server in this way is a forcing function for keeping architectural layering violations out of apiserver.
In combination with a firewall, this separation of networks protects against security concerns such as 
[Security Impact of Kubernetes API server external IP address proxying](https://groups.google.com/d/msg/kubernetes-security-announce/tyd-MVR-tY4/tyREP9-qAwAJ).

### Goals

Delete the SSH Tunnel/Node Dialer code from Kube APIServer.
Enable admins to fix https://groups.google.com/d/msg/kubernetes-security-announce/tyd-MVR-tY4/tyREP9-qAwAJ.
Allow isolation of the Control network from the Cluster network.

### Non-Goals

Build a general purpose Proxy which does everything. (Users should build their own
custom proxies with the desired behavior, based on the provided proxy)

## Definitions

- **Master Network** An IP reachable network space which contains the master components, such as Kubernetes API Server, 
Connectivity Proxy and ETCD server.
- **Cluster Network** An IP reachable network space which contains (some/all?) the clusters Nodes. 
Worth noting that the Cluster Network may be completely disjoint from the Master network. 
It may have overlapping IP addresses to the Master Network or other means of network isolation. 
Direct IP routability between cluster and master networks should not be assumed.
- **KAS** Kubernetes API Server, responsible for serving the Kubernetes API to clients.
- **KMS** Key Management Service, plugins for secrets encryption key management
- **Connectivity service** A component built into the KAS which provides a golang dialer for outgoing connection requests. 
The dialer provided depends on NetworkContext information.
- **Connectivity proxy** The proxy that runs in the master network. 
It has a secure channel established to the cluster network. 
It exposes a gRPC interface to KAS to provide connectivity service.
- **Flat Network** A network where traffic can be successfully routed using IP. 
Implies no overlapping (i.e. shared) IPs on the network.

## Proposal

We will run a connectivity proxy inside the master network. 
The connectivity proxy exposes a gRPC interface to the KAS. 
Each connectivity proxy allows secure connections to one or more cluster networks. 
Any network addressed by a connectivity proxy must be flat. 
Currently the only mechanism for handling overlapping IP ranges in Kubernetes is the Proxy. 
Non IP routable traffic, past the proxy, would need to be a non Kubernetes mechanism to route.

Running the connectivity proxy in a separate process has a few advantages. 
The connectivity proxy can be extended without recompiling the KAS. 
Administrators can run their own variants of the connectivity proxy. 
Traffic can be audited or forwarded (eg. via a proprietary VPN) using a custom connectivity proxy.
The separation removes master <-> cluster connectivity concerns from the KAS.  
The code and responsibility separation lowers the complexity of the KAS code base. 
The separation reduces the effects of issue such as crashes in the connectivity impacting the KAS. 
Connectivity issues will not stop the KAS from serving API requests. 
This is important as serving API requests may be necessary in order to fix the crashes. 
A problem with a node, set of nodes or load-balancers configuration, may be fixed with API requests. 

![Network Proxy Simple Cluster](NetworkProxySimpleCluster.png)
The diagram shows API Server’s outgoing traffic flow. 
The user (in blue box), master network (in purple cloud) and 
a cluster network (in green cloud) are represented.  

The user (blue) initiates communication to the KAS. 
The KAS then initiates connections to other components. 
It could be node/pod/service in cluster networks (red dotted arrow to green cloud), 
or etcd for storage in the same master network (blue arrow) or mutate the request 
based on an admission web-hook (red dotted arrow to purple cloud). 
The KAS handles these cases based on NetworkContext based traffic routing. 
The connectivity proxy should be able to do routing solely based on IP. 
The proxy should not require the NetworkContext. This means the service CIDR, 
node CIDR and pod CIDR of each cluster network cannot overlap. 

### Network Context

The minimal NetworkContext looks like the following struct in golang:

```go
type NetworkContext struct {
        // NetworkName is the unique name of the network 
        // to select to route the traffic to. 
        NetworkName string
}
```

NetworkName specifies the network to route traffic to. 
The KAS starts with a list of registered network names and 
the corresponding Connectivity services. So the KAS knows where to 
proxy the traffic to, otherwise it return an “Unknown network” error. 

The KAS starts with a proxy configuration like the below example. 
The example specifies 4 networks. "direct" specifies the KAS talking directly 
on the local network (no proxy). "master" specifies the KAS talks to a proxy
listening at 1.2.3.4:5678. "cluster" specifies the KAS talk to a proxy 
listening at 1.2.3.5:5679. While these are represented as resources 
they are not intended to be loaded dynamically. 
The KAS loads them as configuration at start time.

```yaml
apiVersion: connectivityservice.k8s.io/v1alpha1
kind: ConnectivityServiceConfiguration
connectionService:
  name: direct
  connection:
    type: direct
---
apiVersion: connectivityservice.k8s.io/v1alpha1
kind: ConnectivityServiceConfiguration
connectionService:
  name: Master
  connection:
    type: grpc
    url: grpc://1.2.3.4:5678
    caBundle: file1.ca
---
apiVersion: connectivityservice.k8s.io/v1alpha1
kind: ConnectivityServiceConfiguration
connectionService:
  name: Cluster
  connection:
    type: grpc
    url: grpc://1.2.3.5:5679
    caBundle: file2.ca

```

NetworkContext could be extended to contain more contextual information.
This would allow smarter routing based on the k8s object KAS is processing 
or which user/tenant tries to initiate the request, etc.

### Proxy gRPC definition

In order to serve a proxy request, one gRPC bidirectional stream on proxy
server is created to serve it. It's a 1:1 mapping from TCP connection to a
gRPC stream, so the state of TCP connection is exactly the same as the gRPC
stream state.

```grpc
syntax = "proto3";

service ProxyService {
  // Proxy a TCP connection to a remote address defined by ConnectParam.
  // The ConnectParam is defined in metadata under key "x-kube-net-proxy".
  // metadata["x-kube-net-proxy"] = base64.Encode(proto.Marshal(connectOptions))
  rpc Proxy(stream Payload) returns (stream Payload) {}
}

// ConnectOptions defines the remote TCP endpoint to connect
message ConnectOptions {
  string remote_addr = 1; // remote address to connect to. e.g. 8.8.8.8:53
}

// Payload defines a TCP payload.
message Payload {
  bytes data = 1;
}
```

### Connectivity Proxy

The connectivity proxy(s) can run in the same container as the KAS. 
It should run on the same machine and must run in the same flat network as the KAS. 
It listens on a port for gRPC connections from the KAS. 
This port would be for forwarding traffic to the appropriate cluster. 
It should have an administrative port speaking https. 
The administrative port serves the liveness probe and metrics. 
The liveness probe prevents a partially broken cluster 
where the KAS cannot connect to the cluster. This port also serves 
pprof debug commands and monitoring data for the proxy. 

### Direct Connection

This connection type uses the default dialer. 
This allows use of the connectivity service without the connectivity proxy. 
This is a quick way to run the system in a “legacy” or fallback mode. 
Simple clusters (not needing network segregation) run this way to avoid the overhead 
(in latency or configuration) of the connectivity proxy.

### Kubernetes API Server Outbound Requests

The majority of the KAS communication originates from incoming requests. 
Here we cover the outgoing requests. This is our understanding of those requests 
and some details as to how they fit in this model.

- **ETCD** It is possible to make etcd talk via the proxy. 
The etcd client takes a transport. 
(https://github.com/etcd-io/etcd/blob/master/client/client.go#L101) 
We will add configuration as to which proxy an etcd client should use. 
(https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/storage/storagebackend/config.go) 
This will add an extra process hop to our main scaling axis. 
We will scale test the impact and publish the results. As a precaution
we will add an extra network configuration 'etcd' separate from ‘master’.
Etcd requests can be configured separately from the rest of 'master'.  
- **Pods/Exec**, **Pods/Proxy**, **Pods/Portforward**, **Pods/Attach**, **Pods/Log**
Pod requests (and pod sub-resource requests) are meant for the cluster 
and will be routed based on the ‘cluster’ NetworkContext.
- **Nodes/Proxy**
Node requests (and node sub-resource requests) are meant for the cluster 
and will be routed based on the ‘cluster’ NetworkContext.
- **Services/Proxy**
Service requests (and service sub-resource requests) are meant for the cluster 
and will be routed based on the ‘cluster’ NetworkContext.
- **Admission Webhooks**
Admission webhooks can be destined for a service or a URL. 
If destined for a service then the service rules apply (send to 'cluster'). 
If destined for a URL then we will use the ‘master’ NetworkContext. 
- **Aggregated API Server (and OpenAPI requests for aggregated resources)**
Aggregated API Servers can be destined for a service or a URL. 
If destined for a service then the service rules apply. 
If destined for a URL then we will use the ‘master’ NetworkContext.
- **Authentication, Authorization and Audit Webhooks**
These Webhooks use a kube config file to determine destination. 
Given that we use a ‘master’ NetworkContext.
- **ImagePolicyWebhook**
The image policy webhook uses a kube config file to determine destination. 
Given that we use a ‘master’ NetworkContext.
- **KMS GRPC Service**
KMS connects with an ‘endpoint’ (not the resource) via gRPC. 
The service at the endpoint provides the secret information for use in encryption. 
This is not a user space configurable system. 
Given that we use a ‘master’ NetworkContext.

### Testing the Solution

We will test using a network namespace to partition the KAS from the test nodes. 
It is then impossible to connect directly from the KAS to the test nodes. 
This ensures that the proxy must be used for logs, exec, port forward, aggregation and webhooks. 
We run with this configuration and a direct configuration for these specific features. 
This ensures that the solution works and will continue to work.

### Security

One motivation for network proxy is providing a mechanism to secure 
https://groups.google.com/d/msg/kubernetes-security-announce/tyd-MVR-tY4/tyREP9-qAwAJ. 
This, in conjunction with a firewall or other network isolation, fixes the security concern. 

### Implementation Details/Notes/Constraints

You may want to check the original design doc for alternatives and futures considered. https://goo.gl/qiARUK.


## User Stories

#### Combined Master and Cluster Network

Customers can run a cluster which combines the master and cluster networks. 
They configure all their connectivity configuration to direct.
This bypasses the proxy and optimizes the performance. For a customer with no
security concerns with combined network, this is a fairly simple straight forward configuration.

#### Master and Untrusted Cluster Network

A customer may want to isolate their master from their cluster network. This may be a 
simple separation of concerns or due to something like running untrusted workloads on 
the cluster network. Placing a firewall between the master and 
cluster networks accomplishes this. A few ports for the KAS public port and Proxy public port 
are opened between these networks. Separation of concerns minimizes the 
accidental interactions between the master and cluster networks. It minimizes bandwidth 
consumption on the cluster network negatively impacting the control plane. The 
combination of firewall and proxy minimizes the interaction between the networks to
a set which can be more easily reasoned about, checked and monitored.

#### Master and Cluster Networks which are not IP Routable

If master and cluster network CIDRs are not controlled by the same entity, then they
can end up having conflicting IP CIDRs. Traffic cannot be routed between
them based strictly on IP address. The connection proxy solves this issue. 
It also solves connectivity using a VPN tunnel. The proxy offloads the work off sending traffic
to the cluster network from the KAS. The proxy gives us extensibility.

#### Better Monitoring

Instrumenting the network proxy requests with out of band data 
(Eg. requester identity/tradition context) enables a Proxy to 
provide increased monitoring of Master originated requests.


## Design Details

### Risks and Mitigations

The primary risk of this solution would seem to be some portion of the proxy or agent failing.
For existing clusters which do not depend on SSH Tunnels or any of the new functionality, the
mitigation would be to set all networks to direct. This should bypass the proxy and allow
the system to work as it does today. For anyone using SSH Tunnels we are planning to support 
both for several releases. 

### Test Plan

The primary test plan is to set up a network namespace with a firewall dividing the master and cluster
networks. Then running the existing tests for logs, proxy and portforward to ensure the 
routing works correctly. It should work with the correct configuration and fail correctly
with a direct configuration. Normal tests would be run with the direct 
configuration to ensure the mitigation is working correctly.


Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial KEP should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives [optional]

Similar to the `Drawbacks` section the `Alternatives` section is used to highlight and record other possible approaches to delivering the value proposed by a KEP.

## Infrastructure Needed [optional]

Use this section if you need things from the project/SIG.
Examples include a new subproject, repos requested, github details.
Listing these here allows a SIG to get the process for these resources started right away.
