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
  - [<code>ClusterClaim</code> CRD](#-crd)
  - [Well known claims](#well-known-claims)
    - [Claim: <code>id.k8s.io</code>](#claim-)
      - [Uniqueness](#uniqueness)
      - [Lifespan](#lifespan)
      - [Contents](#contents)
      - [Consumers](#consumers)
      - [Notable scenarios](#notable-scenarios)
    - [Claim: <code>clusterset.k8s.io</code>](#claim--1)
      - [Lifespan](#lifespan-1)
      - [Contents](#contents-1)
      - [Consumers](#consumers-1)
  - [Additional Claims](#additional-claims)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Implementing the <code>ClusterClaim</code> CRD](#implementing-the--crd)
    - [<code>id.k8s.io ClusterClaim</code>](#)
    - [<code>clusterset.k8s.io ClusterClaim</code>](#-1)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
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
  * Enable cluster-granularity DNS names for multicluster services
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
Each cluster in a ClusterSet will be assigned a unique identifier, that lives at least as long as that cluster is a member of the given ClusterSet, and is immutable for that same lifetime. This identifier will be stored in a new cluster-scoped `ClusterClaim` CR with the well known name `id.k8s.io` that may be referenced by workloads within the cluster. The identifier must be a valid [RFC-1123](https://tools.ietf.org/html/rfc1123) DNS label, and may be created by an implementation dependent mechanism.

While a member of a ClusterSet, a cluster will also have an additional `clusterset.k8s.io ClusterClaim` which describes its current membership. This claim must be present exactly as long as the cluster's membership in a ClusterSet lasts, and removed when the cluster is no longer a member.

More detail and examples of the uniqueness, lifespan, immutability, and content requirements for both the `id.k8s.io ClusterClaim` and `clusterset.k8s.io ClusterClaim` are described further below. The goal of these requirements are to provide to the MCS API a cluster ID of viable usefulness to address known user stories without being too restrictive or prescriptive.

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


### `ClusterClaim` CRD
  ```
  <<[UNRESOLVED]>>
  The actual name of the CRD is not finalized and is provisionally titled `ClusterClaim` for the remainder of this document.
  <<[/UNRESOLVED]>>
  ```

The `ClusterClaim` resource provides a way to store identification related, cluster scoped information for multi-cluster tools while creating flexibility for implementations. A cluster may have multiple `ClusterClaim`s, each holding a different identification related value. Each claim contains the following information:

*   **Name** - a well known or custom name to identify the claim.
*   **Value** - a claim-dependent string, up to 128 KB.

The schema for `ClusterClaim` is intentionally loose to support multiple forms of information, including arbitrary additional identification related claims described by users (see "Additional Claims", below), but certain well-known claims will add additional schema constraints, such as those described in the next section.


### Well known claims

The `ClusterClaim` CRD will support two specific claims under the well known names `id.k8s.io` and `clusterset.k8s.io`. Being "well known" means that they must conform to the requirements described below, and therefore can be depended on by multi-cluster implementations to achieve use cases dependent on knowledge of a cluster's ID or ClusterSet membership.

The requirements below use the keywords **must, should,** and **may** purposefully in accordance with [RFC-2119](https://tools.ietf.org/html/rfc2119).


#### Claim: `id.k8s.io`

Contains a unique identifier for the containing cluster.


##### Uniqueness

*   The identifier **must** be unique within the ClusterSet to which its cluster belongs for the duration of the cluster’s membership.
*   The identifier **should** be unique beyond the ClusterSet within the scope of expected use.
*   The identifier **may** be globally unique beyond the scope of its ClusterSet.
*   The identifier **may** be unique beyond the span of its cluster’s membership and lifetime.


##### Lifespan

*   The identifier **must** exist and be immutable for the duration of a cluster’s membership in a ClusterSet, and as long as a `clusterset.k8s.io` claim referring to that cluster in that ClusterSet exists.
*   The identifier **must** exist for the lifespan of a cluster.
*   The identifier **should** be immutable for the lifespan of a cluster.


##### Contents

*   The identifier **must** be a valid RFC-1123 DNS label [as described for object names in the Kubernetes docs](https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#dns-label-names).
    *   Following the most restrictive standard naming constraint ensures maximum usefulness and portability.
    *   Can be used as a component in MCS DNS.
*   The identifier **may** be a human readable description of its cluster.


##### Consumers

*   **Must** be able to rely on the identifier existing, unmodified for the entire duration of its membership in a ClusterSet.
*   **Should** watch the `id.k8s.io` claim to handle potential changes if they live beyond the ClusterSet membership.
*   **May** rely on the existence of an identifier for clusters that do not belong to a ClusterSet so long as the implementation provides one.


##### Notable scenarios

**Renaming a cluster**: Since a `id.k8s.io ClusterClaim` must be immutable for the duration of its *membership* in a given ClusterSet, the claim contents can be "changed" by unregistering the cluster from the ClusterSet and reregistering it with the new name.

**Reusing cluster names**: Since an `id.k8s.io ClusterClaim` has no restrictions on whether or not a ClusterClaim can be repeatable, if a cluster unregisters from a ClusterSet it is permitted under this standard to rejoin later with the same `id.k8s.io ClusterClaim` it had before. Similarly, a *different* cluster could join a ClusterSet with the same `id.k8s.io ClusterClaim` that had been used by another cluster previously, as long as both do not have membership in the same ClusterSet at the same time. Finally, two or more clusters may have the same `id.k8s.io ClusterClaim` concurrently (though they **should** not; see "Uniqueness" above) *as long as* they both do not have membership in the same ClusterSet.

#### Claim: `clusterset.k8s.io`

Contains an identifier that relates the containing cluster to the ClusterSet in which it belongs.


##### Lifespan

*   The identifier **must** exist and be immutable for the duration of a cluster’s membership in a ClusterSet.
*   The identifier **must not** exist when the cluster is not a member of a ClusterSet.


##### Contents

*   The identifier **must** associate the cluster with a ClusterSet.
*   The identifier **may** be either unique or shared by members of a ClusterSet.


##### Consumers

*   **Must** be able to rely on the identifier existing, unmodified for the entire duration of its membership in a ClusterSet.
*   **Should** watch the clusterset claim to detect the span of a cluster’s membership in a ClusterSet.


### Additional Claims

Implementers are free to add additional claims as they see fit, so long as they do not conflict with the well known claims. `*.k8s.io`, `*.kubernetes.io`, and `sigs.k8s.io` claims are reserved for Kubernetes and related projects.


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

### Rationale behind the `ClusterClaim` CRD

This proposal suggests a CRD composed of objects all of the same `Kind` `ClusterClaim`, and that are distinguished using certain well known values in their `metadata.name` fields. This design avoids cluster-wide singleton `Kind`s for each claim, reduces access competition for the same metadata by making each claim its own resource (instead of all in one), allows for RBAC to be applied in a targeted way to individual claims, and supports the user prerogative to store other simple metadata in one centralized CRD by creating CRs of the same `Kind` `ClusterClaim` but with their own names.

Storing arbitrary facts about a cluster can be implemented in other ways. For example, Cluster API subproject stopgapped their need for cluster name metadata by leveraging the existing `Node` `Kind` and storing metadata there via annotations, such as `cluster.x-k8s.io/cluster-name` ([ref](https://github.com/kubernetes-sigs/cluster-api/pull/4048)). While practical for their case, this KEP avoids adding cluster-level info as annotations on child resources so as not to be dependent on a child resource's existence, to avoid issues maintaining parity across multiple resources of the same `Kind` for identical metadata, and maintain RBAC separation between the cluster-level metadata and the child resources. Even within the realm of implementing as a CRD, the API design could focus on distinguishing each fact by utilizing different `spec.Type`s (as `Service` objects do e.g. `spec.type=ClusterIP` or `spec.type=ExternalName`), or even more strictly, each as a different `Kind`.  The former provides no specific advantages since multiple differently named claims for the same fact are unnecessary, and is less expressive to query (it is easier to query by name directly like `kubectl get clusterclaims id.k8s.io`). The latter would result in the proliferation of cluster-wide singleton `Kind` resources, and be burdensome for users to create their own custom claims.


### Implementing the `ClusterClaim` CRD and its admission controllers

#### `id.k8s.io ClusterClaim`

The actual implementation to select and store the identifier of a given cluster could occur local to the cluster. It does not necessarily ever need to be deleted, particularly if the identifier selection mechanism chooses an identifier that is compliant with this specification's most broad restrictions -- namely, being immutable for a cluster's lifetime and unique beyond just the scope of the cluster's membership. A recommended option that meets these broad restrictions is a cluster's kube-system.uuid. 

That being said, for less stringent identifiers, for example a user-specified and human-readable value, a given `id.k8s.io ClusterClaim` may need to change if an identical identifier is in use by another member of the ClusterSet it wants to join. It is likely this would need to happen outside the cluster-local boundary; for example, whatever manages memberships would likely need to deny the incoming cluster, and potentially assign (or prompt the cluster to assign itself) a new ID.

Since this KEP does not formally mandate that the cluster ID *must* be immutable for the lifetime of the cluster, only for the lifetime of its membership in a ClusterSet, any dependent tooling explicitly *cannot* assume the `id.k8s.io ClusterClaim` for a given cluster will stay constant on its own merit. For example, log aggregation of a given cluster ID based on this claim should only be trusted to be referring to the same cluster for as long as it has one ClusterSet membership; similarly, controllers whose logic depends on distinguishing clusters by cluster ID can only trust this claim to disambiguate the same cluster for as long as the cluster has one ClusterSet membership.

Despite this flexibility in the KEP, clusterIDs may still be useful before ClusterSet membership needs to be established; again, particularly if the implementation chooses the broadest restrictions regarding immutability and uniqueness. Therefore, having a controller that initializes it early in the lifecycle of the cluster, and possibly as part of cluster creation, may be a useful place to implement it, though within the bounds of this KEP that is not strictly necessary.

The most common discussion point within the SIG regarding whether an implementation should favor a UUID or a human-readable clusterID string is when it comes to DNS. Since DNS names are originally intended to be a human readable technique of address, clunky DNS names composed from long UUIDs seems like an anti-pattern, or at least unfinished. While some extensions to this spec have been discussed as ways to leverage the best parts of both (ex. using labels on the `id.k8s.io ClusterClaim` to store aliases for DNS), an actual API specification to allow for this is outside the scope of this KEP at this time (see the Non-Goals section).

```
# An example object of `id.k8s.io ClusterClaim` 
# using a kube-system ns uuid as the id value (recommended above):

apiVersion: multicluster.k8s.io/v1
kind: ClusterClaim
metadata:
  name: id.k8s.io
spec:
  value: 721ab723-13bc-11e5-aec2-42010af0021e
```

```
# An example object of `id.k8s.io ClusterClaim` 
# using a human-readable string as the id value:

apiVersion: multicluster.k8s.io/v1
kind: ClusterClaim
metadata:
  name: id.k8s.io
spec:
  value: cluster-1
```

#### `clusterset.k8s.io ClusterClaim`

A cluster in a ClusterSet is expected to be authoritatively associated with that ClusterSet by an external process and storage mechanism with a purview above the cluster local boundary, whether that is some form of a cluster registry or just a human running kubectl. (The details of any specific mechanism is out of scope for the MCS API and this KEP -- see the Non-Goals section.) Mirroring this information in the cluster-local `ClusterClaim` CRD will necessarily need to be managed above the level of the cluster itself, since the properties of `clusterset.k8s.io` extend beyond the boundaries of a single cluster, and will likely be something that has access to whatever cluster registry-esque concept is implemented for that multicluster setup. It is expected that the mcs-controller ([as described in the MCS API KEP](https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api#proposal)), will act as an admission controller to verify individual objects of this claim.

Because there are obligations of the `id.k8s.io ClusterClaim` that are not meanigfully verifiable until a cluster tries to join a ClusterSet and set its `clusterset.k8s.io ClusterClaim`, the admission controller responsible for setting a `clusterset.k8s.io ClusterClaim` will need the ability to reject such an attempt when it is invalid, and alert `[UNRESOLVED]` or possibly affect changes to that cluster's `id.k8s.io ClusterClaim` to make it valid `[/UNRESOLVED]`. Two symptomatic cases of this would be:

1. When a cluster with a given `id.k8s.io ClusterClaim` tries to join a ClusterSet, but a cluster with that same `id.k8s.io ClusterClaim` appears to already be in the set.
2. When a cluster that does not have a `id.k8s.io ClusterClaim` tries to join a ClusterSet.

In situations like these, the admission controller will need to fail to add the invalid cluster to the ClusterSet by refusing to set its `clusterset.k8s.io ClusterClaim`, and surface an error that is actionable to make the claim valid.

```
# An example object of `clusterset.k8s.io ClusterClaim`:

apiVersion: multicluster.k8s.io/v1
kind: ClusterClaim
metadata:
  name: clusterset.k8s.io
spec:
  value: environ-1
```

### CRD upgrade path

#### To CRD or not to CRD?

_That is the question._

`[UNRESOLVED] Must resolve before alpha`

> While this document has thus far referred to the `ClusterClaim` resource as being implemented as a CRD, another implementation point of debate has been whether this belongs in the core Kubernetes API, particularly the `id.k8s.io ClusterClaim`. A dependable cluster ID or cluster name has previously been discussed in other forums (such as [this SIG-Architecture thread](https://groups.google.com/g/kubernetes-sig-architecture/c/mVGobfD4TpY/m/nkdbkX1iBwAJ) from 2018, or, as mentioned above, the [Cluster API subproject](https://github.com/kubernetes-sigs/cluster-api/issues/4044) which implemented [their own solution](https://github.com/kubernetes-sigs/cluster-api/pull/4048).) It is the opinion of SIG-Multicluster that the function of the proposed `ClusterClaim` CRD is of broad utility and becomes more useful the more ubiquitous it is, not only in multicluster set ups.

> This has led to the discussion of whether or not we should pursue adding this resource type not as a CRD associated with SIG-Multicluster, but as a core Kubernetes API implemented in `kubernetes/kubernetes`. A short pro/con list is enclosed at the end of this section.

> One effect of that decision is related to the upgrade path. Implementing this resource only in k/k will restrict the types of clusters that can use cluster ID to only ones on the target version (or above) of Kubernetes, unless a separate backporting CRD is made available to them. At that point, with two install options, other issues arise. How do backported clusters deal with migrating their CRD data to the core k/k objects during upgrade -- will the code around the formal k/k implementation be sensitive to the backport CRD and migrate itself? Will users have to handle upgrades in a bespoke manner?

|                       | CRD                                                                              | k/k                                               |
|-----------------------|----------------------------------------------------------------------------------|---------------------------------------------------|
| Built-in / ubiquitous | Unlikely (?)                                                                     | Likely (?)                                        |
| Deployment            | Must be installed by the cluster lifecycle management, or as a manual setup step | In every cluster over target milestone            |
| Schema validation     | Can use OpenAPI v3 validation                                                    | Can use the built-in Kubernetes schema validation |
| Blockers     | Making a sigs-repo                                                    | Official API review |
| Conformance testing     | Not possible now, and no easy path forward                                                   | Standard |

`[/UNRESOLVED]`


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

### Graduation Criteria

#### Alpha -> Beta Graduation

- Determine if an `id.k8s.io ClusterClaim` be strictly a valid DNS label, or is allowed to be a subdomain.
- To CRD or not to CRD (see section above)

#### Beta -> GA criteria

- At least one headless implementation using clusterID for MCS DNS

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
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha -> Beta Graduation

- Gather feedback from developers and surveys
- Complete features A, B, C
- Tests are in Testgrid and linked in KEP

#### Beta -> GA Graduation

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

#### Removing a Deprecated Flag

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag

**For non-optional features moving to GA, the graduation criteria must include 
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

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

## Production Readiness Review Questionnaire

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
  - [ ] Other
    - Describe the mechanism:
    - Will enabling / disabling the feature require downtime of the control
      plane?
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

* **Does enabling the feature change any default behavior?**
  Any change of default behavior may be surprising to users or break existing
  automations, so be extremely careful here.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Also set `disable-supported` to `true` or `false` in `kep.yaml`.
  Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).

* **What happens if we reenable the feature if it was previously rolled back?**

* **Are there any tests for feature enablement/disablement?**
  The e2e framework does not currently support enabling or disabling feature
  gates. However, unit tests in each component dealing with managing data, created
  with and without the feature, are necessary. At the very least, think about
  conversion tests if API types are being modified.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Try to be as paranoid as possible - e.g., what if some components will restart
   mid-rollout?

* **What specific metrics should inform a rollback?**

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  Describe manual testing that was done and the outcomes.
  Longer term, we may want to require automated upgrade/rollback tests, but we
  are missing a bunch of machinery and tooling and can't do that now.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs, 
fields of API types, flags, etc.?**
  Even if applying deprecation policies, they may still surprise some users.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
  checking if there are objects with field X set) may be a last resort. Avoid
  logs or events for this purpose.

* **What are the SLIs (Service Level Indicators) an operator can use to determine 
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  At a high level, this usually will be in the form of "high percentile of SLI
  per day <= X". It's impossible to provide comprehensive guidance, but at the very
  high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code

* **Are there any missing metrics that would be useful to have to improve observability 
of this feature?**
  Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
  implementation difficulties, etc.).

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  Think about both cluster-level services (e.g. metrics-server) as well
  as node-level agents (e.g. specific version of CRI). Focus on external or
  optional services that are needed. For example, if this feature depends on
  a cloud provider API, or upon an external software-defined storage or network
  control plane.

  For each of these, fill in the following—thinking about running existing user workloads
  and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
  focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)

* **Will enabling / using this feature result in introducing new API types?**
  Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)

* **Will enabling / using this feature result in any new calls to the cloud 
provider?**

* **Will enabling / using this feature result in increasing size or count of 
the existing API objects?**
  Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)

* **Will enabling / using this feature result in increasing time taken by any 
operations covered by [existing SLIs/SLOs]?**
  Think about adding additional work or introducing new steps in between
  (e.g. need to do X to start a container), etc. Please describe the details.

* **Will enabling / using this feature result in non-negligible increase of 
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  Things to keep in mind include: additional in-memory state, additional
  non-trivial computations, excessive access to disks (including increased log
  volume), significant amount of data sent and/or received over network, etc.
  This through this both in small and large cases, again with respect to the
  [supported limits].

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

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
