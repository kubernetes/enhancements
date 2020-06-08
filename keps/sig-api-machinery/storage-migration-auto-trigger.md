---
title: Automated Storage Version Migration with Storage Version Hash
authors:
  - "@xuchao"
owning-sig: sig-api-machinery
reviewers:
  - "@deads2k"
  - "@yliaog"
approvers:
  - "@deads2k"
  - "@lavalamp"
creation-date: 2019-01-23
last-updated: 2019-01-23
status: provisional
---

## Table of Contents

<!-- toc -->
- [Goal](#goal)
- [API design](#api-design)
- [Storage migration triggering controller](#storage-migration-triggering-controller)
- [Implications to cluster operators](#implications-to-cluster-operators)
- [Life-cycle of a StorageState object](#life-cycle-of-a-storagestate-object)
- [Future work: HA clusters](#future-work-ha-clusters)
- [Future work: persisted discovery document](#future-work-persisted-discovery-document)
<!-- /toc -->

## Goal

As the discovery document now exposes the [storage version hash][], storage version
migration should start automatically when the hash changes.

[storage version hash]:https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/35-storage-version-hash.md

## API design

We introduce the `StorageState` API, as a CRD defined in the `migration.k8s.io`
group.

```golang
// StorageState is the state of the storage, for a specific resource.
type StorageState struct {
  metav1.TypeMeta
  // The name is "<resource>.<group>". The API validation will enforce it by
  // comapring the name with the spec.resource.
  metav1.ObjectMeta
  Spec StorageStateSpec
  Status StorageStateStatus
}

type StorageStateSpec {
  // The resource this StorageState is about.
  Resource GroupResource
}

type StorageStateStatus {
  // The hash values of storage versions that persisted instances of
  // spec.resource might still be encoded in.
  // "Unknown" is a valid value in the list, and is the default value.
  // It is not safe to upgrade or downgrade to an apiserver binary that does not
  // support all versions listed in this field, or if "Unknown" is listed.
  // Once the storage version migration for this resource has completed, the
  // value of this field is refined to only contain the
  // currentStorageVersionHash.
  // Once the apiserver has changed the storage version, the new storage version
  // is appended to the list.
  // +optional
  PersistedStorageVersionHashes []string
  // The hash value of the current storage version, as shown in the discovery
  // document served by the API server.
  // Storage Version is the version to which objects are converted to
  // before persisted.
  CurrentStorageVersionHash string
  // LastHeartbeatTime is the last time the storage migration triggering
  // controller checks the storage version hash of this resource in the
  // discovery document and updates this field.
  // +optional
  LastHeartbeatTime metav1.Time
}
```

We had considered making `PersistedStorageVersionHashes` part of the `spec`
instead of the `status`, because the stored versions cannot be deduced
immediately by inspecting Kubernetes API. However, we decided to put it in the
status because
* Its value eventually is determined by the `StorageVersionMigration` API, i.e.,
  it's set to the storage version when the corresponding migration has
  completed.
* Even though we cannot reconstruct this field immediately, this field is
  initialized to ["Unknown"], which truthfully reflects the state.
* Putting it in the spec is even more unnatural. Spec describes the desired
  state the system should eventually reach in the future, while this field is
  about states in the history.
* Keeping it in the status is consistent with the [CRD.status.StoredVersions][]
  API.

[CRD.status.StoredVersions]:https://github.com/kubernetes/kubernetes/blob/697c2316faaabae8ef8371032b60be65d7795e68/staging/src/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1/types.go#L305

## Storage migration triggering controller

We will add a controller that monitors the discovery documents and the
storageStates to trigger storage version migration.

The controller fetches the discovery document periodically. For each resource,
if the storageState doesn't exist for the resource, the controller
  * stops any ongoing migration of this resource by deleting existing
    [storageVersionMigrations][],
  * creates a new storageVersionMigration for this resource,
  * creates the storageState for the resource, setting
    .status.persistedStorageVersionHashes to ["Unknown"], and setting
    status.currentStorageVersionHash to the value shown in the discovery
    document.

If the storageState exists, the controller compares the storageVersionHash in
the discovery document with storageState.status.currentStorageVersionHash. If
they are the same, the controller simply updates the status.lastHeartbeatTime
and is done.
Otherwise, the controller
  * stops any ongoing migration of this resource by deleting existing
    [storageVersionMigrations][],
  * creates a new storageVersionMigration for this resource,
  * as a single operation, updates the status.lastHeartbeatTime, sets
    storageState.status.currentStorageVersionHash to the value shown in the
    discovery document, and appends the hash to the
    storageState.status.persistedStorageVersionHashes.

The refresh rate of the discovery document needs to satisfy two conflicting
goals: *i)* not causing too much traffic to apiservers, and *ii)* not missing
storage version configuration changes. More precisely, the storage version of a
certain resource should not change twice within a refresh period. 10 minutes is
a good middle ground.

The controller uses an informer to monitor all [storageVersionMigrations][]. If a
migration completes, the controller finds the storageState of that resource,
updates its status.persistedStorageVersionHashes to equal its
status.currentStorageVersionHash.

If the controller crash loops and misses storage version changes, the
storageState.status.persistedStorageVersionHashes can be stale. Thus, when the
controller bootstraps, for every storageState object, it checks the
.status.lastHeartbeatTime, if the timestamp is more than 10 minutes old, it
recreates storageState object with persistedStorageVersionHashes set to ["Unknown"].

[storageVersionMigrations]:https://github.com/kubernetes-sigs/kube-storage-version-migrator/blob/444c1beafd4a22684c2b4ba50fa489ec24873c10/pkg/apis/migration/v1alpha1/types.go#L29

## Implications to cluster operators

* Cluster operators can do fast upgrade/downgrade, without waiting for all
  storage migrations to complete, which had been a requirement in the [alpha
  workflow][]. Instead, the cluster operators just need to make sure the
  to-be-deployed apiserver binaries understand all versions recorded in
  storageState.status.persistedStorageVersionHashes. This is useful when a
  cluster needs to roll back.

* On the other hand, if the cluster operators consecutively upgrade or downgrade
  the cluster in a 10 minute window, the migration triggering controller, which
  polls the discovery doc every 10 minutes, will miss the intermediate states of
  the cluster. The storageState will be inaccurate, which will endanger
  the next upgrade/downgrade. Operators need to manually delete all the
  storageState objects to reset.

* The caveat applies to Custom Resource admins, too. If the storage version of a
  CR changes twice in a 10 minute window, admin needs to manually delete the
  storageState of the CR to reset.

[alpha workflow]:https://github.com/kubernetes/enhancements/blob/master/keps/sig-api-machinery/0030-storage-migration.md#alpha-workflow

## Life-cycle of a StorageState object

Let's put all the bits together and describe the life cycle
of a StorageState object. We will use the StorageState of CronJob as an
example.

When the migration triggering controller is installed on an 1.14 cluster for the
first time, there is no StorageState object. The controller creates the
StorageState object for CronJob, setting .status.persistedStorageVersionHashes to
["Unknown"] and .status.currentStorageVersionHash to batch/v2alpha1. It also
creates a storageVersionMigration object to request a migration. When the
migration completed, the migration triggering controller sets
.status.persistedStorageVersionHashes to [batch/v2alpha1].

The migration triggering controller crashes and has been offline for less than
10 minutes. When it's back online, it sees the .status.lastHeartbeatTime is less
than 10 minutes old, thus it considers the storageState still valid. The
controller does nothing other than updating the lastHeartbeatTime.

The cluster admin upgrades the cluster to 1.15. Let's assume the storage version
of the cronjob is changed to batch/v2beta1. The migration triggering controller
observes the change. It sets the storageState.status.currentStorageVersionHash to
batch/v2beta1 and storageState.status.persistedStorageVersionHashes to [v2alpha1,
v2beta1].  It also creates a migration request for cronjobs to migrate to
v2beta1.

The cluster admin finds a critical issue in 1.15 and downgrades the cluster
back to 1.14. The migration triggering controller sets the
storageState.status.currentStorageVersionHash to batch/v2alpha1 and recreates the
storageVersionMigration object to migrate to v2alpha1.

## Future work: HA clusters

When the masters of an HA cluster undergoes rolling upgrade, the storage
versions might be configured differently in different API servers. Storage
migration shouldn't start until the rolling upgrade is done.

Ideally, for an HA cluster undergoing rolling upgrade, the following should
happen for each resource:
* the discovery document should expose the lack of consensus by listing all
  storage version hashes supported by different API servers.
* the storage migration triggering controller should
  * add all listed storage version hashes to
    storageState.status.persistedStorageVersionHashes.
  * set storageState.status.currentStorageVersionHash to `Unknown`.
  * delete any in progress migration by deleting existing
    storageVersionMigrations.

After the API server rolling upgrade is done, the discovery document would
expose the agreed storage version hashes, and the storage migration triggering
controller will resume the work described in the [previous section][].

[previous section]:storage-migration-triggering-controller

How to develop a convergence mechanism for HA masters is beyond the scope of
this KEP. We will implement this KEP for non-HA case first. Operators of HA
clusters need to disable the triggering controller and start migration
manually, e.g., deploying the [initializer][] manually.

[initializer]:https://github.com/kubernetes-sigs/kube-storage-version-migrator/tree/master/cmd/initializer

## Future work: persisted discovery document

Today, the discovery document is held in apiserver's memory. Previous state is
lost when the apiserver reboots. Thus we need the migration triggering
controller to poll the apiserver and keep track of the stored versions. And thus
we have the limit that the storage version cannot change more than once in
the controller's polling period. If the apiserver persists the discovery
document in etcd, then we can remove the limit.
Persisting the discovery document in etcd also helps HA apiservers to agree on
the discovery document.
