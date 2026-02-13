# KEP-1979: Object Storage Support

<!-- update with hack/update-toc.sh >
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
    - [Functionality](#functionality)
    - [System Properties](#system-properties)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Personas](#user-personas)
- [Design details](#design-details)
  - [Important changes between versions](#important-changes-between-versions)
    - [v1alpha1 to v1alpha2](#v1alpha1-to-v1alpha2)
  - [COSI Architecture](#cosi-architecture)
  - [COSI API Overview](#cosi-api-overview)
  - [COSI Object Lifecycle](#cosi-object-lifecycle)
  - [Usability](#usability)
    - [User Self-Service](#user-self-service)
    - [Mutating Buckets](#mutating-buckets)
    - [Sharing Buckets across Namespaces](#sharing-buckets-across-namespaces)
  - [Controller overview](#controller-overview)
  - [Control Flows](#control-flows)
    - [Installing the COSI System](#installing-the-cosi-system)
    - [Creating a Bucket](#creating-a-bucket)
    - [Accessing an Existing OSP Bucket](#accessing-an-existing-osp-bucket)
    - [Deleting a Bucket](#deleting-a-bucket)
    - [Generating Bucket Access Credentials](#generating-bucket-access-credentials)
    - [Deleting a BucketAccess](#deleting-a-bucketaccess)
    - [Attaching Bucket Information to Pods](#attaching-bucket-information-to-pods)
  - [COSI API Reference](#cosi-api-reference)
    - [Annotations and finalizers](#annotations-and-finalizers)
    - [Conditions](#conditions)
    - [Bucket](#bucket)
    - [BucketClaim](#bucketclaim)
    - [BucketClass](#bucketclass)
    - [BucketAccess](#bucketaccess)
    - [BucketAccessClass](#bucketaccessclass)
    - [BucketAccess Secret data](#bucketaccess-secret-data)
      - [S3](#s3)
      - [Azure](#azure)
      - [GCS (Google Cloud Storage)](#gcs-google-cloud-storage)
  - [COSI Driver](#cosi-driver)
    - [COSI Driver gRPC API](#cosi-driver-grpc-api)
      - [DriverGetInfo](#drivergetinfo)
      - [DriverGenerateBucketId](#drivergeneratebucketid)
      - [DriverCreateBucket](#drivercreatebucket)
      - [DriverGetBucket](#drivergetbucket)
      - [DriverDeleteBucket](#driverdeletebucket)
      - [DriverGenerateBucketAccessId](#drivergeneratebucketaccessid)
      - [DriverGrantBucketAccess](#drivergrantbucketaccess)
      - [DriverRevokeBucketAccess](#driverrevokebucketaccess)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Alternatives Considered](#alternatives-considered)
  - [Automatically mount buckets to Pods](#automatically-mount-buckets-to-pods)
  - [Encode BucketAccess connection information in a JSON blob](#encode-bucketaccess-connection-information-in-a-json-blob)
  - [Cross-resource protection finalizers](#cross-resource-protection-finalizers)
  - [Bucket creation annotation](#bucket-creation-annotation)
  - [BucketClass field on Bucket resource](#bucketclass-field-on-bucket-resource)
  - [Updating BucketAccess Secrets](#updating-bucketaccess-secrets)
  - [BucketAccess static provisioning](#bucketaccess-static-provisioning)
  - [BucketAccess Read/Write AccessMode](#bucketaccess-readwrite-accessmode)
  - [Multi-bucket BucketAccess considerations](#multi-bucket-bucketaccess-considerations)
    - [User consumption considerations](#user-consumption-considerations)
    - [Handling systems that cannot support multi-bucket BucketAccesses](#handling-systems-that-cannot-support-multi-bucket-bucketaccesses)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
- [Implementation History](#implementation-history)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [X] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    - Not applicable because COSI is not in-tree Kubernetes.
- [X] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [X] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
  - COSI is not in-tree Kubernetes, but doc website is here: https://container-object-storage-interface.sigs.k8s.io
- [X] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes
  - All necessary details are captured in this KEP

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal introduces the *Container Object Storage Interface* (COSI), a standard for provisioning and consuming object storage in Kubernetes.

## Motivation

File and block storage are treated as first class citizens in the Kubernetes ecosystem via CSI. Workloads using CSI volumes enjoy the benefits of portability across vendors and across Kubernetes clusters without the need to change application manifests. **An equivalent standard does not exist for Object storage.**

Object storage has been rising in popularity in the recent years as an alternative form of storage to filesystems and block devices. Object storage paradigm promotes disaggregation of compute and storage. This is done by making data available over the network, rather than locally. Disaggregated architectures allow compute workloads to be stateless, which consequently makes them easier to manage, scale and automate.

### Goals

#### Functionality

- Support automated **Bucket Creation**
- Support automated **Bucket Deletion**
- Support automated **Access Credential Generation**
- Support automated **Access Credential Revocation**
- Support a standardized interface for workloads Pods to access buckets
- Support **Bucket Reuse** to use existing buckets via COSI (static provisioning)

#### System Properties

- Support **workload portability** across clusters
- Achieve all goals in a **vendor neutral** manner
- Standardize the mechanism for third-party vendors to integrate easily with COSI
- Allow users (non-admins) to create and utilize buckets (self-service)
- Establish best-in-class **Access Control** practices for bucket creation and sharing

### Non-Goals

- **Data Plane** API standardization will not be addressed by this project
- **Bucket Mutation** is not supported as of v1alpha2

## Proposal

### User Personas

We define 3 kinds of stakeholders:

- **Administrators** (a.k.a., admin)
  - Administrators of the Kubernetes cluster's COSI functionality
  - Deploy, configure, and manage COSI vendor drivers in the Kubernetes cluster
  - Establish cluster-wide policies for access control over COSI resources

- **Application developers** (a.k.a., DevOps, user)
  - End-users of COSI resources claim and use object storage buckets for their applications
  - Request a bucket to be created and/or provisioned for workloads
  - Request access to be created/deleted for a specific bucket
  - Request deletion of a created bucket

- **Object Storage Providers** (a.k.a., OSP, vendor)
  - Vendors who offer object storage capabilities for any given object storage system
  - Comparable to a storage vendor in the CSI domain
  - Create a COSI driver for their object storage system following the COSI API specs

## Design details

### Important changes between versions

#### v1alpha1 to v1alpha2

- DeletionPolicy is now a required field on all API objects. The user must explicitly specify it, leaving no room for confusion.

- DeletionPolicy behavior was clarified. This field controls retention/deletion of both the Bucket object and the underlying OSP bucket. This behavior mirrors the Volume Snapshot KEP, which is considered to model the latest sig-storage best-practices.

- DeletionPolicy is now a mutable field on Bucket objects. BucketClass (like StorageClass) is still immutable.

- Sidecar has a reduced set of permissions. It now only reconciles Buckets and BucketAccesses and no longer reads Bucket(Access)Class or BucketClaim. Keeping Sidecar operations limited in scope helps avoid version skew incompatibility issues between Controller and Sidecar. It also means that COSI Controller updates are more likely to be able to resolve COSI bugs without driver deployments needing to be updated. BucketAccess provisioning is required to be more complex to allow for the Controller to do initial processing and then hand off provisioning to the Sidecar.

- To support a sidecar with reduced permissions, BucketAccesses now have several status fields that are set by the Controller based on content from the referenced BucketClaim and provision-time BucketAccessClass details.

- BucketClassName was removed from Bucket spec. All BucketClass configurations and parameters are expected to be copied to the Bucket at provision-time to ensure BucketClass mutation (deletion and subsequent re-creation with different configuration) over time does not affect ongoing reconciles of Bucket objects, rendering BucketClassName irrelevant.

- Annotations are used to track cross-resource references, and COSI manages object cleanup (or cleanup blocking) up based on annotation states. This cross-resource cleanup management was previously done with finalizers instead of annotations. A single finalizer per-resource is still used to prevent resource deletion before backend cleanup. This comes from a recommendation by sig-storage leadership that finalizers are often removed by users and therefore don't offer much system protection/stability in the real world. These may be added back in the future if needed.

- The BucketAccess secret now has individually-named fields and no longer holds a JSON blob containing connection/authentication details. Several users reported challenges or the inability to configure their applications using the JSON blob and strongly prefer individual fields that can be loaded into environment variables or files as desired. COSI chose to not support both JSON blob and individual fields to keep the implementation simpler.

- A new Sidecar gRPC call was added for validating statically-provisioned Buckets have a viable OSP backend bucket.

- To avoid create/delete races that can leave leaked backend resources orphaned, change to a 2-phase provisioning flow. In phase 1, request a persistent, reusable bucket ID (or access ID). In phase 2, after COSI persists the returned ID, request backend resource creation.

- BucketClaim status `BucketName` was changed to `BoundBucketName` for API clarity.

- The BucketAccess secret no longer includes the name of the Bucket resource it is associated with. It is not necessary for end-user Pods to know the name of the Bucket resource. For the S3 protocol, the Pod may need to know the OSP backend bucket ID, which is included as part of the S3 info.

- BucketAccess `Protocol` is now required. This requires users to state their intentions/needs upfront, which allows COSI to ensure compatibility with what can be provisioned.

- `DriverGetInfo` gRPC call now expects drivers to return (advertise) the list of supported protocols. COSI will check that Bucket/Claim requested Protocols match what is advertised.

- BucketAccess(Class) AuthenticationType values have been renamed:
  - `KEY` -\> `Key` - Key is not an acronym, so the value was changed to be capitalized based on Kubernetes API best practices.
  - `IAM` -\> `ServiceAccount` - IAM may or may not be used by the OSP, and this detail is irrelevant to COSI itself. What is relevant is that the driver ties OSP authentication to a user's chosen Kubernetes ServiceAccount.

- A BucketAccess can now reference (request access for) multiple BucketClaims. This has been one of the most requested features for COSI after v1alpha1.

- A BucketAccess can now specify the desired Read/Write access mode for each referenced BucketClaim. This is another highly requested feature for COSI after v1alpha1. Permissions are separated into 3 categories: object data, object metadata, bucket metadata.

- Resource statuses will use Conditions to report `Provisioned` status instead of `ReadyToUse` boolean. Conditions will also report driver RPC errors and resource spec/status errors to help with user debugging.

### COSI Architecture

Since this is an entirely new feature, it is possible to implement this completely out of tree.
COSI is modeled after Kubernetes PV/PVCs and CSI as well as the CSI Volume Snapshot feature.

The following components are proposed for this architecture:

- COSI Controller
- COSI Sidecar
- COSI Driver

1. The COSI Controller is the central controller that validates, authorizes, and binds COSI-created resources to their corresponding claims.
   Only one active instance of Controller should be present.
2. The COSI Sidecar is the point of integration between COSI and each driver.
   Each operation that requires communication with the OSP is triggered by the Sidecar using gRPC calls to the driver.
   One active instance of Sidecar should be present **for each driver**.
3. The COSI driver communicates with the OSP to fulfill gRPC requests from the Sidecar.

The [COSI driver](#cosi-driver) section provides more detail about vendor driver requirements.

### COSI API Overview

COSI defines these new API types:

- [Bucket](#bucket) - Analogous to Persistent Volume
- [BucketClaim](#bucketclaim) - Analogous to Persistent Volume Claim
- [BucketAccess](#bucketaccess) - Requests credentials that allow a user to consume a bucket. Analogous to PV/PVC mounting functionality
- [BucketClass](#bucketclass) - Analogous to StorageClass, for BucketClaims
- [BucketAccessClass](#bucketaccessclass) - Also analogous to StorageClass, for BucketAccesses

### COSI Object Lifecycle

The following resources are managed by Admins. All are cluster-scoped.

- BucketClass
- BucketAccessClass
- Bucket - in case of a bucket that already exists in OSP backend being (static provisioning)

The following resources are managed by a User. All are namespace-scoped.
Each is created with a reference to a corresponding class object.

- BucketClaim -\> BucketClass
- BucketAccess -\> BucketClaim, BucketAccessClass

For dynamically-provisioned BucketClaim resources created by a User, the COSI controller responds by creating an intermediate Bucket object as shown.

- BucketClaim -\> new(Bucket)

Notes:

- There are **NO** cycles in the relationship graph of the above mentioned API objects.
- Mutations are not supported in the for any COSI object except when noted otherwise.
- Class objects have a lifecycle independent of objects that reference them.
  - BucketClaim, BucketAccess, and Bucket must have all necessary class parameters copied to them during provisioning to allow themselves to be deleted if class objects have been mutated/deleted.

### Usability

#### User Self-Service

User self-service is made possible using BucketClaim and BucketAccess resources (namespace-scoped).
Users do not require admin privileges to create, modify, and/or delete them.

An admin is responsible for creating class objects (BucketClass, BucketAccessClass) which configure OSP-specific storage parameters.
The creation of COSI class objects is deliberately analogous to creation and management of Kubernetes StorageClasses for PVCs.
This is a well-understood pattern, and relying on familiarity will aid COSI users.

Importing a bucket that already exists in an OSP backend (a statically-provisioned brownfield bucket) requires special permissions because its lifecycle is not managed by COSI.
Special care needs to be taken to prevent unintended clones, accidental deletion, and other mishaps that could affect the OSP bucket.
For instance, setting the deletion policy to Delete for a brownfield bucket should be disallowed.
Admins are thus responsible for creating Bucket resources for brownfield buckets.

#### Mutating Buckets

COSI v1alpha2 does not support mutating bucket/access properties except where noted otherwise.
However, the current design does not prevent future support for property mutation.
Property mutation is intended to be supported in future versions, possibly by a mechanism analogous to PVCs's Volume Attribute Classes.

#### Sharing Buckets across Namespaces

COSI v1alpha2 does not support associating a Bucket in one namespace with a BucketClaim in another namespace.
i.e. no automatic cross-namespace bucket sharing is possible.
In future versions, a namespace-level access control will be enforced.
Buckets may be constrained to particular namespaces, possibly using namespace selectors.
Admins will be able to control which namespaces can access which buckets, possibly using namespace selectors.
In the current implementation, an Admin can create a new static reference to an existing Bucket to allow it to be consumed from an additional namespace.

### Controller overview

COSI is split into two controllers: the main COSI Controller and a driver Sidecar. This mirrors CSI's system design.

The Controller watches for all COSI resource CRD kinds.
The Sidecar watches for the minimal set of CRD kinds in an effort to keep Sidecar implementation complexity low and to limit the effects of version skew between Controller and Sidecar.

Minimal Sidecar permissions:
- Read Bucket spec; read/write Bucket metadata, status, and finalizers
- Read BucketAccess spec; read/write BucketAccess metadata, status, and finalizers
- No permissions for BucketClaim, BucketClass, BucketAccessClass
- Create/Update/Patch/Delete Secrets in any namespace, for managing BucketAccess Secrets
- No delete permissions for other resources (including Bucket/BucketAccess)

For dynamic provisioning, a user creates a new BucketClaim object referencing a BucketClass object corresponding to a driver.
This causes the COSI Controller to trigger creation of a Bucket object to represent the to-be-created OSP bucket.

The creation of a new Bucket object causes the Sidecar to provision a new bucket in the OSP driver.
When the OSP bucket is successfully provisioned, the Sidecar updates the Bucket status to represent the new OSP bucket.

The COSI Controller will update the status field of the BucketClaim object accordingly based on the status field of the Bucket object to indicate the bucket is ready to be used or failed.

When a BucketClaim object is deleted, the COSI Controller adds an annotation to the Bucket object to indicate that the parent claim is being deleted.
This is an indication to the Sidecar that it's safe to unbind the Bucket from the BucketClaim.
The Controller then sets a deletion timestamp on the Bucket object.

If the deletion policy indicates that the data should be deleted, the Sidecar will call the OSP driver to delete the bucket.
If deletion succeeds, the Sidecar will allow the Bucket object to be deleted by deleting the CRD finalizer.

If the deletion policy indicates that data should be retained, the BucketClaim will be deleted while the corresponding Bucket resource (as well as the underlying OSP bucket) will be retained.

### Control Flows

This section outlines the scenarios that COSI personas will initiate, and for what purpose.
Each scenario includes enough detail to express the important interaction requirements between personas and the COSI system, and between COSI components.
This section avoids unnecessarily naming specific API elements so as not to confuse complex system interaction requirements with specific implementation/spec details.

#### Installing the COSI System

Admin installs the COSI system and driver(s) to allow User self-service.

1. Assume that a Vendor has already created a COSI driver
2. Admin deploys the COSI controller
3. Admin deploys vendor COSI driver
4. Admin creates BucketClass and BucketAccessClass configuring COSI and vendor driver features

#### Creating a Bucket

User self-provisions a bucket to store their workload's data.

The fundamental key to this design is the bi-directional "pointer" between Bucket and BucketClaim, which is represented in the claim status and bucket spec.
The bi-directionality is complicated to manage in a transactionless system, but without it we can't ensure sane behavior in the face of different forms of trouble.
For example, a rogue HA controller instance could end up racing and making multiple bindings that are indistinguishable, resulting in potential data loss.

1. User creates BucketClaim that uses BucketClass
2. COSI controller observes BucketClaim
   1. Controller applies `objectstorage.k8s.io/protection` finalizer to the BucketClaim
   2. Controller looks up bound bucket name from BucketClaim status
   3. If bound bucket name is empty string, generate a bound bucket name: `bc-`+`<BucketClaim.UID>`
   4. Controller looks up corresponding Bucket
   5. If Bucket does not exist and bound bucket name is non-empty, error out
      1. This indicates a BucketClaim whose once-existing Bucket has gone missing
   6. If Bucket does not exist and bound name is empty, Controller creates intermediate Bucket resource with these details:
      1. BucketClass parameters are copied to Bucket (needed for deletion/modification)
      2. Full BucketClaim reference info (with UID) is set on intermediate Bucket spec (Bucket is bound to claim)
   7. Update BucketClaim status with bound bucket name
   8. Controller waits for the intermediate Bucket to be reconciled by COSI sidecar
3. COSI Sidecar detects intermediate Bucket resource
   1. If the Bucket's driver matches the sidecar's driver, continue
   2. Sidecar applies `objectstorage.k8s.io/protection` finalizer to intermediate Bucket
   3. Sidecar calls the COSI driver via gRPC to generate a bucket ID
   4. COSI persists the returned bucket ID in Bucket status
   5. Sidecar calls the COSI driver via gRPC to provision the OSP bucket, by prior bucket ID
   6. If OSP returns provision fail, COSI sidecar reports error to Bucket status and retries w/ backoff
4. When OSP returns provision success, COSI sidecar updates Bucket status `ReadyToUse` to true
5. Controller detects that the Bucket is provisioned successfully (`ReadyToUse`==true)
   1. Controller finishes BucketClaim reconciliation processing
   2. Controller validates BucketClaim and Bucket fields to ensure provisioning success
   3. Controller copies Bucket status items to BucketClaim status as needed. Importantly:
      1. Supported protocols
      2. `ReadyToUse`

#### Accessing an Existing OSP Bucket

User needs access to a bucket that already exists in an OSP object store.

This is the static provisioning scenario.
This can be used to migrate "brownfield" buckets that pre-existed COSI installation in a Kuberentes cluster.

In early COSI feedback and in other object storage self-service frameworks, users commonly want access to OSP buckets that are preexisting.
However, giving end users unrestricted access to OSP storage would allow them to easily gain access to sensitive data they may not be intended to access.
To resolve this, only the Admin is expected to allow access to existing OSP buckets.

1. Admin creates a Bucket object that represents the existing OSP bucket
   1. Admin must specify the existing OSP bucket ID in the Bucket spec
   2. Admin should specify driver parameters (normally copied from BucketClass) for all parameters needed for driver functionality
   3. Admin must ensure that the Bucket binds only to a specific BucketClaim by specifying the BucketClaim parent reference by namespace and name
2. COSI sidecar detects the Bucket resource
   1. If the Bucket's driver matches the sidecar's driver, continue
   2. Sidecar applies `objectstorage.k8s.io/protection` finalizer to Bucket
   3. Sidecar calls the COSI driver via gRPC call to check that the existing OSP bucket exists
   4. Sidecar exits with retry backoff if existing bucket is nonexistent
   5. When Bucket prep is successful, COSI sidecar updates Bucket status `ReadyToUse` to true
      (otherwise retry each time Bucket is updated)
3. User (or Admin) creates BucketClaim referencing Bucket above
4. COSI controller observes BucketClaim
   1. Controller validates BucketClaim fields
   2. Controller looks up corresponding Bucket - if DNE, retry when Bucket is created
   3. Set BucketClaim status to point to Bucket (claim is now bound to Bucket)
   4. Controller applies `objectstorage.k8s.io/protection` finalizer to BucketClaim
   5. If BucketClaim reference set by admin on Bucket doesn't match, error without retry
   6. Apply Full BucketClaim reference info (with UID) to Bucket spec (Bucket is now bound to claim)
   7. If Bucket status `ReadyToUse` is not true, wait for Bucket to be updated
   8. Controller validates BucketClaim and Bucket fields to ensure provisioning success
   9. Controller copies Bucket status items to BucketClaim status as needed. Importantly:
      1. Supported protocols
      2. `ReadyToUse`

#### Deleting a Bucket

User deletes a BucketClaim they no longer need.

Users cannot delete global-scope Buckets directly.
Instead, a User deletes a BucketClaim they have delete permissions for, and COSI coordinates deleting the Bucket.

Each Bucket has a deletion policy that determines whether COSI deletes the Bucket resource when the BucketClaim it's bound to is deleted.
Deletion policy options:
- `Delete`: Bucket and underlying OSP bucket are deleted as part of BucketClaim deletion process
- `Retain`: Bucket is unbound, but the Bucket resource and underlying OSP bucket are kept

An Admin can delete any Bucket resource at any time.
When a Bucket is deleted, COSI should prevent the Bucket from being deleted until the BucketClaim it is bound to is also in deleting state.

BucketClaims having valid BucketAccesses (i.e., claims in use) will not be deleted until all the BucketAccesses are cleaned up.

When a BucketClaim with Bucket reclaim policy `Reclaim` is deleted, the Bucket is left in place when the BucketClaim is removed.
COSI does not support the Bucket being automatically re-bound to a new BucketClaim without Admin intervention.
Admins must follow the [static provisioning workflow](#accessing-an-existing-osp-bucket) to allow re-binding.

1. User deletes BucketClaim object
2. COSI Controller detects BucketClaim resource's deletion timestamp
   1. If `objectstorage.k8s.io/has-bucketaccess-references` annotation exists on the BucketClaim, exit without retry
   2. Controller looks up the corresponding Bucket
      1. If Bucket doesn't exist, go to **CLEANUP** (Bucket was deleted or never existed)
      2. If Bucket-BucketClaim binding is not valid, error out
      3. Apply `objectstorage.k8.io/bucketclaim-being-deleted` annotation to Bucket
         (tells Sidecar that it's safe to proceed with Bucket deprovisioning)
      4. If Bucket deletion policy is `Delete`, add deletion timestamp to Bucket
      5. If `Retain`, nothing more for Controller to do
      6. **CLEANUP**: Controller removes BucketClaim `objectstorage.k8s.io/protection` finalizer
3. COSI Sidecar detects Bucket update
   1. If the Bucket's driver matches the sidecar's driver, continue
   2. If `objectstorage.k8.io/bucketclaim-being-deleted` annotation, continue
   3. If reclaim policy is `Delete`
      1. If Bucket has nil deletion timestamp, exit (do not deprovision without deletion timestamp)
      2. Sidecar calls the COSI driver via gRPC to de-provision the OSP bucket
      3. If OSP returns provision fail, Sidecar reports error to Bucket status and retries gRPC call
      4. When OSP returns provision success, remove Bucket `objectstorage.k8s.io/protection` finalizer
   4. If deletion policy is `Retain`, nothing more to do

COSI Sidecar should not have Bucket delete permissions.

#### Generating Bucket Access Credentials

User requests access to a BucketClaim for their workload's application.

Access credentials are represented by BucketAccess objects.
The separation of BucketClaim and BucketAccess is a reflection of the usage pattern of object storage.
Object storage is always accessed over the network, and all access is subject to authentication and authorization.
The lifecycle of a bucket and its access are not tightly coupled.

If a BucketClaim is in deleting state, no new BucketAccesses can be created for it.

1. User creates BucketAccess that uses BucketAccessClass and references one or more BucketClaims
   1. For each BucketClaim referenced, user specifies a Kubernetes Secret name into which BucketAccess information will be stored upon successful provisioning
2. COSI Sidecar detects the BucketAccess resource
   1. Initially, corresponding Bucket in BucketAccess status is unknown, so sidecar exits with no action
3. COSI Controller detects the BucketAccess resource
   1. Controller looks up corresponding BucketAccessClass
   2. If the BucketAccessClass is configured to prevent the BucketAccess from being provisioned as specified, exit without retry
   3. Controller looks up corresponding BucketClaim(s)
   4. Controller sets `objectstorage.k8s.io/protection` finalizer on BucketAccess
   5. Controller sets `objectstorage.k8s.io/has-bucketaccess-references` annotation on corresponding BucketClaim(s)
      (block claim from being deleted until access is deleted)
   6. If any BucketClaim-Bucket binding is not valid, retry when BucketClaim updated
      1. Bucket does not have to be provisioned, but Bucket must be known
   7. If any BucketClaims are being deleted, return an error
   8. Once everything looks good on Bucket+Claim(s):
      1. Set corresponding Bucket references on BucketAccess status
      2. Copy BucketAccessClass specs and parameters to BucketAccess status
4. COSI Sidecar detects the BucketAccess resource update
   1. BucketAccess status now shows corresponding Bucket(s) BucketAccessClass info, so sidecar can provision
   2. If the BucketAccess's driver matches the sidecar's driver, continue
   3. Sidecar applies `objectstorage.k8s.io/protection` finalizer to the BucketAccess if needed
   4. Sidecar looks up the Bucket(s) to get necessary info
   5. If any Bucket has `objectstorage.k8.io/bucketclaim-being-deleted` annotation or deletion timestamp, error without retry
      (this indicates one or more claims are being deleted, possibly race condition missed in Controller)
   6. Sidecar calls the COSI driver via gRPC to generate an account ID for the access
   7. Sidecar persists the account ID in the BucketAccess status
   8. Sidecar calls the COSI driver via gRPC to generate unique access credentials for the Bucket(s), by prior account ID
   9. If OSP returns provision fail, Sidecar reports error to BucketAccess status and retries gRPC call
   10. When OSP returns provision success, COSI sidecar:
      1. Applies `objectstorage.k8s.io/protection` finalizer to all Secrets
      2. Updates all BucketAccess Secrets with all info needed to access each OSP bucket
      3. Updates BucketAccess status `ReadyToUse` to true

#### Deleting a BucketAccess

User deletes a BucketAccess they no longer need.

COSI does not set up or manage mounting BucketAccess information to Pods consuming the BucketAccess.
As such, COSI will delete a BucketAccess and its associated Secret without checking if the Secret is mounted to any Pods.

1. User deletes BucketAccess object
2. COSI Controller detects BucketAccess resource's deletion timestamp
   1. Initially, Controller does nothing, waiting for Sidecar to set `objectstorage.k8s.io/sidecar-cleanup-finished` annotation
3. COSI Sidecar detects BucketAccess resource's deletion timestamp
   1. Sidecar removes `objectstorage.k8s.io/protection` finalizer from the BucketAccess Secret
   2. Sidecar deletes the BucketAccess Secret (should happen before OSP access is removed via gRPC)
   3. Sidecar calls the COSI driver via gRPC to revoke the associated access credentials
   4. If OSP returns de-provision fail, COSI sidecar reports error to BucketAccess status and retries gRPC call
   5. When OSP returns de-provision success, COSI sidecar:
      1. Sets `objectstorage.k8s.io/sidecar-cleanup-finished` annotation on BucketAccess
4. Controller detects BucketAccess resource update, with deletion timestamp
   1. Controller removes `objectstorage.k8s.io/has-bucketaccess-references` from BucketClaim if this is the last BucketAccess against the BucketClaim (this allows BucketClaim to start deletion, if applicable)
   2. Controller removes `objectstorage.k8s.io/protection` from BucketAccess

#### Attaching Bucket Information to Pods

User attaches bucket information to their workload's application pod.

1. User references the BucketAccess secret using the pod volume downward API in their pod spec
2. User configures pod container(s) to mount secret data items as desired

See [BucketAccess Secret data](#bucketaccess-secret-data) section for more details.

The BucketAccess secret can be provided to the pod using any Kubernetes {Secret -> Pod} attachment mechanism.
This naturally includes mounting data into environment variables and files.
Mounting credential data into files is slightly more secure than environment variables and is thus recommended.
However, each application has different requirements, and some may require environment variables for configuring access.

### COSI API Reference

#### Annotations and finalizers

Annotations:
- `objectstorage.k8s.io/bucketclaim-being-deleted`: applied to a Bucket when the Controller detects that the Bucket's bound BucketClaim is being deleted
- `objectstorage.k8s.io/has-bucketaccess-references`: applied to a BucketClaim when the Controller detects that one or more BucketAccesses reference the claim
- `objectstorage.k8s.io/sidecar-cleanup-finished`: applied to a BucketAccess when the Sidecar has finished cleaning up, allowing the Controller to begin its final cleanup operations
- `objectstorage.k8s.io/bucketaccess-reference`: applied to BucketAccess Secrets with the value `<namespace>/<name>` of the BucketAccess the Secret references. This is not functional and exists simply to assist users with cross-referencing when inspecting a Secret.
- `objectstorage.k8s.io/bucketclaim-reference`: applied to BucketAccess Secrets with the value `<namespace>/<name>` of the BucketClaim the Secret references. This is not functional and exists simply to assist users with cross-referencing when inspecting a Secret.

Finalizers:
- `objectstorage.k8s.io/protection`: applied to BucketClaims, Buckets, BucketAccesses, and BucketAccess Secrets to prevent them from being deleted until COSI has cleaned up intermediate and underlying resources

#### Conditions

COSI defines the following conditions for its resource statuses.
Behavior and requirements shared by COSI multiple resources is noted here.
Not all conditions are used for all resources.

- `Provisioned` indicates whether the backend resource (bucket/access) was provisioned successfully by the driver.
  - `Unknown` initially.
  - `True` when a driver successfully provisions a resource.
  - `False` when COSI determines that the resource can't be provisioned (or is definitively and permanently lost).
  - Because an RPC call to a driver may occasionally fail when the resource is still present in the backend,
    this value is not transitioned from `True` to another state unless there is a definite, permanent change/loss.
  - For issues after initial provisioning (this=`True`), an admin (or user) will need to use other conditions below
    to determine what action(s) are needed to resolve an issue.

- `ResourcesValidated` indicates whether the Kubernetes resource(s) are valid.
  - `Unknown` initially.
  - `Unknown` while COSI is waiting on referenced resources to exist.
  - `True` when the resource (and its referenced resource(s)) are valid.
  - `False` when a resource (or its referenced resource(s)) are invalid or become degraded.
  - This condition is needed to provide feedback about resource degradation that might happen after `Provisioned` is successful.
  - A degraded resource may have lost info that COSI needs to initiate further RPC calls.
  - E.g., a referenced resource goes missing.

- `ProvisionFailed` indicates when driver provisioning fails.
  - `Unknown` initially.
  - `False` when driver provisioning returns OK via RPC.
  - `True` when driver provisioning returns non-OK via RPC.
  - This condition is needed to provide feedback about driver or backend issues that might happen after `Provisioned` is successful.
  - E.g., this may help an admin diagnose a temporary DNS issue or a permanent backend data loss that the driver couldn't programmatically determine.

Note: Kubernetes API issues (e.g., update conflicts, rate limiting) are reported as Events.
They aren't as important to capture permanently as conditions.

#### Bucket

Resource to represent a Bucket in an OSP. Bucket is cluster-scoped.

```go
Bucket {
  TypeMeta
  ObjectMeta

  Spec BucketSpec {
    // DriverName is the name of driver associated with this bucket.
    // +required
    DriverName string

    // DeletionPolicy determines whether a Bucket created through the BucketClass should be deleted
    // when its bound BucketClaim is deleted.
    // Possible values:
    //  - Retain: keep the Bucket object as well as the bucket in the underlying storage system
    //  - Delete: delete the Bucket object as well as the bucket in the underlying storage system
    // +required
    DeletionPolicy DeletionPolicy

    // Protocols is a list of protocols that the provisioned Bucket must support. COSI will verify
    // that each item in this list is advertised as supported by the OSP driver.
    Protocols []Protocol

    // Parameters is an opaque map for passing in configuration to a driver for creating the bucket.
    // +optional
    Parameters map[string]string

    // References the BucketClaim that resulted in the creation of this Bucket.
    // For statically-provisioned buckets, set the namespace and name of the BucketClaim that is
    // allowed to bind to this Bucket.
    BucketClaimRef BucketClaimReference

    // ExistingBucketID is the unique id of the bucket in the OSP. This field should be used to
    // specify a bucket that has been statically provisioned.
    // This field will be empty when the Bucket is dynamically provisioned by COSI.
    // +optional
    ExistingBucketID string
  }

  Status BucketStatus {
    // Conditions - described below
    Conditions []metav1.Condition

    // BucketID is the unique ID of the bucket in the OSP. This field will be populated by COSI once
    // the ID in the OSP is known.
    BucketID string

    // Protocols is the set of protocols the provisioned Bucket supports. BucketAccesses can request
    // to access this BucketClaim using any of the values given here.
    // Possible values: S3, Azure, GCS
    Protocols []Protocol

    // BucketInfo reported by the driver, rendered in the COSI_<PROTOCOL>_<KEY> format used for the
    // BucketAccess Secret. e.g., COSI_S3_ENDPOINT, COSI_AZURE_STORAGE_ACCOUNT
    // This is opaque and should not contain any sensitive data.
    BucketInfo map[string]string
  }
}

// All info needed to identify a BucketClaim
BucketClaimReference {
  Name      string
  Namespace string
  UID       types.UID
}
```

`BucketInfo` is provided as a means of allowing Administrators more insight into how OSP drivers have provisioned Buckets.
This may be important for debugging.
This information is not copied to the BucketClaim status so that it is not visible to end Users.
For user familiarity, the info is rendered in the [BucketAccess Secret Data](#bucketaccess-secret-data) format.

Once created, a Bucket object is immutable, except for fields specifically noted:
- `DeletionPolicy` is mutable to allow Admins to change to `Retain` policy after creation.

Conditions:
- `Provisioned` - Indicates the backend bucket was created and should exist (or not).
  - `Unknown` if provisioning fails with retryable resource or RPC error.
  - `False` if provisioning fails with a non-retryable RPC error or non-recoverable resource error.
  - `True` when RPC call for bucket create/getinfo returns OK.
- `ResourcesValidated` - Indicates Bucket spec validity.
  - E.g., Bucket spec becomes degraded after provisioning (though this is unlikely).
- `ProvisionFailed` - Records results of latest RPC bucket create/getinfo.
  - E.g., driver auth expired (easily resolvable), or backend data was lost (requires restoring from backup).

#### BucketClaim

A claim to create Bucket. BucketClaim is namespace-scoped.

```go
BucketClaim {
  TypeMeta
  ObjectMeta

  Spec BucketClaimSpec {
    // Name of the BucketClass.
    BucketClassName string

    // Protocols is a list of protocols that the provisioned Bucket must support. COSI will verify
    // that each item in this list is advertised as supported by the OSP driver.
    Protocols []Protocol

    // Name of a bucket object that was manually created to import a bucket created outside of COSI.
    // If unspecified, then a new Bucket will be dynamically provisioned.
    // +optional
    ExistingBucketName string
  }

  Status BucketClaimStatus {
    // Conditions - described below
    Conditions []metav1.Condition

    // BoundBucketName is the name of the provisioned Bucket in response to this BucketClaim. It is
    // generated and set by the COSI controller before making the creation request to the OSP backend.
    BoundBucketName string

    // Protocols is the set of protocols the provisioned Bucket supports. BucketAccesses can request
    // to access this BucketClaim using any of the values given here.
    // Possible values: S3, Azure, GCS
    Protocols []Protocol
  }
```

Conditions:
- `Provisioned` - Indicates the backend bucket was created and should exist (or not).
  - `Unknown` while waiting on corresponding Bucket to provision
  - `True`/`False`: Copied from the corresponding Bucket resource's `True`/`False` status (do not copy `Unknown` status).
- `ResourcesValidated` - Indicates validity of BucketClaim, referenced BucketClass, and corresponding Bucket.
  - `Unknown` when BucketClass is not present (waiting) while provisioning
  - `False` if BucketClass is invalid while provisioning
  - `False` when Claim is/becomes degraded
  - `False` when the corresponding Bucket reports `ResourcesValidated=False`
- `ProvisionFailed` - Mirrors what the corresponding Bucket status reports.

#### BucketClass

Resource for configuring common properties for multiple Buckets. BucketClass is cluster-scoped.

```go
BucketClass {
  TypeMeta
  ObjectMeta

  Spec BucketClassSpec {
    // DriverName is the name of driver associated with this bucket.
    // +required
    DriverName string

    // DeletionPolicy determines whether a Bucket created through the BucketClass should be deleted
    // when its bound BucketClaim is deleted.
    // Possible values:
    //  - Retain: keep the Bucket object as well as the bucket in the underlying storage system
    //  - Delete: delete the Bucket object as well as the bucket in the underlying storage system
    // +required
    DeletionPolicy DeletionPolicy

    // Parameters is an opaque map for passing in configuration to a driver for creating the bucket.
    // +optional
    Parameters map[string]string
  }
}
```

Conditions: None.

#### BucketAccess

The BucketAccess is used to request access to a bucket.
It contains fields for choosing BucketClaims for which the credentials will be generated.
It also includes a bucketAccessClassName field, which in-turn contains configuration for authorizing users to access buckets.

A resource to access a Bucket. BucketAccess is namespace-scoped.

For many OSP drivers, each BucketAccess will correspond with a unique OSP backend 'user' (e.g., an S3 user).
COSI suggests this as a starting point, but this is not a strict requirement.

Users can specify the read/write access mode they need for each referenced BucketClaim.
Read/write access modes are split into 3 categories:
- object data - the ability to read/write objects within the bucket
- object metadata - the ability to read/write metadata on objects within the bucket
- bucket metadata - the ability to read/write metadata on the bucket itself

Driver implementers should use `InvalidArgument` to report unsupported permission requests.
Drivers should include a clear message about what permissions are unsupported.

BucketAccess A referencing BucketClaim A should generally not be able to read or write to an un-referenced BucketClaim B.
COSI drivers may choose to have options for different behavior to support niche scenarios, but this should be the default assumption for BucketAccess provisioning.

COSI does not support static provisioning for BucketAccesses.
Portability is maintained because object storage accesses do not hold critical application data.
Any BucketClaim can have a new, valid BucketAccess created for it at any time to provide access to the data.
The [Alternatives considered section](#bucketaccess-static-provisioning) provides more details.

```go
// BucketAccessMode is the read/write mode a BucketAccess should have for a BucketClaim's data.
// Users can assume access also allows corresponding access to bucket and object metadata except
// metadata which would alter behavior of the storage backend.
// Supported values:
// - ReadWrite - read and write access allowed
// - WriteOnly - only write access allowed
// - ReadOnly - only read access allowed
BucketAccessMode string

// All access modes that can be requested for a bucket.
BucketAccessModes struct {
  // The ability to read/write objects within the bucket.
  // +optional
  ObjectData BucketAccessMode

  // The ability to read/write metadata on objects within the bucket.
  // +optional
  ObjectMetadata BucketAccessMode

  // The ability to read/write metadata on the bucket itself.
  // +optional
  BucketMetadata BucketAccessMode
}

BucketClaimReference {
  // The name of the BucketClaim this BucketAccess should have permissions for.
  // The BucketClaim must be in the same namespace as the BucketAccess.
  BucketClaimName string

  // AccessSecretName is the name of the Kubernetes secret that COSI should populate with access
  // details and credentials for the referenced BucketClaim.
  // This secret is deleted when the BucketAccess is deleted.
  // +required
  AccessSecretName string

  // AccessModes represents Read/Write access mode requests that this BucketAccess should have for
  // the BucketClaim. In the context of requested access modes, an empty mode means the user does
  // not require the permission and does not care if the corresponding access is allowed or not.
  // At least one permission must be requested.
  // +required
  AccessModes BucketAccessModes
}

AccessedBucket {
  // The name of the Bucket resource that this BucketAccess should have permissions for.
  BucketName

  // The unique ID of the bucket in the OSP that this BucketAccess should have permissions for.
  BucketID string

  // The name of the BucketClaim within spec.bucketClaims that this Bucket corresponds to.
  // This allows the Sidecar to cross-reference the Bucket with the claim's access parameters.
  BucketClaimName string
}

BucketAccess {
  TypeMeta
  ObjectMeta

  Spec BucketAccessSpec {
    // BucketClaims is the list of BucketClaims this access should have permissions for.
    // Multiple references to the same BucketClaim are not permitted.
    // +required
    BucketClaims []BucketClaimReference

    // BucketAccessClassName is the name of the BucketAccessClass.
    // +required
    BucketAccessClassName string

    // Protocol is the name of the Protocol that this access credential is expected to support.
    // +required
    Protocol Protocol

    // ServiceAccountName is the name of the Kubernetes ServiceAccount that user applications intend
    // to use for bucket access. Must be in the same Namespace as the BucketAccess.
    // This has different behavior based on the BucketAccessClass's defined AuthenticationType:
    // - Key - This field is ignored.
    // - ServiceAccount - This field is required. The driver should configure the system so that
    //   Pods using the ServiceAccount authenticate to the OSP backend automatically.
    // +optional
    ServiceAccountName string
  }

  Status BucketAccessStatus {
    // Conditions - described below
    Conditions []metav1.Condition

    // AccountID is the unique ID for the account in the OSP. It will be populated by the COSI
    // sidecar once access has been successfully granted.
    AccountID string

    // AccessedBuckets is the list of Buckets the access corresponds to, each with its related
    // access mode. This is filled in by the Controller based on the BucketClaim so that the Sidecar
    // knows what Buckets to allow access to for this BucketAccess.
    AccessedBuckets []AccessedBucket

    // DriverName holds a copy of the BucketAccessClass driver name at the time of BucketAccess
    // provisioning. This is kept to ensure the BucketAccess can be modified/deleted even after
    // BucketAccessClass mutation/deletion.
    DriverName string

    // AuthenticationType holds a copy of the BucketAccessClass authentication type at the time of
    // BucketAccess provisioning. This is kept to ensure the BucketAccess can be modified/deleted
    // even after BucketAccessClass mutation/deletion.
    AuthenticationType AuthenticationType

    // Parameters holds a copy of the BucketAccessClass opaque parameters at the time of
    // BucketAccess provisioning. These parameters are kept to ensure the BucketAccess can be
    // modified/deleted even after BucketAccessClass mutation/deletion.
    Parameters map[string]string
  }
```

The `accessSecretName` is the name of the Kubernetes Secret that COSI will generate.
The Secret will contain endpoint, credentials, and other information needed to access the OSP bucket.
The same Secret should be referenced by Pods to access the OSP bucket.

In the future, sharing buckets across namespaces can be allowed by adding a namespace field to BucketClaimReference.

Conditions:
- `Provisioned` - Indicates the backend access was created and should exist (or not).
- `ResourcesValidated` - Indicates validity of BucketAccess, BucketAccessClass, referenced BucketClaims, and referenced access Secrets.
  - `Unknown` when BucketAccessClass or BucketClaim reference(s) are not present (waiting) while provisioning.
  - `False` when BucketAccessClass is invalid while provisioning.
  - `False` when BucketClaim references are/become invalid.
  - `False` when referenced BucketClaim(s) go missing after provisioning.
  - `False` when access Secrets cannot be written to.
  - `False` if BucketAccess spec/status is/becomes internally inconsistent (degraded).
- `ProvisionFailed` - Records results of latest RPC access create.
  - E.g., driver auth expired (easily resolvable), or backend auth was lost (requires new access provisioning).

#### BucketAccessClass

The BucketAccessClass represents a set of common properties shared by multiple BucketAccesses.
It is used to specify policies for creating access credentials and for configuring driver-specific access parameters.

Resource for configuring common properties for multiple BucketAccesses. BucketAccessClass is cluster-scoped.

```go
DisallowedBucketAccessModes {
  // The ability to read/write objects within the bucket.
  // +optional
  ObjectData []BucketAccessMode

  // The ability to read/write metadata on objects within the bucket.
  // +optional
  ObjectMetadata []BucketAccessMode

  // The ability to read/write metadata on the bucket itself.
  // +optional
  BucketMetadata []BucketAccessMode
}

BucketAccessClass {
  TypeMeta
  ObjectMeta

  Spec BucketAccessClassSpec {
    // DriverName is the name of driver associated with this BucketAccess.
    // +required
    DriverName string

    // AuthenticationType denotes the mechanism of authentication.
    // Supported values:
    // - Key - (default) Pods authenticate explicitly to the OSP using security key(s) or token(s)
    // - ServiceAccount - Pods authenticate implicitly to the OSP based on mappings to a Kubernetes ServiceAccount
    AuthenticationType AuthenticationType

    // Parameters is an opaque map for passing in configuration to a driver for granting access to a bucket.
    // +optional
    Parameters map[string]string

    // multiBucketAccess specifies whether a BucketAccess using this class can reference multiple
    // BucketClaims. When omitted, this means no opinion, and COSI will choose a reasonable default,
    // which is subject to change over time.
    // Possible values:
    //  - SingleBucket: (default) A BucketAccess may reference only a single BucketClaim.
    //  - MultipleBuckets: A BucketAccess may reference multiple (1 or more) BucketClaims.
    // +optional
    MultiBucketAccess MultiBucketAccess

    // disallowedBucketAccessModes is a list of disallowed Read/Write access modes for each access
    // type. A BucketAccess using this class will not be allowed to request access to a BucketClaim
    // with any access mode listed here.
    // This is particularly useful for administrators to restrict access to a statically-provisioned
    // bucket that is managed outside the BucketAccess Namespace or Kubernetes cluster.
    DisallowedBucketAccessModes DisallowedBucketAccessModes
  }
}
```

Conditions: None.

#### BucketAccess Secret data

BucketAccess secrets contain information about the OSP bucket as well as credentials for accessing the OSP bucket.

Data is broken down into two types:
- **Bucket info** contains information about the bucket itself.
  This info is returned by drivers when provisioning both Buckets and BucketAccesses.
  - When Buckets are provisioned, returned information is stored as status on the Bucket, and drivers may opt not to return this info.
  - When BucketAccesses are provisioned, the returned information is stored in the BucketAccess Secret, and drivers must return this info.
- **Credential info** contains information about the bucket and/or access that is sensitive.
  The information is only stored in the BucketAccess Secret for security.

When a BucketAccess references multiple BucketClaims and is successfully provisioned, all backend buckets share the same credentials.
In this case, COSI includes the same credentials on all BucketAccess Secrets.
Credentials are duplicated across the multiple outputs, allowing users to retrieve them from any output Secret.

All Buckets and BucketAccesses have this top-level **bucket** info:

- `COSI_PROTOCOL`: (required) The protocol for accessing the bucket. (`S3`, `Azure`, `GCS`)

All BucketAccesses have this top-level **credential** info:

- `COSI_CERTIFICATE_AUTHORITY`: (optional) The certificate authority that applications can use to authenticate the object storage service endpoint.

Where available, protocol-specific variables below are defined to be well-known environment variable names taken from official protocol documentation.

##### S3

S3 bucket info:

AWS_STORAGE_BUCKET_NAME

- `AWS_ENDPOINT_URL`: S3 endpoint URL, e.g., `https://s3.amazonaws.com`
- `BUCKET_NAME`: S3 bucket name/ID (must be client-facing OSP bucket ID).
  Env var name is not official, but does appear in AWS docs and many projects.
- `AWS_DEFAULT_REGION`: S3 region, e.g., `us-west-1`
- `AWS_S3_ADDRESSING_STYLE`: S3 addressing style. one of `path` or `virtual`.
  Env var name is not official, but is used by both Boto and Django projects.
  See: https://docs.aws.amazon.com/AmazonS3/latest/userguide/VirtualHosting.html

S3 credential info:

- `AWS_ACCESS_KEY_ID`: S3 access key ID, e.g., `AKIAIOSFODNN7EXAMPLE`
- `AWS_SECRET_ACCESS_KEY`: S3 access secret key, e.g., `wJalrXUtnFEMI/K...`

##### Azure

Azure bucket info:

- `AZURE_STORAGE_ACCOUNT`: the ID of the Azure storage account

Azure credential info:

- `AZURE_STORAGE_SAS_TOKEN`: Azure access token. Note that the Azure spec includes the resource URI as well as token in its definition.
  https://learn.microsoft.com/en-us/azure/storage/common/media/storage-sas-overview/sas-storage-uri.svg
- `AZURE_STORAGE_SAS_TOKEN_EXPIRY_TIMESTAMP`: Empty if unset. Otherwise, date+time in ISO 8601 format.
  This is not used by clients to connect but is important information that may be read as needed.

##### GCS (Google Cloud Storage)

Note that COSI maintainership currently lacks GCS input or experience as of v1alpha2.
This spec attempts to add the fields that are likely needed, with the expectation that some fields may be missing.

GCS does not appear to have defined "official" environment variables.
Variable names have been created to closely match lower-case configuration fields where possible.

GCS bucket info:

- `PROJECT_ID`: GCS project ID
- `BUCKET_NAME`: GCS bucket name (must be client-facing OSP bucket ID)

GCS credential info:

- `SERVICE_ACCOUNT_NAME`: GCS service account name
- `CLIENT_EMAIL`: GCS service account email
- `CLIENT_ID`: GCS client ID
- `PRIVATE_KEY_ID`: GCS private key ID
- `PRIVATE_KEY`: GCS private key
- `HMAC_ACCESS_ID`: HMAC access ID
- `HMAC_SECRET`: HMAC secret

References:
- Service account management: https://docs.cloud.google.com/iam/docs/keys-create-delete
- HMAC management: https://docs.cloud.google.com/storage/docs/authentication/hmackeys

### COSI Driver

A component that runs alongside COSI Sidecar and satisfies the COSI gRPC API specification.
Sidecar and driver work together to orchestrate changes in the OSP.
The driver acts as a gRPC server to the COSI Sidecar.
Each COSI driver is identified by a unique ID.

The sidecar uses the unique ID to direct requests to the appropriate driver.
Multiple instances of drivers with the same ID will be added into a group, and only one of them will act as the leader at any given time.

#### COSI Driver gRPC API

##### DriverGetInfo

This gRPC call responds with the name of the driver.
The name is used to identify which resources the driver should manage.

```go
DriverGetInfo(DriverGetInfoRequest) DriverGetInfoResponse

DriverGetInfoRequest{}

DriverGetInfoResponse{
  "name": "<driver name>" // e.g., "s3.amazonaws.com"
  "supported_protocols": [ // one or more of:
    "S3",
    "Azure",
    "GCS"
  ]
}
```

##### DriverGenerateBucketId

This gRPC call requests a bucket identifier from the driver.
This API must be idempotent. This call does not apply to statically-provisioned Buckets.

This gRPC call is phase 1 of the 2-phase bucket provisioning process.

The recommended 2-phase provisioning process is outlined below:
1. DriverGenerateBucketId is called to generate a persistent `bucket_id` without provisioning backend resources
2. DriverCreateBucket is called to provision the backend bucket

This call requests a `bucket_id` that COSI will use for all subsequent gRPC calls related to the Bucket,
including DriverCreateBucket, which is phase 2 of the 2-phase provisioning process.
The returned ID must be unique, and the driver must be able to correlate `bucket_id` to an OSP backend bucket provisioned in phase 2.
It is easiest for drivers to use the request `name` field as both `bucket_id` and as the OSP backend bucket identifier, but this is not strictly required.

If the the Bucket resource is deleted before COSI can persist `bucket_id`, the DriverDeleteBucket gRPC will not be called.
Therefore, drivers must ensure that their implementation does not leak backend resources in this deletion edge case.
If possible, COSI recommends that each driver use a deterministic rule for generating the `bucket_id` without provisioning backend resources.
Using or appending random identifiers can lead to multiple unused buckets being created in the OSP backend in the event of timing-related driver/sidecar failures or restarts.

COSI uses `Request.name` as an idempotency key.
If COSI is unable to persistently store the returned `bucket_id`, COSI will retry the gRPC call with the same `name` later.
The Sidecar uses the name of the Bucket resource as the input value for `Request.name`.
This will be `bc-<BucketClaim.UID>` for dynamically-provisioned Buckets.
This is statistically likely to be globally unique even between multiple Kubernetes clusters.

The input parameters (`CreateParameters` below) given in this rRPC call are the same parameters later used for phase-2 provisioning.
This ensures drivers may use any of the creation parameters they desire when determining the `bucket_id`.

Input `protocols` is an optional list of protocols that the provisioned bucket must support.
This is a mechanism by which end users can "hint" about upcoming BucketAccesses to the backend driver.
This is useful for drivers that can support multiple protocols, or conceivable "meta-drivers".
If empty, different drivers/configs might provision a bucket differently.
A driver might provision a bucket with support for the most protocols possible, or a single preferred protocol.
Another might return an `InvalidArgument` error with a message saying that the driver requires this field.

Input `parameters` are the opaque parameters copied from the Bucket (originating from BucketClass).
Drivers can use these parameters to configure OSP bucket features based on the Administrator's BucketClass configuration.

```go
DriverGenerateBucketId(DriverGenerateBucketIdRequest) DriverGenerateBucketIdResponse

DriverGenerateBucketIdRequest{
  "name": "<Bucket.name>", // will be "bc-<BucketClaim.UID>" for dynamically-provisioned Buckets
  CreateParameters
}

DriverGenerateBucketIdResponse{
  "bucket_id": "<ID returned by driver>" // will be applied to Bucket.status.bucketID
}

CreateParameters {
  "protocols": ["S3", "Azure", "GCS"], // an optional list of protocols the bucket MUST support
  "parameters": { // copied from Bucket.parameters
    "<key>": "<value>"
    // ...
  }
}
```

Important return driver codes:
- `AlreadyExists` (not retryable) if the bucket already exists but is incompatible with the request
- `InvalidArgument` (not retryable) if any parameters are invalid for the backend

##### DriverCreateBucket

This gRPC call creates a bucket in the OSP, and returns information about the new bucket.
This API must be idempotent. This call does not apply to statically-provisioned Buckets.

This gRPC call is phase 2 of the 2-phase bucket provisioning process.

This call is issued after DriverGenerateBucketId's `bucket_id` result is persistently stored by COSI.
If possible, COSI recommends provisioning the backend bucket in this phase (phase 2).
Waiting until phase 2 to provision backend resources ensures backend resources are not leaked during a deletion edge case.
See DriverGenerateBucketId design above for detailed description of the deletion edge condition.

The driver should ensure that multiple DriverCreateBucket calls for the same `bucket_id` do not result in more than one OSP backend bucket being provisioned corresponding to that ID.

Returned `protocols` will serve as an identification of which protocols are supported for subsequent access to the bucket.

```go
DriverCreateBucket(DriverCreateBucketRequest) DriverCreateBucketResponse

DriverCreateBucketRequest{
  "bucket_id": "<Bucket.status.bucketID>" // e.g., "ba-<BucketClaim.UID>", "my-static-bucket"
  CreateParameters // same create parameters from generate ID request
}

DriverCreateBucketResponse{
  BucketInfo
}

BucketInfo {
  "protocols" : { // all protocols supported by the bucket, with bucket info (but not credential info)
    "S3": { // at least one protocol must be returned, or provisioning will be considered failed
      // returned bucket info is applied to Bucket status and should not contain any sensitive data
      // info fields are optional, in case info is considered sensitive by driver implementers
      "bucket_name": "<bucket ID in S3 backend>",
      "region": "<bucket region>",
      "endpoint": "<endpoint where bucket can be accessed>"
    },
    "Azure": { // e.g., second protocol
      "storage_account": "<storage account name>"
    }
    // ... optional additional protocols
  }
}
```

Important return driver codes:
- `AlreadyExists` (not retryable) when the bucket already exists but is incompatible with the request
- `InvalidArgument` (not retryable) if any parameters are invalid for the backend

##### DriverGetBucket

This gRPC call is used to get details about a statically-provisioned bucket that should already exist in the OSP backend.
This call does not apply to dynamically-provisioned Buckets.

`bucket_id` used for input is taken from `Bucket.existingBucketID`.
It should be a unique identifier for the OSP bucket known to the driver.
This value will be used by COSI to make all subsequent calls related to this bucket,
so the driver must be able to correlate `bucket_id` to the OSP backend bucket.

The input `parameters` are optional and passed the same was as for DriverCreateBucket.
If the backend bucket isn't compatible with the input parameters, error `InvalidArgument` must be returned.
Drivers are responsible for determining and documenting which parameters are required for this call.

```go
DriverGetBucket(DriverGetBucketRequest) DriverGetBucketResponse

DriverGetBucketRequest {
  "bucket_id": "<name of statically-provisioned bucket>" // e.g., "my-static-bucket"
  "protocols": ["S3", "Azure", "GCS"], // an optional list of protocols the bucket MUST support
  "parameters": { // copied from Bucket.parameters
    "<key>": "<value>"
    // ...
  }
}

DriverGetBucketResponse {
  BucketInfo // same response defined in DriverCreateBucket
}
```

The protocols returned in the response will be validated by COSI to match user requirements.

Important driver return codes:
- `NotFound` (retryable) when the bucket does not exist in the backend
- `AlreadyExists` (not retryable) when the bucket exists but isn't compatible with the input parameters

##### DriverDeleteBucket

This gRPC call deletes a bucket in the OSP.

```go
DriverDeleteBucket(DriverDeleteBucketRequest) DriverDeleteBucketResponse

DriverDeleteBucketRequest {
  bucket_id: "<Bucket.status.bucketID>" // e.g., "ba-<BucketClaim.UID>", "my-static-bucket"
  parameters: { // copied from Bucket.parameters
    "key": "value"
    // ...
  }
}

DriverDeleteBucketResponse{} // empty with return code
```

Important driver return codes:
- `OK` when OSP bucket already does not exist

##### DriverGenerateBucketAccessId

This gRPC call requests an account identifier from the driver.
This API must be idempotent.

This gRPC call is phase 1 of the 2-phase bucket provisioning process.

The recommended 2-phase provisioning process is outlined below:
1. DriverGenerateBucketAccessId is called to generate a persistent `account_id` without provisioning backend resources
2. DriverGrantBucketAccess is called to provision the backend access

This call requests an `account_id` that COSI will use for all subsequent gRPC calls related to the access,
including DriverGrantBucketAccess, which is phase 2 of the 2-phase provisioning process.
The returned ID must be unique, and the driver must be able to correlate `account_id` to an OSP backend account provisioned in phase 2.
It is easiest for drivers to use the request `name` field as both `account_id` and as the OSP backend account identifier, but this is not strictly required.

If the the Bucket resource is deleted before COSI can persist `account_id`, the DriverRevokeBucketAccess gRPC will not be called.
Therefore, drivers must ensure that their implementation does not leak backend resources in this deletion edge case.
If possible, COSI recommends that each driver use a deterministic rule for generating the `account_id` without provisioning backend resources.
Using or appending random identifiers can lead to multiple unused buckets being created in the OSP backend in the event of timing-related driver/sidecar failures or restarts.

COSI uses `Request.account_name` as an idempotency key.
This `account_name` field is the concatenation of the characters `ba-` (short for BucketAccess) and the BucketAccess UID.
Using or appending random identifiers can lead to multiple unused bucket accesses being created in the OSP backend in the event of timing-related driver/sidecar failures or restarts.
If COSI is unable to persistently store the returned `account_id`, COSI will retry the gRPC call with the same `account_name` later.
The Sidecar uses the BucketAccess resource UID as part of the input value for `Request.account_name`.
This will be `ba-<BucketAccess.UID>`.
This is statistically likely to be globally unique even between multiple Kubernetes clusters.

The input parameters (`GrantParameters` below) given in this rRPC call are the same parameters later used for phase-2 provisioning.
This ensures drivers may use any of the creation parameters they desire when determining the `account_id`.

When multiple `buckets` are input in the request, the expectation is that the driver will provision and return a single access with permissions for all given buckets.
For example, S3 using `Key` authentication can make multiple S3 buckets accessible by a single S3 user and its credentials.
The single S3 user's `Key` credentials are returned to satisfy the request.

Some drivers or backends may not be able to support multi-bucket requests.
For example, [Azure](#azure) using `Key` authentication uses the combined URI and access token to grant access to a single blob.
An Azure driver could not create a single credential that could access all Buckets (blobs).
In this case, the Azure driver should return gRPC code `OutOfRange` to indicate that it cannot support the multi-bucket request.
The COSI system will surface this error to users on the BucketAccess so they adjust their usage clearly.

Each `bucket_id` used for input is the same ID returned by the driver in DriverGenerateBucketId (dynamically-provisioned)
or given by `bucket_id` (statically-provisioned).

Input `access_parameters` are the opaque parameters copied from the BucketAccessClass. Drivers can use these parameters to configure OSP bucket access features based on the Administrator's BucketAccessClass configuration.

The returned `account_id` should be a unique identifier for the account in the OSP.
This value will be included in all subsequent calls to the driver for changes to the BucketAccess.
As `bucket_id` is to Bucket, `account_id` is to BucketAccess.

```go
DriverGrantBucketAccess(DriverGrantBucketAccessRequest) DriverGrantBucketAccessResponse

DriverGrantBucketAccessRequest{
  "account_name": "ba-<BucketAccess.UID>",
  GrantParameters
}

DriverGrantBucketAccessResponse {
  "account_id": "<ID returned by driver>", // will be applied to BucketAccess.status.accountID
}

GrantParameters {
  "buckets": [
    {
      "bucket_id": "<Bucket.status.bucketID>", // e.g., "ba-<BucketClaim.UID>", "my-static-bucket"
      "access_mode": "<accessMode>", // ReadWrite, ReadOnly, WriteOnly
    },
    // optional additional buckets the access should have permissions for
  ],
  "protocol": "<protocol>", // e.g., "S3", copied from BucketAccess.spec.protocol
  "authentication_type": "<[Key|ServiceAccount]>", // copied from BucketAccess.status.authenticationType
  "service_account_name": "<saName>", // copied from BucketAccess.spec.serviceAccountName
  "parameters": { // copied from BucketAccess.status.parameters
      "key": "value",
      // ...
  }
}
```

Important driver return codes:
- `AlreadyExists` (not retryable) when the bucket already exists but is incompatible with the request
- `InvalidArgument` (not retryable) if `AuthenticationType` is not supported
- `InvalidArgument` (not retryable) if any parameters are invalid for the backend
- `OutOfRange` (not retryable) if (and only if) the driver does not support creating a single shared access credential for multiple buckets

##### DriverGrantBucketAccess

This gRPC call creates a set of access credentials for one or more buckets.
This API must be idempotent.

The driver should ensure that multiple DriverGrantBucketAccess calls for the same `account_name` do not result in more than one OSP backend bucket access being provisioned corresponding to that name.

Returned `bucket_info` and `credentials` will be transformed into [BucketAccess secret data](#bucketaccess-secret-data) for end user consumption.

```go
DriverGrantBucketAccess(DriverGrantBucketAccessRequest) DriverGrantBucketAccessResponse

DriverGrantBucketAccessRequest{
  "account_id": "<BucketAccess.status.accountID>",
  GrantParameters // same parameters from DriverGenerateBucketAccessId
}

DriverGrantBucketAccessResponse {
  "buckets": [ // array/list of 'Bucket' type
      {
        "bucket_id": "<a bucket ID from input>"
        "bucket_info": {
          "S3": { // must match input protocol
            // unlike DriverCreateBucket, required fields are not optional since this info is used by
            // workloads to gain access to buckets
            "bucket_name": "<bucket ID in S3 backend>",
            "region": "<bucket region>",
            "endpoint": "<endpoint where bucket can be accessed>"
          }
        }
      }
      // ... additional buckets
    ]
  },
  "credentials": { // bucket access credentials
    "S3": { // must match input protocol. provisioning failure if required fields are missing
      "access_key_id": "<s3 access key id>", // e.g., "AKIAODNN7EXAMPLE"
      "access_secret_key": "<s3 access secret key>" // e.g., "wJaUtnFEMI/K..."
    }
  }
}
```

When `authentication_type` is not `ServiceAccount`, the COSI sidecar will always set `serviceAccountName` to an empty string in the gRPC call.
This helps steer driver implementation by ensuring the drivers supporting `Key` type do not rely on the optional input.

Important driver return codes:
- `AlreadyExists` (not retryable) when the bucket already exists but is incompatible with the request
- `InvalidArgument` (not retryable) if `AuthenticationType` is not supported
- `InvalidArgument` (not retryable) if any parameters are invalid for the backend
- `OutOfRange` (not retryable) if (and only if) the driver does not support creating a single shared access credential for multiple buckets

##### DriverRevokeBucketAccess

This gRPC call revokes access granted to a particular account.

```go
DriverRevokeBucketAccess(DriverRevokeBucketAccessRequest) DriverRevokeBucketAccessResponse

DriverRevokeBucketAccessRequest{
  "account_id": "<BucketAccess.status.accountID>"
  "buckets": [
    {
      "bucket_id": "<Bucket.status.bucketID>", // e.g., "ba-<BucketClaim.UID>", "my-static-bucket"
    },
    // ... additional buckets
  ],
  "protocol": "<protocol>", // e.g., "S3", copied from BucketAccess.spec.protocol
  "authentication_type": "<[Key|ServiceAccount]>", // copied from BucketAccess.status.authenticationType
  "service_account": {
    "namespace": "<sa namespace>",
    "name": "<sa name>",
  },
  "<saName>", // copied from BucketAccess.spec.serviceAccountName
  "parameters": { // copied from BucketAccess.status.parameters
      "key": "value",
      // ...
  }
}

DriverRevokeBucketAccessResponse{} // empty with return code
```

Important driver return codes:
- `OK` when OSP bucket access already does not exist

### Test Plan

- Unit tests will cover the functionality of the controllers.
- Unit tests will cover the new APIs.
- An end-to-end test suite will cover testing all the components together.
- Component tests will cover testing each controller in a blackbox fashion.
- Tests need to cover both correctly and incorrectly configured cases.

### Graduation Criteria

#### Alpha

- API is reviewed and accepted
- Design COSI APIs to support dynamic (greenfield) and static (brownfield) provisioning
- Design COSI APIs to support authentication using access/secret keys
- Evaluate gaps, update KEP and conduct reviews for all design changes
- Develop unit test cases to demonstrate that the above mentioned use cases work correctly

#### Beta

- Implement all COSI components to support agreed design.
- Basic unit and e2e tests as outlined in the test plan.
- Metrics for bucket create and delete, and granting and revoking bucket access.
- Metrics in provisioner for bucket create and delete, and granting and revoking bucket access.

#### GA

- Stress tests to iron out possible race conditions in the controllers.
- Users deployed in production and have gone through at least one K8s upgrade.
- Certification tests that help driver developers ensure their driver meets COSI
  requirements/expectations -- following the pattern of CSI sanity tests.

### Upgrade / Downgrade Strategy

No Kubernetes changes are required on upgrade to maintain previous behavior.

The COSI resource APIs have breaking changes from v1alpha1 to v1alpha2.
Migrations between versions are not supported via automation.

A v1alpha1 COSI Driver/Sidecar will be incompatible with the v1alpha2 Controller, and vice versa.
COSI v1alpha1 and v1alpha2 Controllers cannot be running at the same time.

To upgrade, first uninstall the v1alpha1 COSI Controller as well as any v1alpha1 Drivers.
Then, deploy the v1alpha2 COSI Controller and desired v1alpha2 Drivers.

Any COSI resources created using a v1alpha1 system will be incompatible as well.
The COSI project will document the static provisioning workflow.
Backend buckets previously created using a COSI v1alpha1 system be made accessible using this workflow.
The COSI project will document how to manually migrate from v1alpha1 to v1alpha2.

### Version Skew Strategy

COSI is out-of-tree, so version skew strategy is N/A

## Alternatives Considered

This KEP has had a long journey and many revisions.
Here we capture the main alternatives and the reasons why we decided on a different solution.
Any alternative can be reconsidered in the future as needed/desired.

### Automatically mount buckets to Pods

Early iterations of the COSI v1alpha1 spec tried to design a system that could mount bucket information to user Pods automatically, to serve as an analog to Pod PVC mounts.
Each Pod application may have different means of and needs for connecting to object storage buckets, so there is not a clear way for this to be implemented in a way that could accommodate all drivers and all users.

### Encode BucketAccess connection information in a JSON blob

In COSI v1alpha1, BucketAccess connection/authentication information was encoded in a JSON blob in the BucketAccess's chosen Secret.
This was intended to be mounted to user Pods as a file, which is considered generally safer than mounting via environment variables.
However, several driver implementers gave feedback that they were unable to process the JSON blob file into forms suitable for their applications.
They instead requested that each entry in the JSON blob be encoded as individual Secret data fields that could be loaded as files or environment variables based on the needs of their individual applications.
COSI v1alpha2 deprecated the JSON blob file in favor of the more flexible approach of encoding each field in a separate Secret data key.
Both forms are not supported in order to keep development and usage consistent.

### Cross-resource protection finalizers

In the v1alpha2 design cycle, COSI received feedback from sig-storage regarding the finalizers used for PV/PVC protection (`.../pv-protection` and `.../pvc-protection`).
In the real world, administrators/users have often removed the PV/PVC protection finalizers when issues are encountered, leading to broken system states or incorrect cleanup behavior for PV/PVC drivers.
To avoid these pitfalls, the COSI project was cautioned to design a system that doesn't rely on finalizers for managing order-of-operations concerns between different COSI resources.
COSI will use finalizers as needed but will attempt to avoid them for cross-resource protections.
COSI v1alpha2 has used the volume snapshot design as inspiration and is using annotations to help inform cross-resource references and bindings.
During development, COSI v1alpha2 may add cross-resource finalizers as an implementation detail if needed.

### Bucket creation annotation

Volume Snapshotter uses an annotation to track snapshot creation, which can take a long time.
This mechanism helps identify and prevent orphan snapshots from being created.
During the v1alpha2 KEP planning, COSI developers don't believe such a mechanism is necessary for COSI.
However, should such a protection be needed, COSI may implement it as an implementation detail outside of the KEP process.

### BucketClass field on Bucket resource

Persistent Volumes track the StorageClass on the PV object. This is used to control binding during static provisioning.
For COSI, the intent is that the Bucket should have a copy of any relevant BucketClass parameters from the time when the OSP bucket was provisioned.
For static provisioning, only the admin knows those parameters, and they are expected to copy them to the Bucket.
As of COSI v1alpha2, the plan is to use the Bucket's BucketClaim reference as the sole binding control mechanism, and not to require BucketClass to be tracked on the Bucket object.

### Updating BucketAccess Secrets

COSI v1alpha1 specified that if a BucketAccess Secret already exists, then it is assumed that credentials have already been generated, and the Secret is not overridden.
In practice, this disallows Administrators from rotating credentials internally.
Because other important bucket connection details are present in the Secret (e.g., endpoints), this also makes it impossible for systems to change hosting locations over time.
COSI must be able to update the Secret to reflect the OSP's most up-to-date information.

### BucketAccess static provisioning

Static provisioning has been requested by some COSI users who want the ability to make modifications to OSP accesses that are not supported by the OSP driver.
This suggests that static provisioning is a desire based around the need to work around driver limitations rather than an intrinsic need of the COSI specification.
Static BucketAccess provisioning would be possible to implement, but doing so would be incredibly complicated.
COSI, OSP drivers, Admins, and/or Users could (and probably would) each have different expectations and desires about how to handle corner cases in static access policies.
The COSI spec couldn't reasonably enforce consistent handling of those expectations.
Design and dev work would be large, and risk for bugs would be high.
Because of this COSI design decision, some Admins will undoubtedly make modifications to OSP accesses using manual OSP access tools.
These manual modifications will be done at the Admin's risk.
The OSP driver is not guaranteed to preserve such modifications, and the OSP driver may get stuck in a position where it doesn't know how to continue managing the BucketAccess.

### BucketAccess Read/Write AccessMode

Read/Write access modes to buckets might mean different things depending on context.
For example, object stores typically allow arbitrary metadata or tags to be applied to both buckets and objects within a bucket.
S3 and GCS backends separate bucket, object, and data read/write permissions.
Azure, conversely, does not have separate these metadata permissions.
Within S3 or GCS protocols, bucket metadata can also control certain behaviors of the backend system.
Administrators might or might not want end users to self-select behavior-modifying metadata.

COSI's best options for implementing this control are:
1. Include a single read/write permission that applies to all data and metadata access.
   - Con: Admins could want to only allow data read/write for most users
2. Include a single read/write permission that only applies to data access.
   - Con: Drivers would each have to implement any desired metadata read/write permissions as Buck(Access)Class parameters.
3. Separate the permissions into data, object metadata, and bucket metadata
   - Con: More complicated for users and COSI maintainers

After v1alpha2 review discussions, this design proposes option 3, and COSI maintainers will observe feedback.

### Multi-bucket BucketAccess considerations

COSI v1alpha2 introduced support for a BucketAccess to reference multiple BucketClaims.
Several alternative designs were considered when planning this feature.
As with all alternatives considered, these decisions are not final (yet) and can be revisited if/when new feedback arrives.

#### User consumption considerations

We expect the general 1-access:1-bucket case to be the most common among users.
For these users, having a single BucketAccess Secret for output seems ideal.
The single Secret containing both bucket information and access credentials allows users to access their bucket as simply as we could design it.

In the 1-access:many-buckets case, each bucket may have some different information.
Outputting the bucket info into a separate Secret per referenced bucket allows the above general-case usage to be re-used.
This is beneficial for users so that their usage is consistent and more easeful.
However, in the 1:many case, COSI and users expect all buckets to share the same access credentials.
This introduces some conceptual friction into the end user experience.

Should COSI create a separate Secret to house the single set of access credentials?
In a way, this more clearly represents the backend system's configuration.
However, this diverges from the general 1:1 case by requiring users to reference an additional Secret in the 1:many case.

The cases could be brought back in line by making the general 1:1 case output two secrets (bucket info and access credentials).
This is non-ideal because the general case is then made more complex.
Relatedly, the Rook project's ObjectBucketClaim (OBC) users have shown some misunderstandings and difficulty with the bucket info ConfigMap and bucket credential Secret architecture.
We don't believe adding complexity to the general case is worth the likely user confusion.

We ultimately settled on a design that re-uses the general 1:1 case usage for 1:many.
In this design, the single, shared access credentials are copied (duplicated) to all output Secrets.
This may introduce some conceptual confusion for users when provisioning 1:many access.
They may ask, "which secret contains my credentials?"
In this case, users can retrieve the shared credentials from **any** output Secret.
This is a detail that COSI must document clearly so as to alleviate as much confusion as possible.

The COSI team also considered whether it would be useful to have separate CRDs for 1:1 and 1:many use cases.
The existing BucketAccess would be used for the 1:1 case and would use a single Secret for output.
A new MultiBucketAccess would be created for the 1:many case and would output N+1 Secrets for output.
N secrets to house bucket info for each referenced BucketClaim, plus one additional for the access credentials.
This could help alleviate some user confusion by allowing each CRD to be documented separately.
This also doesn't remove the complexity or concerns noted above entirely: the complexity is mostly just moved elsewhere.
We didn't believe that this would improve the usability enough to justify the added KEP complexity.

#### Handling systems that cannot support multi-bucket BucketAccesses

Azure blob storage using SAS tokens for `Key`-based access cannot support multi-bucket BucketAccesses.
There may be other protocols, systems, or drivers that do not support multi-bucket BucketAccesses.

One of COSI's primary goals is portability across Kubernetes systems.
Many ways of handing this challenge conflict with portability.
The final resolution for this issue should be one that minimizes portability issues.

The most obvious alternative is, "what if we don't support multi-bucket BucketAccesses?"
This has consistently been the most-requested COSI and ObjectBucketClaim feature.
We do not believe that omitting it from the spec is in users's interests.

COSI could introduce a new gRPC call (or response) that would allow drivers to indicate whether they support the feature or not.
However, taking Azure as an example, `Key` auth access cannot be provisioned, but `ServiceAccount` auth could be.
More generally, there could be unknown reasons or specific configurations that would prevent a driver from supporting this.
The driver would have to indicate which permutations of features do and don't support this.
Even if it is possible, the design logic and gRPC documentation would likely be infeasible.

COSI could have a separate gRPC call for this case. E.g., `DriverGrantMultipleBucketAccess`.
A driver could return a gRPC code `Unimplemented` for this call.
This idea could be combined with the `MultiBucketAccess` CRD idea above, or be implemented separately.
This could be explored more, but the current `DriverGrantBucketAccess` call doesn't seem overloaded currently.

The design proposed is twofold:
1. By default, disallow multi-bucket access to keep the most portable configuration as the default.
   Make this configurable via BucketAccessClass.
2. Instruct drivers to return gRPC code `OutOfRange` if (and only if) it cannot support a multi-bucket access request.
   This allows drivers to clearly report misconfigurations that may be especially helpful for corner cases.

In the absence of an alternative design that allows more portability, we believe the current design is an acceptable tradeoff.
As a somewhat parallel comparison, porting a PVC from one Kubernetes cluster to another may fail, even if it is rare.
Reserving a dedicated error code helps users and admins quickly identify the issue and determine what their recovery options are.

## Production Readiness Review Questionnaire

<!--

Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In som. e cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

Kubernetes cluster admins can deploy COSI via this high-level flow:

1. Create CRDs and Deployment for COSI controller including any supporting resources like ServiceAccounts
2. Create Deployment for at least one COSI driver (includes sidecar) and any supporting resources (not part of core COSI KEP)
3. Create at least one BucketClass and BucketAccessClass for end user selection

Will enabling / disabling the feature require downtime of the control plane?
- No

Will enabling / disabling the feature require downtime or reprovisioning of a node?
- No

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->
No

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. Delete the resources created when installing COSI.

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

All COSI components are out-of-tree.
We don't need extra feature gate testing that would be needed for in-tree features.
COSI API is CRD-related and does not require core API conversion.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

Resources are deployed via Kubernetes Deployments with already-existing rollout/rollback systems.

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

COSI's controllers don't impact the data path of Pods using already-running object storage applications.

However, if upgrade fails resulting in COSI unavailability, users will be unable to create new Buckets or Bucket Accesses.
Other features that alter existing buckets also might not be available during rollout/rollback failure.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade-\>downgrade-\>upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

The operator can query Bucket* objects to find if their workloads are associated with them.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- Bucket
  - [ ] Events
    - FailedCreateBucket - Report when COSI fails to create a bucket, with error message
    - FailedDeleteBucket - Report when COSI fails to delete a bucket, with error message
  - [ ] API .status
    - [x] ReadyToUse bool
    - [ ] ErrorMessage string - last error message; cleared when provisioning is successful
    - [x] BucketID string
- BucketClaim
  - [ ] Events
    - FailedCreateBucket - Report when COSI fails to create bucket for BC, with error message
    - FailedDeleteBucket - Report when COSI fails to delete bucket for BC, with error message
  - [ ] API .status
    - [x] ReadyToUse bool
    - [ ] ErrorMessage string - last error message; cleared when provisioning is successful
    - [x] BucketName string
- BucketAccess
  - [ ] Events
    - WaitingForBucket - Report when COSI cannot grant access because bucket does not yet exist
    - FailedGrantAccess - Report when COSI fails to grant access to a bucket, with error message
    - FailedRevokeAccess - Report when COSI fails to revoke access to a bucket, with error message
  - [ ] API .status
    - [x] ReadyToUse bool
    - [ ] ErrorMessage string - last error message; cleared when provisioning is successful
    - [x] AccountID string
- BucketClass
  - Does not have events or status
- BucketAccessClass
  - Does not have events or status
- COSI Controller
  - Does not have events or status; it will add events and status to CRs
  - Logs will be sufficient for deeper info
- COSI Provisioner Sidecar
  - Does not have events or status; it will add events and status to CRs
  - Logs will be sufficient for deeper info

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] `cosi_operation_total_seconds`
  - Type: Histogram
    - Histogram Buckets: 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 15, 30, 60, 120, 300, 600, '+Inf'
  - Reported by: COSI Controller
  - Definition: COSI operation end-to-end duration in number of seconds. For example, the duration
    from when a BucketClaim resource is created until BucketClaim has `Status.ReadyToUse=true`.
  - Labels:
    - `driver_name` - name of COSI driver the operation runs against
    - `resource_kind` - Bucket, BucketClaim, BucketAccess
    - `operation` - Create, Delete
  - Calculation note:
    - Create:
      - Time delta between the resource's meta.creationTimestamp and when Status.XReady=true is successfully applied
    - Delete:
      - Time delta between the resource's meta.deletionTimestamp and when the resource's finalizer is successfully removed
- [ ] `cosi_operation_count`
  - Type: Counter
  - Reported by: COSI Controller
  - Definition: Total number of end-to-end reconciliations conducted by the COSI controller.
  - Labels:
    - `driver_name` - name of COSI driver the operation runs against
    - `resource_kind` - Bucket, BucketClaim, BucketAccess
    - `operation` - Create, Delete
    - `status` - Unknown, Succeeded, Canceled
- [ ] `cosi_sidecar_operation_duration_seconds`
  - Type: Histogram
    - Histogram buckets: 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 15, 30, 60, 120, 300, 600, '+Inf'
  - Reported by: COSI provisioner sidecar
  - Definition: Total number of seconds spent by the controller on a gRPC operation from end to end
  - Labels:
    - `driver_name` - name of the COSI driver the operation runs against
    - `method_name` - gRPC operation name (e.g., `DriverCreateBucket`, `DriverGetInfo`)
    - `grpc_status_code` (e.g., "OK", "InvalidArgument")
- [ ] `cosi_sidecar_operation_errors_total`
  - Type: Counter
  - Definition: Total number of errors returned from a gPRC operation
  - Reported by: COSI provisioner sidecar
  - Labels:
    - `driver_name` - name of the COSI driver the operation runs against
    - `method_name` - gRPC operation name (e.g., `DriverCreateBucket`, `DriverGetInfo`)

- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

No

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

Existing components will not make any new API calls.

The API load of COSI components will be a factor of the number of buckets being managed and the number of bucket-accessors for these buckets.
Essentially O(num-buckets * num-bucket-access).
COSI has no no per-node or per-namespace load.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

Yes, the following cluster scoped resources

- Bucket
- BucketClass
- BucketAccessClass

and the following namespaced scoped resources

- BucketAccess

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

Not by the framework itself. Calls to external systems will be made by vendor drivers for COSI.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

No

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

Yes. Containers requesting Buckets will not start until Buckets have been provisioned.
This is similar to dynamic volume provisioning

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

Not likely to increase resource consumption in a significant manner

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

Major milestones:
- KEP created 25 Nov. 2019
- v1alpha1 approved alongside Kubernetes v1.25
- v1alpha2 approval targeted alongside Kubernetes v1.36

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

N/A
