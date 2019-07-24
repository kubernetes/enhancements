---
title: Node Condition Teardown Pressure & Teardown Priority
authors:
  - "@poelzi"
owning-sig: sig-scheduling
participating-sigs:
reviewers:
  - TBD
  - "@alicedoe"
approvers:
  - TBD
  - "@oscardoe"
editor: TBD
creation-date: 2019-07-15
last-updated: 2019-07-24
status: provisional
---

# Node Condition: Teardown Pressure & Teardown Priority

Make the kubernetes-scheduler aware of teardown costs of containers and prevent overload of nodes.

## Table of Contents

<!-- toc -->
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
      - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
      - [Beta -&gt; GA Graduation](#beta---ga-graduation)
      - [Removing a deprecated flag](#removing-a-deprecated-flag)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
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

For the current implementation of the kubernetes scheduler the teardown phase of the container, which may consist of removing a large amount of files, is considered free. This may result in a buildup of containers in termination phase which will render a node unusable due to io congestion.

This KEP proposes a teardown pressure condition that is triggered when the termination load reaches a critical threshold signaling the scheduler to spare the node any additional workload until the backlog is resolved.

Additionally, to help distribute load better, a TeardownPriority classifier is suggested, that negatively weighs the number of 
containers under removal into the node weight.

## Motivation

While trying to stabilize a very particular workload, we where able to find a corner case in the kubernetes scheduler that can cause runaway overload of one or more nodes, seriously impacting the cluster performance.
The corner case very much depends on the specific hardware setup and is not always easy triggerable, but in general results in a buildup of containers in termination state on one or multiple nodes until they become unresponsive.

  1. After starting of the container, download a very large archive with 10ks of files.
  2. Do something very minor, short lived
  3. Exit

The container then enters the termination phase in which docker removes the files from the overlay filesystem. If the removal process is to slow to finish before next container enters the termination phase, a buildup of those containers can occure.
We found while stressing a [elasticio] workload, that some nodes accumulated 70 containers in termination state, rendering them unusable. Kubernetes-scheduler still scheduled the pods on this node while other nodes had very little load.

This condition happens more often if the filesystem docker is running on is network based.

[elasticio]: https://elastic.io

### Goals

Prevent buildup of containers in termination process and more equally distribution of load in corner cases.

## Proposal

We propose the "Teardown Pressure" condition as a node state that can be set by the kubelet when a threshold of containers in termination state is reached. Unlike other conditions, this will not cause the eviction manager to trigger
pod removal, this condition only notifies the scheduler that the node should no receive any additional workload until the backlog is resolved. Additionally the kubelet should report the number of containers under removal in the RuntimeStats so the scheduler can make a more informed decision.

* The RuntimeStatus message between container-manager and kublet gets enhanced by a ContainersUnderRemoval integer.
* Kubelet gets enhanced by MaxTeardown config variable which will trigger the `NodeConditionType "TeardownPressure"` if the runtime state exceeds the threshold.
* Kubernetes-Scheduler respects the `TeardownPressure` condition in the sense that no new work is scheduled on the node until the flag condition is `false` again.
* Kubernetes-Scheduler gets a teardown priority which negatively weighs the node depending on the number of containers under removal.

### User Stories


### Implementation Details/Notes/Constraints [optional]

We implemented the proposed KEP in this [pull request]. 
We iterated to this relatively slim design to prevent this runaway condition from occuring. Implementing `MaxTeardown` as config variable allows the user to adjust the value to its needs, or even disabling the mechanism by setting it to `0`.
The new teardown default priority class, the average user should see small speedup in certain batch job loads through better spreading. This phenomena is less frequent in clusters of 100+ nodes.

[pull request]: https://github.com/kubernetes/kubernetes/pull/80085

### Risks and Mitigations

If the MaxTeardown value is set to low, the node may flap it's state very often. The default value is set to a value that should not concern normal operation.

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Since runaway effects are highly dependant on the hardware setup, only unit tests seem feasible.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Upgrade / Downgrade Strategy

The `TeardownPressure` will only be considered if all components are updated. Any mixed combination will cause no usage of the condition or negative impact.

### Version Skew Strategy

If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Alternatives

The kubelet could report back all containers together with the 'RemovalInProgress' flag set (currently it sends a filtered list). This information together with local limits could be taken into account. This approach was first tried, but the much leaner approach here was then developed.