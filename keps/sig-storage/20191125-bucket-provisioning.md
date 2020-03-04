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
    - [Create Bucket](#create-bucket)
    - [Delete Bucket](#delete-bucket)
    - [Grant Bucket Access](#grant-bucket-access)
    - [Revoke Bucket Access](#revoke-bucket-access)
  - [Custom Resource Definitions](#custom-resource-definitions)
      - [ObjectBucketClaim (OBC)](#objectbucketclaim-obc)
      - [ObjectBucket (OB)](#objectbucket-ob)
      - [BucketClass](#bucketclass)
<!-- /toc -->

# Summary

This proposal introduces the Container Object Storage Interface (COSI), whose purpose is to provide a familiar and standardized method of provisioning object storage buckets in an manner agnostic to the storage vendor. The COSI environment is comprised of Kubernetes CRDs, operators to manage these CRDs, and a gRPC interface by which these operators communicate with vendor plugins.  This design is of course heavily inspired by the Kubernetes’ implementation of the Container Storage Interface (CSI).  However, bucket management lacks some of the notable requirements of block and file provisioning and allows for an architecture with lower overall complexity.  This proposal does not include a standardized protocol or abstraction of storage vendor APIs.  

## Motivation 

File and block are first class citizens within the Kubernetes ecosystem.  Object, though different on a fundamental level and lacking an open, committee controlled interface like POSIX, is a popular means of storing data, especially against very large data sources.   As such, we feel it is in the interest of the community to elevate buckets to a community supported feature.  In doing so, we can provide Kubernetes cluster users and administrators a normalized and familiar means of managing object storage.

## Goals
+ Define a control plane API in order to standardize and formalize a Kubernetes bucket provisioning
+ Minimize privileges required to run a storage driver.
+ Minimize technical ramp-up for storage vendors.
+ Be un-opinionated about the underlying object-store.
+ Use Kubernetes Secrets to inject bucket information into application workflows.
+ Present similar workflows for both _greenfield_  and _brownfield_ bucket provisioning.
+ Present a design that is familiar to CSI storage plugin authors and Kubernetes storage admins.

## Non-Goals
+ Define a native _data-plane_ object store API which would greatly improve object store app portability.

## Vocabulary
+  _Brownfield_ - buckets created outside the COSI system but granted access via COSI.
+ _Bucket_ - A namespace where objects reside, similar to a flat POSIX directory.
+ _BucketClass_ (BC) - A provisioner-namespaced custom resource containing fields defining the provisioner and immutable parameter set(s).  Referenced by ObjectBucketClaims and populated into the OB.  Only used for provisioning new buckets.
+ _Container Object Storage Interface (COSI)_ -  The specification of gRPC data and methods making up the communication protocol between the plugin and the sidecar.
+ _COSI Controller_ - A single, centralized controller which manages OBCs, OBs, and Secrets cluster-wide.
+ _Greenfield_ - new buckets dynamically created by a provisioner (plugin).
+ _OB_ (Object Bucket) - A provisioner-namespaced custom resource representing the provisioned bucket and relevant metadata.
+ _OBC_ (Object Bucket Claim) - A user-namespaced custom resource representing a user’s bucket request. 
+ _Object_ - An atomic, immutable unit of data stored in buckets.
+ _Plugin_ - A containerized gRPC server which implements a storage vendor’s business logic through the COSI interface. It can be written in any language supported by gRPC and is independent of Kubernetes.
+ _Sidecar_ - A controller deployed in the same pod as the Plugin, responsible for managing OBs within its namespace and communicating with the Plugin via gRPC. 

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

+ The COSI Controller runs in a protected namespace with RBAC privileges for managing OBCs, OBs, and Secrets cluster wide.
  
    + Responsible for the binding relationship of OBs and OBCs
+ The Plugin and Sidecar containers run together in a single Pod, which may be managed by a Deployment.
    + The gRPC connection is made through the Pod’s localhost.
+ No node affinity or other requirements exist.
+ Operations must be idempotent in order to handle failure recovery.
  
## Workflows

### Create Bucket
1. The user creates an OBC in the app’s namespace.
1. The Controller detects the OBC and creates an OB in the Plugin’s namespace, containing provisioner-relevant data collected from the OBC and BC.
1. The Sidecar detects the OB and issues a CreateBucket() rpc, passing the provisioner-relevant data.
1. The Plugin creates the bucket and returns pertinent endpoint, credential, and metadata.
1. The Sidecar creates a secret in its namespace containing essential connection information.
1. The Controller copies the secret to the OBC’s namespace and the app is ready to run.

### Delete Bucket
1. The user deletes the OBC, which blocks until the finalizer is removed.
1. The Controller deletes the OB, which also blocks until its finalizer is removed.
1. The Sidecar detects this and issues a DeleteBucket() rpc to the Plugin.  It passes data stored in the OB.
1. The Plugin deletes the bucket and responds with no errors.
1. The Sidecar deletes the OB’s secret.
1. The Controller deletes the secret’s copy in the app’s namespace.
1. The Controller removes the finalizers from the OBC, OB, and Secret allowing them to be deleted.

### Grant Bucket Access
Reminder: BucketClass is ignored since it is used only for dynamic provisioning.

1. An OB is manually created in the Plugin’s namespace with at least enough information to identify an existing bucket in the object store.
1. The Controller detects the new OB and sets its status to “Available”.
1. The user creates an OBC in the app’s namespace which references this pre-existing OB.
1. The Controller detects the OBC and updates OB Phase and OBC reference.
1. The Sidecar detects the OB update and calls the `Grant()` gRPC method.
1. The Plugin grants access to the bucket and returns pertinent endpoint, credential, and metadata.
1. The Sidecar creates a secret in its namespace containing essential connection information.
1. The Controller copies the secret to the OBC’s namespace and the app is ready to run.

### Revoke Bucket Access

Reminder: BucketClass is ignored in browfield operations.

1. The user deletes the OBC, which blocks until the finalizer is removed.
1. The Controller detects the OBC change and updates OB’s Phase.
1. The Sidecar detects the OB update and calls the `Revoke()` gRPC method.
1. The Plugin revokes access to the bucket and responds with no errors.
1. The Sidecar deletes the OB’s secret.
1. The Controller deletes the secret’s copy in the app’s namespace.
1. The Controller removes the finalizers from the OBC, OB, and Secret.
1. The OBC and its secret are garbage collected and the OB remains.

##  Custom Resource Definitions

#### ObjectBucketClaim (OBC)

A user facing API object representing a request for a bucket, or access to an existing bucket. OBCs are created by users in their namespaces. Once the request is fulfilled, the OBC is “bound” to an Object Bucket (OB). The binding is used to mark the request as fulfilled and prevent further binds to the OB.


```yaml
apiVersion: cosi.io/v1alpha1
kind: ObjectBucketClaim
metadata:
  name:
  namespace:
  labels:
    “cosi.io/plugin”: [1]
  finalizers:
  - cosi.io/finalizer [2]
spec:
  bucketName: [3]
  generateBucketName: [4]
  bucketClassRef: [5]
    name:
    namespace:
  objectBucketRef: [6]
    name:
    namespace:
  bucketConfig: [7]
status:
  phase:
  conditions: []ObjectBucketClaimCondition
```
1. `labels`: COSI Controller adds the label to its managed resources for easy GET ops.  Value is the plugin name returned by GetDriverInfo() rpc*.
1. `finalizers`: COSI Controller adds the finalizer to defer OBC deletion until backend deletion ops succeed.
1. `bucketName`: Desired name of the bucket to be created**.
1. `generateBucketName`: Prefix to a randomly generated name. Mutually exclusive with `bucketName`**.
1. `bucketClassRef`: Name and Namespace of the target BucketClass**.
1. `objectBucketRef`: Name and Namespace of a bound OB.

   - Injected by the controller during greenfield ops.

   - Defined by the OBC author for brownfield ops.

1. `bucketConfig`: Map of key-values defined per provisioner. 

   - Keys here override `BucketClass.defaultConfig` keys and are overridden by `BucketClass.enforceConfig` keys.


\* Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.

** Ignored for brownfield usage.

#### ObjectBucket (OB)
An admin facing API object representing the bucket. The OB resides in the same namespace as the plugin, and is typically not seen by bucket consumers. The OB is expected to store stateful data relevant to bucket deprovisioning.

```yaml
apiVersion: cosi.io/v1alpha1
kind: ObjectBucket
Metadata:
  name: [1]
  namespace: [2]
  labels:
    “cosi.io/plugin”: [3]
  finalizers:
  - "cosi.io/finalizer" [4]
spec:
  bucketClassRef: [5]
     name:
     namespace:
  releasePolicy: {"Delete", "Retain"} [6]
  bucketConfig: map[string]string [7]
  objectBucketClaimRef: [8]
    name:
    namespace:
  secretRef: [9]
    name:
status:
  bucketMetaData: <map[string]string> [10]
  phase: {"Bound", "Released", "Failed", “Errored”} [11]
  conditions: []ObjectBucketCondition
```
1. `name`: Generated in the pattern of “obc-”\<OBC-NAMESPACE>"-"\<OBC-NAME>
1. `namespace`: The namespace of the Plugin.
1. `labels`: COSI Controller adds the label to its managed resources for easy GET ops.  Value is the plugin name returned by GetDriverInfo() rpc*.
1. `finalizers`: Set and cleared by the COSIController and prevents accidental deletion of an OB.
1. `bucketClassRef`: Name and namespace of the bucket class
1. `releasePolicy`: release policy from the Bucket Class referenced in the OBC. See `BucketClass` spec for values.
1. `bucketConfig`: a string:string map of plugin defined key-value pairs
1. `objectBucketClaimReference`: the name & namespace of the bound OBC.
1. `secretReference`: the name & namespace of Sidecar-generated secret. 
1. `bucketMetaData`: stateful data relevant to the managing of the bucket but potentially inappropriate user knowledge (e.g. the user’s in-store user name)
1. `phase`: is the current state of the ObjectBucket:
    - `Bound`: the operator finished processing the request and linked the OBC and OB
    - `Released`: the OBC has been deleted, leaving the OB unclaimed.
    - `Failed`: error and all retries have been exhausted.
    - `Retrying`: set when a recoverable plugin or kubernetes error is encountered during bucket creation or access granting. Will be retried.


\* Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.

#### BucketClass

During greenfield workflows, an OBC references a Bucket Class(BC).  The bucket class defines a release policy, and specifies plugin specific parameters, such as region, bucket lifecycle policies, etc. as well as the name of the plugin as returned by the GetDriverInfo() rpc.  The plugin name is used to filter OBs meant to be handled by the given plugin.

```yaml
apiVersion: cosi.io/v1alpha1
kind: BucketClass
metadata:
  name: 
  namespace: [1]
plugin: [2]
defaultConfig: string:string [3]
enforceConfig: string:string [4]
releasePolicy: {"Delete", "Retain"} [5]
```

1. `namespace`: BucketClasses are co-namespaced with their associated plugin
1. `plugin`: Name of the plugin, provided via the GetDriverInfo() rpc. Used to filter OBs.
1. `defaultConfig`: key-value pairs used as defaults by the plugin. May be overridden by OBCs.
1. `enforceConfig`: key-value pairs which override OBC keys of the same name (e.g. max object count)
1.  `releasePolicy`: Prescribes outcome of an OBC/OB deletion.
    - `Delete`:  the bucket and its contents are destroyed
    - `Retain`:  the bucket and its contents are preserved, only the user’s access privileges are terminated


