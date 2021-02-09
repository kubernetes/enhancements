# Enabling clients to tell if resource endpoints serve the same set of objects

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
    - [Correctness](#correctness)
    - [Efficiency](#efficiency)
- [Goals](#goals)
- [Proposal](#proposal)
  - [API changes to the discovery API](#api-changes-to-the-discovery-api)
  - [Implementation details](#implementation-details)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Graduation Criteria](#graduation-criteria)
- [Alternatives](#alternatives)
<!-- /toc -->

## Summary

We propose to expand the discovery API to enable clients to tell if resource
endpoints (e.g., `extensions/v1beta1/replicaset` and `apps/v1/replicaset`) are
referring the same set of objects.

## Motivation

Today, some workload objects (e.g., `replicasets`) are accessible from both the
`extensions` and the `apps` endpoints. This can happen to more of Kubernetes
APIs in the future. Enabling the clients to programmatically detect aliasing
endpoints can improve both the correctness and the efficiency.

#### Correctness
* Resource quotas: although `extensions/v1beta1/replicasets` and
  `apps/v1/replicasets` refer to the same set of objects, they have separate
  resource quotas today. User can bypass the resource quota of one endpoint by
  using the other endpoint.

* [Admission webhook rules][]: after a cluster upgrade, if a
  resource becomes accessible from a new group, the new endpoint will bypass
  existing rules. For example, here is a webhook configuration that will enforce
  webhook checks on all replicasets.

  ```yaml
  rules:
    - apiGroups:
      - "extensions,apps"
      apiVersions:
      - "*"
      resources:
      - replicasets
  ```

  However, if in a future release, replicasets are accessible via
  `fancy-apps/v1` as well, requests sent to `fancy-apps/v1` will be left
  unchecked. Note that the admins of cluster upgrades are not necessarily the
  admins of the admission webhooks, so it is not always possible to coordinate
  cluster upgrades with webhook configuration upgrades.

  If the webhook controller can detect all aliasing resource endpoints, then it
  can convert `fancy-apps/v1` replicasets to the `apps/v1` and send it to the
  webhook.

[Admission webhook rules]:https://github.com/kubernetes/kubernetes/blob/18778ea4a151d5f8b346332cb2822b2b0f9d1981/staging/src/k8s.io/api/admissionregistration/v1beta1/types.go#L29

  As a side note, one would expect the [RBAC rules][] would suffer the same
  problem. However, RBAC is deny by default, so access to the new endpoint is
  denied unless admin explicitly updates the policy, which is what we want.

[RBAC rules]:https://github.com/kubernetes/kubernetes/blob/18778ea4a151d5f8b346332cb2822b2b0f9d1981/staging/src/k8s.io/api/authorization/v1/types.go#L249

#### Efficiency

* The [storage migrator][] migrates the same objects multiple times if they are
served via multiple endpoints.

[storage migrator]:https://github.com/kubernetes-sigs/kube-storage-version-migrator


## Goals

The successful mechanism should
* enable clients to tell if two resource endpoints refer to the same objects.
* prevent clients from relying on the implementation details of the mechanism.
* work for all resources, including built-in resources, custom resources, and
  aggregated resources.

## Proposal

### API changes to the discovery API

We add a new field `ResourceID` to the [APIResource][] type. We intentionally avoid
mentioning any implementation details in the field name or in the comment.

[APIResource]:https://github.com/kubernetes/kubernetes/blob/f22334f14d92565ec3ff9d4ff2b995eae9af622a/staging/src/k8s.io/apimachinery/pkg/apis/meta/v1/types.go#L881-L905

```golang
type APIResource struct {
        // An opaque token which can be used to tell if separate resources
        // refer to the same underlying objects. This allows clients to follow
        // resources that are served at multiple versions and/or groups.
        ResourceID string
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

For built-in resources, their `resourceID`s are set to `SHA256(<etcd key
prefix>)`. For example, for both `extensions/v1beta1/replicasets` and
`apps/v1/replicasets`, the etcd key prefix is `/registry/replicasets`, so their
resourceIDs are the same. Serving the hashed prefix instead of the prefix in
plain text is to encourage the clients to only test the equality of the hashed
values, instead of relying on the absolute value.

For custom resources, their `resourceID`s are also set to `SHA256(<etcd key
prefix>)`. In the current implementation, the etcd key prefix for a custom
resource is `/registry/<crd.spec.group>/<crd.spec.names.plural>`.

For aggregated resources, because their discovery doc is fully controlled
by the aggregated apiserver, the kube-apiserver has no means to validate their
`resourceID`. If the server is implemented with the generic apiserver library,
the `resourceID` will be `SHA256(<etcd key prefix>)`.

For subresources, the `resourceID` field is left empty for simplicity. Clients
can tell if two subresource endpoints refer the same objects by checking if the
`resourceID`s of the main resources are the same.

For non-persistent resources like `tokenReviews` or `subjectAccessReviews`,
though the objects are not persisted, the [forward compatibility][] motivation
still applies, e.g., admins might configure the admission webhooks to intercept
requests sent to all endpoints. Thus, the `resourceID` cannot be left empty, it
will be set to `SHA256(<the would-be etcd key prefix>)`, e.g., for
`tokenReviews`, it's `SHA256(/registry/tokenreviews)`.

[forward compatibility]:#broken-forwards-compatibility

### Risks and Mitigations

In the future, the "etcd key prefix" might not be sufficient to uniquely
identify a set of objects. We can always add more factors to the `resourceID` to
ensure their uniqueness. It does not break backwards compatibility because the
`resourceID` is opaque.

Another risk is that an aggregated apiserver accidentally reports `resourceID`
that's identical to the built-in or the custom resources, this will confuse
clients. Because the kube-apiserver has zero control over the discovery doc of
aggregated resources, it cannot do any validation to prevent this kind of error.
It will be aggregated apiserver provider's responsibility to prevent such errors.

## Graduation Criteria

The field will be alpha in 1.14, protected by a feature flag. By default the
`ResourceID` is not shown in the discovery document as the feature flag
is default to `False`.

If we don't find any problem with the field, we will promote it to beta in 1.15
and GA in 1.16. Otherwise we just remove the field while keeping a tombstone
for the protobuf tag.

The above is a simplified version of the Kubernetes API change [guideline][],
because the discovery API is read-only.

[guideline]:https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api_changes.md#alpha-field-in-existing-api-version

## Alternatives
1. Adding to the discovery API a reference to the canonical endpoint. For
   example, in the discovery API, `extensions/v1beta1/replicasets` reports
   `apps/v1/replicasets` as the canonical endpoint. This approach is similar to
   `resourceID` proposal, but because the resource names are explicitly exposed,
   clients might use the information in unintended ways.

2. Serving a list of all sets of aliasing resources via a new API. Aggregated
   apiservers make such a design complex. For example, we will need to design how
   the aggregated apiserver registers its resource aliases.

3. Hard coding UUIDs for built-in resources, instead of hashes. This doesn't
   work for CRDs.
