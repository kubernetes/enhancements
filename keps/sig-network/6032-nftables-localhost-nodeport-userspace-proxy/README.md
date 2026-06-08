# KEP-6032: nftables Localhost NodePort Userspace Proxy

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Service Feature Support](#service-feature-support)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Stable](#stable)
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
  - [HostPort DaemonSet Proxy](#hostport-daemonset-proxy)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP adds an opt-in userspace TCP proxy to the nftables kube-proxy backend to serve localhost
NodePort services on IPv4 and IPv6.

## Motivation

Kubernetes plans to make nftables the default kube-proxy backend, but nftables cannot serve localhost
NodePorts. iptables can, via the `route_localnet` kernel setting, but there is no nftables equivalent.
Workloads that depend on `localhost:<NodePort>` cannot migrate to nftables without breaking. An opt-in userspace proxy on nftables closes that gap. See [#132955](https://github.com/kubernetes/kubernetes/issues/132955).

### Goals

- Enable localhost NodePort on nftables (IPv4 and IPv6) as an opt-in via `--nodeport-addresses`

### Non-Goals

- Implement the userspace proxy for iptables or ipvs
- Change any backend's default `--nodeport-addresses`
- Deprecate or stop setting `route_localnet`
- Implement UDP or other non-TCP protocols in the userspace proxy

## Proposal

The nftables kube-proxy backend gains a userspace TCP proxy that listens on `127.0.0.1` / `::1` for each
NodePort and forwards to the service's endpoints.

`--nodeport-addresses` is extended in two ways:

- New keywords `localhost` (loopback CIDRs for the node's IP families) and `all` (zero CIDRs).
- `primary` may now appear alongside other entries instead of being required as the sole value.

The proxy activates whenever `--nodeport-addresses` explicitly includes a loopback address. For instance, it would be activated with `primary,localhost` or `0.0.0.0/0,127.0.0.1/32`, while `all` would not activate it. nftables' default (`primary`) is unchanged.

iptables and ipvs are unchanged. iptables IPv4 continues to use `route_localnet`.

### User Stories

#### Story 1

In-cluster registries in disconnected environments are a popular use case for localhost NodePorts.
[Zarf](https://github.com/zarf-dev/zarf) has a built-in process to stand up a localhost nodeport registry on an airgapped cluster. 
The process is distro agnostic and uses localhost nodeports because popular CRIs (containerd, cri-o) allow insecure 
connections to localhost registries. Without this, operators would need to edit their host node configuration to either include TLS 
certificates or allow insecure connections to node IPs. Today this only works on IPv4 iptables; with this feature, operators can switch to nftables.

### Risks and Mitigations

The proxy is opt-in for nftables and does not change any defaults, so risks are low.

## Design Details

### Service Feature Support

The userspace localhost NodePort proxy is a Layer 4 TCP forwarder. The following covers Service spec fields and how the proxy handles them.

**Implemented:**

- `protocol: TCP` — the proxy creates a listener for every TCP NodePort included in the sync.
- `sessionAffinity: ClientIP` — the proxy pins traffic to a single endpoint for `sessionAffinityConfig.clientIP.timeoutSeconds`. Because all connections originate from `127.0.0.1`/`::1`, ClientIP affinity effectively pins *all* traffic through the listener to one endpoint—not per external client—until the timeout elapses.
- `externalTrafficPolicy: Local` — the proxy restricts its endpoint pool to node-local endpoints. If no local endpoints exist the listener is removed and connections are refused.
- `trafficDistribution` — implemented, endpoints are already categorized before the userspace proxy is created.

**Not implemented:**

- `protocol: UDP` / `protocol: SCTP` — the proxy will not create a listener for these protocols. nftables will not log an error when these services are created since it won't be known if the user plans to connect to them over localhost. If a user tries to connect to a non-TCP protocol service using localhost NodePorts, then an nftables ruleset will reject the packets. Rejections will be tracked by a new metric, `kubeproxy_nftables_localhost_nodeport_rejected_packets_total{protocol}`.

**Not applicable:**

- `internalTrafficPolicy` — governs ClusterIP access; NodePort traffic is governed by `externalTrafficPolicy`.
- Any loadBalancer-specific fields (e.g. `loadBalancerSourceRanges`, `allocateLoadBalancerNodePorts`).

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

Here are the current coverage percentages on master to the relevant packages for this feature. 

- `k8s.io/kubernetes/pkg/proxy/nftables`: `2026-05-12` - `77.5`
- `k8s.io/kubernetes/pkg/proxy/util`: `2026-05-12` - `82.7`

A new package will be introduced `k8s.io/kubernetes/pkg/proxy/localnodeportproxy` with greater than 80% unit test coverage. Changes to existing packages will be unit tested. 

##### Integration tests

No integration tests to be added or updated. E2E tests are sufficient since this feature only affects kube-proxy.

##### e2e tests

E2E tests will be introduced to verify that:
- curling `localhost:<NodePort>` from a node's netns can reach both the local node and a remote node.
- sessionAffinity=ClientIP is honored when the userspace proxy is used.
- pulling an image from a registry served over a localhost NodePort service is successful.

All tests will be implemented behind a feature tag since they'll need nodeport addresses to contain localhost to work properly.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature gate
- E2E tests implemented

#### Beta

- All functionality completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

#### Stable

- Any issues and gaps identified as feedback during beta are resolved

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KubeProxyNFTablesLocalhostNodePorts`
  - Components depending on the feature gate: `kube-proxy`

Toggling the gate requires a kube-proxy restart on the affected nodes.

###### Does enabling the feature change any default behavior?

No. The userspace proxy only activates when an operator includes `localhost` (or a literal loopback CIDR)
in `--nodeport-addresses` on nftables. iptables and ipvs are not modified by this KEP.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate and restarting kube-proxy stops the userspace listeners; existing in-flight connections through the proxy are closed.
The nftables ruleset additions to reject UDP/SCTP traffic to localhost NodePorts will be rolled back when the feature is disabled.
Traffic that previously reached the cluster via `localhost:<NodePort>` will fail until clients fall back to a node IP, but no other workload is affected.

###### What happens if we reenable the feature if it was previously rolled back?

The proxy starts up on the next kube-proxy sync, recreates listeners for any matching NodePort services, and traffic to `localhost:<NodePort>` works again.

###### Are there any tests for feature enablement/disablement?

The "registry on localhost" e2e test will cover the ideal path for feature enablement (FG ON + nftables + `--nodeport-addresses=localhost`). 

Unit tests will cover the decision logic cover the various `--nodeport-addresses` combinations

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout failure looks like kube-proxy on some nodes failing to bind a loopback NodePort (e.g. another host process already owns the port). In that case, the listener for that one service does not come up and the error is logged, but kube-proxy still starts and the rest of the services continue to work.

Rollback has a limited failure mode since it only turns off the userspace proxy.

###### What specific metrics should inform a rollback?

None.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes, manual testing was performed with the following steps:
1. Create an nftables cluster without the `KubeProxyNFTablesLocalhostNodePorts` gate enabled.
1. Create a service with a backend pod.
1. Enable the gate and restart kube-proxy; verify calls to localhost NodePort succeed.
1. Disable the gate and restart kube-proxy; verify calls to localhost NodePort fail.
1. Re-enable the gate and restart kube-proxy; verify calls to localhost NodePort succeed.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The feature can only be used after an operator has explicitly changed their kube-proxy configuration to allow localhost connections on NodePort addresses.
After opting into the feature, operators can view the metric `kubeproxy_nftables_localhost_nodeport_listeners{ip_family}` to see if their users' services have bound to localhost,
but this still does not guarantee that workloads are connecting to services over localhost.

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - Details: The user can know that it is working by trying to connect to their service on localhost.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

We expect [time-to-first-packet](https://github.com/kubernetes/community/blob/main/sig-scalability/slos/first_packet_latency.md) and [throughput](https://github.com/kubernetes/community/blob/main/sig-scalability/slos/throughput.md) to be similar to the old Userspace kube-proxy. We don't have specific requirements as we don't expect very high throughput use cases for localhost NodePorts. 

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `kubeproxy_nftables_localhost_nodeport_listener_creation_failures_total{ip_family}`
  - Components exposing the metric: kube-proxy

A nonzero `kubeproxy_nftables_localhost_nodeport_listener_creation_failures_total` indicates the proxy failed to bind to one or more loopback NodePort listeners.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

We could add a metric similar to the existing `kubeproxy_iptables_localhost_nodeports_accepted_packets_total` to allow operators to view how often the userspace localhost proxy is being hit.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Yes. The userspace proxy adds CPU overhead since each proxied TCP connection passes through the kernel twice per packet (kernel -> userspace -> kernel), unlike the pure in-kernel nftables path.

Memory impact is minimal. Go uses the `splice` system call (see https://go.dev/doc/go1.11#netpkgnet) for TCP-to-TCP copies on Linux, transferring data between socket buffers without copying into userspace. The proxy will only run on Linux where splice is available, so there is no significant per-connection buffering in kube-proxy's memory.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

The userspace proxy will take up sockets. Each proxied connection holds two sockets (one client-side, one upstream), so socket usage is roughly `2 × (total concurrent proxied connections)` across all NodePorts on the node. If no sockets are available the listener goroutine retries without backoff until a descriptor frees up or the listener is torn down. This will cause a CPU spike for however long the fd ceiling remains maxed out.

NodePort services can exhaust the node's port range, but this is already possible with NodePort services and is bounded by the operator's NodePort allocation.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The userspace proxy keeps serving traffic from its last-synced state. Existing listeners stay up, existing connections keep flowing, and endpoint changes are not picked up until kube-proxy can sync again. This matches kube-proxy's existing behavior for the rest of its data path.

###### What are other known failure modes?

- **Loopback port already bound by a host process.**
  - Detection: error log `Failed to listen on 127.0.0.1:<port>: address already in use` from kube-proxy.
  - Mitigations: change the NodePort service or free the port on the host.
  - Diagnostics: kube-proxy logs at error level.
  - Testing: covered by unit tests for the listener constructor.

###### What steps should be taken if SLOs are not being met to determine the problem?

N/A

## Implementation History

- 2026-04-28 first draft created
- 2026-05-18 scoped down to nftables


## Drawbacks

## Alternatives

### HostPort DaemonSet Proxy

Users could instead use hostPort DaemonSet proxies to expose a port on the node, but this does not work on IPv6. `hostNetwork` could be used for IPv6 but introduces a security risk since every pod in the DaemonSet has access to the entirety of the node's network namespace.
