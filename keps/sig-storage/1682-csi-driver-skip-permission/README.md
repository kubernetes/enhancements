# KEP-1682: Allow CSIDriver opt-in to volume ownership and permission changes

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history) 
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website


## Summary

Currently before bind mounting a CSI volume for a Pod, we use imprecise heuristics,
such as presence of fsType on the PVC to determine if the volume supports fsGroup based
permission change. These heuristics are known to be fragile, and cause problems with different
storage types.

To solve this issue we will add a new field called `CSIDriver.Spec.SupportsFSGroup` 
that allows the driver to define if it supports volume ownership modifications via
fsGroup.

## Motivation

When a volume is mounted on the node, we modify volume ownership 
before bind mounting the volume inside container; however, if the volume
does not support these operations (such as NFS), then the change is still attempted. 
This results in errors being reported to the user if the volume doesn't 
support these operations.

### Goals

 - Allow CSIDrivers to opt-in to volume ownership changes.

### Non-Goals

 - This does not attempt to update existing CSIDrivers themselves.

## Proposal

We propose that the `CSIDriver` type include a field that defines if the volume 
provided by the driver supports changing volume ownership. This will be enabled
with a new feature gate, `CSIVolumeSupportFSGroup`.   

### Risks and Mitigations

- The CSIDriver objects will need to be redeployed after this field is introduced if the desired behavior is modified.
- If a cluster enables the `CSIVolumeSupportFSGroup` feature gate and then this feature gate is disabled,
such as due to an upgrade or downgrade, then the cluster will revert to the current behavior of examining
volumes and attempting to apply volume ownerships and permissions based on the defined `fsGroup`.

## Design Details

Currently volume permission and ownership change is examined for every volume. If the PodSecurityPolicy's
`fsGroup` is defined, `fsType` is defined, and the PersistentVolumes's `accessModes` is RWO, then we will 
attempt to modify the volume ownership and permissions.

As part of this proposal we will change the algorithm that modifies volume ownership and permissions
for CSIDrivers to check the new field, and skip volume ownership modifications if it is found to be
`Never`.

When defining a `CSIDriver`, we propose that `CSIDriver.Spec` be expanded to include a new field entitled 
`SupportsFSGroup` which can have following possible values:

 - `OnlyRWO` --> Current behavior. Attempt to modify the volume ownership and permissions to the defined `fsGroup` when the volume is 
 mounted if accessModes is RWO.
 - `Never` --> New behavior. Attach the volume without attempting to modify volume ownership or permissions.
 - `Always` --> New behavior. Always attempt to apply the defined fsGroup to modify volume ownership and permissions.

```go
type SupportsFsGroup string

const(
    OnlyRWO SupportsFsGroup = "OnlyRWO"
    Always SupportsFsGroup = "Always"
    Never SupportsFsGroup = "Never"
)

type CSIDriverSpec struct {
    // SupportsFSGroup ← new field
    // Defines if the underlying volume supports changing ownership and 
    // permission of the volume before being mounted. 
    // If set to Always, SupportsFSGroup indicates that 
    // the volumes provisioned by this CSIDriver support volume ownership and 
    // permission changes, and the filesystem will be modified to match the 
    // defined fsGroup every time the volume is mounted.
    // If set to Never, then the volume will be mounted without modifying
    // the volume's ownership or permissions.
    // Defaults to OnlyRWO, which results in the volume being examined
    // and the volume ownership and permissions attempting to be updated
    // only when the PodSecurityPolicy's fsGroup is explicitly defined, the
    // fsType is defined, and the PersistentVolumes's accessModes is RWO.
    // + optional
    SupportsFSGroup *SupportsFsGroup
}
```
### Test Plan

A test plan will include the following tests:

* Basic tests including a permutation of the following values:
  - CSIDriver.Spec.SupportsFSGroup  (`Always`/`Never`/`OnlyRWO`)
  - PersistentVolumeClaim.Status.AccessModes (`ReadWriteOnly`, `ReadOnlyMany`,`ReadWriteMany`)
* E2E tests

### Graduation Criteria

* Alpha in 1.19 provided all tests are passing.
* All functionality is guarded by a new alpha `CSIVolumeSupportFSGroup` feature gate.

* Beta in 1.20 with design validated by at least two customer deployments
  (non-production), with discussions in SIG-Storage regarding success of
  deployments. 
* The `CSIVolumeSupportFSGroup` feature gate will graduate to beta.


* GA in 1.21, with E2E tests in place tagged with feature Storage.
* The `CSIVolumeSupportFSGroup` feature gate will graduate to GA.

[issues]: https://github.com/kubernetes/enhancements/issues/1682

## Production Readiness Review Questionnaire

### Feature enablement and rollback
* **How can this feature be enabled / disabled in a live cluster?**
  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: CSIVolumeSupportFSGroup 
    - Components depending on the feature gate: kubelet

* **Does enabling the feature change any default behavior?**
  Enabling the feature gate will **not** change the default behavior.
  Users must also define the `SupportsFsGroup` type for behavior to
  be modified.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  Yes. Disabling the feature gate will revert back to the existing behavior.

* **What happens if we reenable the feature if it was previously rolled back?**
  If reenabled, any subsequent CSIDriver volumes that are mounted
  will respect the user-defined values for `SupportsFSGroup`. Existing mounted
  volumes will not be modified.


* **Are there any tests for feature enablement/disablement?**
  We will need to create unit tests that enable this feature as outlined
  in [Test Plan](#test-plan).

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g. what if some components will restart
  in the middle of rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring requirements

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
There are no dependencies for this feature. It is a modification to an
existing API.

### Scalability
* **Will enabling / using this feature result in any new API calls?**
There will be no new API calls. A new API field will be added to the CSIDriver
API, which will be used to determine if we should continue with applying
permission and ownership changes.

* **Will enabling / using this feature result in introducing new API types?**
No new API types, but there will be new fields in the CSIDriver API, as
described in [Design Details](#design-details).

* **Will enabling / using this feature result in any new calls to cloud provider?**
There should be no new calls to the cloud providers.

* **Will enabling / using this feature result in increasing size or count 
  of the existing API objects?**
There will be a new string that may be defined on each CSIDriver.

* **Will enabling / using this feature result in increasing time taken by any
  operations covered by [existing SLIs/SLOs][]?**
There should be no increase to the amount of time taken.

* **Will enabling / using this feature result in non-negligible increase of
  resource usage (CPU, RAM, disk, IO, ...) in any components?**
There should be no noticeable increase to resource usage for any components.

### Troubleshooting

## Implementation History

- 2020-04-27 Initial KEP pull request submitted
- 2020-05-12 Updated to use new KEP template
