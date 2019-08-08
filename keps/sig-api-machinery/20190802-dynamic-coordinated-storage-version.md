---
title: Coordinated dynamic storage version 
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
creation-date: 2019-08-02
last-updated: 2019-08-02
status: provisional
---

# Coordinated dynamic storage version

## Table of Contents

* [Overview](#overview)
* [API changes](#api-changes)
   * [Storage Version API](#storage-version-api)
* [Changes to API servers](#changes-to-api-servers)
   * [Curating a list of participating API servers in HA master](#curating-a-list-of-participating-api-servers-in-ha-master)
   * [Voting for Storage Version](#voting-for-storage-version)
   * [Voting in action](#voting-in-action)
   * [The selected storage version is supported by all API servers during upgrades/downgrades](#the-selected-storage-version-is-supported-by-all-api-servers-during-upgradesdowngrades)
   * [Garbage collection of the StorageVersion objects of removed resources](#garbage-collection-of-the-storageversion-objects-of-removed-resources)
   * [Using the selected storage version when serializing data for etcd](#using-the-selected-storage-version-when-serializing-data-for-etcd)
   * [Updating the discovery document](#updating-the-discovery-document)
   * [CRDs](#crds)
   * [Aggregated API servers](#aggregated-api-servers)
* [Backwards Compatibility](#backwards-compatibility)
* [Risks](#risks)
* [Graduation Plan](#graduation-plan)
* [FAQ](#faq)
* [Future work: solving other coordination problems](#future-work-solving-other-coordination-problems)
* [Alternatives](#alternatives)
   * [Exposing if API servers in an HA setup are using the same storage version](#exposing-if-api-servers-in-an-ha-setup-are-using-the-same-storage-version)
   * [Letting the storage migrator detect if API server instances are in agreement](#letting-the-storage-migrator-detect-if-api-server-instances-are-in-agreement)
* [References](#references)

## Overview

During the rolling update of an HA master, API server instances *i)* use
different storage versions for a built-in API resource, and *ii)* show different
storageVersionHash in the discovery documents. These facts make the storage
migrator not working properly ([details][]).

[storageVersionHash]:https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L979
[details]:https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/35-storage-version-hash.md#ha-masters

We propose a mechanism for HA API servers to vote on the storage version to use,
then show the selected storage version in the discovery document and switch to
use the selected version atomically.

## API changes

### Storage Version API

We introduce a new API `StorageVersion`, in a new API group
`kube-apiserver.internal.k8s.io/v1alpha1`. 

```golang
// Storage version for a specific resource.
type StorageVersion struct {
  TypeMeta
  // The name is the GVR, i.e., <group>.<version>.<resource>.
  ObjectMeta
  // The selected storage version. All API server intances encode the resource
  // objects in this version when committing it to etcd.
  SelectedVersion string
  // Proposed storage version candidate, keyed by the ID of the proposing API
  // server.
  CandidateVersions []VersionCandidate
}

type VersionCandidate struct {
  // The ID of the reporting API server. 
  APIServerID string
  // The preferred storage version of the reporting API server. The preferred
  // storage version is hardcoded in each Kubernetes release.
  PreferredVersion string
}
```

We will discuss how the `StorageVersion` is updated and used in the [Changes to API
servers](#changes-to-api-servers) section.

## Changes to API servers

### Curating a list of participating API servers in HA master

Currently, API servers maintain such a list in the "kubernetes" endpoints. See
this [design doc][] for details.

We will formalize the API in another KEP. For the purpose of this KEP, all we
need to know is that HA master is able to maintain a list of participating API
servers in etcd.

[design doc]:https://github.com/kubernetes/community/pull/939

### Voting for Storage Version

During bootstrap, for each resource, the kube-apiserver 
* gets the StorageVersion object for this resource,
  * if the object does not exist, creates one, setting this kube-apiserver's
  preferred storage version as the SelectedVersion, also adding its preferred
  storage version to the CandidateVersions list. Then jumps to the last step
  (installing the resource handler).
* gets the list of participating API servers,
* updates the StorageVersion locally. Specifically,
  * creates or updates the CandidateVersions, to express this kube-apiserver's
    preferred storage version.
  * checks if there is any version candidate whose APIServerID is not a
    participating API server. This means the API server has gone and the entry
    is stale. Removes such entries.
  * checks if all participating API servers vote for the same version. If so,
    sets the version as the SelectedVersion.
* updates the StorageVersion object, using the resourceVersion in the first step
  to avoid conflicting with other API servers.
* installs the resource handler. Uses the SelectedVersion as the storage
  version.

We need to make sure that an API server establishes its membership in the list
of participating API servers before reporting its preferred storage version via
the StorageVersion API. Then the above procedure makes it impossible to mistake
a new CandidateVersions entry as a stale one.

### Voting in action

**Scenario 1**, rolling upgrading HA API servers to a release that starts using
the storage version voting.

In this case, the first upgraded API server sets its preferred storage version as
the selected version. Because the other not-upgraded-yet API servers do not
respect the voting system, they might use a different version as the storage
version. When they get upgraded, they will use the selected version as the
storage version.

**Scenario 2**, rolling upgrading HA API servers that have already enabled the
storage version voting.

In this case, the API servers will keep using the old selected storage version
until all instances are upgraded to the new release. The last upgraded API
server will change the selected version to the new storage version, and all API
servers start to use the new version.

### The selected storage version is supported by all API servers during upgrades/downgrades

A nice property of the selected storage version is that it is supported by all
API servers during rolling upgrades/downgrades. This means that the proposed
dynamic storage version mechanism does not introduce any API server downtime.

The speed of Kubernetes API evolution is restricted by the [deprecation
policy][] rule `#4a` and `#4b`. The following table illustrates the
fastest possible evolution of the storage version of a resource.

|                           | release x | release x+1 | release x+2 | release x+3 |
|---------------------------|-----------|-------------|-------------|-------------|
| supported versions        | v1        | v1, v2      | v1, v2      | v2          |
| preferred storage version | v1        | v1          | v2          | v2          |

Note that the selected storage version only changes between release x+1 and x+2.
Because both releases support both v1 and v2, during the rolling
upgrade/downgrade, all API server instances support the selected storage
version.

### Garbage collection of the StorageVersion objects of removed resources

During bootstrap, the kube-apiserver
* gets the list of all participating API servers,
* gets all the StorageVersion objects,
* removes the StorageVersion object if none of its CandidateVersions.APIServerID
  is a participating API server.

### Using the selected storage version when serializing data for etcd

Depending on how strict the guarantee we want to provide, we have two
alternative designs.

**1. strict guarantee:** kube-apiserver always use the selected storage version

To guarantee this, kube-apiserver needs to add a transaction condition to the
write requests, to make sure the version it uses to serialize the API object is
the selected storage version recorded in the API server.

The performance cost is small, because all write operations from the
kube-apiserver to etcd are transactional already.  For example, kube-apiserver
checks if key does not already exist before committing a create operation. As
another example, kube-apiserver checks that the resourceVersion is the latest
before committing an update operation. Adding another transaction condition does
not add extra RPC call.

If the transaction fails, the kube-apiserver gets the latest selected storage
version from etcd, re-serializes the data and retries the commit. The
kube-apiserver will use this latest selected storage version in the following
operations.

**2. loose guarantee:** kube-apiserver has a very high probability to use the
selected storage version 1 minute after the version is selected in etcd.

To guarantee this, the kube-apiserver can rely on a watch channel to deliver
the latest storage version within 1 minute. 

This saves the performance cost for the server, but shifts the complexity to the
clients. For example, the storage migrator should start migration 1 minute after
detecting the storage version change.


### Updating the discovery document

Currently, the apiservers expose the storage version via the
[storageVersionHash][] field in the discovery document. The storage versions for
built-in resources have been static. With this KEP, the kube-apiserver needs to
dynamics update the storage version hash in the discovery document when the it
gets the latest selected storage version from etcd.

[storageVersionHash]:https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L979

### CRDs

Today, a [apiextension-apiserver][] uses informer to watch for the latest
change to the CRD spec, and [builds][] new storage encoder with the latest storage
version.

The CRDs do not need the `StorageVersion` API to vote for a selected storage
version. The storage version in the CRD spec is the "selected storage version".

The current implementation of the apiextension-apiserver already provides the
"loose guarantee".

If we choose to pursue the "strict guarantee", most code changes will be done in
the [etcd3 storage layer], which is shared by the kube-apiserver and
apiextension-apiservers, so the CR storage will get the same enhancement. 

[apiextension-apiserver]:https://github.com/kubernetes/kubernetes/blob/c91761da0d854aa842ea55ba4d7c6cbc8d675892/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_handler.go#L204
[builds]:https://github.com/kubernetes/kubernetes/blob/c91761da0d854aa842ea55ba4d7c6cbc8d675892/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_handler.go#L678
[etcd3 storage layer]:https://github.com/kubernetes/kubernetes/blob/74c0cc27902548243d8a863c9e2c6345a8e6b548/staging/src/k8s.io/apiserver/pkg/storage/etcd3/store.go

### Aggregated API servers

Most code changes will be done in the generic apiserver library, so aggregated
API servers using the library will get the same behavior.

## Backwards Compatibility

Clients do not see any behavioral change, so there is no backwards compatibility
concern.

## Risks

1. If an operator jumps versions (e.g., upgrade directly from v1.15 to v1.17)
   when rolling upgrade its HA master, and if the new API server does not
   support the old SelectedVersion, then there are two consequences:

   a. during the rolling upgrade, an upgraded API server does not support the
   SelectedVersion. Depending on the implementation, it may either rejects all
   write operation to that resource, or use a different storage version. Neither
   is great.

   b. after the rolling upgrade is done, because the existing data in etcd is
   encoded in the old storage version, which is not supported by the API
   servers, no one can read or write the existing data.

## Graduation Plan

* Alpha: in 1.16, the newly added API, including the `DiscoveryDocHashes` and
  the `Consistent` field, will be feature gated by the
  `EnableCoordinatedDiscoveryDocument` flag.

* Beta & GA: if we don't find problems, we will graduate the API quarterly.

## FAQ

1. Q: how to prevent the following race between a client and the HA master:
   client gets the discovery document, see storage version v1; HA master rolling
   upgrades to use storage version v2; client sends a create request, expecting
   the object to be encoded as v1 in etcd.

   A: Unless API servers support transactions, the race is inevitable, and
   exists in all parts of Kubernetes API. Operators' involvement is necessary
   to overcome the race. Let's take the storage migrator as an example. For the
   storage migrator to work properly, operators needs to make sure that at most
   one master version change could happen in any 5 minute window. Then, the
   storage migrator just checks the discovery document every 5 minutes, as long
   as the storage version in the discovery document hasn't changed, the storage
   migrator can safely assume that the API servers have only used this storage
   version in this 5 minute window.

## Future work: solving other coordination problems

The storage version selection and the coordinated storage version switching
isn't required to make the storage migrator work properly. In the alternative
[KEP][], API servers expose if they use the same storage version, and the
storage migrator only takes action when API servers are in agreement. That
mechanism requires the client, the storage migrator, to deal with the internals
of HA API servers, thus the mechanism is not extensible to solve other
coordination issues.

On the other hand, the voting mechanism and the coordinated storage version
switching mechanism in this KEP hides all the internal coordination from the
clients. The semantics is more intuitive. The mechanisms can be extended to
solve other coordination issues facing the HA master. For example,
kube-apiservers can vote on what API is served, and only serve the API that is
supported by all servers.

[KEP]:https://github.com/kubernetes/enhancements/pull/1176

## Alternatives

### Exposing if API servers in an HA setup are using the same storage version

This is described in this [KEP][]. In short, API servers report the storage
version they are using. A "Consistent" field is added to the discovery document.
If API servers are using different storage version, "Consistent" is set to
"false", and otherwise set to "true". 

Cons:
1. clients need to understand the semantics of the "Consistent" field.
2. The "Consistent" field can flip back and forth. For example, if an
   kube-apiserver is downgraded temporarily, the "Consistent" field will change
   to "false" immediately. In comparison, the selected storage version proposed
   in this KEP will not change.

[KEP]:https://github.com/kubernetes/enhancements/pull/1176

### Letting the storage migrator detect if API server instances are in agreement

See [#920](https://github.com/kubernetes/enhancements/pull/920)

Cons: it has many assumptions, see [cons][].
[cons]:https://github.com/kubernetes/enhancements/pull/920/files#diff-a1d206b4bbac708bf71ef85ad7fb5264R339

## References
1. Email thread [kube-apiserver: Self-coordination](https://groups.google.com/d/msg/kubernetes-sig-api-machinery/gTS-rUuEVQY/9bUFVnYvAwAJ)
