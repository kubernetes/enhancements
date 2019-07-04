---
title: Instance Host Labelling
authors:
  - "@alculquicondor"
owning-sig: sig-cloud-provider
participating-sigs:
  - sig-scheduling
reviewers:
  - TBD
approvers:
  - TBD
editor: TBD
creation-date: 2019-07-04
last-updated: 2019-07-04
status: provisional
see-also:
  - "/keps/sig-scheduling/20190221-even-pods-spreading.md"
  - "https://github.com/kubernetes/enhancements/pull/839"
replaces: []
superseded-by: []
---

# Instance Host Labelling

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

Many kubernetes clusters are deployed on virtual machines (also referred to as "instances"),
and information about the physical host for those instances (also referred to as "instance host")
is not consistently available.
Instances which share a host can fail simultaneously if there is a problem with the host.

We propose to add a well-known label for nodes to be filled with the instance host ID.
This label can then be used by controllers (namely, the scheduler) to implement features such as inter-pod affinity and anti-affinity and even pod spreading.

## Motivation

Today, well-known labels for zone and region can be used as topology keys for inter-pod affinity and anti-affinity.
There is also ongoing work to implement even pod spreading, for which these keys can also be used.

We want to provide more built-in options to improve reliability of workloads running in a kubernetes cluster.
An instance host can fail and thus disrupting all of its pods, so it should be identified as a topology or failure domain as well.

In addition to reliability, having a well-defined label for the instance host increases the portability of pod specs for customers that have on-prem and cloud deployments.

### Goals

- Introduce a well known label to identify the instance host of a node.
- Define a default value for the label so that components maintain their current behavior if they use the label.

### Non-Goals

- Add a cloud provider API to get instance host ID.
- Populate the instance host label with the instance host ID.
- Controller behavior (e.g. deployment, statefulset) that considers instance host label.
- Any decision-making that is based on the physical host label.

## Proposal

Add a well-known label for instance host: `topology.kubernetes.io/instance-host` and make its default value empty.

The label will be named `failure-domain.kubernetes.io/instance-host` if KEP [Promoting cloud provider labels to GA](https://github.com/kubernetes/enhancements/pull/839) doesn’t land.


### User Stories

#### Story 1

As an application developer, I want pods to be scheduled in the same instance host so that networking between them is faster.
I use `topology.kubernetes.io/instance-host` as topology key in [inter-pod affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity-beta-feature) rules.

#### Story 2

As an application developer, I want pods to be spread in different instance hosts so that if the host goes down, my service doesn’t have to scale from zero.
I use `topology.kubernetes.io/instance-host` as topology key in [even pod spreading](/keps/sig-scheduling/20190221-even-pods-spreading.md) rules.

### Implementation Details/Notes/Constraints

- The label will initially be named `topology.beta.kubernetes.io/instance-host` before its promoted to GA.

- If the label is not explicitly used in inter-pod affinity or anti-affinity, current behavior in the scheduler remains.

    There is one case worth mentioning:
    An empty topology key for `preferredDuringSchedulingIgnoredDuringExecution` in inter-pod anti-affinity is interpreted as
    "all topologies", so the instance host should be considered as well.
    If the label value is empty, then the anti-affinity is still ensured by the hostname.
    Additionally, a node [should always be considered in the same topology as itself](https://groups.google.com/d/msg/kubernetes-sig-cloud-provider/32N59IYXogY/FXIUHeYWDwAJ), regardless of the labels.

### Risks and Mitigations

- Risk: Some or all nodes are missing a value for the instance host label and an application developer tries to use the label as a topology key.

  Mitigation: This will be documented as undefined behavior.
  
- Risk: If a cloud provider or hypervisor performs a live migration of a VM, the host will change.

  Mitigation: The label should be updated. Evicting running pods is out-of-scope for the scheduler.
  New pods are scheduled according to the new labels.


## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

### Upgrade / Downgrade Strategy

TBD

### Version Skew Strategy

TBD

## Implementation History

- 2019-07-04: Initial KEP sent out for review (Summary, Motivation, Proposal, Drawbacks and Alternatives)

## Drawbacks

Cloud providers or other stakeholders might argue in favor of adding more well-known labels to identify other topologies (such as rack).
Adding instance host label as a well-known label might blur the criteria for making certain topology standard versus custom or provider-specific.

We argue that all cloud providers (and most on-prem clusters) have their nodes deployed as VMs running on physical hosts.
Other topology terms don't have such clear definition.
Thus, the instance host topology should be part of the known topologies.

## Alternatives

Do nothing.
Cloud providers and on-prem clusters could support this topology by defining their own label and controllers to populate it.
