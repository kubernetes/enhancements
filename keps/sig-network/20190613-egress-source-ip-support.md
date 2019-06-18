---
title: egress-source-ip-support
authors:
  - "@mkimuram"
owning-sig: sig-network
participating-sigs:
reviewers:
  - TBD
approvers:
  - TBD
editor: "@mkimuram"
creation-date: 2019-06-13
last-updated: 2019-06-18
status: provisional
see-also:
  - TBD
replaces:
superseded-by:
---

# egress-source-ip-support

## Table of Contents

- [Title](#title)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories [optional]](#user-stories-optional)
      - [Story 1](#story-1)
      - [Story 2](#story-2)
    - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
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

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

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

Egress source IP is a feature to assign a static egress source IP for packets from one or more pods to outside k8s cluster.
The words "outside k8s cluster" here includes both (1) private network where k8s cluster is running and (2) internet that is outside the private network where k8s cluster is running.

## Motivation

In k8s, egress traffic has its source IP translated (SNAT) to appear as the node IP when it leaves the cluster. However, there are many devices and software that use IP based ACLs to restrict incoming traffic for security reasons and bandwidth limitations. As a result, this kind of ACLs outside k8s cluster will block packets from the pod, which causes a connectivity issue. To resolve this issue, we need a feature to assign a particular static egress source IP to one or more particular pods.

Related discussions are done in [here](https://github.com/kubernetes/kubernetes/issues/12265) and [here](https://github.com/cloudnativelabs/kube-router/issues/434).

### Goals

Provide users with an official and common way to assign a static egress source IP for packets from one or more pods to outside k8s cluster.

### Non-Goals

TBD

## Proposal

Expose an egress API to users like below to allow users to assign a static egress source IP to specific pod(s).
With below API, users will be able to make source IPs of pods that have label `app: MyApp` to be 192.168.122.222.

```
apiVersion: v1
kind: Egress
metadata:
  name: my-egress
spec:
  selector:
    app: MyApp
  ip: 192.168.122.222
```

In "(1) private network where k8s cluster is running" case, the ip is restricted to the private network segment where k8s cluster is running.
On the other hand, in "(2) internet that is outside the private network where k8s cluster is running" case, this restriction won't be applied.

Note that PoC implementation for (1) private network where k8s cluster is running is
  - https://github.com/mkimuram/egress-mapper
  - https://github.com/steven-sheehy/kube-egress/pull/1

In the PoC implementation, it uses k8s operator with below CRD. Then, daemonset reconcile the iptables and routing tables rules to SNAT packets from particluar PodIP according to the CRDs. Note that VIPs are assinged to one of the nodes by using keepalived-vip and the packets are SNATed from the node that has matching VIP.

```
apiVersion: egress.mapper.com/v1alpha1
kind: Egress
metadata:
  name: my-egress
spec:
  ip: 192.168.122.222
  kind: pod
  namespace: default
  name: pod1
```

Design details for this KEP version will be discussed in [Design Details](#design-details), and it won't necessary be the same to the PoC one, as long as it meets the requirements above. Also, the PoC implementation doesn't cover "(2) internet that is outside the private network where k8s cluster is running" case, so we will need to find a way to achieve this.

### User Stories [optional]

#### Story 1
As a user of Kubernetes, I have a pod which requires an access to a database that restricts access by source ip and exists outside the k8s cluster. 
So, a pod which requires database access needs a specific egress source IP when sending packets to the database.

#### Story 2
As a user of Kubernetes, I have multiple pods which require an access to a database that restricts access by source ip and exists outside the k8s cluster. 
So, multiple pods which require database access need a specific egress source IP when sending packets to the database.

#### Story 3
As a user of Kubernetes, I have some pods which require an access to different databases that restrict access by source ip and exists outside the k8s cluster. 
So, some pods which require database access need a specific egress source IP when sending packets to the database, and other pods need another specific egress source IP.

### Implementation Details/Notes/Constraints [optional]

TBD

### Risks and Mitigations

TBD

## Design Details

Below are design details that need to be discussed:

1. Reconcile in k8s controller vs external tool (eg. k8s operator)
  * As PoC implementation shows this feature could be implemented as an external tool. However, there are many functions that are duplicated with k8s, so it would be better to leverage existing mechanism, in order to avoid duplicate implementation and conflict between them. Functions that seem duplicated are PodIP tracking functions in service and forwarding rule update functions in kube-proxy.
2. k8s core API vs CRD
  * Both should work. It would depend on whether we choose to implement it as external tool or not.
3. Label based approach vs resource name based approach
  * Resource name is easily be changed. (It is easy to implement as a prototype.) --> Label based approach will be preferred.
4. Fixed implementation vs Pluggable implementation
  * There are already many CNI implementations and kube-proxy implementations that have different behavior inside. For example, some CNI plugin have their own features to achieve it (eg. [Calico case](https://docs.projectcalico.org/v3.7/reference/cni-plugin/configuration#requesting-a-specific-ip-address)) and some kube-proxy implmentations have another way to achieve forwarding packets that might conflict with egress rule. To make any combinations of them work well, we might need to choose pluggable implementation. --> Pluggable implementation will be preferred.

### Test Plan

**Note:** *Section not required until targeted at a release.*

TBD

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

TBD

#### Examples

TBD

##### Alpha -> Beta Graduation

TBD

##### Beta -> GA Graduation

TBD

##### Removing a deprecated flag

TBD

### Upgrade / Downgrade Strategy

TBD

### Version Skew Strategy

TBD

## Implementation History

TBD

## Drawbacks [optional]

TBD

## Alternatives [optional]

TBD

## Infrastructure Needed [optional]

TBD
