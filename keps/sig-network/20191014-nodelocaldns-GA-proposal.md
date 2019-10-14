---
title: NodeLocal DNSCache GA Proposal
authors:
  - "@prameshj"
owning-sig: sig-network
reviewers:
  - "@bowei"
  - "@thockin"
  - "@johnbelamaric"
approvers:
creation-date: 2019-10-14
last-updated: 2019-10-14
status: implementable
see-also:
  - "/keps/sig-network/20190424-NodeLocalDNS-beta-proposal.md"
  - "/keps/sig-network/0030-nodelocal-dns-cache.md"
---

# Graduate NodeLocal DNSCache to GA


## Table of Contents

A table of contents is helpful for quickly jumping to sections of a KEP and for highlighting any additional information provided beyond the standard KEP template.
[Tools for generating][] a table of contents from markdown are available.

- [Title](#title)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Graduation Criteria](#graduation-criteria)
  - [Implementation History](#implementation-history)


[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
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

NodeLocal DNSCache is a feature that runs a caching agent on each cluster node(as a daemonset). Pods on the same node talk to the local agent for DNS queries. This avoids connection tracking and iptables NAT for all cached DNS responses. The feature has significantly [reduced DNS timeouts](https://github.com/kubernetes/kubernetes/issues/56903#issuecomment-511772954) in production clusters.
The purpose of this proposal is to graduate NodeLocal DNSCache to GA.

## Motivation

* NodeLocal DNSCache mitigates a number of issues related to [connection tracking/NAT in k8s DNS](https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/0030-nodelocal-dns-cache.md#motivation).
* It has been Alpha since k8s 1.13, Beta since k8s 1.15 and the user-feedback has been positive.
* It uses CoreDNS(the default clusterDNS in k8s) as the caching agent.

### Goals

* Bump up NodeLocal DNSCache to be GA.
* Upgrade to a newer CoreDNS version(1.16.x) in [node-cache](https://github.com/kubernetes/dns/pull/328).
* Provide an HA solution that works in IPTables and IPVS mode of kube-proxy. Options were described in the [previous KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/20190424-NodeLocalDNS-beta-proposal.md#design-details).

### Non-Goals

* Making NodeLocal DNSCache enabled by default. Since this fixes DNS issues that crop up when clusterDNS is used at scale, this will be left as an opt-in feature.

## Proposal

HA was a common ask when NodeLocal DNSCache feature was introduced in Alpha. The proposed solution here is to provide the option to user to select one of the HA modes described in the [previous KEP](https://github.com/kubernetes/enhancements/blob/master/keps/sig-network/20190424-NodeLocalDNS-beta-proposal.md#design-details). The yaml to deploy either 2 daemonsets using the link-local IP or reuse the kube-dns service VIP for NodeLocal DNSCache will be provided, as part of graduating the feature to GA.

### Graduation Criteria
- Ensure that Kubernetes [e2e tests with NodeLocal DNSCache](https://k8s-testgrid.appspot.com/sig-network-gce#gci-gce-kube-dns-nodecache) are passing.
- Scalability tests with NodeLocal DNSCache enabled, verifying the HA modes as well as the regular mode.
- Have N clusters(number TBD) running in production with NodeLocal DNSCache enabled.

## Implementation History

-2019-10-14 Initial KEP sent for review.



