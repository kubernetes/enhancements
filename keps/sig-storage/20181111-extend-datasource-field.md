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
  - @saad-ali
  - @thockin
editor: @j-griffith
creation-date: 2018-11-11
last-updated: 2019-02-25
status: provisional
see-also:
replaces:
superseded-by:
---

# Allow the use of the dataSource field for clones (existing PVCs)

## Table of Contents

* [Table of Contents](#table-of-contents)
* [Summary](#summary)
* [Motivation](#motivation)
    * [Goals](#goals)
    * [Non-Goals](#non-goals)
* [Proposal](#proposal)
    * [User Stories [optional]](#user-stories-optional)
      * [Story 1](#story-1)
      * [Story 2](#story-2)
    * [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    * [Risks and Mitigations](#risks-and-mitigations)
* [Graduation Criteria](#graduation-criteria)
* [Implementation History](#implementation-history)
* [Drawbacks [optional]](#drawbacks-optional)
* [Alternatives [optional]](#alternatives-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Summary

This KEP proposes adding support for specifying existing PVCs in the DataSource field to indicate a user would like to Clone a Volume.  Note that this KEP also applies ONLY to dynamic provisioner, and ONLY CSI Provisioner's.

For the purpose of this KEP, a Clone is defined as a duplicate of an existing Kubernetes Volume that can be consumed as any standard Volume would be.  The only difference is that upon provisioning, rather than creating a "new" empty Volume, the back end device creates an exact duplicate of the specified Volume.

Clones are different than Snapshots, a Clone is "another Volume", it counts against user volume quota, it follows the same create flow and validation checks as any other PVC request, it has the same life-cycle and work flow.  While Snapshots are unique objects with their own API, they're commonly used for backups.
(See alternatives section for info regarding Snapshot implemented Clones).

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

### Risks and Mitigations

The primary risk of this feature is a Plugin that doesn't handle Cloning in a safe way for running applications.  It's assumed that the responsibility for reporting Clone Capabilities in this case is up to the Plugin, and if a Plugin is reporting Clone support that implies that they can in fact Clone Volumes without disrupting or corrupting users that may be actively using the specified source volume.

Due to the similarities between Clones and Snapshots, it is possible that some back ends may require queiscing in-use volumes before cloning.  This proposal suggests that initially, if a plugin is unable to safely perform the requested clone operation, it's the plugins responsibility to reject the request.  Going forward, when execution hooks are available (currently being proposed for consistent snapshots), that same mechanism should be made generally usable to apply to Clones as well.

## Graduation Criteria
* API changes allowing specification of a PVC as a valid DataSource in a new PVC spec
* Implementation of the PVC DataSource in the CSI external provisioner

The API can be immediately promoted to Beta, as it's just an extension of the existing DataSource field.  There are no implementations or changes needed in Kubernetes other than accepting Object Types in the DataSource field.  This should be promoted to GA after a release assuming no major issues or changes were needed/required during the Beta stage.

## Implementation History

## Drawbacks [optional]

## Alternatives [optional]

Snapshots and Clones are very closely related, in fact some back ends my implement cloning via snapshots (take a snapshot, create a volume from that snapshot).  Users can do this currently with Kubernetes, and it's good, however some back ends have specific clone functionality that is much more efficient, and even for those that don't, this proposal provides a simple one-step process for a user to request a Clone of a volume.

## Infrastructure Needed [optional]

