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
  - [COSI Architecture](#cosi-architecture)
  - [COSI API Overview](#cosi-api-overview)
  - [COSI Object Lifecycle](#cosi-object-lifecycle)
  - [Usability](#usability)
    - [User Self-Service](#user-self-service)
    - [Mutating Buckets](#mutating-buckets)
  - [Control Flows](#control-flows)
    - [Installing the COSI System](#installing-the-cosi-system)
    - [Creating a Bucket](#creating-a-bucket)
    - [Generating Bucket Access Credentials](#generating-bucket-access-credentials)
    - [Attaching Bucket Information to Pods](#attaching-bucket-information-to-pods)
    - [Accessing an Existing OSP Bucket](#accessing-an-existing-osp-bucket)
  - [Deleting a BucketAccess](#deleting-a-bucketaccess)
  - [Deleting a Bucket](#deleting-a-bucket)
  - [Sharing Buckets](#sharing-buckets)
  - [COSI API Reference](#cosi-api-reference)
    - [Bucket](#bucket)
    - [BucketClaim](#bucketclaim)
    - [BucketClass](#bucketclass)
    - [BucketAccess](#bucketaccess)
    - [BucketAccessClass](#bucketaccessclass)
    - [BucketAccess secret data](#bucketaccess-secret-data)
  - [COSI Driver](#cosi-driver)
    - [COSI Driver gRPC API](#cosi-driver-grpc-api)
      - [DriverGetInfo](#drivergetinfo)
      - [DriverCreateBucket](#drivercreatebucket)
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
  - [Add Bucket Instance Name to BucketAccessClass (brownfield)](#add-bucket-instance-name-to-bucketaccessclass-brownfield)
    - [Motivation](#motivation-1)
    - [Problems](#problems)
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

The following resources are managed by a User. All are namespace-scoped.
Each is created with a reference to a corresponding class object.

- BucketClaim -\> BucketClass
- BucketAccess -\> BucketClaim, BucketAccessClass

For a greenfield BucketClaim/Bucket resources created by a User, the COSI controller responds by
creating the an intermediate Bucket object as shown below.

- BucketClaim -\> new(Bucket)

Notes:

- There are **NO** cycles in the relationship graph of the above mentioned API objects.
- Mutations are not supported in the API.
- Class objects have a lifecycle independent of objects that reference them.
  - BucketClaim, BucketAccess, and Bucket must have all necessary class parameters copied to them
    during provisioning to allow themselves to be deleted if class objects have been mutated/deleted.

### Usability

#### User Self-Service

User self-service is made possible using BucketClaim and BucketAccess resources (namespace-scoped).
Users do not require admin privileges to create, modify, and/or delete them.

An admin is responsible for creating class objects (BucketClass, BucketAccessClass) which configure
OSP-specific storage parameters. The creation of COSI class objects is deliberately analogous to
creation and management of Kubernetes StorageClasses for PVCs. This is a well-understood pattern,
and relying on familiarity will aid COSI users.

Importing a bucket that already exists in an OSP backend (a brownfield bucket) requires special
permissions because its lifecycle is not managed by COSI. Special care needs to be taken to prevent
unintended clones, accidental deletion, and other mishaps that could affect the OSP bucket.
For instance, setting the deletion policy to Delete for a brownfield bucket should be disallowed.
Admins are thus responsible for creating Bucket imports for brownfield buckets.

#### Mutating Buckets

As of the current design of COSI, mutating bucket properties is not supported. However, the current
design does not prevent us from supporting it in the future. Mutable properties include encryption,
object lifecycle configuration, replication configuration, etc. These properties will be supported
in future versions along with the capability to mutate them.

### Control Flows

This section outlines the scenarios that COSI personas will initiate, and for what purpose. Each
scenario includes enough detail to express the important interaction requirements between personas
and the COSI system, and between COSI components. This section avoids unnecessarily naming specific
API elements so as not to confuse complex system interaction requirements with specific
implementation/spec details.

#### Installing the COSI System

Admin installs the COSI system and driver(s) to allow User self-service.

1. Assume that a Vendor has already created a COSI driver
2. Admin deploys the COSI controller
3. Admin deploys vendor COSI driver
4. Admin creates BucketClass and AccessClass configuring COSI and vendor driver features

#### Creating a Bucket

User self-provisions a bucket to store their workload's data.

1. Admin allows User to use BucketClass
2. User creates BucketClaim that uses BucketClass
3. COSI controller observes BucketClaim
   1. Controller applies a finalizer to the BucketClaim to prevent object deletion before cleanup
   2. Controller then creates a new intermediate Bucket resource
   3. Controller applies a finalizer to intermediate Bucket to prevent it from being deleted before
      the BucketClaim is deleted
   4. Controller copies BucketClaim and BucketClass parameters to Bucket
   5. Bucket status `BucketReady` is false initially, indicating bucket is not yet provisioned in OSP
   6. Controller is now waiting for the intermediate Bucket to be reconciled by COSI sidecar before
      continuing with BucketClaim reconciliation
4. COSI sidecar detects the intermediate Bucket resource
   1. If the Bucket's driver matches the sidecar's driver, continue
   2. Sidecar calls the COSI driver via gRPC to provision the OSP bucket
   3. To ensure idempotency, gRPC call must include enough information to identify repeat requests
      for the same Bucket
      <!-- TODO: discuss this; namespace and name are not strictly enough info b/c a user could access
      the same OSP from multiple k8s clusters. COSI system will need some kind of optional specified ID? -->
5. If OSP returns provision fail, COSI sidecar reports error to Bucket status and retries gRPC call
6. When OSP returns provision success, COSI sidecar updates Bucket status `BucketReady` to true
7. Controller detects that the Bucket is provisioned successfully (`BucketReady`==true)
   1. Controller finishes BucketClaim reconciliation processing/validation as needed
   2. Controller applies `BucketReady` true status to BucketClaim, finishing BucketClaim reconcile

The mechanism by which the Controller waits for the intermediate Bucket to be provisioned in the
middle of BucketClaim reconciliation is not specified here. The important behavior (waiting) is
defined, and the logic coordinating the init-wait-finish reconcile process is left as an
implementation detail.

#### Generating Bucket Access Credentials

User requests access to a BucketClaim for their workload's application.

For the purposes of this part of the COSI spec, access credentials can refer to any supported access
control mechanism. This could be username+password, an IAM-style service account, or other
protocol-specific access terminology.

Access credentials are represented by BucketAccess objects. The separation of BucketClaim and
BucketAccess is a reflection of the usage pattern of object storage, where buckets are always
accessed over the network, and all access is subject to authentication and authorization. The
lifecycle of a bucket and its access are not tightly coupled.

1. Admin allows User to use BucketAccessClass
2. User creates BucketAccess that uses BucketAccessClass and references BucketClaim
   1. User specifies a Kubernetes Secret name into which BucketAccess information will be stored
      upon successful access provision
3. COSI sidecar detects the BucketAccess resource
   1. If the BucketAccess's driver matches the sidecar's driver, continue
   2. If the BucketClaim is ready, continue
   3. If the BucketClaim has a deletion timestamp, fail with error
   4. Sidecar applies a finalizer to the BucketClaim to prevent the bucket from being deleted before
      access is no longer needed
   5. Sidecar applies a finalizer to the BucketAccess to prevent the object from being deleted
      separately from the OSP access credentials
   6. Sidecar calls the COSI driver via gRPC to generate unique access credentials for the
      BucketClaim's OSP bucket
   7. To ensure idempotency, gRPC call must include enough information to identify repeat requests
      for the same BucketAccess
4. If OSP returns provision fail, COSI sidecar reports error to BucketAccess status and retries gRPC call
5. When OSP returns provision success, COSI sidecar:
   1. Updates the BucketAccess secret with all info needed to access the OSP bucket
   2. Applies a finalizer to the secret to ensure it isn't deleted before the BucketAccess is deleted
   3. Updates BucketAccess status `AccessGranted` to true

#### Attaching Bucket Information to Pods

User attaches bucket information to their workload's application pod.

1. User references the BucketAccess secret using the pod volume downward API in their pod spec
2. User configures pod container(s) to mount secret data items as desired

<!-- TODO: link to location where bucket access secret data is specified -->

The BucketAccess secret can be provided to the pod using any Kubernetes {secret -> pod} attachment
mechanism. This naturally includes mounting data into environment variables and files. Mounting
credential data into files is slightly more secure than environment variables and is thus
recommended. However, each application has different requirements, and some may require environment
variables for configuring access.

#### Accessing an Existing OSP Bucket

User needs access to a bucket that already exists in an OSP object store.

In early COSI feedback and in other object storage self-service frameworks, users commonly want
access to OSP buckets that are preexisting. However, giving end users unfettered access to OSP
storage would allow them to easily gain access to sensitive data they may not be intended to access.
To resolve this, the Admin is expected to allow access to existing OSP buckets.

1. Admin creates a (non-intermediate) Bucket object that represents the existing OSP bucket
   1. Admin must specify the existing OSP bucket ID in the Bucket spec
   2. Admin must ensure that the bucket binds only to a specific BucketClaim by specifying the
      BucketClaim namespace and name
      <!-- TODO: This seems unnecessarily restricted. how can we allow RBAC to dictate which Users
      can create BucketClaims for this bucket? -->
2. User creates BucketClaim specified above
3. COSI controller observes BucketClaim, then validates it against the Bucket
   1. Controller applies finalizer to Bucket (and claim?) <!-- TODO: discuss -->
4. <!-- TODO: make sure spec is clarifying how COSI controller is expected to validate and set status on the BucketClaim -->

### Deleting a BucketAccess

User deletes a BucketAccess they no longer need.

1. User deletes BucketAccess object
2. COSI sidecar detects BucketAccess resource's deletion timestamp
   1. If there are no unknown finalizers on the BucketAccess, continue
   2. Sidecar deletes the BucketAccess secret containing connection details (remove finalizer first)
   3. Sidecar calls the COSI driver via gRPC to revoke the associated access credentials
3. If OSP returns deprovision fail, COSI sidecar reports error to BucketAccess status and retries gRPC call
4. When OSP returns deprovision success, COSI sidecar:
   1. Removes the BucketAccess's finalizer from BucketClaim
   2. <!-- TODO: Is it necessary to set AccessGranted=false here or before? --->
   3. Removes the finalizer from BucketAccess, allowing Kubernetes to clean up the object

This will remove the BucketAccess and the BA secret without checking if the secret is mounted to any
pods.

All information necessary for COSI and the OSP to delete the bucket must be encoded on the objects
and in gRPC calls to ensure that BucketAccessClass isn't required for deletion.

### Deleting a Bucket

User deletes a BucketClaim they no longer need.

User self-provisions a bucket to store their workload's data.

<!-- 1. Admin allows User to use BucketClass
2. User creates BucketClaim that uses BucketClass
3. COSI controller observes BucketClaim
   1. Controller applies a finalizer to the BucketClaim to prevent object deletion before cleanup
   2. Controller then creates a new intermediate Bucket resource
   3. Controller applies a finalizer to intermediate Bucket to prevent it from being deleted before
      the BucketClaim is deleted
   4. Controller copies BucketClaim and BucketClass parameters to Bucket
   5. Bucket status `BucketReady` is false initially, indicating bucket is not yet provisioned in OSP
4. COSI sidecar detects the intermediate Bucket resource
   1. If the Bucket's driver matches the sidecar's driver, continue
   2. Sidecar calls the COSI driver via gRPC to provision the OSP bucket
   3. To ensure idempotency, gRPC call must include enough information to identify repeat requests
      for the same Bucket
5. If OSP returns provision fail, COSI sidecar reports error to Bucket status and retries gRPC call
6. When OSP returns provision success, COSI sidecar updates Bucket status `BucketReady` to true -->

1. User deletes BucketClaim object
2. COSI controller detects BucketClaim resource's deletion timestamp
   1. Controller

- A Bucket created by COSI as a result of a BucketClaim can deleted by deleting the BucketClaim
- A Bucket created outside of COSI, once bound, can be deleted by deleting the BucketClaim to which
  it is bound
- A Bucket created outside of COSI, unless it is bound to a particular BucketClaim, cannot be
  deleted by users from any particular namespace. Privileged users can however delete the Bucket
  object at their discretion.

Once a delete has been issued to a bucket, no new BucketAccesses can be created for it. Buckets
having valid BucketAccesses (Buckets in use) will not be deleted until all the BucketAccesses are
cleaned up.

Buckets can be created with one of two deletion policies:
- Retain
- Delete

When the deletion policy is Retain, then the underlying bucket is not cleaned up when the Bucket
object is deleted. When the deletion policy is Delete, then the underlying bucket is cleaned up when
the Bucket object is deleted.

Only when all accessors (BucketAccesses) of the Bucket are deleted, is the Bucket itself cleaned up.
There is a finalizer on the Bucket that prevents it from being deleted until all the accessors are
done using it.

When a user deletes a BucketAccess, the corresponding secret/serviceaccount are also deleted. If a
pod has that secret mounted when delete is called, then a finalizer on the secret will prevent it
from being deleted. Instead, the deletionTimestamp will be set on the secret. In this way, access to
a Bucket is preserved until the application pod dies.

When an admin deletes any of the class objects, it does not affect existing Buckets as fields from
the class objects are copied into the Buckets during creation.

If a Bucket is manually deleted by an admin, then a finalizer on the Bucket prevents it from being
deleted until the binding BucketClaim is deleted.

### Sharing Buckets

This section describes the current design for sharing buckets with other namespaces. As of the current milestone (alpha) of COSI, any bucket created in one namespace cannot be accessed in another namespace. i.e. no bucket sharing is possible. In future versions, a namespace level access control will be enforced, and buckets will be constrained to particular namespaces using selectors. Admins will be able to control which namespaces can access which buckets using namespace selectors.

### COSI API Reference

<!-- TODO: we don't have a `Protocol` struct definition in the kep
           in the API it's just a string "S3, Azure, GCP" -->

<!-- TODO: clarify how to reuse access for multiple buckets
           reuse credential secret name or service account name -->

#### Bucket

Resource to represent a Bucket in an OSP. Bucket is cluster-scoped.

```go
Bucket {
  TypeMeta
  ObjectMeta

  Spec BucketSpec {
    // DriverName is the name of driver associated with this bucket.
    DriverName string

    // DeletionPolicy is used to specify how COSI should handle deletion of the OSP bucket when the
    // Bucket object is deleted.
    // There are 3 possible values:
    //  - Retain: Indicates that the bucket should not be deleted from the OSP (default)
    //  - Delete: Indicates that the bucket should be deleted from the OSP
    // +optional
    DeletionPolicy DeletionPolicy

    // ^^^ TODO: discuss changing this to ReclaimPolicy as with PVCs

    // Name of the BucketClass specified in the BucketRequest
    BucketClassName  string

    // Name of the BucketClaim that resulted in the creation of this Bucket.
    // In case the Bucket object was created manually, then this should refer to the BucketClaim
    // with which this Bucket should be bound.
    BucketClaim corev1.ObjectReference

    // Protocols are the set of data APIs this bucket is expected to support.
    // The possible values for protocol are:
    //  - S3: Indicates Amazon S3 protocol
    //  - Azure: Indicates Microsoft Azure BlobStore protocol
    //  - GCS: Indicates Google Cloud Storage protocol
    Protocols []Protocol

    // Parameters is an opaque map for passing in configuration to a driver for creating the bucket.
    // +optional
    Parameters map[string]string

    // ExistingBucketID is the unique id of the bucket in the OSP. This field should be used to
    // specify a bucket that has been created outside of COSI.
    // This field will be empty when the Bucket is dynamically provisioned by COSI.
    // +optional
    ExistingBucketID string
  }

  Status BucketStatus {
    // BucketReady is a boolean condition to reflect the successful creation of a bucket.
    BucketReady bool

    // ^^^ TODO: consider renaming to BucketProvisioned or BucketCreated ? Ready might imply that
    // it's accessible by HTTP, which COSI doesn't test

    // BucketID is the unique ID of the bucket in the OSP. This field will be populated by COSI once
    // the ID in the OSP is known.
    BucketID string

    // ErrorMessage is the most recent error message. This cleared when provisioning is successful.
    ErrorMessage string

    // TODO: PVs have this. Should we do something similar?
    // LastPhaseTransitionTime
    LastPhaseTransitionTime
  }
}
```

#### BucketClaim

A claim to create Bucket. BucketClaim is namespace-scoped.

```go
BucketClaim {
  TypeMeta
  ObjectMeta

  Spec BucketClaimSpec {
    // Name of the BucketClass.
    BucketClassName string

    // Protocols are the set of data API this bucket is required to support.
    // The possible values for protocol are:
    //  - S3: Indicates Amazon S3 protocol
    //  - Azure: Indicates Microsoft Azure BlobStore protocol
    //  - GCS: Indicates Google Cloud Storage protocol
    Protocols []Protocol

    // Name of a bucket object that was manually created to import a bucket created outside of COSI.
    // If unspecified, then a new Bucket will be dynamically provisioned.
    // +optional
    ExistingBucketName string
  }

  Status BucketClaimStatus {
    // BucketReady indicates that the bucket is ready for consumption by workloads.
    BucketReady bool

    // BucketName is the name of the provisioned Bucket in response to this BucketClaim. It is
    // generated and set by the COSI controller before making the creation request to the OSP backend.
    BucketName string

    // ErrorMessage is the most recent error message. This cleared when provisioning is successful.
    ErrorMessage string
  }
```

#### BucketClass

Resouce for configuring common properties for multiple Buckets. BucketClass is cluster-scoped.

```go
BucketClass {
  TypeMeta
  ObjectMeta

  // DriverName is the name of driver associated with this bucket.
  DriverName string

  // DeletionPolicy is used to specify how COSI should handle deletion of OSP buckets when Bucket
  // objects created from this BucketClass are deleted.
  // There are 3 possible values:
  //  - Retain: Indicates that the bucket should not be deleted from the OSP (default)
  //  - Delete: Indicates that the bucket should be deleted from the OSP
  DeletionPolicy DeletionPolicy

  // Parameters is an opaque map for passing in configuration to a driver for creating the bucket.
  // +optional
  Parameters map[string]string
}
```

#### BucketAccess



The BucketAccessClass represents a set of common properties shared by multiple BucketAccesses. It is used to specify policies for creating access credentials, and also for configuring driver-specific access parameters. More information about BucketAccessClass is [here](#bucketaccessclass)

The BucketAccess is used to request access to a bucket. It contains fields for choosing the Bucket for which the credentials will be generated, and also includes a bucketAccessClassName field, which in-turn contains configuration for authorizing users to access buckets. More information about BucketAccess is [here](#bucketaccess)

A resource to access a Bucket. BucketAccess is namespace-scoped.

```go
BucketAccess {
  TypeMeta
  ObjectMeta

  Spec BucketAccessSpec {
    // BucketClaimName is the name of the BucketClaim.
    BucketClaimName string

    // Protocol is the name of the Protocol that this access credential is supposed to support.
    // If left empty, it will choose the protocol supported by the bucket.
    // If the bucket supports multiple protocols, the end protocol is determined by the driver.
    // +optional
    Protocol Protocol

    // BucketAccessClassName is the name of the BucketAccessClass.
    BucketAccessClassName string

    // CredentialsSecretName is the name of the Kubernetes secret that COSI should populate with
    // access credentials. If a secret by this name already exists, then it is assumed that
    // credentials have already been generated, and the secret is not overridden.
    // This secret is deleted when the BucketAccess is deleted.
    CredentialsSecretName string

    // ^^^ TODO: this has more info than just credentials, like endpoint, etc. Let's rename this to "AccessSecretName" or similar. Discuss

    // ServiceAccountName is the name of the serviceAccount that COSI will map to the OSP service
    // account when IAM-style authentication is specified.
    // +optional
    ServiceAccountName string
  }

  Status BucketAccessStatus {
    // AccessGranted indicates the successful grant of privileges to access the bucket.
    AccessGranted bool

    // AccountID is the unique ID for the account in the OSP. It will be populated by the COSI
    // sidecar once access has been successfully granted.
    AccountID string

    // ErrorMessage is the most recent error message. This cleared when provisioning is successful.
    ErrorMessage string
  }
```

BucketAccessClass can be used to specify a authorization mechanism. It can be one of
- KEY  (__default__)
- IAM

The KEY based mechanism is where access and secret keys are generated to be provided to pods. IAM
style is where pods are implicitly granted access to buckets by means of a metadata service. IAM
style access provides greater control for the infra/cluster administrator to rotate secret tokens,
revoke access, change authorizations etc., which makes it more secure.

TODO: Add clarity that users choose a single protocol that is supported by their application.
If the backend driver supports it, the BucketAccess will be granted. If not, COSI should return
an error stating which protocols are available for the BAC.

The `credentialsSecretName` is the name of the secret that COSI will generate containing credentials to access the bucket. The same secret name has to be set in the podSpec as well as the projected secret volumes.

In case of IAM style authentication, along with the `credentialsSecretName`, `serviceAccountName` field must also be specified. This will map the specified serviceaccount to the appropriate service account in the OSP.

#### BucketAccessClass

Resource for configuring common properties for multiple BucketClaims. BucketAccessClass is a clustered resource

```go
BucketAccessClass {
  TypeMeta
  ObjectMeta

  // DriverName is the name of driver associated with
  // this BucketAccess
  DriverName string

  // AuthenticationType denotes the style of authentication.
  // It can be one of:
  //  - KEY: access, secret tokens based authentication
  //  - IAM: implicit authentication of pods to the OSP based on service account mappings
  AuthenticationType AuthenticationType

  // Parameters is an opaque map for passing in configuration to a driver
  // for granting access to a bucket
  // +optional
  Parameters map[string]string
}
```

#### BucketAccess secret data

All buckets have this data:

- `COSI_BUCKET_NAME`: Name of the Bucket object. <!-- TODO: Isn't it more important that this be the bucket as known by the OSP? -->
- `COSI_AUTHENTICATION_TYPE`: The authentication type for accessing the bucket.
- `COSI_PROTOCOLS`: The protocol for accessing the bucket. KEY/IAM
- `COSI_ENDPOINT`: The endpoint where the bucket can be accessed in the OSP.

S3 buckets have this additional data:

<!-- TODO: I changed ENDPOINT to S3_ENDPOINT some months ago but don't remember why; thoughts? -->
- `COSI_S3_ENDPOINT`: https://s3.amazonaws.com <!-- delete? -->
- `COSI_S3_ACCESS_KEY_ID`: e.g., AKIAIOSFODNN7EXAMPLE
- `COSI_S3_ACCESS_SECRET_KEY`: e.g., wJalrXUtnFEMI/K...
- `COSI_S3_REGION`: e.g., us-west-1

<!-- TODO: any other fields needed? -->

<!-- TODO: mention of when fields are not present. e.g., no keys when IAM enabled ??? -->

Azure buckets (blobs) have this additional data:

- TODO!

GCP buckets have this additional data:

- TODO!

### COSI Driver

A component that runs alongside COSI Sidecar and satisfies the COSI gRPC API specification. Sidecar
and driver work together to orchestrate changes in the OSP. The driver acts as a gRPC server to the
COSI Sidecar. Each COSI driver is identified by a unique ID.

The sidecar uses the unique ID to direct requests to the appropriate driver. Multiple instances of
drivers with the same ID will be added into a group, and only one of them will act as the leader at
any given time.

#### COSI Driver gRPC API

##### DriverGetInfo

This gRPC call responds with the name of the driver. The name is used to identify the appropriate driver for a given BucketRequest or BucketClaim.

```
    DriverGetInfo
    |------------------------------------------|       |---------------------------------------|
    | grpc DriverGetInfoRequest{}              | ===>  | DriverGetInfoResponse{                |
    |------------------------------------------|       |   "name": "s3.amazonaws.com"          |
                                                       | }                                     |
                                                       |---------------------------------------|
```

##### DriverCreateBucket

This gRPC call creates a bucket in the OSP, and returns information about the new bucket. This api must be idempotent. The input to this call is the name of the bucket and an opaque parameters field.

The returned `bucketID` should be a unique identifier for the bucket in the OSP. This value could be the name of the bucket too. This value will be used by COSI to make all subsequent calls related to this bucket.

TODO: note that the driver is expected to return the well-known gRPC return code `AlreadyExists`
when the bucket already exists but is incompatible with the request.

```
    DriverCreateBucket
    |------------------------------------------|       |-----------------------------------------------|
    | grpc DriverCreateBucketRequest{          | ===>  | DriverCreateBucketResponse{                   |
    |     "name": "br-$uuid",                  |       |   "bucketID": "br-$uuid",                     |
    |     "parameters": {                      |       |   "bucketInfo": {                             |
    |        "key": "value"                    |       |      "s3": {                                  |
    |     }                                    |       |        "bucketName": "br-$uuid",              |
    | }                                        |       |        "region": "us-west1",                  |
    | -----------------------------------------|       |        "endpoint": "s3.amazonaws.com"         |
                                                       |      }                                        |
                                                       |   }                                           |
                                                       | }                                             |
                                                       |-----------------------------------------------|
```

##### DriverGrantBucketAccess

This gRPC call creates a set of access credentials for a bucket. This api must be idempotent. The input to this call is the id of the bucket, a set of opaque parameters and name of the account. This `accountName` field is the concatenation of the characters ba (short for BucketAccess) and its UID. It is used as the idempotency key for requests to the drivers regarding a particular BA.

The returned `accountID` should be a unique identifier for the account in the OSP. This value could be the name of the account too. This value will be included in all subsequent calls to the driver for changes to the BucketAccess.

```
    DriverGrantBucketAccess
    |---------------------------------------------|       |-----------------------------------------------|
    | grpc DriverGrantBucketAccessRequest{        | ===>  | DriverGrantBucketAccessResponse{              |
    |     "bucketID": "br-$uuid",                 |       |   "accountID": "bar-$uuid",                   |
    |     "accountName": "bar-$uuid"              |       |   "credentials": {                            |
    |     "authenticationType": "KEY"             |       |      "s3": {                                  |
    |     "parameters": {                         |       |        "accessKeyID": "AKIAODNN7EXAMPLE",     |
    |          "key": "value",                    |       |        "accessSecretKey": "wJaUtnFEMI/K..."   |
    |      }                                      |       |      }                                        |
    | }                                           |       |   }                                           |
    |---------------------------------------------|       | }                                             |
                                                          |-----------------------------------------------|
```

##### DriverDeleteBucket

This gRPC call deletes a bucket in the OSP.

Initiated by sidecar to driver:
```
grpc DriverDeleteBucketRequest{
  bucketID: "br-$uuid"
  parameters: {
    "key": "value"
  }
}
```

Driver response:
```grpc
DriverDeleteBucketResponse{} // empty with return code
```

// TODO: specify return codes for error/success situations here

##### DriverRevokeBucketAccess

// TODO: do we need to specify this somewhere, or is it captured?

BAC-A     BA
params -> params

BAC-A'
params'

del(BA) params (not params')

This gRPC call revokes access granted to a particular account.

Initiated by sidecar to driver:
```grpc
grpc DriverRevokeBucketAccessRequest{
  bucketID: "br-$uuid"
  accountID: "bar-$uuid"
  parameters: { // should be from the params copied to BucketAccess
    "key": "value"
  }
}
```

Driver response:
```grpc
DriverRevokeBucketAccessResponse{} // empty with return code
```

// TODO: specify return codes for error/success situations here

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
- Design COSI APIs to support authentication using access/secret keys, and IAM.
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

No changes are required on upgrade to maintain previous behavior.

### Version Skew Strategy

COSI is out-of-tree, so version skew strategy is N/A

## Alternatives Considered

This KEP has had a long journey and many revisions. Here we capture the main alternatives and the reasons why we decided on a different solution.

### Add Bucket Instance Name to BucketAccessClass (brownfield)

<!-- TODO: this is super out of date. fix it -->

#### Motivation

1. To improve workload _portability_ user namespace resources should not reference non-deterministic generated names. If a `BucketAccessRequest` (BAR) references a `Bucket` instance's name, and that name is pseudo random (eg. a UID added to the name) then the BAR, and hence the workload deployment, is not portable to another cluser.

2. If the `Bucket` instance name is in the BAC instead of the BAR then the user is not burdened with knowledge of `Bucket` names, and there is some centralized admin control over brownfield bucket access.

#### Problems

1. The greenfield -\> brownfield workflow is very awkward with this approach. The user creates a `BucketRequest` (BR) to provision a new bucket which they then want to access. The user creates a BAR pointing to a BAC which must contain the name of this newly created \``Bucket` instance. Since the `Bucket`'s name is non-deterministic the admin cannot create the BAC in advance. Instead, the user must ask the admin to find the new `Bucket` instance and add its name to new (or maybe existing) BAC.

2. App portability is still a concern but we believe that deterministic, unique `Bucket` and `BucketAccess` names can be generated and referenced in BRs and BARs.

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
    - [x] BucketReady bool
    - [ ] ErrorMessage string - last error message; cleared when provisioning is successful
    - [x] BucketID string
- BucketClaim
  - [ ] Events
    - FailedCreateBucket - Report when COSI fails to create bucket for BC, with error message
	  - FailedDeleteBucket - Report when COSI fails to delete bucket for BC, with error message
  - [ ] API .status
    - [x] BucketReady bool
    - [ ] ErrorMessage string - last error message; cleared when provisioning is successful
    - [x] BucketName string
- BucketAccess
  - [ ] Events
    - WaitingForBucket - Report when COSI cannot grant access because bucket does not yet exist
    - FailedGrantAccess - Report when COSI fails to grant access to a bucket, with error message
    - FailedRevokeAccess - Report when COSI fails to revoke access to a bucket, with error message
  - [ ] API .status
    - [x] AccessGranted bool
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
    from when a BucketClaim resource is created until BucketClaim has `Status.BucketReady=true`.
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
  - Definition: Total number of reconciliations conducted by the snapshot controller that result
    in status changes.
  - Labels:
    - `driver_name` - name of COSI driver the operation runs against
    - `resource_kind` - Bucket, BucketClaim, BucketAccess
    - `operation` - Create, Delete
    - `status` - Ready, Waiting, Failed
    - DISCUSS: We could output status as BucketReady, AccessGranted, FailedCreateBucket,
      FailedGrantAccess, FailedDeleteBucket, FailedRevokeAccess, WaitingForBucket, but the
      operation is already included in these statuses, which makes 'operation' less useful; it also
      makes it harder to filter across all kinds like: `resource_kind=<any>, operation=Create, status=Failed`.
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

[1]:    #release-signoff-checklist
[2]:    #summary
[3]:    #motivation
[4]:    #user-stories
[5]:    #goals
[6]:    #non-goals
[7]:    #vocabulary
[8]:    #proposal
[9]:    #apis
[10]:   #storage-apis
[11]:   #bucketrequest
[12]:   #bucket
[13]:   #bucketclass
[14]:   #access-apis
[15]:   #bucketaccessrequest
[16]:   #bucketaccess
[17]:   #bucketaccessclass
[18]:   #app-pod
[19]:   #topology
[20]:   #object-relationships
[21]:   #workflows
[22]:   #finalizers
[23]:   #create-bucket
[24]:   #sharing-cosi-created-buckets
[25]:   #delete-bucket
[26]:   #grant-bucket-access
[27]:   #revoke-bucket-access
[28]:   #delete-bucketaccess
[29]:   #delete-bucket-1
[30]:   #setting-access-permissions
[31]:   #dynamic-provisioning
[32]:   #static-provisioning
[33]:   #grpc-definitions
[34]:   #drivergetinfo
[35]:   #drivercreatebucket
[36]:   #driverdeletebucket
[37]:   #drivergrantbucketaccess
[38]:   #driverrevokebucketaccess
[39]:   #test-plan
[40]:   #graduation-criteria
[41]:   #alpha
[42]:   #alpha---beta
[43]:   #beta---ga
[44]:   #alternatives-considered
[45]:   #add-bucket-instance-name-to-bucketaccessclass-brownfield
[46]:   #motivation-1
[47]:   #problems
[48]:   #upgrade--downgrade-strategy
[49]:   #version-skew-strategy
[50]:   #production-readiness-review-questionnaire
[51]:   #feature-enablement-and-rollback
[52]:   #rollout-upgrade-and-rollback-planning
[53]:   #monitoring-requirements
[54]:   #dependencies
[55]:   #scalability
[56]:   #infrastructure-needed-optional
[57]:   https://git.k8s.io/enhancements
[58]:   https://git.k8s.io/website
[59]:   https://kubernetes.io/
