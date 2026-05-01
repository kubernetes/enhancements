<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [X] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [X] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [X] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [X] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-6032: Localhost NodePort Userspace Proxy

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1 (Optional)](#story-1-optional)
    - [Story 2 (Optional)](#story-2-optional)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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

This KEP proposes implementing a userspace proxy to enable localhost NodePort services across all proxy backends. The iptables proxy already works with localhost NodePorts; however, it requires enabling a dangerous permission, `route_localnet`, on each node to allow the kernel to rewrite traffic to localhost. By instead rewriting traffic directly at the userspace level, no extra permissions are set, and localhost NodePorts will work on all proxy backends and IPv6.

## Motivation

`route_localnet` is a dangerous kernel permission set on every node node using IPv4 and iptables by default. Implementing a userspace proxy will enable a secure default without breaking existing use cases. See [#132955](https://github.com/kubernetes/kubernetes/issues/132955).

In-cluster registries in disconnected environments are a popular production use case for localhost NodePorts. The tool [Zarf](https://github.com/zarf-dev/zarf) has a built-in process to start up a registry served over a localhost NodePort on an airgapped cluster. The process is distro and container runtime interface (CRI) agnostic. This is possible because popular CRIs such as containerd and cri-o allow insecure localhost connections to registries. Without this, users would need to edit their host node configuration to either include TLS certificates or allow insecure connections to node IPs. Enabling localhost NodePorts makes it easier for these users to move from iptables to nftables.

### Goals

- Enable localhost NodePort across all proxies and IPv6 without `route_localnet`
- Stop enabling `route_localnet`

### Non-Goals

- Implement UDP or other non-TCP protocol in the userspace proxy

## Proposal

Add a userspace TCP-only proxy to "kube-proxy" that listens on the loopback address (`127.0.0.1` / `::1`) for each NodePort and forwards connections to a service's endpoints. Each backend proxy (`iptables`, `nftables`, `ipvs`) instantiates this proxy when `--nodeport-addresses` contains a loopback CIDR, and reconciles its set of listeners on every sync.

`iptables` and `ipvs` default to `0.0.0.0/0` or `::/0`, allowing all IPs, including localhost, when `--nodeport-addresses` is unset. `nftables` currently defaults to `primary`, accepting connections only to the node IP. `nftables` will shift to accepting `primary,127.0.0.1/32` to include localhost.

This unblocks localhost NodePorts for `nftables` (IPv4 and IPv6), `ipvs` (IPv4 and IPv6), and `iptables` IPv6. `iptables` IPv4 will continue to use `route_localnet` until this feature graduates to GA, then transition to the userspace proxy. The field `--iptables-localhost-nodeports` will be deprecated in favor of explicitly setting `--nodeport-addresses`.

### User Stories

#### Story 1

A cluster operator migrating from `iptables` to `nftables` discovers their workload depends on localhost NodePorts. Today the migration breaks that workload; with userspace localhost NodePorts it keeps working.

### Risks and Mitigations

Some operators may not want localhost NodePort to be reachable in their cluster. These operators can set `--nodeport-addresses=primary`.

## Design Details

Design details are covered in the proposal section.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

All code should be covered.

##### Integration tests

No integration tests to be added or updated. E2E tests are sufficient since this feature only affects kube-proxy.

##### e2e tests

E2E tests are added under the `LocalhostNodePortUserspaceProxy` feature tag to verify that:
- curling `localhost:<NodePort>` from a node's netns can reach both the local node and a remote node.
- sessionAffinity=ClientIP is honored when the userspace proxy is used.

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- E2E tests implemented

#### Beta

- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

#### Stable

- Any issues and gaps identified as feedback during beta are resolved
- `--iptables-localhost-nodeports` is deprecated

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `LocalhostNodePortUserspaceProxy`
  - Components depending on the feature gate: `kube-proxy`

Toggling the gate requires a kube-proxy restart on the affected nodes.

###### Does enabling the feature change any default behavior?

Yes. With the feature gate on, localhost NodePort starts working out of the box on backends where it didn't before:

- **`iptables` IPv4**: unchanged at Alpha. Continues to use `route_localnet`.
- **`iptables` IPv6**: localhost NodePort begins working via the userspace proxy. Previously unsupported (no `route_localnet` for IPv6).
- **`nftables` (both families)**: localhost NodePort begins working via the userspace proxy. To support this, the `nftables` `--nodeport-addresses` default shifts from `primary` to `primary,127.0.0.1/32` so IPv4 works without any operator action. IPv6 still requires the operator to add `::1/128`.
- **`ipvs` (both families)**: localhost NodePort begins working via the userspace proxy. Previously the IPVS service chain dropped localhost-sourced traffic.

At GA, kube-proxy stops setting `route_localnet`; `iptables` IPv4 transitions to the userspace proxy path (matching IPv6), and `--iptables-localhost-nodeports` is deprecated. The `primary` default is not changed for `iptables`.

Operators who do not want localhost NodePort reachable can opt out by explicitly setting `--nodeport-addresses` to a value that excludes loopback (e.g. `--nodeport-addresses=primary`).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate and restarting kube-proxy stops the userspace listeners; existing in-flight connections through the proxy are closed. Traffic that previously reached the cluster via `localhost:<NodePort>` will fail until clients fall back to a node IP, but no other workload is affected.

###### What happens if we reenable the feature if it was previously rolled back?

The proxy starts up on the next kube-proxy sync, recreates listeners for any matching NodePort services, and traffic to `localhost:<NodePort>` works again.

###### Are there any tests for feature enablement/disablement?

For the proxy backends, no, since the userspace proxy is standalone and does not alter existing data.

For the NodePort address changes, yes.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout failure looks like kube-proxy on some nodes failing to bind a loopback NodePort (e.g. another host process already owns the port). In that case the listener for that one service does not come up, but every other service and every other node continue to work.

Rollback has a limited failure mode since it only turns off the userspace proxy.

###### What specific metrics should inform a rollback?

`kubeproxy_localhost_nodeport_listeners{ip_family}` should be > 0 on nodes where the operator expects loopback NodePorts to be reachable. If it stays at 0 after the gate is enabled and matching services exist then a rollback may be required.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes, manual testing was performed with the following steps:
1. Create an `nftables` cluster without the `LocalhostNodePortUserspaceProxy` gate enabled.
1. Create a service with a backend pod.
1. Enable the gate and restart kube-proxy; verify calls to localhost NodePort succeed.
1. Disable the gate and restart kube-proxy; verify calls to localhost NodePort fail.
1. Re-enable the gate and restart kube-proxy; verify calls to localhost NodePort succeed.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes, `--iptables-localhost-nodeports` will be deprecated in favor of controlling localhost NodePorts through `--nodeport-addresses`.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Not possible currently. `kubeproxy_localhost_nodeport_listeners{ip_family}` shows that kube-proxy has bound loopback listeners, but it does not by itself prove that any workload is connecting to them.

###### How can someone using this feature know that it is working for their instance?

- [X] Other (treat as last resort)
  - Details: An operator can scrape `kubeproxy_localhost_nodeport_listeners{ip_family}` from kube-proxy on a node and confirm it equals the number of TCP NodePort services they expect to be reachable on loopback.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name: `kubeproxy_localhost_nodeport_listeners`
  - Components exposing the metric: `kube-proxy`

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

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

A client could create many NodePort services to exhaust the node's port range, but this is already possible with NodePort services and is bounded by the operator's NodePort allocation.

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

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

- 2026-04-28 first draft created

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

### HostPort DaemonSet Proxy

Users could instead use hostPort DaemonSet proxies to expose a port on the node, but this does not work on IPv6. `hostNetwork` could be used for IPv6 but introduces a security risk since every pod in the DaemonSet has access to the entirety of the node's network namespace.
