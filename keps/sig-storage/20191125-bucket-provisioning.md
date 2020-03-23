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

### Greenfield Buckets

#### Create Bucket

1. The user creates a `Bucket` in the app’s namespace.
1. The COSI controller detects the `Bucket` and creates a `BucketContent` containing driver-relevant data.
1. The sidecar detects the `BucketContent` and issues a _CreateBucket()_ rpc with parameters defined in the `BucketClass`.
1. The driver creates the bucket and returns pertinent endpoint, credential, and metadata information.
1. The sidecar creates a Secret in its namespace containing endpoint and credential information and an owner reference to the `BucketContent`.
1. The COSI controller copies the sidecar's secret to the `Bucket`'s namespace.
1. The COSI controller adds a finalizer to the `BucketContent`, `Bucket`, and app secret.
1. The COSI controller sets owner references on the `Bucket` and secret. Note: the `BucketContent` being cluster-scoped does not have an owner reference.
1. The workflow ingests the app Secret to begin operation.

#### Delete Bucket

1. The user deletes their `Bucket`, which blocks due to a finalizer until backend deletion operations complete.
1. The COSI controller detects the update and deletes the `BucketContent`, which also blocks until backend deletion operations complete.
1. The sidecar detects this and issues a _DeleteBucket()_ rpc to the driver.  It passes pertinent data stored in the `BucketContent`.
1. The driver deletes the bucket from the object store and responds with no error.
1. The sidecar sets the `BucketContent`'s status to indicate the delete occurred.
1. The COSI controller deletes the `BucketContent` and app secret.
1. The COSI controller removes the `Bucket`, app secret, and `BucketContent` finalizers so they are garbage collected.

### Brownfield Buckets

#### Grant Access

1. A `BucketClass` is defined specifically referencing an object store bucket.
1. A user creates a `Bucket` in the app namespace, specifying the `BucketClass`.
1. The COSI controller detects the `Bucket` and creates a `BucketContent` object.
1. The sidecar detects the `BucketContent` object and calls the _GrantAccess()_ rpc to the driver, returing a set of credentials for accessing the bucket.
1. The sidecar writes the credentials and endpoint information to a Secret in its namespace, with an owner reference to the BucketContent.
1. The COSI controller copies the sidecar's secret to the `Bucket`'s namespace.
1. The COSI controller adds a finalizer to the `BucketContent`, `Bucket`, and app secret.
1. The COSI controller sets owner references on the `Bucket` and secret. Note: the `BucketContent` being cluster-scoped does not have an owner reference.
1. The workflow ingests the Secret to begin operation.

#### Revoke Access

1. The user deletes the `Bucket`, which blocks until revoke operations complete.
1. The COSI controller detects the update and deletes the `BucketContent`, which also blocks until backend revoke operations complete.
1. The sidecar detects the `BucketContent` update and invokes the _RevokeAccess()_ rpc method.
1. The driver removes access for the associated credentials.
1. The sidecar removes the finalizer from and deletes its Secret.
1. The sidecar sets the `BucketContent`'s status to indicate the revoke occurred.
1. The COSI controller deletes the `BucketContent` and app secret.
1. The COSI controller removes the `Bucket`, app secret, and `BucketContent` finalizers so they are garbage collected.

### Static Buckets

> Note: No driver, and thus no sidecar, are present to manage provisioning.

#### Grant Access

1. A `BucketClass` is defined specifically naming an object store bucket.
1. This `BucketClass` also names an app-based secret and its namespace.
1. A user creates a `Bucket` in the app namespace, specifying the `BucketClass`.
1. The COSI controller detects the `Bucket`, sees the `BucketClass`'s secret, and creates a `BucketContent`.
1. The COSI controller copies the secret referenced in the `BucketClass` to the `Bucket`'s namespace.
1. The COSI controller adds a finalizer to the `BucketContent`, `Bucket`, and app secret.
1. The COSI controller sets owner references on the `Bucket` and secret. Note: the `BucketContent` being cluster-scoped does not have an owner reference.
1. The workflow ingests the Secret to begin operation.

#### Revoke Access

1. The user deletes the `Bucket`, which blocks until revoke operations complete.
1. The COSI controller deletes the associated `BucketContent` and app secret.
1. The COSI controller removes the `Bucket` and app secret finalizers so they are garbage collected.

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
  protocol: [6]
  accessMode: {"ro", "rw"} [7]
status:
  bucketContentName: [8]
  phase:
  conditions: 
