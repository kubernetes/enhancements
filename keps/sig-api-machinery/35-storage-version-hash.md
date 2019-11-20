---
title: exposing hashed storage versions via the discovery API
authors:
  - "@xuchao"
owning-sig: sig-api-machinery
reviewers:
  - "@deads2k"
  - "@lavalamp"
approvers:
  - "@deads2k"
  - "@lavalamp"
creation-date: 2019-01-02
last-updated: 2019-01-04
status: provisional
---

# Exposing storage versions in opaque values via the discovery API

## Table of Contents

<!-- toc -->
- [Terms](#terms)
- [Summary](#summary)
- [Motivation](#motivation)
- [Proposal](#proposal)
  - [API changes to the discovery API](#api-changes-to-the-discovery-api)
  - [Implementation details](#implementation-details)
- [Graduation Criteria](#graduation-criteria)
- [Risks and mitigation](#risks-and-mitigation)
  - [HA masters](#ha-masters)
- [Alternatives](#alternatives)
<!-- /toc -->

## Terms

**storage versions**: Kubernetes API objects are converted to specific API
versions before stored in etcd. "Storage versions" refers to these API versions.

## Summary

We propose to expose the hashed storage versions in the discovery API.

## Motivation

We intend to use the exposed storage version hash to trigger the [storage
version migrator][].

In short, the storage version migrator detects if objects in etcd are stored in
a version different than the configured storage version. If so, the migrator
issues no-op update for the objects to migrate them to the storage version.

The storage version migrator can keep track of the versions the objects are
stored as. However, today the migrator has no way to tell what the expected
storage versions are. Thus we propose to expose this piece of information via
the discovery API.

[storage version migrator]:https://github.com/kubernetes-sigs/kube-storage-version-migrator

## Proposal

### API changes to the discovery API

We add a new field `StorageVersionHash` to the [APIResource][] type.

So far, the only valid consumer of this information is the storage version
migrator, which only needs to do equality comparisons on the storage versions.
Hence, we use the hash value instead of the plain text to avoid clients misusing
the information. The hash function needs to ensure hash value differs if the
storage versions are different.

[APIResource]:https://github.com/kubernetes/kubernetes/blob/f22334f14d92565ec3ff9d4ff2b995eae9af622a/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L881-L905

```golang
type APIResource struct {
        // The hash value of the storage version, the version this resource is
        // converted to when written to the data store. Value must be treated 
        // as opaque by clients. Only equality comparison on the value is valid.
	// This is an alpha feature and may change or be removed in the future.
        // The field is populated by the apiserver only if the
        // StorageVersionHash feature gate is enabled.
        // This field will remain optional even if it graduates. 
        // +optional
        StorageVersionHash string `json:"storageVersionHash,omitempty"`
        // These are the existing fields.
        Name string
        SingularName string
        Namespaced bool
        Group string
        Version string
        Kind string
        Verbs Verbs
        ShortNames []string
        Categories []string
}
```

### Implementation details

The hash function needs to ensure hash value differs if the storage versions are
different. SHA-256 is good enough for our purpose.

For different categories of resources,

* For built-in resources, the kube-apiserver will set the `StorageVersionHash`
when it bootstraps.

* For custom resources, the storage version is already exposed through the [CRD
spec][]. The apiextension apiserver will hash the value and sets the
`StorageVersionHash` when registers the discovery doc for the CRD.

* For aggregated resources, if the aggregated apiserver is implemented using the
generic apiserver library, the `StorageVersionHash` will be set in the same way
as the kube-apiserver. Otherwise, the aggregated apiserver is responsible
to come up with a proper `StorageVersionHash`.

* For sub-resources, the `StorageVersionHash` field will be left empty. No
sub-resource persists in the data store and thus does not require storage
version migration.

[CRD spec]:https://github.com/kubernetes/kubernetes/blob/7d8554643e2e05fda714f30fc71f34ce05514b68/staging/src/k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1/types.go#L167

## Graduation Criteria

The discovery API is read-only and the `StorageVersionHash` field is only
intended to be used by the storage version migrator, the graduation story is
simple.

The field will be alpha in 1.14, protected by a feature flag. By default the
`StorageVersionHash` is not shown in the discovery document as the feature flag
is default to `False`.

If we don't find any problem with the field, we will promote it to beta in 1.15
and GA in 1.16. Otherwise we just remove the field while keeping a tombstone
for the protobuf tag.

The above is a simplified version of Kubernetes API change [guideline][].

[guideline]:https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#alpha-field-in-existing-api-version

## Risks and mitigation

### HA masters

Kubernetes does not have a convergence mechanism for HA masters. During HA
master rolling upgrade/downgrade, depending on which apiserver handles the
request, the discovery document and the storage version may vary.

This breaks the auto-triggered storage migration. For example, the storage
version migrator gets the discovery document from an upgraded apiserver. The new
storage version for *deployment* is `apps/v1`, while the old storage version is
`apps/v1beta2`. The migrator issues no-op updates for all `deployments` to
migrate them to `apps/v1`. However, other clients might write to `deployments`
via the other to-be-upgraded apiservers and thus accidentally revert the
migration. The migrator cannot detect the reversion.

Ideally, the HA masters can detect that the masters haven't converged on the
storage versions, and manifest the disagreement in the discovery API. The
migrator waits for the masters to converge before starting migration. Before
such a convergence mechanism exists, the auto-triggered storage version
migration is **not** safe for HA masters.

To workaround, the cluster admins of HA clusters need to
* turn off the migration triggering controller, which is a standalone controller.
* manually trigger migration after the rolling upgrade/downgrade is done.

Though the `StorageVersionHash` cannot be used to automate HA cluster storage
migration, manually triggered migration can determine use this information if
the storage version for a particular resource has changed since the last run and
skip unnecessary migrations.

## Alternatives
1. We had considered triggering the storage version migrator by the change of
   the apiserver's binary version. However, the correctness of this approach
   relies on that the storage versions never change without changing the binary
   version first. This might not hold true in the future.
