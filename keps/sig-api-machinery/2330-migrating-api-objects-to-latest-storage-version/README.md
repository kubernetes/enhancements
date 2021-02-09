# Migrating API objects to latest storage version

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
- [Proposal](#proposal)
  - [Alpha workflow](#alpha-workflow)
  - [Alpha API](#alpha-api)
  - [Failure recovery](#failure-recovery)
  - [Beta workflow - Automation](#beta-workflow---automation)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Beta Graduation Criteria](#beta-graduation-criteria)
- [Alternatives](#alternatives)
  - [update-storage-objects.sh](#update-storage-objectssh)
<!-- /toc -->

## Summary

We propose a solution to migrate the stored API objects in Kubernetes clusters.
In 2018 Q4, we will deliver a tool of alpha quality. The tool extends and
improves based on the [oc adm migrate storage][] command.

[oc adm migrate storage]:https://www.mankier.com/1/oc-adm-migrate-storage

## Motivation

"Today it is possible to create API objects (e.g., HPAs) in one version of
Kubernetes, go through multiple upgrade cycles without touching those objects,
and eventually arrive at a version of Kubernetes that can’t interpret the stored
resource and crashes. See k8s.io/pr/52185."[1][]. We propose a solution to the
problem.

[1]:https://docs.google.com/document/d/1eoS1K40HLMl4zUyw5pnC05dEF3mzFLp5TPEEt4PFvsM

### Goals

A successful storage version migration tool must:
* work for Kubernetes built-in APIs, custom resources (CR), and aggregated APIs.
* do not add burden to cluster administrators or Kubernetes distributions.
* only cause insignificant load to apiservers. For example, if the master has
  10GB memory, the migration tool should generate less than 10 qps of single
  object operations(TODO: measure the memory consumption of PUT operations;
  study how well the default 10 Mbps bandwidth limit in the oc command work).
* work for big clusters that have ~10^6 instances of some resource types.
* make progress in flaky environment, e.g., flaky apiservers, or the migration
  process get preempted.
* allow system administrators to track the migration progress.

We will deliver a vendor-agnostic solution to automatically detect and migrate
resources when the default storage version has changed.

## Proposal

### Alpha workflow

At the alpha stage, the migrator needs to be manually launched, and does not
handle custom resources or aggregated resources.

After all the kube-apiservers are at the desired version, the cluster
administrator runs `kubectl apply -f migrator-initializer-<k8s-versio>.yaml`.
The apply command
* creates a *kube-storage-migration* namespace
* creates a *storage-migrator* service account
* creates a *system:storage-migrator* cluster role that can *get*, *list*, and
  *update* all resources, and in addition, *create* and *delete* CRDs.
* creates a cluster role binding to bind the created service account with the
  cluster role
* creates a **migrator-initializer** job running with the
  *storage-migrator* service account.

The **migrator-initializer** job
* deletes any existing deployment of **kube-migrator controller**
* creates a **kube-migrator controller** deployment running with the
  *storage-migrator* service account.
* generates a comprehensive list of resource types via the discovery API
* discovers all custom resources via listing CRDs
* discovers all aggregated resources via listing all `apiservices` that have
  `.spec.service != null`
* removes the custom resources and aggregated resources from the comprehensive
  resource list. The list now only contains Kubernetes built-in resources.
* removes resources that share the same storage. At the alpha stage, the
  information is hard-coded, like in this [list][].
* creates `migration` CRD (see the [API section][] for the schema) if it does
  not exist.
* creates `migration` CRs for all remaining resources in the list. The
  `ownerReferences` of the `migration` objects are set to the **kube-migrator
  controller** deployment. Thus, the old `migration`s are deleted with the old
  deployment in the first step.

The control loop of **kube-migrator controller** does the following:
* runs a reflector to watch for the instances of the `migration` CR. The list
  function used to construct the reflector sorts the `migration`s so that the
  *Running* `migration` will be processed first.
* syncs one `migration` at a time to avoid overloading the apiserver,
  * if `migration.status` is nil, or `migration.status.conditions` shows
    *Running*, it creates a **migration worker** goroutine to migrate the
    resource type.
  * adds the *Running* condition to `migration.status.conditions`.
  * waits until the **migration worker** goroutine finishes, adds either the
    *Succeeded* or *Failed* condition to `migration.status.conditions` and sets
    the *Running* condition to false.

The **migration worker** runs the equivalence of `oc adm migrate storage
--include=<resource type>` to migrate a resource type. The **migration worker**
uses API chunking to retrieve partial lists of a resource type and thus can
migrate a small chunk at a time. It stores the [continue token] in the owner
`migration.spec.continueToken`. With the inconsistent continue token
introduced in [#67284][], the **migration worker** does not need to worry about
expired continue token.

[list]:https://github.com/openshift/origin/blob/2a8633598ef0dcfa4589d1e9e944447373ac00d7/pkg/oc/cli/admin/migrate/storage/storage.go#L120-L184
[#67284]:https://github.com/kubernetes/kubernetes/pull/67284
[API section]:#alpha-api

The cluster admin can run the `kubectl wait --for=condition=Succeeded
migrations` to wait for all migrations to succeed.

Users can run `kubectl create` to create `migration`s to request migrating
custom resources and aggregated resources.

### Alpha API

We introduce the `storageVersionMigration` API to record the intention and the
progress of a migration. Throughout this doc, we abbreviated it as `migration`
for simplicity. The API will be a CRD defined in the `migration.k8s.io` group.

Read the [workflow section][] to understand how the API is used.

```golang
type StorageVersionMigration struct {
  metav1.TypeMeta
  // For readers of this KEP, metadata.generateName will be "<resource>.<group>"
  // of the resource being migrated.
  metav1.ObjectMeta
  Spec StorageVersionMigrationSpec
  Status StorageVersionMigrationStatus
}

// Note that the spec only contains an immutable field in the alpha version. To
// request another round of migration for the resource, clients need to create
// another `migration` CR.
type StorageVersionMigrationSpec {
  // Resource is the resource that is being migrated. The migrator sends
  // requests to the endpoint tied to the Resource.
  // Immutable.
  Resource GroupVersionResource
  // ContinueToken is the token to use in the list options to get the next chunk
  // of objects to migrate. When the .status.conditions indicates the
  // migration is  "Running", users can use this token to check the progress of
  // the migration.
  // +optional
  ContinueToken string
}

type MigrationConditionType string

const (
  // MigrationRunning indicates that a migrator job is running.
  MigrationRunning MigrationConditionType = "Running"
  // MigrationSucceed indicates that the migration has completed successfully.
  MigrationSucceeded MigrationConditionType = "Succeeded"
  // MigrationFailed indicates that the migration has failed.
  MigrationFailed MigrationConditionType = "Failed"
)

type MigrationCondition struct {
	// Type of the condition
	Type MigrationConditionType
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus
	// The last time this condition was updated.
	LastUpdateTime metav1.Time
	// The reason for the condition's last transition.
	Reason string
	// A human readable message indicating details about the transition.
	Message string
}

type StorageVersionMigrationStatus {
  // Conditions represents the latest available observations of the migration's
  // current state.
  Conditions []MigrationCondition
}
```

[continue token]:https://github.com/kubernetes/kubernetes/blob/972e1549776955456d9808b619d136ee95ebb388/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L82
[workflow section]:#alpha-workflow

### Failure recovery

As stated in the goals section, the migration has to make progress even if the
environment is flaky. This section describes how the migrator recovers from
failure.

Kubernetes **replicaset controller** restarts the **migration controller** `pod`
if it fails. Because the migration states, including the continue token, are
  stored in the `migration` object, the **migration controller** can resume from
  where it left off.

[workflow section]:#alpha-workflow

### Beta workflow - Automation

It is a beta goal to automate the migration workflow. That is, migration does
not need to be triggered manually by cluster admins, or by custom control loops
of Kubernetes distributions.

The [storage version hash][] is added to the Kubernetes discovery document as an
alpha feature in 1.14. A [triggering controller][] is added to poll the discovery
document, and creates migrations when the storage version hash of a resource
changes. See [KEP][] for the details on the automated migration workflow.

[storage version hash]:https://github.com/kubernetes/kubernetes/pull/73191
[triggering controller]:https://github.com/kubernetes-sigs/kube-storage-version-migrator/pull/21
[KEP]:storage-migration-auto-trigger.md

### Risks and Mitigations

The migration process does not change the objects, so it will not pollute
existing data.

If the rate limiting is not tuned well, the migration can overload the
apiserver. Users can delete the migration controller and the migration
jobs to mitigate.

Before upgrading or downgrading the cluster, the cluster administrator must run
`kubectl wait --for=condition=Succeeded migrations` to make sure all
migrations have completed. Otherwise the apiserver can crash, because it cannot
interpret the serialized data in etcd. To mitigate, the cluster administrator
can rollback the apiserver to the old version, and wait for the migration to
complete.

With the newly introduced [storageStates API][], the cluster administrator can
fast upgrade/downgrade as long as the new apiserver binaries understand all
versions recorded in storageState.status.persistedStorageVersionHashes.

[storageStates API]:storage-migration-auto-trigger.md#api-design

## Beta Graduation Criteria

* Visibility
  * metrics for the number of migrated objects per resource. This metric also
    indirectly manifests the speed migration per resource.
  * metrics for pending, succeeded, and failed migrations. This metric also
    indirectly manifests the frequency of migrations per resource.

* End-to-end testing
  * testing migration for CRDs
  * chaos testing the migrator with injected network errors, and injected
    crashes of both the migrator and the apiserver.
  * stress testing the migrator with large lists of to-be-migrated objects
  * optional: integrating the migrator into the Kubernetes upgrade tests,
    verifying all objects are readable after the cluster upgrade

* Deployment
  * example manifests for installation (manifests are currently kept at
    https://github.com/kubernetes-sigs/kube-storage-version-migrator/tree/master/manifests).

## Alternatives

### update-storage-objects.sh

The Kubernetes repo has an update-storage-objects.sh script. It is not
production ready: no rate limiting, hard-coded resource types, no persisted
migration states. We will delete it, leaving a breadcrumb for any users to
follow to the new tool.
