# Object Storage Support

## Table of Contents

<!-- toc -->
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Introduction](#introduction)
  - [Motivation](#motivation)
  - [User Stories](#user-stories)
      - [Admin](#admin)
      - [DevOps](#devops)
    - [Object Storage Provider (OSP)](#object-storage-provider-osp)
  - [Goals](#goals)
      - [Functionality](#functionality)
      - [System Properties](#system-properties)
    - [Non-Goals](#non-goals)
  - [COSI architecture](#cosi-architecture)
  - [COSI API](#cosi-api)
  - [Bucket Creation](#bucket-creation)
  - [Generating Access Credentials for Buckets](#generating-access-credentials-for-buckets)
  - [Attaching Buckets](#attaching-buckets)
  - [Sharing Buckets](#sharing-buckets)
  - [Accessing existing Buckets](#accessing-existing-buckets)
- [Usability](#usability)
  - [Self Service](#self-service)
  - [Mutating Buckets](#mutating-buckets)
- [Object Lifecycle](#object-lifecycle)
- [COSI API Reference](#cosi-api-reference)
  - [Bucket](#bucket)
  - [BucketClaim](#bucketclaim)
  - [BucketClass](#bucketclass)
  - [BucketAccess](#bucketaccess)
  - [BucketAccessClass](#bucketaccessclass)
  - [BucketInfo](#bucketinfo)
- [COSI Driver](#cosi-driver)
  - [COSI Driver gRPC API](#cosi-driver-grpc-api)
      - [ProvisionerGetInfo](#provisionergetinfo)
      - [ProvisionerCreateBucket](#provisionercreatebucket)
      - [ProvisionerGrantBucketAccess](#provisionergrantbucketaccess)
      - [ProvisionerDeleteBucket](#provisionerdeletebucket)
      - [ProvisionerRevokeBucketAccess](#provisionerrevokebucketaccess)
- [Test Plan](#test-plan)
- [Graduation Criteria](#graduation-criteria)
  - [Alpha](#alpha)
  - [Alpha -&gt; Beta](#alpha---beta)
  - [Beta -&gt; GA](#beta---ga)
- [Alternatives Considered](#alternatives-considered)
  - [Add Bucket Instance Name to BucketAccessClass (brownfield)](#add-bucket-instance-name-to-bucketaccessclass-brownfield)
    - [Motivation](#motivation-1)
    - [Problems](#problems)
    - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Version Skew Strategy](#version-skew-strategy)
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

## Introduction

This proposal introduces the *Container Object Storage Interface* (COSI), a standard for provisioning and consuming object storage in Kubernetes.


## Motivation

File and block storage are treated as first class citizens in the Kubernetes ecosystem via CSI.  Workloads using CSI volumes enjoy the benefits of portability across vendors and across Kubernetes clusters without the need to change application manifests. **An equivalent standard does not exist for Object storage.**

Object storage has been rising in popularity in the recent years as an alternative form of storage to filesystems and block devices. Object storage paradigm promotes disaggregation of compute and storage. This is done by making data available over the network, rather than locally. Disaggregated architectures allow compute workloads to be stateless, which consequently makes them easier to manage, scale and automate.

## User Stories

We define 3 kinds of stakeholders:

+ **Admins** - Members with escalated privileges and authority to manage site wide/cluster wide policies to control access, assign storage quotas and other concerns related to hardware or application resources.
+ **Application Developer/Operator (User)** - Members with privileges to run workloads, request storage for them, and manage resources inside a specific namespace.
+ **Object Storage Provider (OSP)** - Vendors offering Object Storage capabilities

#### Admin

+ Establish cluster/site wide policies for data redundancy, durability and other data related parameters
+ Establish cluster/site wide policies for access control over buckets
+ Enable/restrict users who can create, delete and reuse buckets

#### DevOps

+ Request a bucket to be created and/or provisioned for workloads
+ Request access to be created/deleted for a specific bucket
+ Request deletion of a created bucket

### Object Storage Provider (OSP)

+ Integrate with the official standard for orchestrating Object Storage resources in Kubernetes

## Goals

#### Functionality

+ Support automated **Bucket Creation**
+ Support automated **Bucket Deletion**
+ Support automated **Access Credential Generation**
+ Support automated **Access Credential Revokation**
+ Support automated **Bucket Provisioning** for workloads (enabling pods to access buckets)
+ Support **Bucket Reuse** (use existing buckets via COSI)
+ Support **Automated Bucket Sharing across namespaces**

#### System Properties

+ Support **workload portability** across clusters
+ Achieve the above goals in a **vendor neutral** manner
+ Standardize mechanism for third-party vendors to integrate easily with COSI
+ Allow users (non-admins) to create and utilize buckets (self-service)
+ Establish best-in-class **Access Control** practices for bucket creation and sharing

### Non-Goals

+ **Data Plane** API standardization will not be addressed by this project
+ **Bucket Mutation** will not be supported as of now

## COSI architecture

Since this is an entirely new feature, it is possible to implement this completely out of tree. The following components are proposed for this architecture:

 - COSI ControllerManager
 - COSI Sidecar
 - COSI Driver

1. The COSI ControllerManager is the central controller that validates, authorizes and binds COSI created buckets to BucketClaims. At least one active instance of ControllerManager should be present.
2. The COSI Sidecar is the point of integration between COSI and drivers. All operations that require communication with the OSP is triggered by the Sidecar using gRPC calls to the driver. One active instance of Sidecar should be present **for each driver**.
3. The COSI driver communicates with the OSP to conduct Bucket related operations.

More information about COSI driver is [here](#cosi-driver)

## COSI API

COSI defines 5 new API types

 - [Bucket](#bucket)
 - [BucketClaim](#bucketclaim)
 - [BucketAccess](#bucketaccess)
 - [BucketClass](#bucketclass)
 - [BucketAccessClass](#bucketaccessclass)

Detailed information about these API types are provided inline with user stories. 

Here is a TL;DR version:

 - BucketClaims/Bucket are similar to PVC/PV. 
 - BucketClaim is used to request generation of new buckets. 
 - Buckets represent the actual Bucket. 
 - BucketClass is similar to StorageClass. It is meant for admins to define and control policies for Bucket Creation
 - BucketAccess is required before a bucket can be "attached" to a pod.
 - BucketAccess both represents the attachment status and holds a pointer to the access credentials secret.
 - BucketAccessClass is meant for admins to control authz/authn for users requesting access to buckets.

## Bucket Creation

The following stakeholders are involved in the lifecycle of bucket creation:

 - Users  - request buckets to be created for their workload
 - Admins - setup COSI drivers in the cluster, and specify their configurations

Here are the steps for creating a Bucket:

###### 1. Admin creates BucketClass, User requests Bucket to be created using BucketClaim

The BucketClass represents a set of common properties shared by multiple buckets. It is used to specify the driver for creating Buckets, and also for configuring driver-specific parameters. More information about BucketClass is [here](#bucketclass)

The BucketClaim is a claim to create a new Bucket. This resource can be used to specify the bucketClassName, which in-turn includes configuration pertinent to bucket creation. More information about BucketClaim is available [here](#bucketclaim)

```
    BucketClaim - bcl1                                    BucketClass - bc1
    |------------------------------|                      |--------------------------------|
    | metadata:                    |                      | deletionPolicy: delete         |
    |   namespace: ns1             |                      | provisioner: s3.amazonaws.com  |
    | spec:                        |                      | protocol: s3                   |
    |   bucketClassName: bc1       |                      | parameters:                    |
    |                              |                      |   key: value                   |
    |------------------------------|                      |--------------------------------|
```

###### 2. COSI creates an intermediate Bucket object

The ControllerManager creates a Bucket object by copying fields over from both the BucketClaim and BucketClass. This step can only proceed if the BucketClass exists. This Bucket object is an intermediate object, with its status condition `BucketReady` to false. This indicates that the Bucket is yet to be created in the OSP.

More information about Bucket is [here](#bucket)

```
    Bucket - bcl-$uuid
    |--------------------------------------|
    | name: bcl-$uuid                      |
    | spec:                                |
    |   bucketClassName: bc1               |
    |   protocol: s3                       |
    |   parameters:                        |
    |     key: value                       |
    |   provisioner: s3.amazonaws.com      |
    | status:                              |
    |   conditions:                        |
    |   - type: BucketReady                |
    |     status: False                    |
    |--------------------------------------|
```

###### 3. COSI calls appropriate driver to create Bucket

The COSI sidecar, which runs alongside each of the drivers, listens for Bucket events. All but the sidecar with the specified provisioner will ignore the Bucket object. Only the appropriate sidecar will make a gRPC request to create a Bucket in the OSP.

More information about COSI gRPC API is [here](#cosi-grpc-api)

```
    COSI Driver (s3.amazonaws.com)
    |------------------------------------------|
    | grpc ProvisionerCreateBucket({           |
    |      "name": "bcl-$uuid",                |
    |      "protocol": "s3",                   |
    |      "parameters": {                     |
    |         "key": "value"                   |
    |      }                                   |
    | })                                       |
    |------------------------------------------|
```
Once the Bucket is successfully created in the OSP, and a successful status is retrieved from the gRPC request, sidecar sets `BucketReady`condition to `True`. At this point, the Bucket is ready to be utilized by workloads.

## Generating Access Credentials for Buckets

The following stakeholders are involved in the lifecycle of access credential generation:

 - Users  - request access to buckets
 - Admins - establish cluster wide access policies

Access credentials are represented by BucketAccess objects. The separation of BucketClaim and BucketAccess is a reflection of the usage pattern of Object Storage, where buckets are always accessed over the network, and all access is subject to authentication and authorization i.e. lifecycle of a bucket and its access are not tightly coupled. 

__Example: for the same bucket, one might need a BucketAccess with a "read-only" policy and another to with a "write" policy__
 

Here are the steps for creating a BucketAccess:

###### 1. Admin creates BucketAccessClass, User creates BucketAccess

The BucketAccessClass represents a set of common properties shared by multiple BucketAccesses. It is used to specify policies for creating access credentials, and also for configuring driver-specific access parameters. More information about BucketAccessClass is [here](#bucketaccessclass)

The BucketAccess is used to request access to a bucket. It contains fields for choosing the Bucket for which the credentials will be generated, and also includes a bucketAccessClassName field, which in-turn contains configuration for authorizing users to access buckets. More information about BucketAccess is [here](#bucketaccess)

BucketAccessClass can be used to specify a authorization mechanism. It can be one of 
 - KEY  (__default__) 
 - IAM 

The KEY based mechanism is where access and secret keys are generated to be provided to pods. IAM style is where pods are implicitly granted access to buckets by means of a metadata service. IAM style access provides greater control for the infra/cluster administrator to rotate secret tokens, revoke access, change authorizations etc., which makes it more secure.

```
    BucketAccess - ba1                                        BucketAccessClass - bac1
    |---------------------------------------|                 |----------------------------------|
    | metadata:                             |                 | provisioner: s3.amazonaws.com    |
    |   namespace: ns1                      |                 | parameters:                      |
    | spec:                                 |                 |   key: value                     |
    |   bucketAccessClassName: bac1         |                 | authenticationType: KEY          |
    |   bucketClaimName: bcl1               |                 |----------------------------------|
    |   credentialsSecretName: bucketcreds1 |
    | status:                               |
    |   conditions:                         |
    |   - name: AccessGranted               |
    |     value: False                      |
    |---------------------------------------|
```

The `credentialsSecretName` is the name of the secret that COSI will generate containing credentials to access the bucket. The same secret name has to be set in the podSpec as well as the projected secret volumes.

In case of IAM style authentication, along with the `credentialsSecretName`, `serviceAccountName` field must also be specified. This will map the specified serviceaccount to the appropriate service account in the OSP.

```
    BucketAccess - ba1                                        BucketAccessClass - bac2
    |---------------------------------------|                 |----------------------------------|
    | metadata:                             |                 | provisioner: s3.amazonaws.com    |
    |   namespace: ns1                      |                 | parameters:                      |
    | spec:                                 |                 |   key: value                     |
    |   bucketAccessClassName: bac2         |                 | authenticationType: IAM          |
    |   bucketClaimName: bcl1               |                 |----------------------------------|
    |   credentialsSecretName: bucketcreds1 |
    |   serviceAccountName: svacc1          |
    | status:                               |
    |   conditions:                         |
    |   - name: AccessGranted               |
    |     value: False                      |
    |---------------------------------------|
```

###### 3. COSI calls driver to generate credentials

All but the sidecar for the specified provisioner will ignore the BucketClaim object. Only the appropriate sidecar will make a gRPC request to its driver to generate credentials/map service accounts.

This step will only proceed if the Bucket already exist. The Bucket's `BucketReady` condition should be true. Until access been granted, the `AccessGranted` condition in BucketAccess will be false.

```
    COSI Driver (s3.amazonaws.com)
    |------------------------------------------|
    | grpc ProvisionerGrantBucketAccess({      |
    |      "name": "ba-$uuid",                 |
    |      "bucketID": "br-$uuid",             |
    |      "parameters": {                     |
    |         "key": "value"                   |
    |      }                                   |
    | })                                       |
    |------------------------------------------|
```

Each BucketAccess is meant to map to a unique service account in the OSP. Once the requested privileges have been granted, a secret by the name specified in `credentialsSecretName` in the BucketClaim is created. The secret will reside in the namespace of the BucketClaim. The secret will contain either keys or service account mappings based on the chosen authentication type. 

If this call returns successfully, the sidecar sets `AccessGranted` condition to `True` in the BucketAccess.

NOTE:
 - The secret will not be created until the credentials are generated/service account mappings are complete.
 - Within a namespace, one BucketAccess and secret pair is generally sufficient, but cases which may want to control this more granularly can use multiple.

## Attaching Bucket Information to Pods

The following stakeholders are involved in the lifecycle of attaching bucket information into pods:

 - Users  - specify bucket in the pod definition

Once a valid BucketAccess is available (`AccessGranted` is `True`), pods can use them.

###### 1. User creates pod with a projected volume pointing to the secret in BucketAccess

The secret mentioned in the `credentialsSecretName` field of the BucketAccess should be set as a projected volume in the PodSpec. If either the Bucket provisioning is incomplete, or the access generation is incomplete - the pod will not run. It will wait indefinitely for those two conditions to be true.

```
    PodSpec - pod1
    |-------------------------------------------------|
    | spec:                                           |
    |   containers:                                   |
    |   - volumeMounts:                               |
    |       name: cosi-bucket                         |
    |       mountPath: /cosi/bucket1                  |
    | volumes:                                        |
    | - name: cosi-bucket                             |
    |   projected:                                    |
    |     sources:                                    |
    |     - secret:                                   |
    |       name: bucketcreds1                        |
    |-------------------------------------------------|
```

If IAM style authentication was specified, then the `serviceAccountName` specified in BucketAccess must be specified as the `serviceAccountName` in the podSpec.

```
    PodSpec - pod1
    |-------------------------------------------------|
    | spec:                                           |
    |   serviceAccountName: svacc1                    |
    |   containers:                                   |
    |   - volumeMounts:                               |
    |       name: cosi-bucket                         |
    |       mountPath: /cosi/bucket1                  | 
    | volumes:                                        |
    | - name: cosi-bucket                             |
    |   projected:                                    |
    |     sources:                                    |
    |     - secret:                                   |
    |       name: bucketcreds1                        |
    |-------------------------------------------------|
```

The volume `mountPath` will be the directory where bucket credentials and other related information will be served.

NOTE: the contents of the files served in mountPath will be a COSI generated file containing credentials and other information required for accessing the bucket. **This is NOT intended to specify a mountpoint to expose the bucket as a filesystem.**

###### 2. The secret containing BucketInfo is mounted in the specified directory

The above volume definition will prompt kubernetes to retrieve the secret and place it in the volumeMount path defined above. The contents of the secret will be of the format shown below:

```
    bucket_info.json
    |-----------------------------------------------|
    | {                                             |
    |   apiVersion: "v1alpha1",                     |
    |   kind: "BucketInfo",                         | 
    |   metadata: {                                 |
    |       name: "bc-$uuid"                        |
    |   },                                          |
    |   spec: {                                     |
    |       bucketName: "bc-$uuid",                 |
    |       authenticationType: "KEY",              |
    |       endpoint: "https://s3.amazonaws.com",   |
    |       accessKeyID: "AKIAIOSFODNN7EXAMPLE",    |
    |       accessSecretKey: "wJalrXUtnFEMI/K...",  |
    |       region: "us-west-1",                    |
    |       protocol: "s3"                          |
    |   }                                           |
    | }                                             |
    |-----------------------------------------------|

```

In case IAM style authentication was specified, then metadataURL and serviceAccountTokenPath will be provided.

```
    bucket_info.json
    |-------------------------------------------------|
    | {                                               |
    |   apiVersion: "v1alpha1",                       |
    |   kind: "BucketInfo",                           | 
    |   metadata: {                                   |
    |       name: "bc-$uuid"                          |
    |   },                                            |
    |   spec: {                                       |
    |       bucketName: "bc-$uuid",                   |
    |       authenticationType: "IAM",                |
    |       endpoint: "https://s3.amazonaws.com",     |
    |       region: "us-west-1",                      |
    |       protocol: "s3"                            |
    |   }                                             |
    | }                                               |
    |-------------------------------------------------|

```

Workloads are expected to read the definition in this file to access a bucket. The `BucketInfo` API will not be a CRD in the cluster, however, it follows the same conventions as the rest of the COSI APIs. More details can be found [here](#bucketinfo)

## Sharing Buckets

This section describes the current design for sharing buckets with other namespaces. As of the current milestone (alpha) of COSI, any bucket created in one namespace cannot be accessed in another namespace. i.e. no bucket sharing is possible. In future versions, a namespace level access control will be enforced, and buckets will be constrained to particular namespaces using selectors. Admins will be able to control which namespaces can access which buckets using namespace selectors.

## Accessing existing Buckets

The benefits of COSI can also be brought to existing buckets/ones created outside of COSI. This user story explains the steps to import a bucket:

###### 1. Admin creates a Bucket API object

When a Bucket object is manually created, and has its `bucketID` set, then COSI assumes that this Bucket has already been created. 

```
    Bucket - br-$uuid
    |-------------------------------------------------|
    | name: bucketName123                             |
    | spec:                                           |
    |   bucketID: bucketname123                       |
    |   protocol: s3                                  |
    |   parameters:                                   |
    |     key: value                                  |
    |   provisioner: s3.amazonaws.com                 |
    |-------------------------------------------------|
```

###### 2. User creates BucketAccess to generate credentials for that bucket

Unlike the BucketAccess for COSI created bucket, this BucketAccess should directly reference the Bucket instead of a BucketClaim.

```
    BucketAccess - bac2
    |-------------------------------|
    | spec:                         |
    |   bucketName: bucketName123   |
    |                               |
    |-------------------------------|
```

Note that, as of the alpha version of COSI, there is no authorization mechanism to restrict users who can refer to buckets imported into COSI. In the future, access to imported buckets will also follow the namespace selector approach described [above](#sharing-buckets).

## Bucket deletion

 - A Bucket created by COSI as a result of a BucketClaim can deleted by deleting the BucketClaim 
 - A Bucket created outside of COSI, once imported, cannot be deleted by users from any particular namespace. Privileged users can however delete a Bucket at their discretion. 
 
Once a delete has been issued to a bucket, no new BucketAccesses can be created for it. Buckets having valid BucketAccesses (Buckets in use) will not be deleted until all the BucketAccesses are cleaned up. 

Buckets can be created with one of two deletion policies:
 - Retain
 - Delete

When the deletion policy is Retain, then the underlying bucket is not cleaned up when the Bucket object is deleted. When the deletion policy is Delete, then the underlying bucket is cleaned up when the Bucket object is deleted.

# Usability

## Self Service

Self service is easily possible with the current design as both the BucketRequest and BucketClaim resources are namespace scoped, and users need not have admin privileges to create, modify and delete them.

The only admin steps are creation of class objects(BucketClass, BucketAccessClass) and Bucket imports. The creatio of class object is no different from requiring a StorageClass for provisioning PVCs. It is a well-understood pattern among kubernetes users. Importing a Bucket requires special permissions because its lifecycle is not managed by COSI, and special care needs to be taken to prevent clones, accidental deletions and other mishaps (for instance, setting the deletion policy to Delete).

## Mutating Buckets

As of the current design of COSI, mutating bucket properties is not supported. However, the current design does not prevent us from supporting it in the future. Mutable properties include encryption, object lifecycle configuration, replication configuration etc. These properties will be supported in future versions along with the capability to mutate them.

These properties will be specified in the BucketRequest and follow the same pattern of events as Bucket creation. i.e. Bucket API object will be updated to reflect the properties set in BucketRequest, and then a controller will pick-up these changes and call the appropriate APIs to reflect them in the backend.

# Object Lifecycle

The following resources are managed by admins

- BucketClass
- BucketAccessClass

The following resources are managed by a user. These are created with a reference to their corresponding class objects:

- BucketClaim -> BucketClass
- BucketAccess -> Bucket, BucketClaim, BucketAccessClass

The COSI controller responds by creating the following objects

- BucketRequest <-> Bucket (1:1)
- BucketClaim -> Bucket

Notes:

 - There are **NO** cycles in the relationship graph of the above mentioned API objects.
 - Mutations are not supported in the API.

When a user deletes the BucketRequest, then depending on the DeletionPolicy, the following happens:

- If deletionPolicy is Delete, then Bucket deletion is also triggered. 
- If deletionPolicy is Retain, then Bucket is left as is, but the BucketRequest is deleted.

Only when all accessors (BucketAccesses) of the Bucket are deleted, is the Bucket itself cleaned up. Having orphaned buckets in the cluster is an invalid state, unless the Bucket was imported into the cluster. 

When a user deletes a BucketAccess, the corresponding secret/serviceaccount are also deleted. If a pod has that secret mounted when delete is called, then the secret will not be deleted, but will instead have its deletionTimestamp set. In this way, access to a Bucket is preserved until the application pod dies. 

When an admin deletes any of the class objects, it does not affect existing Buckets as fields from the class objects are copied into the Buckets during creation. 

If a Bucket is manually deleted by an admin, without deleting the corresponding BucketClaim, then the cluster is considered to be in an invalid state. Manual recovery is possible if data is not already lost. 

# COSI API Reference

## Bucket

Resource to represent a Bucket in OSP. Buckets are cluster-scoped.

```yaml
Bucket {
  TypeMeta
  ObjectMeta

  Spec BucketSpec {
    // Provisioner is the name of driver associated with this bucket
    Provisioner string

    // DeletionPolicy is used to specify how COSI should handle deletion of this
    // bucket. There are 3 possible values:
    //  - Retain: Indicates that the bucket should not be deleted from the OSP (default)
    //  - Delete: Indicates that the bucket should be deleted from the OSP
    //        once all the workloads accessing this bucket are done
    // +optional
    DeletionPolicy DeletionPolicy

    // Name of the BucketClass specified in the BucketRequest
    BucketClassName  string

    // Name of the BucketClaim that resulted in the creation of this Bucket
    // Optional in case Bucket was created without a BucketClaim i.e. Imported 
    // +optional
    BucketClaim corev1.ObjectReference

    // Protocol is the data API this bucket is expected to adhere to.
    // The possible values for protocol are:
    // -  S3: Indicates Amazon S3 protocol
    // -  Azure: Indicates Microsoft Azure BlobStore protocol
    // -  GCS: Indicates Google Cloud Storage protocol
    Protocol Protocol

    // Parameters is an opaque map for passing in configuration to a driver
    // for creating the bucket
    // +optional
    Parameters map[string]string

    // ExistingBucketID is the unique id of the bucket in the OSP. This field should be
    // used to specify a bucket that has been created outside of COSI.
    // This field will be empty when the Bucket is dynamically provisioned by COSI.
    // +optional
    ExistingBucketID string
  }

  Status BucketStatus {
    // BucketReady is a boolean condition to reflect the successful creation
    // of a bucket.
    BucketReady bool

    // BucketID is the unique id of the bucket in the OSP. This field will be
    // populated by COSI.
    // +optional
    BucketID string
  }
}
```

## BucketClaim

A claim to create Bucket. BucketClaim is namespace-scoped

```yaml
BucketClaim {
  TypeMeta
  ObjectMeta

  Spec BucketClaimSpec {
    // Name of the BucketClass
    BucketClassName string
  }

  Status BucketClaimStatus {
    // BucketReady indicates that the bucket is ready for consumpotion
    // by workloads
    BucketReady bool

    // BucketName is the name of the provisioned Bucket in response
    // to this BucketClaim
    // +optional
    BucketName string
  }
```
## BucketClass

Resouce for configuring common properties for multiple Buckets. BucketClass is cluster-scoped.

```yaml
BucketClass {
  TypeMeta
  ObjectMeta

  // Provisioner is the name of driver associated with this bucket
  Provisioner string

  // Protocol is the data API this bucket is expected to adhere to.
  // The possible values for protocol are:
  // -  S3: Indicates Amazon S3 protocol
  // -  Azure: Indicates Microsoft Azure BlobStore protocol
  // -  GCS: Indicates Google Cloud Storage protocol
  Protocol Protocol

  // DeletionPolicy is used to specify how COSI should handle deletion of this
  // bucket. There are 3 possible values:
  //  - Retain: Indicates that the bucket should not be deleted from the OSP
  //  - Delete: Indicates that the bucket should be deleted from the OSP
  //        once all the workloads accessing this bucket are done
  DeletionPolicy DeletionPolicy

  // Parameters is an opaque map for passing in configuration to a driver
  // for creating the bucket
  // +optional
  Parameters map[string]string
}
```

## BucketAccess

A resource to access a Bucket. BucketAccess is namespace-scoped

```yaml
BucketAccess {
  TypeMeta
  ObjectMeta

  Spec BucketAccessSpec {
    // BucketClaimName is the name of the BucketClaim.
    // Exactly one of BucketClaimName or BucketName must be set.
    // +optional
    BucketClaimName string

    // BucketName is the name of the Bucket for which
    // credentials need to be generated
    // Exactly one of BucketClaimName or BucketName must be set.
    // +optional
    BucketName string

    // BucketAccessClassName is the name of the BucketAccessClass
    BucketAccessClassName string

    // CredentialsSecretName is the name of the secret that COSI should populate
    // with the credentials. If a secret by this name already exists, then it is
    // assumed that credentials have already been generated. It is not overridden.
    CredentialsSecretName string
    
    // ServiceAccountName is the name of the serviceAccount that COSI will map
    // to the OSP service account when IAM styled authentication is specified
    ServiceAccountName string
  }

  Status BucketAccessStatus {
    // AccessGranted indicates the successful grant of privileges to access the bucket
    AccessGranted bool
    
    // AccountID is the unique ID for the account in the OSP
    // +optional
    AccountID string
  }
```

## BucketAccessClass

Resouce for configuring common properties for multiple BucketClaims. BucketAccessClass is a clustered resource

```yaml
BucketAccessClass {
  TypeMeta
  ObjectMeta

  // AuthenticationType denotes the style of authentication
  // It can be one of
  // KEY - access, secret tokens based authentication
  // IAM - implicit authentication of pods to the OSP based on service account mappings
  AuthenticationType AuthenticationType

  // Parameters is an opaque map for passing in configuration to a driver
  // for granting access to a bucket
  // +optional
  Parameters map[string]string
}
```

## BucketInfo

Resource mounted into pods containing information for applications to gain access to buckets.

```yaml
BucketInfo {
  TypeMeta
  ObjectMeta

  Spec BucketInfoSpec {
    // BucketName is the name of the Bucket 
    BucketName string

    // AuthenticationType denotes the style of authentication
    // It can be one of
    // KEY - access, secret tokens based authentication
    // IAM - implicit authentication of pods to the OSP based on service account mappings
    AuthenticationType AuthenticationType

    // Endpoint is the URL at which the bucket can be accessed
    Endpoint string
    
    // Region is the vendor-defined region where the bucket "resides"
    Region string
    
    // Protocol is the data API this bucket is expected to adhere to.
    // The possible values for protocol are:
    // -  S3: Indicates Amazon S3 protocol
    // -  Azure: Indicates Microsoft Azure BlobStore protocol
    // -  GCS: Indicates Google Cloud Storage protocol
    Protocol Protocol
  }
}
```

# COSI Driver

A component that runs alongside COSI Sidecar and satisfies the COSI gRPC API specification. Sidecar and driver work together to orchestrate changes in the OSP. The driver acts as a gRPC server to the COSI Sidecar. Each COSI driver is identified by a unique id.

The sidecar uses the unique id to direct requests to the appropriate driver. Multiple instances of drivers with the same id will be added into a group and only one of them will act as the leader at any given time.

## COSI Driver gRPC API

#### ProvisionerGetInfo

This gRPC call responds with the name of the provisioner. The name is used to identify the appropriate driver for a given BucketRequest or BucketClaim.

```
    ProvisionerGetInfo
    |------------------------------------------|       |---------------------------------------|
    | grpc ProvisionerGetInfoRequest{}         | ===>  | ProvisionerGetInfoResponse{           |
    |------------------------------------------|       |   "name": "s3.amazonaws.com"          |
                                                       | }                                     |
                                                       |---------------------------------------|
```

#### ProvisionerCreateBucket

This gRPC call creates a bucket in the OSP, and returns information about the new bucket. This api must be idempotent. The input to this call is the name of the bucket and an opaque parameters field.

The returned `bucketID` should be a unique identifier for the bucket in the OSP. This value could be the name of the bucket too. This value will be used by COSI to make all subsequent calls related to this bucket.

```
    ProvisionerCreateBucket
    |------------------------------------------|       |-----------------------------------------------|
    | grpc ProvisionerCreateBucketRequest{     | ===>  | ProvisionerCreateBucketResponse{              |
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

#### ProvisionerGrantBucketAccess

This gRPC call creates a set of access credentials for a bucket. This api must be idempotent. The input to this call is the id of the bucket, a set of opaque parameters and name of the account. This `accountName` field is used to ensure that multiple requests for the same BucketClaim do not result in multiple credentials.

The returned `accountID` should be a unique identifier for the account in the OSP. This value could be the name of the account too. This value will be used by COSI to make all subsequent calls related to this account.

```
    ProvisionerGrantBucketAccess
    |---------------------------------------------|       |-----------------------------------------------|
    | grpc ProvisionerGrantBucketAccessRequest{   | ===>  | ProvisionerGrantBucketAccessResponse{         |
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

#### ProvisionerDeleteBucket

This gRPC call deletes a bucket in the OSP.

```
    ProvisionerDeleteBucket
    |---------------------------------------------|       |-----------------------------------------------|
    | grpc ProvisionerDeleteBucketRequest{        | ===>  | ProvisionerDeleteBucketResponse{}             |
    |     "bucketID": "br-$uuid"                  |       |-----------------------------------------------|
    | }                                           |
    |---------------------------------------------|
```

#### ProvisionerRevokeBucketAccess

This gRPC call revokes access granted to a particular account.

```
    ProvisionerDeleteBucket
    |---------------------------------------------|       |-----------------------------------------------|
    | grpc ProvisionerRevokeBucketAccessRequest{  | ===>  | ProvisionerRevokeBucketAccessResponse{}       |
    |     "bucketID": "br-$uuid",                 |       |-----------------------------------------------|
    |     "accountID": "bar-$uuid"                |
    | }                                           |
    |---------------------------------------------|

```

# Test Plan

- Unit tests will cover the functionality of the controllers.
- Unit tests will cover the new APIs.
- An end-to-end test suite will cover testing all the components together.
- Component tests will cover testing each controller in a blackbox fashion.
- Tests need to cover both correctly and incorrectly configured cases.


# Graduation Criteria

## Alpha
- API is reviewed and accepted
- Implement all COSI components to support Greenfield, Green/Brown Field, Brownfield and Static Driverless provisioning
- Evaluate gaps, update KEP and conduct reviews for all design changes
- Develop unit test cases to demonstrate that the above mentioned use cases work correctly

## Alpha -\> Beta
- Basic unit and e2e tests as outlined in the test plan.
- Metrics in kubernetes/kubernetes for bucket create and delete, and granting and revoking bucket access.
- Metrics in provisioner for bucket create and delete, and granting and revoking bucket access.

## Beta -\> GA
- Stress tests to iron out possible race conditions in the controllers.
- Users deployed in production and have gone through at least one K8s upgrade.

# Alternatives Considered
This KEP has had a long journey and many revisions. Here we capture the main alternatives and the reasons why we decided on a different solution.

## Add Bucket Instance Name to BucketAccessClass (brownfield)

### Motivation
1. To improve workload _portability_ user namespace resources should not reference non-deterministic generated names. If a `BucketAccessRequest` (BAR) references a `Bucket` instance's name, and that name is pseudo random (eg. a UID added to the name) then the BAR, and hence the workload deployment, is not portable to another cluser.

1. If the `Bucket` instance name is in the BAC instead of the BAR then the user is not burdened with knowledge of `Bucket` names, and there is some centralized admin control over brownfield bucket access.

### Problems
1. The greenfield -\> brownfield workflow is very awkward with this approach. The user creates a `BucketRequest` (BR) to provision a new bucket which they then want to access. The user creates a BAR pointing to a BAC which must contain the name of this newly created \``Bucket` instance. Since the `Bucket`'s name is non-deterministic the admin cannot create the BAC in advance. Instead, the user must ask the admin to find the new `Bucket` instance and add its name to new (or maybe existing) BAC.

1. App portability is still a concern but we believe that deterministic, unique `Bucket` and `BucketAccess` names can be generated and referenced in BRs and BARs.


### Upgrade / Downgrade Strategy

No changes are required on upgrade to maintain previous behaviour.

### Version Skew Strategy

COSI is out-of-tree, so version skew strategy is N/A

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
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled). No

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

N/A since we are only targeting alpha for this Kubernetes release

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

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

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [ ] Other (treat as last resort)
  - Details:

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

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
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
- BucketAccess
- BucketAccessClass

and the following namespaced scoped resources

- BucketRequest
- BucketAccessRequest

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
[34]:   #provisionergetinfo
[35]:   #provisonercreatebucket
[36]:   #provisonerdeletebucket
[37]:   #provisionergrantbucketaccess
[38]:   #provisionerrevokebucketaccess
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