```
1. `labels`: COSI controller adds the label to its managed resources to easy CLI GET ops.  Value is the driver name returned by GetDriverInfo() rpc. Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.
1. `finalizers`: COSI controller adds the finalizer to defer `Bucket` deletion until backend deletion ops succeed.
1. `bucketPrefix`: (Optional) prefix prepended to a randomly generated bucket name, eg. "YosemitePhotos-". If empty no prefix is appended.
1. `bucketClassName`: Name of the target `BucketClass`.
1. `secretName`: Desired name for user's credential Secret. API validation for unique name. Defining this name allows for a single manifest workflow. Of course, there is a window here where API validation passes but the secret creation fails due to an existing secret of the same name just created. Attempting to create the user's (app's) secret will continue until a timeout occurs.
1. `protocol`: (Optional) String array of protocols (e.g. s3, gcs, swift, etc.) requested by the user. If the specified `BucketClass` does not match one of the protocols then `Bucket` creation fails. If omitted then the `BucketClass` is assumed correct.
1. `accessMode`:  (Optional) The requested level of access provided to the returned access credentials. If omitted "rw" is assumed.
1. `bucketContentName`: Name of a bound `BucketContent`.


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
  bucketClassName: [4]
  supportedProtocols: [5]
  releasePolicy: {"Delete", "Retain"} [6]
  bucketRef: [7]
    name:
    namespace:
  secretName: [8]
  accessMode: {"rw", "ro"} [9]
status:
  bucketAttributes: <map[string]string> [10]
  phase: {"Bound", "Released", "Failed", "Errored"} [11]
  conditions:
```
1. `name`: Generated in the pattern of `“bucket-”<BUCKET-NAMESPACE>"-"<BUCKET-NAME>`. We may validate the length of the `Bucket` name and namespace to ensure that this _metadata.name_ fits.
1. `labels`: central controller adds the label to its managed resources for easy CLI GET ops.  Value is the driver name returned by GetDriverInfo() rpc. Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.
1. `finalizers`: COSI controller adds the finalizer to defer Bucket deletion until backend deletion ops succeed.
1. `bucketClassName`: Name of the associated `BucketClass`.
1. `supportedProtocols`:  String array of protocols (e.g. s3, gcs, swift, etc.) supported by the associated object store.
1. `releasePolicy`: the release policy defined in the associated BucketClass (see [BucketClass](#BucketClass) for more information).
1. `bucketRef`: the name & namespace of the associated `Bucket`. For both brownfield cases, `bucketRef` is set by an admin and names the bucket to be accessed. For greenfield, it is added by the COSI controller.
1. `secretName`: the name of the app secret in the `Bucket`'s namespace, as define in `bucketRef.namespace`. For static brownfield, `secretName` is set by an admin; otherwise it is added by the COSI controller.
1. `accessMode`: The level of access granted to the credentials stored in `secretRef`, one of "read only" or "read/write".
1. `bucketAttributes`: stateful data relevant to the managing of the bucket but potentially inappropriate user knowledge (e.g. user's IAM role name).
1. `phase`: is the current state of the `BucketContent`:
    - _Bound_: the operator finished processing the request and bound the `Bucket` and `BucketContent`
    - _Released_: the `Bucket` has been deleted, leaving the `BucketContent` unclaimed.
    - _Failed_: error and all retries have been exhausted.
    - _Retrying_: set when a recoverable driver or Kubernetes error is encountered during bucket creation or access granting. Will be retried.

#### BucketClass

A cluster-scoped custom resource used to describe both greenfield and brownfield buckets.
The `BucketClass` defines a release policy, and specifies driver specific parameters, such as region, bucket lifecycle policies, etc., as well as the driver (provisioner) name. The driver name is used to filter `BucketContent` events.
In dynamic brownfield workflows, the `BucketClass` contains a reference to a `BucketContent` object which names the existing bucket.
In static brownfield workflows, the `provisioner` field is empty, a reference to a `BucketContent` is needed, and the secret used to grant access to the bucket must be specified.

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
secretRef: [6]
  name:
  namespace:
parameters: string:string [7]
```

1. `provisioner`: (Optional) The name of the driver in the form _"driver-namespace/driver-name"_. If supplied the driver container and sidecar container are expected to be deployed in the same pod in the supplied namespace. If omitted the `bucketContentName` is required for static provisioning.
1. `supportedProtocols`: (Optional) A strings array of protocols the associated object store supports (e.g. swift, s3, gcs, etc.). If empty then protocol checking is skipped.
1. `accessModes`: (Optional) Declares the level of access given to credentials provisioned through this class. If empty then "rw" is assumed.
1.  `releasePolicy`: Prescribes outcome of a Delete events. **Note:** `releasePolicy` is ignored for all brownfield cases.
    - _Delete_:  the bucket and its contents are destroyed
    - _Retain_:  the bucket and its contents are preserved, only the user’s access privileges are terminated
    - _Reuse_ :  TBD
    - _Erase_ :  TBD
1. `bucketContentName`: (Optional). An admin defined `BucketContent` representing a brownfield or static bucket. A non-empty value in this field indicates brownfield provisioning. An empty value indicates greenfield provisioning.
1. `secretRef`: (Optional) The name and namespace of an existing secret to be copied to the `Bucket`'s namespace for static brownfield provisioning. If omitted then provisioning is either greenfield or dynamic brownfield, depending on `bucketContentName`.
1. `parameters`: (Optional) Object store specific key-value pairs passed to the driver.

#### COSIRegistration

A cluster-scoped custom resource used to register COSI drivers. It is created by the sidecar and primarily used to guarantee unique driver names. The sidecar is expected to delete this resource upon termination.

```yaml
apiVersion: cosi.io/v1alpha1
kind: COSIRegistration
metadata:
  name: [1]
driverNamespace: [2]
flags: [3]
```

1. `name`: The name here must match the name of the driver, which means that driver names follow Kubernetes naming rules.
1. `driverNamespace`: The name of the driver's namespace.
1. `flags`: (Optional) string:string map. The flags passed to the driver. # or do we want sidecar flags?

