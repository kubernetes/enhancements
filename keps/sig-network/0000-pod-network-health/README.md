# Pod Network Health API

## Summary
Kubernetes currently lacks a native mechanism to represent basic
pod-to-pod network health such as reachability and latency.
This KEP proposes a Kubernetes-native API to express these signals
in a standardized and extensible way.

## Motivation
Network issues are one of the most common causes of outages in Kubernetes.
Today, operators rely on ad-hoc scripts, CNI-specific tools, or
external observability systems to diagnose pod-to-pod connectivity issues.

A standardized API enables:
- Faster diagnosis of networking issues
- Vendor-neutral observability
- Better tooling integration

## Goals
- Define a Kubernetes-native abstraction for pod network health
- Represent basic signals such as reachability and latency
- Remain CNI-agnostic and implementation-neutral
- Introduce the API as alpha behind a feature gate

## Non-Goals
- Deep packet inspection
- Mandatory probing behavior
- Automatic remediation
- Replacing service meshes or observability platforms

## User Stories

### Cluster Operator
As a cluster operator, I want to know whether two pods can communicate
so that I can debug outages faster.

### Platform Engineer
As a platform engineer, I want a standard API to surface network health
signals that can be consumed by monitoring systems.

## Proposal
Introduce an alpha Kubernetes API resource that represents observed
network health between a source pod and a target pod.

The API focuses on **representation**, not how data is collected.

## API Design (High-Level)
The API may include:
- Source pod reference
- Target pod reference
- Reachability status
- Optional latency metrics
- Timestamp of last observation

Exact fields will be refined during review.

## Implementation Details
- Introduced as alpha
- Feature gated
- No default probing required
- Implementations may be controller-based, node-based, or vendor-provided

## Why a Core Kubernetes API?

While pod-to-pod network health can be measured using an external
controller and CRD, the motivation for a core Kubernetes API is to
standardize semantics across clusters, CNIs, and tooling.

Existing tools (for example, kube-latency:
https://github.com/simonswine/kube-latency) demonstrate that connectivity
and latency can be measured externally. However, these tools define
their own schemas and reporting mechanisms, making it difficult for
operators and platforms to build portable integrations.

A core API would define a common contract and vocabulary, while allowing
actual data collection and implementation to remain pluggable and
external. As a validation step, this proposal is compatible with first
prototyping the design as an external controller + CRD before considering
promotion into core Kubernetes.


## Alternatives Considered
- CNI-specific tooling (not portable)
- External observability systems (not Kubernetes-native)
- CLI-only debugging tools (not programmatic)

## Risks and Mitigations
**API stability risk**  
Mitigated by alpha status and feature gating.

**Performance impact**  
Mitigated by avoiding mandatory probing.

## Graduation Criteria

### Alpha
- API introduced behind feature gate
- Experimental usage

### Beta
- Feedback from users
- Stable semantics

### GA
- Production usage
- Documented best practices

## References
- SIG-Network discussions (TBD)
