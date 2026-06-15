# KEP-5502: EmptyDir Volume Permission Mode

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Restricted Directory Permissions](#story-1-restricted-directory-permissions)
    - [Story 2: Shared Temporary Storage with Sticky Bit](#story-2-shared-temporary-storage-with-sticky-bit)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Implementation](#implementation)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (target 1.37)](#alpha-target-137)
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
    - [Alternative 1: Boolean stickyBit field](#alternative-1-boolean-stickybit-field)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation---e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

EmptyDir volumes are always created with mode `0777`, and there's no way to change it. This KEP adds an optional `mode` field to `EmptyDirVolumeSource` that lets users set the directory permission bits -- anything from `0000` to `01777`. This follows the same `defaultMode` pattern already established by Secret, ConfigMap, and DownwardAPI volume sources.

## Motivation

The emptyDir volume creates directories with a hardcoded mode of `0777`. This means any process with access can read, write, and delete anything in the volume, regardless of who created it. There is no way to change this today.

This causes real problems:

1. Multi-container pods sharing an emptyDir cannot prevent one container from deleting another's files. The sticky bit (`01777`) would solve this, but there's no way to set it.
2. For containerized applications (e.g., Ruby apps), `/tmp` directories without a sticky bit are rejected for security reasons. This means emptyDir can't reliably serve as `/tmp`, forcing users to resort to ephemeral volumes or RWX volumes - both significantly more complex to manage and not well supported across providers.
3. Platform engineers who want tighter permissions (e.g., `0750` for owner+group only) have to use init containers running `chmod`, which adds unnecessary complexity.

Secret, ConfigMap, and DownwardAPI volume sources all expose a `defaultMode` field for controlling permissions. EmptyDir is the only volume type that lacks any permission control.

### Goals

- Add an optional `mode` field to `EmptyDirVolumeSource` that accepts Unix permission values between `0000` and `01777`
- Allow users to set directory permissions at creation time (e.g., `0750` for restricted access, `01777` for sticky bit)
- Keep the default behavior unchanged - mode `0777` when the field is not set
- Follow the existing `defaultMode` pattern used by Secret, ConfigMap, and DownwardAPI volume sources

### Non-Goals

- Changing the default behavior of existing emptyDir volumes
- Implementing this for volume types other than emptyDir
- Supporting this on platforms that don't have Unix-style file permissions (e.g., Windows)

## Proposal

Add a new optional field `mode` to `EmptyDirVolumeSource`. The field takes an integer between `0000` and `01777` representing standard Unix permission bits. When set, the kubelet creates the emptyDir directory with the specified permissions. When not set, the existing behavior (`0777`) is preserved.

```yaml
volumes:
  - name: app-data
    emptyDir:
      mode: 0750
```

### User Stories

#### Story 1: Restricted Directory Permissions

As a platform engineer, I want to restrict an emptyDir volume so that only the application user and its group can access it. Setting `mode: 0750` ensures the directory is not world-readable or writable, reducing the attack surface for multi-tenant workloads.

```yaml
volumes:
  - name: app-data
    emptyDir:
      mode: 0750
containers:
  - name: app
    volumeMounts:
      - name: app-data
        mountPath: /data
```

#### Story 2: Shared Temporary Storage with Sticky Bit

As a developer running containerized workloads, I need a shared `/tmp` directory where all users can write but only file owners can delete their own files. Without the sticky bit, emptyDir cannot be reliably used as a `/tmp` mount - some application frameworks (e.g., Ruby) reject `/tmp` directories that don't have sticky bit set, and multi-process workloads risk one process deleting files owned by another.

Setting `mode: 01777` gives standard `/tmp` behavior.

```yaml
volumes:
  - name: tmp
    emptyDir:
      mode: 01777
containers:
  - name: app
    volumeMounts:
      - name: tmp
        mountPath: /tmp
```

### Risks and Mitigations

**Risk**: Users set incorrect permission values without understanding Unix modes.
**Mitigation**: The feature is opt-in. Default `0777` is preserved when `mode` is not set. Documentation will include common examples (`0755`, `0700`, `01777`). Validation rejects values outside the `0000`-`01777` range.

**Risk**: The `mode` field may conflict with `fsGroup` in the pod security context, which modifies group ownership and permission bits on mounted volumes.
**Mitigation**: This is the same interaction that already exists for `SecretVolumeSource.defaultMode` and `ConfigMapVolumeSource.defaultMode`. The API comment will document this caveat, consistent with how those volume sources handle it today.

**Risk**: Existing workloads could be affected.
**Mitigation**: The default is unchanged. Existing workloads continue to use mode `0777` unless explicitly configured.

## Design Details

### API Changes

Add a new optional field to the `EmptyDirVolumeSource` struct:

```go
type EmptyDirVolumeSource struct {
    // ... existing fields (Medium, SizeLimit) ...

    // mode specifies the permission bits for the emptyDir directory, in numeric
    // notation (e.g., 0777, 01777). Must be a value between 0000 and 01777.
    // If not specified, defaults to 0777.
    // This might be in conflict with other options that affect the file
    // mode, like fsGroup, and the result can be other mode bits set.
    // +featureGate=EmptyDirVolumeMode
    // +optional
    Mode *int32 `json:"mode,omitempty" protobuf:"varint,3,opt,name=mode"`
}
```

The feature is gated behind `EmptyDirVolumeMode`. When the gate is disabled:
- **kube-apiserver**: Strips the `mode` field from new pod specs via field dropping (unless already persisted on an existing pod).
- **Validation**: Rejects pods with `mode` set outside `0000`-`01777`.

### Implementation

The implementation is in the emptyDir volume plugin (`pkg/volume/emptydir/empty_dir.go`):

1. Read the `mode` field from `EmptyDirVolumeSource`:
   ```go
   perm := os.FileMode(0777)
   if ed.mode != nil {
       perm = os.FileMode(*ed.mode)
   }
   ```

2. Apply the permissions when creating the directory:
   ```go
   if err := os.MkdirAll(dir, perm); err != nil {
       return err
   }
   // MkdirAll applies the umask, so explicitly chmod to set the exact mode
   if err := os.Chmod(dir, perm); err != nil {
       return err
   }
   ```

The `os.Chmod` after `os.MkdirAll` is necessary because `MkdirAll` applies the process umask, which may strip requested bits. `Chmod` sets the exact mode regardless of umask.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates are required.

##### Unit tests

Unit tests will cover:
- Directory creation with `mode: 0750` results in mode `0750`
- Directory creation with `mode: 01777` results in mode `01777`
- Directory creation without `mode` defaults to `0777`
- Mode is applied correctly for all emptyDir medium types (default, tmpfs, hugepages)
- Validation rejects values outside `0000`-`01777`
- Feature gate drop test: `mode` is stripped when the gate is disabled, preserved when enabled

Coverage targets:
- `pkg/volume/emptydir`
- `pkg/apis/core/validation`
- `pkg/api/pod`

##### Integration tests

No integration tests are planned. The feature is a simple permission change at directory creation time and is well covered by unit and e2e tests.

##### e2e tests

`e2e_node` tests will verify:
- A pod with emptyDir `mode: 0750` has the correct permissions (verified via `stat`)
- A pod with emptyDir `mode: 01777` has the correct permissions including the sticky bit
- Default behavior is preserved when `mode` is not set

### Graduation Criteria

#### Alpha (target 1.37)

- Feature implemented behind `EmptyDirVolumeMode` feature gate (disabled by default)
- Kubelet applies `mode` when creating emptyDir volumes
- Unit tests and `e2e_node` tests completed

#### Beta

- Feature gate enabled by default
- No major bugs reported during alpha
- Address any feedback from alpha users
- Upgrade/downgrade testing completed

#### GA

- At least 2 releases in beta with no major bugs
- Feature gate removed
- Conformance tests in place

### Upgrade / Downgrade Strategy

**Upgrade**: Enabling the feature gate allows new pods to use the `mode` field. Existing running pods are unaffected - their directories are already created.

**Downgrade**: Disabling the feature gate causes:
- The API server strips `mode` from new pod specs via field dropping.
- The kubelet ignores `mode` and creates emptyDir volumes with the default `0777`.

Running pods are not affected by downgrade. Only newly created or rescheduled pods are affected.

### Version Skew Strategy

The `mode` field is optional and additive:

- **Apiserver ON, kubelet OFF**: The apiserver accepts `mode`, but the kubelet doesn't recognize it and falls back to `0777`. Safe - matches previous behavior, but the user's requested permissions are not applied.
- **Apiserver OFF, kubelet ON**: The apiserver strips `mode`. The kubelet never sees it. Falls back to `0777`.
- **Both ON**: Full enforcement.
- **Both OFF**: Feature disabled, existing behavior.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `EmptyDirVolumeMode`
  - Components depending on the feature gate: kubelet, kube-apiserver

###### Does enabling the feature change any default behavior?

No. The feature only takes effect when users explicitly set the `mode` field. Existing emptyDir volumes continue to use mode `0777`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Set the `EmptyDirVolumeMode` feature gate to `false`. The API server will strip `mode` from new pods, and the kubelet will ignore it. Existing running pods are unaffected - their directories are already created. New pods will get `0777`.

###### What happens if we reenable the feature if it was previously rolled back?

Works as expected for new pods. Existing pods created while the feature was disabled keep `0777` until they are deleted and recreated.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests verify that `mode` is stripped when the gate is disabled and preserved when enabled or when the field is already persisted.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Running workloads are not affected. The feature is opt-in, so only pods that explicitly set `mode` are affected. Rolling back to an older kubelet is safe - `mode` is ignored and emptyDir volumes get `0777`.

###### What specific metrics should inform a rollback?

Increased pod startup failures or volume mount errors correlated with pods using the `mode` field.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested manually before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

```bash
kubectl get pods -A -o json | jq '.items[] | select(.spec.volumes[]?.emptyDir?.mode != null)'
```

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: Exec into the pod and check directory permissions with `stat -c '%a' /path/to/emptydir`. For example, mode `0750` should output `750`, and mode `01777` should output `1777`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No impact on existing SLOs. The change is one argument to a `mkdir` + `chmod` system call.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `storage_operation_duration_seconds` (existing metric)
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It only requires the host OS to support Unix file permissions (standard on Linux).

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No. It adds a field to an existing type (`EmptyDirVolumeSource`).

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type: Pod (`EmptyDirVolumeSource`)
- Estimated increase: One int32 field per emptyDir volume that uses the feature.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature runs in the kubelet and doesn't contact the API server or etcd after the pod spec is retrieved.

###### What are other known failure modes?

None beyond standard emptyDir failure modes.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check if affected pods use the `mode` field and review kubelet logs for errors.

## Implementation History

- 2025-02-19: Initial implementation started (kubernetes/kubernetes#130277)
- 2025-08-25: KEP issue created (kubernetes/enhancements#5502)
- 2026-01-30: Initial KEP draft with boolean `stickyBit` field
- 2026-06-15: KEP redesigned to use `mode *int32` field, targeting alpha in v1.37

## Drawbacks

- Adds a new API field, slightly increasing API surface.
- Users unfamiliar with Unix permissions may set incorrect values. Mitigated by validation and documentation.
- The `mode` field may interact with `fsGroup` in the pod security context. This is the same caveat that exists for Secret/ConfigMap `defaultMode` and is documented in the API comment.
- Not supported on Windows, but emptyDir permissions work differently there anyway.

## Alternatives

#### Alternative 1: Boolean stickyBit field

An alternative design is to add an optional `StickyBit` field to the `EmptyDirVolumeSource` struct:

```go
type EmptyDirVolumeSource struct {
    // ... existing fields ...

    // StickyBit sets the emptyDir permission to 01777 instead of 0777.
    // When enabled, only the owner of a file can delete or rename it,
    // even if the directory is world-writable.
    // This is similar to the /tmp directory behavior on Unix systems.
    // +optional
    StickyBit *bool `json:"stickyBit,omitempty" protobuf:"varint,3,opt,name=stickyBit"` 
}
```

When `true`, the directory would be created with mode `01777` instead of `0777`. This was not chosen because:

- **Limited scope**: Only solves the sticky bit use case. Every future permission request (e.g., tighter directory ownership) would require a new field.
- **Inconsistent with existing patterns**: Secret, ConfigMap, and DownwardAPI volume sources all use `defaultMode *int32` for permission control. A boolean is inconsistent with this established pattern.
- **Less flexible**: Cannot express permissions like `0750` or `0700`. The `mode` field covers all Unix permission use cases with a single field.

## Infrastructure Needed (Optional)
