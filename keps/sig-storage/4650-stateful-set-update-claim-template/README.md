<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
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
# KEP-4650: StatefulSet Support for Updating Volume Claim Template

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
  - [Kubernetes API Changes](#kubernetes-api-changes)
  - [Updated Reconciliation Logic](#updated-reconciliation-logic)
  - [What PVC is compatible](#what-pvc-is-compatible)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Batch Expand Volumes](#story-1-batch-expand-volumes)
    - [Story 2: Migrating Between Storage Providers](#story-2-migrating-between-storage-providers)
    - [Story 3: Migrating Between Different Implementations of the Same Storage Provider](#story-3-migrating-between-different-implementations-of-the-same-storage-provider)
    - [Story 4: Shinking the PV by Re-creating PVC](#story-4-shinking-the-pv-by-re-creating-pvc)
    - [Story 5: Asymmetric Replicas](#story-5-asymmetric-replicas)
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
  - [Extensively validate the updated <code>volumeClaimTemplates</code>](#extensively-validate-the-updated-volumeclaimtemplates)
  - [Only support for updating storage size](#only-support-for-updating-storage-size)
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

Kubernetes does not support the modification of the `volumeClaimTemplates` of a StatefulSet currently.
This enhancement proposes to support arbitrary modifications to the `volumeClaimTemplates`,
automatically updating the associated PersistentVolumeClaim objects in-place if applicable.
Currently, PVC `spec.resources.requests.storage` and `spec.volumeAttributesClassName`
fields can be updated in-place.
For other fields, we support updating existing PersistentVolumeClaim objects with `OnDelete` strategy.
All the updates to PersistentVolumeClaim can be coordinated with `Pod` updates
to honor any dependencies between them.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

Currently there are very few things that users can do to update the volumes of
their existing StatefulSet deployments.
They can only expand the volumes, or modify them with VolumeAttributesClass
by updating individual PersistentVolumeClaim objects as an ad-hoc operation.
When the StatefulSet scales up, the new PVC(s) will be created with the old
config and this again needs manual intervention.
Modifying immutable parameters, shinking, or even switch to another
storage provider is not possible currently.
This brings many headaches in a continuously evolving environment.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
* Allow users to update the `volumeClaimTemplates` of a `StatefulSet` in place.
* Automatically update the associated PersistentVolumeClaim objects in-place if applicable.
* Support updating PersistentVolumeClaim objects with `OnDelete` strategy.
* Coordinate updates to `Pod` and PersistentVolumeClaim objects.
* Provide accurate status and error messages to users when the update fails.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
* Support automatic rolling update of PersistentVolumeClaim.
* Validate the updated `volumeClaimTemplates` as how PVC update does.
* Update ephemeral volumes.


## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->
1. Change API server to allow any updates to `volumeClaimTemplates` of a StatefulSet.

2. Modify StatefulSet controller to add PVC reconciliation logic.

3. Collect the status of managed PVCs, and show them in the StatefulSet status.

### Kubernetes API Changes

Changes to StatefulSet `spec`:

1. Introduce a new field in StatefulSet `spec`: `volumeClaimUpdateStrategy` to
   specify how to coordinate the update of PVCs and Pods. Possible values are:
   - `OnDelete`: the default value, only update the PVC when the the old PVC is deleted.
   - `InPlace`: update the PVC in-place if possible. Also includes the `OnDelete` behavior.

2. Introduce a new field in StatefulSet `spec.updateStrategy.rollingUpdate`: `volumeClaimSyncStrategy`
   to specify how to update PVCs and Pods. Possible values are:
   - `Async`: the default value, preseve the current behavior.
   - `LockStep`: update PVCs first, then update Pods. See below for details.

Changes to StatefultSet `status`:

Additionally collect the status of managed PVCs, and show them in the StatefulSet status.

For each PVC in the template:
- compatible: the number of PVCs that are compatible with the template.
  These replicas will not be blocked on Pod recreation if `volumeClaimSyncStrategy` is `LockStep`.
- updating: the number of PVCs that are being updated in-place.
- overSized: the number of PVCs that are over-sized.
- totalCapacity: the sum of `status.capacity` of all the PVCs.

Some fields in the `status` are also updated to reflect the staus of the PVCs:
- readyReplicas: in addition to pods, also consider the PVCs status. A PVC is not ready if:
  - `volumeClaimUpdateStrategy` is `InPlace` and the PVC is updating;
- availableReplicas: total number of replicas of which both Pod and PVCs are ready for at least `minReadySeconds`
- currentRevision, updateRevision, currentReplicas, updatedReplicas
  are updated to reflect the status of PVCs.

With these changes, user can still use `kubectl rollout status` to monitor the update process,
both for in-place update and for the PVCs that need manual intervention.

### Updated Reconciliation Logic

How to update PVCs:
1. If `volumeClaimUpdateStrategy` is `InPlace`,
   and if `volumeClaimTemplates` and actual PVC only differ in mutable fields
   (`spec.resources.requests.storage`, `spec.volumeAttributesClassName`, `metadata.labels`, and `metadata.annotations` currently),
   update the PVC in-place to the extent possible.
   Do not perform the update that will be rejected by API server, such as
   decreasing the storage size below its current status.
   Note that decrease the size can help recover from a failed expansion if 
   `RecoverVolumeExpansionFailure` feature gate is enabled.

2. If it is not possible to make the PVC [compatible](#what-pvc-is-compatible),
   do nothing. But when recreating a Pod and the corresponding PVC is deleting,
   wait for the deletion then create a new PVC together with the new Pod (already implemented).
<!--
Tested on Kubernetes v1.28, and I can see this event:
Warning  FailedCreate         3m58s (x7 over 3m58s)  statefulset-controller  create Pod test-rwop-0 in StatefulSet test-rwop failed error: pvc data-test-rwop-0 is being deleted
-->

3. Use either current or updated revision of the `volumeClaimTemplates` to create/update the PVC,
   just like Pod template.

When to update PVCs:
1. If `volumeClaimSyncStrategy` is `LockStep`,
   before advancing `status.updatedReplicas` to the next replica,
   additionally check that the PVCs of the next replica are 
   [compatible](#what-pvc-is-compatible) with the new `volumeClaimTemplates`.
   If not, and we are not going to update it in-place automatically,
   wait for the user to delete/update the old PVC manually.

2. When doing rolling update, A replica is considered ready if the Pod is ready
   and all its volumes are not being updated in-place.
   Wait for a replica to be ready for at least `minReadySeconds` before proceeding to the next replica.

3. Whenever we check for Pod update, also check for PVCs update.
   e.g.:
   - If `spec.updateStrategy.type` is `RollingUpdate`,
     update the PVCs in the order from the largest ordinal to the smallest.
   - If `spec.updateStrategy.type` is `OnDelete`,
     Only update the PVC when the Pod is deleted.
   
4. When updating the PVC in-place, if we also re-create the Pod,
   update the PVC after old Pod deleted, together with creating new pod.
   Otherwise, if pod is not changed, update the PVC only.

Failure cases: don't left too many PVCs being updated in-place. We expect to update the PVCs in order.

- If the PVC update fails, we should block the update process.
  If the Pod is also deleted (by controller or manually), don't block the creation of new Pod.
  We should retry and report events for this.
  The events and status should look like those when the Pod creation fails.

- While waiting for the PVC to reach the compatible state,
  We should update status, just like what we do when waiting for Pod to be ready.
  We should block the update process if the PVC is never compatible.

- If the `volumeClaimTemplates` is updated again when the previous rollout is blocked,
  similar to [Pods](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#forced-rollback),
  user may need to manually deal with the blocking PVCs (update or delete them).


### What PVC is compatible

A PVC is compatible with the template if:
- All the immutable fields match exactly; and
- `metadata.labels` and `metadata.annotations` of PVC is a superset of the template; and
- `status.capacity.storage` of PVC is greater than or equal to
  the `spec.resources.requests.storage` of the template; and
- `status.currentVolumeAttributesClassName` of PVC is equal to
  the `spec.volumeAttributesClassName` of the template.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1: Batch Expand Volumes

We're running a CI/CD system and the end-to-end automation is desired.
To expand the volumes managed by a StatefulSet,
we can just use the same pipeline that we are already using to update the Pod.
All the test, review, approval, and rollback process can be reused.

#### Story 2: Migrating Between Storage Providers

We decide to switch from home-made local storage to the storage provided by a cloud provider.
We can not afford any downtime, so we don't want to delete and recreate the StatefulSet.
Our app can automatically rebuild the data in the new storage from other replicas.
So we update the `volumeClaimTemplates` of the StatefulSet,
delete the PVC and Pod of one replica, let the controller re-create them,
then monitor the rebuild process.
Once the rebuild completes successfully, we proceed to the next replica.

#### Story 3: Migrating Between Different Implementations of the Same Storage Provider

Our storage provider has a new version that provides new features, but can not be upgraded in-place.
We can prepare some new PersistentVolumes using the new version, but referencing the same disk
from the provider as the in-use PVs.
Then the same update process as Story 2 can be used.
Although the PVCs are recreated, the data is preserved, so no rebuild is needed.

#### Story 4: Shinking the PV by Re-creating PVC

After running our app for a while, we optimize the data layout and reduce the required storage size.
Now we want to shrink the PVs to save cost.
The same process as Story 2 can be used.

#### Story 5: Asymmetric Replicas

The storage requirement of different replicas are not identical,
so we still want to update each PVC manually and separately.
Possibly we also update the `volumeClaimTemplates` for new replicas,
but we don't want the controller to interfere with the existing replicas.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

When designing the `InPlace` update strategy, we update the PVC like how we re-create the Pod.
i.e. we update the PVC whenever we would re-create the Pod;
we wait for the PVC to be compatible whenever we would wait for the Pod to be ready.

`volumeClaimSyncStrategy` is introduce to keep capability of current deployed workloads.
StatefulSet currently accepts and uses existing PVCs that is not created by the controller,
So the `volumeClaimTemplates` and PVC can differ even before this enhancement.
Some users may choose to keep the PVCs of different replicas different.
We should not block the Pod updates for them.

If `volumeClaimSyncStrategy` is `Async`,
we just ignore the PVCs that cannot be updated to be compatible with the new `volumeClaimTemplates`,
as what we do currently.
Of course, we report this in the status of the StatefulSet.

However, a workload may rely on some features provided by a specific PVC,
So we should provide a way to coordinate the update.
That's why we also need `LockStep`.

The StatefulSet controller should also keeps the current and updated revision of the `volumeClaimTemplates`,
so that a `LockStep` StatefulSet can still re-create Pods and PVCs that are yet-to-be-updated.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->
TODO: Recover from failed in-place update (insufficient storage, etc.)
What else is needed in addition to revert the StatefulSet spec?

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

We can use Server Side Apply to update the PVCs in-place,
so that we will not interfere with the user's manual changes,
e.g. to `metadata.labels` and `metadata.annotations`.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: StatefulSetUpdateVolumeClaimTemplate
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
No.
If `volumeClaimUpdateStrategy` is `OnDelete` and `volumeClaimSyncStrategy` is `Async` (the default values),
the behavior of StatefulSet controller is almost the same as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
### Extensively validate the updated `volumeClaimTemplates`

[KEP-0661] proposes that we should do extensive validation on the updated `volumeClaimTemplates`.
e.g., prevent decreasing the storage size, preventing expand if the storage class does not support it.
However, this have saveral drawbacks:
* Not reverting the `volumeClaimTemplates` when rollback the StatefulSet is confusing, 
* The validation can be a barrier when recovering from a failed update.
  If RecoverVolumeExpansionFailure feature gate is enabled, we can recover from failed expansion by decreasing the size.
* The validation is racy, especially when recovering from failed expansion.
  We still need to consider most abnormal cases even we do those validations.
* This does not match the pattern of existing behaviors.
  That is, the controller should take the expected state, retry as needed to reach that state.
  For example, StatefulSet will not reject a invalid `serviceAccountName`.
* `volumeClaimTemplates` is also used when creating new PVCs, so even if the existing PVCs cannot be updated,
  a user may still want to affect new PVCs.

### Only support for updating storage size

[KEP-0661] only enables expanding the volume by updating `volumeClaimTemplates[*].spec.resources.requests.storage`.
However, because the StatefulSet can take pre-existing PVCs,
we still need to consider what to do when template and PVC don't match.
The complexity of this proposal will not decrease much if we only support expanding the volume.

By enabling arbitrary updating to the `volumeClaimTemplates`,
we just acknowledge and officially support this use case.

[KEP-0661]: https://github.com/kubernetes/enhancements/pull/3412

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
