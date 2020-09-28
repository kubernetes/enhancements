# KEP-2025: Extending Apiserver Network Proxy to handle traffic originated from Node network

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Definitions](#definitions)
- [Proposal](#proposal)
  - [Traffic Flow](#traffic-flow)
  - [Agent additional flags](#agent-additional-flags)
  - [Handling the Traffic from the pods to the agent](#handling-the-traffic-from-the-pods-to-the-agent)
  - [Handling the Traffic from the Kubelet to the agent](#handling-the-traffic-from-the-kubelet-to-the-agent)
  - [Deployment Model](#deployment-model)
  - [Listening Interface for Konnectivity agent](#listening-interface-for-konnectivity-agent)
  - [Authentication between Konnectivity agent and server](#authentication-between-konnectivity-agent-and-server)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Allowed Destination](#allowed-destination)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The goal of this proposal is to allow traffic to flow from the Node Network to the Master Network, when those networks are otherwise isolated and there is a desire not to expose the Kubernetes API Server publicly.

## Motivation

API server network proxy has been originally introduced to allow running the cluster nodes on distinct isolated networks with respect to the one hosting the control plane components. This provides a way to handle traffic originating from the Kube API Server and going to the node networks. When using this setup, there are no other options than to directly expose the KAS to the Internet or setting up a VPN to handle traffic originated from the cluster nodes (i.e. Kubelet, pods).
Exposing Konnectivity Proxy Server only would add an additional layer of security, given that the tunnels with proxy agents are secured with mTLS or Token authentication. This would protect it, for instance, from KAS misconfigurations or vulnerabilities that could expose sensitive information and/or access to unauthenticated users.
Furthermore there is no standard solution for this kind of setup. It is possible to rely on VPNs for example to achieve a similar goal, but this requires specific implementations. What we propose here is to build on top of what we have, and having a consistent approach for master to node and node to master communications.
Extending the API server network proxy to allow handling traffic in both directions seems to be the most consistent approach to address this issue.
Moreover, this will provide the possibility of routing agents’ traffic to their dedicated KAS based on SNI, enabling the option of load balancing traffic directed to different clusters with a single load balancer.

### Goals

* Handle requests from the nodes to the control plane. Enable communication from the Node Network to the Master Network without having to expose the KAS to the Node Network.

### Non-Goals

* Define a mechanism for exchanging authentication information used for establishing the secure channels between agents and server (e.g. certificates, tokens).
* Define a solution involving less than one agent per node.
* Being able to reach destinations other than the KAS on the master network, this could be considered in the future if some use-cases arise.

## Definitions

* **Master Network** An IP reachable network space which contains the master components, such as Kubernetes API Server, Connectivity Proxy and ETCD server.
* **Node Network** An IP reachable network space which contains all the clusters Nodes, for alpha. Worth noting that the Node Network may be completely disjoint from the Master network. It may have overlapping IP addresses to the Master Network or other means of network isolation. Direct IP routability between cluster and master networks should not be assumed. Later versions may relax the all node requirement to some.
* **KAS** Kubernetes API Server, responsible for serving the Kubernetes API to clients.
* **Konnectivity Server** The proxy server which runs in the master network. It has a secure channel established to the node network. It could work on either a HTTP Connect mechanism or gRPC. If the former it would expose a gRPC interface to KAS to provide connectivity service. If the latter it would use standard HTTP Connect. Formerly known as the Network Proxy Server.
* **Konnectivity Agent** A proxy agent which runs in the node network for establishing the tunnel. Formerly known as the Network Proxy Agent.
* **Flat Network** A network where traffic can be successfully routed using IP. Implies no overlapping (i.e. shared) IPs on the network.

## Proposal

Currently the Konnectivity Server is accepting requests from the KAS either with the gRPC or the HTTP Connect interfaces and is taking care of forwarding the traffic to the Konnectivity Agent using the previously established connections (initiated by the agents).

In order to enable traffic from Kubelets and pods running on Node Network, the Konnectivity Agents have to expose an endpoint that will be listening on a specific port and forward the traffic to the KAS on the Master Network (only traffic generated from within the host will be able to target it). As opposed to the traffic flowing from the Master Network to the Node Network, the Konnectivity Agent should act transparently: From a Kubelets or pods standpoint, the Konnectivity Agent should be the final destination instead of acting as a proxy.

The reason why we do not intend to use the same strategy used in the other direction is that we do not have control over the clients using the Kubernetes default service to communicate with the KAS.

In order to enable the feature the following configurations are needed:

* On the Konnectivity Agent, enable the feature gate, define the listening port and addess/port of the KAS.
* On Konnectivity server, enable the feature gate and allow the address/port to reach the KAS.
* On KAS, configure the IP address and port that will be used by the agent with `advertise-address` and the `secure-port` flags.

### Traffic Flow

```
client =TCP=> (:6443) agent GRPC=> server =TCP=> KAS(:6443)
                         |            ^
                         |   Tunnel   |
                         +------------+
```

The agent listens for TCP connections at the configured port. When a TCP connection is established between the client and the Konnectivity Agent, the following happens:
1. A GRPC DIAL_REQ message is sent to the Konnectivity server containing the destination address associated with the port.
2. Upon reception of the DIAL_REQ the Konnectivity Server verifies if the destination is allowed (see [Allowed Destination](#allowed-destination) section). If not allowed a DIAL_RSP containing an error is sent back immediately to the agent that terminates the connection.
3. In case the destination is allowed Konnectivity Server opens a TCP connection with the destination host/port and replies to the Konnectivity Agent with a GRPC DIAL_RES message.
4. At this point the tunnel is established and data is piped through it, carried over GRPC DATA packets.

### Agent additional flags

* `--target=local_port:dst_host_ip:dst_port`:
    - local_port: The local port where the agent binds to.
    - dst_host_ip: Ip or hostname used by the KAS. In case of IPv6
    the address should be wrapped in squared brackets, consistently with [RFC3986](https://www.ietf.org/rfc/rfc3986.txt) notation.
    - dst_port: The remote port used by the KAS.
e.g. --target=6443:apiserver.svc.cluster.local:6443
e.g. --target=6443:[2001:db8:1f70::999:de8:7648:6e8]:6443

* `--bind-address=ip`: Local IP address where the Konnectivity Agent will listen for incoming requests. It will be bound to a dummy interface with IP x.y.z.v defined by the user. Must be used with the previous one to enable incoming requests. If not, and for backward compatibility, only the traffic initiated from the Control Plane will be allowed.

### Handling the Traffic from the pods to the agent

As mentioned above, pods make use of the Kubernetes default service to reach the KAS. To keep things transparent from a pod perspective, they will hit the Konnectivity Agent using the Kubernetes default service. The endpoints will contain the Konnectivity Agent address instead of the KAS one.
The configuration will be done using the KAS flags `advertise-address ip` and `secure-port port`, that should match respectively the `bind-address` and the `local_port` of the Konnectivity Agent described above.

### Handling the Traffic from the Kubelet to the agent

Kubelet does not use the Kubernetes default service to reach the KAS. Instead it relies on a bootstrap kubeconfig file that is used to connect to the KAS. It then generates a proper kubeconfig file that will be using the same URL.
Instead of specifying the KAS FQDN/address in the bootstrap kubeconfig file, we will be using the local IP address of the Konnectivity agent (`--bind-address ip`).

### Deployment Model

The agent can be run as static pod or systemd unit. In any case the agent should be started to give access to the KAS and kubelet first and to the hosted pods later. This means that using DaemonSet or Deployment is not an option in this setup, because the kubelet would not be able to get the pod manifests from the KAS.
Note that no communications between kubelet/pods on the node network and KAS will be possible until the agent is up and running. Therefore the kubelet must handle the absence of connectivity with the KAS properly (the retry logic has to be verified).

### Listening Interface for Konnectivity agent

As mentioned before, we will be using the Kubernetes default service to route traffic to the agent. The service in itself has a couple of limitations: it can’t be used as a type externalName, thus preventing usage of DNS names. But also, some general services limitations apply: endpoints can't use the link-local range and the localhost range. This means that we are left with the 3 private IPs ranges (10.0.0.0/8, 172.16.0.0/12 and 192.168.0.0/16).

The agent will create a dummy interface, assign it the ip provided with the `bind-address` flag, using `host` scope, and will start listening on this IP:local_port (local_port is defined with the `target` flag). This will allow all agents to bind to the ip address advertised by the KAS, that will be valid only inside the node.

### Authentication between Konnectivity agent and server

Konnectivity agent currently support mTLS or Token based authentication. Note that API objects such as Secrets cannot be accessed neither when static pod nor systemd service deployment strategy is used. The authentication secret will be made available to the agent through a different channel (e.g. provisioned in the worker node file-system at creation time).

### Risks and Mitigations

#### Allowed Destination

> A generic TCP tunnel is fraught with security risks. First, such
> authorization should be limited to a small number of known ports.
> The Upgrade: mechanism defined here only requires onward tunneling at
> port 80. Second, since tunneled data is opaque to the proxy, there
> are additional risks to tunneling to other well-known or reserved
> ports. A putative HTTP client CONNECTing to port 25 could relay spam
> via SMTP, for example.
> -- <cite>[rfc2817][http_upgrade_tls_rfc]</cite>

Similarly to CONNECT tunnel this feature opens the door to generic TCP tunnels to arbitrary destinations.

A mechanism for restricting access to the possible destinations on the Master Network should be implemented on the Konnectivity server.
The allowed destination will be provided with the following flag:

* `--allowed-destination=dst_host:dst_port`: The address and port of the KAS.

When no `allowed-destination` flag is provided no destinations on the Master Network will be allowed, and all traffic will be dropped.

Note: if this feature will be extended to allow reaching arbitrary destinations in the master network, this can be easily generalized by allowing multiple occurrences of this flag and maintaining a list of allowed destinations.

## Design Details


### Test Plan

The primary test plan is to set up a network namespace with a firewall dividing the master and node networks, making sure the only accessible endpoint in the master network from the node network is the Konnectivity server. The test should verify that nodes joins the cluster as expected, meaning that kubelets can communicate with the KAS. Then it should also verify that pods scheduled in the nodes get to ready state and are able to access the KAS by using the kubernetes service in the default namespace.

### Graduation Criteria


### Upgrade / Downgrade Strategy

As we cannot rely on Kubernetes resources like DaemonSet or Deployment (see [Deployment Model](#deployment-model)), upgrading the agent will be a similar process as upgrading the Kubelet. This can be achieved either in place, by updating and restarting the systemd service on every node or by changing the manifests if static pods are used. The alternative, in case in-place updates are not an option, is to create a new set of nodes with the new configuration and replace the old ones.

### Version Skew Strategy


## Production Readiness Review Questionnaire


### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [*] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: EnableClusterToMasterTraffic
    - Components depending on the feature gate: Konnectivity server and agent
  - [*] Other
    - This is the procedure to enable in a live cluster assuming that the
      secure port of KAS is `6443`, the ip address chosen for the agent is
      `10.0.0.1` and the Konnectivity server is deployed as a DaemonSet on all
      master nodes:
      1. Deploy the Konnectivity agent in the nodes by using a static pod or a
         systemd unit. Add the following flags to enable the feature and to
         forward local port `6443` to `localhost:6443` on Konnectivity server
         side:
         `--feature-gates="EnableClusterToMasterTraffic=true"`
         `--target=6443:localhost:6443`
      2. Deploy the Konnectivity server on master nodes following the
         documented procedure, and adding the following flags to enable the
         feature and to allow traffic to `localhost:6443`:
         `--feature-gates="EnableClusterToMasterTraffic=true"`
         `--allowed-destination=localhost:6443`.
      3. Add the following flags to the KAS `--advertise-address=10.0.0.1`,
         `--secure-port=6443`, and restart the process for all master nodes.
    - In order to disable the feature you just need to remove or re-establish
      the original `--advertise-address` flag and restart the KAS services on
      all nodes.

* **Does enabling the feature change any default behavior?**
  Yes, all traffic originated from node network will pass from the Konnectivity
  agents. In case network proxies are used, the agent should be configured
  properly, and the ip used by the agents should be included in `no_proxy`
  environment variable.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, it can be disabled by simply changing the KAS `--advertise-address` and
  `--secure-port` flags, assuming that KAS is accessible to kubelets.
  In addition the `target` flag should be removed from Konnectivity agents,
  `allowed-destination` should be removed from Konnectivity server and
  feature gates `EnableClusterToMasterTraffic=true` should be removed from
  both.

* **What happens if we reenable the feature if it was previously rolled back?**
  Rolling back has no impact on subsequent re-enablements.

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of these, fill in the following—thinking about running existing user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

Enabling this feature will result in no additional API calls.

* **Will enabling / using this feature result in introducing new API types?**

No new API types are required.

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing pod)

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History


## Drawbacks

Main drawback is the increased complexity of the nodes bootstrap, as kubelets
need to have the Konnectivity server up and running in order to communicate
with KAS. Upgrading the Konnectivity agents is also more complicated.

## Alternatives

* Setting up a VPN (e.g. IPSec, Wireguard). More complicated to set-up (e.g. MTU size
  issues, specific configuration, kernel requirements and ports to open).

## Infrastructure Needed (Optional)

