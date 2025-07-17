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
      - [DriverCreateBucket](#drivercreatebucket)
      - [DriverGetExistingBucket](#drivergetexistingbucket)
      - [DriverGrantBucketAccess](#drivergrantbucketaccess)
      - [DriverDeleteBucket](#driverdeletebucket)
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
  - [IAM authentication type](#iam-authentication-type)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements][57] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website][58], for publication to [kubernetes.io][59]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This proposal introduces the *Container Object Storage Interface* (COSI), a standard for provisioning and consuming object storage in Kubernetes.

## Motivation

File and block storage are treated as first class citizens in the Kubernetes ecosystem via CSI.  Workloads using CSI volumes enjoy the benefits of portability across vendors and across Kubernetes clusters without the need to change application manifests. **An equivalent standard does not exist for Object storage.**

Object storage has been rising in popularity in the recent years as an alternative form of storage to filesystems and block devices. Object storage paradigm promotes disaggregation of compute and storage. This is done by making data available over the network, rather than locally. Disaggregated architectures allow compute workloads to be stateless, which consequently makes them easier to manage, scale and automate.

### Goals

#### Functionality

- Support automated **Bucket Creation**
- Support automated **Bucket Deletion**
- Support automated **Access Credential Generation**
- Support automated **Access Credential Revocation**
- Support automated **Bucket Provisioning** for workloads (enabling pods to access buckets)
- Support **Bucket Reuse** (use existing buckets via COSI)
- Support **Automated Bucket Sharing across namespaces**

#### System Properties

- Support **workload portability** across clusters
- Achieve the above goals in a **vendor neutral** manner
- Standardize mechanism for third-party vendors to integrate easily with COSI
- Allow users (non-admins) to create and utilize buckets (self-service)
- Establish best-in-class **Access Control** practices for bucket creation and sharing

### Non-Goals

- **Data Plane** API standardization will not be addressed by this project
- **Bucket Mutation** will not be supported as of now  <!-- TODO: change when we design attribute classes -->

## Proposal

### User Personas

We define 3 kinds of stakeholders:

- **Administrators** (a.k.a., admin)
  - Administrators of the Kubernetes cluster's COSI functionality
  - Deploy, configure, and manage COSI vendor drivers in the Kubernetes cluster
  - Establish cluster-wide policies for access control over COSI resources

- **Application developers** (a.k.a., DevOps, user)
  - End-users of COSI resources to claim and use object storage buckets for their applications
  - Request a bucket to be created and/or provisioned for workloads
  - Request access to be created/deleted for a specific bucket
  - Request deletion of a created bucket

- **Object Storage Providers** (a.k.a., OSP, vendor)
  - Vendors who offer object storage capabilities for any given object storage system
  - Comparable to a storage vendor in the CSI domain
  - Creates a COSI driver for their object storage system following the COSI API specs

## Design details

### Important changes between versions

v1alpha1 to v1alpha2:

- DeletionPolicy is now a required field. This way the user has to explicitly specify it, leaving no room for confusion.

- DeletionPolicy behavior was clarified. This field controls both retention/deletion of both the Bucket object and the underlying OSP bucket. This behavior mirrors the Volume Snapshot KEP, which is considered to be the latest best-practice.

- DeletionPolicy is now a mutable field.

- Sidecar has a reduced set of permissions. It now only reconciles Bucket and BucketAccess and no longer reads Bucket(Access)Class or BucketClaim. Keeping Sidecar operations limited in scope helps avoid version skew incompatibility issues between Controller and Sidecar. It also means that that COSI Controller updates are more likely to be able to resolve COSI bugs without driver deployments needing to be updated. BucketAccess provisioning is required to be more complex to allow for the Controller to do initial processing and then hand off provisioning to the Sidecar.

- To support a sidecar with reduced permissions, BucketAccesses now have several status fields that are set by the Controller based on content from the referenced BucketClaim and provision-time BucketAccessClass details.

- BucketClassName was removed from Bucket spec. All BucketClass configurations and parameters are expected to be copied to the Bucket at provision-time to ensure BucketClass mutation over time does not affect ongoing reconciles of Bucket objects, rendering BucketClassName irrelevant.

- Finalizers that prevent deletion of objects that are referenced by other objects are removed from the design. Instead, annotations are used, and the COSI Controller itself manages whether objects are prevented from being cleaned up. This comes from a recommendation by sig-storage maintainers that finalizers are often removed by users and therefore don't offer much system stability in the real world. These may be added back in the future if needed.

- The BucketAccess secret now has individually-named fields and no longer holds a JSON blob containing connection/authentication details. Several users reported challenges or the inability to configure their applications using the JSON blob and strongly prefer individual fields that can be loaded into environment variables or files as needed. COSI chose to not support both JSON blob and individual fields to keep the implementation simpler.

- A new Sidecar gRPC call was added for validating statically-provisioned Buckets have a viable OSP backend bucket.

- BucketClaim status `BucketName` was changed to `BoundBucketName` for API clarity.

- The BucketAccess secret no longer includes the name of the Bucket resource it is associated with. It is not necessary for end-user Pods to know the name of the Bucket resource. For the S3 protocol, the Pod may need to know the OSP backend bucket ID, which is included as part of the S3 info.

- BucketAccess protocol is now required. This requires users to state their intentions upfront, which allows COSI to ensure compatibility with what is provisioned.

- DriverGetInfo gRPC call now expects drivers to return (advertise) the list of supported protocols. COSI will check that Bucket/Claim requested Protocols match what is advertised.

- BucketAccess `Protocol` field is now required.

### COSI Architecture

Since this is an entirely new feature, it is possible to implement this completely out of tree.
The following components are proposed for this architecture:

- COSI Controller
- COSI Sidecar
- COSI Driver

1. The COSI Controller is the central controller that validates, authorizes, and binds COSI created
   buckets to BucketClaims. Only one active instance of Controller should be present.
2. The COSI Sidecar is the point of integration between COSI and each driver. Each operation that
   requires communication with the OSP is triggered by the Sidecar using gRPC calls to the driver.
   One active instance of Sidecar should be present **for each driver**.
3. The COSI driver communicates with the OSP to conduct object operations and fulfill gRPC requests
   from the Sidecar.

More information about COSI driver is [here](#cosi-driver)

### COSI API Overview

COSI defines these new API types:

- [Bucket](#bucket)
- [BucketClaim](#bucketclaim)
- [BucketAccess](#bucketaccess)
- [BucketClass](#bucketclass)
- [BucketAccessClass](#bucketaccessclass)

Detailed information about these API types are provided inline with user stories.

Here is a TL;DR version:

- BucketClaim/Bucket are analogous to Kubernetes PVC/PV.
- BucketClaim is used to request generation of new buckets.
- Bucket represents a bucket (or blob) in the OSP backend.
- BucketClass is similar to StorageClass. It allows admins to define and control policies for Bucket creation.
- BucketAccess requests credentials that allow a user to consume a bucket. This approximates PV/PVC
  mounting functionality.
- BucketAccessClass allows admins to control authz/authn for users requesting access to buckets.

### COSI Object Lifecycle

The following resources are managed by Admins. All are cluster-scoped.

- BucketClass
- BucketAccessClass
- Bucket (in case of a bucket that already exist in OSP backend)

The following resources are managed by a User. All are namespace-scoped. Each is created with a reference to a corresponding class object.

- BucketClaim -\> BucketClass
- BucketAccess -\> BucketClaim, BucketAccessClass

For a greenfield BucketClaim/Bucket resources created by a User, the COSI controller responds by creating an intermediate Bucket object as shown below.

- BucketClaim -\> new(Bucket)

Notes:

- There are **NO** cycles in the relationship graph of the above mentioned API objects.
- Mutations are not supported in the API.
- Class objects have a lifecycle independent of objects that reference them.
  - BucketClaim, BucketAccess, and Bucket must have all necessary class parameters copied to them during provisioning to allow themselves to be deleted if class objects have been mutated/deleted.

### Usability

#### User Self-Service

User self-service is made possible using BucketClaim and BucketAccess resources (namespace-scoped). Users do not require admin privileges to create, modify, and/or delete them.

An admin is responsible for creating class objects (BucketClass, BucketAccessClass) which configure OSP-specific storage parameters. The creation of COSI class objects is deliberately analogous to creation and management of Kubernetes StorageClasses for PVCs. This is a well-understood pattern, and relying on familiarity will aid COSI users.

Importing a bucket that already exists in an OSP backend (a brownfield bucket) requires special permissions because its lifecycle is not managed by COSI. Special care needs to be taken to prevent unintended clones, accidental deletion, and other mishaps that could affect the OSP bucket. For instance, setting the deletion policy to Delete for a brownfield bucket should be disallowed. Admins are thus responsible for creating Bucket resources for brownfield buckets.

#### Mutating Buckets

As of the current design of COSI, mutating bucket properties is not supported. However, the current design does not prevent us from supporting it in the future. Mutable properties will be supported in future versions along with the capability to mutate them.

#### Sharing Buckets across Namespaces

As of the current design of COSI, any Bucket associated with a BucketClaim in one namespace cannot be accessed in another namespace. I.e. no automatic cross-namespace bucket sharing is possible. In future versions, a namespace-level access control will be enforced, and Buckets will be constrained to particular namespaces using selectors. Admins will be able to control which namespaces can access which buckets using namespace selectors. In theory, an Admin can create a new static reference to an existing Bucket to allow it to be consumed for an additional namespace.

### Controller overview

COSI is split into two controllers: the main COSI Controller and a driver Sidecar. This mirrors CSI's system design.

The Controller watches for all COSI resource CRD kinds. The Sidecar watches for the minimal set of CRD kinds in an effort to keep Sidecar implementation complexity low and to limit the effects of version skew between Controller and Sidecar.

Minimal Sidecar permissions:
- Read Bucket spec; read/write Bucket metadata, status, and finalizers
- Read BucketAccess spec; read/write BucketAccess metadata, status, and finalizers
- No permissions for BucketClaim, BucketClass, BucketAccessClass
- Create/Update/Patch/Delete Secrets in any namespace, for managing BucketAccess Secrets
- No delete permissions for other resources (including Bucket/BucketAccess)

For dynamic provisioning, a user creates a new BucketClaim object referencing a BucketClass object corresponding to a driver. This causes the COSI Controller to trigger creation of a Bucket object to represent the to-be-created OSP bucket.

The creation of a new Bucket object causes the Sidecar to provision a new bucket in the OSP driver. When the OSP bucket is successfully provisioned, the Sidecar updates the Bucket status to represent the new OSP bucket.

The COSI Controller will update the status field of the BucketClaim object accordingly based on the status field of the Bucket object to indicate the bucket is ready to be used or failed.

When a BucketClaim object is deleted, the COSI Controller adds an annotation to the Bucket object to indicate that the parent claim is being deleted. This is an indication to the Sidecar that it's safe to unbind the Bucket from the BucketClaim. The Controller then sets a deletion timestamp on the Bucket object.

If the deletion policy indicates that the data should be deleted, the Sidecar will call the OSP driver to delete the bucket. If deletion succeeds, the Sidecar will allow the Bucket object to be deleted by deleting the CRD finalizer.

If the deletion policy indicates that data should be retained, the BucketClaim will be deleted while the corresponding Bucket resource (as well as the underlying OSP bucket) will be retained.

### Control Flows

This section outlines the scenarios that COSI personas will initiate, and for what purpose. Each scenario includes enough detail to express the important interaction requirements between personas and the COSI system, and between COSI components. This section avoids unnecessarily naming specific API elements so as not to confuse complex system interaction requirements with specific implementation/spec details.

#### Installing the COSI System

Admin installs the COSI system and driver(s) to allow User self-service.

1. Assume that a Vendor has already created a COSI driver
2. Admin deploys the COSI controller
3. Admin deploys vendor COSI driver
4. Admin creates BucketClass and AccessClass configuring COSI and vendor driver features

#### Creating a Bucket

User self-provisions a bucket to store their workload's data.

The fundamental key to this design is the bi-directional "pointer" between Bucket and BucketClaim, which is represented in the claim status and bucket spec. The bi-directionality is complicated to manage in a transactionless system, but without it we can't ensure sane behavior in the face of different forms of trouble. For example, a rogue HA controller instance could end up racing and making multiple bindings that are indistinguishable, resulting in potential data loss.

1. User creates BucketClaim that uses BucketClass
2. COSI controller observes BucketClaim
   1. Controller applies `objectstorage.k8s.io/protection` finalizer to the BucketClaim
   2. Controller looks up corresponding Bucket
   3. If Bucket does not exist, Controller creates intermediate Bucket resource with these details:
      1. Bucket.name is `bc-`+`<BucketClaim.UID>` (safe if multiple controllers active)
      2. BucketClass parameters are copied to Bucket (needed for deletion/modification)
      3. Full BucketClaim reference info (with UID) is set on intermediate Bucket spec (Bucket is bound to claim)
   4. Controller fills in BucketClaim status to point to intermediate Bucket (claim is now bound to Bucket)
   5. Controller waits for the intermediate Bucket to be reconciled by COSI sidecar
3. COSI Sidecar detects intermediate Bucket resource
   1. If the Bucket's driver matches the sidecar's driver, continue
   2. Sidecar applies `objectstorage.k8s.io/protection` finalizer to intermediate Bucket
   3. Sidecar calls the COSI driver via gRPC to provision the OSP bucket
   4. If OSP returns provision fail, COSI sidecar reports error to Bucket status and retries w/ backoff
4. When OSP returns provision success, COSI sidecar updates Bucket status `ReadyToUse` to true
5. Controller detects that the Bucket is provisioned successfully (`ReadyToUse`==true)
   1. Controller finishes BucketClaim reconciliation processing
   2. Controller validates BucketClaim and Bucket fields to ensure provisioning success
   3. Controller copies Bucket status items to BucketClaim status as needed. Importantly:
      1. Supported protocols
      2. `ReadyToUse`

#### Accessing an Existing OSP Bucket

User needs access to a bucket that already exists in an OSP object store.

This is the static provisioning scenario. This can be used to migrate "brownfield" buckets that pre-existed COSI installation in a Kuberentes cluster.

In early COSI feedback and in other object storage self-service frameworks, users commonly want access to OSP buckets that are preexisting. However, giving end users unrestricted access to OSP storage would allow them to easily gain access to sensitive data they may not be intended to access. To resolve this, the Admin is expected to allow access to existing OSP buckets.

1. Admin creates a Bucket object that represents the existing OSP bucket
   1. Admin must specify the existing OSP bucket ID in the Bucket spec
   2. Admin should specify driver parameters (normally copied from BucketClass) for all parameters
      needed for driver functionality
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
   2. Controller looks up corresponding Bucket - if DNE, retry with backoff (or when Bucket is created)
   3. Controller applies `objectstorage.k8s.io/protection` finalizer to BucketClaim
   4. If BucketClaim reference set by admin on Bucket doesn't match, error out
   5. Apply Full BucketClaim reference info (with UID) to Bucket spec (Bucket is now bound to claim)
   6. Set BucketClaim status to point to Bucket (claim is now bound to Bucket)
   7. If Bucket status `ReadyToUse` is not true, wait for Bucket to be updated
   8. Controller validates BucketClaim and Bucket fields to ensure provisioning success
   9. Controller copies Bucket status items to BucketClaim status as needed. Importantly:
      1. Supported protocols
      2. `ReadyToUse`

#### Deleting a Bucket

User deletes a BucketClaim they no longer need.

Users cannot delete global-scope Buckets directly. Instead, a User deletes a BucketClaim they have have delete permissions for, and COSI coordinates deleting the Bucket.

Each Bucket has a deletion policy that determines whether COSI deletes the Bucket resource when the BucketClaim it's bound to is deleted.
Deletion policy options:
- `Delete`: Bucket and underlying OSP bucket are deleted as part of BucketClaim deletion process
- `Retain`: Bucket is unbound, but the Bucket resource and underlying OSP bucket are kept

An Admin (or other privileged user) can delete any Bucket resource at any time. When a Bucket is deleted, COSI should prevent the Bucket from being deleted until the BucketClaim it is bound to is also in deleting state.

BucketClaims having valid BucketAccesses (i.e., claims in use) will not be deleted until all the BucketAccesses are cleaned up.

When a BucketClaim with Bucket reclaim policy `Reclaim` that is deleted, the Bucket is left in place when the BucketClaim is removed. COSI does not support the Bucket being automatically re-bound to a new BucketClaim without Admin intervention. Admins must follow the [static provisioning workflow](#accessing-an-existing-osp-bucket) to allow re-binding.

1. User deletes BucketClaim object
2. COSI Controller detects BucketClaim resource's deletion timestamp
   1. Controller looks up the corresponding Bucket
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

Access credentials are represented by BucketAccess objects. The separation of BucketClaim and BucketAccess is a reflection of the usage pattern of object storage, where buckets are always accessed over the network, and all access is subject to authentication and authorization. The lifecycle of a bucket and its access are not tightly coupled.

If a BucketClaim is in deleting state, no new BucketAccesses can be created for it.

1. User creates BucketAccess that uses BucketAccessClass and references BucketClaim
   1. User specifies Kubernetes Secret name into which BucketAccess information will be stored upon successful access provisioning
2. COSI Sidecar detects the BucketAccess resource
   1. Initially, corresponding Bucket in BucketAccess status is unknown, so sidecar exits with no action
3. COSI Controller detects the BucketAccess resource
   1. Controller looks up corresponding BucketClaim
   2. If BucketClaim is being deleted, error without retry
   3. Controller sets `objectstorage.k8s.io/protection` finalizer on BucketAccess
   4. Controller sets `objectstorage.k8s.io/has-bucketaccess-references` annotation on corresponding BucketClaim
      (block claim from being deleted until access is deleted)
   5. If BucketClaim not ready, exit with retry
   6. If Bucket-BucketClaim binding is not valid, exit and retry when Bucket/Claim updated
   7. Once everything looks good on Bucket+Claim:
      1. Set corresponding Bucket name on BucketAccess status
      2. Copy BucketAccessClass specs and parameters to BucketAccess status
4. COSI Sidecar detects the BucketAccess resource
   1. BucketAccess status now shows corresponding Bucket name and BucketAccessClass info, so sidecar can provision
   2. If the BucketAccess's driver matches the sidecar's driver, continue
   3. Sidecar applies `objectstorage.k8s.io/protection` finalizer to the BucketAccess if needed
   4. Sidecar looks up the Bucket to get necessary info
   5. If Bucket has `objectstorage.k8.io/bucketclaim-being-deleted` annotation or deletion timestamp, error without retry
      (this indicates the claim is being deleted, possibly race condition missed in Controller)
   6. Sidecar calls the COSI driver via gRPC to generate unique access credentials for the Bucket
   7. If OSP returns provision fail, Sidecar reports error to BucketAccess status and retries gRPC call
   8. When OSP returns provision success, COSI sidecar:
      1. Applies `objectstorage.k8s.io/protection` finalizer to the Secret
      2. Updates the BucketAccess Secret with all info needed to access the OSP bucket
      3. Updates BucketAccess status `ReadyToUse` to true

#### Deleting a BucketAccess

User deletes a BucketAccess they no longer need.

COSI does not set up or manage mounting BucketAccess information to Pods consuming the BucketAccess. As such, COSI will delete a BucketAccess and its associated Secret without checking if the Secret is mounted to any Pods.

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

The BucketAccess secret can be provided to the pod using any Kubernetes {Secret -> Pod} attachment mechanism. This naturally includes mounting data into environment variables and files. Mounting credential data into files is slightly more secure than environment variables and is thus recommended. However, each application has different requirements, and some may require environment variables for configuring access.

### COSI API Reference

<!-- TODO: clarify how to reuse access for multiple buckets -->

#### Annotations and finalizers

Annotations:
- `objectstorage.k8s.io/bucketclaim-being-deleted`: applied to a Bucket when the Controller detects that the Bucket's bound BucketClaim is being deleted
- `objectstorage.k8s.io/has-bucketaccess-references`: applied to a BucketClaim when the Controller detects that one or more BucketAccesses reference the claim
- `objectstorage.k8s.io/sidecar-cleanup-finished`: applied to a BucketAccess when the Sidecar has finished cleaning up, allowing the Controller to begin its final cleanup operations

Finalizers:
- `objectstorage.k8s.io/protection`: applied to BucketClaims, Buckets, BucketAccesses, and BucketAccess Secrets to prevent them from being deleted until COSI has cleaned up intermediate and underlying resources

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
    BucketClaim corev1.ObjectReference

    // ExistingBucketID is the unique id of the bucket in the OSP. This field should be used to
    // specify a bucket that has been statically provisioned.
    // This field will be empty when the Bucket is dynamically provisioned by COSI.
    // +optional
    ExistingBucketID string
  }

  Status BucketStatus {
    // ReadyToUse is a boolean condition to reflect the successful creation of a bucket.
    ReadyToUse bool

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

    // Error holds the most recent error message, with a timestamp.
    // This is cleared when provisioning is successful.
    Error *TimestampedError
  }
}
```

`BucketInfo` is provided as a means of allowing Administrators more insight into how OSP drivers have provisioned Buckets. This may be important for debugging. This information is not copied to the BucketClaim status so that it is not visible to end Users. Info is rendered in the [BucketAccess Secret Data](#bucketaccess-secret-data) format. This serves two purposes: (1) it is a format that will be familiar and defined, and (2) it makes it easy for COSI to copy the data into the BucketAccess Secret later.

Once created, a Bucket object is immutable, except for fields specifically noted:
- `DeletionPolicy` should be mutable to allow Admins to change to `Retain` policy after creation.

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
    // ReadyToUse indicates that the bucket is ready for consumption by workloads.
    ReadyToUse bool

    // BoundBucketName is the name of the provisioned Bucket in response to this BucketClaim. It is
    // generated and set by the COSI controller before making the creation request to the OSP backend.
    BoundBucketName string

    // Protocols is the set of protocols the provisioned Bucket supports. BucketAccesses can request
    // to access this BucketClaim using any of the values given here.
    // Possible values: S3, Azure, GCS
    Protocols []Protocol

    // Error holds the most recent error message, with a timestamp.
    // This is cleared when provisioning is successful.
    Error *TimestampedError
  }
```

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

#### BucketAccess

The BucketAccess is used to request access to a bucket. It contains fields for choosing the Bucket for which the credentials will be generated, and also includes a bucketAccessClassName field, which in-turn contains configuration for authorizing users to access buckets.

A resource to access a Bucket. BucketAccess is namespace-scoped.

For many OSP drivers, each BucketAccess will correspond with a unique OSP backend 'user' (e.g., an S3 user). COSI suggests this as a starting point, but this is not a strict requirement.

COSI end Users should generally expect that BucketAccess-BucketClaim pairings should not result in BucketAccesses that are not able to read or write unrelated BucketClaims. Stated another way, BucketAccess A referencing BucketClaim A should generally not be able to read or write to BucketClaim B. COSI drivers may choose to have options for different behavior to support niche scenarios, but this should be the default assumption for BucketAccess provisioning.

COSI does not support static provisioning for BucketAccesses. Portability is still maintained because object storage accesses do not hold critical application data. Any BucketClaim can have a new, valid BucketAccess created for it at any time to provide access to the data. Because of this, it follows that it is possible to reclaim access to a Bucket that was ported without need for static access provisioning.

```go
BucketAccess {
  TypeMeta
  ObjectMeta

  Spec BucketAccessSpec {
    // BucketClaimName is the name of the BucketClaim.
    // +required
    BucketClaimName string

    // BucketAccessClassName is the name of the BucketAccessClass.
    // +required
    BucketAccessClassName string

    // Protocol is the name of the Protocol that this access credential is expected to support.
    // +required
    Protocol Protocol

    // AccessSecretName is the name of the Kubernetes secret that COSI should populate with access
    // details and credentials.
    // This secret is deleted when the BucketAccess is deleted.
    // +required
    AccessSecretName string
  }

  Status BucketAccessStatus {
    // ReadyToUse indicates the successful grant of privileges to access the bucket.
    ReadyToUse bool

    // AccountID is the unique ID for the account in the OSP. It will be populated by the COSI
    // sidecar once access has been successfully granted.
    AccountID string

    // AccessedBucketName is the name of the Bucket resource that the access corresponds to. This is
    // filled in by the Controller based on the BucketClaim so that the Sidecar knows what Bucket
    // to allow access to for this BucketAccess.
    // TODO: will have to update this to a list when 1-access:many-buckets support is added
    AccessedBucketName string

    // DriverName holds a copy of the BucketAccessClass driver name at the time of BucketAccess
    // provisioning. This is kept to ensure the BucketAccess can be modified/deleted even after
    // BucketAccessClass mutation/deletion.
    DriverName string

    // Parameters holds a copy of the BucketAccessClass opaque parameters at the time of
    // BucketAccess provisioning. These parameters are kept to ensure the BucketAccess can be
    // modified/deleted even after BucketAccessClass mutation/deletion.
    Parameters map[string]string

    // Error holds the most recent error message, with a timestamp.
    // This is cleared when provisioning is successful.
    Error *TimestampedError
  }
```

The `accessSecretName` is the name of the Kubernetes Secret that COSI will generate containing endpoint, credentials, and other information needed to access the OSP bucket. The same Secret can be referenced by Pods to access the OSP bucket.

A User or Administrator who needs to make modifications to an OSP access underlying a BucketAccess likely needs to do so because the OSP driver is missing features for managing a desired configuration and not because static access provisioning is intrinsically necessary. Static BucketAccess provisioning would be possible to implement, but doing so would be complicated. COSI, OSP drivers, Admins, and/or Users could (and probably would) each have different expectations and desires about how to handle corner cases in static access policies, and COSI couldn't reasonably enforce consistent handling of those expectations. Design and dev work would be large, and risk for bugs would be high. Because of this COSI design decision, some Admins will undoubtedly make modifications to OSP accesses using manual OSP access tools. These manual modifications will be done at the Admin's risk. The OSP driver is not guaranteed to preserve such modifications, and the OSP driver may get stuck in a position where it doesn't know how to continue managing the BucketAccess.

#### BucketAccessClass

The BucketAccessClass represents a set of common properties shared by multiple BucketAccesses. It is used to specify policies for creating access credentials, and also for configuring driver-specific access parameters.

Resource for configuring common properties for multiple BucketAccesses. BucketAccessClass is cluster-scoped.

```go
BucketAccessClass {
  TypeMeta
  ObjectMeta

  Spec BucketAccessClassSpec {
    // DriverName is the name of driver associated with this BucketAccess
    // +required
    DriverName string

    // Parameters is an opaque map for passing in configuration to a driver
    // for granting access to a bucket
    // +optional
    Parameters map[string]string
  }
}
```

#### BucketAccess Secret data

BucketAccess secrets contain information about the OSP bucket as well as credentials for accessing the OSP bucket.

All buckets have this top-level data:

- `COSI_PROTOCOL`: The protocol for accessing the bucket. (`S3`, `Azure`, `GCS`)

##### S3

S3 bucket info:

- `COSI_S3_ENDPOINT`: S3 endpoint URL, e.g., `https://s3.amazonaws.com`
- `COSI_S3_BUCKET_ID`: S3 bucket ID (must be client-facing OSP bucket ID)
- `COSI_S3_REGION`: S3 region, e.g., `us-west-1`
- `COSI_S3_SIGNATURE_VERSION`: signature version for signing all S3 requests

S3 credential info:

- `COSI_S3_ACCESS_KEY_ID`: S3 access key ID, e.g., `AKIAIOSFODNN7EXAMPLE`
- `COSI_S3_ACCESS_SECRET_KEY`: S3 access secret key, e.g., `wJalrXUtnFEMI/K...`

##### Azure

Azure bucket info:

- `COSI_AZURE_STORAGE_ACCOUNT`: the ID of the Azure storage account

Azure credential info:

- `COSI_AZURE_ACCESS_TOKEN`: Azure access token. Note that the Azure spec includes the resource URI as well as token in its definition. https://learn.microsoft.com/en-us/azure/storage/common/media/storage-sas-overview/sas-storage-uri.svg
- `COSI_AZURE_EXPIRY_TIMESTAMP`: Can be empty if unset. Otherwise, date+time in ISO 8601 format.

##### GCS (Google Cloud Storage)

Note that COSI maintainership currently lacks GCS input or experience as of v1alpha2. This spec attempts to add the fields that are likely needed, with the expectation that some fields may be missing.

GCS bucket info:

- `COSI_GCS_PROJECT_ID`: GCS project ID
- `COSI_GCS_BUCKET_NAME`: GCS bucket name (must be client-facing OSP bucket ID)
- `COSI_GCS_SERVICE_ACCOUNT`: GCS service account name
- `COSI_GCS_PRIVATE_KEY_NAME`: GCS private key name

GCS credential info:

- `COSI_GCS_ACCESS_ID`: HMAC access ID
- `COSI_GCS_ACCESS_SECRET`: HMAC secret

### COSI Driver

A component that runs alongside COSI Sidecar and satisfies the COSI gRPC API specification. Sidecar and driver work together to orchestrate changes in the OSP. The driver acts as a gRPC server to the COSI Sidecar. Each COSI driver is identified by a unique ID.

The sidecar uses the unique ID to direct requests to the appropriate driver. Multiple instances of drivers with the same ID will be added into a group, and only one of them will act as the leader at any given time.

#### COSI Driver gRPC API

##### DriverGetInfo

This gRPC call responds with the name of the driver. The name is used to identify which resources the driver should manage.

```go
DriverGetInfo(DriverGetInfoRequest) DriverGetInfoResponse

DriverGetInfoRequest{}

DriverGetInfoResponse{
  "name": "<driver name>" // e.g., "s3.amazonaws.com"
  "supportedProtocols": [ // one or more of:
    "S3",
    "Azure",
    "GCS"
  ]
}
```

##### DriverCreateBucket

This gRPC call creates a bucket in the OSP, and returns information about the new bucket. This API must be idempotent. This call does not apply to statically-provisioned Buckets.

COSI uses `Request.name` as an idempotency key. The driver should ensure that multiple DriverCreateBucket calls for the same name do not result in more than one OSP backend bucket being provisioned corresponding to that name. Using or appending random identifiers can lead to multiple unused buckets being created in the OSP backend in the event of timing-related driver/sidecar failures or restarts.

The Sidecar uses the name of the Bucket resource as the input value for `Request.name`. This will be `bc-<BucketClaim.UID>` for dynamically-provisioned Buckets -- statistically likely to be globally unique even between multiple Kubernetes clusters.

Input `parameters` are the opaque parameters copied from the Bucket (originating from BucketClass). Drivers can use these parameters to configure OSP bucket features based on the Administrator's BucketClass configuration.

The returned `bucketID` should be a unique identifier for the OSP bucket known to the driver. This value will be used by COSI to make all subsequent calls related to this bucket, so the driver must be able to correlate `bucketID` to the OSP backend bucket. It is easiest for drivers to use the request `name` field as both `bucketID` and as the OSP backend bucket identifier, but this is not strictly required.

```go
DriverCreateBucket(DriverCreateBucketRequest) DriverCreateBucketResponse

DriverCreateBucketRequest{
  "name": "<Bucket.name>", // will be "bc-<BucketClaim.UID>" for dynamically-provisioned Buckets

  "parameters": { // copied from Bucket.parameters
    "<key>": "<value>"
    // ...
  }
}

DriverCreateBucketResponse{
  Bucket bucket
}

Bucket {
  "bucketID": "<ID returned by driver>" // will be applied to Bucket.status.bucketID
  "protocols": { // all protocols supported by the bucket, with bucket info
    "S3": { // at least one protocol must be returned, or provisioning will be considered failed
      // returned bucket info is applied to Bucket status and should not contain any sensitive data
      // info fields are optional, in case info is considered sensitive by driver implementers
      "bucketName": "<bucket ID in S3 backend>",
      "region": "<bucket region>",
      "endpoint": "<endpoint where bucket can be accessed>"
    },
    "Azure": {
      "storageAccount": "<storage account name>"
    }
  }
}

```

Note: the driver is expected to return the well-known gRPC return code `AlreadyExists` when the bucket already exists but is incompatible with the request.

##### DriverGetExistingBucket

This gRPC call is used to get details about a statically-provisioned bucket that should already exist in the OSP backend. This call does not apply to dynamically-provisioned Buckets.

`existingBucketID` used for input is taken from `Bucket.existingBucketID`.

```go
DriverGetExistingBucket(DriverGetExistingBucketRequest) DriverGetExistingBucketResponse

DriverGetExistingBucketRequest {
  "existingBucketID": "<name of statically-provisioned bucket>" // e.g., "my-static-bucket"
}

DriverGetExistingBucketResponse {
  Bucket bucket // same Bucket response defined in DriverCreateBucket
}
```

The returned `bucketID` should be a unique identifier for the OSP bucket known to the driver. This value will be used by COSI to make all subsequent calls related to this bucket, so the driver must be able to correlate `bucketID` to the OSP backend bucket. It is easiest for drivers to use the request `existingBucketID` field the `bucketID`, but this is not strictly required.

##### DriverGrantBucketAccess

This gRPC call creates a set of access credentials for a bucket. This API must be idempotent.

`bucketID` used for input is the same ID returned by the driver in [DriverCreateBucket](#drivercreatebucket) (dynamically-provisioned) or [DriverGetExistingBucket](#drivergetexistingbucket) (statically-provisioned).

This `accountName` field is the concatenation of the characters `ba-` (short for BucketAccess) and the BucketAccess UID. It is used as the idempotency key for requests to the drivers regarding a particular BA. The driver should ensure that multiple DriverGrantBucketAccess calls for the same `accountName` do not result in more than one OSP backend bucket access being provisioned corresponding to that name. Using or appending random identifiers can lead to multiple unused bucket accesses being created in the OSP backend in the event of timing-related driver/sidecar failures or restarts.

Input `accessParameters` are the opaque parameters copied from the BucketAccessClass. Drivers can use these parameters to configure OSP bucket access features based on the Administrator's BucketAccessClass configuration.

The returned `accountID` should be a unique identifier for the account in the OSP. This value will be included in all subsequent calls to the driver for changes to the BucketAccess. As `bucketID` is to Bucket, `accountID` is to BucketAccess.

Returned `credentials` will be transformed into [BucketAccess secret data](#bucketaccess-secret-data).

```go
DriverGrantBucketAccess(DriverGrantBucketAccessRequest) DriverGrantBucketAccessResponse

DriverGrantBucketAccessRequest{
  "bucketID": "<Bucket.status.bucketID>", // e.g., "ba-<BucketClaim.UID>", "my-static-bucket"
  "accountName": "ba-<BucketAccess.UID>"
  "protocol": "<protocol>", // e.g., "S3", copied from BucketAccess.spec.protocol
  "parameters": { // copied from BucketAccess.status.parameters
      "key": "value",
      // ...
  }
}

DriverGrantBucketAccessResponse {
  "accountID": "<ID returned by driver>", // will be applied to BucketAccess.status.accountID
  "credentials": { // credentials info (authn/authx)
    "S3": { // must match input protocol, and if required fields are missing, will be treated as provisioning failure
      "accessKeyID": "<s3 access key id>", // e.g., "AKIAODNN7EXAMPLE"
      "accessSecretKey": "<s3 access secret key>" // e.g., "wJaUtnFEMI/K..."
  }
}
```

Important driver return codes:
- Driver is expected to return the well-known gRPC return code `AlreadyExists` when the bucket already exists but is incompatible with the request.
- Driver is expected to return `InvalidArgument` when the authentication type is not supported

##### DriverDeleteBucket

This gRPC call deletes a bucket in the OSP.

```go
DriverDeleteBucket(DriverDeleteBucketRequest) DriverDeleteBucketResponse

DriverDeleteBucketRequest{
  bucketID: "<Bucket.status.bucketID>" // e.g., "ba-<BucketClaim.UID>", "my-static-bucket"
  parameters: { // copied from Bucket.parameters
    "key": "value"
    // ...
  }
}

DriverDeleteBucketResponse{} // empty with return code
```

Expected to be idempotent and return OK when OSP bucket does not exist.

##### DriverRevokeBucketAccess

This gRPC call revokes access granted to a particular account.

```go
DriverRevokeBucketAccess(DriverRevokeBucketAccessRequest) DriverRevokeBucketAccessResponse

DriverRevokeBucketAccessRequest{
  bucketID: "<Bucket.status.bucketID>"
  accountID: "<BucketAccess.status.accountID>"
  parameters: { // copied from BucketAccess.status.parameters
    "key": "value"
    // ...
  }
}

DriverRevokeBucketAccessResponse{} // empty with return code
```

Expected to be idempotent and return OK when OSP bucket access does not exist.

### Test Plan

- Unit tests will cover the functionality of the controllers.
- Unit tests will cover the new APIs.
- An end-to-end test suite will cover testing all the components together.
- Component tests will cover testing each controller in a blackbox fashion.
- Tests need to cover both correctly and incorrectly configured cases.

### Graduation Criteria

#### Alpha

- API is reviewed and accepted
- Design COSI APIs to support Greenfield, Green/Brown Field, Brownfield and Static Driverless provisioning
- Design COSI APIs to support authentication using access/secret keys
- Evaluate gaps, update KEP and conduct reviews for all design changes
- Develop unit test cases to demonstrate that the above mentioned use cases work correctly

#### Beta

- Consider using a typed configuration for Bucket properties (parameter fields in Bucket, BucketClass, BucketAccess, BucketAccessClass)
- Implement all COSI components to support agreed design.
- Design and implement support for sharing buckets across namespaces.
- Design and implement quotas/restrictions for Buckets and BucketAccess.
- Basic unit and e2e tests as outlined in the test plan.
- Metrics for bucket create and delete, and granting and revoking bucket access.
- Metrics in provisioner for bucket create and delete, and granting and revoking bucket access.

#### GA

- Stress tests to iron out possible race conditions in the controllers.
- Users deployed in production and have gone through at least one K8s upgrade.

### Upgrade / Downgrade Strategy

No Kubernetes changes are required on upgrade to maintain previous behavior.

The COSI resource APIs have breaking changes from v1alpha1 to v1alpha2, and migrations between versions are not automatically supported. The static provisioning workflow can be used to migrate existing v1alpha1 Buckets to v1alpha2.

### Version Skew Strategy

COSI is out-of-tree, so version skew strategy is N/A

## Alternatives Considered

This KEP has had a long journey and many revisions. Here we capture the main alternatives and the reasons why we decided on a different solution.

### Automatically mount buckets to Pods

Early iterations of the COSI v1alpha1 spec tried to design a system that could mount bucket information to user Pods automatically, to serve as an analog to Pod PVC mounts. Each Pod application may have different means of and needs for connecting to object storage buckets, so there is not a clear way for this to be implemented in a way that could accommodate all drivers and all users. This topic can be revisited in the future after COSI maintainers can get more user and driver developer feedback if needed.

### Encode BucketAccess connection information in a JSON blob

In COSI v1alpha1, BucketAccess connection/authentication information was encoded in a JSON blob in the BucketAccess's chosen Secret. This was intended to be mounted to user Pods as a file, which is considered generally safer than mounting via environment variables. However, several driver implementers gave feedback that they were unable to process the JSON blob file into forms suitable for their applications. They instead requested that each entry in the JSON blob be encoded as individual Secret data fields that could be loaded as files or environment variables based on the needs of their individual applications. COSI v1alpha2 deprecated the JSON blob file in favor of the more flexible approach of encoding each field in a separate Secret data key. Both forms are not supported in order to keep development and usage consistent.

### Cross-resource protection finalizers

In the v1alpha2 design cycle, COSI received feedback from sig-storage regarding the finalizers used for PV/PVC protection (`.../pv-protection` and `.../pvc-protection`). In the real world, administrators/users have often removed the PV/PVC protection finalizers when issues are encountered, often leading to broken system states or incorrect cleanup behavior for PV/PVC drivers. To avoid these pitfalls, the COSI project was cautioned to design a system that doesn't rely on finalizers for managing order-of-operations concerns between different COSI resources. COSI will use finalizers as needed but will attempt to avoid them for cross-resource protections. COSI v1alpha2 has used the volume snapshot design as inspiration and is using annotations to help inform resource references and bindings.

### Bucket creation annotation

Volume Snapshotter uses an annotation to track snapshot creation, which can take a long time. This mechanism helps identify and prevent orphan snapshots from being created. During the v1alpha2 KEP planning, COSI developers don't believe such a mechanism is necessary for COSI. However, should such a protection be needed, COSI may implement it as an implementation detail outside of the KEP process.

### BucketClass field on Bucket resource

Persistent Volumes track the StorageClass on the PV object. According to sig-storage experts, this is used to control binding during static provisioning. For COSI, the intent is that the Bucket should have a copy of any relevant BucketClass parameters from the time when the OSP bucket was provisioned. For static provisioning, only the admin knows those parameters, and they are expected to copy them to the Bucket. As of COSI v1alpha2, the plan is to use the Bucket's BucketClaim reference as the sole binding control mechanism, and not to require BucketClass to be tracked on the Bucket object. This will be revisited if someone provides evidence that the field is needed/useful.

### Updating BucketAccess Secrets

COSI v1alpha1 specified that if a BucketAccess Secret already exists, then it is assumed that credentials have already been generated, and the Secret is not overridden. In practice, this disallows Administrators from rotating credentials internally. Because other important bucket connection details are present in the Secret (e.g., endpoints), this also makes it impossible for systems to change hosting locations over time. COSI must be able to update the Secret to reflect the OSP's most up-to-date information.

### IAM authentication type

COSI v1alpha1 included specifications for supporting two authentication types:
- KEY: standard key-based authentication
- IAM: AWS IAM-style authentication tied to Kubernetes ServiceAccounts

COSI's v1alpha1 spec and implementation did not set any expectations, designs, or implementation in place to manage IAM or ServiceAccounts. The selection and configuration were merely passed to drivers to do with as desired. The implied expectation was that drivers would be able to create or modify Kubernetes ServiceAccounts, with unclear boundaries and guidelines. In COSI v1alpha2, COSI removed authentication options entirely, assuming key-based authentication is the standard. This removes a large area of ambiguity from the COSI spec and also removes a potentially-troublesome expectation that COSI drivers have permissions to modify Kubernetes ServiceAccounts.

Users of the ObjectBucketClaim precursor to COSI only have key-based auth options, and users have not requested ServiceAccount support. If this is any prediction for COSI, it is likely to be some time before COSI users ask for ServiceAccount support. COSI decided to defer possible designs for this feature to the future, where they can be designed with more clarity and safety.

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

<!--
Pick one of these and delete the rest.
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [X] Other
  - Describe the mechanism: Create Deployment and DaemonSet resources (along with supporting secrets, configmaps etc.) for the three controllers that COSI requires
  - Will enabling / disabling the feature require downtime of the control
    plane? No
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No

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

Yes. Delete the resources created when installing COSI

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

All COSI components are out-of-tree. We don't need extra feature gate testing that would be needed
for in-tree features. COSI API is CRD-related and does not require core API conversion.

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

COSI's controllers don't impact the data path of Pods using already-running object storage
applications.

However, if upgrade fails resulting in COSI unavailability, users will be unable to create new
Buckets or Bucket Accesses. Other features that alter existing buckets also might not be available
during rollout/rollback failure.

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

The API load of COSI components will be a factor of the number of buckets being managed and the number of bucket-accessors for these buckets. Essentially O(num-buckets * num-bucket-access). There is no per-node or per-namespace load by COSI.

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

Yes. Containers requesting Buckets will not start until Buckets have been provisioned. This is similar to dynamic volume provisioning

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

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

We need Linux VMs for e2e testing in CI.
