---
title: Skip Volume Ownership Change
authors:
  - "@mattsmithdatera"
  - "@gnuified"
owning-sig: sig-storage
participating-sigs: sig-auth
reviewers:
  - "@msau42"
  - "@liggit"
  - "@tallclair"
approvers:
  - "@saad-ali"
editor: TBD
creation-date: 2019-01-17
last-updated: 2019-01-30
status: implementable
see-also:
replaces:
superseded-by:

---

# Skip Volume Ownership Change

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Drawbacks [optional]](#drawbacks-optional)
* [Alternatives [optional]](#alternatives-optional)

## Summary

Currently before a volume is bind-mounted inside a container the permissions on
that volume are changed recursively to the provided fsGroup value.  This change
in ownership can take an excessively long time to complete, especially for very
large volumes (>=1TB) as well as a few other reasons detailed in [Motivation].
To solve this issue we will add a new field called `PermissionChangePolicy` and
allow the user to specify whether they want the ownership change to occur.

## Motivation

When a volume is mounted on the node, we recursively change permissions of volume
before bind mounting the volume inside container. The reason of doing this is to ensure
that volumes are readable/writable by provided fsGroup.

But this presents following problems:
 - An application(many popular databases) which is sensitive to permission bits changing
   underneath may refuse to start whenever volume being used inside pod gets mounted on
   different node.
 - If volume has a large number of files, performing recursive `chown` and `chmod`
   could be slow and could cause timeout while starting the pod.

### Goals

 - Allow volume ownership and permission to be skipped during mount

### Non-Goals

 - In some cases if user brings in a large enough volume from outside, the first time ownership and permission change still could take lot of time.
 - On SELinux enabled distributions we will still do recursive chcon whenever applicable and handling that is outside the scope.

## Proposal

We propose that an user can optionally opt-in to skip recursive ownership(and permission) change on the volume if volume already has right permissions.

### Implementation Details/Notes/Constraints [optional]

When creating a pod, we propose that `PersistentVolumeClaimVolumeSource` field expanded to include a new field called `PermissionChangePolicy` which can have following possible values:

 - `NoChange` --> Don't change permissions and ownership.
 - `Always` --> Always change the permissions and ownership to match fsGroup. This is the current behavior and it will be the default one when this proposal is implemented.
 - `OnDemand` --> Only change permissions when fsGroup of pod and PVC don't match.

```go
type PersistentVolumeClaimVolumeSource struct {
    // ClaimName is the name of a PersistentVolumeClaim in the same namespace as the pod using this volume
    ClaimName string
    // Optional: Defaults to false (read/write).  ReadOnly here
    // will force the ReadOnly setting in VolumeMounts
    // +optional
    ReadOnly bool
    // PermissionChangePolicy ← new field
    PermissionChangePolicy string
}
```

In addition to this, we propose a new field to PVC's status:

```go
type PersistentVolumeClaimStatus struct {
     // Phase represents the current phase of PersistentVolumeClaim
     // +optional
     Phase PersistentVolumeClaimPhase
     // AccessModes contains all ways the volume backing the PVC can be mounted
     // +optional
     AccessModes []PersistentVolumeAccessMode
     // Represents the actual resources of the underlying volume
     // +optional
     Capacity ResourceList
     // FSGroup of PVC
     // + optional
     FSGroup *int64     // <------ NEW ------
     // +optional
     Conditions []PersistentVolumeClaimCondition
}
```


When a volume is mounted by the kubelet, volumemanager will check
`pvc.Status.FSGroup` and `pod.SecurityContext.FSGroup` and depending on
`PermissionChangePolicy` of `PersistentVolumeClaimVolumeSource` it will:

 - Do no permission and ownership change if `PermissionChangePolicy` is
   `NoChange`.
 - Will change ownership and permission of volume to match
   `pod.SecurityContext.FSGroup` if `PermissionChangePolicy` is set to
   `Always`.
 - if `PermissionChangePolicy` is set to `OnDemand`, volume permissions will
   ONLY be changed if `pvc.Status.FSGroup` and `pod.SecurityContext.FSGroup`
   don't have the same value.

After permissions are changed, pvc.Status.FSGroup will be updated to reflect
latest value.

Currently Kubelet can update PVC's Status if `ExpandPersistentVolumes` feature
is enabled. This feature similarly depends on the ability of kubelet to update
`pvc.Status.FSGroup` and permissions of node will be expanded to include it

### Risks and Mitigations

## Graduation Criteria

* Alpha in 1.14 provided all tests are passing and gated by the feature Gate
   ConfigurableVolumeFilePermissions and set to a default of `False`

* Beta in 1.15 with design validated by at least two customer deployments
  (non-production), with discussions in SIG-Storage regarding success of
  deployments.  A metric will be added to report time taken to perform a
  volume ownership change.
* GA in 1.16, with Node E2E tests in place tagged with feature Storage


[umbrella issues]: https://github.com/kubernetes/kubernetes/issues/69699

### Test Plan

A test plan will consist of the following tests

* Basic tests including a permutation of the following values
  - PersistentVolumeClaimVolumeSource.PermissionChangePolicy (Never/OnDemand/Always)
  - PersistentVolumeClaimStatus.FSGroup (matching, non-matching)
  - Volume Filesystem existing permissions (none, matching, non-matching, partial-matching?)
* E2E tests


### Monitoring

We will add a metric that measures the volume ownership change times.

## Implementation History

- 2019-01-17 Initial KEP pull request submitted

## Drawbacks [optional]

 - It will only work for PVC sources and not for inline volumes. But I do not see this being a huge deal.
 - In some cases it could result in volume being mounted as not readable/writable from within the pod, but I think this can be easily fixed by changing `PermissionChangePolicy`.

## Alternatives [optional]

## Infrastructure Needed [optional]
