# KEP-6058: CSI global mount fallback for volume reconstruction

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
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
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#summary-of-changes) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation, e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

When kubelet restarts, it reconstructs in-memory volume state from disk. For each
pod-local mount it loads `vol_data.json`, which carries the data needed to
rebuild a `volume.Spec`. If that file is missing or corrupt, reconstruction
fails: the volume is added to `volumesFailedReconstruction`, `cleanupMounts`
unmounts only the pod-local bind mount, and the global mount under
`/var/lib/kubelet/plugins/kubernetes.io/csi/<driver>/<hash>/globalmount`
remains live. The volume is removed from `node.status.volumesInUse`, the
attach/detach controller may then attach it to a different node, and a
double-mount results. On RWO filesystems (FibreChannel, iSCSI, EBS, etc.) this
corrupts data.

KEP-3756 ("Robust VolumeManager reconstruction after kubelet restart")
documented this case and prescribed manual operator intervention to clean up
the leaked global mount. This KEP closes the loop for CSI volumes by making
reconstruction recover automatically: `MountDevice` already writes a
`vol_data.json` next to the global mount; we extend that file with the two
fields needed to rebuild a `volume.Spec` (`specVolID` and
`volumeLifecycleMode`), and `ConstructVolumeSpec` falls back to it when the
pod-local copy cannot be loaded. The fallback is gated behind a new alpha
feature gate, `VolumeReconstructionFallback`.

## Motivation

