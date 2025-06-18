<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
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
  - [Kubernetes Controller Changes](#kubernetes-controller-changes)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Batch Expand Volumes](#story-1-batch-expand-volumes)
    - [Story 2: Asymmetric Replicas](#story-2-asymmetric-replicas)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
  - [Support for updating arbitrary fields in <code>volumeClaimTemplates</code>](#support-for-updating-arbitrary-fields-in-volumeclaimtemplates)
  - [Patch PVC size regardless of the immutable fields](#patch-pvc-size-regardless-of-the-immutable-fields)
  - [Support for automatically skip not managed PVCs](#support-for-automatically-skip-not-managed-pvcs)
  - [Reconcile all PVCs regardless of Pod revision labels](#reconcile-all-pvcs-regardless-of-pod-revision-labels)
  - [Treat all incompatible PVCs as unavailable replicas](#treat-all-incompatible-pvcs-as-unavailable-replicas)
  - [Integrate with RecoverVolumeExpansionFailure feature](#integrate-with-recovervolumeexpansionfailure-feature)
  - [Order of Pod / PVC updates](#order-of-pod--pvc-updates)
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

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
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
This enhancement proposes relaxing validation of StatefulSet's VolumeClaim template.
Specifically, we will allow modifying the following fields of `spec.volumeClaimTemplates`:
* increasing the requested storage size (`spec.volumeClaimTemplates.spec.resources.requests.storage`)
* modifying Volume AttributesClass used by the claim (`spec.volumeClaimTemplates.spec.volumeAttributesClassName`)
* modifying VolumeClaim template's labels (`spec.volumeClaimTemplates.metadata.labels`)
* modifying VolumeClaim template's annotations (`spec.volumeClaimTemplates.metadata.annotations`)

When `volumeClaimTemplates` is updated, the StatefulSet controller will reconcile the
PersistentVolumeClaims in the StatefulSet's pods.
The behavior of updating PersistentVolumeClaim is similar to updating Pod.
The updates to PersistentVolumeClaim will be coordinated with Pod updates to honor any dependencies between them.

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
This brings many headaches in a continuously evolving environment.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
* Allow users to update some fields of `volumeClaimTemplates` of a `StatefulSet`, specifically:
  * increasing the requested storage size (`spec.volumeClaimTemplates.spec.resources.requests.storage`)
  * modifying Volume AttributesClass used by the claim( `spec.volumeClaimTemplates.spec.volumeAttributesClassName`)
  * modifying VolumeClaim template's labels (`spec.volumeClaimTemplates.metadata.labels`)
  * modifying VolumeClaim template's annotations (`spec.volumeClaimTemplates.metadata.annotations`)
* Add `.spec.volumeClaimUpdatePolicy` allowing users to decide how the volume claim will be updated: in-place or on PVC deletion.


### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
* Support automatic re-creating of PersistentVolumeClaim. We will never delete a PVC automatically.
* Validate the updated `volumeClaimTemplates` as how PVC patch does.
* Update ephemeral volumes.
* Patch PVCs that are different from the template, e.g. StatefulSet adopts the pre-existing PVCs.
* Support for volumes that only support offline expansion.


## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### Kubernetes API Changes  

Change API server to allow specific updates to `volumeClaimTemplates` of a StatefulSet:
   * `spec.volumeClaimTemplates.spec.resources.requests.storage` (increase only)
   * `spec.volumeClaimTemplates.spec.volumeAttributesClassName`
   * `spec.volumeClaimTemplates.metadata.labels`
   * `spec.volumeClaimTemplates.metadata.annotations`

Introduce a new field in StatefulSet `spec`: `volumeClaimUpdatePolicy` to
specify how to coordinate the update of PVCs and Pods. Possible values are:
- `OnClaimDelete`: the default value, only update the PVC when the the old PVC is deleted.
- `InPlace`: patch the PVC in-place if possible. Also includes the `OnClaimDelete` behavior.


Additionally collect the status of managed PVCs, and show them in the StatefulSet status.
Some fields in the `status` are updated to reflect the status of the PVCs:
- currentRevision, updateRevision, currentReplicas, updatedReplicas
  are updated to reflect the status of PVCs.

With these changes, user can still use `kubectl rollout status` to monitor the update process,
both for automated patching and for the PVCs that need manual intervention.

A PVC is considered ready if:
* PVC's `status.capacity.storage` is greater than or equal to min(template spec, PVC spec).
  If the template is 10Gi, PVC is 10Gi and is expanding to 100Gi but failed, we still consider it ready.
* PVC's `status.currentVolumeAttributesClassName` equals to `spec.volumeAttributesClassName`.

A new label `controller-revision-hash` is added to the PVCs,
to ensure we have the correct version of PVC in cache when determining whether the PVC is ready.

### Kubernetes Controller Changes

Additionally watch for events from PVCs, in order to kickoff the update process when the PVC becomes ready.

If `volumeClaimUpdatePolicy` is `OnClaimDelete`, nothing changes. This field acts like a per-StatefulSet feature-gate.
The changes described below applies only for `InPlace` policy.

Include `volumeClaimTemplates` in the `ControllerRevision`.

Since modifying `volumeClaimTemplates` will change the hash,
Add support for updating `controller-revision-hash` label of the Pod without deleting and recreating the Pod,
if the pod template is not changed.

Before creating a new Pod, or, if the Pod template is not changed, updating the label,
use server-side apply to update the PVCs used by the Pod.

The patch used in server-side apply is the volumeClaimTemplates in the StatefulSet, except:
* `spec.resources.requests.storage` is set to max(template `spec.resources.requests.storage`, PVC `spec.resources.requests.storage`),
  so that we will never decrease the storage size.
* `controller-revision-hash` label is added to the PVCs.

Naturally, most of the update control logic also applies to PVCs.
* If `updateStrategy` is `RollingUpdate`, update the PVCs in the order from the largest ordinal to the smallest.
* If `updateStrategy` is `OnDelete`, only update the PVCs if the Pod is deleted manually.
However, `minReadySeconds` is not considered when only PVCs are updated.
because it is hard to determine when the PVC become ready.
And updating PVCs is unlikely to disrupt workloads, so it should be unnecessary to inject delay into the update process.

When creating new PVCs, use the `volumeClaimTemplates` from the same revision that is used to create the Pod.


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

<!--StatefulSet is not allowed to shrink currently.
#### Story 2: Shinking the PV by Re-creating PVC

After running our app for a while, we optimize the data layout and reduce the required storage size.
Now we want to shrink the PVs to save cost.
We can not afford any downtime, so we don't want to delete and recreate the StatefulSet.
We also don't have the infrastructure to migrate between two StatefulSets.
Our app can automatically rebuild the data in the new storage from other replicas.
So we update the `volumeClaimTemplates` of the StatefulSet,
delete the PVC and Pod of one replica, let the controller re-create them,
then monitor the rebuild process.
Once the rebuild completes successfully, we proceed to the next replica. -->

#### Story 2: Asymmetric Replicas

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

When designing the `InPlace` update strategy, we want to reuse the infrastructures controlling Pod rollout.
We apply the changes to the PVCs before we set new `controller-revision-hash` label.
New invariance established about PVCs:
If the Pod has revision A label, all its PVCs are either not existing yet, or updated to revision A and ready.

We introduce `controller-revision-hash` label on PVCs to:
* Record where have progressed, to ensure each PVC is only updated once per rollout.
* When waiting for PVCs to become ready, we can check the label to ensure we got the correct version in the informer cache.

The rational of using server-side apply to update PVCs:
Avoid interference with other controllers or human operators that operate on PVCs.
* If additional annotations/labels are added to the PVCs by others, do not remove them.
* If storage class is not set in the template, We should not care the storage class of the PVCs.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->


Since we don't allow decreasing the storage size of `volumeClaimTemplates`,
it is not possible to run `kubectl rollout undo` after increasing it.
We may loose this restriction in the future.
But unfortunately, since volume expansion cannot be fully cancelled,
undoing StatefulSet changes may not be enough to revert the system to the previous state,
but should be enough to unblock StatefulSet rollout.

The user who can update the StatefulSet gains implicit permission to update the PVCs.
This can incur extra fee to cloud providers.
Cluster administrators should setup appropriate quota or validation to mitigate this.

Interfering with other controllers or human operators.
Over the years, the user may have deployed third-party controllers to e.g., expand the volume automatically.
We should not interfere with them. Like Pods, we use `controller-revision-hash` label to record whether we have updated the PVCs.
If the `controller-revision-hash` label on either Pod or PVC is already matched, we will not touch the PVCs again.
So we will not interfere with them as long as the `controller-revision-hash` label is preserved by them.

New Pod may still see old PVC configuration.
We already ensure that the PVC is updated before the new Pod is created.
However, the operation on PVCs can be asynchronous. And expansion may not finish without a running Pod.


## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

When `volumeClaimUpdatePolicy` is `OnClaimDelete`, APIServer should accept the changes to `volumeClaimTemplates`,
but StatefulSet controller should not touch the PVCs and preserve the current behaviour.
Following describes the workflow when `volumeClaimUpdatePolicy` is `InPlace`.

When updating volumeClaimTemplates along with pod template, we will go through the following steps:
1. Apply the changes to the PVCs used by this replica.
2. Wait for the PVCs to be ready.
3. Delete the old pod.
4. Create the new pod with new `controller-revision-hash` label.
5. Wait for the new pod to be ready.
6. Advance to the next replica and repeat from step 1.

When only updating the volumeClaimTemplates:
1. Apply the changes to the PVCs used by this replica.
2. Wait for the PVCs to be ready.
3. Update the pod with new `controller-revision-hash` label.
4. Advance to the next replica and repeat from step 1.

Assuming we are updating a replica from revision A to revision B:

| Pod | PVC | Action |
| --- | --- | --- |
| not existing | not existing | create PVC at revision B |
| not existing | at revision A | update PVC to revision B |
| not existing | at revision B | create Pod at revision B |
| at revision A | not existing | create PVC at revision B |
| at revision A | at revision A | update PVC to revision B |
| at revision A | at revision B | wait for PVC to be ready, then delete Pod or update Pod label |
| at revision B | not existing | create PVC at revision B |
| at revision B | existing | wait for Pod to be ready |

Note that when Pod is at revision B but PVC is at revision A, we will not update PVC.
Such state can only happen when user set `volumeClaimUpdatePolicy` to `InPlace` when the feature-gate of KCM is disabled,
or disable the previously enabled feature-gate.
We require user to initiate another rollout to update the PVCs, to avoid any surprise.

Failure cases: don't left too many PVCs being updated in-place. We expect to update the PVCs in order.

- If the PVC update fails, we should block the StatefulSet rollout process.
  We should retry and report events for this.
  The events and status should look like those when the Pod creation fails.
  We update PVC before deleting the old Pod, so failure of PVC update should not disrupt running Pods,
  and user should have time to fix this manually.
  The failure cases of this kind includes (but not limited to):
  - immutable fields mismatch (e.g. storageClassName)
  - webhook
  - [storage quota](https://kubernetes.io/docs/concepts/policy/resource-quotas/#storage-resource-quota)
  - [VAC quota](https://kubernetes.io/docs/concepts/policy/resource-quotas/#resource-quota-per-volumeattributesclass)
  - StorageClass.allowVolumeExpansion not set to true

- While waiting for the PVC to become ready,
  We should update status, just like what we do when waiting for Pod to be ready.
  We should block the StatefulSet rollout process if the PVC is never ready.

- When individual PVC failed to become ready, the user can update that PVC manually to bring it back to ready.

- If the `volumeClaimTemplates` is updated again when the previous rollout is blocked,
  similar to [Pods](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#forced-rollback),
  user may need to manually deal with the blocking PVCs (update or delete them).
  - If the PVC cannot become ready because of the old Pod (e.g. unable to schedule),
    user can delete the Pod and the StatefulSet controller will create a new Pod at new revision.

In all cases, if the user determines the failure of updating PVCs is not critical,
he can change `volumeClaimUpdatePolicy` back to `OnClaimDelete` to unblock normal Pod rollout.


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
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

For alpha, the core package we will be touching:
- `pkg/controller/statefulset`: `2025-05-25` - `86.5%`
- `pkg/controller/history`: `2025-05-25` - `84.5`
- `pkg/apis/apps/validation`: `2025-05-25` - `92.5%`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

- When the feature gate is enabled, existing StatefulSets gains a default `volumeClaimUpdatePolicy` of `OnClaimDelete`, and can be updated to `InPlace`.
  Then disable the feature gate, `volumeClaimUpdatePolicy` field should remain unchanged, but user can clear it manually.

- When the feature gate is disabled in the mid of the PVC rollout, we should not update or wait for the PVCs anymore.
  `volumeClaimTemplate` should remains in the controllerRevision. And the current rollout should finish successfully.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.
-->

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

- When feature gate is enabled, update the StatefulSet `volumeClaimTemplates` with `volumeClaimUpdatePolicy: InPlace` can successfully expand the PVCs.
  And running Pods are not restarted.

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
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

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

#### Alpha

- Feature implemented behind a feature flag
- Initial unit, integration and e2e tests completed

#### Beta

- Gather feedback from developers and surveys
- Complete features: StatefulSet status reporting and `kubectl rollout status` support.
- Additional tests are in Testgrid and linked in KEP
- Downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved 

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- 3 examples of real-world usage
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved


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

No changes required to maintain previous behavior.

To make use of the enhancement, user can update `volumeClaimTemplates` of existing StatefulSets.
One can also update `volumeClaimUpdatePolicy` to `InPlace` in order to rollout the changes automatically.

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

No coordinating between the control plane and nodes are required, since this KEP does not involve nodes.

Should enable this feature for APIServer before kube-controller-manager.
An n-1 kube-controller-manager should ignore the `volumeClaimUpdatePolicy` field and never touch PVCs.
It should always create PVCs with the latest `volumeClaimTemplates`.

If `volumeClaimUpdatePolicy` is set to `InPlace` while the kube-controller-manager is down,
when new kube-controller-manager starts, it should pick this up and start rolling out PVCs immediately.

If `volumeClaimUpdatePolicy` is set to `InPlace` when the feature-gate of kube-controller-manager is disabled,
kube-controller-manager should still update the controllerRevision and label on Pods.
After that, when the feature-gate of kube-controller-manager is enabled,
user needs to update the `volumeClaimTemplates` again to trigger another rollout.

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

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
The update to StatefulSet `volumeClaimTemplates` will be accepted by the API server while it is previously rejected.
StatefulSets gains a new field `volumeClaimUpdatePolicy` with default value `OnClaimDelete`.

Otherwise No.
If `volumeClaimUpdatePolicy` is `OnClaimDelete` (the default values),
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
Yes. Since the `volumeClaimTemplates` can already differ from the actual PVCs now,
disable this feature gate should not leave any inconsistent state.

The `volumeClaimUpdatePolicy` field will not be cleared automatically.
When it is set to `InPlace`, `volumeClaimTemplates` also remains in the controllerRevision.
User can rollback each StatefulSet manually by deleting the `volumeClaimUpdatePolicy` field.

###### What happens if we reenable the feature if it was previously rolled back?

If the `volumeClaimUpdatePolicy` is already set to `InPlace`,
user needs to update the `volumeClaimTemplates` again to trigger a rollout.

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
Will add unit tests for the StatefulSet controller with and without the feature gate,
`volumeClaimUpdatePolicy` set to `InPlace` and `OnClaimDelete` respectively.

Will add unit tests for exercising the switch of feature gate when `volumeClaimUpdatePolicy` already set. 

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
CSI drivers with in-place ExpandVolume or ModifyVolume capabilities,
when `spec.resources.requests.storage` or `spec.volumeAttributesClassName` of `volumeClaimTemplates` is updated respectively.


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
- PATCH StatefulSet
  - kubectl or other user agents
- PATCH PersistentVolumeClaim (server-side apply)
  - 1 per PVC in the StatefulSet (number of updated claim template * replica)
  - StatefulSet controller (in KCM)
  - triggered by the StatefulSet spec update

StatefulSet controller will watch PVC updates.
(although statefulset controller does not watch PVCs before, KCM does)


###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
Not directly. The cloud provider may be called when the PVCs are updated, by CSI.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
StatefulSet:
- `spec`: 1 new enum fields, ~10B
PersistentVolumeClaim:
- new label `controller-revision-hash` of size 32B

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
The logic of StatefulSet controller is more complex, more CPU will be used.
TODO: measure the actual increase.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No.

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

Not very different from the current StatefulSet controller workflow.

If the API server and/or etcd is unavailable, we either cannot apply the update to PVCs, or cannot gather status of PVCs.
In both cases, the rollout will be blocked until the API server and/or etcd is available again.

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

- Rollout of the StatefulSet blocked due to failing to update PVCs
  - Detection: apiserver_request_total{resource="persistentvolumeclaims",verb="patch",code!="200"} increased. Events on StatefulSet.
  - Mitigations: 
    - Undo `volumeClaimTemplates` changes
    - Set `volumeClaimUpdatePolicy` to `OnClaimDelete`
  - Diagnostics: Events on StatefulSet
  - Testing: Will test the Event is emitted

- Rollout of the StatefulSet blocked due to PVCs never becomes ready, expansion or modify volume failed
  - Detection: Events on PVC. controller_{modify,expand}_volume_errors_total metrics on external-resizer
  - Mitigations:
    - Undo `volumeClaimTemplates` changes
    - Set `volumeClaimUpdatePolicy` to `OnClaimDelete`
    - Edit PVC manually to correct the issue
  - Diagnostics: Events on PVC, logs of external-resizer
  - Testing: No. the error is already reported on the PVC, by external-resizer.


###### What steps should be taken if SLOs are not being met to determine the problem?

When SLOs are not being met, events of PVC or StatefulSet are emitted.
If problem is not determined from events, operator should check whether the PVC spec is updated correctly.
If so, follow the troubleshooting instructions of expanding or modifying volume.
If not, look into the KCM log to determine why the PVC is not updated, rasing the log level if necessary.

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
- 2024-05-17: initial version
- 2025-06-09: targeting v1.34 for alpha

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
* If we disallow decreasing, we make the editing a one-way road.
  If a user edited it then found it was a mistake, there is no way back.
  The StatefulSet will be broken forever. If this happens, the updates to pods will also be blocked. This is not acceptable.
* To mitigate the above issue, we will want to prevent the user from going down this one-way road by mistake.
  We are forced to do way more validations on APIServer, which is very complex, and fragile (please see KEP-0661).
  For example: check storage class allowVolumeExpansion, check each PVC's storage class and size,
  basically duplicate all the validations we have done to PVC.
  And even if we do all the validations, there are still race conditions and async failures that we are impossible to catch.
  I see this as a major drawback of KEP-0661 that I want to avoid in this KEP.
* Validation means we should disable rollback of storage size. If we enable it later, it can surprise users, if it is not called a breaking change.
* The validation is conflict to RecoverVolumeExpansionFailure feature.
* `volumeClaimTemplates` is also used when creating new PVCs, so even if the existing PVCs cannot be updated,
  a user may still want to affect new PVCs.
* It violates the high-level design.
  The template describes a desired final state, rather than an immediate instruction.
  A lot of things can happen externally after we update the template.
  For example, I have an IaaC platform, which tries to `kubectl apply` one updated StatefulSet + one new StorageClass to the cluster to trigger the expansion of PVs.
  We don't want to reject it just because the StorageClass is applied after the StatefulSet.

### Support for updating arbitrary fields in `volumeClaimTemplates`

No technical limitations. Just that we want to be careful and keep the changes small, so that we can move faster.
This is just an extra validation in APIServer. We may remove it later if we find it is not needed.

### Patch PVC size regardless of the immutable fields

We propose to patch the PVC as a whole, so it can only succeed if the immutable fields matches.

If only expansion is supported, patching regardless of the immutable fields can be a logical choice.
But this KEP also integrates with volumeAttributesClass (VAC). VAC is closely coupled with storage class.
Only patching VAC if storage class matches is a very logical choice.
And we'd better follow the same operation model for all mutable fields.


### Support for automatically skip not managed PVCs

Introduce a new field in StatefulSet `spec.updateStrategy.rollingUpdate`: `volumeClaimSyncStrategy`.
If it is set to `Async`, then we skip patching the PVCs that are not managed by the StatefulSet (e.g. StorageClass does not match).

The rules to determine what PVCs are managed are a little bit tricky.
We have to check each field, and determine what to do for each field.
This makes us deeply coupled with the PVC implementation.

And still, we want to keep the changes small.

### Reconcile all PVCs regardless of Pod revision labels

Like Pods, we only update the PVCs if the Pod revision labels is not the update revision.

We need to unmarshal all revisions used by Pods to determine the desired PVC spec.
Even if we do so, we don't want to send a apply request for each PVC at each reconcile iteration.
We also don't want to replicate the SSA merging/extraction and validation logic, which can be complex and CPU-intensive.


### Treat all incompatible PVCs as unavailable replicas

Currently, incompatible PVCs only blocks the rolling update, not scaling up or down.
Only the update revision is used for checking.

We need to unmarshal all revisions used by Pods to determine the compatibility.
Even if we do so, old StatefulSets do not have claim info in its history.
If we just use the latest version, then all replicas may suddenly become unavailable,
and all operations are blocked.

[KEP-0661]: https://github.com/kubernetes/enhancements/pull/3412

### Integrate with RecoverVolumeExpansionFailure feature

We may decrease the size in PVC spec automatically to help recover from a failed expansion
if `RecoverVolumeExpansionFailure` feature gate is enabled.
However, when reducing the spec size of PVC, it must still be greater than its status (not equal to).
So we don't know what to set if `volumeClaimTemplates` is smaller than PVC status.

User can still update PVC manually.

### Order of Pod / PVC updates

We've considered delete the Pod while/before updating the PVC, but realized several issues:
* The admission of PVC update is fairly complex, it can fail for many reasons.
  We want to make sure the Pod is still running if we cannot update the PVC.
* As described in [KEP-5381], we want to allow affinity change when the VolumeAttributesClass is updated.
  Updating PVC and Pod concurrently may trigger a race condition where the Pod can be scheduled to wrong node.

The current order (wait for PVC ready before delete old Pod) has an extra advantage:
When Pod is ready, it is guaranteed that the PVC is ready too.
So any existing tools to monitor StatefulSet rollout process does not need to change.

This downside is that the concurrency is lower, so the rolling update may take longer.

[KEP-5381]: https://github.com/kubernetes/enhancements/blob/0602a5f744b8e4e201d7bd90eb69e67f1b9baf62/keps/sig-storage/5381-mutable-pv-affinity/README.md#notesconstraintscaveats-optional

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
