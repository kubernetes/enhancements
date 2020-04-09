---
title: Extend usage of Volume DataSource to allow PVCs for Cloning
authors:
  - "@j-griffith"
owning-sig: sig-storage
participating-sigs:
  - sig-architecture
reviewers:
  - TBD
approvers:
  - "@saad-ali"
  - "@thockin"
editor: "@j-griffith"
creation-date: 2018-11-11
last-updated: 2020-01-27
status: implementable
see-also:
replaces:
superseded-by:
---

# Allow the use of the dataSource field for clones (existing PVCs)

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories [optional]](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
    - [Story 4](#story-4)
  - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Test Plan](#test-plan)
  - [Unit tests](#unit-tests)
  - [E2E tests](#e2e-tests)
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Summary

This KEP proposes adding support for specifying existing PVCs in the DataSource field to indicate a user would like to Clone a Volume.  Note that this KEP also applies ONLY to dynamic provisioner, and ONLY CSI Provisioner's.

For the purpose of this KEP, a Clone is defined as a duplicate of an existing Kubernetes Volume that can be consumed as any standard Volume would be.  The only difference is that upon provisioning, rather than creating a "new" empty Volume, the back end device creates an exact duplicate of the specified Volume.

Clones are different than Snapshots. A Clone results in a new, duplicate volume being provisioned from an existing volume -- it counts against the users volume quota, it follows the same create flow and validation checks as any other volume provisioning request, it has the same life cycle and work flow. Snapshots, on the other hand, results in a point-in-time copy of a volume that is not, itself, a usable volume -- it can be used to either to provision a new volume (pre-populated with the snapshot data) or to restore the existing volume to a previous state (represented by the snapshot).

## Motivation

Features like Cloning are common in most storage devices, not only is the capability available on most devices, it's also frequently used in various use cases whether it be for duplicating data or to use as a disaster recovery method.

### Goals

* Add ability to specify a PVC in a users Namespace as a DataSource
  - Add Core PVC Object to the permitted DataSource types to API Validation
* Provide ability to pass Clone intent to a CSI Plugin that reports it supports Clone capability
  - Proposal is limited to allowing a user to specify a Clone request, and for that Clone request to be passed to CSI Plugins that report support for cloning via capabilities

### Non-Goals

* This KEP does NOT propose the addition of other types of DataSource including Populators
* This KEP does NOT propose support for special cases like "out of band" cloning (support for back ends that don't have Cloning features), that sort of implementation would fall under Populators.
* This KEP does NOT propose any ability to shrink a PVC during a Clone request (e.g. it's considered an invalid request to clone PVC-a with a size of 10Gib to a PVC with a requested size of less than 10Gib, expansion is "ok" if the driver supports it but it's not required)
* This KEP does NOT propose adding any ability to transfer a Clone to a different Namespace, the new PVC (Clone) will be in the same Namespace as the origin that was specified.  This also means that since this is namespaced, a user can not request a Clone of a PVC that is another Namespace.  A user can only request a Clone of PVCs in his or her Namespace.
* Cloning will only be available in CSI, cloning features will NOT be added to existing in-tree plugins or Flex drivers
* Cloning will only be available within the same storage class (see [Implementation Details](#implementation-detailsnotesconstraints-optional) section for more info)

## Proposal

Add the Core object PVC to the allowed types for PVC DataSource.  Currently API validation only allows Snapshot Object Types, this proposal is to also add the Core PersistentVolumeClaim object as an accepted DataSource entry.

The following example assumes a PVC with the name `pvc-1` exists in the Namespace `myns` and has a size less than or equal to 10Gi:

```yaml
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
    name: pvc-2
    namespace: myns
spec:
  capacity:
    storage: 10Gi
  dataSource:
    kind: PersistentVolumeClaim
    name: pvc-1
```

The result would be a new and independent PV/PVC (pvc-2) on the back end device that is a duplicate of the data that existed on pvc-1.  It is required that this PVC be in the same Namespace as the original (that specified by DataSource).

### User Stories [optional]

#### Story 1
As a cluster user, I want to easily test changes to my production data base without risking issues to my customer facing applications

#### Story 2
As a cluster user, I want to be able to easily Clone a volume and run a different set of PODs/Applications against it

#### Story 3
As a cluster user, I want to be able to easily duplicate an existing deployment that's running on my Cluster to use for testing or the next version of my application

#### Story 4
As a cluster admin or user, I want to be able to provide the equivalent of data templates to users in the Cluster to ensure consistent and secure data sets

### Implementation Details/Notes/Constraints [optional]

This proposal requires adding PersistentVolumeClaims as allowed Object Types to the API validation checks against the DataSource field.  Currently the only allowed Object Type is SnapshotDataSource, this proposal would require the addition of the Core Object PersistentVolumeClaim as well, in addition to unit tests.  In addition this would also require a feature gate specifically for the Clone option (PVCDataSource).

Currently the CSI provisioner already accepts the DataSource field in new provisioning requests.  The additional implementation that's needed is to add acceptance of PVC types in the current CSI external-provisioner.  Once that's added, the PVC info can then be passed to the CSI Plugin in the DataSource field and used to instruct the backend device to create a Clone.

To emphasize above, this feature will ONLY be available for CSI.  This feature wil NOT be added to In-tree plugins or Flex drivers, this is strictly a CSI only feature.

It's important to note that we intentionally require that a source PVC be in the same StorageClass as the PVC being created.  This is currently required because the StorageClass determintes characteristics for a volume like `fsType`.  Performing a clone from an xfs volume to an ext4 volume for example would not be acceptable, given that a storageClass can have unique information, cloning across storage classes is not something we're able to try and determine safely at this time.

### Risks and Mitigations

The primary risk of this feature is requesting a PVC DataSource when using a CSI Driver that doesn't handle Cloning in a safe way for running applications.  It's assumed that the responsibility for reporting Clone Capabilities in this case is up to the CSI Driver, and if a CSI Driver is reporting Clone support that implies that they can in fact Clone Volumes without disrupting or corrupting users that may be actively using the specified source volume.

Due to the similarities between Clones and Snapshots, it is possible that some back ends may require queiscing in-use volumes before cloning.  This proposal suggests that initially, if a csi plugin is unable to safely perform the requested clone operation, it's the csi plugins responsibility to reject the request.  Going forward, when execution hooks are available (currently being proposed for consistent snapshots), that same mechanism should be made generally usable to apply to Clones as well.

## Test Plan

### Unit tests

Unit tests are already in place and include:
* Unit tests ensure that spec that includes PVC DataSource is interpretted correctly
* When a PVC DataSource is specified in the spec the resultant PVC object includes the corresponding DataSource entry

Additional unit tests to be added:
* Attempt to clone while in deleting/failed/in-use state

### E2E tests

Require addition of E2E tests using the clone feature of the CSI host provisioner
* Clone a volume that is in raw block mode
* Same invalid state testing as in unit tests; deleting/failed/in-use
* Multiple simultaneous clone requests to the same volume

## Graduation Criteria
* API changes allowing specification of a PVC as a valid DataSource in a new PVC spec
* Implementation of the PVC DataSource in the CSI external provisioner

Currently the only feature gate related to DataSources is the VolumeSnapshotDataSource gate.  This KEP would require an additional Data Source related feature gate `VolumeDataSource`.  Going forward we may continue to add additional feature gates for new DataSource types.  This KEP proposes that feature for Alpha, then following through the standard process for graduation based on feedback and stability during it's alpha release cycle.

Given that the only implementation changes in Kuberenetes is to enable the feature in the API (all of the actual Clone implementation is handled by the CSI Plugin and back end device) the main criteria for completion will be successful implementation and agreement from the CSI community regarding the Kubernetes API.

## Implementation History

1.15 - Alpha status
1.16 - Beta status

## Drawbacks [optional]

## Alternatives [optional]

Snapshots and Clones are very closely related, in fact some back ends my implement cloning via snapshots (take a snapshot, create a volume from that snapshot).  Plugins that provide true `smart clone` functionality are strongly discouraged from using this sort of an implementation, instead they should perform an actual clone if they have the ability to do so.

Users can do this currently with Kubernetes, and it's good, however some back ends have specific clone functionality that is much more efficient, and even for those that don't, this proposal provides a simple one-step process for a user to request a Clone of a volume.  It's also important to note that using this work around requires management of two object for the user, and in some cases those two object are linked and the new volume isn't truly an independent entity.

## Infrastructure Needed [optional]

