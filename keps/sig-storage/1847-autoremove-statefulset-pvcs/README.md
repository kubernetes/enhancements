<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1847: Auto remove PVCs created by StatefulSet

<!--
This is the title of your KEP.  Keep it short, simple, and descriptive.  A good
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
  - [Background](#background)
  - [Changes required](#changes-required)
  - [User Stories (optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Volume reclaim policy for the StatefulSet created PVCs](#volume-reclaim-policy-for-the-statefulset-created-pvcs)
- [Cluster role change for statefulset controller](#cluster-role-change-for-statefulset-controller)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha release](#alpha-release)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary
The proposal is to add a feature to autodelete the PVCs created by StatefulSet.

<!--
This section is incredibly important for producing high quality user-focused
documentation such as release notes or a development roadmap.  It should be
possible to collect this information before implementation begins in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself.  KEP editors, SIG Docs, and SIG PM
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

## Motivation

Currently, the PVCs created automatically by the StatefulSet are not deleted when 
the StatefulSet is deleted. As can be seen by the discussion in the issue 
[55045](https://github.com/kubernetes/kubernetes/issues/55045) there are several use
cases where the PVCs which are automatically created are deleted as well. In many 
StatefulSet use cases, PVCs have a different lifecycle than the pods of the 
StatefulSet, and should not be deleted at the same time. Because of this, PVC 
deletion will be opt-in for users.

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users.  The
motivation section can optionally provide links to [experience reports][] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

### Goals

Provide a feature to auto delete the PVCs created by StatefulSet. 
Ensure that the pod restarts due to non scale down events such as rolling 
update or node drain does not delete the PVC.

<!--
List the specific goals of the KEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->

### Non-Goals

This proposal does not plan to address how the underlying PVs are treated on PVC deletion. 
That functionality will continue to be governed by the ReclaimPolicy of the storage class. 

<!--
What is out of scope for this KEP?  Listing non-goals helps to focus discussion
and make progress.
-->

## Proposal

### Background

Controller `garbagecollector` is responsible for ensuring that when a statefulset 
set is deleted the corresponding pods spawned from the StatefulSet is deleted. 
The `garbagecollector` uses `OwnerReference` added to the `Pod` by statefulset controller
to delete the Pod. Similar mechanism is leveraged by this proposal to automatically 
delete the PVCs created by the StatefulSet controller.

### Changes required

The following changes are required:

1. Add `PersistentVolumeClaimReclaimPolicy` entry into StatefulSet spec inorder to make this feature an opt-in.
2. Provide the following PersistentVolumeClaimPolicies:
   * `Retain` - this is the default policy and is considered in cases where no policy is specified. This would be the existing behaviour - when a StatefulSet is deleted, no action is taken with
       respect to the PVCs created by the StatefulSet.
   * `RemoveOnScaledown` - When a pod is deleted on scale down, the corresponding PVC is deleted as well. 
       A scale up following a scale down, will wait till old PVC for the removed Pod is deleted and ensure 
       that the PVC used is a freshly created one.
   * `RemoveOnStatefulSetDeletion` - PVCs corresponding to the StatefulSet are deleted when StatefulSet
       themselves get deleted.
3. Add `patch` to the statefulset controller rbac cluster role for `persistentvolumeclaims`.

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.  The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories (optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system.  The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1
User environment is such at the content of the PVCs which are created automatically during StatefulSet 
creation need not be retained after the StatefulSet is deleted. User also requires that the scale 
up/down occurs in a fast manner, and leverages any previously existing auto created PVCs within the 
life time of the StatefulSet. An option needs to be provided for the user to auto-delete the PVCs 
once the StatefulSet is deleted. 

User would set the `PersistentVolumeClaimReclaimPolicy` as `RemoveOnStatefulSetDelete` which would ensure that 
the PVCs created automatically during the StatefulSet activation is removed once the StatefulSet 
is deleted.

#### Story 2
User is cost conscious but at the same time can sustain slower scale up(after a scale down) speeds. Needs 
a provision where the PVC created for a pod(which is part of the StatefulSet) is removed when the Pod 
is deleted as part of a scale down. Since the subsequent scale up needs to create fresh PVCs, it will 
be slower than scale ups relying on existing PVCs(from earlier scale ups). 

User would set the `PersistentVolumeClaimReclaimPolicy` as 'RemoveOnScaledown' ensuring PVCs are deleted when corresponding
Pods are deleted. New Pods created during scale  up followed by a scaledown will wait for freshly created PVCs.

### Notes/Constraints/Caveats (optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

This feature applies to PVs which are dynamically provisioned from the volumeClaimTemplate of a StatefulSet. Any PVC and PV provisioned from this mechanism will function with this feature.

### Risks and Mitigations

Currently the PVCs created by statefulset are not deleted automatically. Using the 
`RemoveOnScaledown` or `RemoveOnStatefulSetDeletion` would delete the PVCs 
automatically. Since this involves persistent data being deleted, users should take 
appropriate care using this feature. Having the `Retain` behaviour as default 
will ensure that the PVCs remain intact by default and only a conscious choice 
made by user will involve any persistent data being deleted. Also, PVCs associated with the StatefulSet will be more durable than ephemeral volumes would be, as they are only deleted on scaledown or StatefulSet deletion, and not on other pod lifecycle events like being rescheduled to a new node, even with the new retain policies.

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable.  This may include API specs (though not always
required) or even code snippets.  If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Volume reclaim policy for the StatefulSet created PVCs

When a statefulset spec has a `VolumeClaimTemplate`, PVCs are dynamically created 
using a static naming scheme. A new field named `PersistentVolumeClaimReclaimPolicy` of the 
type `StatefulSetPersistentVolumeClaimReclaimPolicy` will be added to the StatefulSet. This 
field will represent the user indication on whether the associated PVCs can be automatically 
deleted or not. The default policy would be `Retain`. 

If `PersistentVolumeClaimReclaimPolicy` is set to `RemoveOnScaledown`, Pod is set as the owner of the PVCs created
from the `VolumeClaimTemplates` just before the scale down is performed by the statefulset controller. 
When a Pod is deleted, the PVC owned by the Pod is also deleted. When `RemoveOnScaledown` 
policy is set and the Statefulset gets deleted the PVCs also will get deleted 
(similar to `RemoveonStatefulSetDeletion` policy).

Current scaleset controller implementation ensures that the manually deleted pods are restored 
before the scale down logic is run. This combined with the fact that the owner references are set 
only before the scale down will ensure that manual deletions do not automatically delete the PVCs 
in question.

During scale-up, if a PVC has an OwnerRef that does not match the Pod, it 
potentially indicates that the PVC is referred by the deleted Pod and is in the process of 
getting deleted. Controller will exit the current reconcile loop and attempt to reconcile in the 
next iteration. This avoids a race with PVC deletion.

When `PersistentVolumeClaimReclaimPolicy` is set to `RemoveOnStatefulSetDeletion` the owner reference in 
PVC points to the StatefulSet. When a scale up or down occurs, the PVC would remain the same. 
PVCs previously in use before scale down will be used again when the scale up occurs. The PVC deletion 
should happen only after the Pod gets deleted. Since the Pod ownership has `blockOwnerDeletion` set to 
`true` pods will get deleted before the StatefulSet is deleted. The `blockOwnerDeletion` for PVCs will 
be set to `false` which ensures that PVC deletion happens only after the StatefulSet is deleted. This 
chain of ownership ensures that Pod deletion occurs before the PVCs are deleted.


`Retain` `PersistentVolumeClaimReclaimPolicy` will ensure the current behaviour - no PVC deletion is performed as part
of StatefulSet controller.

In alpha release we intend to keep the `PersistentVolumeClaimReclaimPolicy` immutable after creation. 
Based on user feedback we will consider making this field mutable in future releases.

## Cluster role change for statefulset controller
Inorder to update the PVC ownerreference, the `buildControllerRoles` will be updated with 
`patch` on PVC resource.

### Test Plan

1. Unit tests

1. e2e tests
    - RemoveOnScaleDown
      1. Create 2 pod stateful set, scale to 1 pod, confirm PV deleted
      1. Create 2 pod stateful set, add data to PVs, scale to 1 pod, scale back to 2, confirm PV empty
      1. Create 2 pod stateful set, delete stateful set, confirm PVs deleted
      1.Create 2 pod stateful set, add data to PVs, manually delete one pod, confirm pod comes back and PV has data (PV not deleted)
      1. As above, but manually delete all pods in stateful set
      1. Create 2 pod stateful set, add data to PVs, manually delete one pod, immediately scale down to one pod, confirm PV is deleted
      1. Create 2 pod stateful set, add data to PVs, manually delete one pod, immediately scale down to one pod, scale back to two pods, confirm PV is empty
    - RemoveOnStatefulSetDeletion
      1. Create 2 pod stateful set, scale to 1 pod, confirm PV still exists
      1. Create 2 pod stateful set, add data to PVs, scale to 1 pod, scale back to 2, confirm PV has data (PV not deleted)
      1. Create 2 pod stateful set, delete stateful set, confirm PVs deleted
      1. Create 2 pod stateful set, add data to PVs, manually delete one pod, confirm pod comes back and PV has data (PV not deleted)
      1. As above, but manually delete all pods in stateful set
      1. Create 2 pod stateful set, add data to PVs, manually delete one pod, immediately scale down to one pod, confirm PV exists
      1. Create 2 pod stateful set, add data to PVs, manually delete one pod, immediately scale down to one pod, scale back to two pods, confirm PV has data
    - Retain: 
      1. same tests as above, but PVs not removed in any case 

1. Upgrade/Downgrade tests
    1. Create statefulset in previous version and upgrade to the version 
       supporting this feature. The PVCs should remain intact.
    2. Downgrade to earlier version and check the PVCs with Retain
       remain intact and the others with set policies before upgrade 
       gets removed based on if the references were already set.
1. Feature disablement/enable test for alpha feature flag `statefulset-autodelete-pvcs`.

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.  Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

#### Alpha release
- Complete adding the items in the 'Changes required' section.
- Add unit, functional, upgrade and downgrade tests to automated k8s test.


<!--
**Note:** *Not required until targeted at a release.*

#### Alpha -> Beta Graduation

- Gather feedback from developers and users via k8s github issues.
- Add more tests.


Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

There is a new field getting added to the StatefulSet. The upgrade will not 
change the previously expected behaviour of existing Statefulset. 

If the statefulset had been set with the RemoveOnStatefulSetDeletion 
and RemoveOnScaleDown and the version of the kube-controller downgraded,
even though the `PersistentVolumeClaimReclaimPolicy` field will go away, the references
would still be acted upon by the garbage collector and cleaned up 
based on the settings before downgrade. 

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?
-->

### Version Skew Strategy
There is only kubecontroller manager changes involved, hence not applicable for
version skew involving other components.

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: statefulset-autodelete-pvcs
    - Components depending on the feature gate: kube-controller-manager
  
* **Does enabling the feature change any default behavior?**
  The default behaviour is only changed when user explicitly specifies the `PersistentVolumeClaimReclaimPolicy`. 
  Hence no change in any user visible behaviour change by default.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?**
  Yes, but with side effects for users who already started using the feature by means of 
  specifying non-retain `PersistentVolumeClaimReclaimPolicy`. We will an annotation to the
  PVC indicating that the references have been set from previous enablement. Hence a reconcile
  loop which goes through the requried PVCs and removes the references will be added. 
  The side effect is that if there was pod deletion before the references were removed after the
  feature flag was diabled, the PVCs could get deleted.
  
* **What happens if we reenable the feature if it was previously rolled back?** 
The reconcile loop which removes references on disablement will not come into action. Since the 
StatefulSet field would persist through the disablment we will have to ensure that the required
references get set in the next set of reconcile loops.

* **Are there any tests for feature enablement/disablement?**
Feature enablement disablement tests will be added. 

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable, can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/20190731-production-readiness-review-process.md

Production readiness review questionnaire must be completed for features in
v1.19 or later, but is non-blocking at this time. That is, approval is not
required in order to be in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->


## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks
The Statefulset field update is required.
<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives
Users can delete the PVC manually. This is the motivation of the KEP.
<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
