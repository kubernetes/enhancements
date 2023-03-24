<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [x] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [x] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [x] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [x] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [x] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [x] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [x] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-2149: ClusterId for ClusterSet identification

<!--
This is the title of your KEP. Keep it short, simple, and descriptive. A good
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
  - [Overview](#overview)
  - [User Stories](#user-stories)
    - [ClusterSet membership](#clusterset-membership)
    - [Joining or moving between ClusterSets](#joining-or-moving-between-clustersets)
    - [Multi-Cluster Services](#multi-cluster-services)
    - [Diagnostics](#diagnostics)
    - [Multi-tenant controllers](#multi-tenant-controllers)
  - [<code>ClusterProperty</code> CRD](#-crd)
  - [Well known properties](#well-known-properties)
    - [Property: <code>cluster.clusterset.k8s.io</code>](#property-)
      - [Uniqueness](#uniqueness)
      - [Lifespan](#lifespan)
      - [Contents](#contents)
      - [Consumers](#consumers)
      - [Notable scenarios](#notable-scenarios)
    - [Property: <code>clusterset.k8s.io</code>](#property--1)
      - [Lifespan](#lifespan-1)
      - [Contents](#contents-1)
      - [Consumers](#consumers-1)
  - [Additional Properties](#additional-properties)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Rationale behind the <code>ClusterProperty</code> CRD](#rationale-behind-the--crd)
  - [Implementing the <code>ClusterProperty</code> CRD and its admission controllers](#implementing-the--crd-and-its-admission-controllers)
    - [<code>cluster.clusterset.k8s.io ClusterProperty</code>](#)
    - [<code>clusterset.k8s.io ClusterProperty</code>](#-1)
  - [CRD upgrade path](#crd-upgrade-path)
    - [To CRD or not to CRD?](#to-crd-or-not-to-crd)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA criteria](#beta---ga-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

<!--
This section is incredibly important for producing high-quality, user-focused
documentation such as release notes or a development roadmap. It should be
possible to collect this information before implementation begins, in order to
avoid requiring implementors to split their attention between writing release
notes and implementing the feature itself. KEP editors and SIG Docs
should help to ensure that the tone and content of the `Summary` section is
useful for a wide audience.

A good summary is probably at least a paragraph in length.

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->
The new multi-cluster services API (see [KEP-1645](https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api)) 
expanded the ways clusters can communicate with each other and organized them into `ClusterSet`s, but as of now there is no way for a cluster to be uniquely identified 
in a Kubernetes-native way. This document by SIG-Multicluster proposes a standard 
for how cluster IDs should be stored and managed, based on concrete use cases 
discussed and observed in `ClusterSet` deployments. While existing implementations may not currently or plan to abide by this standard, future expansions to the Multi-Cluster API will be designed on top of this standard and existing MCS API implementations are encouraged to adopt it.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->
That there must be some way to identify individual clusters in a multi-cluster 
deployment has felt like a given to SIG-Multicluster; it has been discussed in
a broad sense previously ([see this doc](https://docs.google.com/document/d/1F__vEKeI41P7PPUCMM9PVPYY34pyrvQI5rbTJVnS5c4/edit?usp=sharing)), and was scoped 
down in response to actual observed use cases in the latest community discussion on
which this KEP is based ([doc](https://docs.google.com/document/d/1S0u6xzP2gcJKPipA6tBNDNuid76nVKeGhTk7PrCIuQY/edit?usp=sharing)). The motivation
of this KEP is to provide a flexible but useful baseline for clusterID that can
work with the known use cases (see the User Stories section). 

Existing implementations of the MCS API may have addressed the need for a cluster ID in their own ways, inconsistent with this current standard. It is the perspective of SIG-Multicluster that future additions to the MCS API will depend when necessary on the proposal laid out here, and existing implementations are encouraged to migrate any existing cluster ID assignment and storage mechanism to fit within the specifications of this KEP.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
* Propose a standard for how cluster identification metadata should be stored and 
managed as Kubernetes resources
* Define the standard to be strict enough to be useful in the following user stories:
  * Establish reliable coordinates for determining clusterset membership and identity of a cluster within its cluster set
  * Enable disambiguation of DNS names for multicluster Headless services with the same hostnames
  * Facilitate enrichment of log / event / metrics data with cluster id / set coordinates

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
* Define any characteristics of the system that tracks cluster IDs within a cluster (i.e. a cluster registry)
* Solve any problems without specific, tangible use cases (though we will leave room for extension).
* In particular, this KEP explicitly does not consider 
   * a cluster joining multiple ClusterSets
   * how or whether users should be able to specify aliases for cluster IDs and what they could be used for


## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### Overview
Each cluster in a ClusterSet will be assigned a unique identifier, that lives at least as long as that cluster is a member of the given ClusterSet, and is immutable for that same lifetime. This identifier will be stored in a new cluster-scoped `ClusterProperty` CR with the well known name `cluster.clusterset.k8s.io` that may be referenced by workloads within the cluster. The identifier must be a valid [RFC-1123](https://tools.ietf.org/html/rfc1123) DNS label, and may be created by an implementation dependent mechanism.

While a member of a ClusterSet, a cluster will also have an additional `clusterset.k8s.io ClusterProperty` which describes its current membership. This property must be present exactly as long as the cluster's membership in a ClusterSet lasts, and removed when the cluster is no longer a member.

More detail and examples of the uniqueness, lifespan, immutability, and content requirements for both the `cluster.clusterset.k8s.io ClusterProperty` and `clusterset.k8s.io ClusterProperty` are described further below. The goal of these requirements are to provide to the MCS API a cluster ID of viable usefulness to address known user stories without being too restrictive or prescriptive.

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### ClusterSet membership

I have some set of clusters working together and need a way to uniquely identify them within the system that I use to track membership, or determine if a given cluster is in a ClusterSet.

_For example, SIG-Cluster-Lifecycle's Cluster API subproject uses a management cluster to deploy resources to member workload clusters, but today member workload clusters do not have a way to identify their own management cluster or any interesting metadata about it, such as what cloud provider it is hosted on._

#### Joining or moving between ClusterSets

I want the ability to add a previously-isolated cluster to a ClusterSet, or to move a cluster from one ClusterSet to another and be aware of this change.

#### Multi-Cluster Services

I have a headless multi-cluster service deployed across clusters in my ClusterSet with similarly named pods in each cluster. I need a way to disambiguate each backend pod via DNS.

_For example, an exported headless service of services name `myservice` in namespace `test`,  backed by pods in two clusters with clusterIDs `clusterA` and `clusterB`, could be disambiguated by different DNS names following the pattern `<clusterID>.<svc>.<ns>.svc.clusterset.local`: `clusterA.myservice.test.svc.clusterset.local.` and `clusterB.myservice.test.svc.clusterset.local.`. This way the user can implement whatever load balancing they want (as is usually the case with headless services) by targeting each cluster's available backends directly._

#### Diagnostics

Clusters within my ClusterSet send logs/metrics to a common monitoring solution and I need to be able to identify the cluster from which a given set of events originated.

#### Multi-tenant controllers

My controller interacts with multiple clusters and needs to disambiguate between them to process its business logic.

_For example, [CAPN's virtualcluster project](https://github.com/kubernetes-sigs/cluster-api-provider-nested) is implementing a multi-tenant scheduler that schedules tenant namespaces only in certain parent clusters, and a separate syncer running in each parent cluster controller needs to compare the name of the parent cluster to determine whether the namespace should be synced. ([ref](https://github.com/kubernetes/enhancements/issues/2149#issuecomment-768486457))._


### `ClusterProperty` CRD

The `ClusterProperty` resource provides a way to store identification related, cluster scoped information for multi-cluster tools while creating flexibility for implementations. A cluster may have multiple `ClusterProperty`s, each holding a different identification related value. Each property contains the following information:

*   **Name** - a well known or custom name to identify the property.
*   **Value** - a property-dependent string, up 128k Unicode code points (see _Note_).

The schema for `ClusterProperty` is intentionally loose to support multiple forms of information, including arbitrary additional identification related properties described by users (see "Additional Properties", below), but certain well-known properties will add additional schema constraints, such as those described in the next section.

_Note: While prior Kubernetes API constructs containing arbitrary string values, such as annotations, are limited by a byte length, the OpenAPI validation this CRD depends on defines string length as Unicode code points at validation time. The encoded length of the string in bytes as observed on input or output by the user may vary depending on which of the valid JSON encodings are used (UTF-8, UTF-16, or UTF-32). Therefore, the value limit of 128k code points could take up to 512KB using the least space efficient allowable encoding, UTF-32, which uses 4 bytes per code point._


### Well known properties

The `ClusterProperty` CRD will support two specific properties under the well known names `cluster.clusterset.k8s.io` and `clusterset.k8s.io`. Being "well known" means that they must conform to the requirements described below, and therefore can be depended on by multi-cluster implementations to achieve use cases dependent on knowledge of a cluster's ID or ClusterSet membership.

The requirements below use the keywords **must, should,** and **may** purposefully in accordance with [RFC-2119](https://tools.ietf.org/html/rfc2119).


#### Property: `cluster.clusterset.k8s.io`

Contains a unique identifier for the containing cluster.


##### Uniqueness

*   The identifier **must** be unique within the ClusterSet to which its cluster belongs for the duration of the cluster’s membership.
*   The identifier **should** be unique beyond the ClusterSet within the scope of expected use.
*   The identifier **may** be globally unique beyond the scope of its ClusterSet.
*   The identifier **may** be unique beyond the span of its cluster’s membership and lifetime.


##### Lifespan

*   The identifier **must** exist and be immutable for the duration of a cluster’s membership in a ClusterSet, and as long as a `clusterset.k8s.io` property referring to that cluster in that ClusterSet exists.
*   The identifier **must** exist for the lifespan of a cluster.
*   The identifier **should** be immutable for the lifespan of a cluster.


##### Contents

*   The identifier **must** be a valid RFC-1123 DNS label [as described for object names in the Kubernetes docs](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names).
    *   Following the most restrictive standard naming constraint ensures maximum usefulness and portability.
    *   Can be used as a component in MCS DNS.
*   The identifier **may** be a human readable description of its cluster.


##### Consumers

*   **Must** be able to rely on the identifier existing, unmodified for the entire duration of its membership in a ClusterSet.
*   **Should** watch the `cluster.clusterset.k8s.io` property to handle potential changes if they live beyond the ClusterSet membership.
*   **May** rely on the existence of an identifier for clusters that do not belong to a ClusterSet so long as the implementation provides one.


##### Notable scenarios

**Renaming a cluster**: Since a `cluster.clusterset.k8s.io ClusterProperty` must be immutable for the duration of its *membership* in a given ClusterSet, the property contents can be "changed" by unregistering the cluster from the ClusterSet and reregistering it with the new name.

**Reusing cluster names**: Since an `cluster.clusterset.k8s.io ClusterProperty` has no restrictions on whether or not a ClusterProperty can be repeatable, if a cluster unregisters from a ClusterSet it is permitted under this standard to rejoin later with the same `cluster.clusterset.k8s.io ClusterProperty` it had before. Similarly, a *different* cluster could join a ClusterSet with the same `cluster.clusterset.k8s.io ClusterProperty` that had been used by another cluster previously, as long as both do not have membership in the same ClusterSet at the same time. Finally, two or more clusters may have the same `cluster.clusterset.k8s.io ClusterProperty` concurrently (though they **should** not; see "Uniqueness" above) *as long as* they both do not have membership in the same ClusterSet.

#### Property: `clusterset.k8s.io`

Contains an identifier that relates the containing cluster to the ClusterSet in which it belongs.


##### Lifespan

*   The identifier **must** exist and be immutable for the duration of a cluster’s membership in a ClusterSet.
*   The identifier **must not** exist when the cluster is not a member of a ClusterSet.


##### Contents

*   The identifier **must** associate the cluster with a ClusterSet.
*   The identifier **may** be either unique or shared by members of a ClusterSet.


##### Consumers

*   **Must** be able to rely on the identifier existing, unmodified for the entire duration of its membership in a ClusterSet.
*   **Should** watch the clusterset property to detect the span of a cluster’s membership in a ClusterSet.


### Additional Properties

Implementers are free to add additional properties as they see fit, so long as they do not conflict with the well known properties _and_ utilize a suffix. The following suffixes are reserved for Kubernetes and related projects: `.k8s.io`, `.kubernetes.io`. For example, an implementation may utilize the `Kind` `ClusterProperty` to store objects with the name `fingerprint.coolmcsimplementation.com` but not `fingerprint.k8s.io` and not simply `fingerprint`.


### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Rationale behind the `ClusterProperty` CRD

This proposal suggests a CRD composed of objects all of the same `Kind` `ClusterProperty`, and that are distinguished using certain well known values in their `metadata.name` fields. This design avoids cluster-wide singleton `Kind`s for each property, reduces access competition for the same metadata by making each property its own resource (instead of all in one), allows for RBAC to be applied in a targeted way to individual properties, and supports the user prerogative to store other simple metadata in one centralized CRD by creating CRs of the same `Kind` `ClusterProperty` but with their own names.

Storing arbitrary facts about a cluster can be implemented in other ways. For example, Cluster API subproject stopgapped their need for cluster name metadata by leveraging the existing `Node` `Kind` and storing metadata there via annotations, such as `cluster.x-k8s.io/cluster-name` ([ref](https://github.com/kubernetes-sigs/cluster-api/pull/4048)). While practical for their case, this KEP avoids adding cluster-level info as annotations on child resources so as not to be dependent on a child resource's existence, to avoid issues maintaining parity across multiple resources of the same `Kind` for identical metadata, and maintain RBAC separation between the cluster-level metadata and the child resources. Even within the realm of implementing as a CRD, the API design could focus on distinguishing each fact by utilizing different `spec.Type`s (as `Service` objects do e.g. `spec.type=ClusterIP` or `spec.type=ExternalName`), or even more strictly, each as a different `Kind`.  The former provides no specific advantages since multiple differently named properties for the same fact are unnecessary, and is less expressive to query (it is easier to query by name directly like `kubectl get clusterproperties cluster.clusterset.k8s.io`). The latter would result in the proliferation of cluster-wide singleton `Kind` resources, and be burdensome for users to create their own custom properties.


### Implementing the `ClusterProperty` CRD and its admission controllers

#### `cluster.clusterset.k8s.io ClusterProperty`

The actual implementation to select and store the identifier of a given cluster could occur local to the cluster. It does not necessarily ever need to be deleted, particularly if the identifier selection mechanism chooses an identifier that is compliant with this specification's most broad restrictions -- namely, being immutable for a cluster's lifetime and unique beyond just the scope of the cluster's membership. A recommended option that meets these broad restrictions is a cluster's kube-system.uuid. 

That being said, for less stringent identifiers, for example a user-specified and human-readable value, a given `cluster.clusterset.k8s.io ClusterProperty` may need to change if an identical identifier is in use by another member of the ClusterSet it wants to join. It is likely this would need to happen outside the cluster-local boundary; for example, whatever manages memberships would likely need to deny the incoming cluster, and potentially assign (or prompt the cluster to assign itself) a new ID.

Since this KEP does not formally mandate that the cluster ID *must* be immutable for the lifetime of the cluster, only for the lifetime of its membership in a ClusterSet, any dependent tooling explicitly *cannot* assume the `cluster.clusterset.k8s.io ClusterProperty` for a given cluster will stay constant on its own merit. For example, log aggregation of a given cluster ID based on this property should only be trusted to be referring to the same cluster for as long as it has one ClusterSet membership; similarly, controllers whose logic depends on distinguishing clusters by cluster ID can only trust this property to disambiguate the same cluster for as long as the cluster has one ClusterSet membership.

Despite this flexibility in the KEP, clusterIDs may still be useful before ClusterSet membership needs to be established; again, particularly if the implementation chooses the broadest restrictions regarding immutability and uniqueness. Therefore, having a controller that initializes it early in the lifecycle of the cluster, and possibly as part of cluster creation, may be a useful place to implement it, though within the bounds of this KEP that is not strictly necessary.

The most common discussion point within the SIG regarding whether an implementation should favor a UUID or a human-readable clusterID string is when it comes to DNS. Since DNS names are originally intended to be a human readable technique of address, clunky DNS names composed from long UUIDs seems like an anti-pattern, or at least unfinished. While some extensions to this spec have been discussed as ways to leverage the best parts of both (ex. using labels on the `cluster.clusterset.k8s.io ClusterProperty` to store aliases for DNS), an actual API specification to allow for this is outside the scope of this KEP at this time (see the Non-Goals section).

```
# An example object of `cluster.clusterset.k8s.io ClusterProperty` 
# using a kube-system ns uuid as the id value (recommended above):

apiVersion: about.k8s.io/v1
kind: ClusterProperty
metadata:
  name: cluster.clusterset.k8s.io
spec:
  value: 721ab723-13bc-11e5-aec2-42010af0021e
```

```
# An example object of `cluster.clusterset.k8s.io ClusterProperty` 
# using a human-readable string as the id value:

apiVersion: about.k8s.io/v1
kind: ClusterProperty
metadata:
  name: cluster.clusterset.k8s.io
spec:
  value: cluster-1
```

#### `clusterset.k8s.io ClusterProperty`

A cluster in a ClusterSet is expected to be authoritatively associated with that ClusterSet by an external process and storage mechanism with a purview above the cluster local boundary, whether that is some form of a cluster registry or just a human running kubectl. (The details of any specific mechanism is out of scope for the MCS API and this KEP -- see the Non-Goals section.) Mirroring this information in the cluster-local `ClusterProperty` CRD will necessarily need to be managed above the level of the cluster itself, since the properties of `clusterset.k8s.io` extend beyond the boundaries of a single cluster, and will likely be something that has access to whatever cluster registry-esque concept is implemented for that multicluster setup. It is expected that the mcs-controller ([as described in the MCS API KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#proposal)), will act as an admission controller to verify individual objects of this property.

Because there are obligations of the `cluster.clusterset.k8s.io ClusterProperty` that are not meanigfully verifiable until a cluster tries to join a ClusterSet and set its `clusterset.k8s.io ClusterProperty`, the admission controller responsible for setting a `clusterset.k8s.io ClusterProperty` will need the ability to reject such an attempt when it is invalid, and alert `[UNRESOLVED]` or possibly affect changes to that cluster's `cluster.clusterset.k8s.io ClusterProperty` to make it valid `[/UNRESOLVED]`. Two symptomatic cases of this would be:

1. When a cluster with a given `cluster.clusterset.k8s.io ClusterProperty` tries to join a ClusterSet, but a cluster with that same `cluster.clusterset.k8s.io ClusterProperty` appears to already be in the set.
2. When a cluster that does not have a `cluster.clusterset.k8s.io ClusterProperty` tries to join a ClusterSet.

In situations like these, the admission controller will need to fail to add the invalid cluster to the ClusterSet by refusing to set its `clusterset.k8s.io ClusterProperty`, and surface an error that is actionable to make the property valid.

```
# An example object of `clusterset.k8s.io ClusterProperty`:

apiVersion: about.k8s.io/v1
kind: ClusterProperty
metadata:
  name: clusterset.k8s.io
spec:
  value: environ-1
```

### CRD upgrade path

#### To CRD or not to CRD?

_That is the question._

While this document has thus far referred to the `ClusterProperty` resource as being implemented as a CRD, another implementation point of debate has been whether this belongs in the core Kubernetes API, particularly the `cluster.clusterset.k8s.io ClusterProperty`. A dependable cluster ID or cluster name has previously been discussed in other forums (such as [this SIG-Architecture thread](https://groups.google.com/g/kubernetes-sig-architecture/c/mVGobfD4TpY/m/nkdbkX1iBwAJ) from 2018, or, as mentioned above, the [Cluster API subproject](https://github.com/kubernetes-sigs/cluster-api/issues/4044) which implemented [their own solution](https://github.com/kubernetes-sigs/cluster-api/pull/4048).) It is the opinion of SIG-Multicluster that the function of the proposed `ClusterProperty` CRD is of broad utility and becomes more useful the more ubiquitous it is, not only in multicluster set ups.

This has led to the discussion of whether or not we should pursue adding this resource type not as a CRD associated with SIG-Multicluster, but as a core Kubernetes API implemented in `kubernetes/kubernetes`. A short pro/con list is enclosed at the end of this section.

One effect of that decision is related to the upgrade path. Implementing this resource only in k/k will restrict the types of clusters that can use cluster ID to only ones on the target version (or above) of Kubernetes, unless a separate backporting CRD is made available to them. At that point, with two install options, other issues arise. How do backported clusters deal with migrating their CRD data to the core k/k objects during upgrade -- will the code around the formal k/k implementation be sensitive to the backport CRD and migrate itself? Will users have to handle upgrades in a bespoke manner?

|                       | CRD                                                                              | k/k                                               |
|-----------------------|----------------------------------------------------------------------------------|---------------------------------------------------|
| Ubiquitous | No                                                                     | Yes                                        |
| Default always set | No                                                                     | Yes                                        |
| Deployment            | Must be installed by the cluster lifecycle management, or as a manual setup step | In every cluster over target milestone            |
| Schema validation     | OpenAPI v3 validation                                                    | Can use the built-in Kubernetes schema validation |
| Blockers     | Official API review if using *.k8s.io                                                    | Official API review |
| Conformance testing     | Not possible now, and no easy path forward                                                   | Standard |

**In the end, SIG-Multicluster discussed this with SIG-Architecture and it was decided to stick with the plan to use a CRD.** Notes from this conversation are in the [SIG-Architecture meeting agenda](https://docs.google.com/document/d/1BlmHq5uPyBUDlppYqAAzslVbAO8hilgjqZUTaNXUhKM/preview) for 3/25/2021. A graduation criteria set for Alpha->Beta stage to fully immortalize this decision is intended to be the last chance to consider including this design in k/k or not.


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

This KEP proposes and out-of-tree CRD that is not expected to integrate with any of the Kubernetes CI infrastructure. In addition, it explicitly provides only the CRD definition and generated clients for use by third party implementers, and does not provide a controller or any other binary with business logic to test. For these reasons, we only expect to provide unit tests for a dummy controller to confirm that the generated CRD can be installed and the generated clients can be instantiated. Today those tests are available [here](https://github.com/kubernetes-sigs/about-api/blob/master/clusterproperty/controllers/suite_test.go).

However, similar to other out-of-tree CRDs that serve third party implementers, such as Gateway API and MCS API, there is rationale for the project to provide conformance tests for implementers to use to confirm they adhere to the restrictions set forth in this KEP that are not otherwise enforced by the CRD definition; in thise case, the constraints defined on the well-known properties `clusterset.k8s.io` and `cluster.clusterset.k8s.io`. Providing these tests are not considered blocking graduation requirements for the maturity level of this API.

These tests will be provided in such a way that implementers can expose one or more clusters that have the About API CRD installed in them, and run a series of tests that confirms any well-known properties stored in those clusters' `ClusterProperty` objects conform to the constraints in [Well known properties](#well-known-properties). 

### Graduation Criteria

#### Alpha -> Beta Graduation

- Determine if an `cluster.clusterset.k8s.io ClusterProperty` be strictly a valid DNS label, or is allowed to be a subdomain.
- To CRD or not to CRD (see section above)
- Determine if CRD implementation should use CEL validation to limit byte length instead of code points; this would make it only compatible with 1.23+ where CEL validation is behind a feature gate for alpha.

#### Beta -> GA criteria

- At least one headless implementation using clusterID for MCS DNS

### Upgrade / Downgrade Strategy

Any changes to the API definition will follow the official Kubernetes API groups and versioning guidance [here](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-groups-and-versioning) and [here](https://kubernetes.io/docs/reference/using-api/#api-versioning). In short, the API will be provided in order through `v1alphaX`, `v1betaX`, to `v1`, where compatibility will be preserved from `v1beta1` and onwards; clients will be expected to eventually migrate to the `v1` implementation of the API as the prior versions are deprecated.

### Version Skew Strategy

As a CRD, this API is dependent on any changes in the version and compatibility of the CRD feature itself on which it is built. As the CRD system is in `v1` as of Kubernetes 1.14, and the Kubernetes versioning guarantees `v1` APIs to be maintained through the Kubernetes major release, and as the About API does not depend on any new features of the CRD system since then, there is no expected coordination required with any core Kubernetes components until and unless Kubernetes proceeds to version 2.X.

This CRD /is/ a direct dependency of the MCS API and any mcs-controller implementation as defined by that KEP. As discussed later in the PRR, it is expected that the mcs-controller (or any other controller taking this CRD as its dependency) would manage the lifecycle of this CRD, including any version skew.

As also mentioned below, we are aware that other features (in or out of tree) may want to use this CRD (as debated in "To CRD or Not to CRD" section, above) but we believe it is in the scope of those future features to assess the impact of this CRD's version strategy on their component's version skew and their feature's stability if they do.

## Production Readiness Review Questionnaire

**NOTE: While this KEP represents only the schema of a CRD that will be implemented
out-of-tree and maintained separately from core Kubernetes, a best effort on the PRR 
questionnaire is enclosed below.**

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness/README.md.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.

-->

### Feature Enablement and Rollback

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism:
      - This feature is independently installed via a CRD hosted on the kubernetes-sigs Github.
    - Will enabling / disabling the feature require downtime of the control
      plane?
      - No
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node?
      - No

* **Does enabling the feature change any default behavior?**
  _Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here._
  - No default Kubernetes behavior is currently planned to be based on this feature; it is 
  designed to be used by the separately installed, out-of-tree, MCS controller. That being said,
  we are of the opinion that future features (default or not) may want to use this CRD (as debated
  in "To CRD or Not to CRD" section, above) but we believe it is in the scope of those future features
  to assess the impact of requiring CRD bootstrapping has on their feature stability if they do.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  _Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?)._
    - Yes, as this feature only describes a CRD, it can most directly be disabled by uninstalling the CRD. 
  However in practice it is expected that the bootstrapping of this CRD and the management of the well known property CRs themselves will be managed 
  by the mcs-controller, and the recommended way to disable this feature will be to disable the mcs-controller.
  It is expected the mcs-controller will be responsible for detecting the presence
  of this CRD to gracefully fail or otherwise raise error messages that can be acted on if the
  CRD has been disabled by a mechanism other than the mcs-controller's lifecycle management of the CRD.

* **What happens if we reenable the feature if it was previously rolled back?**
   - Purely from this KEP's standpoint, feature reenablement - namely, reinstallation of the CRD - will
  do no more than reinstall the CRD schema. In relation to the expected lifecycle manager of this CRD (the mcs-controller), it is expected that on reenablement of the mcs-controller it will reinstall the CRD, will reestablish lifecycle management of the well known properties it is dependent on, including re-creating any relevant CRs.

* **Are there any tests for feature enablement/disablement?**
  _The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified._
    - As a dependency only for an out-of-tree component, there will not be e2e tests for feature enablement/disablement of
   this CRD in core Kubernetes, but e2e tests for this can be implemented in the 
   [kubernetes-sigs/mcs-api repo](https://github.com/kubernetes-sigs/mcs-api) where a basic mcs-controller 
   implementation lives. In reality, multiple mcs-controller implementations are expected to be produced outside of core
   and these production-ready mcs-controllers are responsible for their own e2e testing.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  
   CRDs themselves are Kubernetes objects, and can fail to be applied if the schema definition is corrupt or incompatible with the CustomResourceDefinition schema. Unit tests and manual tests continuously confirm that as the built CRD yaml produced by this project is valid against the stable `v1 CustomResourceDefinition`. (It also could fail if the CRD is applied to a version of Kubernetes that does not have the CRD system is used (<1.14), or the API Server is unreachable, but these are both considered catastrophic failures out of scope of this KEP.) 
  
  Ultimately, the failure of a rollout of any CRD has the potential to disrupt all features or workloads that depend on it. Watches in controllers will fail to receive updates as the client would fail to find the CRD; a concrete known example for this CRD, the CoreDNS multicluster DNS plugin, would fail to program new DNS records and CoreDNS will answer SERVFAIL to any request made for a Kubernetes record that has not yet been synchronized. Features or workloads that depend on this CRD should plan to manage the lifecycle of this CRD or to provide transparent failure modes if the CRD is not present.

* **What specific metrics should inform a rollback?**

  Metrics should be configured using a metrics solutions implementing the [Custom Metrics API](https://kubernetes.io/docs/tasks/debug/debug-cluster/resource-usage-monitoring/#full-metrics-pipeline), for example, the [metrics plugin for Custom Resources in kube-state-metrics](https://github.com/kubernetes/kube-state-metrics/blob/main/docs/customresourcestate-metrics.md). Kubernetes does not provide default metrics for CRDs.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
Unit tests and manual tests confirm that the CRD is capable of being uninstalled and reinstalled.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

  Kubernetes does not provide default metrics for CRDs so an operator would need to depend on custom metrics, or filter 404s from Kubernetes API server against this CRD.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**

  N/A: This KEP does not propose a service, only leverages the existing Kuebernetes API service and CRD extension mechanism.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  N/A: This KEP does not propose a service, only leverages the existing Kuebernetes API service and CRD extension mechanism.

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  
  Default metrics for CRDs in general for number of requests by workload source would improve 

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
    This feature depends only on the CustomResourceDefinition v1 in Kubernetes API server, available in Kubernetes versions 1.14+.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

  Installing the CRD will require a single API call to POST the new `CustomResourceDefinition` resource that represents it.

* **Will enabling / using this feature result in introducing new API types?**

  Yes, installing the CRD introduces the cluster-scoped `ClusterProperty` Kind. As there is no related service proposed as part of this KEP, there are no specific limits on the supported number of objects per cluster outside of Kubernetes API server storage limits. 

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

  No.

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**

  Besides the trivial single `CustomResourceDefinition` required to install this CRD, no other size or count of existing API objects will be affected by this KEP.

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**

  No, this KEP does not affect any of the operations covered by existing SLIs/SLOs, particularly since CustomResourceDefinitions are excluded from those SLOs.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  This CRD will utilize the validation mechanism provided by the CRD extension for validation of structural schemas of CRDs which requires some amount of resources to validate on create or update of a CR. However, the number of expected resources (2 as of this KEP) and their rate of change (related to clusterset membership changes, itself expected to be a human decision and rarely changing state) is expected to be trivial.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

  This KEP itself proposes a CRD applied to the API server; if the API server and/or etcd is unavailable, so is this CRD. Features dependent on this CRD must assess the impact of this CRD's availability on their component's availability. Most concretely today, components of the mcs-controller are expected to serve as an admission controller to this CRD or are dependent on this CRD to program DNS. If the API server and/or etcd is unavailable, those controllers will be unable to update a cluster's ClusterProperty data regarding its well-known properties as part of a ClusterSet, or to program any updates to DNS, respectively.

* **What are other known failure modes?**

  - [CRD cannot be installed]
    - Detection: Custom metrics or dependent feature metrics; increased 404 rate on Kube API server for the CRD.
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Warning and above, as this is the level that 404s against the CRD will be seen.
    - Testing: Unit tests against generated CRD schema installation and usage of generated client.

* **What steps should be taken if SLOs are not being met to determine the problem?**

  N/A: SLOs are not defined as there is no service provided by this KEP.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
