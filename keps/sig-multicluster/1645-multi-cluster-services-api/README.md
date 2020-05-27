<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up.  KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please ensure to complete all
  fields in that template.  One of the fields asks for a link to the KEP.  You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "title", "authors", "owning-sig",
  "status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary", and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG that are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly.  The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved.  Any KEP
marked as a `provisional` is a working document and subject to change.  You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused.  If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement", for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example.  If there are
new details that belong in the KEP, edit the KEP.  Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable` or significant changes once
it is marked `implementable` must be approved by each of the KEP approvers.
If any of those approvers is no longer appropriate than changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross cutting KEPs).
-->
# KEP-1645: Multi-Cluster Services API

<!--
This is the title of your KEP.  Keep it short, simple, and descriptive.  A good
title can help communicate what the KEP is and should be considered as part of
any review.
-->

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
    - [Terminology](#terminology)
  - [User Stories](#user-stories)
    - [Different Services Each Deployed to Separate Cluster](#different-services-each-deployed-to-separate-cluster)
    - [Single Service Deployed to Multiple Clusters](#single-service-deployed-to-multiple-clusters)
  - [Constraints](#constraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Exporting Services](#exporting-services)
    - [Restricting Exports](#restricting-exports)
  - [Importing Services](#importing-services)
  - [Supercluster Service Behavior Expectations](#supercluster-service-behavior-expectations)
    - [Service Types](#service-types)
    - [SuperclusterIP](#superclusterip)
    - [DNS](#dns)
    - [EndpointSlice](#endpointslice)
    - [Endpoint TTL](#endpoint-ttl)
- [Constraints and Conflict Resolution](#constraints-and-conflict-resolution)
  - [Global Properties](#global-properties)
    - [Service Port](#service-port)
  - [Cluster Level Properties](#cluster-level-properties)
    - [Session Affinity](#session-affinity)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [<code>ObjectReference</code> in <code>ServiceExport.Spec</code> to directly map to a Service](#-in--to-directly-map-to-a-service)
  - [Export services via label selector](#export-services-via-label-selector)
  - [Export via annotation](#export-via-annotation)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high quality user-focused
documentation such as release notes or a development roadmap.  It should be
possible to collect this information before implementation begins in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself.  KEP editors, SIG Docs, and SIG PM
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.
-->
There is currently no standard way to connect or even think about Kubernetes
services beyond the cluster boundary, but we increasingly see users deploy
applications across multiple clusters designed to work in concert. This KEP
proposes a new API to extend the service concept across multiple clusters. It
aims for minimal additional configuration, making multi-cluster services as easy
to use as in-cluster services, and leaves room for multiple implementations.

*Converted from this [original proposal doc](http://bit.ly/k8s-mc-svc-api-proposal).*


## Motivation

<!--
This section is for explicitly listing the motivation, goals and non-goals of
this KEP.  Describe why the change is important and the benefits to users.  The
motivation section can optionally provide links to [experience reports][] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->
There are [many
reasons](http://bit.ly/k8s-multicluster-conversation-starter-doc) why a K8s user
may want to split their deployments across multiple clusters, but still retain
mutual dependencies between workloads running in those clusters. Today the
cluster is a hard boundary, and a service is opaque to a remote K8s consumer
that would otherwise be able to make use of metadata (e.g. endpoint toplogy) to
better direct traffic. To support failover or temporarily during migration,
users may want to consume services spread across clusters, but today that
requires non-trivial bespoke solutions.

The Multi-Cluster Services API aims to fix these problems.

### Goals

<!--
List the specific goals of the KEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->
- Define a minimal API to support service discovery and consumption across clusters.
  - Consume a service in another cluster.
  - Consume a service deployed in multiple clusters as a single service.
- When a service is consumed from another cluster its behavior should be
  predictable and consistent with how it would be consumed within its own
  cluster.
- Allow gradual rollout of changes in a multi-cluster environment.
- Create building blocks for multi-cluster tooling.
- Support multiple implementations.
- Leave room for future extension and new use cases.

### Non-Goals

<!--
What is out of scope for this KEP?  Listing non-goals helps to focus discussion
and make progress.
-->
- Define specific implementation details beyond general API behavior.
- Change behavior of single cluster services in any way.
- Define what NetworkPolicy means for multi-cluster services.
- Solve mechanics of multi-cluster service orchestration.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation.  The "Design Details" section below is for the real
nitty-gritty.
-->
#### Terminology

- **supercluster** - A placeholder name for a group of clusters with a high
  degree of mutual trust and shared ownership that share services amongst
  themselves. Membership in a supercluster is symmetric and transitive. The set
  of member clusters are mutually aware, and agree about their collective
  association. Within a supercluster, [namespace sameness] applies and all
  namespaces with a given name are considered to be the same namespace.
- **mcs-controller** - A controller that syncs services across clusters and
  makes them available for multi-cluster service discovery and connectivity.
  There may be multiple implementations, this doc describes expected common
  behavior. The controller may be a single controller, multiple decentralized
  controllers, or a human using kubectl to create resources. This document aims
  to support any implementation that fulfills the behavioral expectations of
  this API.
- **cluster name** - A unique name or identifier for the cluster, scoped to the
  implementation's cluster registry. We do not attempt to define the registry.
  Each cluster must have a name that can uniquely identify it within the
  supercluster. A cluster name must be a valid [RFC
  1123](https://tools.ietf.org/html/rfc1123) DNS label.

[namespace sameness]: https://github.com/kubernetes/community/blob/master/sig-multicluster/namespace-sameness-position-statement.md

We propose a new CRD called `ServiceExport`, used to specify which services
should be exposed across all clusters in the supercluster. `ServiceExports` must
be created in each cluster that the underlying `Service` resides in. Creation of
a `ServiceExport` in a cluster will signify that the `Service` with the same
name and namespace as the export should be visible to other clusters in the
supercluster.

Another CRD called `ServiceImport` will be introduced to act as the in-cluster
representation of a multi-cluster service in each importing cluster. This is
analogous to the traditional `Service` type in Kubernetes. Importing clusters
will have a coresponding `ServiceImport` for each uniquely named `Service` that
has been exported within the supercluster, referenced by namespaced name.
`ServiceImport` resources will be managed by the MCS implementation's
mcs-controller.

If multiple clusters export a `Service` with the same namespaced name, they will
be recognized as a single combined service. For example, if 5 clusters export
`my-svc.my-ns`, each cluster will have one `ServiceImport` named `my-svc` in the
`my-ns` namespace and it will be associated with endpoints from all exporting
clusters. Properties of the `ServiceImport` (e.g. ports, topology) will be
derived from a merger of component `Service` properties.

Existing implementations of Kubernetes Service API (e.g. kube-proxy) can be
extended to present `ServiceImports` alongside traditional `Services`.


### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system.  The goal here is to make this feel real for users without getting
bogged down.
-->

#### Different Services Each Deployed to Separate Cluster

I have 2 clusters, each running different services managed by different teams,
where services from one team depend on services from the other team. I want to
ensure that a service from one team can discover a service from the other team
(via DNS resolving to VIP), regardless of the cluster that they reside in. In
addition, I want to make sure that if the dependent service is migrated to
another cluster, the dependee is not impacted.

#### Single Service Deployed to Multiple Clusters

I have deployed my stateless service to multiple clusters for redundancy or
scale. Now I want to propagate topologically-aware service endpoints (local,
regional, global) to all clusters, so that other services in my clusters can
access instances of this service in priority order based on availability and
locality. Requests to my replicated service should seamlessly transition (within
SLO for dropped requests) between instances of my service in case of failure or
removal without action by or impact on the caller. Routing to my replicated
service should optimize for cost metric (e.g.prioritize traffic local to zone,
region).


### Constraints

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above.
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

This proposal intends to rely on the K8s [Service Topology API] for topology
aware routing, but that API is currently in flux. As a result this proposal is
only suited to same-region multi-cluster services until the topology API
progresses.

[Service Topology API]: https://kubernetes.io/docs/concepts/services-networking/service-topology/

### Risks and Mitigations

<!--
What are the risks of this proposal and how do we mitigate.  Think broadly.
For example, consider both security and how this will impact the larger
kubernetes ecosystem.

How will security be reviewed and by whom?

How will UX be reviewed and by whom?

Consider including folks that also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable.  This may include API specs (though not always
required) or even code snippets.  If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->
### Exporting Services

Services will not be visible to other clusters in the supercluster by default.
They must be explicitly marked for export by the user. This allows users to
decide exactly which services should be visible outside of the local cluster.

Tooling may (and likely will, in the future) be built on top of this to simplify
the user experience. Some initial ideas are to allow users to specify that all
services in a given namespace or in a namespace selector or even a whole cluster
should be automatically exported by default. In that case, a `ServiceExport`
could be automatically created for all `Services`. This tooling will be designed
in a separate doc, and is secondary to the main API proposed here.

To mark a service for export to the supercluster, a user will create a
ServiceExport CR:

```golang
// ServiceExport declares that the associated service should be exported to
// other clusters.
type ServiceExport struct {
        metav1.TypeMeta `json:",inline"`
        // +optional
        metav1.ObjectMeta `json:"metadata,omitempty"`
        // +optional
        Status ServiceExportStatus `json:"status,omitempty"`
}

