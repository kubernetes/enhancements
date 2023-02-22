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
# KEP-3756: Robust VolumeManager reconstruction after kubelet restart
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Introduction](#introduction)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Proposed VolumeManager startup](#proposed-volumemanager-startup)
  - [Old VolumeManager startup](#old-volumemanager-startup)
    - [Observability](#observability)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

After kubelet is restarted, it looses track of all volume it mounted for
running Pods. It tries to restore this state from the API server, where kubelet
can find Pods that _should_ be running, and from the host's OS, where it can
find actually mounted volumes. We know this process is imperfect.
This KEP tries to rework the process. While the work is technically a bugfix,
it changes large parts of kubelet, and we'd like to have it behind a feature
gate to provide users a way to get to the old implementations in case of
problems.

This work started as part of
[KEP 1790](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1710-selinux-relabeling)
and even went alpha in v1.26, but we'd like to have a separate feature + feature
gate to be able to graduate VolumeManager reconstruction faster.

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

## Motivation

### Goals

* During kubelet startup, allow it to populate additional information about
  _how_ are existing volumes mounted.
  [KEP 1710](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1710-selinux-relabeling)
  needs to know what mount options did the previous kubelet used when mounting
  the volumes, to be able to tell if they need any change or not.
* Fix [#105536](https://github.com/kubernetes/kubernetes/issues/105536): Volumes
  are not cleaned up (unmounted) after kubelet restart, which needs a similar
  VolumeManager refactoring.
* In general, make volume cleanup more robust.

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

## Introduction

*VolumeManager* is a piece of kubelet that mounts volumes that should be
mounted (i.e. a Pod that needs the volume exists) and unmounts volumes that are
not needed any longer (all Pods that used them were deleted).

VolumeManager keeps two caches:
* *DesiredStateOfWorld* (DSW) contains volumes that should be mounted.
* *ActualStateOfWorld* (ASW) contains currently mounted volumes.
  A volume in ASW can be marked as:
  * Globally mounted - it is mounted in `/var/lib/kubelet/volumes/<plugin>/...`
    * This mount is optional and depends on volume plugin / CSI driver
      capabilities. If it's supported, each volume has only a single global
      mount.
  * Mounted into Pod local directory - it is mounted in
    `/var/lib/kubelet/pods/<pod UID>/volumes/...`. Each pod that uses a volume
    gets its own local mount, because each pod has a different `<pod UID>`.
    If the volume plugin / CSI driver supports the global mount mentioned above,
    each pod local mount is typically a bind-mount from the global mount.

  In addition, both global and local mounts can be marked as *uncertain*, when
  kubelet is not 100% sure if the volume is fully mounted there. Typically,
  this happens when a CSI driver times out NodeStage / NodePublish calls
  and kubelet can't be sure if the CSI driver has finished mounting the volume
  *after* the timeout. Kubelet then needs to call NodeStage / NodePublish again
  if the volume is still needed by some Pods, or call NodeUnstage /
  NodeUnpublish if all Pods that needed the volume were deleted.

VolumeManager runs two separate goroutines:
* *[reconciler](https://github.com/kubernetes/kubernetes/blob/44b72d034852eb6da8916c82ce722af604b196c5/pkg/kubelet/volumemanager/reconciler/reconciler.go#L47-L69)
  that periodically compares ASW and DSW and tries to move ASW towards DSW.
* *DesiredStateOfWorldPopulator* (DSWP) that
  [periodically lists Pods from PodManager and adds them to DSW](https://github.com/kubernetes/kubernetes/blob/cca3d557e6ff7f265eca8517d7c4fa719077c8d1/pkg/kubelet/volumemanager/populator/desired_state_of_world_populator.go#L175-L189).
  This DSWP is marked as `hasAddedPods=true` ("fully populated") only after
  it has read all Pods from files (static pods) **and** the API server (i.e.
  [`sourcesReady.AllReady` returns `true` here](https://github.com/kubernetes/kubernetes/blob/cca3d557e6ff7f265eca8517d7c4fa719077c8d1/pkg/kubelet/volumemanager/populator/desired_state_of_world_populator.go#L150-L159)).

Both ASW and DSW caches exist only in memory and are lost when kubelet process
dies. It's relatively easy to populate DSW - just list all Pods from the API
server and static pods and collect their volumes. Populating ASW is complicated
and actually source of several problems that we want to change in this KEP.

*Volume reconstruction* is a process where kubelet tries to create a single
valid `PersistentVolumeSpec` or `VolumeSpec` for a volume from the OS.
Typically from mount table by looking at what's mounted at
`/var/lib/kubelet/pods/*/volumes/XYZ`. This process is imperfect,
it populates only `(Persistent)VolumeSpec` fields that are necessary to unmount
the volume (i.e. to call `volumePlugin.TearDown` + `UnmountDevice` calls).

Today, kubelet populates VolumeManager's DSW first, from static Pods and pods
received from the API server. ASW is populated from the OS
after DSW is fully populated (`hasAddedPods==true`) and **only volumes missing
in DSW are added there**. In other words, kubelet reconstructs only the volumes
for Pods that were running, but were deleted from API server before kubelet
started. (If the pod is still in the API server, Running, its volumes would be
in DSW).

We assumed that this was enough, because if a volume is in DSW, the
VolumeManager will try to mount the volume, and it will eventually reach ASW.

We needed to add
[a complex workaround](https://github.com/kubernetes/kubernetes/pull/110670)
to actually unmount a volume if it's initially in DSW, but user deletes all
Pods that need it before the volume reaches ASW.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

We propose to reverse the kubelet startup process.

1. Quickly reconstruct ASW from the OS and add **all** found volumes to ASW
   when kubelet starts as *uncertain*. "Quickly" means the process should look
   only at the OS and files/directories in `/var/lib/kubelet/pods` and it should
   not require the API server or any network calls. Esp. the API server may
   not be available at this stage of kubelet startup.
2. In parallel to 1., start DSWP and populate DSW from the API server and
   static pods.
3. When connection to the API server becomes available, complete reconstructed
   information in ASW with data from the API server (e.g. from `node.status`).
   This typically happens in parallel to the previous step.

Benefits:

* All volumes are reconstructed from the OS. As result, ASW can contain the
  real information how are the volumes mounted, e.g. their mount options.
  This will help with
  [KEP 1710](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1710-selinux-relabeling).
* Some issues become much easier to fix, e.g.
  * [#105536](https://github.com/kubernetes/kubernetes/issues/105536)
  * We can remove workarounds for
    [#96635](https://github.com/kubernetes/kubernetes/issues/96635)
    and [#70044](https://github.com/kubernetes/kubernetes/issues/70044),
    they will get fixed naturally by the refactoring.

We also propose to split this work out of
[KEP 1710](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1710-selinux-relabeling),
as it can be useful outside of SELinux relabeling and could graduate separately.
to split the feature, we propose feature gate `NewVolumeManagerReconstruction`.

### User Stories (Optional)

#### Story 1

(This is not a new story, we want to keep this behavior)

As a cluster admin, I want kubelet to resume where it stopped when it was
restarted or its machine was rebooted, so I don't need to clean up / unmount
any volumes manually.

It must be able to recognize what happened in the meantime and either unmount
any volumes of Pods that were deleted in the API server or mount volumes for
newly created Pods.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

The whole VolumeManager startup was rewritten as part of
[KEP 1710](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1710-selinux-relabeling).
It can contain bugs that are not trivial to find, because kubelet can be used
in number of situations that we don't have in CI. For example, we found out
(and fixed) a case where the API server is actually a static Pod in kubelet
that is starting. We don't know what other kubelet configurations people use,
so we decided to write a KEP and move the new VolumeManager startup behind
a feature gate.

## Design Details

This section serves as a design document of the proposed *and* the old
VolumeManager startup + volume reconstruction during that.

### Proposed VolumeManager startup

When kubelet starts, VolumeManager starts DSWP and reconciler
[in parallel](https://github.com/kubernetes/kubernetes/blob/575616cc72dbfdd070ead81ec29c0d4f00226487/pkg/kubelet/volumemanager/volume_manager.go#L288-L292).

However, the first thing that the reconciler does before reconciling DSW and ASW
is that it scans `/var/lib/kubelet/pods/*` and reconstructs **all** found
volumes and adds them to ASW as *uncertain*. Only information that is available
[in the Pod directory on the disk are reconstructed into ASW at this point.
* Since the volume reconstruction can be imperfect and can miss `devicePath`,
]()  VolumeManager adds all reconstructed volumes to `volumesNeedDevicePath`
  array, to finish their reconstruction from `node.status.volumesAttached`
  later.
* All volumes that failed reconstruction are added to
  `volumesFailedReconstruction` list.

After **ASW** is populated, reconciler starts its
[reconciliation loop](https://github.com/kubernetes/kubernetes/blob/cca3d557e6ff7f265eca8517d7c4fa719077c8d1/pkg/kubelet/volumemanager/reconciler/reconciler_new.go#L33-L69):
1. `mountOrAttachVolumes()` - mounts (and attaches, if necessary) volumes that
   are in DSW, but not in ASW. This can happen even before DSW is fully
   populated.

2. `updateReconstructedDevicePaths()` - once kubelet gets connection to the API
   server and reads its own `Node.status`, volumes in `volumesNeedDevicePath`
   (i.e. all reconstructed volumes) are updated from
   `node.status.attachedVolumes`, overwriting any previous `devicePath` in
   *uncertain* mounts (i.e. potentially overwriting the reconstructed
   `devicePath` or even `devicePath` from `MountDevice` / `SetUp` that ended
   as *uncertain* (e.g. timed out). This happens only once,
   `volumesNeedDevicePath` is cleared afterwards.

3. (Only once): Add all reconstructed volumes to `node.status.volumesInUse`.

4. Only after DSW was fully populated (i.e. VolumeManager can tell if a volume
   is really needed or not), **and** `devicePaths` were populated from
   `node.status`, VolumeManager can start unmounting volumes and calls:
   1. `unmountVolumes()` - unmounts pod local volume mounts (`TearDown`) that
      are in ASW and are not in DSW.
   2. `unmountDetachDevices()` - unmounts global volume mounts (`UnmountDevice`)
      of volumes that are in ASW and are not in DSW.
   3. `cleanOrphanVolumes()` - tries to clean up `volumesFailedReconstruction`.
      Here kubelet cannot call appropriate volume plugin to unmount a
      volume, because kubelet failed to reconstruct the volume spec from
      `/var/lib/kubelet/pods/<uid>/volumes/xyz`. Kubelet at least tries to
      unmount the directory and clean up any orphan files there.
      This happens only once, `volumesFailedReconstruction` is cleared
      afterwards.

Note that e.g. `mountOrAttachVolumes` can call `volumePlugin.MountDevice` /
`SetUp()` on a reconstructed volume (because it was added to ASW as *uncertain*)
and finally update ASW, while the VolumeManager is still waiting for the API
server to update `devicePath` of the same volume in ASW (step 2. above). We made
sure that `updateReconstructedDevicePaths()` will update the `devicePath` only
for volumes that are still *uncertain*, not to overwrite the *certain* ones.

### Old VolumeManager startup

When kubelet starts, VolumeManager starts DSWP and the reconciler
[in parallel](https://github.com/kubernetes/kubernetes/blob/575616cc72dbfdd070ead81ec29c0d4f00226487/pkg/kubelet/volumemanager/volume_manager.go#L288-L292).

[The reconciler](https://github.com/kubernetes/kubernetes/blob/44b72d034852eb6da8916c82ce722af604b196c5/pkg/kubelet/volumemanager/reconciler/reconciler.go#L33-L45)
then periodically does:
1. `unmountVolumes()` - unmounts (`TearDown`) pod local volumes that are in
   ASW and are not in DSW. Since the ASW is initially empty, this call
   becomes useful later.
2. `mountOrAttachVolumes()` - mounts (and attaches, if necessary) volumes that
   are in DSW, but not in ASW. This will eventually happen for all volumes in
   DSW, because ASW is empty. This actually the way how AWS is populated.
3. `unmountDetachDevices()` - unmounts (`UnmountDevice`) global volume mounts
   of volumes that are in ASW and are not in DSW.
4. Only once after DSW is fully populated:
   1. VolumeManager calls `sync()`, which scans `/var/lib/kubelet/pods/*`
      and reconstructs **only** volumes that are not already in  ASW.
      In addition, volumes that are in DSW are reconstructed, but not added to
      ASW (If a volume is in DSW, we expect that it reaches ASW during step 3.)
      * `devicePath` of reconstructed volumes is populated from
        `node.status.attachedVolumes` right away.
      * In the next reconciliation loop, reconstructed volumes that are not in
        DSW are finally unmounted in step 1. above.
      * There is a workaround to add a reconstructed volume to ASW when it was
        initially in DSW, but all pods that used the volume were deleted before
        the volume was mounted and reached ASW.
        ([#110670](https://github.com/kubernetes/kubernetes/pull/110670))
   2. VolumeManager reports all reconstructed volumes in
      `node.status.volumesInUse` (that's why VolumeManager reconstructs volumes,
      even if it does not add them to DSW).
   3. For volumes that failed reconstruction kubelet cannot call appropriate
      volume plugin to unmount them. Kubelet at least tries to unmount the
      directory and clean up any orphan files there.

#### Observability

Today, any errors during volume reconstruction are exposed only as log messages.
We propose adding these new metrics, both to the old and new VolumeManager code:

* `reconstruct_volume_operations_total` / `reconstruct_volume_operations_errors_total`:
  nr. of all / unsuccessfully reconstructed volumes.
  * In the new VolumeManager code, this will include all volume mounts in
    `/var/lib/kubelet/pods/*/volumes`
  * In the old VolumeManager it will include only volumes that were not already
    in ASW (those are not reconstructed).
* `force_cleaned_failed_volume_operations_total` / `force_cleaned_failed_volume_operation_errors_total`: nr.
  of all / unsuccessful cleanups of volumes that failed reconstruction.
* `orphaned_volumes_cleanup_errors_total`: nr. of reports
  like `orphaned pod "<uid>" found, but XYZ failed`
  ([example](https://github.com/kubernetes/kubernetes/blob/4fac7486d41c033d6bba9dfeda2356e8189035cd/pkg/kubelet/kubelet_volumes.go#L215)).
  These messages can be a symptom of failed reconstruction (e.g.
  [#105536](https://github.com/kubernetes/kubernetes/issues/105536)).
  Note that kubelet logs this periodically and bumping this metric periodically
  would not be useful.
  [`cleanupOrphanedPodDirs`](https://github.com/kubernetes/kubernetes/blob/4fac7486d41c033d6bba9dfeda2356e8189035cd/pkg/kubelet/kubelet_volumes.go#L168)
  needs to be changed to collect errors found during
  one `/var/lib/kubelet/pods/` check and report collected "nr of errors during
  the last housekeeping sweep (every 2 seconds)".
    * TODO: do we want to have a label to distinguish each error reason,
      e.g. "Pod found, but volumes are still mounted on disk" from say
      "orphaned pod %q found, but error occurred during reading of
      volume-subpaths dir from disk"?

### Test Plan

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

All files are in `k8s.io/kubernetes/pkg/kubelet/volumemanager/reconciler/`,
data taken on
[2023-01-26](https://storage.googleapis.com/kubernetes-jenkins/logs/ci-kubernetes-coverage-unit/1613337898885582848/artifacts/combined-coverage.html).

The old reconciler + reconstruction:
- `reconciler.go`: `77.1`
- `reconstruct.go`: `75.7%`

- The new reconciler + reconstruction
- `reconciler_new.go`: `73.3%`
    - The coverage is lower than `reconciler.go`, because parts of
      `reconcile.go` code are tested by unit tests in different packages.
      With force-enabled `SELinuxMountReadWriteOnce` gate in
      today's master(`f21c60341740874703ce12e070eda6cdddfd9f7b`), I got
      `reconciler_new.go` coverage `93.3%`.

- `reconstruct_new.go`: `66.2%`
  - `updateReconstructedDevicePaths` does not have unit tests, this will be
    added before Beta release.

Common code:
- `reconciler_common.go`: `86.2%`
- `reconstruct_common.go`: `75.8%`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

None.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- "Should test that pv used in a pod that is deleted while the kubelet is down
  cleans up when the kubelet returns":
  https://storage.googleapis.com/k8s-triage/index.html?sig=storage&test=Should%20test%20that%20pv%20used%20in%20a%20pod%20that%20is%20deleted%20while%20the%20kubelet%20is%20down%20cleans%20up%20when%20the%20kubelet%20returns
- "Should test that pv used in a pod that is force deleted while the kubelet is
  down cleans up when the kubelet returns":
  https://storage.googleapis.com/k8s-triage/index.html?sig=storage&test=Should%20test%20that%20pv%20used%20in%20a%20pod%20that%20is%20force%20deleted%20while%20the%20kubelet%20is%20down%20cleans%20up%20when%20the%20kubelet%20returns

Both are for the old reconstruction code, we don't have a job that enables
alpha features + runs `[Disruptive]` tests.

Recent results:

> *235 failures (3 in last day) out of 130688 builds from 1/11/2023, 1:00:33 AM
> to 1/25/2023*

I checked couple of the recent flakes and all failed because they could not
create namespace for the test:

https://prow.k8s.io/view/gs/kubernetes-jenkins/logs/ci-cri-containerd-e2e-cos-gce-serial/1620328095124819968:

> Unexpected error while creating namespace: Post
> "https://35.247.99.121/api/v1/namespaces": dial tcp 35.247.99.121:443:
> connect: connection refused

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag

#### Beta

- Gather feedback from developers

#### GA

- Allowing time for feedback.
- No flakes in CI.

#### Deprecation

- Announce deprecation and support policy of the existing flag
- No need to wait for two versions passed since introducing the functionality that deprecates the flag (to address version skew). The feature is local to a single kubelet.
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

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

The feature is enabled by a single feature gate on kubelet and does not require
any special upgrade / downgrade handling.

### Version Skew Strategy

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

The feature affects only how kubelet starts. It has no implications on
other Kubernetes components or other kubelets. Therefore, we don't see any
issues with any version skew.

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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `NewVolumeManagerReconstruction`
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

It changes how kubelet starts and how it cleans volume mounts. It has no
visible effect in any API object.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

The feature can be disabled without any issues.

###### What happens if we reenable the feature if it was previously rolled back?

Nothing interesting happens. This feature changes how kubelet starts and how it
cleans volume mounts. It has no visible effect in any API object nor structure
of data / mount table in the host OS.

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

We have unit tests for the feature disabled or enabled.
It affects only kubelet startup and we don't change format of data present in
the OS (mount table, content of `/var/lib/kubelet/pods/`), so we don't have
automated tests to start kubelet with the feature enabled and then disable it
or a vice versa.

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

If this feature is buggy, kubelet either does not come up at
all (crashes, hangs) or does not unmount volumes that it should unmount.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

`reconstruct_volume_operations_total`,
`reconstruct_volume_operations_errors_total`,
`force_cleaned_failed_volume_operations_total`,
`force_cleaned_failed_volume_operation_errors_total`,
`orphaned_volumes_cleanup_errors_total`

See Observability in the detail design section. All newly introduced metrics
will be added both to "old" and "new" VolumeManager, so users can compare
these metrics with the feature gate enabled and disabled and see if downgrade
actually helped.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Not yet. This will be a manual test.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

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

They can check if the FeatureGate is enabled on a node, e.g. by monitoring
`kubernetes_feature_enabled` metric. Or read kubelet logs.

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
- [X] Other (treat as last resort)
  - Details: logs during kubelet startup.

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

These two metrics are populated during kubelet startup:

* `reconstruct_volume_operations_errors_total` should be zero. An error here
means that kubelet was not able to reconstruct its cache of mounted volumes
and appropriate volume plugin was not called to clean up a volume mount.
There could be a leaked file or directory on the filesystem.

* `force_cleaned_failed_volume_operation_errors_total` should be zero. An error
here means that kubelet was not able to unmount a volume even with all
fallbacks it has. There *is* at least a leaked directory on the filesystem,
there could be also a leaked mount.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [X] Metrics
  - Metric name:
    - `reconstruct_volume_operations_total`
    - `reconstruct_volume_operations_errors_total`
    - `force_cleaned_failed_volume_operations_total`
    - `force_cleaned_failed_volume_operation_errors_total`
    - `orphaned_volumes_cleanup_errors_total`
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

No

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

No.

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

No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Kubelet startup could be slower, but that would be a bug. In theory, the old
and new VolumeManager startup does the same things, just in a different order.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
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

Kubelet won't start *unmounting* volumes that are not needed. But that was the
behavior also before this KEP.

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

Check kubelet logs. There should be errors about a failed volume reconstruction,
together with the directory where the volume was supposed to be mounted.
Ensure that:

1. There is no Pod that uses the volume on the node.
2. The directory of the volume is not mounted there.
3. The directory and all its parents up to `/var/lib/kubelet/pods/<uid>/volumes`
   are removed.
4. If possible, locate global mount of the volume (if it exists) in
   `/var/lib/kubelet/plugins/<volume plugin name>` and unmount + remove it.
   The actual directory varies by volume plugin.
   * For CSI volumes, if the CSI driver supports `NodeStageVolume` CSI call,
     the location is `/var/lib/kubelet/plugins/kubernetes.io/csi/<csi driver name>/<sha256sum of pv.spec.csi.volumeHandle>/globalmount`.
     Otherwise, there is no global mount directory.
   * EmptyDir, Projected, DownwardAPI, Secrets and ConfigMaps do not have global
     mount directory.

## Implementation History

* 1.26: Alpha version was implemented as part of
  [KEP 1710](https://github.com/kubernetes/enhancements/tree/master/keps/sig-storage/1710-selinux-relabeling)
  and behind `SELinuxMountReadWriteOnce` feature gate.

* 1.27: Splitting out as a separate KEP, targeting Beta in this release.

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
