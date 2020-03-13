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
  - [Workflows](#workflows)
    - [Greenfield Buckets](#greenfield-buckets)
      - [Create Bucket](#create-bucket)
      - [Delete Bucket](#delete-bucket)
    - [Brownfield Buckets](#brownfield-buckets)
      - [Grant Access](#grant-access)
      - [Revoke Access](#revoke-access)
    - [Static Buckets](#static-buckets)
      - [Grant Access](#grant-access-1)
      - [Revoke Access](#revoke-access-1)
  - [Custom Resource Definitions](#custom-resource-definitions)
      - [Bucket](#bucket)
      - [BucketContent](#bucketcontent)
      - [BucketClass](#bucketclass)
<!-- /toc -->

# Summary

This proposal introduces the Container Object Storage Interface (COSI), whose purpose is to provide a familiar and standardized method of provisioning object storage buckets in an manner agnostic to the storage vendor. The COSI environment is comprised of Kubernetes CRDs, operators to manage these CRDs, and a gRPC interface by which these operators communicate with vendor drivers.  This design is heavily inspired by the Kubernetes’ implementation of the Container Storage Interface (CSI).  However, bucket management lacks some of the notable requirements of block and file provisioning and allows for an architecture with lower overall complexity.  This proposal does not include a standardized protocol or abstraction of storage vendor APIs.  

## Motivation 

File and block are first class citizens within the Kubernetes ecosystem.  Object, though different on a fundamental level and lacking an open, committee controlled interface like POSIX, is a popular means of storing data, especially against very large data sources.   As such, we feel it is in the interest of the community to elevate buckets to a community supported feature.  In doing so, we can provide Kubernetes cluster users and administrators a normalized and familiar means of managing object storage.

## Goals
+ Define a control plane API in order to standardize and formalize a Kubernetes bucket provisioning
+ Minimize privileges required to run a storage driver.
+ Minimize technical ramp-up for storage vendors.
+ Be un-opinionated about the underlying object-store.
+ Use Kubernetes Secrets to inject bucket information into application workflows.
+ Present similar workflows for both _greenfield_  and _brownfield_ bucket provisioning.
+ Present a design that is familiar to CSI storage driver authors and Kubernetes storage admins.

## Non-Goals
+ Define a native _data-plane_ object store API which would greatly improve object store app portability.
+ Mirror the static workflow of PersistentVolumes wherein users are given the first available Volume.  Pre-provisioned buckets are expected to be non-empty and thus unique.

##  Vocabulary

+  _Brownfield Bucket_ - externally created, represented by a BuckteClass and managed by a provisioner
+ _Bucket_ - A user-namespaced custom resource representing an object store bucket.
+  _BucketClass_ - A cluster-scoped custom resource containing fields defining the provisioner and an immutable parameter set.
   + _In Greenfield_: an abstraction of new bucket provisioning.
   + _In Brownfield_: an abstration of an existing objet store bucket.
+ _BucketContent_ - A cluster-scoped custom resource, bound to a Bucket and containing relevant metadata. 
+ _Container Object Storage Interface (COSI)_ -  A specification of gRPC data and methods making up the communication protocol between the driver and the sidecar.
+ _COSI Controller_ - A central controller responsible for mananing Buckets, BucketContents, and Secrets cluster-wide.
+ _Driver_ - A containerized gRPC server which implements a storage vendor’s business logic through the COSI interface. It can be written in any language supported by gRPC and is independent of Kubernetes.
+ _Greenfield Bucket_ - created and managed by the COSI system.
+  _Object_ - An atomic, immutable unit of data stored in buckets.
+ _Sidecar_ - A BucketContent controller that communicates to the driver via a gRPC client.
+ _Static Bucket_ - externally created and manually integrated but **lacking** a provisioner

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

+ The COSI controller runs in the `cosi-system` namespace where it manages Buckets, BucketContents, and Secrets cluster-wide.
+ The Driver and Sidecar containers run together in a Pod and are deployed in the `cosi-system` namespace, communicating via the Pod's internal network.
+ Operations must be idempotent in order to handle failure recovery.
## Workflows

### Greenfield Buckets

#### Create Bucket

1. The user creates Bucket in the app’s namespace.
1. The COSI Controller detects the Bucket and creates a BucketContent object containing driver-relevant data.
1. The Sidecar detects the BucketContent and issues a CreateBucket() rpc with any parameters defined in the BucketClass.
1. The driver provisions the bucket and returns pertinent endpoint, credential, and metadata information.
1. The sidecar creates a Secret in it's namespace (`cosi-system`) containing endpoint and credential information and an owener reference to the BucketContent.
1. The COSI controller generates a Secret in the Bucket's namespace containing the data of it's parent Secret, with an owner reference to the Bucket.  
1. The workflow ingests the Secret to begin operation.

#### Delete Bucket

1. The user deletes their Bucket, which blocks until backend deletion operations complete.
1. The COSI controller detects the update and deletes the BucketContent, which also blocks until backend deletion operations complete.
1. The sidecar detects this and issues a DeleteBucket() rpc to the Driver.  It passes pertitent data stored in the BucketContent.
1. The driver deletes the bucket from the object store and responds with no error.
1. The sidecar removes the BucketContent's finalizer.  The BucketContent and parent Secret are garbage collected.
1. The COSI controller removes the finalizer on the Bucket.  The Bucket and the child Secret are garbage collected.

### Brownfield Buckets

#### Grant Access

1. A BucketClass is defined specifically referencing an object store bucket.
1. A user creates a Bucket in the app namespace, specifying the BucketClass.
1. The COSI controller detects the Bucket and creates a BucketContent object in the `cosi-system` namespace.
1. The sidecar detects the BucketContent object and calls the GrantAccess() rpc to the driver, returing a set of credentials for accessing the bucket.
1. The sidecar writes the credentials and endpoint information to a Secret in the `cosi-system` namespace, with an owner reference to the BucketContent.
1. The COSI controller generates a Secret in the Bucket's namespace containing the parent Secret's data, with an owner reference to the Bucket.
1. The workflow ingests the Secret to begin operation.

#### Revoke Access

1. The user deletes the Bucket, which blocks until revoke operations complete.
2. The COSI controller detects the update and deletes the BucketContent, which also blocks until backend revoke operations complete.
3. The sidecar detects the BucketContent update and invokes the RevokeAccess() rpc method.
4. The driver terminates the access for the associated credentials.
5. The sidecar removes the finalizer from the BucketContent, allowing it and the parent Secret to be garbage collected.
6. The COSI controller removes the finalizer from the Bucket,  allowing it and the child Secret to be garbage collected.

### Static Buckets

> Note: No driver is present to manage provisioning.  The COSI controller only automates Bucket/BucketContent binding operations.

#### Grant Access

1. A BucketClass is defined specifically referencing an object store bucket and a Secret in the `cosi-system` namespace containing access credentials.
1. A user creates a Bucket in the app namespace, specifying the BucketClass.
1. The COSI controller detects the object store bucket and Secret in the BucketClass and creates a BucketContent in the `cosi-system` namespace.
1. The COSI controller generates a Secret in the Bucket's namespace containing the parent Secret's data, with an owner reference to the Bucket.
1. The workflow ingests the Secret to begin operation.

#### Revoke Access

1. The user deletes the Bucket, which blocks until revoke operations complete.
1. The COSI controller deletes the Bucket's bound BucketContent object.
1. The COSI controller removes the finalizer from the Bucket, allowing it and the child Secret to be garbage collected.

##  Custom Resource Definitions

#### Bucket

A user facing API object representing an object store bucket. Created by a user in their app's namespace. Once provisiong is complete, the Bucket is "bound" to the corresponding BucketContent. The is used to prevent further binds to the BucketContent.


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
  bucketName: [3]
  generateBucketName: [4]
  bucketClassName: [5]
  secretName: [6]
  protocol: [7]
	accessMode: {"ro", "rw"} [8]
status:
  bucketContentName: [9]
  phase:
  conditions: 
```
1. `labels`: COSI controller adds the label to its managed resources to easy CLI GET ops.  Value is the driver name returned by GetDriverInfo() rpc*.
1. `finalizers`: COSI controller adds the finalizer to defer Bucket deletion until backend deletion ops succeed.
1. `bucketName`: Desired name of the bucket to be created**.
1. `generateBucketName`: Desired prefix to a randomly generated bucket name. Mutually exclusive with `bucketName`**.
1. `bucketClassName`: Name of the target BucketClass.
1. `secretName`: Desired name for user's credential Secret. Fails on name collisions. Deterministic names allow for a single manifest workflow.
1. `protocol`: String array of protocols (e.g. s3, gcs, swift, etc.) requested by the user.  Used in matching Buckets to BucketClasses and ensuring compatibility with backing object stores.
1. `accessMode`:  The requested level of access provided to the returned access credentials.
1. `bucketContentName`: Name of a bound BucketContent

> \* Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.

> ** Ignored in Brownfield and Static operations

#### BucketContent

A cluster-scoped resource representing an object store bucket. The BucketContent is expected to store stateful data relevant to bucket deprovisioning. The BucketContent is bound to the Bucket in a 1:1 mapping.

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
  bucketClassName: [4]
  supportedProtocols: [5]
  releasePolicy: {"Delete", "Retain"} [6]
  bucketRef: [7]
    name:
    namespace:
  secretRef: [8]
  accessMode: {"rw", "ro"} [9]
status:
  bucketAttributes: <map[string]string> [10]
  phase: {"Bound", "Released", "Failed", "Errored"} [11]
  conditions:
```
1. `name`: Generated in the pattern of `“bucket-”<BUCKET-NAMESPACE>"-"<BUCKET-NAME>`
1. `labels`: central controller adds the label to its managed resources for easy CLI GET ops.  Value is the driver name returned by GetDriverInfo() rpc*.
1. `finalizers`: COSI controller adds the finalizer to defer Bucket deletion until backend deletion ops succeed.
1. `bucketClassName`: Name of the associated BucketClass
1. `supportedProtocols`:  String array of protocols (e.g. s3, gcs, swift, etc.) supported by the associated object store.
1. `releasePolicy`: the release policy defined in the associated BucketClass. (see [BucketClass](#BucketClass) for more information)
1. `bucketRef`: the name & namespace of the bound Bucket.
1. `secretRef`: the name of the sidecar-generated secret. It's namespace is assumed `cosi-system`.
1. `accessMode`: The level of access granted to the credentials stored in `secretRef`, one of "read only" or "read/write".
1. `bucketAttributes`: stateful data relevant to the managing of the bucket but potentially inappropriate user knowledge (e.g. user's IAM role name)
1. `phase`: is the current state of the BucketContent:
    - `Bound`: the operator finished processing the request and bound the Bucket and BucketContent
    - `Released`: the Bucket has been deleted, leaving the BucketContent unclaimed.
    - `Failed`: error and all retries have been exhausted.
    - `Retrying`: set when a recoverable driver or kubernetes error is encountered during bucket creation or access granting. Will be retried.

> \* Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.

#### BucketClass

A cluster-scoped custom resource. The BucketClass defines a release policy, and specifies driver specific parameters, such as region, bucket lifecycle policies, etc., as well as the name of the driver.  The driver name is used to filter BuckeContenst meant to be handled by a given driver.  In static bucket workflows, the driver name may be empty if the object bucket is defined.

```yaml
apiVersion: cosi.io/v1alpha1
kind: BucketClass
metadata:
  name: 
provisioner: [1]
supportedProtocols: [2]
accessMode: {"ro", "rw"} [3]
releasePolicy: {"Delete", "Retain"} [4]
bucketContentName: [5]
parameters: string:string [6]
```

1. `provisioner`: Used by sidecars to filter BucketContent objects 
1. `supportedProtocols`: A strings array of protocols the associated object store supports (e.g. swift, s3, gcs, etc.)
1. `accessModes`: Declares the level of access given to credentials provisioned through this class.
1.  `releasePolicy`: Prescribes outcome of a Deletion and Revoke events.
    - `Delete`:  the bucket and its contents are destroyed
    - `Retain`:  the bucket and its contents are preserved, only the user’s access privileges are terminated
- `bucketContentName`: (Optional). An admin defined BucketContent representing a Brownfield or Static bucket.  A non-nil value in this field prevents the BucketClass from being used for Greenfield.
- `parameters`: object store specific key-value pairs passed to the driver.