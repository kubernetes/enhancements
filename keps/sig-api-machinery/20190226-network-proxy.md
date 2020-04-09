---
title: API Server Network Proxy
authors:
  - "@cheftako"
  - "@anfernee"
owning-sig: sig-api-machinery
participating-sigs:
  - sig-network
  - sig-cloud-provider
reviewers:
  - TBD
  - "@lavalamp"
  - "@deads2k"
  - "@bowei"
  - "@andrewsykim"
  - "@justinsb"
  - "@krousey"
  - "@khenidak"
  - "@mikedanese"
approvers:
  - "@deads2k - For Kube API Server portion of KEP"
  - "@bowei - For networking/proxy portion of KEP"
editor: "@calebamiles"
creation-date: 2019-02-25
last-updated: 2019-04-30
status: implementable
see-also:
  - "https://goo.gl/qiARUK - Network Proxy design proposal"
  - "https://goo.gl/ipwDkX - Explicit API server to node communications"
  - "https://github.com/kubernetes-sigs/apiserver-network-proxy - Reference implementations of API Server Network Proxy"
replaces:
superseded-by:
---

# API Server Network Proxy

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Definitions](#definitions)
- [Proposal](#proposal)
  - [Network Context](#network-context)
  - [Proxy gRPC definition](#proxy-grpc-definition)
  - [Konnectivity Server](#konnectivity-server)
  - [Direct Connection](#direct-connection)
  - [Kubernetes API Server Outbound Requests](#kubernetes-api-server-outbound-requests)
  - [Testing the Solution](#testing-the-solution)
  - [Security](#security)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
- [User Stories](#user-stories)
    - [Combined Master and Node Network](#combined-master-and-node-network)
    - [Master and Untrusted Node Network](#master-and-untrusted-node-network)
    - [Master and Node Networks which are not IP Routable](#master-and-node-networks-which-are-not-ip-routable)
    - [Better Monitoring](#better-monitoring)
- [Design Details](#design-details)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Summary

We will build an extensible system which controls network traffic from the Kube API Server.
We will add a traffic egress or network proxy system. The KAS can be configured to send traffic
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
Handle requests from the Cluster to the Control Plane. (The proxy can be extended to
do this. However that is left to the User if they want that behavior)

## Definitions

- **Master Network** An IP reachable network space which contains the master components, such as Kubernetes API Server,
Connectivity Proxy and ETCD server.
- **Node Network** An IP reachable network space which contains all the clusters Nodes, for alpha.
Worth noting that the Node Network may be completely disjoint from the Master network.
It may have overlapping IP addresses to the Master Network or other means of network isolation.
Direct IP routability between cluster and master networks should not be assumed.
Later version may relax the all node requirement to some.
- **KAS** Kubernetes API Server, responsible for serving the Kubernetes API to clients.
- **KMS** Key Management Service, plugins for secrets encryption key management
- **Egress Selector** A component built into the KAS which provides a golang dialer for outgoing connection requests.
The dialer provided depends on NetworkContext information.
- **Konnectivity Server** The proxy server which runs in the master network.
It has a secure channel established to the cluster network.
It could work on either a HTTP Connect mechanism or gRPC.
If the former it would exposes a gRPC interface to KAS to provide connectivity service.
If the latter it would use standard HTTP Connect.
Formerly known the the Network Proxy Server.
- **Konnectivity Agent** A proxy agent which runs in the node network for
  establishing the tunnel.
Formerly known as the Network Proxy Agent.
- **Flat Network** A network where traffic can be successfully routed using IP.
Implies no overlapping (i.e. shared) IPs on the network.

## Proposal

We will run a connectivity server inside the master network.
It could work on either a HTTP Connect mechanism or gRPC.
For the alpha version we will attempt to get this working with HTTP Connect.
We will evaluate HTTP Connect for scalability, error handling and traffic types.
For scalability we will be looking at the number of required open connections.
Increasing usage of webhooks means we need better than 1 request per connection (multiplexing).
We also need the tunnel to be tolerant of errors in the requests it is transporting.
HTTP-Connect only supports HTTP requests and not things like DNS requests.
We assume that for HTTP URL request,s it will be the proxy which does the DNS lookup.
However this means that we cannot have the KAS perform a DNS request to then do a follow on request.
If no issues are found with HTTP Connect in these areas we will proceed with it.
If an issue is found then we will update the KEP and switch the client to the gRPC solution.
This should be as simple as switching the connection mode of the client code.

It may be desirable to allow out of band data (metadata) to be transmitted from the KAS to the Proxy Server.
We expect to handle metadata in the HTTP Connect case using http 'X' headers on the Connect request.
This means that the metadata can only be sent when establishing a KAS to Proxy tunnel.
For the GRPC case we just update the interface to the KAS.
In this case the metadata can be sent even during tunnel usage.

Each connectivity proxy allows secure connections to one or more cluster networks.
Any network addressed by a connectivity proxy must be flat.
Currently the only mechanism for handling overlapping IP ranges in Kubernetes is the Proxy.
Non IP routable traffic, past the proxy, would need to be a non Kubernetes mechanism to route.

Running the connectivity proxy in a separate process has a few advantages.
- The connectivity proxy can be extended without recompiling the KAS.
Administrators can run their own variants of the connectivity proxy.
- Traffic can be audited or forwarded (eg. via a proprietary VPN) using a custom connectivity proxy.
- The separation removes master <-> cluster connectivity concerns from the KAS.
- The code and responsibility separation lowers the complexity of the KAS code base.
- The separation reduces the effects of issue such as crashes in the connectivity impacting the KAS.
Connectivity issues will not stop the KAS from serving API requests.
This is important as serving API requests may be necessary in order to fix the crashes.
A problem with a node, set of nodes or load-balancers configuration, may be fixed with API requests.

![API Server Network Proxy Simple Cluster](NetworkProxySimpleCluster.png)
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
type EgressType int

const (
    // Master is the EgressType for traffic intended to go to the control plane.
    Master EgressType = iota
    // Etcd is the EgressType for traffic intended to go to Kubernetes persistence store.
    Etcd
    // Cluster is the EgressType for traffic intended to go to the system being managed by Kubernetes.
    Cluster
)

// NetworkContext is the struct used by Kubernetes API Server to indicate where it intends traffic to be sent.
type NetworkContext struct {
    // EgressSelectionName is the unique name of the
    // EgressSelectorConfiguration which determines
    // the network we route the traffic to.
    EgressSelectionName EgressType
}
```

EgressSelectionName specifies the network to route traffic to.
The KAS starts with a list of registered konnectivity service names. These
correspond to networks we route traffic to. So the KAS knows where to
proxy the traffic to, otherwise it return an “Unknown network” error.

The KAS starts with a proxy configuration like the below example.
The example specifies 4 networks. "direct" specifies the KAS talking directly
on the local network (no proxy). "master" specifies the KAS talks to a proxy
listening at 1.2.3.4:5678. "cluster" specifies the KAS talk to a proxy
listening at 1.2.3.5:5679. While these are represented as resources
they are not intended to be loaded dynamically. The names are not case
sensitive. The KAS loads this resource lsit as a configuration at start time.

```yaml
apiVersion: apiserver.k8s.io/v1alpha1
kind: EgressSelectorConfiguration
egressSelections:
- name: direct
  connection:
    type: direct
- name: master
  connection:
    type: grpc
    url: grpc://1.2.3.4:5678
    caBundle: file1.pem
    clientKeyFile: proxy-client1.key
    clientCertFile: proxy-client1.crt
- name: cluster
  connection:
    type: grpc
    url: grpc://1.2.3.5:5679
    caBundle: file2.pem
    clientKeyFile: proxy-client2.key
    clientCertFile: proxy-client2.crt
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

### Konnectivity Server

The Konnectivity Server (connectivity proxy(s)) can run in the same container as the KAS.
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
and some details as to how they fit in this model. For the alpha release we
support 'master', 'etcd' and 'cluster' connectivity service names.

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
Please make sure you are a member of kubernetes-dev@googlegroups.com to view the doc.
It is also worth looking at https://github.com/kubernetes-sigs/apiserver-network-proxy as it contains the reference
implementation of the API Server Network Proxy.

## User Stories

#### Combined Master and Node Network

Customers can run a cluster which combines the master and cluster networks.
They configure all their connectivity configuration to direct.
This bypasses the proxy and optimizes the performance. For a customer with no
security concerns with combined network, this is a fairly simple straight forward configuration.

#### Master and Untrusted Node Network

A customer may want to isolate their master from their cluster network. This may be a
simple separation of concerns or due to something like running untrusted workloads on
the cluster network. Placing a firewall between the master and
cluster networks accomplishes this. A few ports for the KAS public port and Proxy public port
are opened between these networks. Separation of concerns minimizes the
accidental interactions between the master and cluster networks. It minimizes bandwidth
consumption on the cluster network negatively impacting the control plane. The
combination of firewall and proxy minimizes the interaction between the networks to
a set which can be more easily reasoned about, checked and monitored.

#### Master and Node Networks which are not IP Routable

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

Alpha:

- Feature is turned off in the KAS by default. Enabled by adding ConnectivityServiceConfiguration.
- Kubernetes will not ship with a network proxy. The feature will work with the sample network proxy in https://github.com/kubernetes-sigs/apiserver-network-proxy
- Demonstrate that the API Server Network Proxy eliminates the need for the SSH Tunnels.

Beta:

- All [Kube API Server egress points](#kubernetes-api-server-outbound-requests) have been implemented to use the
  EgressSelector.
- Have official releases of the [Konnectivity Server and Agent](https://github.com/kubernetes-sigs/apiserver-network-proxy) reference implementations.
- Have at least one OSS kube-up implementation where the feature can be turned on and
  demonstrated.
- Have run a basic load test with egresses enabled through the Konnectivity
  Server to demonstrate that concurrent requests work with Admission Webhooks.
- Tests for EgressSelector.
- e2e test with a functioning cluster with the EgressSelector conifgured to use
  a KonnectivityService.
- Add metrics and trace around the Egress Lookup/Dial code. Make sure we know
  how many egresses of each type are returned. Make sure we know how long we
  are spending dialing out.
- Ensure we have metrics on each existing egress use case.

## Implementation History

- Feature went Alpha in 1.16 with limited functionality. It will cover the log
  sub resource and communication to the etcd server.

## Alternatives [optional]

- Leave SSH Tunnels (deprecated) in the KAS. Prevents us from making the KAS cloud provider agnostic. Blocks out of tree effort.
- Build equivalent functionality into the KAS. Is not extensible. Essentially has the same issues as SSH Tunnels.
- Use a socks5 proxy. No standard mTLS mechanism for securing traffic. Does not actually act as a standard. More complicated implementation.

## Infrastructure Needed [optional]

Any one wishing to use this feature will need to create network proxy images/pods on the master and set up the ConnectivityServiceConfiguration.
The network proxy provided is meant as a reference implementation. Users as expected to extend it for their needs.
