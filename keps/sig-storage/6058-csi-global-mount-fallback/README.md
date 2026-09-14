# KEP-6058: CSI global mount reconstruction

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: node drain and reboot while unstage is in progress](#story-1-node-drain-and-reboot-while-unstage-is-in-progress)
    - [Story 2: the pod-local vol_data.json is lost or corrupt](#story-2-the-pod-local-vol_datajson-is-lost-or-corrupt)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#summary-of-changes) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation, e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

After a kubelet restart or a node reboot, kubelet can report a volume as unused
while it is still mounted on the node. The attach/detach controller believes it,
attaches the volume elsewhere, and two nodes write to the same device.

This happens because reconstruction learns about mounts only by walking
`/var/lib/kubelet/pods`. A global mount whose pod directory is gone, or whose
pod-local state file is unreadable, is invisible to it. A node drain reaches the
first case with no corruption and no operator mistake.

This KEP makes reconstruction find mounted volumes directly and finish their
unmount, so that a volume kubelet is still holding is unstaged instead of being
left for an operator to find. Behind an alpha feature gate,
`CSIGlobalMountReconstruction`, default off.

## Motivation

Kubelet does not wait for the device-level unstage before it lets a pod go. A
pod is deleted once `TearDown` / `NodeUnpublishVolume` has succeeded for its
volumes, while `UnmountDevice` / `NodeUnstageVolume` is still in flight. A node
drain waits for pods, so the drain reports success and the node reboots with
the unstage unfinished. What is left on disk is a global mount directory and
its `vol_data.json`, with no pod directory anywhere that names the volume.

That is the shape of [#121937][], and it is the case reconstruction cannot see.
Reconstruction is driven by `/var/lib/kubelet/pods`: for each pod-local mount
it finds, it asks the volume plugin to rebuild a `volume.Spec` and records the
volume in the ActualStateOfWorld. A global mount with no pod directory
generates no candidate, so `ConstructVolumeSpec` is never called for it, the
volume never enters the ActualStateOfWorld, and it never appears in
`node.status.volumesInUse`. Nothing on the node unstages the mount, and nothing
tells the driver the volume is gone, while the attach/detach controller is free
to attach it to another node. No recovery keyed off the pod directory can reach
this, because the pod directory is exactly what is missing. Recovery has to
start from the global mount itself.

Issue [#101791][] has tracked the same symptom, a live global mount with no
in-memory record, since 2021, and reaches it by a second route. The pod-local
`vol_data.json` is the only source of truth reconstruction uses, so any
condition that loses or corrupts it (disk full at write time, a partial write
during shutdown, an operator deleting the directory while chasing a different
problem, filesystem corruption) produces the same orphaned mount with the pod
directory still in place. KEP-3756 made reconstruction robust against
most kubelet bugs and documented this case as one for manual cleanup.

Both routes are recoverable from data already on disk. The global mount's own
`vol_data.json`, written by `csiAttacher.MountDevice`, already stores
`volumeHandle` and `driverName`; two more fields make it self-sufficient. With
that file trusted as a source, reconstruction can start from the global mount
whether or not a pod directory exists, and a manual recovery path documented in
KEP-3756 becomes an automatic one, with no change to any contract with CSI
drivers.

[#101791]: https://github.com/kubernetes/kubernetes/issues/101791
[#121937]: https://github.com/kubernetes/kubernetes/issues/121937

### Goals

- Recover a CSI global mount whose pod directory no longer exists, by having
  reconstruction ask each volume plugin for the global mounts it holds.
- Recover the same mount when the pod directory is still there but its
  `vol_data.json` cannot be read, for reconstruction and for unmounting alike.
- Report a recovered volume in `node.status.volumesInUse` while the node still
  holds it, on the same terms as a volume reconstructed from a pod directory.
- Keep the change additive: no behavior change on the path where reconstruction
  already succeeds, and leave the door open for other plugins with global
  mounts to reuse the same machinery.
- Gate the new behavior behind an alpha feature gate, default off.

### Non-Goals

- In-tree FibreChannel and iSCSI. They stay in-tree until someone writes a
  compatible replacement, and no implementation for them is proposed here. The
  architecture is meant to accommodate them: reconstruction asks plugins for
  their global mounts through an interface, so adding FC or iSCSI is
  implementing one call, not changing the volume manager. NFS does not use
  `MountDevice` and is unaffected.
- Raw block volumes. Their volume data is already node-global, so the pod-local
  failure mode cannot happen for them, and the staging path they can still leak
  waits for beta: `GlobalVolume` carries one path, while a block volume's
  staging, publish and device paths sit under different roots.
- Changing the CSI specification or any contract with drivers. Nothing here
  requires a driver change.
- Recovering a volume whose global `vol_data.json` is also unreadable. With
  both copies gone there is nothing on disk that still names the driver and the
  volume handle, and operator intervention remains necessary.

## Proposal

`csiAttacher.MountDevice` writes `/var/lib/kubelet/plugins/kubernetes.io/csi/<driver>/<sha256(volumeHandle)>/vol_data.json`
with `volumeHandle` and `driverName`. We extend it to also write `specVolID`
(the PV name, equal to the basename of the pod-local mount path) and
`volumeLifecycleMode` (hardcoded to `Persistent` since `MountDevice` only
runs for device-mountable volumes). With those fields the global file names the volume it belongs to, which is
what lets reconstruction start from it with no pod directory in hand. Writing
them changes no behavior on its own.

Reconstruction gains a second, independent source of candidates: each volume
plugin that keeps global mounts is asked to list them. The CSI plugin walks its own directory and reports what it
finds; the volume manager registers every mount it does not already track in
the ActualStateOfWorld as *uncertain*, following the semantics KEP-3756
introduced, and resolves it from there. A volume a pod still needs is
re-verified; a volume no pod needs is unstaged through `NodeUnstageVolume` and
its directory removed only after that succeeds. Registering the mount before
touching it is what allows it to be reported in `node.status.volumesInUse`
while the cleanup runs, so no competing attach starts in the middle.

The layout of a global mount is the plugin's business, so the volume manager
asks rather than globbing a path of its own. That is also what lets
FibreChannel and iSCSI reuse this machinery later by implementing one call.

The scan starts from the plugin directory, so it cannot help the case where the
pod directory is still present and it is the pod-local `vol_data.json` that is
unreadable. For that, `ConstructVolumeSpec` and `NewUnmounter` fall back to the
global file, finding it through the bind mount the pod-local mount already
holds. Where there is no such mount reference, reconstruction fails as today.

### User Stories

#### Story 1: node drain and reboot while unstage is in progress

The drain completes because every pod finished `NodeUnpublishVolume`, the node
reboots before `NodeUnstageVolume` does, and the volume is left staged with no
pod directory naming it ([#121937][]). Reconstruction now finds it through the
plugin, no pod claims it, and it is unstaged and released.

#### Story 2: the pod-local vol_data.json is lost or corrupt

Kubelet is OOM-killed mid-write, or an operator clears a stuck pod's volume
directory. Reconstruction and `cleanupMounts` both fail on the same unreadable
file; both now read the global mount's own copy instead, and the volume takes
the normal unmount path.

### Notes/Constraints/Caveats

- The pod-local fallback needs the bind mount, so it covers a kubelet restart
  and not a node reboot. A reboot takes the mount table with it, and the
  pod-local bind mount never comes back, since nothing outside kubelet
  recreates it. After a reboot no key rooted in the pod directory works, its
  name included, and the volume is recovered by the scan instead.
- Recovery can therefore reach `NodeUnstageVolume` for a volume this kubelet
  never sent a `NodeUnpublishVolume` for. In the drain case the unpublish
  succeeded before the reboot, so the CSI ordering requirement was met in the
  node's previous life; where it did not, nothing is published at that
  `target_path` any more, and `NodeUnstageVolume` is required to be idempotent.
- Listing global mounts happens once per reconstruction pass, that is, once
  per kubelet startup, and each plugin reads only the directories it staged.

### Risks and Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Wrong global mount is matched | Low | The candidate comes from the mount table, so it shares the pod-local mount's device. A driver staging several volumes from one export can offer more than one, so the recovered `specVolID` must name the volume being reconstructed; a mismatch fails rather than guesses. A pre-feature file carries no `specVolID` and is accepted on the mount reference alone |
| Global `vol_data.json` is also corrupt | Medium: one root cause can hit both files | Reconstruction fails with both errors wrapped together, matching today's behavior |
| Feature gate disabled mid-cluster | Low | The new fields are written unconditionally, so a node with the gate off still produces files an enabled node can use |
| Stale global mount data after detach | Low | Detach removes the directory and the file; if it failed previously, that is what this KEP recovers from |
| An orphan is unstaged while a pod still needs it | Low | The scan registers it as uncertain rather than unmounting it; the reconciler re-verifies anything still in the desired state |

## Design Details

The change adds one plugin interface, its CSI implementation, the call to it
from reconstruction, and a fallback in two CSI call sites:

1. `csi_attacher.go`: `MountDevice` adds `specVolID` (from `spec.Name()`) and
   `volumeLifecycleMode` (constant `string(storage.VolumeLifecyclePersistent)`)
   to the data map written to the global `vol_data.json`. Field additions are
   unconditional: written regardless of the feature gate, so a downgrade does
   not produce stale or partial files.

2. `pkg/volume/plugins.go`: a new optional plugin interface, so that the volume
   manager never learns CSI's directory layout.

   ```go
   // Implemented by plugins that keep a global mount outside any pod directory.
   type GlobalVolumeListerPlugin interface {
   	DeviceMountableVolumePlugin
   	ListGlobalVolumes() ([]GlobalVolume, error)
   }

   type GlobalVolume struct {
   	ReconstructedVolume // Spec, SELinuxMountContext
   	DeviceMountPath     string
   	VolumeMode          v1.PersistentVolumeMode
   }
   ```

   `FindGlobalVolumeListerPlugins` returns every plugin implementing it. Unlike
   the other `Find` helpers it takes neither a spec nor a name, because
   reconstruction has neither when it starts: it asks each plugin what it holds. The interface only reads: what
   is on disk is removed by `UnmountDevice`, which already calls `removeMountDir`
   after a successful `NodeUnstageVolume`, so a recovered volume is cleaned up
   by the path every other volume takes.

3. `csi_global_volumes.go`: the CSI implementation. It walks its own plugin
   directory,
   loads each `vol_data.json`, and builds a spec with `constructPVSourceSpec`.
   It excludes its own `volumeDevices` subtree, the raw block subtree that is a
   sibling of the per-driver directories, by asking the host for that path
   rather than matching a literal name. Inline ephemeral volumes never stage a
   global mount, so they never appear. A volume whose `vol_data.json` will not
   load is skipped rather than reported, so one unreadable directory does not
   hide the volumes around it.

   FibreChannel and iSCSI can implement the same call when someone wants
   reconstruction for them: both already rebuild their spec from the global
   directory name alone (`parsePDName`, `extractPortalAndIqn`), and each knows
   its own exclusions, of which `volumeDevices` is one.

   A directory is only reported if it is the one `MountDevice` would have
   staged the volume its `vol_data.json` names, that is, `sha256(volumeHandle)`.
   Without that check a directory holding another volume's data would be
   unstaged at a path that is not itself, since the unmount recomputes the path
   from the spec, and would report success while leaving this mount in place.

4. `pkg/kubelet/volumemanager` (reconstruction): after the existing walk of
   `/var/lib/kubelet/pods`, reconstruction calls each such plugin and, for every
   entry not already tracked, derives the name with
   `GetUniqueVolumeNameFromSpec` and registers it through
   `AddAttachUncertainReconstructedVolume` and `MarkDeviceAsUncertain`. The
   reconciler resolves it from there: a volume still in the desired state is
   re-verified, one no pod wants is unstaged and only then removed. The name is
   derived here rather than returned by the plugin, so that it matches what the
   desired state produces.

   Two kinds of entry are declined before anything is registered. A volume the
   plugin cannot device mount is skipped, because the desired state names that
   kind by pod and there is no pod here to name it with, so unstaging it would
   use a name the desired state never produces. A raw block volume is skipped
   too: it reaches the ActualStateOfWorld through the block mapper rather than
   through a device mount, which is why `GlobalVolume` carries a volume mode.
   If marking the device fails after the volume was added, it is removed again
   rather than left behind, since a volume with no pod and no mounted device
   reads to the reconciler as one to detach.

   Attachability is where a recovered volume differs from one that was never
   lost. `AddAttachUncertainReconstructedVolume` records it as
   attach-uncertain, and `GetVolumesInUse` reports only volumes whose
   attachability resolved to true, so the volume is queued in
   `volumesNeedUpdateFromNodeStatus` and settled on the next read of
   `node.status.volumesAttached`: found there, it is reported in
   `volumesInUse`; absent, the controller has already detached it and it is
   not. That is how reconstruction from a pod directory behaves today. The
   unstage does not depend on it either way, because `DeviceMayBeMounted` is
   true for an uncertain device, and the same queue holds the reconciler back
   from unmounting anything until that read succeeds.

   Two constraints follow. `GenerateUnmountDeviceFunc` recomputes the device
   mount path from the spec, and `GetVolumeName` derives the unique volume name
   from it, so the returned spec carries the real `volumeHandle`;
   `DeviceMountPath` is advisory. And the spec is synthetic: `addVolume`
   requires one and offers no path keyed by plugin name, so CSI returns a
   `PersistentVolume` built from the fields in `vol_data.json`, which is what
   `ConstructVolumeSpec` already returns today. A volume staged before this
   feature carries no `specVolID`, which names it only for a reader of the
   logs; the unique name comes from the driver and handle, so those volumes are
   recovered too.

   Prior art: [#136771](https://github.com/kubernetes/kubernetes/pull/136771),
   opened by @shivamwayal37 in February 2026 against [#121937][], reaches the
   same ActualStateOfWorld state from the same directories.

5. `csi_util.go`: new helper `findGlobalMountDataFromPodMount(host, mountPath)`
   that asks the mount table which global mount the pod-local mount belongs to.
   `NodePublishVolume` publishes the staged volume into the pod directory, which
   drivers implement as a bind mount from the staging path, so
   `GetReliableMountRefs` on `<mountPath>/mount` returns it, and the data
   directory is its parent. A driver that publishes some other way leaves no
   reference, and the fallback declines rather than guessing. A reference is accepted when it is named
   `globalmount` and the `vol_data.json` beside it names both a driver and a
   volume handle. References that fail either check are logged at V(4) and
   skipped; "no global mount is bind mounted here" is propagated up.

   Matching on the directory name instead of the mount table is not sound where
   both are available. `Spec.Name()` is the
   PV name for a persistent volume but the pod-spec entry for an inline
   ephemeral one, and those namespaces overlap, so a name like `data` denotes
   both; since an inline volume never stages a global mount, every name match
   it could produce belongs to somebody else's volume. `volumeLifecycleMode`
   cannot break that tie, because `MountDevice` writes it as the constant
   `Persistent` and the real mode lives only in the file this fallback exists
   because it cannot read.

6. `csi_plugin.go`: `ConstructVolumeSpec` calls `loadVolumeData` as today. On
   error, and with the gate on, it retries through the helper and continues
   with the recovered map; if both loads fail it returns the original error
   with the fallback error wrapped alongside. The success path is unchanged.
   A global file staged by an older kubelet carries no `specVolID`, and the
   `volumeName` argument is used instead, which is the pod directory name that
   `SetUpAt` would have stored.

7. `csi_plugin.go`: `NewUnmounter` gets the same fallback through the same
   helper, and it is load-bearing. `GenerateUnmountVolumeFunc` builds the unmounter before running `TearDown`,
   and `NewUnmounter` reads the same unreadable pod-local file, so without this
   the unmount operation is never generated: the pod stays in the
   ActualStateOfWorld, `UnmountDevice` is never reached, and the mount stays
   orphaned. Rescuing the spec without rescuing the unmount closes nothing.
   Writing the recovered data back to the pod-local file instead was rejected;
   see Alternatives.

Feature gate registration is in `pkg/features/kube_features.go` with
`Default: false, PreRelease: featuregate.Alpha`.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

Coverage for the changed packages:

- `k8s.io/kubernetes/pkg/volume`: 77%
- `k8s.io/kubernetes/pkg/volume/csi`: 78%
- `k8s.io/kubernetes/pkg/kubelet/volumemanager/reconciler`: 74%

Unit tests in `pkg/volume/csi`, already in the implementation PR
[#138454](https://github.com/kubernetes/kubernetes/pull/138454), cover both
sides of the gate and each way the lookup can go wrong: a missing pod-local
file with the bind mount still in place is rebuilt from the global one and
still fails with the gate off; an inline ephemeral volume sharing a name with
an unrelated staged PersistentVolume is not paired with it; a pod mount with no
mount references, and a reference whose `vol_data.json` is unreadable or names
no driver, are reported rather than guessed at; a global file with no
`specVolID` is named from the pod directory, and one naming a different volume
is refused; and `NewUnmounter` recovers the driver and handle on the same
terms.

Both fallbacks were exercised end to end on a kind cluster running a kubelet
built from this branch, confirming the rescued volume reaches `UnmountDevice`
and that `NodeUnpublishVolume` and `NodeUnstageVolume` are called. That run
predates the listing interface and does not cover it.

Unit tests cover both sides of the interface. In `pkg/volume/csi`,
`ListGlobalVolumes` returns one entry per global mount with the real volume
handle in the spec, skips `volumeDevices`, skips a directory whose volume data
will not load or names another volume, and still recovers one staged before this
feature existed. In `pkg/volume`, only a plugin implementing the interface is
returned. In `pkg/kubelet/volumemanager`, a listed volume is registered with an
uncertain device and the reported mount path, one already tracked is left alone,
a raw block volume is left to the block path, a plugin that cannot list is not
fatal, and the gate is exercised in both positions.

##### Integration tests

None planned for alpha. The relevant code paths are reached only at kubelet
startup, which integration tests do not exercise meaningfully without a real
node.

##### e2e tests

For beta: node e2e tests that cover both recovery paths.

Pod-local fallback:

1. Mount a CSI volume via a pod using a mock CSI driver.
2. Kill kubelet, delete the pod-local `vol_data.json`, restart kubelet.
3. Assert the volume is reconstructed (no `volumesFailedReconstruction`
   entry, normal unmount on pod deletion).

Orphaned global mount:

1. Mount a CSI volume via a pod using a mock CSI driver that delays
   `NodeUnstageVolume`.
2. Delete the pod and stop kubelet after `NodeUnpublishVolume` succeeds but
   before the unstage completes, then restart kubelet.
3. Assert the scan registers the global mount, `NodeUnstageVolume` is
   called, the directory is removed, and the volume leaves
   `node.status.volumesInUse` only after the unstage.

Tests will live in `test/e2e_node/csi_volume_reconstruction_test.go`.

### Graduation Criteria

#### Alpha

- `GlobalVolumeListerPlugin` in `pkg/volume`, with reconstruction calling it
  for every plugin that implements it.
- The CSI implementation of that interface, listing global mounts and excluding
  its own `volumeDevices` subtree. Nothing is removed by the listing: a
  recovered volume is cleaned up by `UnmountDevice` like any other.
- The global mount's `vol_data.json` carries `specVolID`, so a staged volume can
  be identified with no pod directory in hand. Written unconditionally, so a
  node with the gate off still produces files an enabled node can use.
- `ConstructVolumeSpec` and `NewUnmounter` fall back to that file when the
  pod-local one cannot be loaded, refusing any global mount whose `specVolID`
  names a different volume.
- All of it behind `CSIGlobalMountReconstruction`, default off, with unit tests
  in `pkg/volume/csi` and `pkg/kubelet/volumemanager` for both gate states.
- KEP merged.

Raw block volumes are excluded from alpha: `GlobalVolume` carries one path and
no volume mode, and describing a block volume's staging, publish and device
paths needs that field.

#### Beta

- Raw block volumes, which need a volume mode on `GlobalVolume` before their
  staging path can be described, with unit tests in
  `pkg/kubelet/volumemanager`.
- Node e2e test in CI for at least one release, covering both recovery paths.
- Metrics: a label on `reconstruct_volume_operations_total` distinguishing
  `pod-local` from `global-mount`, so operators can see recovery frequency.
- One release cycle at alpha with no open bugs against either recovery path.
- Default gate flipped to on.

#### GA

- Two releases at beta with the gate default-on, no regressions.
- Conformance test if SIG Architecture deems applicable.
- Production usage documented (CSI driver vendors confirm no surprises).

### Upgrade / Downgrade Strategy

- Upgrade with gate disabled: no behavior change. New global
  `vol_data.json` files include extra fields; older code ignores unknown
  fields, so a future downgrade is safe.
- Upgrade with gate enabled: reconstruction uses the fallback when needed.
  No interaction with control plane.
- Downgrade: kubelet stops reading the extra fields. Global files written
  during the upgraded period contain extra keys that are ignored. No data
  migration needed.

### Version Skew Strategy

This is a kubelet-only feature gate. No skew between control plane and
kubelet is possible. Skew between two kubelets is not possible (volumes are
node-local).

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `CSIGlobalMountReconstruction`
  - Components depending on the feature gate: `kubelet`

###### Does enabling the feature change any default behavior?

No. With both `vol_data.json` files intact, the success path is unchanged.
The feature only alters failure paths that today produce orphaned global
mounts: a failed pod-local load now falls back to the global file, and a
global mount left behind by an interrupted unstage is now unstaged cleanly
instead of leaking.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate restores the previous behavior on the next kubelet
restart, and there is no on-disk state migration. `MountDevice` writes the two
extra fields unconditionally, so a kubelet with the gate off keeps producing
files that carry them; the disabled code path simply does not read them. They
are not leftover state either: kubelet deletes that file when it cleans up the
global mount, since `UnmountDevice` calls `removeMountDir` after a successful
`NodeUnstageVolume`, which removes the `globalmount` directory, the
`vol_data.json` beside it, and the volume directory itself. The fields live
exactly as long as the mount they describe.

###### What happens if we reenable the feature if it was previously rolled back?

Same as initial enablement: the next failed pod-local load triggers the
fallback. No state recovery needed.

###### Are there any tests for feature enablement/disablement?

The added unit tests exercise both gate-on and gate-off paths.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A volume with an intact pod-local file reaches neither path, so a rollout
touches only volumes that fail today. The failure to watch for is a
reconstruction that succeeds with the wrong data: the candidate comes from the
mount table rather than from a name that could be shared, and its `specVolID`
must name the volume being reconstructed. For the listing, the failure to watch
for is unstaging a volume a pod still needs, which the uncertain registration
prevents: the reconciler re-verifies anything still in the desired state.

###### What specific metrics should inform a rollback?

`reconstruct_volume_operations_errors_total` should not increase post-rollout
versus pre-rollout. If it does, disable the gate.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be exercised in the e2e test added at beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

At alpha, three V(2) log lines: `plugin.ConstructVolumeSpec recovered vol_data
from global mount`, `unmounter recovered vol_data from global mount`, and
`Global mount with no pod directory is marked uncertain and added into the
actual state`.

The `reconstruct_volume_operations_total` metric already exists in kubelet at
ALPHA stability, today as an unlabelled counter incremented once per pod
volume directory reconstruction attempt, with
`reconstruct_volume_operations_errors_total` counting the failures among them.
As part of this KEP we plan to add a label to it distinguishing `pod-local`
from `global-mount`, which turns the counter into a labelled one, a `NewCounter` to `NewCounterVec` change across its callers; its alpha
stability allows that without a deprecation cycle.

###### How can someone using this feature know that it is working for their instance?

Look for the V(2) log line above; or, after beta, inspect the metric label.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Listing runs once per kubelet start, bounded by the volumes each plugin has
staged on the node, and reads one small file per volume. On the happy
path the pod-local fallback costs one parse of `/proc/self/mountinfo` and one
file read, tens of milliseconds. `GetReliableMountRefs` retries while the mount table reads
inconsistently, so a pathological node can spend longer per attempt; kubelet
startup does not block on it, the reconciler retries.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- Count of `reconstruct_volume_operations_errors_total` (should not rise
  after enabling).
- Frequency of the V(2) fallback log line (should be near zero in healthy
  clusters; non-zero indicates a real corruption issue worth investigating
  separately).

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The labelled counter described above distinguishes the two recovery paths at
beta, so an operator can see which one is firing.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. Only drivers that advertise the optional `STAGE_UNSTAGE_VOLUME` capability
create a global mount, so only those are affected at all.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new call types. The orphaned global mount scan registers what it finds in
the ActualStateOfWorld, so those volumes appear in the node status update
kubelet already sends: `node.status.volumesInUse` carries one extra entry per
recovered mount until `NodeUnstageVolume` completes. No additional request is
issued, the existing periodic update carries a slightly longer list.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, on the `Node` object, and that growth is the fix rather than a cost of it.
`node.status.volumesInUse` is the attach/detach controller's only interlock
against detaching a device that is still staged: `processVolumesInUse` copies
the list in as `MountedByNode`, and the detach reconciler skips anything
carrying that flag unless a force detach or the `node.kubernetes.io/out-of-service`
taint overrides it. A global mount that outlived a kubelet restart is still
staged, so it belongs in that list by the field's own definition. No new API
objects, and one extra `UniqueVolumeName` entry per recovered mount the node
still has attached, until `NodeUnstageVolume` completes. On a healthy node that
count is zero.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Negligible for reconstruction, which runs once at startup. For a volume stuck
without a pod-local `vol_data.json`, unmount generation adds one mount table
parse per reconciler pass until the volume is released.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Two extra string fields per global `vol_data.json` (a few dozen bytes).
The fallback parses `/proc/self/mountinfo` and reads one file already on
disk.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Reconstruction itself reads local disk only and completes without the API
server, including both recovery paths. Acting on what the scan found does need
it: the reconciler decides whether a recovered mount is still wanted by
comparing against the desired state of world, which kubelet populates from the
API server. With the API server unavailable that comparison never runs, the
recovered mounts stay *uncertain*, and nothing is unstaged. That is the safe
direction to fail, because the feature never removes a mount it cannot prove is
unwanted.

###### What are other known failure modes?

- Both `vol_data.json` files corrupt: reconstruction fails with a wrapped
  error message naming both files. Operator intervention required, same as
  today.
- Mount table unreadable, or the pod-local mount has no reference to any
  global mount: the fallback returns an error which is wrapped alongside the
  original one, and reconstruction fails as it does today.
- A staged directory whose `vol_data.json` will not load: the plugin logs it at
  V(4) and leaves it out of the listing, so one unreadable directory does not
  hide the volumes around it. That mount stays orphaned, as it is today, and
  still needs an operator.

###### What steps should be taken if SLOs are not being met to determine the problem?

If `reconstruct_volume_operations_errors_total` rises after enabling, capture
kubelet logs and inspect for the wrapped fallback error message. If the
issue is the fallback itself (not the underlying corruption), disable the
gate and report the bug.

## Implementation History

- 2026-04-18: Implementation PR opened ([#138454](https://github.com/kubernetes/kubernetes/pull/138454)).
- 2026-05-04: KEP drafted.
- 2026-07-07: Extended per SIG Storage review to scan the plugin directory for
  global mounts with no pod directory, rather than working from pod
  directories alone.
- 2026-09-06: Retargeted to alpha in v1.38 and moved to `implementable`.
- 2026-09-07: Restructured per SIG Storage review so that recovery of a mount
  with no pod directory leads the document; gate renamed to
  `CSIGlobalMountReconstruction`. Matching moved from directory names to the
  mount table, and `NewUnmounter` given the same fallback.
- 2026-09-08: Moved recovery behind `GlobalVolumeListerPlugin` per SIG Storage
  review, so that the volume manager asks each plugin rather than reading the
  CSI layout itself, and brought it into alpha.

## Drawbacks

The scan is the larger of the two changes: it registers volumes in the
ActualStateOfWorld that no pod asked about, and unstages them on that basis. It
also adds an interface to `pkg/volume`, a surface shared with sig-node that the
project then has to keep. Against that, the case it recovers is today left to
an operator who has to notice it first.

## Alternatives

1. Make the pod-local write atomic and durable enough that it can never be
   partial or missing. Useful but does not address operator-deletion or
   filesystem-level corruption, and atomic writes alone do not protect
   against disk-full at write time.

2. Detect the orphaned global mount during cleanup and unmount it defensively.
   Considered in early discussion of [#101791][]. By the time `cleanupMounts`
   runs the volume is out of the ActualStateOfWorld entirely, so nothing keeps
   it from being reported as released mid-cleanup. Registering it first, which
   is what this KEP does, is what puts it back under the normal unmount path.

3. Reconstruct purely from `/proc/mounts`. Possible for some plugins but
   loses the spec information CSI needs (driver-specific options, lifecycle
   mode, etc.). Falling back to a structured file we already write keeps the
   spec intact.

4. Repair the pod-local `vol_data.json` from the recovered data, so the
   existing `NewUnmounter` finds it. Rejected: that file has one writer today
   and reconstruction is a reader, so a second writer into a directory that may
   be mid-teardown is the larger change. It also makes a transient failure
   permanent, since once the file parses the fallback never runs for that
   volume again, and `saveVolumeData` truncates in place with no atomic rename,
   so a partial repair destroys the evidence an operator would need.
