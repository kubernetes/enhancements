---
title: Object Bucket Provisioning
authors:
  - "@jeffvance"
  - "@copejon"
owning-sig: sig-storage
participating-sigs:
  - sig-storage
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
last-updated: 2019-11-15
status: implementable
see-also:
  - "https://github.com/kube-object-storage/lib-bucket-provisioner"
  - "https://github.com/kube-object-storage/lib-bucket-provisioner/blob/master/doc/design/object-bucket-lib.md"
  - "https://github.com/kube-object-storage/lib-bucket-provisioner/tree/master/doc/examples"
---

# Object Bucket Provisioning


## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Vocabulary](#vocabulary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [CSI](#csi)
  - [Pros](#pros)
  - [Cons](#cons)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Custom Resources](#custom-resources)
  - [Moving Parts](#moving-parts)
  - [Interfaces](#interfaces)
  - [Library Usage](#library-usage)
  - [Current Restrictions](#current-restrictions)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Examples](#examples)
      - [Alpha -&gt; Beta Graduation](#alpha---beta)
      - [Beta -&gt; GA Graduation](#beta---ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
<!-- /toc -->

## Summary

Object storage is different from file and block storage in that there are no mounts or attatches, but most importantly, there is no standard such as POSIX or iSCSI.
And although AWS S3 could be considered a de-facto standard, it is, in fact, proprietary, closed and subject to any changes the vendor wishes to make.
These issues are the main reasons why Kubernetes today lacks object store related APIs.

However, despite the lack of a formal standard for an object data path, we can still define an API for bucket management.
We propose a common Kubernetes control-plane API for the management of object store bucket lifecycles which has no dependencies on the underlying providers.
This proposal does not address all aspects of object storage, e.g. creating an object store, creating bucket users, etc.
It focuses on provisioning new buckets, granting access to existing buckets, and deleting buckets created by provisioners.

## Vocabulary

+ _bucket_ - a container where objects reside, similar to a POSIX directory.
+ _bucket owner_ - an object store user with permissions to create a new bucket. The credentials for this user are not the credentials stored in the secret.
+ _bucket user_ - the user created or used by the provisioner who is granted access to the bucket. This user's credentials are store in the secret.
+ _endpoint_ - the URL describing the bucket.
+ _library_ - the bucket provisioning library proposed in this document and imported by provisiuoners.
+ _object_ - in this proposal object usually means content residing in a container, excluding metadata, similar in concept to a file.
+ _provisioner_ - code running in a pod that implements the library's `Provision`, `Grant`, `Delete` and `Revoke` interfaces.

## Motivation

By defining a Kubernetes API for object bucket provsioning, we provide a consistent framework for all object store providers and thus enhance the end user's storage experience. This API allows object store vendors to focus on how they wish to provision buckets instead of writing and testing the controller aspects of provisioning.

### Goals

+ Define a _control-plane_ object bucket management API thus relieving object store providers from Kubernetes controller details.
+ Minimize technical ramp-up for storage vendors.
+ Make bucket provisioning similar to file or block provisioning using familiar concepts and commands.
+ ~~Use native Kubernetes resources where possible, again, to keep the bucket experience for users and admins similar to existing storage provisioning.~~
This was an original goal and the OBC is define to reference a storage class like PVCs do.
However, there could be confusion using storage classes for bucket provisioning since storage classes are designed only for file and block storage, and thus many fields have no meaning for buckets.
We are open to defining a namespaced `BucketClass` Custom Resource to be used similarly to storage classes.
+ Be unopinionated about the underlying object-store and at the same time provide a flexible API such that provisioner specific features can be supported.
+ Ensure bucket consuming pods wait until the target buckets have been created and are accessible. Thus there is no specific order required for when a bucket claim is created vs. when the app pod is run.
+ Present similar user and admin experiences for both _greenfield_ (new) and _brownfield_ (existing) bucket provisioning.

### Non-Goals

+ Update the native Kubernetes PVC-PV API to support object buckets.
+ Define a native _data-plane_ object store API.
+ Handle the small percentage of apps that will not be portable due to use of non-compatible object-store features.

## CSI

There has been some excellent discussion around CSI vs. a library approach.
We'd like this section to collect comments on the pros and cons of using CSI.

### Pros
+ Kubernetes strategic direction for all storage.
This may be the only argument needed to discard the library and instead support CSI.
+ Familiar to most storage vendors.
+ gRPC supports many languages.

### Cons
+ Being independent of Kubernetes, perhaps there is more up-front ramp up?

## Proposal

Create a bucket provisioning library, similar to the Kubernetes [sig-storage-lib-external-provisioner](https://github.com/kubernetes-sigs/sig-storage-lib-external-provisioner/blob/master/controller/controller.go) library, which handles all of the controller and Kubernetes resources related to provisioning.
This library will implement a simple and familiar API via Custom Resources, and define a _contract_ between app developers and provisioners regarding the Kubernetes resources related to bucket provisioning.

The actual creation of physical buckets and the generation of appropriate credentials belong to each object store provisioner, while the library handles watches on bucket claims, reconciles desired state, and creates and deletes k8s resources.

Since certain object store-specific artifacts, e.g. user name, are needed in order to clean up after a bucket is deleted or revoked, this proposal must allow for some provisioner values to be stored across API calls.

## Design Details

### Custom Resources

(see the [full specs](#custom-resource-details))

The bucket provisioning library utilizes two Custom Resources to abstract an object store bucket and a claim/request for such a bucket.
+ a namespaced _**ObjectBucketClaim**_ (OBC), similar to a PVC, is a formal request for a new or an existing bucket.
+ an _**ObjectBucket**_ (OB), similar to a PV, is the k8s representation of the provisioned bucket and may contain object store-specific data needed to allow the bucket to be de-provisioned. The OB resides in the same namespace as the object store, and is typically not seen by bucket consumers.
**Note:** namespaced OBs is a change from the original design and was suggested at the 2019 NA Kubecon storage face-to-face with the premise being that if there is no _technical_ reason for a cluster scoped resource then it should be namespaced.

As is true for PVCs-PVs, there is a 1:1 relationship between an OBC and an OB, and as will be seen below, there is a [_binding_](#binding) between an OBC and OB. 

OBCs reference a storage class. The storage class references the external provisioner, defines a reclaim policy, and specifies object-store specific parameters, such as region, owner secret, bucket lifecycle policies, etc.
**Note:** the original design uses storage classes but we are open to using a namespaced _BucketClass_ Custom Resource based on community input.

### Moving Parts

Before deploying an object bucket consuming app, the OB and OBC CRDs need to be created with RBAC rules granting _get_, _list_, _watch_, and _delete_ verbs.
**Note:** OBCs do not need delete permission but OBs, secrets, and configmaps do.
An object store needs to be created and referenced in the storage classes used by OBCs.
The OBCs needed by the app must exist in the app's namespace.

As is true for dynamic PV provisioning, a bucket provisioner pod needs to be running for each type of object-store existing in the Kubernetes cluster.
For example, for AWS S3, the developer creates an OBC referencing a storage class which references the S3 store.
This storage class needs to be created by the admin.
The S3 provisioner pod watches for OBCs whose storage classes point to the AWS S3 provisioner, while ignoring all other OBCs.
Likewise, the same cluster can also run the Rook-Ceph RGW provisioner, which also watches OBCs, only handling OBCs that reference storage classes which define ceph-rgw.

#### Greenfield (new)

An OBC for a new bucket is defined by its inclusion of a generated or static bucket name.
In this case the referenced storage class omits a bucket name.
A new bucket OBC triggers the correct provisioner to create a new bucket and the associated artifacts (credentials, policy, etc). 
The library, in turn, creates a secret, configmap, and OB based on information returned by the provisioner.

The secret contains the credentials needed to access the bucket and the configmap contains bucket endpoint information.
**Note:** it was mentioned at the 2019 NA Kubecon storage face-to-face that to improve scaling we could decide to move the configmap info into the secret, thus reducing k8s resources required.
The OB is an abstraction of the bucket and can save some state data needed by provisioners.
The secret and configmap reside in the same namespace as the OBC, and the OB lives in the provisioner's namespace.
The app pod consumes the secret and configmap and thus can reference the bucket.

When a _greenfield_ OBC is deleted the associated provisioner is expected to delete the newly provisioned bucket and the related object store-specific artifacts.
The library will delete the secret, configmap and OB.

#### Brownfield (existing)

An OBC for an existing bucket is defined by the omission of a bucket name.
Instead the associated storage class contains the bucket name.
An existing bucket OBC triggers the correct provisioner to grant access to the existing bucket.
The library then creates the secret, configmap, and OB based on information returned by the provisioner.
Like new bucket provisioning, associated bucket related artifacts (credentials, policy, etc) are created and managed by the provisioner.

A deleted _brownfield_ OBC is the same as for _greenfield_ except that provisioners typically will not delete the physical bucket. Instead, bucket access is revoked and the related artifacts are cleaned up.
Again, the library will delete the secret, configmap and OB.

### Interfaces

There are four interface methods that must be defined by all bucket provisioners.
Additionally there are two required and one optional library function used. 

#### Library Functions

- **`NewProvisioner`** is a required function called by provisioners to create the library's controller struct which is returned to the provisioner.
Each provisioner defines their own struct, passed to `NewProvision`, which implements the Interfaces below.
The returned controller struct supports the `Run` and `SetLabels` methods.

- **`Run`** is a required controller method called by provisioners to start the OBC controller.

- **`SetLabels`** is an optional controller method called by provisioners which supports setting custom labels on the library resources.
**Note:** the library adds its own label to the OBC, OB, secret, and configmap. This label value is the provisioner name.

#### Bucket Interfaces

The following interfaces must be implemented on the provisioner-defined structure which is passed to `NewProvisioner`:

- **`Provision`** is a method called by the library when a new OBC is detected and its storage class does not contain the bucket name, meaning "greenfield" provisioning.
In this case the OBC contains a bucket name or a name is generated.
Provisioners are expected to create a new bucket and related artifacts such as user, policies, credentials, etc.
Provisioners return a skeleton OB structure.

- **`Grant`** is a method called by the library when a new OBC is detected and its storage class contains the bucket name, meaning "brownfield" provisioning.
In this case the OBC does not contain the bucket name.
Provisioners are expected to create artifacts such as user, policies, credentials, etc., but not to create a new bucket.
Provisioners return a skeleton OB structure.

- **`Delete`** is a method called by the library when an OBC is deleted, and its storage class does not contain the bucket name (meaning "greenfield" provisioning had occurred), and the storage class's `reclaimPolicy` is "Delete".
Provisioners are expected to remove the bucket and related artifacts.

- **`Revoke`** is a method called by the library when an OBC is deleted and one of the following situations exists.
  - the OBC's storage class contains the bucket name, meaning "brownfield" provisioning had occurred.
In this case the storage class's `reclaimPolicy` is ignored.
  - "greenfield" provisioning occurred and the storage class's `reclaimPolicy` is "Retain".
In both cases provisioners are expected to retain the bucket but delete the object-store related artifacts.
  
### Binding

Bucket binding refers to the steps necessary to make the target bucket available to a pod.
For _greenfield_ a new bucket is physically created.
In both new and existing bucket use cases Kubernetes resources are created by the library: a secret containing access credentials, a configMap containing bucket endpoint info, and an OB describing the bucket.
Each provisioner will likely have to create and delete their own object store-specific resources.

Bucket binding requires these steps before the bucket is accessible to an app pod:
1. (greenfield only) generation of a random bucket name when requested (performed by bucket lib).
1. (greenfield only) the creation of the physical bucket with owner credentials (performed by provisioner).
1. creation of the necessary object store-specific resources, e.g. IAM user, policy, etc.
1. creation of an OB, residing in the object-store's namespace, based on the provisioner's returned OB structure (performed by bucket lib).
1. creation of a ConfigMap, residing in the OBC's namespace, based on the provisioner's returned endpoint info (performed by bucket lib).
1. creation of a Secret, residing in the OBC's namespace, based on the provisioner's returned credentials (performed by bucket lib).

### Custom Resource Details

#### OBC Details (created by user)

```yaml
apiVersion: objectbucket.io/v1alpha1
kind: ObjectBucketClaim
metadata:
  name: MY-BUCKET-1 [1]
  namespace: USER-NAMESPACE [2]
spec:
  bucketName: [3]
  generateBucketName: "photo-booth" [4]
  storageClassName: AN-OBJECT-STORE-STORAGE-CLASS [5]
  additionalConfig: [6]
    ANY_KEY: VALUE ...
```
1. name of the ObjectBucketClaim. This name becomes the name of the Secret and ConfigMap.
1. namespace of the ObjectBucketClaim, which is also the namespace of the ConfigMap and Secret.
1. name of the bucket. If supplied then `generateBucketName` is ignored.
**Not** recommended for new buckets since names must be unique within
an entire object store.
1. if supplied then `bucketName` must be empty. This value becomes the prefix for a randomly generated name and is the preferred way to create a new bucket.
After `Provision` returns `bucketName` is set to this random name.
If both `bucketName` and `generateBucketName` are supplied then `BucketName` has precedence and `GenerateBucketName` is ignored. 
If both `bucketName` and `generateBucketName` are blank or omitted then the storage class is expected to contain the name of an _existing_ bucket. It's an error if all three bucket related names are blank or omitted.
1. storageClass which defines the object-store service and the bucket provisioner.
**Note:** we are happy to define a namespaced _BucketClass_ Custom Resource based on community input.
1. additionalConfig gives providers a location to set store-specific config values (tenant, namespace...).
The value is a list of 1 or more key-value pairs.

**Note:** the OBC is mutated by the library with the addition of a finalizer and label:
```
  labels:
    bucket-provisioner: PROVISIONER-NAME [1]
  finalizers:
  - "objectbucket.io/finalizer" [2]
```
1. the label value is the name of the provisioner but due to Kubernetes restrictions slash (/) is replaced by a dash (-).
1. finalizers set and cleared by the lib's OBC controller. Prevents accidental deletion of an OB.

#### OB Details (created by library)

```yaml
apiVersion: objectbucket.io/v1alpha1
kind: ObjectBucket
Metadata:
  name: obc-my-ns-my-bucket [1]
  namespace: namespace-of-object-store [2]
  labels:
    bucket-provisioner: PROVISIONER-NAME [3]
  finalizers:
  - "objectbucket.io/finalizer" [4]
spec:
  storageClassName: example-obj-prov [5]
  claimRef: *v1.objectreference [6]
  reclaimPolicy: {"Delete", "Retain"} [7]
  endpoint:
    bucketHost: foo.bar.com
    bucketPort: 8080
    bucketName: my-photos-1xj4a
    region: # provisioner dependent
    subRegion: # provisioner dependent
    additionalConfigData: [] #string:string
  additionalState: [] #string:string
status:
  phase: {"Bound", "Released", "Failed"} [8]
```
1. name is constructed in the pattern: obc-OBC_NAMESPACE-OBC_NAME
1. namespace of the object store service as defined in the storage class
1. the label value shown is the name of the provisioner but due to Kubernetes restrictions slash (/) is replaced by a dash (-).
1. finalizers set and cleared by the lib's OBC controller. Prevents accidental deletion of an OB.
   replaced by a dash (-). In this example the provisioner name is `aws-s3.io/bucket`.
1. name of the storage class, referenced by the OBC, containing the provisioner and object store service name.
1. objectReference to the associated OBC.
1. reclaim policy from the Storge Class referenced in the OBC.
1. phase is the current state of the ObjectBucket:
    - _Bound_: the operator finished processing the request and linked the OBC and OB
    - _Released_: the OBC has been deleted, leaving the OB unclaimed.
    - _Failed_: not currently set.

#### Secret (created by library)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: MY-BUCKET-1 [1]
  namespace: OBC-NAMESPACE [2]
  finalizers: [3]
  - objectbucket.io/finalizer
  labels: [4]
    bucket-provisioner: aws-s3.io-bucket [5]
  ownerReferences:
  - name: MY-BUCKET-1 [6]
    ...
type: Opaque
data:
  ACCESS_KEY_ID: BASE64_ENCODED-1
  SECRET_ACCESS_KEY: BASE64_ENCODED-2
  ... [5]
```
1. same name as the OBC. Unique since the secret is in the same namespace as the OBC.
1. namespce of the originating OBC.
1. finalizers set and cleared by the lib's OBC controller. Prevents accidental deletion of the Secret.
1. the library adds a label (seen here) but each provisioner can supply their own labels.
1. the label value shown is the name of the provisioner but due to Kubernetes restrictions slash (/) is replaced by a dash (-).
1. ownerReference makes this secret a child of the originating OBC for clean up purposes.
1. ACCESS_KEY_ID and SECRET_ACCESS_KEY are the only secret keys defined by the library.
Provisioners are able to cause the lib to create additional keys by returning  the `AdditionalSecretConfig` field.
**Note:** the library will create the Secret using `stringData:` and let the Secret API base64 encode the values.

#### ConfigMap (created by library)

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: MY-BUCKET-1 [1]
  namespace: OBC-NAMESPACE [2]
  finalizers: [3]
  - objectbucket.io/finalizer
  labels: [4]
    bucket-provisioner: aws-s3.io-bucket [5]
  ownerReferences: [6]
  - name: MY-BUCKET-1
    ...
data: 
  BUCKET_HOST: http://MY-STORE-URL [7]
  BUCKET_PORT: 80 [8]
  BUCKET_NAME: MY-BUCKET-1 [9]
  BUCKET_REGION: us-west-1
  ... [10]
```
1. same name as the OBC. Unique since the configMap is in the same namespace as the OBC.
1. determined by the namespace of the ObjectBucketClaim.
1. finalizers set and cleared by the lib's OBC controller. Prevents accidental deletion of the ConfigMap.
1. the library adds a label (seen here) but each provisioner can supply their own labels.
1. the label value shown is the name of the provisioner but due to Kubernetes restrictions slash (/) is
   replaced by a dash (-). In this example the provisioner name is `aws-s3.io/bucket`.
1. ownerReference sets the ConfigMap as a child of the ObjectBucketClaim. Deletion of the ObjectBucketClaim causes the deletion of the ConfigMap.
1. host URL.
1. host port.
1. unique bucket name.
1. the above data keys are defined by the library.
Provisioners are able to cause the lib to create additional data keys by returning the `AdditionalConfigData` field.

#### Storage Class (created by admin)

**Note:** depending on community input a storage class may be replaced by a namespaced `BucketClass` custom resource.

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: MyAwsS3Class
  labels: 
    aws-s3/object [1]
provisioner: aws-s3.io/bucket [2]
parameters: [3]
  region: us-west-1
  secretName: s3-bucket-owner
  secretNamespace: s3-provisioner
  bucketName: existing-bucket [4]
reclaimPolicy: Delete [5]
```
1. (optional) the label here associates this StorageClass to a specific provisioner.
1. provisioner responsible for handling OBCs referencing this StorageClass.
1. **all** parameter keys and values are specific to a provisioner, are optional, and are not validated by the StorageClass API.
Fields to consider are object-store endpoint, version, possibly a secretRef containing info about credential for new bucket owners, etc.
1. bucketName is required for access to existing buckets.
Unlike greenfield provisioning, the brownfield bucket name appears in the storage class, not the OBC.
1. each provisioner decides how to treat the _reclaimPolicy_ when an OBC is deleted. Supported values are:
+ _Delete_ = (typically) physically delete the bucket.
Depending on new vs. existing bucket, the provisioner's `Delete` or `Revoke` methods are called.
+ _Retain_ = (typically) do not physically delete the bucket.
Depending on the provisioner, various clean up steps can be performed, such as deleting users, revoking credentials, etc.
For both new and existing buckets the provisioner's `Revoke` method is called.

#### App Pod (independent of provisioner)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app-pod
  namespace: dev-user
spec:
  containers:
  - name: mycontainer
    image: redis
    envFrom: [1]
    - configMapRef: 
        name: MY-BUCKET-1 [2]
    - secretRef:
        name: MY-BUCKET-1 [3]
```
1. use `env:` if mapping of the defined key names to the env var names used by the app is needed.
1. makes available to the pod as env variables: BUCKET_HOST, BUCKET_PORT, BUCKET_NAME
1. makes available to the pod as env variables: ACCESS_KEY_ID, SECRET_ACCESS_KEY

### Library Usage

The current [bucket library](https://github.com/kube-object-storage/lib-bucket-provisioner) is imported by the following projects:

+ [operator hub](https://operatorhub.io/operator/lib-bucket-provisioner)

+ Rook-Ceph [examples](https://github.com/rook/rook/tree/master/cluster/examples/kubernetes/ceph), [code](https://github.com/rook/rook/tree/master/pkg/operator/ceph/object/bucket)

+ Noobaa [design](https://github.com/rook/rook/blob/master/design/noobaa/noobaa-obc-provisioner.md), [code](https://github.com/noobaa/noobaa-operator)

There is also an [S3 example](https://github.com/yard-turkey/aws-s3-provisioner).

### Current Restrictions

+ there is no event recording thus events are not shown in commands like `kubectl describe obc`.
+ there is no ability to _cancel_ bucket provisioning
+ there is no way to define a _reclaimPolicy_ that supports erasing or suspending a bucket
+ there are no bucket metrics
+ there is no bucket lifecycle management (e.g. ability to define expiration, archive, migration, etc. policies)
+ security relies solely on RBAC, thus there is no way to distinguish bucket access within the same namespace
+ there is no HA due to no leader election in the lib -- if the provisioner is running in a goroutine (e.g. rook-ceph provisioner) and it fails the lib cannot be restarted
+ logging verbosity levels are somewhat arbitrary


### Risks and Mitigations

### Test Plan

**Note:** *Section not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.
Anything that would count as tricky in the implementation and anything particularly challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage expectations).
Please adhere to the [Kubernetes testing guidelines][testing-guidelines] when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md

### Graduation Criteria

**Note:** *Section not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. Initial KEP should keep
this high-level with a focus on what signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

#### Examples

These are generalized examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

##### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

##### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and GA/stable, since there's no opportunity for user feedback, or even bug reports, in back-to-back releases.

##### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/conformance-tests.md

### Upgrade / Downgrade Strategy

If applicable, how will the component be upgraded and downgraded? Make sure this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing cluster required to make on upgrade in order to make use of the enhancement?

### Version Skew Strategy

If applicable, how will the component handle version skew with other components? What are the guarantees? Make sure
this is in the test plan.

Consider the following in developing a version skew strategy for this enhancement:
- Does this enhancement involve coordinating behavior in the control plane and in the kubelet? How does an n-2 kubelet without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI, CRI or CNI may require updating that component before the kubelet.

## Implementation History

Major milestones in the life cycle of a KEP should be tracked in `Implementation History`.
Major milestones might include

- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded

## Drawbacks [optional]

Why should this KEP _not_ be implemented.

## Alternatives [optional]

Various alternative designs were considered before reaching the design described here:
1. Using a service broker to provision buckets.
This doesn't alleviate the pod from consuming env variables which define the endpoint and secret keys.
It also feels too far removed from basic Kubernetes storage -- there is no claim and no object to represent the bucket.
1. A Rook-Ceph only provisioner with built-in watches, reconciliation, etc, **but** no bucket library. The main problem here is that each provisioner would need to write all of the controller code themselves.
This could easily result in different _contracts_ for different provisioners, meaning one provisioners might create the Secret of ConfigMap differently than another.
This could result in the app pod being coupled to the provisioner.
1. Rook-Ceph repo and a centralized controller.
Initially we considered a somewhat generic bucket provisioner living in the Rook repo and embedded in their existing operator (which is used to provision a ceph object store, an object user, etc).
Feedback from the Rook community was that it didn't make sense for a generic (non-rook focused) controller to live inside Rook.
