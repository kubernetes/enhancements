# KEP-5502: EmptyDir Volume Sticky Bit Support

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Shared Temporary Storage for Multi-User Workloads](#story-1-shared-temporary-storage-for-multi-user-workloads)
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
  - [Alternative 1: Provide more flexible mount options on emptyDir](#alternative-1-provide-more-flexible-mount-options-on-emptydir)
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
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding support for the sticky bit permission (mode 01777) to emptyDir volumes in Kubernetes. The sticky bit is a Unix file permission that restricts file deletion within a directory. Only the file owner, directory owner, or root can delete files, even if all users have write permission. Lack of a sticky bit on directories may result in being unable to use these as temporary directories for security reasons, making it impossible to use emptyDir and having to resort to ephemeral volumes.

## Motivation

The emptyDir volume currently creates directories with mode 0777, allowing any process with write access to delete or rename any file in the volume, regardless of who created it. This behavior can cause problems in multi-user or multi-process workloads where:

1. Multiple containers or processes running as different users share the same emptyDir volume
2. One process accidentally or maliciously deletes files created by another process
3. Init containers and main containers need to share files, but the main container should not be able to delete the init container's files

The sticky bit (mode 01777) is a standard Unix permission that solves this problem by ensuring that only the owner of a file (or the directory owner, or root) can delete or rename it, even when the directory is world-writable.

### Goals

- Add an optional `stickyBit` field to the emptyDir volume specification
- When enabled, create emptyDir volumes with mode 01777 instead of 0777
- Maintain backward compatibility by keeping the default behavior (mode 0777) unchanged
- Support the feature on all platforms that support Unix file permissions

### Non-Goals

- Changing the default behavior of existing emptyDir volumes (mode 0777 remains the default)
- Adding support for other advanced file permission features
- Implementing this feature for volume types other than emptyDir
- Supporting this feature on platforms that don't support Unix-style file permissions (e.g., Windows)

## Proposal

Add a new optional boolean field `stickyBit` to the `EmptyDirVolumeSource` API type. When set to `true`, the kubelet will create the emptyDir volume with mode 01777 (0777 | sticky bit) instead of the default 0777.

### User Stories

#### Story 1: Shared Temporary Storage for Multi-User Workloads

For containerized ruby apps, `/tmp` folders will be rejected if they do not have a sticky bit. This means `emptyDir` cannot be reliably used for tmp folders, and ephemeral volumes (more complex to manage) or RWX volumes have to be used (which are not well supported in many providers).

Allowing emptyDir to be mounted with sticky bit set would tremendously reduce complexity for these applications.

### Risks and Mitigations

**Risk**: Users might not understand the sticky bit behavior and be confused when they cannot delete files created by other users.

**Mitigation**: Document the feature clearly with examples. The feature is opt-in, so users must explicitly enable it.

**Risk**: The feature might not work correctly on all container runtimes or storage backends.

**Mitigation**: The sticky bit is a standard Unix permission supported by all major filesystems. The feature is opt-in (users must explicitly set `stickyBit: true`), allowing for gradual adoption and testing.

**Risk**: Existing workloads might be affected if the default changes.

**Mitigation**: The feature is opt-in via a new API field. Existing workloads will continue to use mode 0777 unless explicitly configured otherwise.

## Design Details

### API Changes

Add a new optional field to the `EmptyDirVolumeSource` struct:

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

### Implementation

The implementation is in the emptyDir volume plugin in `pkg/volume/emptydir/empty_dir.go`:

1. Define constants for the sticky bit mode:
   ```go
   const (
       stickyBitMode os.FileMode = 01000
       defaultPerm   os.FileMode = 0777
   )
   ```

2. When creating the emptyDir directory, check if the `StickyBit` field is set:
   ```go
   perm := defaultPerm
   if ed.stickyBit != nil && *ed.stickyBit {
       perm = defaultPerm | stickyBitMode
   }
   ```

3. Apply the appropriate permissions when creating the directory

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

#### Prerequisite testing updates

No prerequisite testing updates are required. The emptyDir volume plugin already has good test coverage.

#### Unit tests

Unit tests have been added to verify:
- Directory creation with sticky bit enabled results in mode 01777
- Directory creation with sticky bit disabled or unset results in mode 0777

Coverage:
- `pkg/volume/emptydir`: Unit tests cover the sticky bit implementation and default behavior

#### Integration tests

If needed, integration tests could additionally verify:
- A pod with emptyDir volume and stickyBit enabled mounts correctly
- Older kubelets ignore the field gracefully

#### e2e tests

TBD - e2e tests will be added as part of the implementation.

### Graduation Criteria

#### Alpha

- API field implemented and functional
- Unit tests passing
- Documentation available

#### Beta

- No major bugs reported during alpha
- Gather feedback from users

#### GA

- Stable for at least two releases
- No major issues reported

### Upgrade / Downgrade Strategy

No special upgrade/downgrade handling is needed. The `stickyBit` field is optional and ignored by older kubelets that don't recognize it.

### Version Skew Strategy

The feature is kubelet-only. Older kubelets will ignore the `stickyBit` field and create emptyDir volumes with the default mode 0777. This is safe as it matches the previous behavior.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: EmptyDirStickyBit
  - Components depending on the feature gate:
- [ ] Other
  - Will enabling / disabling the feature require downtime of the control plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No

###### Does enabling the feature change any default behavior?

No. The feature only takes effect when users explicitly set `stickyBit: true` on an emptyDir volume. Existing emptyDir volumes and new emptyDir volumes without the field continue to use mode 0777.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The feature is controlled per-pod by setting or omitting the `stickyBit` field.
When the feature gate is activate, to "disable" the feature, simply remove `stickyBit: true` from pod specs.

If rolling back to an older kubelet version that doesn't support the field, the field will be ignored and emptyDir volumes will be created with mode 0777.

**Impact on existing workloads**: Pods that were running with sticky bit enabled will continue to run unchanged (the directory permissions don't change after creation). However, new pods or pods that are rescheduled will have emptyDir volumes created with mode 0777 instead of 01777, which could affect application behavior if the application relies on the sticky bit behavior.

###### What happens if we reenable the feature if it was previously rolled back?

The feature will work as expected for new pods. Existing pods that were created while the feature was disabled will continue to use mode 0777 until they are deleted and recreated.

###### Are there any tests for feature enablement/disablement?

Yes, unit tests verify that:
- When `stickyBit: true`, the directory is created with mode 01777
- When `stickyBit` is false or unset, the directory is created with mode 0777
- The default behavior (mode 0777) is preserved when the field is not specified

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

**Rollout failure scenarios**:
- If the feature has bugs that cause emptyDir volume creation to fail, pods using `stickyBit: true` will fail to start
- If the host OS or filesystem doesn't support sticky bit (unlikely on standard Linux), volume creation could fail

**Impact on running workloads**:
- Already running workloads are not affected by enabling or disabling the feature
- Only new pods or rescheduled pods are affected
- The feature is opt-in, so workloads that don't use it are unaffected

**Rollback scenarios**:
- Rolling back to an older kubelet is safe and will not affect running pods
- On older kubelets, new pods with `stickyBit: true` will get mode 0777 instead of 01777 (the field is ignored), which is a functional change but not a failure

###### What specific metrics should inform a rollback?

Increased pod startup failures or volume mount errors correlated with pods using `stickyBit: true`.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. Will be tested manually before release.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Operators can:
1. Query the API server for pods with emptyDir volumes that have `stickyBit: true`:
   ```bash
   kubectl get pods -A -o json | jq '.items[] | select(.spec.volumes[]?.emptyDir?.stickyBit == true)'
   ```
2. Check kubelet logs for messages related to sticky bit creation
3. Inspect pod specifications directly

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treat as last resort)
  - Details: Users can verify the feature is working by:
    1. Creating a pod with an emptyDir volume with `stickyBit: true`
    2. Exec into the pod and check the directory permissions: `ls -ld /path/to/emptydir`
    3. Verify the permissions show `drwxrwxrwt` (mode 01777, the 't' at the end indicates sticky bit)
    4. Test the behavior by creating a file as one user and attempting to delete it as another user

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature should not affect existing SLOs. The performance impact should be negligible 

- emptyDir volume creation time should not be measurably affected
- Pod startup time should not be measurably affected

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name: storage_operation_duration_seconds (existing metric)
  - Components exposing the metric: kubelet
  - This metric can be filtered by operation_name="setup" to track emptyDir volume creation time

Operators should monitor:
- Pod startup failures
- Volume mount failures
- kubelet errors

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No additional metrics are needed. The feature is a simple file permission change and can be observed using existing pod and volume metrics.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. The feature only depends on:
- The host OS supporting the sticky bit permission (standard on all Linux systems)
- The filesystem supporting sticky bit (standard on all major filesystems)

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No. It adds a new field to an existing API type (EmptyDirVolumeSource).

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- API type(s): Pod (EmptyDirVolumeSource)
- Estimated increase in size: One additional boolean field per emptyDir volume that uses the feature, when set

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The performance impact should be negligible

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The feature only changes one argument to a mkdir system call.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature is implemented in the kubelet and does not depend on the API server or etcd after the pod spec has been retrieved. 
###### What are other known failure modes?

None beyond the standard emptyDir failure modes.

###### What steps should be taken if SLOs are not being met to determine the problem?

This feature should not affect SLOs. If pod startup or volume mounting SLOs are not being met, check if the affected pods are using `stickyBit: true` and verify kubelet logs for errors.

## Implementation History

- 2025-02-19 Initial implementation started (kubernetes/kubernetes#130277)
- 2025-08-25 KEP issue created (kubernetes/enhancements#5502)
- 2026-01-30: KEP created for alpha in v1.36

## Drawbacks

- Adds a new API field, slightly increasing API surface
- Users unfamiliar with Unix permissions may be confused by sticky bit behavior
- Not supported on Windows (but emptyDir permissions work differently there anyway)

## Alternatives

### Alternative 1: Provide more flexible mount options on emptyDir

There appears to be interested to provide more configuration options for mounting, that could entail setting permissions.

References: https://github.com/kubernetes/enhancements/pull/5856

## Infrastructure Needed (Optional)
