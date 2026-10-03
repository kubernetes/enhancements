# KEP-6456: Work API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Terminology](#terminology)
  - [User Stories](#user-stories)
    - [Deploy a workload to a managed cluster](#deploy-a-workload-to-a-managed-cluster)
    - [Track the status of workload in another cluster](#track-the-status-of-workload-in-another-cluster)
    - [Garbage collect the workload](#garbage-collect-the-workload)
    - [Garbage collect the workload when the hub is unreachable](#garbage-collect-the-workload-when-the-hub-is-unreachable)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [Resources in the work](#resources-in-the-work)
    - [Push/Pull Model](#pushpull-model)
    - [Working with higher primitive](#working-with-higher-primitive)
- [Design Details](#design-details)
  - [Work](#work)
    - [API Specification](#api-specification)
    - [Conditions](#conditions)
  - [AppliedWork](#appliedwork)
    - [API Specification](#api-specification-1)
    - [Ownership and garbage collection](#ownership-and-garbage-collection)
    - [Naming](#naming)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This proposal is for a minimalistic start to a new `Work` API which is
intended to group a set of k8s API resources to be applied to one remote
cluster together as a concept of `work` or `workload`. It could be simple
enough that it does not relate to cluster registration mechanisms or any
workload scheduling on multiple clusters.

Alongside `Work`, the API defines `AppliedWork`, a cluster scoped resource on
the managed cluster that records what a `Work` has applied there and anchors
garbage collection of those resources. The API and a reference controller are
developed in
[kubernetes-sigs/work-api](https://github.com/kubernetes-sigs/work-api).

## Motivation

There are already several different techniques to distribute workload on
multiple kubernetes clusters, including `kubefed v1`, `kubefed v2`, and
`gitops`. All these techniques has some common patterns for the function to
deploy workload resource manifests to one or multiple clusters.

1. A single source of truth which could be git, cloud storage, kube-apiserver
   or rpc server.
2. A control loop to apply resources manifests from a single source of truth
   to a cluster, return apply results and track resource status.
3. A way to decide which clusters the resource manifests should apply to.
    - Specific levels of redundancy.
    - Specific kinds of geographic/topological placement or spread.
    - Specific resource availability.

In addition, the control loop will also face the problem of garbage collecting
resources when there is no intent to deploy them on a cluster.

All of the above motivates the notion of work api which:

1. Allow developers to easily integrate with other sources of truth, e.g. git,
   another kube-apiserver. And the control loop of work api could provide a
   generic way to apply resource manifests to a certain cluster.
2. Easy to integrate with other placement primitives.
3. Track which clusters a particular workload is deployed to.
4. Track the deployed resource on a cluster so the control loop could garbage
   collect these resources.

### Goals

- API Specification of Work.
- API Specification of AppliedWork, an anchor on the managed cluster that
  records what a Work has applied there and eases garbage collection.
- A reference implementation of control loops for Work API.

### Non-Goals

- How workload is scheduled among multiple clusters.
- Handling applied resource status feedback through AppliedWork. Status is
  reported on the Work object on the hub.

## Proposal

### Terminology

- **Work Hub**: A place that work API resides. It could be a k8s cluster
  playing the role as the management plane for other k8s clusters. Or it could
  also be an RPC server or a cloud API depending on the detailed
  implementation. Users create work API resources on the work hub. In the rest
  of the doc, we assume the implementation of work hub is based on k8s cluster
  for simplicity.
- **Managed Cluster**: A k8s cluster managed by the work hub. The resources
  defined in the work API are applied on the managed cluster. It was also
  referred to "Spoke" or "Spoke Cluster", though we use "Managed Cluster" in
  this document.
- **Work Controller**: a controller that reconciles the work object on hub,
  and applies resources defined in work to the managed cluster. When it runs
  on the managed cluster it is also referred to as the work agent.
- **AppliedWork**: a cluster scoped object on the managed cluster, created and
  maintained by the work controller, that links back to one Work on the hub
  and records the resources applied for it.

### User Stories

#### Deploy a workload to a managed cluster

I have 3 clusters. One is the work hub, the other two are managed clusters. I
want to declare the workload (deployment/configmap/service etc) on the work
hub, and ensure the workload will be deployed on the desired managed cluster.
I want to ensure that when I update the workload declaration, the real
deployed workload on the managed cluster reflects the update.

#### Track the status of workload in another cluster

After I have declared the workload on the work hub, I want to track whether
the workload has been successfully deployed on the managed cluster, and the
workload is running normally.

#### Garbage collect the workload

When I do not want to deploy the workload on a managed cluster, I can clean
the workload by deleting the api on the work hub.

#### Garbage collect the workload when the hub is unreachable

As the administrator of a managed cluster, I want to see what a work agent
has applied to my cluster, and I want to be able to remove those resources
even when the agent has been removed or the hub is no longer reachable.

### Notes/Constraints/Caveats

We propose a new CRD called `Work` to represent a list of api resources to be
deployed on a cluster. `Work` is created on the work hub, and resides in the
namespace that the work controller is authorized to access. Creation of a
`Work` on the work hub indicates that resources defined in `Work` will be
applied on a certain managed cluster. Update of a `Work` will trigger the
resource update on the managed cluster, and deletion of a `Work` will recycle
the resources on the managed cluster.

If there are multiple managed clusters, multiple work controllers will be
running that monitor the `Work` API in same or different namespaces in the
work hub. It is possible that multiple work controllers watch `Works` in one
namespace on a work hub and deploy the resources on multiple clusters. It is
also possible that multiple work controllers watch `Works` in different
namespaces on the work hub, so a Work created in one namespace triggers the
resource deployment on a certain cluster.

#### Resources in the work

The resources in kubernetes could be classified to several categories:

- Workload related resources: deployments/statefulset, configmaps, ns-scoped
  custom resources etc.
- Clusterwide configuration resources: apiservices, CRDs, storageclasses
- Credentials: secrets

`Work` api should be mainly used for workload related resources. Secrets
should not be declared in `work` apis, other techniques (e.g. vault) should be
considered as a more secure way to transmit secrets among clusters.

Whether a Work may carry cluster scoped resources is not settled by this
KEP. A Work carrying custom resources cannot be self-contained unless the CRD
can be carried too. The AppliedWork design accounts for cluster scoped
resources in its status (see [AppliedWork](#appliedwork)), and whether a given
work controller accepts them in a Work is left to that implementation.

#### Push/Pull Model

There have been discussions in the community on whether the workload
distribution in multicluster should use a push or pull model.

Push model means that a controller on the hub watches APIs defining workload
and "PUSH" the resource manifest to the managed cluster. There are some
limitations in the push model:

- It requires apiserver of each managed cluster must be accessed by the work
  hub where the controller is running. This could be a hard requirement since
  some managed clusters may hide behind firewalls and do not have a public
  accessible IP. Exposing apiserver of all the managed clusters also enlarges
  the surface to be attacked.
- It requires credentials of managed clusters with sufficient permission to be
  put on the work hub. The credential has to be passed in an out of band
  secure way.
- Having a centralized controller to distribute workload to many clusters
  could have scalability limitations.

Pull model means that an agent running in the managed cluster watches APIs
defined on hub, fetches them and applies locally on the managed cluster.
Compared to the push model, the API exposure surface is reduced since only
apiserver of the work hub needs to be publicly accessible. The credential for
the agent to talk to the work hub can have very limited permission only on
certain APIs.

`Work` API itself is not constrained to be used only on push or pull mode. The
work controller could reside in a managed cluster that "PULLS" the API and
applies locally on the managed cluster. It could also reside in the work hub
that watches the Work API and "PUSHES" the workload to the managed cluster.

#### Working with higher primitive

`Work` represents a workload to be deployed in a target namespace on a single
managed cluster. Which cluster the work is to be deployed and how the `Work`
is scheduled to a certain cluster is not defined in the `Work` API. A higher
primitive could be used to generate `Work` based on a scheduling decision and
place the workload on a managed cluster. The higher primitive must coordinate
with clusterset together to decide which clusters the `Work` should place to.

## Design Details

### Work

`Work` is namespace scoped and lives on the work hub.

#### API Specification

```go
// Work is the Schema for the works API
type Work struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// spec defines the workload of a work.
	// +optional
	Spec WorkSpec `json:"spec,omitempty"`
	// status defines the status of each applied manifest on the spoke cluster.
	Status WorkStatus `json:"status,omitempty"`
}

// WorkSpec defines the desired state of Work
type WorkSpec struct {
	// Workload represents the manifest workload to be deployed on spoke cluster
	Workload WorkloadTemplate `json:"workload,omitempty"`
}

// WorkloadTemplate represents the manifest workload to be deployed on spoke cluster
type WorkloadTemplate struct {
	// Manifests represents a list of kubernetes resources to be deployed on the spoke cluster.
	// +optional
	Manifests []Manifest `json:"manifests,omitempty"`
}

// Manifest represents a resource to be deployed on spoke cluster
type Manifest struct {
	// +kubebuilder:validation:EmbeddedResource
	// +kubebuilder:pruning:PreserveUnknownFields
	runtime.RawExtension `json:",inline"`
}

// WorkStatus defines the observed state of Work
type WorkStatus struct {
	// Conditions contains the different condition statuses for this work.
	// Valid condition types are:
	// 1. Applied represents workload in Work is applied successfully on the spoke cluster.
	// 2. Progressing represents workload in Work in the transitioning from one state to another the on the spoke cluster.
	// 3. Available represents workload in Work exists on the spoke cluster.
	// 4. Degraded represents the current state of workload does not match the desired
	// state for a certain period.
	Conditions []metav1.Condition `json:"conditions"`

	// ManifestConditions represents the conditions of each resource in work deployed on
	// spoke cluster.
	// +optional
	ManifestConditions []ManifestCondition `json:"manifestConditions,omitempty"`
}

// ResourceIdentifier provides the identifiers needed to interact with any arbitrary object.
type ResourceIdentifier struct {
	// Ordinal represents an index in manifests list, so the condition can still be linked
	// to a manifest even though manifest cannot be parsed successfully.
	Ordinal int `json:"ordinal"`

	// Group is the group of the resource.
	Group string `json:"group,omitempty"`

	// Version is the version of the resource.
	Version string `json:"version,omitempty"`

	// Kind is the kind of the resource.
	Kind string `json:"kind,omitempty"`

	// Resource is the resource type of the resource
	Resource string `json:"resource,omitempty"`

	// Namespace is the namespace of the resource, the resource is cluster scoped if the value
	// is empty
	Namespace string `json:"namespace,omitempty"`

	// Name is the name of the resource
	Name string `json:"name,omitempty"`
}

// ManifestCondition represents the conditions of the resources deployed on
// spoke cluster
type ManifestCondition struct {
	// resourceId represents a identity of a resource linking to manifests in spec.
	// +required
	Identifier ResourceIdentifier `json:"identifier,omitempty"`

	// Conditions represents the conditions of this resource on spoke cluster
	// +required
	Conditions []metav1.Condition `json:"conditions"`
}
```

An example of the `Work` API looks like:

```yaml
apiVersion: multicluster.x-k8s.io/v1alpha1
kind: Work
metadata:
  name: work-sample
  namespace: cluster
spec:
  workload:
    manifests:
    - apiVersion: v1
      kind: ConfigMap
      metadata:
        name: cm
        namespace: default
      data:
        ui.properties: |
          color=purple
```

which declares the intention to apply a configmap to a certain managed
cluster.

#### Conditions

Manifest conditions represent the status conditions of a certain manifest to
be applied on a managed cluster. The structure of manifest conditions will
include an identifier to link to a resources defined in work.spec field, and a
list of conditions showing the current status of the resource being applied.
Condition types for manifest conditions includes:

- *Applied* indicates that the manifest with the identifier is applied
  successfully in the managed cluster.
- *Available* indicates that the resource relating to the manifest exists in
  the managed cluster.
- *Degraded* indicates that the manifest applied on the managed cluster does
  not match the desired status. Example is the running replica in deployment
  does not fit the desired replica in deployment spec.

In addition to track status of each manifest with manifest condition, a `Work`
should have summarized conditions based on manifest conditions. An example of
`Work` and manifest conditions together as below:

```yaml
conditions:
- lastTransitionTime: "2020-07-02T03:16:26Z"
  message: Apply manifest work complete
  reason: AppliedWorkComplete
  status: "True"
  type: Applied
manifestConditions:
- conditions:
  - lastTransitionTime: "2020-07-02T03:16:26Z"
    message: Apply manifest complete
    reason: AppliedManifestComplete
    status: "True"
    type: Applied
  identifier:
    group: ""
    kind: ConfigMap
    name: cm1
    namespace: default
    ordinal: 0
    resource: configmaps
    version: v1
```

### AppliedWork

`AppliedWork` is a cluster scoped resource on the managed cluster that
represents a workload and/or cluster wide configurations that are applied
there. It links to a Work on the hub and records the resources deployed in
the managed cluster.

The AppliedWork CR is an anchor on the managed cluster to ease garbage
collection of the applied workload and/or the applied cluster scoped
resources. Each applied workload resource has a resource identifier stored in
the AppliedWork status. Additionally, each namespace scoped applied workload
resource has an owner reference pointing at the AppliedWork CR.

By leveraging the AppliedWork API:

- A view is provided on the managed cluster of the work and configurations
  that have been applied.
- When the AppliedWork CR is deleted, all related applied workload resources
  are cascade deleted.

The AppliedWork API is not meant for external use. It is written by the work
controller, not by users, and exists to ease garbage collection of applied
workload on the managed cluster. Reporting the status of applied resources
back to the hub remains the job of the Work status.

#### API Specification

```go
// AppliedWork represents an applied work on managed cluster that is placed
// on a managed cluster. An appliedwork links to a work on a hub recording resources
// deployed in the managed cluster.
// When the agent is removed from managed cluster, cluster-admin on managed cluster
// can delete appliedwork to remove resources deployed by the agent.
// The name of the appliedwork must be unique since the appliedwork is cluster scoped.
//
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
type AppliedWork struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec represents the desired configuration of AppliedWork.
	// +required
	Spec AppliedWorkSpec `json:"spec"`

	// Status represents the current status of AppliedWork.
	// +optional
	Status AppliedWorkStatus `json:"status,omitempty"`
}

// AppliedWorkSpec represents the desired configuration of AppliedWork
type AppliedWorkSpec struct {
	// WorkName represents the name of the related work on the hub.
	// +required
	WorkName string `json:"workName"`

	// WorkNamespace represents the namespace of the related work on the hub.
	// +required
	WorkNamespace string `json:"workNamespace"`
}

// AppliedWorkStatus represents the current status of AppliedWork
type AppliedWorkStatus struct {
	// AppliedResources represents a list of resources defined within the Work that are applied.
	// Only resources with valid GroupVersionResource, namespace, and name are suitable.
	// An item in this slice is deleted when there is no mapped manifest in Work.Spec or by finalizer.
	// The resource relating to the item will also be removed from managed cluster.
	// The deleted resource may still be present until the finalizers for that resource are finished.
	// However, the resource will not be undeleted, so it can be removed from this list and eventual consistency is preserved.
	// +optional
	AppliedResources []AppliedResourceMeta `json:"appliedResources,omitempty"`
}

// AppliedResourceMeta represents the group, version, resource, name and namespace of a resource.
// Since these resources have been created, they must have valid group, version, resource, namespace, and name.
type AppliedResourceMeta struct {
	ResourceIdentifier `json:",inline"`

	// UID is set on successful deletion of the Kubernetes resource by controller. The
	// resource might be still visible on the managed cluster after this field is set.
	// It is not directly settable by a client.
	// +optional
	UID types.UID `json:"uid,omitempty"`
}
```

`AppliedResourceMeta` reuses the `ResourceIdentifier` from the Work status,
so an entry in `appliedResources` can be matched to the manifest ordinal and
the manifest condition on the hub side.

An example of `AppliedWork`:

```yaml
apiVersion: multicluster.x-k8s.io/v1alpha1
kind: AppliedWork
metadata:
  name: applied-work-01
spec:
  workName: work-01
  workNamespace: cluster1
status:
  appliedResources:
  - group: apps
    name: nginx-deployment
    namespace: default
    resource: deployments
    uid: fcfe11e4-c658-4c6e-9985-d98b7a059d14
    version: v1
```

#### Ownership and garbage collection

If the workload contains namespace scoped resources, they have owner
references pointing to their associated AppliedWork. For example:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
  namespace: default
  ownerReferences:
  - apiVersion: multicluster.x-k8s.io/v1alpha1
    kind: AppliedWork
    name: applied-work-01
    uid: 9664526d-c5ec-4f35-9f89-0b6411452385
```

Because AppliedWork is cluster scoped, it is a valid owner for both namespace
scoped and cluster scoped resources. A namespace scoped AppliedWork could not
own cluster scoped resources, and cross namespace owner references are not
permitted, which is what drove the scope decision.

Deletion then works at two levels:

- Normal path: the Work is deleted on the hub. The work controller holds a
  finalizer on the Work, removes the resources it applied on the managed
  cluster (including the AppliedWork), and then removes the finalizer.
- Recovery path: the work controller is gone or cannot reach the hub. A
  cluster-admin on the managed cluster deletes the AppliedWork, and the
  namespace scoped resources are removed by the Kubernetes garbage collector
  through their owner references. Cluster scoped resources are listed in
  `status.appliedResources` so they can be found and removed as well.

#### Naming

An AppliedWork name must be unique on the managed cluster since the resource
is cluster scoped. `spec.workName` and `spec.workNamespace` together identify
the originating Work on the hub. The namespace is included because the hub
namespace a work controller watches is not necessarily the same as the
managed cluster's name, and a managed cluster may be registered with more
than one hub. The naming scheme that guarantees uniqueness across hubs is
left to the work controller implementation. Open Cluster Management, for
example, prefixes the name with a hash of the hub API server URL.

### Test Plan

- [x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- Unit tests cover the functionality of the controllers.
- Unit tests cover the API types and generated client.

##### e2e tests

E2E tests use kind to create a hub and a managed cluster.

- Creating a work resource on hub will trigger the resource creation on a
  managed cluster
- Updating a work resource on hub will trigger the resource update on a
  managed cluster
- Deleting a work resource on hub will trigger the resource deletion on a
  managed cluster
- Create/update/delete an AppliedWork
- Create/update/delete a Work on the hub cluster and verify the AppliedWork on
  the managed cluster

### Graduation Criteria

#### Alpha

- The `Work` and `AppliedWork` APIs are reviewed and accepted.
- CRD definitions and generated clients are published.
- A reference controller implements the Work apply loop.

#### Beta

- E2E tests exists
- GA graduation criteria defined.
- At least one reference implementation using controller.
- At least one component uses the AppliedWork API.
- The reference controller implements the full set of Work conditions and the
  AppliedWork based garbage collection described in this KEP.

#### GA

- Scalability/performance testing.

### Upgrade / Downgrade Strategy

The `Work` CRD is installed on the hub and the `AppliedWork` CRD is installed
on the managed cluster. Adding `AppliedWork` to an existing deployment
requires installing the CRD on the managed cluster and upgrading the work
controller there. Existing `Work` objects are unaffected.

### Version Skew Strategy

There is a single `v1alpha1` version of both types. A work controller must be
built against a version of the API that includes `AppliedWork` in order to
create it; an older controller continues to work against the `Work` CRD
alone.

## Implementation History

- 2021-02-19: KEP created with status `provisional`.
- 2023-02-15: `AppliedWork` API added
  ([kubernetes-sigs/work-api#29](https://github.com/kubernetes-sigs/work-api/pull/29)).
