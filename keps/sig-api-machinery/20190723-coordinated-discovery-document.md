---
title: Coordinated Discovery Documents for HA API Servers
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
creation-date: 2019-07-23
last-updated: 2019-07-25
status: provisional
---

# Coordinated Discovery Documents for HA API Servers

## Table of Contents

* [Problem Statement](#problem-statement)
* [Summary of Proposed changes](#summary-of-proposed-changes)
* [API changes](#api-changes)
* [Changes to API servers](#changes-to-api-servers)
   * [Calculating DiscoveryDocHashes](#calculating-discoverydochashes)
   * [Periodically reporting DiscoveryDocHashes &amp; Garbage Collection](#periodically-reporting-discoverydochashes--garbage-collection)
   * [Handling discovery requests](#handling-discovery-requests)
* [Backwards Compatibility](#backwards-compatibility)
* [Graduation Plan](#graduation-plan)
* [Alternatives](#alternatives)

## Problem Statement

API servers today compute discovery documents based on their local states. When
HA API servers are rolling upgraded, API server instances can return
different discovery documents to clients. This causes problems. For
example, the storage migrator does not work reliably with HA API servers
([details][]).

[details]:https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/35-storage-version-hash.md#ha-masters

This KEP proposes a mechanism for HA API servers to expose to clients if
all API server instances are serving consistent discovery document.

## Summary of Proposed changes

Periodically, each API server instance in an HA setup calculates a hash based on
the discovery documents of the built-in APIs. The hash value is reported in a
new API, `DiscoveryDocHashes`, together with the API server's ID. 

When responding to discovery document requests, apart from the original
discovery documents, the API server also sets a `Consistent` field to indicate if
all API servers are serving the same discovery documents, i.e., if all hash
values recorded in `DiscoveryDocHashes` are equal. 

## API changes

We introduce a new API `DiscoveryDocHashes`, in a new API group
`APIServerCoordination.k8s.io`. 

```golang
type DiscoveryDocHashes struct {
   Hashes []Hash
}

type Hash struct {
   // The ID of the reporting API server. 
   // The ID can be generated in a similar fashion as the controller manager
   // generates leader election id at
   // https://github.com/kubernetes/kubernetes/blob/2321d1e9e8950cc94b3ef2368dfaacec61f1ba4f/cmd/cloud-controller-manager/app/controllermanager.go#L185.
   // Required.
   APIServerID string

   // A hash calculated based on the discovery documents of built-in APIs. If
   // two hashes equal, it means that the two API servers serve the same set of
   // APIs.
   // Required.
   BuiltInAPIHash string

   // LastHeartbeatTime is the last time the reporting API server updates this 
   // field.
   // Required 
   LastHeartbeatTime metav1.Time
}
```

API server instances report the hashes of their discovery documents through this
API.  Although the main users of this API are the API servers, we make this API
public to make debugging convenient.

Another API change is that at each level of the discovery [API][] (APIGroupList,
APIVersions, and APIResourceList), we add a new field `Consistent` to indicate
if all API servers agree on the discovery documents, i.e., if all hashes
reported in `DiscoveryDocHashes` are equal.

Alternative to adding the `Consistent` field, we have considered letting API
servers respond 500 error to discovery requests if there are multiple hash
values in `DiscoveryDocHashes`. This is less desirable as it breaks old clients.

## Changes to API servers

### Calculating DiscoveryDocHashes

Every API server instance calculates the DiscoveryDocHashes based on its in
memory discovery documents for **built-in** resources. One API server instance
only generates one DiscoveryDocHashes. And because built-in resources are
static, the hash does not change through the life time of the API server
process.

The hash does not include the CRDs, because API server instances *can* serve
consistently if using the `CRD` API as a coordination point. For example, when
serving a CR request, if the API server always uses a [handler][] built based on
the latest `CRD` stored in etcd, then all API server instances will behave
consistently.

[handler]:https://github.com/kubernetes/kubernetes/blob/63a43402a365f7a7615e01ea3e174fb8a71d67a8/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/customresource_handler.go#L200

Similarly, for aggregated APIs, the API servers can use the `APIService` API as
a coordination point to achieve consistency.

### Periodically reporting DiscoveryDocHashes & Garbage Collection

Because API server instances come and go, we need a mechanism to garbage collect
stale DiscoveryDocHashes records created by decommissioned API servers.

For this purpose, every API server needs to periodically updates the
`lastHeartbeatTime` of its record in `DiscoveryDocHashes`. Tentatively we will
set the update period to 5 minutes. If an entry hasn't been updated for 3
periods (15 mins), then any API server instance can remove the entry.

### Handling discovery requests

When responding discovery requests, apart from the original discovery documents,
the API server also sets a `Consistent` field to indicate if all API servers are
serving the same discovery documents, i.e., if all hash values recorded in
`DiscoveryDocHashes` are equal. 

## Backwards Compatibility

For old clients that do not understand the `Consistent` field in the discovery
documents, they ignore the new field and API servers appear to behave the same.

In an HA setup, if some API servers support the `CoordinatedDiscoveryDocument`
feature while others do not, then the status reported in the `Consistent` field
is inaccurate. Human intervention is needed to make sure clients only rely on
the feature after all API servers support it.

## Graduation Plan

* Alpha: in 1.16, the newly added API, including the `DiscoveryDocHashes` and
  the `Consistent` field, will be feature gated by the
  `EnableCoordinatedDiscoveryDocument` flag.

* Beta & GA: if we don't find problems, we will graduate the API quarterly.

## Alternatives

### Adding a StorageVersion API to coordinate storage versions used by API servers

The purpose is to make the storage migrator work reliably in an HA setup (see
[details][]).

We introduce a new API, `StorageVersion`. For each API resource, there is a
`StorageVersion` instance recording the storage version of that resource. When
serving a write request, an API server checks if the storage version it plans to
use matches the version declared in the `StorageVersion` object, if not, the API
server rejects the write request. This makes sure that there is only a single
storage version respected by all API server instances at any given time.

[details]:https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/35-storage-version-hash.md#ha-masters

Cons:
   1. this defeats the purpose of HA, as some API servers will not be able to
      handle write requests during rolling upgrades.
   2. it is unclear who (which API server, or admin) has the right to modify the
      `StorageVersion` object.

### Letting the storage migrator detect if API server instances are in agreement

See [#920](https://github.com/kubernetes/enhancements/pull/920)
