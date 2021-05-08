# Object Storage Support

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [User Stories](#user-stories)
    - [Admin](#admin)
    - [User](#user)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Vocabulary](#vocabulary)
- [Proposal](#proposal)
  - [APIs](#apis)
    - [Storage APIs](#storage-apis)
      - [BucketRequest](#bucketrequest)
      - [Bucket](#bucket)
      - [BucketClass](#bucketclass)
    - [Access APIs](#access-apis)
      - [BucketAccessRequest](#bucketaccessrequest)
      - [BucketAccess](#bucketaccess)
      - [BucketAccessClass](#bucketaccessclass)
    - [App Pod](#app-pod)
    - [Topology](#topology)
- [Object Relationships](#object-relationships)
- [Workflows](#workflows)
    - [Finalizers](#finalizers)
  - [Create Bucket](#create-bucket)
  - [Sharing COSI Created Buckets](#sharing-cosi-created-buckets)
  - [Delete Bucket](#delete-bucket)
  - [Grant Bucket Access](#grant-bucket-access)
  - [Revoke Bucket Access](#revoke-bucket-access)
  - [Delete BucketAccess](#delete-bucketaccess)
  - [Delete Bucket](#delete-bucket-1)
  - [Setting Access Permissions](#setting-access-permissions)
    - [Dynamic Provisioning](#dynamic-provisioning)
    - [Static Provisioning](#static-provisioning)
- [gRPC Definitions](#grpc-definitions)
  - [ProvisionerGetInfo](#provisionergetinfo)
  - [ProvisonerCreateBucket](#provisonercreatebucket)
  - [ProvisonerDeleteBucket](#provisonerdeletebucket)
  - [ProvisionerGrantBucketAccess](#provisionergrantbucketaccess)
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

# Release Signoff Checklist

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [X] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website


# Summary

This proposal introduces the *Container Object Storage Interface* (COSI), a system composed of Custom Resources (CRs), APIs for bucket provisioning and access, a controller automation architecture, a gRPC specification, and "COSI node adapter" that interfaces with the kubelet on each node. These components combine to support a standard object storage representation in Kubernetes.  

This KEP describes the above components, defines our goals and target use-cases, and sets the scope of the proposal by defining higher level objectives.  The vocabulary section defines terminology.  Relationships between the APIs are provided to illustrate the interconnections between object storage APIs, users' workloads, and object store service instances.  Lastly, the document describes the proposed API specs, the create and delete workflows, and the gRPC spec.

# Motivation

File and block are first class citizens within the Kubernetes ecosystem.  Object, though very different under the hood, is a popular means of storing data, especially against very large data sources.   As such, we feel it is in the interest of the community to integrate object storage into Kubernetes, supported by the SIG-Storage community.  In doing so, we can provide Kubernetes cluster users and administrators a normalized and familiar means of managing object storage. 
When a workload (app pod, deployment, configs) is moved to another cluster, as long as the new cluster runs a COSI controller that supports the same object protocol, then the workload can be moved to a new cluster without requiring any changes in the user manifests.

## User Stories

### Admin

- As a cluster administrator, I can control access to new and existing buckets when accessed from the cluster, regardless of the backing object store.

### User

- As a developer, I can define my object storage needs in the same manifest as my workload, so that deployments are streamlined and encapsulated within the Kubernetes interface.
- As a developer, I can define a manifest containing my workload and object storage configuration once, so that my app may be ported between clusters as long as the storage provided supports my designated data path protocol.
- As a developer, I want to create a workload controller which is bucket API aware, so that it can dynamically connect workloads to underlying object storage.

## Goals

+ Specify object storage Kubernetes APIs for the purpose of orchestrating/managing object store operations.
+ Implement a Kubernetes controller automation design with support for pluggable provisioners.
+ Support workload portability across clusters.
+ As MVP, be accessible to the largest groups of consumers by supporting the major object storage protocols (S3, Google Cloud Storage, Azure Blob) while being extensible for future protocol additions..
+ Present similar workflows for both greenfield and brownfield bucket operations.

## Non-Goals

+ Defining the _data-plane_ object store protocol to replace or supplement existing vendor protcols/APIs is not within scope.
+ Object store deployment/management is left up to individual vendors.
+ Bucket access lifecycle management is not within the scope of this KEP.  ACLs, access policies, and credentialing need to be handled out-of-band.

# Vocabulary

+ _backend bucket_ - any bucket provided by the object store system, completely separate from Kubernetes.
+ _brownfield bucket_ - an existing backend bucket.
+ _Bucket (Bucket instance)_ - a cluster-scoped custom resource referenced by a `BucketRequest` and containing connection information and metadata for a backend bucket.
+ _BucketAccess (BA)_ - a cluster-scoped custom resource for granting bucket access.
+ _BucketAccessClass (BAC)_ - a cluster-scoped custom resource containing fields defining the provisioner and a ConfigMap reference where policy is defined.
+ _BucketAccessRequest (BAR)_ - a user-namespaced custom resource representing a request for access to an existing bucket.
+ _BucketClass (BC)_ - a cluster-scoped custom resource containing fields defining the provisioner and an immutable parameter set for creating new buckets.
+ _BucketRequest (BR)_ - a user-namespaced custom resource representing a request for a new backend bucket.
+ _COSI_ - Container _Object_ Store Interface, modeled after CSI.
+ _cosi-node-adapter_ - a pod per node which receives Kubelet gRPC _NodeStageVolume_ and _NodeUnstageVolume_ requests, ensures the target bucket has been provisioned, and notifies the kubelet when the pod can be run.
+ _driver_ - a container (usually paired with a sidecar container) that is responsible for communicating with the underlying object store to create, delete, grant access to, and revoke access from buckets. Drivers talk gRPC and need no knowledge of Kubernetes. Drivers are typically written by storage vendors, and should not be given any access outside of their namespace.
+ _greenfield bucket_ - a new backend bucket created by COSI.
+ _green-to-brownfield_ - a use-case where COSI creates a new bucket on behalf of a user, and then supports ways for other users in the cluster to share this bucket.
+ _provisioner_ - a generic term meant to describe the combination of a sidecar and driver. "Provisioning" a bucket can mean creating a new bucket or granting access to an existing bucket.
+ _sidecar_ - a Kubernetes-aware container (usually paired with a driver) that fulfills COSI requests with the driver via gRPC calls. The sidecar's access should be restricted to the namespace where it resides, which is expected to be the same namespace as the provisioner. The COSI sidecar is provided by the Kubernetes community.
+ _static provisioning_ - the admin manually creates the `Bucket` instance representing properties of the target backend bucket.

# Proposal

## APIs

### Storage APIs

#### BucketRequest

A user-facing, namespaced, immutable, custom resource requesting the creation of a new bucket. A `BucketRequest` (BR) lives in the app's namespace.  In addition to a BR, a [BucketAccessRequest](#bucketaccessrequest) is necessary in order to grant credentials to access the bucket. BRs are used only for greenfield.

There is a 1:1 mapping of a `BucketRequest` and the cluster scoped [Bucket](#bucket) instance, meaning there is a single `Bucket` instance for every BR. **Note**: in brownfield uses, where a `Bucket` is manually created, there is no need for a BR; `BucketAccessRequests` are used in this case.

```yaml
apiVersion: objectstorage.k8s.io/v1alpha1
kind: BucketRequest
metadata:
  name:
  namespace:
  labels:
    - objectstorage.k8s.io/provisioner: [1]
  finalizers:
    - objectstorage.k8s.io/bucketrequest-protection [2]
spec:
  bucketPrefix: [3]
  bucketClassName: [4]
status:
  bucketName: [5]
  bucketAvailable: [6]
```

1. `labels`: added by COSI.  Key’s value should be the provisioner name. Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.
1. `finalizers`: added by COSI to defer `BucketRequest` deletion until backend deletion succeeds.
1. `bucketPrefix`: (optional) prefix prepended to a COSI-generated, random, backend bucket name, eg. “yosemite-photos-".
1. `bucketClassName`: (optional) name of the `BucketClass` used to provision this request. If omitted, a default bucket class is used. If the bucket class is missing and a default cannot be found, an error is logged and the request is retried (with exponential backoff).
1. `bucketName`: name of the cluster-wide `Bucket` instance. 
1. `bucketAvailable`: if true the bucket has been provisioned. If false then the bucket has not been provisioned and is unable to be accessed.

#### Bucket

A cluster-scoped, custom resource representing the abstraction of a single backend bucket. The relevant [BucketClass](#bucketclass) fields are copied to the `Bucket`, so that the `Bucket` instance reflects the `BucketClass` at the time of `Bucket` creation.

There is a 1:1 mapping between a (BucketRequest)[#bucketrequest] and a `Bucket`, but a `Bucket` can exist without a BR, which is the brownfield use case. There is a 1:many mapping between a `Bucket` and [BucketAccess](#bucketaccess) instances.

For greenfield, COSI creates the `Bucket` based on values in the `BucketRequest` and `BucketClass`, and two-way links the `Bucket` to the BR. For brownfield access, an admin manually creates the `Bucket`, filling in fields, such as _allowedNamespaces_ and _provisoner_. COSI populates fields returned by the driver, and will link the [BucketAccess](#bucketaccess) to the `Bucket` instance.

> Note: a `Bucket` instance is immutable except for _allowedNamespaces_ and _deletionPolicy_.

```yaml
apiVersion: objectstorage.k8s.io/v1alpha1
kind: Bucket
metadata:
  name: [1]
  labels: [2]
    objectstorage.k8s.io/provisioner:
    objectstorage.k8s.io/bucket-prefix:
  finalizers: [3]
    - objectstorage.k8s.io/br-protect
    - objectstorage.k8s.io/ba-<name1>
    - objectstorage.k8s.io/ba-<name2> ...
spec:
  provisioner: [4]
  deletionPolicy: [5]
  bucketClassName: [6]
  bucketRequest: [7]
  allowedNamespaces: [8]
    - name:
  bucketID: [9]
  protocol: [10]
    azureBlob:
      storageAccount:
    s3:
      region:
      signatureVersion:
    gs:
      privateKeyName:
      projectId:
      serviceAccount:
  parameters: [11]
status:
  bucketAvailable: [12]
  bucketID: [13]
```

1. `name`: when created by COSI, the `Bucket` name is generated in one of these formats: _"br-"+uuid_ (if BR.prefix is empty), or BR.prefix+"-"+uuid (if prefix is supplied). If an admin creates a `Bucket`, as is necessary for static brownfield access, any unique name is allowed.
1. `labels`: added by COSI. The "objectstorage.k8s.io/provisioner" key's value is the provisioner name. The "objectstorage.k8s.io/bucket-prefix" key's value is the `BucketRequest.bucketPrefix` value when specified. Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.
1. `finalizers`: added by COSI to defer `Bucket` deletion until backend deletion succeeds. Implementation may add one finalizer for the BR and one for each BA pointing to this `Bucket`. **Note**: a `BucketAccess` object has one finalizer per accessing pod. So there is no access to a bucket when all `Bucket` finalizers have been removed.
1. `provisioner`: (optional) The provisioner field as defined in the `BucketClass`.
1. `deletionPolicy`: defines the outcome when a greenfield `BucketRequest` is deleted, and is ignored for brownfield buckets. 
   - _Retain_: keep both the `Bucket` instance and the backend bucket along with its contents. This allows for admins to cleanup/migrate the data before manually deleting the `Bucket` instance and subsequently the backend bucket.
   - _Delete_: Wait for all access to the `Bucket` to complete, then delete both the `Bucket` instance and the backend bucket. This includes all content in that bucket. If, for example, a running pod or a `BucketAccess` instance refer to the `Bucket` instance, then the delete is deferred until all accessors have been deleted.
   - _ForceDelete_: Don't wait for accessors/workloads to complete using the bucket, but force-delete both the `Bucket` instance and the backend bucket, including all content in that bucket. BucketAccess referencing this bucket and their generated credentials are also forcibly deleted. The remaining accessors to that Bucket are eventually cleaned up by controllers listening for `BucketAccess` events.
1. `bucketClassName`: (set by COSI for greenfield, and ignored for brownfield) Name of the associated bucket class.
1. `bucketRequest`: (set by COSI for greenfield, ignored for brownfield) an `objectReference` structure referencing the associated `BucketRequest`.
1. `allowedNamespaces`: an array of namespaces that are permitted to create new buckets.
1. `bucketID`: (optional) the string ID of the backend bucket used for manually created `Bucket`s.
1. `protocol`: protocol-specific field the application will use to access the backend storage.
     - `azureBlob`: data required to target a provisioned azure container.
     - `gs`: data required to target a provisioned GS bucket.
     - `s3`: data required to target a provisioned S3 bucket.
1. `parameters`: a copy of the BucketClass parameters.
1. `bucketAvailable`: if true the bucket has been provisioned. If false then the bucket has not been provisioned and is unable to be accessed.

#### BucketClass

An immutable, cluster-scoped, greenfield-only, custom resource to provide admins control over the handling of bucket provisioning. The `BucketClass` (BC) is referenced by a [BucketRequest](#bucketrequest) (BR) for the purpose of instantiating a [Bucket](#bucket) and creating a new backend bucket. A BC defines a deletion policy, driver specific parameters, and the provisioner name. A list of allowed namespaces can be specified to restrict bucket creation to specific namespaces. A default bucket class can be defined which allows the bucket class to be omitted from a BR. Relevant BC fields are copied to the `Bucket` instance to handle the case of the BC being deleted and re-created.  If an object store supports more than one protocol then the admin should create a `BucketClass` per protocol.

A `BucketClass` is required for greenfield use cases. It is not used for brownfield since the information in the BC will be defined in the `Bucket` instance.

> Note: the `BucketClass` object is immutable except for the `isDefault` annotation.

```yaml
apiVersion: objectstorage.k8s.io/v1alpha1
kind: BucketClass
metadata:
  name: 
  annotations:
    objectstorage.k8s.io/is-default-class: [1]
provisioner: [2]
protocol: [3]
  azureBlob:
    storageAccount:
  s3:
    region:
    signatureVersion:
  gs:
    privateKeyName:
    projectId:
    serviceAccount:
deletionPolicy: [4]
allowedNamespaces: [5]
  - name:
parameters: [6]
```

1. `isDefault`: boolean annotaion. If omitted the setting is false. If present but lacking a value the setting is true. If true then a `BucketRequest` may omit the `BucketClass` reference.
1. `provisioner`: (optional) the name of the vendor-specific driver supporting the `protocol`. If empty then there is no driver/sidecar, and the admin has to create the `Bucket` and `BucketAccess` instances manually.
1. `protocol`: protocol-specific field the application will use to access the backend storage. The admin is expected to only include the desired protocol name (e.g. "s3") and not the other details listed. Note: these protocol details are defined in the `Bucket` instance.
     - `azureBlob`: data required to target a provisioned azure container.
     - `gs`: data required to target a provisioned GS bucket.
     - `s3`: data required to target a provisioned S3 bucket.
1. `deletionPolicy`: (required) defines the outcome when a greenfield `BucketRequest` is deleted, and is ignored for brownfield buckets.
   - _Retain_: keep both the `Bucket` instance and the backend bucket and its contents. The `Bucket` instance may be marked unavailable and it may potentially be reused (post MVP).
   - _Delete_: when all access to the `Bucket` is complete, delete both the `Bucket` instance and the backend bucket, including all content in that bucket. If, for example, a running pod or a `BucketAccess` instance refer to the `Bucket` instance, then the delete is deferred until all accessors have been deleted.
   - _ForceDelete_: delete both the `Bucket` instance and the backend bucket, including all content in that bucket, ignoring any accessors or workloads that may be referencing the `Bucket`. BucketAccess referencing this bucket and their generated credentials are also forcibly deleted. 
1. `allowedNamespaces`: a list of namespaces that are permitted to either create new buckets or to access existing buckets. This list is static for the life of the `BucketClass`, but the `Bucket` instance's list of allowed namespaces can be mutated by the admin. 
1. `parameters`: (optional) a map of string:string key values.  Allows admins to set provisioner key-values.

> Note: these protocol details are defined in the `Bucket` instance.

### Access APIs

The Access APIs abstract the backend policy system.  Access policy and user identities are an integral part of most object stores.  As such, a system must be implemented to manage both user/credential creation and the binding of those users to individual buckets via policies.  Object stores differ from file and block storage in how they manage users, with cloud providers typically integrating with an IAM platform.  This API includes support for cloud platform identity integration with Kubernetes ServiceAccounts.  On-prem solutions usually provide their own user management systems, which may look very different from each other and from IAM platforms.  We also account for third party authentication solutions that may be integrated with an on-prem service.

#### BucketAccessRequest

A user namespaced, immutable custom resource representing a request to access an existing backend bucket. A `BucketAccessRequest`(BAR) triggers the instantiation of a [BucketAccess](#bucketaccess) object, which is the access connection to the backend bucket. A BAR references a [BucketAccessClass](#bucketaccessclass) which defines access policy. A BAR optionally defines a ServiceAccount, which supports provisioners that implement cloud provider identity integration with their respective Kubernetes offerings. To connect a BAR to the target backend bucket, it can either reference a [BucketRequest](#bucketrequest) (BR) in its namespace, or a [Bucket](#bucket) instance directly when residing in a different namespace from that of the BR. The workload pod references the BAR in its `volumes.csi.volumeAttributes` section.

There is a 1:1 mapping between a BAR and a `BucketAccess`instance. Many workload pods can reference the same BAR within the same namespace.

```yaml
apiVersion: objectstorage.k8s.io/v1alpha1
kind: BucketAccessRequest
metadata:
  name:
  namespace:
  labels:
    objectstorage.k8s.io/provisioner: [1]
  finalizers:
  - objectstorage.k8s.io/bucketaccessrequest-protection [2]
spec:
  bucketAccessClassName: [3]
  serviceAccountName: [4]
  bucketRequestName: [5]
  bucketName: [6]
status:
  bucketAccessName: [7]
  accessGranted: [8]
```

1. `labels`: added by COSI.  Key’s value should be the provisioner name. Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.
1. `finalizers`: added by COSI to defer `BucketAccessRequest` deletion until backend deletion succeeds.
1. `bucketAccessClassName`: (required) name of the `BucketAccessClass` specifying the desired set of policy actions to be set for a user identity or ServiceAccount.
1. `serviceAccountName`: (optional) the name of a Kubernetes ServiceAccount, in the same namespace, which is expected to be linked to an identity in the cloud provider, such that a pod using the ServiceAccount is authenticated as the linked identity. In this case, COSI applies the policy defined in the `BAC.policyActionsConfigMap` to the identity. If `serviceAccountName` is omitted, a "minted" user is generated by the driver and stored as `BucketAccess.status.accountID`.
1. `bucketRequestName`: (optional) the name of the `BucketRequest` associated with this access request within the same namespace as the BAR. From the BR, COSI knows the `Bucket` instance and thus the backend bucket and its properties. `bucketRequestName` and `BucketName` are mutually exclusive.
1. `bucketName`: (optional) the name of the `Bucket` instance representing the backend bucket. `bucketRequestName` and `BucketName` are mutually exclusive.
1. `bucketAccessName`: name of the bound cluster-scoped `BucketAccess` instance. Set by COSI.
1. `accessGranted`: if true access has been granted to the bucket. If false then access to the bucket has not been granted.

#### BucketAccess

A cluster-scoped, immutable, administrative custom resource which encapsulates fields from the [BucketAccessRequest](#bucketaccessrequest) (BAR) and the [BucketAccessClass](#bucketaccessclass) (BAC), and references a [Bucket](#bucket) instance.  The `BucketAccess` (BA) is an abstraction of a single access point to the backend bucket, and is instantiated when a new BAR is created.

There is a 1:1 mapping between a BA and a BAR. Many workload pods can point to the same BA (via their BAR reference). There is a many:1 mapping of BAs the same `Bucket` instance.

```yaml
apiVersion: objectstorage.k8s.io/v1alpha1
kind: BucketAccess
metadata: 
  name: [1]
  labels:
    objectstorage.k8s.io/provisioner: [2]
  finalizers: [3]
    - objectstorage.k8s.io/bar-protect
    - objectstorage.k8s.io/pod-<uuid1>
    - objectstorage.k8s.io/pod-<uuid2> ...
 spec:
   bucketName: [4]
   bucketAccessRequest: [5]
   serviceAccount: [6]
   accountID: [7]
   credentials: [8]
   policyActionsConfigMapData: [9]
   parameters: [10]
 status:
   accessGranted: [11]
   credentials: [12]
   accountID: [13]
```

1. `name`: generated in this format: _"ba-"+uuid_. The uuid is unique within a cluster.
1. `labels`: added COSI.  Key’s value should be the provisioner name. Characters that do not adhere to [Kubernetes label conventions](https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#syntax-and-character-set) will be converted to ‘-’.
1. `finalizers`: added by COSI to defer `BucketAccess` deletion until backend access is revoked. Implementation may add one finalizer for the BAR, and one for each accessing pod that references this BA. **Note**: a BA represents a potential access, whereas a pod is an actual access.
1. `bucketName`:  name of the `Bucket` instance bound to this BA.
1. `bucketAccessRequest`: an `objectReference` containing the name, namespace and UID of the associated `BucketAccessRequest`.
1. `serviceAccount`: (omitted if `credentials` is used) an `ObjectReference` containing the name, namespace and UID of the associated `BAR.serviceAccountName`. If empty then integrated Kubernetes cloud identity is not being used and `accountID` must contain the user identity. In this case, `status.accountID` is minted by the provisioner.
1. `accountID`: (optional) the username/access-key to be used by the object storage provider to identify the user with access to this bucket. Typically, `accountID` is omitted becauae the driver mints the user identity which is then copied to `status.accountID`. There may be certain brownfield use cases where `accountID` is filled when an admin manually creates a `BucketAccess` instance. For example, when there is an existing storage-side account id to be used for access, the admin sets `spec.accountID,` the driver will (potentially) enable access for that account id to the bucket in question, and return credentials.
1. `credentials`: (omitted if `serviceAccount` is used) a `SecretReference` containing the name, namespace and UID of a secret containing credentials that will be used by the driver to provision the bucket.
1. `policyActionsConfigMapData`: encoded data, known to the driver, and defined by the admin when creating a `BucketAccessClass`. Contains a set of provisioner/platform defined policy actions to a given user identity. Contents of the ConfigMap that the *policyActionsConfigMap* field in the `BucketAccessClass` refers to. A sample value of this field could look like:
```
   {“Effect”:“Allow”,“Action”:“s3:PutObject”,“Resource”:“arn:aws:s3:::profilepics/*“},
   {“Effect”:“Allow”,“Action”:“s3:GetObject”,“Resource”:“arn:aws:s3:::profilepics/*“},
   {“Effect”:“Deny”,“Action”:“s3:*“,”NotResource”:“arn:aws:s3:::profilepics/*“}
```
10. `parameters`:  A map of string:string key values.  Allows admins to control user and access provisioning by setting provisioner key-values. Copied from `BucketAccessClass`. 
11. `accessGranted`: if true access has been granted to the bucket. If false then access to the bucket has not been granted.
12. `credentials`: a `SecretReference` to the sidecar-generated Secret containing access credentials. This secret exists in the provisioner’s namespace, is read by the cosi-node-adapter, and written to the secret mount defined in the app pod's `csi-driver spec`.
13. `accountID`: username/access-key for the object storage provider that identifies the user with access to this bucket. This value is minted by the driver when the `BucketAccessRequest` omits a ServiceAccount.

#### BucketAccessClass

An immutable, cluster-scoped, custom resource providing a way for admins to specify policies used to access buckets. A `BucketAccessClass` (BAC) is referenced by a user [BucketAccessRequest](#bucketaccessrequest) and is used to populate the [BucketAccess](#bucketaccess) instance. A BAC references a config map which is expected to define the access policy for a given provider. It is typical for these config maps to reside in each provisioner's namespace, though this is not required. Unlike a [BucketClass](#bucketclass), there is no default BAC.

```yaml
apiVersion: objectstorage.k8s.io/v1alpha1
kind: BucketAccessClass
metadata: 
  name:
policyActionsConfigMap: [1]
  name:
  namespace:
parameters: [2]
```

1. `policyActionsConfigMap`: (required) a reference to a ConfigMap that contains a set of provisioner/platform-defined policy actions for bucket access. It is seen as typical that this config map's namespace is the same as for the provisioner. In othe words, a namespace that has locked down RBAC rules or prevent modification of this config map.
1. `parameters`: (optional)  A map of string:string key values.  Allows admins to control user and access provisioning by setting provisioner key-values. Optional reserved keys objectstorage.k8s.io/configMap and objectstorage.k8s.io/secrets are used to reference user created resources with provider specific access policies.
---

### App Pod
The application pod utilizes CSI's inline ephemeral volume support to provide the endpoint and secret credentials in an in-memory (ephemeral) volume. This approach also, importantly, prevents the pod from launching before the bucket has been provisioned since the kubelet waits to start the pod until it has received the cosi-node-adapter's _NodeStageVolume_ response.

Here is a sample pod manifest:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: my-app-pod
  namespace: dev-ns
spec:
  serviceAccountName: [1]
  containers:
    - name: my-app
      image: ...
      volumeMounts:
        - mountPath: /cosi [2]
          name: cosi-vol
  volumes:
    - name: cosi-vol
      csi: [3]
        driver: cosi.sigs.k8s.io [4]
        volumeAttributes: [5]
          bucketAccessRequestName: <my-bar-name>
```
1. the service account may be needed depending on cloud IAM integration with kubernetes.
1. the mount path is the directory where the app will read the credentials and endpoint.
1. this is an inline CSI volume.
1. the name of the cosi-node-adapter.
1. information needed by the cosi-node-adapter to verify that the bucket has been provisioned.

The contents of the secret generated by the provisioner is expected to have the following fields in the provider specific format:

- Endpoint for the bucket. The Endpoint should encode the URL for the provider
- Access Key: Only set when serviceAccount is empty 
- Secret Key: Only set when serviceAccount is empty

### Topology

![Architecture Diagram](images/arch.png)

# Object Relationships

The diagram below describes the relationships between various resources in the COSI ecosystem. 

![Object Relationships](images/object-rel.png)

# Workflows
Here we describe the workflows used to automate provisioning of new and existing buckets, and the de-provisioning of these buckets.

When the app pod is started, the kubelet will gRPC call _NodeStageVolume_ and _NodePublishVolume_, which are received by the cosi-node-adapter. The pod waits until the adapter responds to the gRPC requests. The adapter ensures that the target bucket has been provisioned and is ready to be accessed.

When the app pod terminates, the kubelet gRPC calls _NodeUnstageVolume_ and _NodeUnpublishVolume_, which the cosi-node-adapter also receives. The adapter orchestrates tear-down of the ephemeral volume provisioned for this bucket.

General prep:
+ admin creates the needed bucket classes and bucket access classes.

Prep for brownfield:
+ admin creates `Bucket` instances representing backend buckets.

### Finalizers
The following finalizers are added by COSI to orchestrate the deletion of the various COSI resources. Of note is that a finalizer is added to the two user-created instances (BR and BAR) so that these instances remain available until the deletion of the associated resources (Bucket and BA) and the backend bucket have completed.

+ _objectstorage.k8s.io/{bucketrequest|bucketaccessrequest}-protect_, added to BRs and BARs.
+ _objectstorage.k8s.io/br-protect_, added to a `Bucket` indicating a BR created it.
+ _objectstorage.k8s.io/ba-\<name1> ba-\<name2>..._, added to a `Bucket` for each BA referencing the `Bucket.`
+ _objectstorage.k8s.io/bar-protect_, added to `BucketAccess` indicating a BAR created it.
+ _objectstorage.k8s.io/pod-\<uuid1> pod-\<uuid2>..._, added to `BucketAccess` for each pod accessing the BA.


## Create Bucket

![CreateBucket Workflow](images/create-bucket.png)

This workflow describes creating a new (greenfield) backend bucket. Accessing this bucket is covered in [Sharing COSI Created Buckets](#sharing-cosi-created-buckets).

Here is the workflow:

+ The user creates a `BucketRequest` (BR) which is seen by COSI.
+ If BR.bucketClassName is empty COSI finds the default BC.
+ COSI verifies that the BR's namespace is allowed based on the BC.
+ COSI adds a BR finalizer.
+ COSI creates the associated `Bucket` instance, filling in fields from the BC and BR.
+ COSI adds a `Bucket` finalizer.
+ The sidecar sees the new `Bucket` and calls the driver to create a backend bucket.
+ The sidecar updates the `Bucket` instance with the driver returned bucketID.
+ COSI two-way binds the BR and `Bucket` instance.
+ COSI sets the `Bucket` and BR statuses to "BucketAvailable".

## Sharing COSI Created Buckets

This is the greenfield -> brownfield use case, where COSI has created the `Bucket` instance and the driver has provisioned a new bucket. Now, we want to share access to this bucket within the BR's namespace or across other namespaces.

![ShareBucket Workflow](images/share-bucket.png)

Here is the workflow:

+ The user creates a `BucketAccessRequest` (BAR) that specifies either the name of the BR (when sharing in the same namespace) or the `Bucket` instance name (when sharing across namespaces).
+ COSI sees the BAR add event.
   + Note: the `Bucket` instance name is referenced either as BAR.BR.bucketName or BAR.bucketName.
+ COSI checks that the `Bucket` instance has a nil deletionTimestamp.
+ COSI adds the BAR finalizer.
+ COSI creates a BA with a one-way binding to the `Bucket` instance.
+ COSI adds a BA finalizer.
+ The sidecar sees the new BA and calls the driver to grant access to the backend bucket.
   + Note: the backend bucket is found via BA.bucketName.bucketID.
+ COSI updates the BA and BAR statuses to "accessGranted".
+ COSI adds a `Bucket` finalizer, "objectstorage.k8s.io/ba-\<name>".

In addition, a pod references a BAR in the `volumes.csi.driver` spec:
+ The cosi-node-adapter receives a _NodeStageVolume_ request from the kubelet.
+ The adapter adds a "objectstorage.k8s.io/pod-\<name>" finalizer to the associated `BucketAccess`.

## Delete Bucket

![DeleteBucket Workflow](images/delete-bucket.pni)

This workflow describes optionally deleting a `Bucket` instance and the backend bucket. Revoking access to this bucket is covered in [Revoke Bucket Access](#revoke-bucket-access). There are three deletion policies: "Retain", "Delete", and "ForceDelete", which are defined in the `BucketClass` and the `Bucket` instance. Only the bucket instance's _deletionPolicy_ is used to determine the deletion policy used to handle bucket deletes.

It is up to each provisioner whether or not to physically delete the backend bucket content. But, for idempotency, the expectation is that the backend bucket will at least be made unavailable.

> Note: the delete workflow is described as synchronous, but it will likely run asynchronously to accommodate potentially long delete times when buckets contain many objects. The steps should still mostly follow what's outlined below.

Here is the delete workflow:

+ User deletes their BR which is deferred until its finalizer has been removed.
+ COSI gets the `Bucket` instance from the BR and notes the deletion policy.
+ If the deletion policy is "_Retain_":
   + COSI updates the `Bucket` status to "unavailable" or similar.
   + COSI clears the BR reference in the `Bucket`.
   + COSI removes all `Bucket` finalizers.
      + Note: the `Bucket` instance and backend bucket are not deleted.
      + Note: there may be pods and BAs referencing this `Bucket`.
   + Note: at this point no new BARs are able to request access to this bucket.

+ If the deletion policy is "_Delete_":
   + Note: the goal is to not delete the backend bucket or the `Bucket` instance until all access has been cleaned up.
   + COSI deletes the `Bucket` instance which is deferred due to finalizers.
      + Note: new BARs cannot access the `Bucket` now.
   + The sidecar sees the deletionTimestamp set in the `Bucket` and checks to see if there are any BA.name finalizers in the `Bucket`.
      + If none, it calls the driver to delete the backend bucket.
         + COSI removes the "objectstorage.k8s.io/br-protect" finalizer from the `Bucket`, which allows it to be garbage collected.
      + If >0, COSI does not call the driver and does no further processing.

+ If the deletion policy is "_ForceDelete"_:
   + COSI deletes the `Bucket` instance which is deferred due to finalizers.
      + Note: new BARs cannot access the `Bucket` now.
   + COSI gets all BAs referencing the `Bucket`.
      + For each BA, COSI removes all finalizers and deletes the BA.
      + Note: COSI does not delete any BARs that reference the deleted BAs, so a BAR may refer to a BA that no longer exists.
   + The sidecar sees the `Bucket` deletionTimestamp set and the "ForceDelete" deletion policy, and calls the driver to delete the backend bucket.
   + COSI deletes all `Bucket` finalizers and the `Bucket` is garbage collected.
      + Note: pods may still try to access the backend bucket but will likely fail.

+ COSI removes the BR's "objectstorage.k8s.io/bucketrequest-protect" finalizer and the BR is garbage collected.

When the pod teminates:
+ The cosi-node-adapter receives the _NodeUnstageVolume_ request from the kubelet.
+ The adapter gets the BAR.BA and BAR.BA.Bucket.
+ The adapter deletes the "pod-\<uuid>" finalizer from the `Bucket`.

## Grant Bucket Access
This workflow describes granting access to an existing backend bucket and is separate from the greenfield sharing workflow described above.

Here is the workflow:

+ The user creates a BAR that specifies the name of the admin created `Bucket` instance.
+ COSI sees the BAR and adds a finalizer.
+ COSI checks that the `Bucket` instance has a nil deletionTimestamp.
+ COSI creates a BA, one-way binding it to the `Bucket` instance.
+ COSI adds a BA finalizer.
+ The sidecar sees the new BA and calls the driver to grant access to the backend bucket, passing BA.bucketName.bucketID.
+ COSI adds a `Bucket` finalizer, "objectstorage.k8s.io/ba-\<name>".
+ Once a pod is created, the cosi-node-adapter adds the "objectstorage.k8s.io/pod-\<uuid>" finalizer to the BA.

In addition, a pod references a BAR in the `volumes.csi.driver` spec:
+ cosi-node-adapter receives a _NodeStageVolume_ request from kubelet.
+ adapter adds the "objectstorage.k8s.io/pod-\<uuid>" finalizer to the BA.

## Revoke Bucket Access
This workflow describes revoking access to an existing backend bucket and the deletion of the cluster-scoped `BucketAccess` instance.

Here is the workflow:

+ User deletes their BAR which is deferred until finalizer has been removed.
+ COSI gets the BA from the BAR.
+ COSI deletes the BA, but finalizers prevent it from being garbage collected.
+ The sidecar sees the BA deleteTimestamp and calls the driver to revoke access to the backend bucket.
   + Note: this may cause problems with pods accessing this bucket.
+ COSI removes the BA's "objectstorage.k8s.io/bar-protect" finalizer and it may be garbage collected.
+ COSI removes the BAR's finalizer and it is garbage collected.

In addition, the cosi-node-adapter sees the app pod is terminating:
+ The adapter receives a _NodeUnstageVolume_ request from the kubelet.
+ The adapter checks for pods referencing the BA.
   + If none, then it removes the "objectstorage.k8s.io/pod-\<uuid>" finalizer from the BA, which may cause it to be garbage collected.
   + Note: when the finalizer of the last BA referencing a `Bucket` is removed (and thus the last BA is deleted), the `Bucket` may be garbage collected.

## Delete BucketAccess 
The above workflows are triggered by the user. Now we cover worflows caused by the admin, which may be necessary to handle situations where COSI doesn't clean up properly or where, for example, a credential needs to be immediately revoked.

The most common scenario is likely the case where a token is compromised and the admin needs to stop its use. In this case the admin may terminate the app pod(s) and delete the `BucketAccess` instances.

Here is the workflow:

+ Admin deletes a BA which is deferred due to finalizer(s).
+ The sidecar detects the BA delete and calls the driver to revoke access to the backend bucket.
+ Admin may also delete the app pod.
   + The cosi-node-adapter receives the _NodeUnstageVolume_ request from the kubelet.
   + The adapter removes the "objectstorage.k8s.io/pod-\<uuid>" finalizer from the BA.
+ COSI unbinds the BA and BAR and sets BAR.status to "broken", or similar.
+ The user BAR remains pointing to a BA that no longer exists.
   + Note: COSI should not delete user-created instances.

## Delete Bucket
There may be times when an admin needs to delete a `Bucket` instance, whether if it was created by COSI or manually created and is being accessed.

Here is the workflow:

+ Note: admin should mutate the `Bucket` deletion policy to "ForceDelete".
+ Admin deletes a `Bucket` instance which is deferred due to finalizer(s).
+ The sidecar sees the deletionTimestamp and the "ForceDelete" deletion policy, and calls the driver to delete the backend bucket.
+ The rest of the workflow is defined above under "ForceDelete".

##  Setting Access Permissions
### Dynamic Provisioning
Incoming `BucketAccessRequest`s contains a *serviceAccountName* where a cloud provider supports identity integration. The incoming `BucketAccessRequest` represents a user to access the `Bucket` and a corresponding `BucketAccess` will provide the access credentials to the workloads using *serviceAccount* or *mintedSecret* .
When requesting access to a bucket, workloads will go through the scenarios described here:
+  New User: In this scenario, we do not have user account in the backend storage system as well as no access for this user to the `Bucket`. 
	+ Create user account in the backend storage system.
	+ add the access privileges for the user to the `Bucket`.
	+ return the credentials to the workload owning the `BucketAccessRequest`.
+  Existing User and New Bucket: In this scenario, we have the user account created in the backend storage system, but the account is not associated to the `Bucket`.
	+ add the access privileges to the `Bucket`.
	+ return the credentials to the workload owning the `BucketAccessRequest`.
+  Existing User and existing Bucket: In this scenario, the user account has access policy defined on the `Bucket`.  The existing user privileges in the backend may conflict with the privileges that the user is requesting.
	+ FAIL, if existing access policy is different from the requested policy.
	+ if the existing privileges match the requested privileges, return the credentials to the workload owning the `BucketAccessRequest`.
+ A Service Account specified and the cloud platform identity integration maps Kubernetes ServiceAccounts to the account in the backend storage system. No need to create credentials here.
	
Upon success, the `BucketAccess` instance is ready and the app workload can access backend storage.

### Static Provisioning
"Driverless" allows the existing workloads to make use of COSI without the need for vendors to create drivers. The following steps show the details of the workflow:
+ Admin creates `Bucket` instance that references an existing backend bucket.
+ User creates a `BucketRequest` (BR) that references this `Bucket`.
+ User creates `BucketAccessRequest` (BAR) that references their BR, a `BucketAccessClass`, and their secret containing access tokens.
+ COSI detects the existence of the BAR and follows the [grant access](#grant-bucket-access) steps.
+ COSI detects the existence of the BAR, BA and executes a validation process.
+ As a validation step, COSI ensures that the `BucketRequest.bucketName` matches the `BucketAccessRequest.bucketAccessName.bucketName`. In other words, the two related `Bucket` references match, meaning that the BR and BAR->BA point to the same `Bucket`.

---

# gRPC Definitions

There is one service defined by COSI - `Provisioner`. This will need to be satisfied by the vendor-provisioner in order to be COSI-compatible

```
service Provisioner {
    rpc ProvisionerGetInfo (ProvisionerGetInfoRequest) returns (ProvisionerGetInfoResponse) {}

    rpc ProvisionerCreateBucket (ProvisionerCreateBucketRequest) returns (ProvisionerCreateBucketResponse) {}
    rpc ProvisionerDeleteBucket (ProvisionerDeleteBucketRequest) returns (ProvisionerDeleteBucketResponse) {}

    rpc ProvisionerGrantBucketAccess (ProvisionerGrantBucketAccessRequest) returns (ProvisionerGrantBucketAccessResponse);
    rpc ProvisionerRevokeBucketAccess (ProvisionerRevokeBucketAccessRequest) returns (ProvisionerRevokeBucketAccessResponse);
}
```

## ProvisionerGetInfo

This call is meant to retrieve the unique provisioner Identity. This identity will have to be set in `BucketRequest.Provisioner` field in order to invoke this specific provisioner.

```
message Protocol {
	oneof type {
		S3Parameters s3 = 1;
		AzureBlobParameters azureBlob = 2;
		GCSParameters gcs = 3;
	}
}

message ProvisionerGetInfoRequest {
    // Intentionally left blank
}

message ProvisionerGetInfoResponse {
  // The name MUST follow domain name notation format
  // (https://tools.ietf.org/html/rfc1035#section-2.3.1). It SHOULD
  // include the plugin's host company name and the plugin name,
  // to minimize the possibility of collisions. It MUST be 63
  // characters or less, beginning and ending with an alphanumeric
  // character ([a-z0-9A-Z]) with dashes (-), dots (.), and
  // alphanumerics between. This field is REQUIRED.
  string name = 1;
}
```

## ProvisonerCreateBucket

This call is made to create the bucket in the backend. If a bucket that matches both name and parameters already exists, then OK (success) must be returned. If a bucket by same name, but different parameters is provided, then the appropriate error code `ALREADY_EXISTS` must be returned by the provisioner. The call to *ProvisonerCreateBucket* MUST be idempotent.

```
message ProvisionerCreateBucketRequest {    
    // This field is REQUIRED
    // name specifies the name of the bucket that should be created.
    string name = 1;
    // This field is REQUIRED
    // Protocol specific information required by the call is passed in as key,value pairs.
    Protocol protocol = 2;
    // This field is OPTIONAL
    // The caller should treat the values in parameters as opaque. 
    // The receiver is responsible for parsing and validating the values.
    map<string,string> parameters = 3;
}

message ProvisionerCreateBucketResponse {
    // bucket_id returned here is expected to be the globally unique 
    // identifier for the bucket in the object storage provider 
    string bucket_id = 1;
}
```

## ProvisonerDeleteBucket

This call is made to delete the bucket in the backend. If the bucket has already been deleted, then no error should be returned. The call to *ProvisonerDeleteBucket* MUST be idempotent.

```
message ProvisionerDeleteBucketRequest {
    // This field is REQUIRED
    // bucket_id is a globally unique identifier for the bucket
    // in the object storage provider 
    string bucket_id = 1;
}

message ProvisionerDeleteBucketResponse {
     // Intentionally left blank
}
```

## ProvisionerGrantBucketAccess

This call grants access to a particular account id. The _account_id_ is the account for which this access should be granted. 

If the account id is set, then it should be used as the username of the created credentials or in some way should be deterministically used to generate a new credential for this request. This account id will be used as the unique identifier for deleting this access by calling ProvisionerRevokeBucketAccess

If the `account_id` is empty, then a new service account should be created in the backend that satisfies the requested `access_policy`. The username/account_id for this service account should be set in the `account_id` field of `ProvisionerGrantBucketAccessResponse`.

```
message ProvisionerGrantBucketAccessRequest {
    // This field is REQUIRED
    // bucket_id is a globally unique identifier for the bucket
    // in the object storage provider 
    string bucket_id = 1;
    // This field is REQUIRED
    // account_name is a identifier for object storage provider 
    // to ensure that multiple requests for the same account
    // result in only one access token being created
    string account_name = 2;
    // This field is REQUIRED
    // Requested Access policy, ex: {"Effect":"Allow","Action":"s3:PutObject","Resource":"arn:aws:s3:::profilepics/*"}
    string access_policy = 3;
    // This field is OPTIONAL
    // The caller should treat the values in parameters as opaque. 
    // The receiver is responsible for parsing and validating the values.
    map<string,string> parameters = 4;
}

message ProvisionerGrantBucketAccessResponse {
    // This field is OPTIONAL
    // This is the account_id that is being provided access. This will
    // be required later to revoke access. 
    string account_id = 1;
    // This field is OPTIONAL
    // Credentials supplied for accessing the bucket ex: aws access key id and secret, etc.
    string credentials = 2;
} 
```

## ProvisionerRevokeBucketAccess

This call revokes all access to a particular bucket from a account.  

```
message ProvisionerRevokeBucketAccessRequest {
    // This field is REQUIRED
    // bucket_id is a globally unique identifier for the bucket
    // in the object storage provider.
    string bucket_id = 1;

    // This field is REQUIRED
    // This is the account_id that is having its access revoked.
    string account_id = 2;
}

message ProvisionerRevokeBucketAccessResponse {
    // Intentionally left blank
}
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

## Alpha -> Beta
- Basic unit and e2e tests as outlined in the test plan.
- Metrics in kubernetes/kubernetes for bucket create and delete, and granting and revoking bucket access.
- Metrics in provisioner for bucket create and delete, and granting and revoking bucket access.

## Beta -> GA
- Stress tests to iron out possible race conditions in the controllers.
- Users deployed in production and have gone through at least one K8s upgrade.

# Alternatives Considered
This KEP has had a long journey and many revisions. Here we capture the main alternatives and the reasons why we decided on a different solution.

## Add Bucket Instance Name to BucketAccessClass (brownfield)

### Motivation
1. To improve workload _portability_ user namespace resources should not reference non-deterministic generated names. If a `BucketAccessRequest` (BAR) references a `Bucket` instance's name, and that name is pseudo random (eg. a UID added to the name) then the BAR, and hence the workload deployment, is not portable to another cluser.

1. If the `Bucket` instance name is in the BAC instead of the BAR then the user is not burdened with knowledge of `Bucket` names, and there is some centralized admin control over brownfield bucket access.

### Problems
1. The greenfield -> brownfield workflow is very awkward with this approach. The user creates a `BucketRequest` (BR) to provision a new bucket which they then want to access. The user creates a BAR pointing to a BAC which must contain the name of this newly created ``Bucket` instance. Since the `Bucket`'s name is non-deterministic the admin cannot create the BAC in advance. Instead, the user must ask the admin to find the new `Bucket` instance and add its name to new (or maybe existing) BAC.

1. App portability is still a concern but we believe that deterministic, unique `Bucket` and `BucketAccess` names can be generated and referenced in BRs and BARs.

1. Since, presumably, all or most BACs will be known to users, there is no real "control" offered to the admin with this approach. Instead, adding _allowedNamespaces_ or similar to the BAC may help with this.

### Upgrade / Downgrade Strategy

No changes are required on upgrade to maintain previous behaviour.

### Version Skew Strategy

COSI is out-of-tree, so version skew strategy is N/A

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
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

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

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