// ServiceExportStatus contains the current status of an export.
type ServiceExportStatus struct {
        // +optional
        // +patchStrategy=merge
        // +patchMergeKey=type
        // +listType=map
        // +listMapKey=type
        Conditions []ServiceExportCondition `json:"conditions,omitempty"`
}

// ServiceExportConditionType identifies a specific condition.
type ServiceExportConditionType string

const {
      // ServiceExportExported means that the service referenced by this
      // service export has been synced to all clusters in the supercluster
      ServiceExportExported ServiceExportConditionType = "Exported"
      // ServiceExportInvalid means that the service marked for export has an
      // unexportable service type (ExternalName)
      ServiceExportInvalid ServiceExportConditionType = "InvalidServiceType"
      // ServiceExportHeadless describes the headlessness of the supercluster
      // service. Only required to be set if at least one cluster (including
      // the local one) has a matching headless service marked for export.
      // "True" when the corresponding ServiceImport is headless. "False" if
      // at least one matching service is not headless.
      // If any exported services disagree, cluster names and respective
      // headlessness should be listed in the condition message.
      ServiceExportHeadless ServiceExportConditionType = "Headless"
}

// ServiceExportCondition contains details for the current condition of this
// service export.
//
// Once KEP-1623 (sig-api-machinery/1623-standardize-conditions) is
// implemented, this will be replaced by metav1.Condition.
type ServiceExportCondition struct {
        Type ServiceExportConditionType `json:"type"`
        // Status is one of {"True", "False", "Unknown"}
        Status corev1.ConditionStatus `json:"status"`
        // +optional
        LastTransitionTime *metav1.Time `json:"lastTransitionTime,omitempty"`
        // +optional
        Reason *string `json:"reason,omitempty"`
        // +optional
        Message *string `json:"message,omitempty"`
}
```
```yaml
apiVersion: multicluster.k8s.io/v1alpha1
kind: ServiceExport
metadata:
  name: my-svc
  namespace: my-ns
