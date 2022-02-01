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
# KEP-3141: Prevent unauthorised volume mode conversion

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Changes to VolumeSnapshotContent API](#changes-to-volumesnapshotcontent-api)
  - [Changes to Snapshot Controller](#changes-to-snapshot-controller)
  - [Changes to external-provisioner](#changes-to-external-provisioner)
  - [Test Plan](#test-plan)
    - [Unit tests](#unit-tests)
    - [E2E tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Alpha -&gt; Beta](#alpha---beta)
    - [Beta -&gt; GA](#beta---ga)
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
  - [VolumeSecurityPolicy](#volumesecuritypolicy)
  - [VolumeSecurityStandard](#volumesecuritystandard)
  - [Annotation on VolumeSnapshotClass](#annotation-on-volumesnapshotclass)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

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

Users can leverage the `VolumeSnapshot` feature, which GA'd in Kubernetes 1.20, to
create a `PersistentVolumeClaim` or `PVC` from a previously taken VolumeSnapshot.
This is done by pointing the `Spec.dataSource` parameter of the `PVC` to an existing
`VolumeSnapshot` instance. There is no logic that validates whether the original
volume mode of the `PVC`, whose snapshot was taken, matches the volume mode of 
the newly created `PVC` that is being created from the existing `VolumeSnapshot`.
This KEP proposes a solution to prevent unauthorized conversion of the volume
mode during such an operation. 

## Motivation

Malicious users may expose a vulnerability in the kernel by exploiting
this gap. 
Here is an example of how a malicious user can exploit this gap to crash the
kernel.
1. User creates a `PVC` with `volumeMode: Block` and runs a pod with it.
2. User writes malformed ext4 data to it (simple dd)
3. User takes snapshot of this volume.
4. User creates a `PVC` with `volumeMode: Filesystem` from the above snapshot.
5. User uses this `PVC` in a pod.
    1. kubelet tries to mount it during pod creation. If there is a CVE in the
       kernel, the user can crash it.

Note that, as of this writing, there is no known CVE in the kernel that a
malicious user can exploit. However CVE's are regularly discovered that affect
filesystems. For example https://access.redhat.com/security/cve/cve-2020-12655
allows an attacker to trigger a DoS attack on the kernel.
This proposal aims to prevent a security vulnerability in the event that a CVE 
is discovered.

We cannot simply block this operation as some backup vendors try to create a 
volume with the exact same mode as the original volume but may need to do the
conversion for efficiency.
An example workflow of a backup vendor could look like:
1. Assume the original `PVC` is created with `volumeMode: Filesystem`.
2. During backup, the backup software will create a `PVC` from a `VolumeSnapshot`
with `volumeMode: Block`. 
This steps needs volume mode conversion. The purpose of creating this `PVC` with 
block mode is to be able to copy data efficiently and save it to a backup target.
3. The `PVC` created in the previous step is temporary and will be deleted after
data is copied.
4. Finally at restore time, another `PVC` will be created with `volumeMode: Filesystem`.


### Goals

Define a mechanism to mitigate the vulnerability of restoring volumes without
hampering valid use cases.

### Non-Goals

Design that is generic and can be extended to other storage related security
aspects.

## Proposal

The proposal aims to mitigate this issue by modifying the `VolumeSnapshotContent`
API spec as well as the control flows of `snapshot-controller` and `external-provisioner`.
`VolumeSnapshotContent` API will include a field that denotes the volume mode of
the volume that the snapshot was created from. 
This proposal also introduces a new annotation on the `VolumeSnapshotContent` resource
that a trusted user (like a backup software) needs to apply on a VolumeSnapshot.

By introducing these changes, we will leverage existing user access rights to determine
whether the volume mode of a volume can be altered when a `PVC` is being created
from a `VolumeSnapshot`.

### User Stories (Optional)

#### Story 1

When a `VolumeSnapshot` is created from an existing `PVC`, a corresponding 
`VolumeSnapshotContent` is created by the `snapshot-controller`.
Alternatively, a `VolumeSnapshotContent` can be manually created by an admin
if the `Spec.Source.SnapshotHandle` refers to a pre-existing snapshot on the
underlying storage system. In either case, `VolumeSnapshots` and `VolumeSnapshotContents`
maintain a 1:1 mapping. 

Backup vendors that need to convert the volume mode when creating a `PVC`
need to identify the `VolumeSnapshotContent` mapped to the `VolumeSnapshot`
from which the `PVC` is being created. 

Either through software or via manual intervention, the annotation 
`snapshot.storage.kubernetes.io/allowVolumeModeChange: true` needs to be applied
to the `VolumeSnapshotContent`. If the backup software is a privileged user, 
it will have `Update` and `Patch` permissions on `VolumeSnapshotContents`.

Then the backup software can continue with the operation by creating a `PVC` 
with `Spec.DataSource` pointing to the `VolumeSnapshot` instance.

#### Story 2

Here is an example of how this change prevents a malicious user from exploiting
this vulnerability.

1. User creates a `PVC` with `volumeMode: Block` and runs a pod with it.
2. User writes malformed ext4 data to it (simple dd)
3. User takes snapshot of this volume.
4. User attempts to create a `PVC` with `volumeMode: Filesystem` from the snapshot.
   1. This is blocked as the user does not have `Update` or `Patch` permissions
   on `VolumeSnapshotContent` resources.


### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

A new out-of-tree flag named `PreventVolumeModeConversion` will be introduced on 
`external-snapshotter` and `external-provisioner`. Both of these components are 
out-of-tree so this proposal will not require any in-tree feature gates. 

### Changes to VolumeSnapshotContent API

With this design, we will introduce two new changes to the VolumeSnapshotContent API:
1. A new optional field, called `SourceVolumeMode` will be added to the `Spec` of 
`VolumeSnapshotContents`. This field will be immutable.

```go
type VolumeSnapshotContentSpec struct {
...
// SourceVolumeMode is the mode of the volume whose snapshot is taken.
// Can be either “Filesystem” or “Block”.
// If left empty, will be treated as “Unknown”.
// +optional
SourceVolumeMode *SourceVolumeMode
...
```

2. A new annotation to `VolumeSnapshotContent` objects. The onus is on the 
backup vendor (via s/w or manually) to add this annotation to the `VolumeSnapshotContent`
if they intend to alter the volume mode. The `VolumeSnapshotContent` must look
like below after this change:

```go
kind: VolumeSnapshotContent 
metadata: 
	annotations: 
		- snapshot.storage.kubernetes.io/allowVolumeModeChange: "true"
...
```

### Changes to Snapshot Controller

There are two cases to consider:
1. Dynamic Provisioning
   1. `VolumeSnapshot` is created by the user, with `VolumeSnapshotClass` optionally
   specified in the spec.
   2. `VolumeSnapshotContent` is created by the `snapshot-controller` in response to (i).
   3. `snapshot-controller` populates the `Spec` of the given `VolumeSnapshotContent`.
      1. With this change, the controller will fetch the `Spec.PersistentVolumeMode`
      of the `PV` and add that to newly introduced `Spec.SourceVolumeMode` field of
      the VolumeSnapshotContent to be created.
2. Static Provisioning
   `VolumeSnapshotContent` is created by the admin. With this change, the admin will be 
expected to fill the `Spec.SourceVolumeMode` field appropriately. If left nil, `Unknown`
mode will be assumed to preserve existing behavior.

### Changes to external-provisioner

This design leverages the access rights of a user on `VolumeSnapshotContents` to
determine whether the volume mode can be modified when a `PVC` is being created
with a `VolumeSnapshot` as the source.
The volume mode can be altered if the requesting user has `Update` and `Patch` rights
on `VolumeSnapshotContents` (which is a cluster scoped resource).
The control flow for creating a `PVC` from a `VolumeSnapshot` will look like below:

1. A user attempts to create a `PVC` from a `VolumeSnapshot` by specifying the
`Spec.DataSource` parameter of the `PVC` YAML.
2. `external-provisioner` receives a callback to dynamically create the volume.
As part of the preprocessing steps, it will:
   1. Get the `Spec.SourceVolumeMode` of the `VolumeSnapshotContent`.
      1. If `Spec.SourceVolumeMode` doesn't exist or is nil, then continue with
      volume provisioning to preserve existing behavior.
   2. Get the `Spec.VolumeMode` of the `PVC` being created.
   If they do not match:
      1. Get all annotations on the `VolumeSnapshotContent` and verify if 
      `snapshot.storage.kubernetes.io/allowVolumeModeChange: true` exists.
      If it does not exist, block volume provisioning by returning an error.
4. In all other cases, let volume provisioning continue.

NOTE: `external-provisioner` maintains a reference to `PVC` and `VolumeSnapshotContent`
during volume creation. This proposal leverages those references to make additional
decisions.

### Test Plan

E2E tests will be added for this design, that attempt to restore a volume with
and without requisite privileges. 

#### Unit tests

- With feature flag disabled:
  - attempt to convert volume mode when creating a `PVC`
  from a `VolumeSnapshot`.
- With feature flag enabled, attempt to convert volume mode when creating a `PVC`
from a `VolumeSnapshot`:
  - With `Spec.SourceVolumeMode` populated and `snapshot.storage.kubernetes.io/allowVolumeModeChange: true`
  annotation present.
  - With `Spec.SourceVolumeMode` populated but no `snapshot.storage.kubernetes.io/allowVolumeModeChange: true`
  annotation.
  - With `Spec.SourceVolumeMode` set to `nil`.

#### E2E tests

The feature flag will be enabled for e2e tests. The tests will attempt to convert volume 
mode when creating a `PVC` from a `VolumeSnapshot`:
  - With `Spec.SourceVolumeMode` populated and `snapshot.storage.kubernetes.io/allowVolumeModeChange: true`
    annotation present.
  - With `Spec.SourceVolumeMode` populated but no `snapshot.storage.kubernetes.io/allowVolumeModeChange: true`
    annotation.
  - With `Spec.SourceVolumeMode` set to `nil`.

### Graduation Criteria

#### Alpha

- Feature implemented behind an out-of-tree feature flag.
- Feedback from users.
- Implementation of unit and e2e tests.

#### Alpha -> Beta

- One release with positive feedback from users. 

#### Beta -> GA

- Deployed in production and in use by backup software. 
- Gone through one kubernetes upgrade.

### Upgrade / Downgrade Strategy

1. Upgrading `external-snapshotter` and `external-provisioner` with `PreventVolumeModeConversion`
enabled:
- `VolumeSnapshots` created after the upgrade will maintain a reference to the
source volume mode. Newly created `PVCs` will undergo an additional check before the 
provisioning is performed on the storage backend. 
- `VolumeSnapshots` created before the upgrade will leave the new API field 
unpopulated.

2. Downgrading `external-snapshotter` and `external-provisioner` with `PreventVolumeModeConversion`
disabled: 
- `VolumeSnapshots` created prior to the upgrade will still maintain a reference 
to the source volume mode, but `PVCs` can be created from them without the
additional check. 

### Version Skew Strategy

This proposal requires changes to three components - `VolumeSnapshotContent` API, 
`external-snapshotter` and `external-provisioner`. 

If any of the components are not upgraded to a version supporting this feature,
then the feature will not work as expected. From an end user perspective, the
existing behavior will continue, ie, there will be no check to prevent 
unauthorized conversion of the volume mode. 

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: Out-of-tree flag named `PreventVolumeModeConversion`, 
  which will be enabled in `external-provisioner` and `external-snapshotter`.
  - Will enabling / disabling the feature require downtime of the control
    plane? `external-provisioner` and `external-snapshotter` will need to be 
  restarted for the changes to take effect. This means that there will be a few
  seconds of downtime until the newer Pods are Running. There will not be any 
  effect on the previously running applications.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
    No

###### Does enabling the feature change any default behavior?

Yes. Users without requisite privileges cannot alter the volume mode of `VolumeSnapshot`
when it is being used to create a `PVC`. Users with privileges need to add an 
annotation to the corresponding `VolumeSnapshotContent` instance if they
require the volume mode to be converted.
The default behavior does not make any validations prior to provisioning a volume.
The volume mode can be converted by any user when a `PVC` is created from a 
`VolumeSnapshot`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature is supported and will fall back to the existing behavior.

###### What happens if we reenable the feature if it was previously rolled back?

The new behaviour will be re enabled. `VolumeSnapshots` created when the feature
was disabled will not have the new capabilities.

###### Are there any tests for feature enablement/disablement?

We will add unit tests with and without the feature flag enabled. The expectation
is for new fields in `VolumeSnapshotContent` to be dropped when the feature flag
is disabled.

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

###### Will enabling / using this feature result in any new API calls?


This feature does not add any new API calls. 

###### Will enabling / using this feature result in introducing new API types?

This feature adds a new field to the existing `VolumeSnapshotContent` API. 

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

The size of `VolumeSnapshotContents` will increase as we will introduce a new 
field to the API. Also, users will be adding an annotation to individual 
objects on a need basis.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

The latency of CSI's `CreateVolume` may increase due to this change, when the 
`Spec.DataSource` field points to a `VolumeSnapshot` instance. This is because
there is an additional check to determine whether volume provisioning must 
continue. However, this increase is expected to be minimal as there are no new
API calls. 

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

### VolumeSecurityPolicy

Proposal to create a new policy called `VolumeSecurityPolicy`, which will be used 
to control access for creation of `PVCs`. 
This proposal also includes an admission controller that prevents `PVCs` from being 
restored with the wrong volume mode, unless the user that attempts to do so is a 
privileged user (as defined by the `VolumeSecurityPolicy`).  

As part of this proposal, there will be only a single field in the `Spec` - 
`allowVolumeModeModification`, which can be set to `true` or `false`.

Once a `VolumeSecurityPolicy` is created, it must be tied to a user or a service 
account, similar to tying a `PSP` to a user/service account.

An admission controller will be introduced that intercepts requests to create a
`PVC`. In case the `PVC` is being restored from a snapshot and is modifying the 
volumeMode, it validates that the user requesting the `PVC` has the allowed 
privileges. If not, the admission controller rejects the `PVC` create request.

Rejected as PSP was recently deprecated in lieu of PodSecurityStandards. If we 
need a standard for storage security, we should follow that approach.

### VolumeSecurityStandard

Introduce `VolumeSecurityStandards` that enforceable by any mechanism, including
webhooks, similar to `PodSecurityStandards`.

We will define two policies as part of this design:
1. `Privileged` - least restrictive policy that allows the widest level of permissions. 
2. `Restricted` - most restrictive policy that follows security best practices. 
 
A `Mode` defines how a violation of the given security policy is handled. 
There are three modes:
1. `Enforce`: violations of the policy are not allowed.
2. `Audit`: violations trigger an audit annotation, but are otherwise allowed.
3. `Warn`: violations trigger a user-facing warning, but are otherwise allowed.
 
A `VolumeSecurityStandard` is applied on a per-namespace basis. This gives an 
admin the ability to apply different standards based on the users of a namespace.

An admission controller will be introduced that intercepts requests to create a 
PVC. The  VolumeSecurityStandards will be hardcoded into this admission controller.

Rejected as the solution was too generic for a very specific use case. If and when 
there are more storage related security aspects that need a generic solution, we
can reconsider this approach. 

### Annotation on VolumeSnapshotClass

This proposal introduced a new annotation on the `VolumeSnapshotClass` object
`allowModeConversionForUsers: <comma separated list of allowed users>`.

The above comma separated list of users are set by the admin. They will be allowed 
to modify the volume mode when restoring a PVC from a Snapshot. 
The annotation `allowModeConversionForUsers` will be copied to the `VolumeSnapshotContent` 
by the `snapshot-controller` from the `VolumeSnapshotClass`.
`VolumeSnapshotClass` is cluster-scoped therefore applying this annotation is 
restricted to privileged users only.

An admission controller will be introduced that intercepts requests to create a PVC. 

Rejected due to issues with immutability of this lists. For example, if a users 
access is revoked, does the admin need to modify all existing resources that 
allow this user to modify volume mode? Also there were concerns with introducing
a new mechanism for access control.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
