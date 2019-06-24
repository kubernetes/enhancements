---
title: Namespace Transfer API Proposal
authors:
  - "@j-griffith"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
  - sig-architecture
  - sig-api-machinery
reviewers:
  - @thockin
  - @saad-ali
  - "@alicedoe"
approvers:
  - @thockin
  - @saad-ali
editor: @j-griffith
creation-date: 2019-06-21
last-updated: 2019-06-21
status: provisional
see-also:
replaces:
superseded-by:
---

# propose-namespace-transfer-api

## Table of Contents

- [Title](#title)
  - [Table of Contents](#table-of-contents)
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
  - [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [User Stories [optional]](#user-stories-optional)
      - [Story 1](#story-1)
      - [Story 2](#story-2)
    - [Implementation Details/Notes/Constraints [optional]](#implementation-detailsnotesconstraints-optional)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Test Plan](#test-plan)
    - [Graduation Criteria](#graduation-criteria)
      - [Examples](#examples)
        - [Alpha -> Beta Graduation](#alpha---beta-graduation)
        - [Beta -> GA Graduation](#beta---ga-graduation)
        - [Removing a deprecated flag](#removing-a-deprecated-flag)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
  - [Implementation History](#implementation-history)
  - [Drawbacks [optional]](#drawbacks-optional)
  - [Alternatives [optional]](#alternatives-optional)
  - [Infrastructure Needed [optional]](#infrastructure-needed-optional)

[Tools for generating]: https://github.com/ekalinin/github-markdown-toc

## Release Signoff Checklist

- [ ] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [ ] KEP approvers have set the KEP status to `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

The intention of this KEP is to define a general API to enable the transfer of resources from one namespace to another.  This proposal suggests the use of an external CRD that includes a `TransferRequest` and `TransferApproval` object.  Upon availability of the external API, implementations can be provided for both core kubernetes objects as well as external CRD objects to be transferred across namespaces.  This proposal proposes only the API definition itself, and uses `PersistentVolumeClaims` as an example only.  Implementations for each object type are independent works that need to be handled on an object basis.  In addition to defining the `TransferRequest` and `TransferApproval` objects, it's also necessary to provide a mechanism to indicate to a user what objects are transferable.  The ability to transfer resources across namespaces is not something that makes sense (or is even possible) for all kubernetes objects, implementation of namespace transfer is not intended to be a required feature for all object types, but merely provide a consistent API for those objects that choose to implement the functionality. 

While the concept of transferring an object across namespaces is not necessarily something that fits for "all" resource types, there are a number of storage objects today that this would be useful; volumes, snapshots, clones etc.  In addition, the number of data related objects in kubernetes is expected to grow, and include things like external populators, backups and other content sources.  It's important to plan ahead and provide a shared API that can be used the same way regardless of object type if and when it makes sense and they have a valid use case for transfer functionality.

## Motivation

* Provide a standard API for the transfer of kubernetes objects from one namespace to another

### Goals

* Standardized API and process for performing a namespace transfer regardless of object
* Enable implementation for performing a namespace transfer for core objects
* Enable implementation for performing a namespace transfer for external (CRD) objects
* Define a mechanism to indicate whether an object implements the namespace transfer API

### Non-Goals

* Actual implementation of transferring an object between namespaces

## Proposal

Introduce an external `NamespaceTransferRequest` object:

```yaml
apiversion: v1alpha1
kind: NamespaceTransferRequest
metadata:
    name: pvc-transfer-request
    namespace:  destination-namespace
spec:
    source:
        name: source-namespace
        name: source-pvc
        kind: PersistentVolumeClaim

```

Introduce an external `NamespaceTransferApproval` object:

```yaml
apiversion: v1alpha1
kind: NamespaceTransferApproval
metadata:
    name: pvc-transfer-approval
    namespace: source-namespace
spec: 
    source:
        name: source-pvc
        kind: PersistentVolumeClaim
    targetNamespace: destination-namespace

```

Introduce a parameter to objects (including the existing `kubernetes.pkg.apis.core.types.go.ObjectMeta` structure 
to indicate if an object is transferable:

```go
type ObjectMeta struct {
      ...
      Name string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	  // Transferable indicates whether the object implements the namespace transfer API.  By default
	  // the controller responsible for creation of the resource will set this appropriately based on
	  // whether or not it implements the namespace transfer API.  A user may also explicitly set this
	  // to `false` if they wish to prevent any transfer operations against the particular resource being
	  // created regardless of the controllers capabilities.
	  // +optional
	  Transferable bool `json:"export,omitempty" protobuf:"varint,1,opt,name=transferable"`
	  ...
}

### Example Implementation of the API

We have a controller for each type of object we want to transfer (e.g. PVC, VolumeSnapshots, AppSnapshot, etc.) -- this way the logic for how transfer happens is custom per object type.

Proposed logic for PVC transfer controller:

The existing PV/PVC controller can be modified to include new pvc-transfer-controller logic. It will be responsible for transferring PVCs across namespaces. It will do the following:

1. Monitor NamespaceTransferApproval, NamespaceTransferRequest, and PVC objects.
2. Wait for a destination PVC to be created where PVC.DataSource points to a NamespaceTransferRequest (LocalObjectReference).
3. Wait for that NamespaceTransferRequest and a matching NamespaceTransferApproval objects to exist. Matching means the request.source and approval.source match, the request.namespace and approval.targetNamespace match, and the request.source.kind is type PVC (this logic could be put in a library so different transfer controllers can all reuse it).
4. Wait for source PVC to have no pods referencing it.
5. Do additional validation (e.g. maybe StorageClass has a new AllowTransfer that must be set to true).
6. Update the PVC object to indicate it is no longer "available" to use.
7. Update NamespaceTransferRequest.status to indicate that transfer has started.
8. Initiate rebind such that PV is unbound from source PVC and rebound to destination PVC (devil will be in the details here on how to do this safely).


### User Stories [optional]

#### Story 1

1. User in namespace-a  has a 250G volume that includes data for their database.
2. User would like to run the latest dataset in their testing database that is in a different namespace (namespace-b).
3. User creates a `NamespaceTransferRequest` in the namespace they wish to transfer the volume to (`namespace-b`).
4. User creates a `NamespaceTransferApproval` in the volumes current namespace (`namespace-a`).
5. User Creates a new PVC object with DataSource pointing to the `NamespaceTransferRequest`.
6. Wait for `NamespaceTransferRequest.status` to indicate transfer is complete.
7. Delete the (now unbound) claim in the source namespace (or they could be left but marked as transferred and not usable).
8. Use the new claim in the destination namespace.

NOTE that steps 5, 7 and 8 are implementation specific details to illustrate how this API could be used.  They
are not requirements and are subject to modification when proposing the implementation for volumes.


### Implementation Details/Notes/Constraints [optional]

This implementation deals only with the introduction of an API.  The goal is to provide an agreed upon generic API that objects can choose to implement if they find the concept useful.  The API will be implemented as an external CRD that operators could then choose to deploy if desired.

The actual implementations are a separate topic, however it is likely that implementations for core objects would require implementations in the core (in tree) controllers.  These changes must be such that they do not impact any existing usability or functionality in cases where a deployer does not deploy the namespace transfer CRDs.

### Risks and Mitigations

## Design Details

### Test Plan

**Note:** *Section not required until targeted at a release.*

Testing of the API will be dependent upon workign implementations.

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Completed design of the API
- Minimum of one implementation that is also ready to transition from Alpha to Beta (ensure the API is sufficient)
- Gather feedback from developers and surveys

##### Beta -> GA Graduation

- Positive feedback from usage in production
- Minimum of two implementations using the API, all implementations should be at least Beta


##### Removing a deprecated flag

This proposal is an external API, and as a result does not require a featuregate.

### Upgrade / Downgrade Strategy

### Version Skew Strategy

## Implementation History

## Drawbacks [optional]


## Alternatives [optional]


## Infrastructure Needed [optional]