status:
  conditions:
  - type: Initialized
    status: "True"
    lastTransitionTime: "2020-03-30T01:33:51Z"
  - type: Exported
    status: "True"
    lastTransitionTime: "2020-03-30T01:33:55Z"
```

To export a service, a `ServiceExport` should be created within the cluster and
namespace that the service resides in, name-mapped to the service for export -
that is, they reference the `Service` with the same name as the export. If
multiple clusters within the supercluster have `ServiceExports` with the same
name and namespace, these will be considered the same service and will be
combined at the supercluster level.

_Note: A `Service` without a corresponding `ServiceExport` in its local cluster
will not be exported even if other clusters are exporting a `Service` with the
same namespaced name._

This requires that within a supercluster, a given namespace is governed by a
single authority across all clusters. It is that authority’s responsibility to
ensure that a name is shared by multiple services within the namespace if and
only if they are instances of the same service.

All information about the service, including ports, backends and topology, will
continue to be stored in the `Service` objects, which are each name mapped to a
`ServiceExport`.

Deleting a `ServiceExport` will stop exporting the name-mapped `Service`.

#### Restricting Exports ####

Cluster administrators may use RBAC rules to prevent creation of
`ServiceExports` in select namespaces. While there are no general restrictions
on which namespaces are allowed, administrators should be especially careful
about permitting exports from `kube-system` and `default`. As a best practice,
admins may want to tightly or completely prevent exports from these namespaces
unless there is a clear use case.

### Importing Services

To consume a supercluster service, users will use the domain name associated
with their `ServiceExport`. When the mcs-controller sees a `ServiceExport`, a
`ServiceImport` will be introduced in each cluster to represent the imported
service. Users are primarily expected to consume the service via domain name and
supercluster VIP, but the `ServiceImport` may be used for imported service
discovery via the K8s API and will be used internally as the source of routing
and DNS configuration.

A `ServiceImport` is a service that may have endpoints in other clusters.
This includes 3 scenarios:
1. This service is running entirely in different cluster(s)
1. This service has endpoints in other cluster(s) and in this cluster
1. This service is running entirely in this cluster, but is exported to other cluster(s) as well

For each exported service, one `ServiceExport` will exist in each cluster that
runs the service. The mcs-controller will create and maintain a derived
`ServiceImport` in each cluster within the supercluster (see: [constraints and
conflict resolution](#constraints-and-conflict-resolution)). If all
`ServiceExport` instances are deleted, each `ServiceImport` will also be deleted
from all clusters.

Since a given `ServiceImport` may be backed by multiple `EndpointSlices`, a
given `EndpointSlice` will reference its `ServiceImport` using the label
`multicluster.kubernetes.io/imported-service-name` similarly to how an
`EndpointSlice` is associated with its `Service` in a single cluster. Each
imported `EndpointSlice` will also have a
`multicluster.kubernetes.io/source-cluster` label with the cluster name, a
registry-scoped unique identifier for the cluster.

```golang
// ServiceImport describes a service imported from clusters in a supercluster.
type ServiceImport struct {
  metav1.TypeMeta `json:",inline"`
  // +optional
  metav1.ObjectMeta `json:"metadata,omitempty"`
  // +optional
  Spec ServiceImportSpec `json:"spec,omitempty"`
  // +optional
  Status ServiceImportStatus `json:"status,omitempty"`
}

