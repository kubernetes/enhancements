---
title: layer-3-connectivity-between-clusters 
authors:
  - "@mangelajo"
owning-sig: sig-multicluster
participating-sigs:
  - sig-multicluster
  - sig-network
reviewers:
  - "@pmorie"
  - "@thockin"
approvers:
  - "@pmorie" 
  - TBD
editor: TBD
creation-date: 2019-09-17 
last-updated: 2019-09-17
status: implementable
see-also:
replaces:
superseded-by:
---

# Layer 3 Connectivity Between Clusters 

## Table of Contents
<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Infrastructure Needed](#infrastructure-needed-optional)
<!-- /toc -->

## Summary

There is an interest in the community in solving the fundamental use case of layer 3 connectivity between clusters.
The K8s community is heading fast towards adoption of using multiple clusters in each deployment for their workloads.
A natural challenge that comes with this adoption is how to connect the workloads running on different clusters in a
secured, automated fashion.

As a community we need to explore this problem and we believe that submariner is a good foundation because it handles
layer 3 connectivity between clusters running under the same or different network plugins. It does not prescribe any
specific network technology to sustain the connectivity between clusters, although the initial implementation is based
on IPSEC.

Our intention is to standardize CRDs, and agree on the semantics around multi cluster connectivity with the K8s
community. Potentially those CRDs could be implemented directly by any network plugin maintaining the API to
create such connectivity. Submariner would provide a reference implementation that may work for most network
plugins making inter-cluster / inter-network-plugin connectivity automated and secure.

## Motivation

In some installations of Kubernetes, pods and services are limited to their own clusters and are isolated from each
other between clusters. Kubernetes does not require cross-cluster connectivity and there is no standard way of
enabling it in all environments.  Some network plugins provide solutions to this problem but don’t provide
interoperability with other network plugins or a common API.

Working on the multicluster network problem we identified multiple projects trying to solve the same problem
with very different APIs leading to ecosystem fragmentation. We also identified Submariner as a project trying
to solve the connectivity problem from an (as much as it is possible) network plugin agnostic point of view.

We see an  increased demand and interest in the community  to connect workloads across multiple clusters.
Such connectivity can be used to fulfill use cases like :

* Disaster Recovery, Blast-radius (a problem in one cluster doesn’t kill the whole system)
* Scale out (The application does not fit in a single cluster)
* Jurisdiction (for example: GDPR compliance, keeping data in-country)
* Leveraging resources only available on the public clouds (AI, GPUs, etc..)
* Latency (running the app as close as possible to customer)
* Provider diversity (for regulatory, geographic, data gravity, or other reasons)
* Performance isolation (teams don’t want to feel each other)
* Security isolation (sensitive data or untrusted code)
* Organizational isolation (teams have different management domains)
* Cost isolation (teams get different bills)
* HA across clusters, reliability (a zone or region outage does not bring down the app)


### Goals

* Identifying and defining the APIs to describe how multi-cluster network connectivity should be defined.
* Provide a reference implementation of L3 connectivity across different clusters and network plugins.
* Defining how tunnel endpoint state is reported.
* Explore different technologies to provide the tunnelling across clusters, we are starting with IPSEc but interested
  in expanding that to additional technologies.
* Analyze and ensure that the proposed models are also valid for IPv6.

### Non-Goals

* Solve single Cluster networking challenges
* Focus on a specific network plugin

## Proposal

### User Stories

#### Story 1

##### High level
Two clusters exist with non overlapping CIDRs (cluster A, and cluster B), one on a public cloud,
another on-premises.  L4 connectivity exists between at least edge/gateway nodes of those clusters.
Once connected, pods in one cluster can reach Pod / ClusterIP addresses on the other cluster over L3.

##### Low level
Optionally, when L4 connectivity does not exist between all nodes on both sides of the clusters the administrator
labels the nodes on the clusters which will act as gateways for the traffic, including the technology to be used,
and the external IP/port where those nodes would be reachable, where the external IP/port is not available
-node behind nat-, such information is omitted.

A set of CRDs is calculated and distributed to the clusters, those CRDs contain information on the remote clusters
and the remote endpoints to establish the connectivity.

The connectivity services on those clusters implement the connectivity details (routing, tunneling, etc..),
and cooperate with the existing network plugin where that’s necessary. Note: the connectivity services could be
the network plugin itself.

#### Story 2

##### High level

Several clusters exist with no overlapping CIDRs (cluster A, cluster B, ...), all in different public clouds
and on-premise.
L4 connectivity exists between at least edge/gateway nodes of those clusters, while the nodes on-premise could be
behind NAT translation. Once connected, pods in one cluster can reach Pod/ClusterIP addresses on the other clusters
over L3.

##### Low level

Optionally, when L4 connectivity does not exist between all nodes on both sides of the clusters the administrator
labels the nodes on the clusters which will act as gateways for the traffic, including the technology to be used,
and the external IP where those nodes would be reachable.

Based on those labels, a set of CRDs is calculated and distributed to the clusters, those CRDs contain information
on the remote clusters and the remote endpoints to establish the connectivity

The connectivity services on those clusters implement the connectivity details (routing, tunnelling, etc..), and
cooperate with the existing network plugin where that’s necessary. Note: the connectivity services could be the
network plugin itself.


#### Story 3
##### High level
Two clusters exist with overlapping CIDRs (cluster A, and cluster B). L4 connectivity exists between at least
edge/gateway nodes of those clusters. Once connected, pods in one cluster can reach pods/services on the other
cluster over L3.

##### Low level
Optionally, when L4 connectivity does not exist between all nodes on both sides of the clusters the administrator
labels the nodes on the clusters which will act as gateways for the traffic, including the technology to be used,
and the external IP/port where those nodes would be reachable, where the external IP/port is not available -node
behind nat-, such information is omitted.

A set of CRDs is calculated and distributed to the clusters, those CRDs contain information on the remote clusters
and the remote endpoints to establish the connectivity.
The connectivity services on those clusters implement the connectivity details (routing, tunneling, etc..), and
cooperate with the existing network plugin where that’s necessary. Note: the connectivity services could be the
network plugin itself.


#### Story 4
##### High level
Multiple clusters exist but only some of them become L3 inter-connected. L4 connectivity exists between at
least edge/gateway nodes of those clusters.

#### Low level:
The administrator defines how clusters will connect to each other (Connectivity Policy, label, … TBD).
Optionally, when L4 connectivity does not exist between all nodes on both sides of the clusters the administrator labels the nodes on the clusters which will act as gateways for the traffic, including the technology to be used, and the external IP/port where those nodes would be reachable, where the external IP/port is not available -node behind nat-, such information is omitted.
A set of CRDs is calculated and distributed to the clusters, those CRDs contain information on the remote clusters and the remote endpoints to establish the connectivity
The submariner services on those clusters implement the connectivity details (routing, tunneling, etc..), and cooperate with the existing network plugin where that’s necessary.

### Implementation Details/Notes/Constraints

Implementation commonality between most network plugins allows submariner to work without changes. There are
network plugins which won’t work right away (ovn-kubernetes), or which implement their own multi-cluster network
(cilium, …). In such cases, the submariner implementation will need a layer api to talk to the network plugin.

### Risks and Mitigations

Interconnecting clusters at L3 level introduces security risks, for example a compromised pod could access other
pods/services on other clusters simply via IP connectivity. This is alleviated by the use of Network Policies.

When encrypting traffic: certificates, PSKs, secrets, etc can be used to impersonate clusters on the endpoints,
implementations should use multiple factors to identify the endpoints where possible, for example, IP + secret/id.
Where that is not possible (one side is under NAT), we need to look for detection mechanisms.

## Design Details

### Test Plan

Currently submariner does basic E2E testing by deploying multiple clusters with kind. The E2E testing is written with ginkgo/gomega.

E2E testing will verify:

- [x] Pod to service connectivity across clusters
- [ ] Pod to pod connectivity across clusters
- [x] SRC IP preservation (non overlapping CIDRs)
- [ ] Testing overlapping CIDRs
- [ ] A node is added to a cluster
- [ ] A node is removed from a cluster
- [ ] A cluster is added
- [ ] A cluster is removed
- [ ] Testing for upgrades
- [ ] Testing for no dataplane downtime during upgrades

E2E is tested with kind, and images with various network plugins should be considered for testing.
E2E on public clouds (google, amazon, etc…)

### Graduation Criteria

TBD 

### Upgrade / Downgrade Strategy

TBD

### Version Skew Strategy

TBD

## Implementation History

- Initial document discussed on https://docs.google.com/document/d/1JXQdx60JZPMLywXzA7I6l_N4D5Np9zpMVqPD_8urHAs, now
  read-only.

## Infrastructure Needed

- A new subproject to import github.com/submariner-io/submariner
- Not sure if this is possible with the existing infra, but having some form of deploying multiple k8s, and running
  E2E between major clouds would be beneficial.
