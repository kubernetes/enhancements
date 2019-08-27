---
title: StorageVersion API for HA API servers 
authors:
  - "@xuchao"
owning-sig: sig-api-machinery
reviewers:
  - "@deads2k"
  - "@yliaog"
  - "@lavalamp"
approvers:
  - "@deads2k"
  - "@lavalamp"
creation-date: 2019-08-22
last-updated: 2019-08-22
status: provisional
---

# StorageVersion API for HA API servers

## Table of Contents

* [Overview](#overview)
* [API changes](#api-changes)
   * [Resource Version API](#resource-version-api)
* [Changes to API servers](#changes-to-api-servers)
   * [Curating a list of participating API servers in HA master](#curating-a-list-of-participating-api-servers-in-ha-master)
   * [Updating StorageVersion](#updating-storageversion)
   * [Garbage collection](#garbage-collection)
   * [CRDs](#crds)
   * [Aggregated API servers](#aggregated-api-servers)
* [Consuming the StorageVersion API](#consuming-the-storageversion-api)
* [StorageVersion API vs. StorageVersionHash in the discovery document](#storageversion-api-vs-storageversionhash-in-the-discovery-document)
* [Backwards Compatibility](#backwards-compatibility)
* [Graduation Plan](#graduation-plan)
* [FAQ](#faq)
* [Alternatives](#alternatives)
   * [Letting API servers vote on the storage version](#letting-api-servers-vote-on-the-storage-version)
   * [Letting the storage migrator detect if API server instances are in agreement](#letting-the-storage-migrator-detect-if-api-server-instances-are-in-agreement)
* [Appendix](#appendix)
   * [Accuracy of the discovery document of CRDs](#accuracy-of-the-discovery-document-of-crds)
* [References](#references)

## Overview

During the rolling upgrade of an HA master, the API server instances may
use different storage versions encoding a resource. The [storageVersionHash][]
in the discovery document does not expose this disagreement. As a result, the
storage migrator may proceed with migration with the false belief that all API
server instances are encoding objects using the same storage version, resulting
in polluted migration.  ([details][]).

[storageVersionHash]:https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L979
[details]:https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/35-storage-version-hash.md#ha-masters

We propose a way to show what storage versions all API servers are using, so
that the storage migrator can defer migration until an agreement has been
reached.

## API changes

### Resource Version API

We introduce a new API `StorageVersion`, in a new API group
`internal.apiserver.k8s.io/v1alpha1`.

```golang
//  Storage version of a specific resource.
type StorageVersion struct {
  TypeMeta
  // The name is <group>.<resource>.
  // TODO: use the ResourceID [1] as the name to avoid duplicates. 
  ObjectMeta
  
  // Spec is omitted because there is no spec field.
  // Spec StorageVersionSpec

  // API server instances report the version they can decode and the version they
  // encode objects to when persisting objects in the backend.
  Status StorageVersionStatus
}

// API server instances report the version they can decode and the version they
// encode objects to when persisting objects in the backend.
type StorageVersionStatus struct {
  // The reported versions per API server instance.
  ServerStorageVersions []ServerStorageVersion
  // If all API server instances agree on the same encoding storage version, then
  // this field is set to that version. Otherwise this field is set to
  // NoAgreedVersion.
  AgreedEncodingVersion string
}

const (
  // The API server instances haven't reached agreement on the encoding storage
  // version.
  NoAgreedVersion = "No Agreed Version"
)

// An API server instance reports the version it can decode and the version it
// encodes objects to when persisting objects in the backend.
type ServerStorageVersion struct {
  // The ID of the reporting API server. 
  // For a kube-apiserver, the ID is configured via a flag.
  APIServerID string

  // The API server encodes the object to this version when persisting it in
  // the backend (e.g., etcd).
  EncodingVersion string

  // The API server can decode objects encoded in these versions.
  // The encodingVersion must be included in the decodableVersions.
  DecodableVersions []string
}
```

[1]: [ReousrceID](https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/34-storage-hash.md#api-changes-to-the-discovery-api)

## Changes to API servers

In this section, we describe how to update and consume the StorageVersion API.

### Curating a list of participating API servers in HA master

API servers need such a list when updating the StorageVersion API. Currently,
such a list is already maintained in the "kubernetes" endpoints, though it is not
working in all flavors of Kubernetes deployments.

We will inherit the existing [mechanism][], but formalize the API and process in
another KEP. In this KEP, we assume all API servers have access to the list of
all participating API servers via some API.

[mechanism]:https://github.com/kubernetes/community/pull/939

### Updating StorageVersion

During bootstrap, for each resource, the API server 
* gets the storageVersion object for this resource, or creates one if it does
  not exist yet,
* gets the list of participating API servers,
* updates the storageVersion locally. Specifically,
  * creates or updates the .status.serverStorageVersions, to express this API
    server's decodableVersions and encodingVersion.
  * removes .status.serverStorageVersions entries whose server ID is not present
    in the list of participating API servers, such entries are stale.
  * checks if all participating API servers agree on the same storage version.
    If so, sets the version as the status.agreedEncodingVersion. If not, sets
    the status.agreedEncodingVersion to "No Agreed Version".
* updates the storageVersion object, using the rv in the first step
  to avoid conflicting with other API servers.
* installs the resource handler.

### Garbage collection

There are two kinds of "garbage":

1. stale storageVersion.status.serverStorageVersions entries left by API servers
   that have gone away;
2. storageVersion objects for resources that are no longer served.

We can't rely on API servers to remove the first kind of stale entries during
bootstrap, because an API server can go away after other API servers bootstrap,
then its stale entries will remain in the system until one of the other API
servers reboots.

Hence, we propose a leader-elected control loop in API server to clean up the
stale entries, and in turn clean up the obsolete storageVersion objects. The
control loop watches the list of participating API servers, upon changes, it
performs the following actions for each storageVersion object:

* gets a storageVersion object
* gets the list of participating API servers,
* locally, removes the stale entries (1st kind of garbage) in
  storageVersion.status.serverStorageVersions,
  * after the removal, if all participating API servers have the same
    encodingVersion, then sets storageVersion.status.AgreedEncodingVersion. 
* checks if the storageVersion.status.serverStorageVersions is empty,
  * if empty, deletes the storageVersion object (2nd kind of garbage),
  * otherwise updates the storageVersion object,
  * both the delete and update operations are preconditioned with the rv in the
    first step to avoid conflicting with API servers modifying the object.

An API server needs to establish its membership in the list of participating API
servers before updating storageVersion, otherwise the above control loop can
mistake a storageVersion.status.serverStorageVersions entry added by a new API
server as a stale entry.

### CRDs

Today, the [storageVersionHash][] in the discovery document in HA setup can
diverge from the actual storage version being used. See the [appendix][] for
details.

[appendix]:#appendix
[storageVersionHash]:https://github.com/kubernetes/kubernetes/blob/c008cf95a92c5bbea67aeab6a765d7cb1ac68bd7/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L989

To accurately reflect the storage version being used, the apiextension-apiserver
needs to update the storageVersion object when it [creates][] the custom
resource handler upon CRD creation or changes.

[creates]:https://github.com/kubernetes/kubernetes/blob/220498b83af8b5cbf8c1c1a012b64c956d3ebf9b/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_handler.go#L721

### Aggregated API servers

Most code changes will be done in the generic apiserver library, so aggregated
API servers using the library will get the same behavior.

If an aggregated API server does not use the API, then the storage migrator does
not manage its API.

## Consuming the StorageVersion API

The consumer of the StorageVersion API is the storage migrator. The storage
migrator
* starts migration if the storageVersion.status.agreedEncodingVersion differs
  from the storageState.status.[persistedStorageVersionHashes][],
* aborts ongoing migration if the storageVersion.status.agreedEncodingVersion is
  "No Agreed Version".

[persistedStorageVersionHashes]:https://github.com/kubernetes-sigs/kube-storage-version-migrator/blob/60dee538334c2366994c2323c0db5db8ab4d2838/pkg/apis/migration/v1alpha1/types.go#L164

## StorageVersion API vs. StorageVersionHash in the discovery document

We do not change how the storageVersionHash in the discovery document is
updated. The only consumer of the storageVersionHash is the storage migrator,
which will convert to use the new StorageVersion API. After the StorageVersion
API becomes stable, we will remove the storageVersionHash from the discovery
document, following the standard API deprecation process.

## Backwards Compatibility

There is no change to the existing API, so there is no backwards compatibility
concern.

## Graduation Plan

* alpha: in 1.17, the StorageVersion API and related mechanism will be feature
  gated by the `ExposeStorageVersion` flag.
* beta1 in 1.18, beta2 in 1.19. We make two beta releases to allow more time for
  feedback.
* GA in 1.20.

## FAQ

1. Q: if an API server is rolled back when the migrator is in the middle of
   migration, how to prevent corruption? ([original question][])

   A: Unlike the discovery document, the new StorageVersion API is persisted in
   etcd and has the resourceVersion(RV) field, so the migrator can determine if
   the storage version has changed in the middle of migration by comparing the
   RV of the storageVersion object before and after the migration. Also, as an
   optimization, the migrator can fail quickly by aborting the ongoing migration
   if it receives a storageVersion change event via WATCH.

   [original question]:https://github.com/kubernetes/enhancements/pull/1176#discussion_r307977970

## Alternatives

### Letting API servers vote on the storage version

See [#1201](https://github.com/kubernetes/enhancements/pull/920)

The voting mechanism makes sure all API servers in an HA cluster always use the
same storage version, and the discovery document always lists the selected
storage version.

Cons:
* The voting mechanism adds complexity. For the storage migrator to work
  correctly, it is NOT necessary to guarantee all API server instances always
  use the same storage version.

### Letting the storage migrator detect if API server instances are in agreement

See [#920](https://github.com/kubernetes/enhancements/pull/920)

Cons: it has many assumptions, see [cons][].
[cons]:https://github.com/kubernetes/enhancements/pull/920/files#diff-a1d206b4bbac708bf71ef85ad7fb5264R339

## Appendix

### Accuracy of the discovery document of CRDs

Today, the storageVersionHash listed in the discovery document "almost"
accurately reflects the actual storage version used by the apiextension-apiserver.

Upon storage version changes in the CRD spec,
* [one controller][] deletes the existing resource handler of the CRD, so that
  a new resource handler is created with the latest cached CRD spec is created
  upon the next custom resource request. 
* [another controller][] enqueues the CRD, waiting for the worker to updates the
  discovery document.

[one controller]:https://github.com/kubernetes/kubernetes/blob/1a53325550f6d5d3c48b9eecdd123fd84deee879/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_handler.go#L478
[another controller]:https://github.com/kubernetes/kubernetes/blob/1a53325550f6d5d3c48b9eecdd123fd84deee879/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_discovery_controller.go#L258

These two controllers are driven by the [same informer][], so the lag between
when the server starts to apply the new storage version and when the discovery
document is updated is just the difference between when the respective
goroutines finish.
[same informer]:https://github.com/kubernetes/kubernetes/blob/1a53325550f6d5d3c48b9eecdd123fd84deee879/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/apiserver.go#L192-L210

Note that in HA setup, there is a lag between when apiextension-apiserver
instances observe the CRD spec change.

## References
1. Email thread [kube-apiserver: Self-coordination](https://groups.google.com/d/msg/kubernetes-sig-api-machinery/gTS-rUuEVQY/9bUFVnYvAwAJ)