// ServiceImportType designates the type of a ServiceImport
type ServiceImportType string

const (
  // SuperclusterIP are only accessible via the Supercluster IP.
  SuperclusterIP ServiceImportType = "SuperclusterIP"
  // Headless services allow backend pods to be addressed directly.
  Headless ServiceImportType = "Headless"
)

// ServiceImportSpec describes an imported service and the information necessary to consume it.
type ServiceImportSpec struct {
  // +patchStrategy=merge
  // +patchMergeKey=port
  // +listType=map
  // +listMapKey=port
  // +listMapKey=protocol
  Ports []ServicePort `json:"ports"`
  // +optional
  IP string `json:"ip,omitempty"`
  // +optional
  Type ServiceImportType `json:"type"`
}

// ServiceImportStatus describes derived state of an imported service.
type ServiceImportStatus struct {
  // +optional
  // +patchStrategy=merge
  // +patchMergeKey=cluster
  // +listType=map
  // +listMapKey=cluster
  Clusters []ClusterStatus `json:"clusters"`
}

// ClusterStatus contains service configuration mapped to a specific source cluster
type ClusterStatus struct {
 Cluster string `json:"cluster"`
 // +optional
 SessionAffinity corev1.ServiceAffinity `json:"sessionAffinity"`
 // +optional
 SessionAffinityConfig *corev1.SessionAffinityConfig `json:"sessionAffinityConfig"`
}
```
```yaml
apiVersion: multicluster.k8s.io/v1alpha1
kind: ServiceImport
metadata:
  name: my-svc
  namespace: my-ns
spec:
  ip: 42.42.42.42
  type: "SuperclusterIP"
  ports:
  - name: http
    protocol: TCP
    port: 80
  clusters:
    - cluster: us-west2-a-my-cluster
      sessionAffinity: None
---
apiVersion: discovery.k8s.io/v1beta1
kind: EndpointSlice
metadata:
  name: imported-my-svc-cluster-b-1
  namespace: my-ns
  labels:
    multicluster.kubernetes.io/source-cluster: us-west2-a-my-cluster
    multicluster.kubernetes.io/imported-service-name: my-svc
  ownerReferences:
  - apiVersion: multicluster.k8s.io/v1alpha1
    controller: false
    kind: ServiceImport
    name: my-svc
