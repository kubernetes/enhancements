<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.** sig/apps proposed to make the KEP in https://github.com/kubernetes/kubernetes/pull/98307
- [x] **Create an issue in kubernetes/enhancements** https://github.com/kubernetes/enhancements/issues/3088
- [x] **Make a copy of this template directory.**
- [x] **Fill out as much of the kep.yaml file as you can.**
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.** https://github.com/kubernetes/enhancements/pull/3089
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
# KEP-3088: Support pod disruption budget for DaemonSet pods

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
- [KEP-3088: Support pod disruption budget for DaemonSet pods](#kep-3088-support-pod-disruption-budget-for-daemonset-pods)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories](#user-stories)
      - [Story 1](#story-1)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
      - [How can someone using this feature know that it is working for their instance?](#how-can-someone-using-this-feature-know-that-it-is-working-for-their-instance)
  - [Alternatives](#alternatives)
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
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

Support podDusruptionBudget (PDB) for DaemonSet pods. DaemonSet pods are evicted all at once
ignoring PDB. It creates a load spike for kube-apiserver especially in relatively big clusters. This
KEP proposes to change this behavior so that users could rely on PDB for DaemonSet pods like for all
other built-in controllers.

## Motivation

PodDisruptionBudget can be not only a tool to keep user apps available during evictions, but also a
mean of reducing of load on kube-apiserver. The can be caused by a restart of big number of
DaemonSet pods due to eviction.

Additionally, the disruption controller tracks all built-in pod controllers except DaemonSets when
calculates allowed disruption. This behavior in inconsistent across built-in pod controllers and may
cause confusion.

### Goals

- Support the disruption budget for DaemonSet the same way it works for other built-in pod
  controllers.
- Reduce the load to kube-apiserver caused by the recreation of DaemonSet pods being evicted.

### Non-Goals

- Introduce API changes.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

For now, PDB does not let to control DaemonSet pods evictions. If one creates PDB for DaemonSet
pods, it will have no effect. Effectively, PDB status does not change its initial state regardless
of target Pod conditions. This proposal aims to add the support of DaemonSets in the Disruption
Controller as it is supported for other built-in controllers.

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

In a cluster with big number of nodes, DaemonSet pods can be evicted by VPA. The load of re-creation
of pods is scaled as number of nodes multiplied by number of DaemonSets being affected
simultaneously. In order to reduce the load, one can set a PDB for DaemonSet pods.

For example, in case of 100 nodes and 3 DaemonSets affected by eviction, one would prefer the
simultaneous recreation of 30 pods instead of 300 that could be specified in PDB as
MaxUnavailable=10%.

From the API perspective, PDB status for DaemonSet will contain calculated distuprion-related
numbers, as for any other built-in pod controller.

<!-- ### Notes/Constraints/Caveats (Optional) -->

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

<!-- ### Risks and Mitigations -->

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

Currently, the disruption controller (DC) has a common pattern for built-in Pod controllers (except
DaemonSets). For each PDB it finds the list of targeted pods. For each pod from the list, it finds
its corresponding controller and the desired number of pods. Using the desired number of pods, DC
calculates the allowed disruption number for PDB status.

For all built-in controller types, there are ['finder'
functions](https://github.com/kubernetes/kubernetes/blob/d7123a65248e25b86018ba8220b671cd483d6797/pkg/controller/disruption/disruption.go#L182)
which return the controller UID and the desired number of pods. These 'finder' functions contain the
specifics of each built-in controller, and have their corresponding unit tests. Supported built-in
controller are Deployment, StatefulSet, ReplicaSet, and ResplicationController.

Adding DaemonSet support to PDB requires the implementing the 'finder' function for DaemonSets and
covering it with unit tests to adhere the common pattern. The specific part for DaemonSets is that
the desired number of pods can be taken from `status.DesiredNumberScheduled`.

### Test Plan

- Unit tests in the disruption controller

#### How can someone using this feature know that it is working for their instance?

PodDisruptionStatus for DaemonSet pods would be not N/A, but have reasonable numbers as for any
other kubernetes controller.

## Alternatives

[DaemonSet](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/api/apps/v1/types.go#L629)
does not have scale sub-resource. Implementing it could be another solution to the problems outlined
above. At the same time this path would contradict to current approach where built-in controllers
are intentionally supported via shared informers in the disruption controller ([KEP-981 "Risks and
Mitigations"](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/981-poddisruptionbudget-for-custom-resources#risks-and-mitigations)).
