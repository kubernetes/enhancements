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

* [Goal](#goal)
* [API design](#api-design)
* [Storage migration triggering controller](#storage-migration-triggering-controller)
* [Implications to cluster operators](#implications-to-cluster-operators)
* [Life-cycle of a StorageState object](#life-cycle-of-a-storagestate-object)
* [Extended API proposal for HA cluster](#extended-api-proposal-for-ha-cluster)
   * [Extending the storage state API](#extending-the-storage-state-api)
   * [Extending the work flow of the triggering controller](#extending-the-work-flow-of-the-triggering-controller)
   * [Case walkthrough](#case-walkthrough)
   * [Pros and Cons](#pros-and-cons)
* [Future work: persisted discovery document](#future-work-persisted-discovery-document)

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

[previous section]:storage-migration-triggering-controller

How to develop a convergence mechanism for HA masters is beyond the scope of
this KEP. We will implement this KEP for non-HA case first. Operators of HA
clusters need to disable the triggering controller and start migration
manually, e.g., deploying the [initializer][] manually.

[initializer]:https://github.com/kubernetes-sigs/kube-storage-version-migrator/tree/master/cmd/initializer

## Extended API proposal for HA cluster

The proposed storage state and triggering controller are not guaranteed to work
during the rolling upgrade of HA masters. The issue is detailed in the previous
[KEP][]. Essentially, the migrator should abort ongoing storage version
migrations when the HA masters are rolling upgrading, and restart the migration
when the rolling upgrade is done. We propose the following changes to the
storage state API and the triggering controller to detect if there is ongoing
master rolling upgrade.

Note that this design is based on the following two assumptions, which might
not hold true in future HA deployments.
* there is a triggering controller co-locates with every apiserver that
consists the HA master.
* the triggering controller only talks with the local apiserver.

[KEP]:35-storage-version-hash.md#ha-masters

### Extending the storage state API

```golang
// Remaining the same.
type StorageState struct {
  metav1.TypeMeta
  metav1.ObjectMeta
  Spec StorageStateSpec
  Status StorageStateStatus
}

// Remaining the same.
type StorageStateSpec {
  Resource GroupResource
}

// Extended.
type StorageStateStatus {
  // Extended.
  // A list of current storage version hashes. In case of HA setup, the
  // storage version hashes exposed by all master instances are reported here.
  // In case of non-HA setup, this list should only contain one element.
  CurrentStorageVersionHashes []StorageVersionHash
  // Remaining the same.
  // +optional
  PersistedStorageVersionHashes []string
}

// The storage version hash observed by the reporter at lastHeartbeatTime.
type StorageVersionHash struct {
  // The ID of the reporting triggering contoller. This should be the same ID
  // that's used to contend for a lease, see https://github.com/kubernetes/kubernetes/blob/428a8e04d40ef01c28c66fcfd54f306a1aff9a28/staging/src/k8s.io/api/coordination/v1/types.go#L43.
  // Required.
  ReporterID string
  // The hash value of the current storage version, as shown in the discovery
  // document served by the API server local to the reporting controller.
  CurrentStorageVersionHash string
  // LastHeartbeatTime is the last time the reporting controller checks the
  // storage version hash of this resource in the discovery document and updates
  // this field.
  // +optional
  LastHeartbeatTime metav1.Time
}
``` 

### Extending the work flow of the triggering controller

Normally, HA controllers select a leader, and only the leader writes to the API
to update status. In this case, we need every triggering controller to report
its local storage version hash. Specifically, every 10 minutes, each instance of
the triggering controller 
* fetches the discovery doc from the local apiserver.
* adds the storage version hash to PersistedStorageVersionHashes if the local
  storage version hash is not present there yet.
* updates (or creates) the element in status.storageVersionhashes with the
  matching reporterID, recording the latest currentStorageVersionHash and
  lastHeartbeatTime.
* removes storageVersionHashes whose lastHeartbeatTime are more than 3 polling
  periods (30 mins) old. This garbage collection is necessary because the
  reporter ID [changes][] every time a process reboots. The garbage collection
  happens 30 mins after lastHeartbeat to give the reporter a chance to recover
  from transient errors that occur right before the reporter updating the
  status.

[changes]:https://github.com/kubernetes/kubernetes/blob/428a8e04d40ef01c28c66fcfd54f306a1aff9a28/cmd/kube-controller-manager/app/controllermanager.go#L238

Only the leading triggering controller reacts to the storageState changes. The
leading triggering controller
* deletes pending storageVersionMigration objects to cancel any ongoing
  migration, once the hashes in currentStorageVersionHashes stop being
  identical.
* creates a storageVersionMigration object to start migration, if
  * persistedStorageVersionHashes has "Unknown" or has more than one version,
  * **AND** the lastHeartbeatTime in all items of the CurrentStorageVersionHashes
  are no older than 1 polling period (10 mins)
  * **AND** the hashes in currentStorageVersionHashes have been identical in the
    past two polls. In other words, the hashes have been held identical for at
    least one polling period (10 minutes). The purpose is to give newly deployed
    triggering controller a chance to report the hash.

### Case walkthrough

Case 1. Assuming the HA cluster consists of 3 apiservers. The cluster admin
rolling upgrades them. The old storage version is v1, and becomes v2 after
upgrades. The apiserver 3 fails after upgrades, so the cluster admin rolls it
back at the 22nd minute.

| time (min) | events                                                                                                                             | apiserver 1 sv<sup>1</sup>| apiserver 2 sv | apiserver 3 sv | CurrentSV<sup>2</sup> | PersistedSV<sup>3</sup> |
|------------|------------------------------------------------------------------------------------------------------------------------------------|---------------------------|----------------|----------------|-----------------------|-------------------------|
| 0          |                                                                                                                                    | v1                        | v1             | v1             | v1, v1, v1            | v1                      |
| 3          | The cluster admin starts rolling upgrades from apiserver 1.                                                                        | v2                        | v1             | v1             | v1, v1, v1            | v1                      |
| 10         | Triggering controllers update currentSV. The leader does not start migration because of the lack of consensus on storage version.  | v2                        | v1             | v1             | v2, v1, v1            | v1, v2                  |
| 12         | Rolling upgrade continues, apiserver 2 is upgraded.                                                                                | v2                        | v2             | v1             | v2, v1, v1            | v1, v2                  |
| 18         | Rolling upgrade is done, apiserver 3 is upgraded.                                                                                  | v2                        | v2             | v2             | v2, v1, v1            | v1, v2                  |
| 20         | Triggering controllers update currentSV. The leader starts migration as apiservers have the same storage version.                  | v2                        | v2             | v2             | v2, v2, v2            | v1, v2                  |
| 22         | The cluster admin rolls back apiserver 3.                                                                                          | v2                        | v2             | v1             | v2, v2, v2            | v1, v2                  |
| 30         | The triggering controller co-locates with apiserver 3 adds v1 to currentSV and persistedSV. The leader cancels ongoing migration.  | v2                        | v2             | v1             | v2, v2, v1            | v1, v2                  |
| 34         | The cluster admin upgrades apiserver 3 again.                                                                                      | v2                        | v2             | v2             | v2, v2, v1            | v1, v2                  |
| 40         | The triggering controller co-locates with apiserver 3 sets currentSV to v2. The leader starts migration as consensus is reached.   | v2                        | v2             | v2             | v2, v2, v2            | v1, v2                  |
| 45         | The migration has completed. The leader updates persistedSV to v2.                                                                 | v2                        | v2             | v2             | v2, v2, v2            | v2                      |

1. sv: storage version.
2. CurrentSV: currentStorageVersionHashes.
3. PersistedSV: persistedStorageVersionHashes.

### Pros and Cons
Pros:
1. The API changes are limited to the storage migration API, which is only used
   by the migrator, so we have more flexibility to experiment.

Cons:
1. As mentioned earlier, the design is based on two assumptions that might not
   hold in the future
    * there is a triggering controller co-locates with every apiserver that
      consists the HA master.
    * the triggering controller only talks with the local apiserver.

2. Reaching consensus on apiserver status is a general issue in Kubernetes HA
   setup. The proposed solution only solves the consensus issue for the storage
   version hash field. A generic solution would be preferred.

## Future work: persisted discovery document

Today, the discovery document is held in apiserver's memory. Previous state is
lost when the apiserver reboots. Thus we need the migration triggering
controller to poll the apiserver and keep track of the stored versions. And thus
we have the limit that the storage version cannot change more than once in
the controller's polling period. If the apiserver persists the discovery
document in etcd, then we can remove the limit.
Persisting the discovery document in etcd also helps HA apiservers to agree on
the discovery document.