addressType: IPv4
ports:
  - name: http
    protocol: TCP
    port: 80
endpoints:
  - addresses:
      - "10.1.2.3"
    conditions:
      ready: true
    topology:
     topology.kubernetes.io/zone: us-west2-a
```

The `ServiceImport.Spec.IP` (VIP) can be used to access this service from within this cluster. 

### Supercluster Service Behavior Expectations

#### Service Types

- `ClusterIP`: This is the the straightforward case most of the proposal
  assumes. Each `EndpointSlice` associated with the exported service is combined
  with slices from other clusters to make up the supercluster service. They will
  be imported to the cluster behind the supercluster IP.
- `ClusterIP: none` (Headless): Headless services are supported and will be
  imported with a `ServiceImport` and `EndpointSlices` like any other
  `ClusterIP` service, but the way that they are exposed depends on the
  cluster-level services from which they are derived. A supercluster service
  will be exposed with a regular supercluster IP if any source cluster-level
  service specifies a cluster IP. A supercluster service will be headless if and
  only if all cluster-level services from which it is derived are headless.

  _Exporting a non-headless service to an otherwise headless service can
  dynamically change the supercluster service type and potentially break
  existing consumers. This is likely the result of a deployment error.
  Conditions and events on the `ServiceExport` will be used to communicate the
  conflict to the user._
- `NodePort` and `LoadBalancer`: These create `ClusterIP` services that would
  sync as expected. For example If you export a `NodePort` service, the
  resulting cross-cluster service will still be a supercluster IP type. The
  local service will not be affected. Node ports can still be used to access the
  cluster-local service in the source cluster, and only the supercluster IP will
  reoute to endpoints in remote cluster.
- `ExternalName`: It doesn't make sense to export an `ExternalName` service.
  They can't be merged with other exports, and it seems like it would only
  complicate deployments by even attempting to stretch them across clusters.
  Instead, regular `ExternalName` type `Services` should be created in each
  cluster individually. If a `ServiceExport` is created for an `ExternalName`
  service, an `InvalidServiceType` condition will be set on the export.

#### SuperclusterIP

A non-headless `ServiceImport` is expected to have an associated IP address, the
supercluster IP, which may be accessed from within an importing cluster. This IP
may be supercluster-wide, or assigned on a per-cluster basis, but is expected to
be consistent for the life of a `ServiceImport`. Requests to this IP from within
a cluster will route to backends for the aggregated Service.

Note: this doc does not discuss `NetworkPolicy`, which cannot currently be used
to describe a policy that applies to a multi-cluster service.

#### DNS

_Optional, but recommended._

MCS aims to align with the existing [service DNS
spec](https://github.com/kubernetes/dns/blob/master/docs/specification.md). This
section assumes familiarity with in-cluster Service DNS behavior.
```
<<[UNRESOLVED]>>
Full DNS spec needed prior to Beta graduation following:
https://github.com/kubernetes/dns/blob/master/docs/specification.md
<<[UNRESOLVED]>>
```
When a `ServiceExport` is created, this will cause a domain name for the
multi-cluster service to become accessible from within the supercluster. The
domain name will be `<service>.<ns>.svc.supercluster.local`. 

**SuperclusterIP services:** Requests to this domain name from within an importing
cluster will resolve to the supercluster IP, which points to endpoints for pods
within the underlying `Service`(s) across the supercluster.

**Headless services:** Within an importing cluster, the supercluster domain name
will have multiple `A`/`AAAA` records, each containing the address of a ready
endpoint of the headless service. `<service>.<ns>.svc.supercluster.local` will
resolve to the set of all ready pod IPs for the service.

Pods backing a supercluster service may be addressed individually using the
`<hostname>.<clustername>.<svc>.<ns>.svc.supercluster.local` format. Necessary
records will be created based on each ready endpoint's hostname and the
`multicluster.kubernetes.io/source-cluster` label on the `EndpointSlice`. This
allows naming collisions to be avoided for headless services backed by identical
`StatefulSets` deployed in multiple clusters.

_Note: the total length of a FQDN is limited to 253 characters. Each label is
independently limited to 63 characters, so users must choose host/cluster/service
names to avoid hitting this upper bound._

All service consumers must use the `*.svc.supercluster.local` name to enable
supercluster routing, even if there is a matching `Service` with the same
namespaced name in the local cluster. There will be no change to existing
behavior of the `cluster.local` zone.

#### EndpointSlice

When a `ServiceExport` is created, this will cause `EndpointSlice` objects for
the underlying `Service` to be created in each cluster within the supercluster,
associated with the derived `ServiceImport`. One or more `EndpointSlice`
resources will exist for the exported `Service`, with each `EndpointSlice`
containing only endpoints from a single source cluster. These `EndpointSlice`
objects will be marked as managed by the supercluster service controller, so
that the endpoint slice controller doesn’t delete them. `EndpointSlices` will
have an owner reference to their associated `ServiceImport`.

```
<<[UNRESOLVED]>>
We have not yet sorted out scalability impact here. We hope the upper bound for
imported endpoints + in-cluster endpoints will be ~= the upper bound for
in-cluster endpoints today, but this remains to be determined.
<<[/UNRESOLVED]>>
```

#### Endpoint TTL

To prevent stale endpoints from persisting in the event that a cluster becomes
unreachable to the supercluster controller, each `EndpointSlice` is associated
with a lease representing connectivity with its source cluster. The supercluster
service controller is responsible for periodically renewing the lease so long as
the connection with the source cluster is confirmed alive. A separate
controller, that may run inside each cluster, is responsible for watching each
lease and removing all remaining `EndpointSlices` associated with a cluster when
that cluster’s lease expires.

## Constraints and Conflict Resolution

Exported services are derived from the properties of each component service and
their respective endpoints. However, some properties combine across exports
better than others. They generally fall into two categories: global properties,
and cluster-level properties.

### Global Properties

These properties describe how the service should be consumed as a whole. They
directly impact service consumption and must be consistent across all child
services. If these properties are out of sync for a subset of exported services,
there is no clear way to determine how a service should be accessed. **If any
global properties have conflicts that can not be resolved, a condition will be
set on the `ServiceExport` with a description of the conflict. The service will
not be synced, and an error will be set on the status of each affected
`ServiceExport` and any previously-derived `ServiceImports` will be deleted
from each cluster in the supercluster.**


#### Service Port

A derived service will be accessible with the supercluster IP at the ports
dictated by child services. If the external properties of service ports for a
set of exported services don’t match, we won’t know which port is the correct
choice for a service. For example, if two exported services use different ports
with the name “http”, which port is correct? What if a service uses the same
port with different names? As long as there are no conflicts (different ports
with the same name), the supercluster service will expose the superset of
service ports declared on its component services. If a user wants to change a
service port in a conflicting way, we recommend deploying a new service or
making the change in non-conflicting phases.


### Cluster Level Properties

These properties are export-specific and pertain only to the subset of endpoints
backed by a single instance of each exported service. They may be safely carried
throughout the supercluster without risk of conflict. We propagate these
properties forward with no attempt to merge or alter them.


#### Session Affinity

Session affinity affects a service as a whole for a given consumer. What would
it mean for a service to have e.g. client IP session affinity set for half its
backends? Would sessions only be sticky for those backends, or would there be no
affinity? If sessions are selectively sticky, we’d expect to see traffic to skew
toward the sticky subset of endpoints. That said, there’s nothing preventing us
from applying affinity on a per-slice basis so we will carry it forward.

### Test Plan

E2E tests can use [kind](https://kind.sigs.k8s.io/) to create multiple
clusters to test various multi-cluster scenarios. To meet conditions required by
MCS, cluster networks will be flattened by adding static routes between nodes in
each cluster.

- Test cluster A can contact service imported from cluster B and route to
  expected endpoints.
- Test cluster A local service not impacted by same-name imported service.
- Test cluster A can contact service imported from cluster A and B and route to
  expected endpoints in both clusters.

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy.  Anything
that would count as tricky in the implementation and anything particularly
challenging to test should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations).  Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

#### Alpha -> Beta Graduation

- A detailed DNS spec for multi-cluster services.
- NetworkPolicy either solved or explicitly ruled out.
- API group chosen and approved.
- Kube-proxy can consume ServiceImport and EndpointSlice.
- E2E tests exist for MCS services.
- Beta -> GA Graduation criteria defined.
- At least one MCS DNS implementation

#### Beta -> GA Graduation

- Scalability/performance testing, understanding impact on cluster-local service
  scalability.

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning),
or by redefining what graduation means.

In general, we try to use the same stages (alpha, beta, GA), regardless how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real world usage
- N installs
- More rigorous forms of testing e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least 2 releases between beta and
GA/stable, since there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a deprecated flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality which deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include [conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

Kube-proxy and must be updated to a supported version before MCS services may be
used. To take advantage of MCS DNS, the DNS provider must be upgraded to a
version that implements the MCS spec. Kube-proxy MCS support will be guarded by a
`MultiClusterServices` feature gate. When enabled
<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to keep previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade in order to make use of the enhancement?
-->

### Version Skew Strategy

Kube-proxy and DNS must be upgraded before new MCS API versions may be used.
Backwards compatibility will be maintained in accordance with the
[deprecation policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/).
<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

## Implementation History

<!--
Major milestones in the life cycle of a KEP should be tracked in this section.
Major milestones might include
- the `Summary` and `Motivation` sections being merged signaling SIG acceptance
- the `Proposal` section being merged signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

<!--
## Drawbacks

Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider and why did you rule them out?  These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### `ObjectReference` in `ServiceExport.Spec` to directly map to a Service

Instead of name mapping, we could use an explicit `ObjectReference` in a
`ServiceExport.Spec`. This feels familiar and more explicit, but fundamentally
changes certain characteristics of the API. Name mapping means that the export
must be in the same namespace as the `Service` it exports, allowing existing RBAC
rules to restrict export rights to current namespace owners. We are building on
the concept that a namespace belongs to a single owner, and it should be the
`Service` owner who controls whether or not a given `Service` is exported. Using
`ObjectReference` instead would also open the possibility of having multiple
exports acting on a single service and would require more effort to determine if
a given service has been exported.

The above issues could also be solved via controller logic, but we would risk
differing implementations. Name mapping enforces behavior at the API.

### Export services via label selector

Instead of name mapping, `ServiceExport` could have a
`ServiceExport.Spec.ServiceSelector` to select matching services for export.
This approach would make it easy to simply export all services with a given
label applied and would still scope exports to a namespace, but shares other
issues with the `ObjectReference` approach above:

- Multiple `ServiceExports` may export a given `Service`, what would that mean?
- Determining whether or not a service is exported means seaching
  `ServiceExports` for a matching selector.

Though multiple services may match a single export, the act of exporting would
still be independent for individual services. A report of status for each export
seems like it belongs on a service-specific resource.

With name mapping it should be relatively easy to build generic or custom logic
to automatically ensure a `ServiceExport` exists for each `Service` matching a
selector - perhaps by introducing something like a `ServiceExportPolicy`
resource (out of scope for this KEP). This would solve the above issues but
retain the flexibility of selectors.

### Export via annotation

`ServiceExport` as described has no spec and seems like it could just be
replaced with an annotation, e.g. `multicluster.kubernetes.io/export`. When a
service is found with the annotation, it would be considered marked for export
to the supercluster. The controller would then create `EndpointSlices` and an
`ServiceImport` in each cluster exactly as described above. Unfortunately,
`Service` does not have an extensible status and there is no way to represent
the state of the export on the annotated `Service`. We could extend
`Service.Status` to include `Conditions` and provide the flexibility we need,
but requiring changes to `Service` makes this a much more invasive proposal to
achieve the same result. As the use of a multi-cluster service implementation
would be an optional addon, it doesn't warrant a change to such a fundamental
resource.


## Infrastructure Needed
<!--
Use this section if you need things from the project/SIG.  Examples include a
new subproject, repos requested, github details.  Listing these here allows a
SIG to get the process for these resources started right away.
-->
To facilitate consumption by kube-proxy, the MCS CRDS/client need to live in
kubernetes/staging. We will need a new k8s.io/multiclusterservices repo for
published MCS code.