Issue [#101791][] has tracked the orphaned-global-mount class of bugs since
2021. The root cause is that pod-local `vol_data.json` is the only source of
truth used for reconstruction; any condition that loses or corrupts it
(disk full at write time, partial write during shutdown, manual deletion by an
operator chasing a different problem, filesystem corruption) leaves the global
mount live with no in-memory record. The attach/detach controller then sees a
`volumesInUse` set that does not reflect reality and proceeds to attach
elsewhere.

KEP-3756 made reconstruction itself robust against most kubelet bugs but kept
the underlying vol_data.json contract: if the pod-local file is unreadable,
reconstruction must fail and an operator must clean up by hand. That hand
cleanup is documented in the troubleshooting section of KEP-3756. In practice,
operators rarely catch the failure window before the controller detaches and
re-attaches.

The information needed to rebuild the spec is already available on disk: the
global mount's `vol_data.json`, written by `csiAttacher.MountDevice`,
already stores `volumeHandle` and `driverName`. With two more fields it is
self-sufficient. Using it as a fallback turns a manual-recovery path into an
automatic one without changing any contract with CSI drivers.

[#101791]: https://github.com/kubernetes/kubernetes/issues/101791

### Goals

- Eliminate the orphaned-global-mount failure mode for CSI volumes when the
  pod-local `vol_data.json` is missing or corrupt at kubelet restart.
- Keep the change additive: no behavior change when both files are intact.
- Gate the new behavior behind an alpha feature gate, default off, so it can
  be disabled at any release.
- Cover the recovery path with a unit test (added in the implementation PR)
  and a node e2e test (planned for beta).

### Non-Goals

- In-tree volume plugins (FibreChannel, iSCSI, NFS, etc.) that derive state
  from `/proc/mounts` rather than a state file. Those plugins are being
  migrated to CSI; addressing them in-tree is out of scope.
- CSI drivers that do not implement `NodeStageVolume`. Without a global mount
  there is nothing to fall back to. Pods using such drivers continue to use
  the existing pod-local-only reconstruction path.
- Recovery from a *corrupt* global `vol_data.json`. If both files are
  unreadable, reconstruction still fails with a clear error and operator
  intervention is still required.
- Detection of the original corruption cause. This KEP recovers from the
  symptom; the cause (disk full, abrupt shutdown, etc.) is out of scope.

## Proposal

`csiAttacher.MountDevice` writes `/var/lib/kubelet/plugins/kubernetes.io/csi/<driver>/<sha256(volumeHandle)>/vol_data.json`
with `volumeHandle` and `driverName`. We extend it to also write `specVolID`
(the PV name, equal to the basename of the pod-local mount path) and
`volumeLifecycleMode` (hardcoded to `Persistent` since `MountDevice` only
runs for device-mountable volumes). With those four fields the global file
contains everything `ConstructVolumeSpec` needs to rebuild a
`volume.Spec`; no behavior change on its own.

`ConstructVolumeSpec` is changed to fall back to the global file when the
pod-local load fails. It derives the `specVolID` from the basename of the
pod-local mount path (this is how kubelet names the directory) and scans
`/var/lib/kubelet/plugins/kubernetes.io/csi/*/*/vol_data.json` for a file
whose stored `specVolID` matches. The first match is used. If no match is
found (or the directory cannot be scanned), reconstruction fails as today.

The fallback is wrapped by `utilfeature.DefaultFeatureGate.Enabled(features.VolumeReconstructionFallback)`.
With the gate off, `ConstructVolumeSpec` behaves exactly as before. With the
gate on, only the failure path is altered: the success path is unchanged.

### User Stories

#### Story 1: kubelet OOM-kill leaves a partial vol_data.json

A node runs short on memory and the kernel kills kubelet mid-write to a pod's
`vol_data.json`. On the next kubelet start, reconstruction reads a truncated
JSON, errors out, the pod-local mount is unmounted, the global mount stays
live, and the controller eventually re-attaches the EBS volume to a different
node. Two pods now write to the same RWO filesystem and corrupt it.

With `VolumeReconstructionFallback` enabled, reconstruction reads the global
mount's `vol_data.json`, succeeds, and the volume goes through the normal
unmount path on both layers. No data corruption.

#### Story 2: operator deletes a pod-local vol_data.json by mistake

An operator investigating a stuck pod deletes the pod's volume directory
contents. Without the fallback, the symptom is the same as Story 1. With the
fallback, kubelet recovers automatically.

### Notes/Constraints/Caveats

- The fallback relies on `specVolID` being unique within a node. This is true
  by construction: `specVolID` is the PV name and a node has at most one mount
  per PV.
- Scanning `/var/lib/kubelet/plugins/kubernetes.io/csi/*/*/vol_data.json` is
  bounded by the number of CSI volumes attached to the node (typically tens,
  not thousands). The scan happens once per failed pod-local load during
  reconstruction.
- The fallback is per-pod-mount: if N pod-local files are missing, the scan
  runs N times. This is acceptable at alpha; if it shows up in profiles we
  will cache the scan result for the duration of a single reconstruction
  pass at beta.

### Risks and Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| Wrong global file is matched (specVolID collision across drivers) | Very low: specVolID is the PV name, unique cluster-wide | Only the first matching file is used; both files must agree on driverName for the spec to be valid; mismatch surfaces as a downstream error |
| Global vol_data.json is also corrupt | Medium: same root cause may have hit both files | Fallback returns an error and reconstruction fails with the original message plus a wrapped fallback message; behavior matches the no-fallback case |
| Feature gate disabled mid-cluster (skew) | Low | Field additions to global vol_data.json are written unconditionally (gate guards the read fallback only), so a node with the gate disabled still produces files a future enabled node can use |
| Stale global mount data after volume detach | Low | Detach unmounts and removes the global mount directory along with `vol_data.json`; if detach failed previously this KEP is exactly what is supposed to recover from it |

## Design Details

The change touches three CSI files in `pkg/volume/csi`:

1. `csi_attacher.go`: `MountDevice` adds `specVolID` (from `spec.Name()`) and
   `volumeLifecycleMode` (constant `string(storage.VolumeLifecyclePersistent)`)
   to the data map written to the global `vol_data.json`. Field additions are
   unconditional: written regardless of the feature gate, so a downgrade does
   not produce stale or partial files.

2. `csi_util.go`: new helper `findGlobalMountDataBySpecVolID(pluginDir, specVolID)`
   that does the glob, opens each match via the existing `loadVolumeData`,
   compares stored `specVolID`, and returns the first match's directory and
   parsed contents. Errors per-file are logged at V(4) and skipped; only a
   "no match found" condition is propagated up.

3. `csi_plugin.go`: `ConstructVolumeSpec` calls `loadVolumeData` as today.
   On error, if `VolumeReconstructionFallback` is enabled, it derives
   `specVolID` from `filepath.Base(mountPath)`, calls the helper, and on
   success continues with the parsed map. On failure of both loads, it
   returns the original error plus the fallback error in a wrapped message.
   The success path is unchanged.

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

- `k8s.io/kubernetes/pkg/volume/csi`: 76.2% (no significant change)

New unit test `TestPluginConstructVolumeSpecFallsBackToGlobalMount` in
`pkg/volume/csi/csi_plugin_test.go` exercises the fallback path:

1. Pod-local `vol_data.json` is absent.
2. A complete global `vol_data.json` (with all four fields) exists at
   `<plugin-dir>/<driver>/<hash>/vol_data.json`.
3. With the feature gate enabled, `ConstructVolumeSpec` returns a valid
   `volume.Spec` rebuilt from the global file.
4. With the feature gate disabled, the same call returns the original error.

##### Integration tests

None planned for alpha. The relevant code paths are reached only at kubelet
startup, which integration tests do not exercise meaningfully without a real
node.

##### e2e tests

For beta: a node e2e test that:

1. Mounts a CSI volume via a pod using a mock CSI driver.
2. Kills kubelet, deletes the pod-local `vol_data.json`, restarts kubelet.
3. Asserts the volume is reconstructed (no `volumesFailedReconstruction`
   entry, normal unmount on pod deletion).

Test will live in `test/e2e_node/csi_volume_reconstruction_test.go`.

### Graduation Criteria

#### Alpha

- Feature implemented behind `VolumeReconstructionFallback` (default off).
- Unit test for the fallback path.
- KEP merged.

#### Beta

- Node e2e test in CI for at least one release.
- Metrics: increment `reconstruct_volume_operations_total` with a label
  distinguishing pod-local vs global-fallback success, so operators can see
  fallback frequency.
- Two release cycles with no open bugs against the fallback path.
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
  - Feature gate name: `VolumeReconstructionFallback`
  - Components depending on the feature gate: `kubelet`

###### Does enabling the feature change any default behavior?

No. With both `vol_data.json` files intact, the success path is unchanged.
The feature only alters the failure path that today produces orphaned global
mounts.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate restores the previous behavior on the next kubelet
restart. There is no on-disk state migration. Any global `vol_data.json`
files written with the extra fields remain valid and are simply ignored by
the disabled code path.

###### What happens if we reenable the feature if it was previously rolled back?

Same as initial enablement: the next failed pod-local load triggers the
fallback. No state recovery needed.

###### Are there any tests for feature enablement/disablement?

The added unit test exercises both gate-on and gate-off paths.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The fallback runs only at kubelet startup during volume reconstruction.
Already running workloads are unaffected: the code path is not reached for
volumes that are already mounted and tracked in memory.

A rollout failure would manifest as a spurious successful reconstruction
that uses incorrect data. Mitigation: the helper compares `specVolID` exactly
and refuses to proceed without a match; the resulting `volume.Spec` is built
from the same fields that the pod-local file would have provided, so any
downstream component that previously trusted the pod-local file can trust
the global file.

###### What specific metrics should inform a rollback?

`reconstruct_volume_operations_errors_total` should not increase post-rollout
versus pre-rollout. If it does, disable the gate.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be exercised in the e2e test added at beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

At alpha: kubelet logs at V(2) emit `plugin.ConstructVolumeSpec recovered
vol_data from global mount for specVolID %q` whenever the fallback fires.

At beta: a label on the existing `reconstruct_volume_operations_total` metric
distinguishing `pod-local` vs `global-fallback`.

###### How can someone using this feature know that it is working for their instance?

Look for the V(2) log line above; or, after beta, inspect the metric label.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The fallback should add no more than tens of milliseconds per reconstructed
volume (one filesystem glob plus a small number of file reads). It should
not change overall kubelet startup time meaningfully.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- Count of `reconstruct_volume_operations_errors_total` (should not rise
  after enabling).
- Frequency of the V(2) fallback log line (should be near zero in healthy
  clusters; non-zero indicates a real corruption issue worth investigating
  separately).

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A separate counter for fallback successes vs failures will be added at beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. CSI drivers must implement `NodeStageVolume` (only such drivers create a
global mount in the first place); this is already standard.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Negligible. The added work runs only on the failure path of volume
reconstruction at kubelet startup.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Two extra string fields per global `vol_data.json` (a few dozen bytes).
Glob scan reads files that are already on disk.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature runs only at kubelet startup during local volume reconstruction
and does not contact the API server.

###### What are other known failure modes?

- Both `vol_data.json` files corrupt: reconstruction fails with a wrapped
  error message naming both files. Operator intervention required, same as
  today.
- Global plugin directory unreadable (permissions, filesystem error): the
  glob returns an error which is wrapped and returned. Same operator
  procedure as today.

###### What steps should be taken if SLOs are not being met to determine the problem?

If `reconstruct_volume_operations_errors_total` rises after enabling, capture
kubelet logs and inspect for the wrapped fallback error message. If the
issue is the fallback itself (not the underlying corruption), disable the
gate and report the bug.

## Implementation History

- 2026-04-18: Implementation PR opened against kubernetes/kubernetes ([#138454](https://github.com/kubernetes/kubernetes/pull/138454)).
- 2026-05-04: KEP drafted.

## Drawbacks

The fallback adds a small amount of complexity to the CSI plugin's
reconstruction path. The mitigating factor is that the alternative is to keep
shipping a known data-corruption bug behind the documentation in KEP-3756.

## Alternatives

1. Make the pod-local write atomic and durable enough that it can never be
   partial or missing. Useful but does not address operator-deletion or
   filesystem-level corruption, and atomic writes alone do not protect
   against disk-full at write time.

2. Detect the orphaned global mount during cleanup and unmount it
   defensively. Considered in early discussion of [#101791][]. The
   problem is that by the time `cleanupMounts` runs, the volume has already
   been removed from `volumesInUse`; the controller may have started a
   competing attach. The fallback approach prevents the volume from leaving
   `volumesInUse` in the first place.

3. Reconstruct purely from `/proc/mounts`. Possible for some plugins but
   loses the spec information CSI needs (driver-specific options, lifecycle
   mode, etc.). Falling back to a structured file we already write keeps the
   spec intact.
