# Option to Preserve default file mode bit (Mode/DefaultMode) set in AtomicWriter volumes

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
- [Proposal](#proposal)
  - [Preserve DefaultMode Flag](#preserve-defaultmode-flag)
  - [File Permission](#file-permission)
    - [Proposed heuristics](#proposed-heuristics)
    - [Alternatives considered](#alternatives-considered)
  - [Scope Of the Change](#scope-of-the-change)
- [Graduation Criteria](#graduation-criteria)
<!-- /toc -->

## Release Signoff Checklist

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

Kubernetes allows users to use Secret (and other configs) as a VolumeSource
in pod spec. It also allows users to set a default file mode at the secret
or every file level.
The security context set for the pod (runAsUser or fsGroup) conflicts with
default mode specified for the secret/file. In current implementation of the
AtomicWriter volumes, default file mode is overwritten by the runAsUser or fsGroup
bit while setting the volume ownership.

Above results in unexpected behavior in cases where users want to run multiple
containers in a single pod and want them to run under a fsGroup, have different UIDs etc.
and yet have the ability to share same secret/config volume across containers.

For ex: If user has a ssh key, exposed as a secret volume to the pod, which runs
under security context of a fsGroup.
If user sets the DefaultMode on the secret volume to 256. User expects file mode to be set to 0400 on all files.
However, the actual file mode set on the files is 0440, as SetVolumeOwnership applies
the group ownership permissions to the files in the volume.

This proposal proposes to provide users with ability to preserve the file Mode/DefaultMode bit set on the
AtomicWriter volumes (Secret, ConfigMap etc) in an opt-in and backward compatible manner.

## Motivation

Many workloads running on Kubernetes today need the ability to run multiple containers
in the same pod. Primitives such as main, sidecars and init containers form the core
of distributed application development in kubernetes and are very widely accepted/used patterns.

Some workloads may want to run multiple containers in a single pod, under different UIDs and
have the ability to access a shared/common secret or config volume and yet have a
tighter control on the file mode set on the files inside the volume.

As a platform, Kubernetes should evolve to allow the sharing of AtomicWriter volumes (secret, config etc.)
across containers running under different UIDs as a first party scenario and allow users to control the
file mode bit set on the files in a predictable way.

With this feature, we try to provide a backwards compatible way for the users to
to have a tighter/predictable control over the file mode set on the files in AtomicWriter volumes.

We intend to provide a way for the users to opt-in for the new behavior and expect not to break
any existing applications/configurations.

## Proposal

Kubernetes should implement a new boolean flag (PreserveDefaultMode) in the SecretVolumeSource and
other AtomicWriter volumes.

The flag should be optional and default to false in code. This flag should be honored at the time of volume
set-up and call to SetVolumeOwnership should be skipped in case PreserveDefaultMode=true.

### Preserve DefaultMode Flag

A new PreserveDefaultMode boolean will be implemented in SecretVolumeSource and other AtomicWriter volumes.

```go
// SecretVolumeSource adapts a Secret into a volume.
//
// The contents of the target Secret's Data field will be presented in a volume
// as files using the keys in the Data field as the file names.
// Secret volumes support ownership management and SELinux relabeling.
type SecretVolumeSource struct {
    // Name of the secret in the pod's namespace to use.
    // +optional
    SecretName string
    // If unspecified, each key-value pair in the Data field of the referenced
    // Secret will be projected into the volume as a file whose name is the
    // key and content is the value. If specified, the listed keys will be
    // projected into the specified paths, and unlisted keys will not be
    // present. If a key is specified which is not present in the Secret,
    // the volume setup will error unless it is marked optional. Paths must be
    // relative and may not contain the '..' path or start with '..'.
    // +optional
    Items []KeyToPath
    // Mode bits to use on created files by default. Must be a value between
    // 0 and 0777.
    // Directories within the path are not affected by this setting.
    // This might be in conflict with other options that affect the file
    // mode, like fsGroup, and the result can be other mode bits set.
    // If PreserveDefaultMode is set to true then file mode bit is not
    // modified after the file creation, due to any other options set
    // such as fsGroup etc
    // +optional
    DefaultMode *int32
    // Specify whether the Secret or its key must be defined
    // +optional
    Optional *bool
    // Specify whether we want to preserve the DefaultMode set on the files
    // at the time of file creation.
    // If PreserveDefaultMode is set to true the file mode is not modified based
    // on fsGroup after the file is created with the defaultMode.
    // If the PreserveDefaultMode is set to false and fsGroup is specified
    // the volume is modified to be owned by the fsGroup.
    // If PreserveDefaultMode is not specified in the volume spec, it defaults to false.
    // +optional
    PreserveDefaultMode *bool
}
```

A secret volume source with preserveDefaultMode=true:

```yaml
  volumes:
  - name: test-secret
    secret:
      preserveDefaultMode: true
      defaultMode: 256
      secretName: test-secret
```

### File Permission

For the volumes that use AtomicWriter today the files are created with the Mode set in the KeyToPath struct.
If the Mode is not specified then the DefaultMode set for the source is used.
If DefaultMode is not specified then the file is created with mode bit set to 0600.
If pod have fsGroup specified then all the files in the volume are walked,
chown and chmod to the fsGroup

#### Proposed heuristics

- *Case 1*: The volume has PreserveDefaultMode set to false or not specified.
    In this case there is no behavioural change in how the file permissions are
    set for the volume. In other words, we will continue to create the files
    with Mode/Default and then set the volume ownership based on fsGroup if specified.
    This is the default case.
- *Case 2*: The volume has PreserveDefaultMode set to true.
    In this case the payload/volume/files are created using Mode/Default mode.
    Then the call to SetVolumeOwnership is skipped, in turn preserving the Default file mode on the files.

#### Alternatives considered

- Instead of having PreserveDefaultMode flag at the volume source level, we can have the same flag in
    the KeyToPath struct. That will allow us granular control over every file with-in the volume :
  - We would have to pass the payload information along with PreserveDefaultMode to the SetVolumeOwnership routine
        and skip chown/chmod if PreserveDefaultMode is set to true for the file.
  - This will be more invasive change, but would allow more granular control at per file level
  - If users want to preserve Mode for one file in the volume and not for another they can achieve that by creating
        two different volumes, with PreserveDefaultMode set to true for the volume where they want to preserve default mode for all files

### Scope Of the Change

With this change we will try to update the behavior for below AtomicWriter volumes

- SecretVolumeSource
- DownwardAPIVolumeSource
- ConfigMapVolumeSource
- ProjectedVolumeSource

## Graduation Criteria

TBD
