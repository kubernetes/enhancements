---
title: Object Bucket Provisioning
authors:
  - "@jeffvance"
  - "@copejon"
owning-sig: "sig-storage"
reviewers:
  - "@saad-ali"
  - "@alarge"
  - "@erinboyd"
  - "@guymguym"
  - "@travisn"
approvers:
  - TBD
editor: TBD
creation-date: 2019-11-25
last-updated: 2020-02-27
status: provisional
---

# Object Bucket Provisioning

## Table of Contents

<!-- toc -->
- [Summary](#summary)
  - [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Vocabulary](#vocabulary)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
      - [Admin](#admin)
      - [User](#user)
  - [System Configuration](#system-configuration)
    - [Unique Driver Names](#unique-driver-names)
  - [Workflows](#workflows)
    - [Distinguishing Bucket Case](#distinguishing-bucket-case)
      - [Create Bucket (Greenfield)](#create-bucket-greenfield)
      - [Grant Bucket Access (Brownfield)](#grant-bucket-access-brownfield)
      - [Delete Or Revoke Access (Greenfield &amp; Brownfield)](#delete-or-revoke-access-greenfield--brownfield)
    - [Static Buckets](#static-buckets)
      - [Grant Access](#grant-access)
      - [Revoke Access](#revoke-access)
  - [Custom Resource Definitions](#custom-resource-definitions)
      - [Bucket](#bucket)
      - [BucketContent](#bucketcontent)
      - [BucketClass](#bucketclass)
      - [COSIRegistration](#cosiregistration)
<!-- /toc -->

# Summary

This proposal introduces the Container Object Storage Interface (COSI), whose purpose is to provide a familiar and standardized method of provisioning object storage buckets in a manner agnostic to the storage vendor. The COSI environment is comprised of Kubernetes CRDs, operators to manage these CRDs, and a gRPC interface by which these operators communicate with vendor drivers.  This design is heavily inspired by the Kubernetes’ implementation of the Container Storage Interface (CSI).
However, bucket management lacks some of the notable requirements of block and file provisioning, such as no dependency on the kubelet, no node topology constraints, etc. This allows for an architecture with lower overall complexity.
This proposal does _not_ include a standardized protocol or abstraction of storage vendor APIs.  

## Motivation 

File and block are first class citizens within the Kubernetes ecosystem.  Object, though different on a fundamental level and lacking an open, committee controlled interface like POSIX, is a popular means of storing data, especially against very large data sources.   As such, we feel it is in the interest of the community to elevate buckets to a community supported feature.  In doing so, we can provide Kubernetes cluster users and administrators a normalized and familiar means of managing object storage.

## Goals
+ Define a control plane API in order to standardize and formalize Kubernetes bucket provisioning
+ Be un-opinionated about the underlying object-store.
+ Present similar workflows for both _greenfield_  and _brownfield_ bucket provisioning.
+ Minimize privileges required to run a storage driver.
+ Minimize technical ramp-up with a design that is familiar to CSI storage driver authors and Kubernetes storage admins.
+ Use standard Kubernetes mechanisms to sync a pod with the readiness of the bucket it will consume. This can be accomplished via Secrets.

## Non-Goals
+ Define a native _data-plane_ object store API which would greatly improve object store app portability.
+ Mirror the static workflow of PersistentVolumes wherein users are given the first available Volume.  Pre-provisioned buckets are expected to be non-empty and thus unique.

##  Vocabulary

+  _Brownfield Bucket_ - externally created and represented by a `BucketClass` and managed by a provisioner.
+ _Bucket_ - A user-namespaced custom resource representing an object store bucket.
+  _BucketClass_ - A cluster-scoped custom resource containing fields defining the provisioner and an immutable parameter set.
   + _In Greenfield_: an abstraction of new bucket provisioning.
   + _In Brownfield_: an abstration of an existing objet store bucket.
+ _BucketContent_ - A cluster-scoped custom resource bound to a `Bucket` and containing relevant metadata.
+ _Container Object Storage Interface (COSI)_ -  A specification of gRPC data and methods making up the communication protocol between the driver and the sidecar.
+ _COSI Controller_ - A central controller responsible for managing `Buckets`, `BucketContents`, and Secrets.
+ _COSIRegistration_ - A cluster-scoped custom resource which serves the purpose of registering a driver.
+ _Driver_ - A containerized gRPC server which implements a storage vendor’s business logic through the COSI interface. It can be written in any language supported by gRPC and is independent of Kubernetes.
+ _Greenfield Bucket_ - a new bucket created and managed by the COSI system.
+  _Object_ - An atomic, immutable unit of data stored in buckets.
+ _Sidecar_ - A `BucketContent` controller that communicates to the driver via a gRPC client.
+ _Static Bucket_ - externally created and manually integrated but _lacking_ a provisioner.

# Proposal

## User Stories

#### Admin

- As a cluster administrator, I can set quotas and resource limits on generated buckets' storage capacity via the Kubernete's API, so that  I can control monthly infrastructure costs.
- As a cluster administrator, I can use Kubernetes RBAC policies on bucket APIs, so that I may control integration and access to pre-existing buckets from within the cluster, reducing the need to administer an external storage interface.
- As a cluster administrator, I can manage multiple object store providers via the Kubernetes API, so that I do not have to become an expert in several different storage interfaces.

#### User

- As a developer, I can define my object storage needs in the same manifest as my workload, so that deployments are streamlined and encapsulated within the Kubernetes interface.
- As a developer, I can define a manifest containing my workload and object storage configuration once, so that my app may be ported between clusters as long as the storage provided supports my designated data path protocol.

  
## System Configuration

+ The COSI controller runs in the `cosi-system` namespace where it manages `Buckets`, `BucketContents`, and Secrets. This namespace name is not enforced but suggested.
+ The Driver and Sidecar containers run together in a Pod and are deployed in any namespace, communicating via the Pod's internal network (_localhost:some-port_). We expect and will document that different drivers live in separate namespaces.
+ Operations must be idempotent in order to handle failure recovery.

### Unique Driver Names

**Note:** CSI does _not_ ensure unique driver names. We want to provide a mechanism for this but it may prove too difficult or not worth the time for MVP.

It is important that driver names are unique otherwise multiple sidecars would try to handle the same `BucketContent` events (since the sidecar matches on driver name).  To ensure unique driver names, the sidecar creates the `COSIRegistration` object, which is cluster scoped, and its _metadata.name_ is the name of the driver.

Sidecar start up will follow these steps:

1. make gRPC call to get driver's name.
1. create a `COSIRegistration` object using the driver's name.
1. repeat step 2 in an exponential back-off loop until the `COSIRegistration` has been created or we timeout.
1. a timeout fails the sidecar.

**Note:** the `COSIRegistration` object is expected to be deleted when the sidecar exits.

**Note:** Sidecar _restart_ resiliency is needed so that it can distinguish between its own `COSIRegistration` already existing vs. failing on driver name collision.

## Workflows



### Distinguishing Bucket Case

| BucketClassFields             | SecretRef: nil | SecretRef: non-nil |
| ----------------------------- | -------------- | ------------------ |
| **objectBucketName: non-nil** | Brownfield     | Undefined          |
| **objectBucketName: nil**     | Greenfield     | Static             |

#### Create Bucket (Greenfield)

1. The user creates a `Bucket` in their namespace, with reference to a `BucketClass`.
1. The Controller sees the new `Bucket` and applies a `finalizer` for orchestrated deletions.
1. The Controller gets the `BucketClass` referenced by the `Bucket`.
1. The Controller creates a `BucketContent` object with its `BucketClassName` set to the name of the `BucketClass` and a `finalizer`.
1. The Sidecar detects the new `BucketContent` object and gets the associated `BucketClass`.
1. The Sidecar calls the CreateBucket() rpc, passing the `bucketClass.parameters` and is returned a bucket endpoint and credentials.
1. The Sidecar creates a secret containing the endpoint and credentials, with a random/unique name and `ownerRef` set to `BucketContent`.
1. The Sidecar updates `BucketContent.secretRef` with its `secret` name and namespace and sets `BucketContent.status.phase` to *“Ready”.*
1. The Controller detects the `BucketContent` update and sees the *“Ready”* phase. 
1. The Controller copies the generated `secret` to the `Bucket` namespace with name defined in `Bucket.secretName`.  The `secret` is created with an `ownerRef` of the `Bucket`.
1. The Controller “binds” the `Bucket` and `BucketContent` by setting `bucket.status.bucketContentName` and `bucketContent.bucketRef`, and sets both statuses to *“Bound”*.
1. The app `pod` ingests `secret` and runs.

#### Grant Bucket Access (Brownfield)

1. User creates a `Bucket` in their namespace, with reference to a `BucketClass`.
1. Controller sees the new `Bucket` and applies a `finalizer` for orchestrated deletions.
1. Controller gets the `BucketClass` referenced by the `Bucket`.
1. Controller creates a `BucketContent` object with its `BucketClassName` set to the name of its `BucketClass` and a `finalizer`.
1. Sidecar detects the new `BucketContent` object and gets the associated `BucketClass`.
1. Sidecar calls the GrantBucketAccess() rpc, passing the `bucketClass.objectBucketName` and the `bucketClass.parameters` and is returned a bucket endpoint and credentials.
1. Sidecar creates a `secret` containing the endpoint and credentials, with a random/unique name and `ownerRef` set to `BucketContent`.
1. Sidecar updates `BucketContent.secretRef` with its `secret` name and namespace and sets `BucketContent.status.phase` to *“Ready”*.
1. Controller detects the `BucketContent` update and sees the *“Ready”* phase. 
1. Controller copies the generated `secret` to the `Bucket` namespace with name `Bucket.secretName`.  The `secret` is created with an ownerRef of the Bucket.
1. Controller *“binds”* the `Bucket` and `BucketContent` by setting `bucket.status.bucketContentName` and `bucketContent.bucketRef`, and sets both statuses to *“Bound”*.
1. App `pod` ingests `secret` and runs.

#### Delete Or Revoke Access (Greenfield & Brownfield)

1. The user deletes their `Bucket`, which blocks until the `finalizer` is removed.
1. The Controller detects the event and sees the `deletionTimestamp` set in `Bucket` and gets the `BucketClass`. 
1. If the `BucketClass.secretRef` is nil, the object store bucket is not static, and the process continues to step 4.
1. The Controller deletes the referenced `BucketContent` object, which blocks until the `finalizer` is removed.
1. The Sidecar detects the `BucketContent` event and sees the `deletionTimestamp`, and gets the referenced `BucketClass`.
1. If the `BucketClass.objectBucketName` is nil, the Sidecar decides the `BucketClass.objectBucket` is a greenfield object store bucket and calls the rpc associated with the `BucketClass.releasePolicy` (*DeleteBucket* or *RevokeBucketAccess*). Otherwise, the `BucketClass.objectBucketName` is non-nil, indicating that it is a brownfield object store bucket, and the Sidecar calls the *RevokeBucketAccess()* rpc.
1. The Sidecar sets `BucketContent.status.phase` to *“Released”.*
1. The Controller sees `BucketContent` status is “*Released*” and removes `BucketContent`’s `finalizer`.
1. The `BucketContent` and the dependent `Secret` will be garbage collected.
1. The Controller removes the `finalizer` from the `Bucket`, allowing it and it’s dependent `secret` to be garbage collected.

### Static Buckets

> Note: No driver, and thus no sidecar, are present to manage provisioning.

#### Grant Access

1. An admin defines a `BucketClass` and a `Secret` in a protected namespace, with the `BucketClass.secretRef` field referencing the `Secret`.
1. The user creates a `Bucket` in their namespace, with reference to a `BucketClass`.
1. The Controller sees the new `Bucket` and applies a `finalizer` for orchestrated deletions.
1. The Controller gets the `BucketClass` referenced by the `Bucket`.
1. The Controller creates a `BucketContent` object with `BucketClassName` set to the name of its `BucketClass`, a `finalizer`, the `secretRef` set to the `BucketClass`’s `secretRef`, and phase set to *“Ready”*.
1. The Controller detects the `BucketContent` event and sees the *“Ready”* phase. 
1. The Controller copies the admin defined `secret` to the `Bucket` namespace with name `Bucket.secretName`.  The `secret` is created with an `ownerRef` of the `Bucket`.
1. The Controller “binds” the `Bucket` and `BucketContent` by setting `bucket.status.bucketContentName` and `bucketContent.bucketRef`, and sets both statuses to *“Bound”*.
1. The app `pod` ingests `secret` and runs.

#### Revoke Access

1. The user deletes their `Bucket`, which blocks until the `finalizer` is removed.
1. The Controller detects the event and sees the `deletionTimestamp` set in `Bucket` and gets the `Bucket`’s `BucketClass`.
1. If the `BucketClass.secretRef` is non-nil, the object store bucket is static, and the process continues to step 4.
1. The Controller deletes the referenced `BucketContent` object and sets the `BucketContent.status.phase` to *“Released”*.
1. The Controller sees `BucketContent` status is *“Released”* and removes `BucketContent’`s `finalizer`.
1. The `BucketContent` is garbage collected.  The admin/user defined `secret` will not be garbage collected as it is not a dependent of the `BucketContent`.
1. The Controller removes the `finalizer` from the `Bucket`, allowing it and it’s dependent `secret` to be garbage collected.

##  Custom Resource Definitions

#### Bucket

A user facing API object representing an object store bucket. Created by a user in their app's namespace. Once provisiong is complete, the `Bucket` is "bound" to the corresponding `BucketContent`. Binding is 1:1, meaning there is only one `BucketContent` per `Bucket` and vice-versa.


```yaml
apiVersion: cosi.io/v1alpha1
kind: Bucket
metadata:
  name:
  namespace:
  labels:
    cosi.io/provisioner: [1]
  finalizers:
  - cosi.io/finalizer [2]
spec:
  bucketPrefix: [3]
  bucketClassName: [4]
  secretName: [5]
status:
  bucketContentName: [6]
  phase: [7]
  conditions: 
```
1. `labels`: COSI controller adds the label to its managed resources to easy CLI GET ops.  Value is the driver name returned by GetDriverInfo() rpc. Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.
1. `finalizers`: COSI controller adds the finalizer to defer `Bucket` deletion until backend deletion ops succeed.
1. `bucketPrefix`: (Optional) prefix prepended to a randomly generated bucket name, eg. "YosemitePhotos-". If empty no prefix is appended.
1. `bucketClassName`: Name of the target `BucketClass`.
1. `secretName`: Desired name for user's credential Secret. Defining this name allows for a single manifest workflow.  In cases of name collisions, attempting to create the user's secret will continue until a timeout occurs.
1. `bucketContentName`: Name of a bound `BucketContent`.
1. `phase`: 
   - *Pending*: The controller has detected the new `Bucket` and begun provisioning operations
   - *Bound*: Provisioning operations have completed and the `Bucket` has been bound to a `BucketContent`.

#### BucketContent

A cluster-scoped resource representing an object store bucket. The `BucketContent` is expected to store stateful data relevant to bucket deprovisioning. The `BucketContent` is bound to the `Bucket` in a 1:1 mapping. For MVP a `BucketContent` is not reused.

```yaml
apiVersion: cosi.io/v1alpha1
kind: BucketContent
Metadata:
  name: [1]
  labels:
    cosi.io/provisioner: [2]
  finalizers:
  - cosi.io/finalizer [3]
spec:
	provisioner: [4]
	releasePolicy: [5]
	accessModes: [6]
	supportedProtocols: [7]
  bucketClassName: [8]
  bucketRef: [9]
    name:
    namespace:
    uuid:
    resourceVersion:
  secretRef: [10]
    name:
    namespace:
  bucketIdentifier: [11]
  parameters: 
status:
	message: [12]
  phase: [13]
  conditions:
```
1. `name`: Generated in the pattern of `<BUCKET-CLASS-NAME>'-'<RANDOM-SUFFIX>`. 
1. `labels`: COSI controller adds the label to its managed resources for easy CLI GET ops.  Value is the driver name returned by GetDriverInfo() rpc. Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.
1. `finalizers`: COSI controller adds the finalizer to defer Bucket deletion until backend deletion ops succeed.
1. `provisioner`: The name of the drive as defined in the BucketClass.  Used by sidecars to filter BucketContents.
1. `releasePolicy`: Prescribes outcome of a Delete events. **Note:** In Brownfield and Static cases, *Retain* is mandated.
    - _Delete_:  the bucket and its contents are destroyed
    - _Retain_:  the bucket and its data are preserved with only abstracting Kubernetes being destroyed
1. `accessModes`: (Optional) Declares the level of access given to credentials provisioned through this class.     If empty, drivers may set defaults.
1. `supportedProtocols`: (Optional) An array of protocols the associated object store supports (e.g. swift, s3, gcs, etc.). *Only* serves a descriptive purpose and is not verified.
1. `bucketClassName`: Name of the associated `BucketClass`.
1. `bucketRef`: the name & namespace of the associated `Bucket`.
    - `name` : the Bucket’s name
    - `namespace`: the Bucket’s namespace
    - `uuid`: the Bucket’s API server generated UUID
    - `resourceVersion`: the Bucket’s 
1. `secretRef`: the name and namespace of the source secret in the `Bucket`'s namespace.  This `secret` may be generated by the driver or created by an admin.
1. `bucketIdentifier`: a unique set of identifying information 
1. `message`: a human readable description detailing the reason for the current `phase`
1. `phase`: is the current state of the `BucketContent`:
    - _Bound_: the controller finished processing the request and bound the `Bucket` and `BucketContent`
    - _Released_: the `Bucket` has been deleted, signalling that the `BucketContent` is ready for garbage collection.
    - _Failed_: error and all retries have been exhausted.
    - _Retrying_: set when a driver or Kubernetes error is encountered during provisioning operations indicating a retry loop.

#### BucketClass

A cluster-scoped custom resource used to describe both greenfield and brownfield buckets.  The `BucketClass` defines a release policy, and specifies driver specific parameters, such as region, bucket lifecycle policies, etc., as well as the provisioner name. The driver name is used by sidecars to filter `BucketContent` objects.  In dynamic brownfield workflows, the BucketClass contains a reference to a `objectBucketName` which names the existing object store bucket. In static workflows, the provisioner field is empty, the secret used to grant access to the bucket must be specified, and the `objectBucketName` is unnecessary. In static cases, the secret must be manually created exactly as if generated by a driver, including endpoint, credentials, bucket name, etc.

There is currently no default bucket class.

```yaml
apiVersion: cosi.io/v1alpha1
kind: BucketClass
metadata:
  name: 
provisioner: [1]
supportedProtocols: [2]
accessMode: {"ro", "wo", "rw"} [3]
releasePolicy: {"Delete", "Retain"} [4]
bucketIdentifier: [5]
secretRef: [6]
  name:
  namespace:
parameters: string:string [7]
```

1. `provisioner`: (Optional) The name of the driver. If supplied the driver container and sidecar container are expected to be deployed. If omitted the `secretRef` is required for static provisioning.
1. `supportedProtocols`: (Optional) An array of protocols the associated object store supports (e.g. swift, s3, gcs, etc.). *Only* serves a descriptive purpose and is not verified.
1. `accessModes`: (Optional) Declares the level of access given to credentials provisioned through this class.     If empty, drivers may set defaults.
1.  `releasePolicy`: Prescribes outcome of a Delete events. **Note:** In Brownfield and Static cases, *Retain* is mandated.
    - _Delete_:  the bucket and its contents are destroyed
    - _Retain_:  the bucket and its data are preserved with only abstracting Kubernetes being destroyed
- `bucketIdentifier`: (Optional) Signals Brownfield use.  Defines the name of an existing bucket in an object store.
1. `secretRef`: (Optional) Signals Static use. The name and namespace of an existing secret to be copied to the `Bucket`'s namespace for static provisioning.
1. `parameters`: (Optional) Object store specific key-value pairs passed to the driver.

