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
    - [Different ClusterIP Services Each Deployed to Separate Cluster](#different-clusterip-services-each-deployed-to-separate-cluster)
    - [Single Service Deployed to Multiple Clusters](#single-service-deployed-to-multiple-clusters)
  - [Constraints](#constraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Exporting Services](#exporting-services)
    - [Restricting Exports](#restricting-exports)
  - [Importing Services](#importing-services)
  - [ClusterSet Service Behavior Expectations](#clusterset-service-behavior-expectations)
    - [Service Types](#service-types)
    - [ClusterSetIP](#clustersetip)
    - [DNS](#dns)
      - [No PTR records necessary for multicluster DNS](#no-ptr-records-necessary-for-multicluster-dns)
      - [Not allowing cluster-specific targeting via DNS](#not-allowing-cluster-specific-targeting-via-dns)
  - [Tracking Endpoints](#tracking-endpoints)
    - [Using <code>EndpointSlice</code> objects to track endpoints](#using-endpointslice-objects-to-track-endpoints)
    - [Endpoint TTL](#endpoint-ttl)
- [Constraints and Conflict Resolution](#constraints-and-conflict-resolution)
  - [Global Properties](#global-properties)
    - [Service Port](#service-port)
    - [Headlessness](#headlessness)
    - [Session Affinity](#session-affinity)
    - [Internal Traffic Policy](#internal-traffic-policy)
    - [Traffic Distribution](#traffic-distribution)
    - [Labels and Annotations](#labels-and-annotations)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
  - [<code>ObjectReference</code> in <code>ServiceExport.Spec</code> to directly map to a Service](#objectreference-in-serviceexportspec-to-directly-map-to-a-service)
  - [Export services via label selector](#export-services-via-label-selector)
  - [Export via annotation](#export-via-annotation)
  - [Other conflict resolution algorithms](#other-conflict-resolution-algorithms)
  - [Exporting labels/annotations from the Service/ServiceExport objects](#exporting-labelsannotations-from-the-serviceserviceexport-objects)
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

- [ ] Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG
  Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to
  mailing list discussions/SIG meetings, relevant PRs/issues, release notes

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

*Converted from this [original proposal
doc](http://bit.ly/k8s-mc-svc-api-proposal).*

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
that would otherwise be able to make use of metadata (e.g. endpoint topology) to
better direct traffic. To support failover or temporarily during migration,
users may want to consume services spread across clusters, but today that
requires non-trivial bespoke solutions.

The Multi-Cluster Services API aims to fix these problems.

### Goals

<!--
List the specific goals of the KEP.  What is it trying to achieve?  How will we
know that this has succeeded?
-->
- Define a minimal API to support service discovery and consumption across
  clusters.
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

- **clusterset** - A placeholder name for a group of clusters with a high degree
  of mutual trust and shared ownership that share services amongst themselves.
  Membership in a clusterset is symmetric and transitive. The set of member
  clusters are mutually aware, and agree about their collective association.
  Within a clusterset, [namespace sameness] applies and all namespaces with a
  given name are considered to be the same namespace. Implementations of this
  API are responsible for defining and tracking membership in a clusterset. The
  specific mechanism is out of scope of this proposal.
- **mcs-controller** - A controller that syncs services across clusters and
  makes them available for multi-cluster service discovery and connectivity.
  There may be multiple implementations, this doc describes expected common
  behavior. The controller may be a single controller, multiple decentralized
  controllers, or a human using kubectl to create resources. This document aims
  to support any implementation that fulfills the behavioral expectations of
  this API.
- **cluster name** - A unique identifier for a cluster, scoped to the
  implementation's cluster registry. We do not attempt to define the registry.
  The cluster name must be a valid [RFC 1123](https://tools.ietf.org/html/rfc1123)
  DNS label.

  The cluster name should be consistent for the life of a cluster and its
  membership in the clusterset. Implementations should treat name mutation as a
  delete of the membership followed by recreation with the new name.
- **cluster id** - A unique identifier for a cluster, scoped to a clusterset.
  The cluster id must be either:
  - equal to cluster name,
  - or composed of two valid [RFC 1123](https://tools.ietf.org/html/rfc1123)
    DNS labels separated with a dot. The first label equals cluster name and the
    second one gives additional context, allowing the implementation to uniquely
    identify a cluster within a clusterset composed of clusters registered with
    multiple cluster registries.

  The cluster id should be consistent for the life of a cluster and its
  membership in the clusterset. Implementations should treat id mutation as a
  delete of the membership followed by recreation with the new name.

[namespace sameness]:
    https://github.com/kubernetes/community/blob/master/sig-multicluster/namespace-sameness-position-statement.md

We propose a new CRD called `ServiceExport`, used to specify which services
should be exposed across all clusters in the clusterset. `ServiceExports` must
be created in each cluster that the underlying `Service` resides in. Creation of
a `ServiceExport` in a cluster will signify that the `Service` with the same
name and namespace as the export should be visible to other clusters in the
clusterset.

Another CRD called `ServiceImport` will be introduced to act as the in-cluster
representation of a multi-cluster service in each importing cluster. This is
analogous to the traditional `Service` type in Kubernetes. Importing clusters
will have a corresponding `ServiceImport` for each uniquely named `Service` that
has been exported within the clusterset, referenced by namespaced name.
`ServiceImport` resources will be managed by the MCS implementation's
mcs-controller.

If multiple clusters export a `Service` with the same namespaced name, they will
be recognized as a single combined service. For example, if 5 clusters export
`my-svc.my-ns`, each importing cluster will have one `ServiceImport` named
`my-svc` in the `my-ns` namespace and it will be associated with endpoints from
all exporting clusters. Properties of the `ServiceImport` (e.g. ports, topology)
will be derived from a merger of component `Service` properties.

This specification is not prescriptive on exact implementation details. Existing
implementations of Kubernetes Service API (e.g. kube-proxy) can be extended to
present `ServiceImports` alongside traditional `Services`. One often discussed
implementation requiring no changes to kube-proxy is to have the mcs-controller
maintain ServiceImports and create "dummy" or "shadow" Service objects, named
after a mcs-controller managed EndpointSlice that aggregates all cross-cluster
backend IPs, so that kube-proxy programs those endpoints like a regular Service.
Other implementations are encouraged as long as the properties of the API described
in this document are maintained.

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system.  The goal here is to make this feel real for users without getting
bogged down.
-->

#### Different ClusterIP Services Each Deployed to Separate Cluster

I have 2 clusters, each running different ClusterIP services managed by different
teams, where services from one team depend on services from the other team. I want
to ensure that a service from one team can discover a service from the other team
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
service should optimize for cost metric (e.g. prioritize traffic local to zone,
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

As the plan for dual stack support is finalized, the Multi-Cluster Services API
will follow dual stack Service design. Until then, dual stack will not be
supported.

[Service Topology API]:
    https://kubernetes.io/docs/concepts/services-networking/service-topology/

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

Services will not be visible to other clusters in the clusterset by default.
They must be explicitly marked for export by the user. This allows users to
decide exactly which services should be visible outside of the local cluster.

Tooling may (and likely will, in the future) be built on top of this to simplify
the user experience. Some initial ideas are to allow users to specify that all
services in a given namespace or in a namespace selector or even a whole cluster
should be automatically exported by default. In that case, a `ServiceExport`
could be automatically created for all `Services`. This tooling will be designed
in a separate doc, and is secondary to the main API proposed here.

To mark a service for export to the clusterset, a user will create a
ServiceExport CR:

```golang
// ServiceExport declares that the associated service should be exported to
// other clusters.
type ServiceExport struct {
        metav1.TypeMeta `json:",inline"`
        // +optional
        metav1.ObjectMeta `json:"metadata,omitempty"`
        // +optional
        Spec ServiceExportSpec `json:"spec,omitempty"`
        // +optional
        Status ServiceExportStatus `json:"status,omitempty"`
}

// ServiceExportSpec describes an exported service and extra exported information
type ServiceExportSpec struct {
        // +optional
        ExportedLabels map[string]string `json:"exportedLabels"`
        // +optional
        ExportedAnnotations map[string]string `json:"exportedAnnotations"`
}

// ServiceExportStatus contains the current status of an export.
type ServiceExportStatus struct {
        // +optional
        // +patchStrategy=merge
        // +patchMergeKey=type
        // +listType=map
        // +listMapKey=type
        Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
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
  - type: Valid
    status: "True"
    lastTransitionTime: "2020-03-30T01:33:51Z"
    reason: Valid
    message: "The ServiceExport and its Service is exportable."
  - type: Ready
    status: "True"
    lastTransitionTime: "2020-03-30T01:33:55Z"
    reason: Exported
    message: "The service has been exported"
  - type: Conflict
    status: "True"
    lastTransitionTime: "2020-03-30T01:33:55Z"
    reason: TypeConflict
    message: "Conflicting type. Using \"ClusterSetIP\" from oldest service export in \"cluster-1\". 2/5 clusters disagree."
```

To export a service, a `ServiceExport` should be created within the cluster and
namespace that the service resides in, name-mapped to the service for export -
that is, they reference the `Service` with the same name as the export. If
multiple clusters within the clusterset have `ServiceExports` with the same name
and namespace, these will be considered the same service and will be combined at
the clusterset level.

_Note: A `Service` without a corresponding `ServiceExport` in its local cluster
will not be exported even if other clusters are exporting a `Service` with the
same namespaced name._

This requires that within a clusterset, a given namespace is governed by a
single authority across all clusters. It is that authority’s responsibility to
ensure that a name is shared by multiple services within the namespace if and
only if they are instances of the same service.

Most information about the service, including ports, backends, topology and
session affinity, internal traffic policy, and traffic distribution
will continue to be stored in the `Service` objects, which are each name
mapped to a `ServiceExport`. This does not apply for labels and
annotations which are stored in `ServiceExport` directly in `spec.exportedLabels`
and `spec.exportedAnnotations`. Exporting labels and annotations is optionally
supported by MCS-API implementations. If supported, annotations or labels must
not be exported from the `metadata` of the `Service` or `ServiceExport` resources.

Deleting a `ServiceExport` will stop exporting the name-mapped `Service`.

#### Restricting Exports

Cluster administrators may use RBAC rules to prevent creation of
`ServiceExports` in select namespaces. While there are no general restrictions
on which namespaces are allowed, administrators should be especially careful
about permitting exports from `kube-system` and `default`. As a best practice,
admins may want to tightly or completely prevent exports from these namespaces
unless there is a clear use case.

### Importing Services

To consume a clusterset service, the domain name associated with the
multi-cluster service should be used (see [DNS](#dns)). When the mcs-controller
sees a `ServiceExport`, a `ServiceImport` will be introduced in each importing
cluster to represent the imported service. Users are primarily expected to
consume the service via domain name and clusterset VIP, but the `ServiceImport`
may be used for imported service discovery via the K8s API and will be used
internally as the source of truth for routing and DNS configuration.

A `ServiceImport` is a service that may have endpoints in other clusters. This
includes 3 scenarios:
1. This service is running entirely in different cluster(s).
2. This service has endpoints in other cluster(s) and in this cluster.
3. This service is running entirely in this cluster, but is exported to other
   cluster(s) as well.

A multi-cluster service will be imported only by clusters in which the service's
namespace exists. All clusters containing the service's namespace will import
the service. This means that all exporting clusters will also import the
multi-cluster service. _An implementation may or may not decide to create
missing namespaces automatically, that behavior is out of scope of this spec._

Because of the potential wide impact a `ServiceImport` may have within a
cluster, non-cluster-admin users should not be allowed to create or modify
`ServiceImport` resources. The mcs-controller should be solely responsible for
the lifecycle of a `ServiceImport`.

Some errors may occur during the `ServiceImport`'s lifecycle, such as IP protocol
incompatibilities (i.e.: importing an IPv6 only service in an IPv4 cluster). These
errors and general status reporting of a `ServiceImport` should be reported
via its status conditions field.

For each exported service, one `ServiceExport` will exist in each cluster that
exports the service. The mcs-controller will create and maintain a derived
`ServiceImport` in each cluster within the clusterset so long as the service's
namespace exists (see: [constraints and conflict
resolution](#constraints-and-conflict-resolution)). If all `ServiceExport`
instances are deleted, each `ServiceImport` will also be deleted from all
clusters.

```golang
// ServiceImport describes a service imported from clusters in a clusterset.
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
  // ClusterSetIP are only accessible via the ClusterSet IP.
  ClusterSetIP ServiceImportType = "ClusterSetIP"
  // Headless services allow backend pods to be addressed directly.
  Headless ServiceImportType = "Headless"
)

// ServiceImportSpec describes an imported service and the information necessary to consume it.
type ServiceImportSpec struct {
  // +listType=atomic
  Ports []ServicePort `json:"ports"`
  // +kubebuilder:validation:MaxItems:=1
  // +optional
  IPs []string `json:"ips,omitempty"`
  // +optional
  Type ServiceImportType `json:"type"`
  // +optional
  SessionAffinity corev1.ServiceAffinity `json:"sessionAffinity"`
  // +optional
  SessionAffinityConfig *corev1.SessionAffinityConfig `json:"sessionAffinityConfig"`
  // +optional
  InternalTrafficPolicy *corev1.ServiceInternalTrafficPolicy `json:"internalTrafficPolicy,omitempty"`
  // The possible TrafficDistribution values should match what can be similarly
  // defined in a Service, see https://kubernetes.io/docs/concepts/services-networking/service/#traffic-distribution
  // +optional
  TrafficDistribution *string `json:"trafficDistribution,omitempty"`
}

// ServicePort represents the port on which the service is exposed
type ServicePort struct {
  // The name of this port within the service. This must be a DNS_LABEL.
  // All ports within a ServiceSpec must have unique names. When considering
  // the endpoints for a Service, this must match the 'name' field in the
  // EndpointPort.
  // Optional if only one ServicePort is defined on this service.
  // +optional
  Name string `json:"name,omitempty"`

  // The IP protocol for this port. Supports "TCP", "UDP", and "SCTP".
  // Default is TCP.
  // +optional
  Protocol Protocol `json:"protocol,omitempty"`

  // The application protocol for this port.
  // This field follows standard Kubernetes label syntax.
  // Un-prefixed names are reserved for IANA standard service names (as per
  // RFC-6335 and http://www.iana.org/assignments/service-names).
  // Non-standard protocols should use prefixed names such as
  // mycompany.com/my-custom-protocol.
  // Field can be enabled with ServiceAppProtocol feature gate.
  // +optional
  AppProtocol *string `json:"appProtocol,omitempty"`

  // The port that will be exposed by this service.
  Port int32 `json:"port"`
}

// ServiceImportStatus describes derived state of an imported service.
type ServiceImportStatus struct {
  // +optional
  // +patchStrategy=merge
  // +patchMergeKey=cluster
  // +listType=map
  // +listMapKey=cluster
  Clusters []ClusterStatus `json:"clusters"`
  // +optional
  // +patchStrategy=merge
  // +patchMergeKey=type
  // +listType=map
  // +listMapKey=type
  Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`
}

// ClusterStatus contains service configuration mapped to a specific source cluster
type ClusterStatus struct {
 Cluster string `json:"cluster"`
}
```

```yaml
apiVersion: multicluster.k8s.io/v1alpha1
kind: ServiceImport
metadata:
  name: my-svc
  namespace: my-ns
spec:
  ips:
  - 42.42.42.42
  type: "ClusterSetIP"
  ports:
  - name: http
    protocol: TCP
    port: 80
  sessionAffinity: None
status:
  conditions:
  - type: Ready
    reason: Ready
    status: "True"
    lastTransitionTime: "2020-03-30T01:33:51Z"
  clusters:
  - cluster: us-west2-a-my-cluster
```

The `ServiceImport.Spec.IP` (VIP) can be used to access this service from within
this cluster.


### ClusterSet Service Behavior Expectations

#### Service Types

- `ClusterIP`: This is the straightforward case most of the proposal assumes.
  Each endpoint from a producing cluster associated with the exported service is
  aggregated with endpoints from other clusters to make up the clusterset
  service. They will be imported to the cluster behind the clusterset IP, with a
  `ServiceImport` of type `ClusterSetIP`. The details on how the clusterset IP
  is allocated or how the combined slices are maintained may vary by
  implementation; see also [Tracking Endpoints](#TrackingEndpoints).
- `ClusterIP: none` (Headless): Headless services are supported and will be
  imported with a `ServiceImport` like any other `ClusterIP` service, but do not
  configure a VIP and must be consumed via [DNS](#DNS). Their `ServiceImport`s
  will be of type `Headless`. A multi-cluster service's headlessness is derived
  from it's constituent exported services according to the [conflict resolution
  policy](#constraints-and-conflict-resolution).

  _Exporting a non-headless service to an otherwise headless service can
  dynamically change the clusterset service type when an old export is removed,
  potentially breaking existing consumers. This is likely the result of a
  deployment error. Conditions and events on the `ServiceExport` will be used to
  communicate conflicts to the user._
- `NodePort` and `LoadBalancer`: These create `ClusterIP` services that would
  sync as expected. For example if you export a `NodePort` service, the
  resulting cross-cluster service will still be a clusterset IP type. The local
  service will not be affected. Node ports can still be used to access the
  cluster-local service in the source cluster, and only the clusterset IP will
  route to endpoints in remote clusters.
- `ExternalName`: It doesn't make sense to export an `ExternalName` service.
  They can't be merged with other exports, and it seems like it would only
  complicate deployments by even attempting to stretch them across clusters.
  Instead, regular `ExternalName` type `Services` should be created in each
  cluster individually. If a `ServiceExport` is created for an `ExternalName`
  service, a condition type `Valid` with reason `InvalidServiceType` and
  status `false` will be set on the `ServiceExport`.

#### ClusterSetIP

A non-headless `ServiceImport` is expected to have an associated IP address, the
clusterset IP, which may be accessed from within an importing cluster. This IP
may be a single IP used clusterset-wide or assigned on a per-cluster basis, but
is expected to be consistent for the life of a `ServiceImport` from the
perspective of the importing cluster. Requests to this IP from within a cluster
will route to backends for the aggregated Service.

Note: this doc does not discuss `NetworkPolicy`, which cannot currently be used
to describe a selector based policy that applies to a multi-cluster service.

#### DNS

_Optional, but recommended._

The full specification for Multicluster Service DNS is in this KEP's
[specification.md](specification.md). MCS aims to align with the existing
[service DNS
spec](https://github.com/kubernetes/dns/blob/master/docs/specification.md). This
section provides an overview of the multicluster DNS specification and its
rationale, and assumes familiarity with in-cluster Service DNS behavior.

In short, when a `ServiceExport` is created, this will cause a domain name for
the multi-cluster service to become accessible from within the clusterset. The
domain name will be `<service>.<ns>.svc.clusterset.local`. This domain name
operates differently depending on whether the `ServiceExport` refers to a
ClusterSetIP or Headless service:

  * **ClusterSetIP services:** Requests to this domain name from within an
importing cluster will resolve to the clusterset IP. Requests to this IP will be
spread across all endpoints exported with `ServiceExport`s across the
clusterset.
  * **Headless services:** Within an importing cluster, the clusterset domain
name will have multiple `A`/`AAAA` records, each containing the address of a
ready endpoint of the headless service. `<service>.<ns>.svc.clusterset.local`
will resolve to the entire set or the subset of ready pod IPs, depending on the
implementation and endpoint count.

In addition, other resource records are included to conform to in-cluster
Service DNS behavior. SRV records are included to support known use cases such
as VOIP, Active Directory, and etcd cluster bootstrapping. Pods backing a
Headless service may be addressed individually using the
`<hostname>.<clusterid>.<svc>.<ns>.svc.clusterset.local` format; necessary
records will be created based on each ready endpoint's hostname and the
`multicluster.kubernetes.io/source-cluster` label on the `EndpointSlice`. This
allows naming collisions to be avoided for headless services backed by identical
`StatefulSets` deployed in multiple clusters.

_Note: the total length of a FQDN is limited to 253 characters. Each label is
independently limited to 63 characters, so users must choose
host/cluster/service names to avoid hitting this upper bound._

All service consumers must use the `*.svc.clusterset.local` name to enable
clusterset routing, even if there is a matching `Service` with the same
namespaced name in the local cluster. This name allows service consumers to
opt-in to multi-cluster behavior. There will be no change to existing behavior
of the `cluster.local` zone.

_It is expected that the `.clusterset.local` zone is standard and available in
all implementations, but customization and/or aliasing can be explored if
there's demand._

##### No PTR records necessary for multicluster DNS

This specification does not require `PTR` records be generated in the course of
implementing multicluster DNS. By definition, each IP must only have one `PTR`
record, to facilitate reverse DNS lookup. The cluster-local Kubernetes DNS
specification already requires a `PTR` record for the ready IPs for ClusterIP
and Headless Services. As this specification is currently written, by not
requiring any new `PTR` records and leaving the cluster-local `PTR` records as
the only ones, `PTR` record existence becomes potentially inconsistent for
multicluster DNS, especially between importing and exporting clusters (for
example, a Headless pod IP `PTR` record would exist on the exporting cluster,
but not necessarily on an importing cluster). On the other hand, some existing
MCS API implementations create a new "dummy" cluster-local `Service` object for
every `ServiceImport`, and due to the cluster-local DNS specification, they will
already have a `PTR` record generated due to the DNS resolution of the "dummy"
`Service`.

In cases where `PTR` records are not always set, if the specification did
require to backfill in a `clusterset.local` zoned one wherever one is missing
(i.e. for importing clusters), the result would be a patchwork of
`cluster.local` and `clusterset.local` `PTR` records, depending what cluster in
the ClusterSet you are querying from, still resulting in an inconsistent
experience.

Alternatively, the multicluster DNS specification could have required
`clusterset.local` `PTR` records across the board, making the experience
consistent. This would require implementations to overwrite the cluster-local
behavior for MCS services since IPs can only have one `PTR` record. However, the
MCS API purposefully tries to avoid changing cluster-local behavior as much as
possible.

Fundamentally, `PTR` records are used for reverse DNS lookup from an IP to a DNS
name. Besides this, some potentially useful information (ex mapping pod IPs, if
you happen to have one out of context, to their related Service objects) would
be consistently surfaced through reverse DNS lookup if we required
`clusterset.local` `PTR` records. However, the k8s API server contains the same
metadata and is already potentially accessible to any MCS client since the
requests originate in-clusterset. Without a strong use case for requiring them
and given the desire to avoid changing cluster-local behavior, `PTR` records are
not required for multicluster DNS.

##### Not allowing cluster-specific targeting via DNS

While we reserve the form `<clusterid>.<svc>.<ns>.svc.clusterset.local.` for
possible future use, both ClusterSetIP Services and Multicluster Headless
Services are specified to explicitly disallow using this form to create DNS
records that target all 1+N backends in a _specific_ cluster.

For ClusterSetIP services, this rationale is tied to the intent of its
underlying ClusterIP Service. In a single-cluster setup, the purpose of a
ClusterIP service is to reduce the context needed by the application to target
ready backends, especially if those backends disappear or change frequently, and
leverages kube-proxy to do this independent of the limitations of DNS.
([ref](https://kubernetes.io/docs/concepts/services-networking/service/#why-not-use-round-robin-dns))
Similarly, users of exported ClusterIP services should depend on the single
`<clusterset-ip>` (or the single `A`/`AAAA` record mapped to it), instead of
targeting per cluster backends. If a user has a need to target backends in a
different way, they should use headless Services.

For Multicluster Headless Services, the rationale is tied to the intent of its
underlying Headless Service to provide absolutely no load balancing capabilities
on any stateful dimension of the backends (such as cluster locality), and
provide routing to each single backend for the application's purposes.

In both cases, this restriction seeks to preserve the MCS position on [namespace
sameness](https://github.com/kubernetes/community/blob/master/sig-multicluster/namespace-sameness-position-statement.md).
Services of the same name/namespace exported in the multicluster environment are
considered to be the same by definition, and thus their backends are safe to
'merge' at the clusterset level. If these backends need to be addressed
differently based on other properties than name and namespace, they lose their
fungible nature which the MCS API depends on. In these situations, those
backends should instead be fronted by a Service with a different name and/or
namespace.

For example, say an application wishes to target the backends for a
`ClusterSetIP ServiceExport` called `special/prod` in `<clusterid>=cluster-east`
separately from all backends in `<clusterid>=cluster-west`. Instead of depending
on the disallowed implementation of cluster-specific addressing, the Services in
each specific cluster should actually be considered non-fungible and be created
and exported by `ServiceExport`s with different names that honor the boundaries
of their sameness, such as `special-east/prod` for all the backends in
`<clusterid>=cluster-east` and `special-west/prod` for the backends in
`<clusterid>=cluster-west`. In this situation, the resulting DNS names
`special-east.prod.svc.clusterset.local` and
`special-west.prod.svc.clusterset.local` encode the cluster-specific addressing
required by virtue of being two different `ServiceExport`s.

Note that this puts the burden of enforcing the boundaries of a
`ServiceExport`'s fungibility on the name/namespace creator.

Individually addressing pods backing a Headless service is exempt from the rules
described in this section. Such a pod may be addressed using the
`<hostname>.<clusterid>.<svc>.<ns>.svc.clusterset.local` format, where `clusterid`
must uniquely identify a cluster within a clusterset. The implementation may use
cluster name as `clusterid`, and this is not ambiguous if all the clusters on
the clusterset are registered with the same cluster registry. In case a
clusterset contains clusters registered with multiple registries, cluster name
may be ambiguous. The implementation may in such case use `clusterid` composed
of cluster name and an additional DNS label, separated with a dot. The
additional label gives additional context, which is implementation-dependent and
may be used for instance to uniquely identify the cluster registry with which a
cluster is registered.

### Tracking Endpoints

The specific mechanism by which the `mcs-controller` maintains references to the
individual backends for an aggregated service is an implementation detail not
fully prescribed by this specification. Implementations may depend on a higher
level (possibly vendor-specific) API, offload to a load balancer or xDS server
(like Envoy), or use Kubernetes networking APIs. If the implementation depends
on Kubernetes networking APIs, specifically `EndpointSlice` objects, they must
conform to the specification in the following section.

#### Using `EndpointSlice` objects to track endpoints

_Optional to create, but specification defined if present._

If an implementation does create `discovery.k8s.io/v1 EndpointSlice`s, they must
conform to the following structure. This structure was originally required as
part of this specification in alpha, and are the structure on which other
SIG-endorsed reference implementations and tooling, like the [CoreDNS
multicluster plugin](https://github.com/coredns/multicluster/), depend.

When a `ServiceExport` is created, this will cause `EndpointSlice` objects for
the underlying `Service` to be created in each importing cluster within the
clusterset, associated with the derived `ServiceImport`. One or more
`EndpointSlice` resources will exist for the exported `Service`, with each
`EndpointSlice` containing only endpoints from a single source cluster. An
`EndpointSlice` created by an mcs-controller must be marked as managed by the
mcs-controller, not the default `EndpointSlice` controller to avoid any conflicts
between the controllers.

When a service is un-exported, the associated EndpointSlices will be deleted.
The specific mechanism by which they are deleted is an implementation detail.

Since a given `ServiceImport` may be backed by multiple `EndpointSlices`, a
given `EndpointSlice` will reference its `ServiceImport` using the label
`multicluster.kubernetes.io/service-name` similarly to how an `EndpointSlice` is
associated with its `Service` in a single cluster.

Each imported `EndpointSlice` will also have a
`multicluster.kubernetes.io/source-cluster` label with the cluster id, a
clusterset-scoped unique identifier for the cluster. The `EndpointSlice`s
imported for a service are not guaranteed to exactly match the originally
exported `EndpointSlice`s, but each slice is guaranteed to map only to a single
source cluster.

If the implementation is using `EndpointSlice`s in this way, the mcs-controller
is responsible for managing the imported `EndpointSlice`s and making sure they
are conformant with this section.

```yaml
apiVersion: multicluster.k8s.io/v1alpha1
kind: ServiceImport
metadata:
  name: my-svc
  namespace: my-ns
spec:
  ips:
  - 42.42.42.42
  type: "ClusterSetIP"
  ports:
  - name: http
    protocol: TCP
    port: 80
  sessionAffinity: None
status:
  clusters:
  - cluster: us-west2-a-my-cluster
---
apiVersion: discovery.k8s.io/v1beta1
kind: EndpointSlice
metadata:
  name: imported-my-svc-cluster-b-1
  namespace: my-ns
  labels:
    multicluster.kubernetes.io/source-cluster: us-west2-a-my-cluster
    multicluster.kubernetes.io/service-name: my-svc
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

```
<<[UNRESOLVED]>>
We have not yet sorted out scalability impact here. We hope the upper bound for
imported endpoints + in-cluster endpoints will be ~= the upper bound for
in-cluster endpoints today, but this remains to be determined.
<<[/UNRESOLVED]>>
```

#### Endpoint TTL

To prevent stale endpoints from persisting in the event that the mcs-controller
is unable to reach a cluster, it is recommended that an implementation provide
an in-cluster controller to monitor and remove stale endpoints. This may be the
mcs-controller itself in distributed implementations.

We recommend creating leases to represent connectivity with source clusters.
These leases should be periodically renewed by the mcs-controller while the
connection with the source cluster is confirmed alive. When a lease expires, the
cluster id and `multicluster.kubernetes.io/source-cluster` label may be used
to find and remove all `EndpointSlices` containing endpoints from the
unreachable cluster.


## Constraints and Conflict Resolution

Exported services are derived from the properties of each component service and
their respective endpoints. However, some properties combine across exports
better than others.

### Global Properties

These properties describe how the service should be consumed as a whole. They
directly impact service consumption and must be consistent across all child
services. If these properties are out of sync for a subset of exported services,
there is no clear way to determine how a service should be accessed.

Conflict resolution policy: **If any properties have conflicting values that can
not simply be merged, a `Conflict` condition with a `true` status will be set
on all `ServiceExport`s for the conflicted service with a description of the conflict.
The conflict will be resolved by assigning precedence based on each
`ServiceExport`'s `creationTimestamp`, from oldest to newest.**

**Note:** When a `ServiceExport`'s conflict condition changes from `False` to `True` due to this resolution policy, runtime traffic remains unaffected. The oldest cluster will win the conflict and continue to be referenced in the `ServiceImport`, maintaining service continuity. Conversely, when the conflict condition transitions from `True` to `False` (for example, when the oldest cluster's service is unexported), the `ServiceImport` may remain unchanged to avoid potentially disruptive changes to active traffic patterns.  

#### Service Port

A derived service will be accessible with the clusterset IP at the ports
dictated by child services. If the external properties of service ports for a
set of exported services don’t match, the clusterset service will expose the
union of service ports declared on its constituent services.

Like regular services, the resulting ports must respect two rules:
- Have no duplicated names (including unnamed/empty name)
- Two ports must not have the same protocol and port number

As a result, MCS-API implementations should merge ports from constituent
services first based on port name then by the protocol and port number pair.
The conflict resolution policy will determine which of the duplicated ports
are used by the ServiceImport.

#### Headlessness

Headlessness affects a service as a whole for a given consumer. Whether or not a
derived service is headless will be decided according to the conflict resolution
policy.

#### Session Affinity

Session affinity affects a service as a whole for a given consumer. The derived
service's session affinity will be decided according to the conflict resolution
policy.

#### Internal Traffic Policy

Internal traffic policy affects a service as a whole for a given consumer. The derived
service's internal traffic policy will be decided according to the conflict resolution
policy.

#### Traffic Distribution

Traffic distribution affects a service as a whole for a given consumer. The derived
service's traffic distribution will be decided according to the conflict resolution
policy.

#### Labels and Annotations

If supported, exporting labels and annotations would affect a `Service` as a whole
for a given consumer. The derived service's labels and annotations will be decided
according to the conflict resolution if the set of name/value pairs are not identical
between the constituent clusters.

### Test Plan

E2E tests can use [kind](https://kind.sigs.k8s.io/) to create multiple clusters
to test various multi-cluster scenarios. To meet conditions required by MCS,
cluster networks will be flattened by adding static routes between nodes in each
cluster.

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
- E2E tests exist for MCS services.
- Beta -> GA Graduation criteria defined.
- At least one MCS DNS implementation.
- A formal plan for a standard Cluster ID.
- Finalize a name for the "supercluster" concept.
- [Cluster ID KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/2149-clusterid) is in beta

#### Beta -> GA Graduation

- Scalability/performance testing, understanding impact on cluster-local service
  scalability.
- [Cluster ID KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/2149-clusterid) is GA, with at least one other multi-cluster use case.

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
version that implements the MCS spec. Kube-proxy MCS support will be guarded by
a `MultiClusterServices` feature gate. When enabled, kube-proxy will watch the
`serviceimports.multicluster.k8s.io` CRD. MCS support will be dynamically
enabled and disabled as the CRD is created and deleted.
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
Backwards compatibility will be maintained in accordance with the [deprecation
policy](https://kubernetes.io/docs/reference/using-api/deprecation-policy/).
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

- 2020-02-05 - Initial Proposal
- 2020-05-10 - Merged as provisional
- 2020-06-22 - Moved to implementable
- 2020-08-04 - ClusterSet name finalized
- 2020-08-10 - Alpha implementation available at
  [sigs.k8s.io/mcs-api](http://sigs.k8s.io/mcs-api)
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
must be in the same namespace as the `Service` it exports, allowing existing
RBAC rules to restrict export rights to current namespace owners. We are
building on the concept that a namespace belongs to a single owner, and it
should be the `Service` owner who controls whether or not a given `Service` is
exported. Using `ObjectReference` instead would also open the possibility of
having multiple exports acting on a single service and would require more effort
to determine if a given service has been exported.

The above issues could also be solved via controller logic, but we would risk
differing implementations. Name mapping enforces behavior at the API.

### Export services via label selector

Instead of name mapping, `ServiceExport` could have a
`ServiceExport.Spec.ServiceSelector` to select matching services for export.
This approach would make it easy to simply export all services with a given
label applied and would still scope exports to a namespace, but shares other
issues with the `ObjectReference` approach above:

- Multiple `ServiceExports` may export a given `Service`, what would that mean?
- Determining whether or not a service is exported means searching
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

`ServiceExport` initially had no spec and seemed like it could just be
replaced with an annotation, e.g. `multicluster.kubernetes.io/export`. When a
service is found with the annotation, it would be considered marked for export
to the clusterset. The controller would then create `EndpointSlices` and an
`ServiceImport` in each cluster exactly as described above. Unfortunately,
`Service` does not have an extensible status and there is no way to represent
the state of the export on the annotated `Service`. We could extend
`Service.Status` to include `Conditions` and provide the flexibility we need,
but requiring changes to `Service` makes this a much more invasive proposal to
achieve the same result. As the use of a multi-cluster service implementation
would be an optional addon, it doesn't warrant a change to such a fundamental
resource.

### Other conflict resolution algorithms

When a service has a ServiceExport and a ServiceImport, we could have taken the
approach of favoring a "local truth" by giving a higher precedence to the locally
exported Service in the conflict resolution algorithm. This alternative
approach was not adopted, as in this KEP we favored global consistency across
the ClusterSet.

The conflict resolutions algorithm could also have been based on majority
instead of ServiceExport oldness. However, with this approach, we would have
to consider a tie breaking factor that could have also been based on age. This
would complicate the implementation of MCS-API and, most importantly, might be
more confusing for users. Having just one simple deciding factor based on
ServiceExport oldness makes resolving conflicts straightforward, and this
alternative conflict resolution algorithm could hinder this ease of use.

### Exporting labels/annotations from the Service/ServiceExport objects

`Service` and `ServiceExport` have labels and annotations which could be used during
export and propagated to the `ServiceImport`. However various tools such as kubectl or
ArgoCD add some labels and annotations which would then need to be actively
filtered to avoid any conflict. Filtering those labels and annotations is not
something easy and we chose to avoid this problem entirely by not using the metadata
object and adding dedicated fields in the spec of the `ServiceExport` resource.

Also if we were using the labels and annotations from the metadata of either the
`ServiceExport` or `Service` resources, it may be more confusing for users as it
would be the only fields present in both resources. For instance, should an
implementation merge the labels/annotations from both objects? Should it favor one?
Should it takes only from the `Service` object? With dedicated fields for labels
and annotations in the spec of the `ServiceExport` resource, it may becomes more
straightforward that each resource have their own labels and annotations in their
metadata and that the exported labels and annotations are from the dedicated
fields in the `ServiceExport` spec.

We also favored dedicated fields on the `ServiceExport` resource to allow for better
flexibility, as it will allow to export labels and annotations fully decorrelated
from the `Service` and `ServiceExport` metadata. More flexibility could also be
achieved with CEL expression on the `ServiceExport` at the cost of greater
complexity (managing CEL expressions on potentially many `ServiceExport` across clusters).

## Infrastructure Needed
<!--
Use this section if you need things from the project/SIG.  Examples include a
new subproject, repos requested, github details.  Listing these here allows a
SIG to get the process for these resources started right away.
-->
To facilitate consumption by kube-proxy, the MCS CRDs need to live in
kubernetes/staging. We will need a new k8s.io/multiclusterservices repo for
published MCS code.
