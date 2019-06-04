---
title: Exclude nodes from load balancers based on an annotation on the node
authors:
  - "@jiatongw"
owning-sig: sig-network
participating-sigs:
  - sig-cloud-provider
reviewers:
  - "@andrewsykim"
  - "@MrHohn"
approvers:
  - "@thockin"
  - "@MrHohn"
editor: TBD
creation-date: 2019-06-03
last-updated: 2019-06-03
status: implementable
see-also:
replaces:
superseded-by:
---

# Exclude Nodes From Load Balancers Based On An Annotation On The Node

## Table of Contents

- [Title](#title)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
  - [Proposal](#proposal)
    - [Risks and Mitigations](#risks-and-mitigations)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
  - [Implementation History](#implementation-history)

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

We will be adding a standardized label that is read by service controller to ensure that only nodes have annotations can be excluded from load balancers. We also plan to add a method to the LoadBalancer interface for more filter nodes options. Meanwhile, we will continue to support the current exclude nodes logic for the next few releases.

## Motivation

Users may choose to deploy a single node cluster and have it associated with load balancers. This can be failed, however, due to the master node is excluded from service load balancers by default (ref discussion on https://github.com/kubernetes/kubernetes/issues/65618). It's worth to have a improved mechanism to support this need.

### Goals

Exclude nodes from load balancers by nodes specific annotations.

## Proposal

We will introduce a new label `networking.kubernetes.io/exclude-service-load-balancer` that read by service controller. Nodes with that label(annotation) will be excluded from service load balancers. 

We also plan to add a `FilterNodes` method to the LoadBalancer interface and have existing providers to filter nodes they wish. This is an optional for cloud-providers. 

We will continue to support the existing behavior for a few releases. So label with ` alpha.service-controller.kubernetes.io/exclude-balancer` or `node-role.kubernetes.io/master` will still not get load balancers traffic as it. Eventually, however, we will not filter nodes with those labels in service load balancers. 

As such, we are proposing Alpha/Beta/GA phase for this enhancement as below:
- Alpha: nodes with label `networking.kubernetes.io/exclude-service-load-balancer`, `alpha.service-controller.kubernetes.io/exclude-balancer` or `node-role.kubernetes.io/master` will be excluded from service load balancers. Nodes are filtered from `FilterNodes` methods will also be excluded. 

- Beta: nodes with label `networking.kubernetes.io/exclude-service-load-balancer` will be excluded from service load balancers. Nodes are filtered from `FilterNodes` methods will also be excluded. Nodes with label `alpha.service-controller.kubernetes.io/exclude-balancer` or `node-role.kubernetes.io/master` will still be excluded from service load balancers but users will get a warning on suggestion of using new label. 

- GA: nodes with label `networking.kubernetes.io/exclude-service-load-balancer` will be excluded from service load balancers. Nodes are filtered from `FilterNodes` methods will also be excluded. 

### Test Plan

We will implement e2e test cases to ensure the new feature works well on various cloud-providers.

### Graduation Criteria

Beta: Allow Alpha to for two releases

GA: TBD

## Implementation History

- 2019-06-03 - Creation of the KEP
