<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

To get started with this template:

- [X] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [X] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [X] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [X] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [X] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
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
# KEP-5729: DRA: ResourceClaim Support for Workloads

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
  - [User Stories](#user-stories)
    - [Sharing a ResourceClaim among many Pods](#sharing-a-resourceclaim-among-many-pods)
    - [Shareable and replicable ResourceClaims](#shareable-and-replicable-resourceclaims)
    - [Integrating DRA with high-level APIs](#integrating-dra-with-high-level-apis)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Higher memory usage by the device_taint_eviction controller](#higher-memory-usage-by-the-device_taint_eviction-controller)
    - [The number of Pods that can share a ResourceClaim will not be unlimited](#the-number-of-pods-that-can-share-a-resourceclaim-will-not-be-unlimited)
- [Design Details](#design-details)
  - [Background](#background)
    - [Deallocation](#deallocation)
    - [Finding Pods Using a ResourceClaim](#finding-pods-using-a-resourceclaim)
  - [API](#api)
    - [Workload](#workload)
    - [PodGroup](#podgroup)
    - [Pod](#pod)
    - [Example](#example)
  - [ResourceClaim Lifecycle](#resourceclaim-lifecycle)
    - [Create](#create)
    - [Delete](#delete)
    - [Allocate](#allocate)
    - [Deallocate](#deallocate)
  - [Determining Allowed Pods for a ResourceClaim](#determining-allowed-pods-for-a-resourceclaim)
  - [Finding Pods Using a ResourceClaim](#finding-pods-using-a-resourceclaim-1)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
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
  - [Increase the size limit on the <code>status.reservedFor</code> field](#increase-the-size-limit-on-the-statusreservedfor-field)
  - [Allow ResourceClaims to be reserved for any object](#allow-resourceclaims-to-be-reserved-for-any-object)
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
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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
-->

This enhancement describes additions to the [Workload API][KEP-4671] and
[PodGroup API][KEP-5832] which make it possible to associate ResourceClaims
and ResourceClaimTemplates with those objects to better facilitate sharing
DRA resources between the Pods they contain.

A ResourceClaim referenced by a PodGroup will be reserved for that
PodGroup as a whole instead of its individual Pods, addressing the
limit on the number of entries in a ResourceClaim's `status.reservedFor` list.

A ResourceClaimTemplate referenced by a PodGroup will cause a
ResourceClaim to be generated once for that PodGroup, like how
ResourceClaimTemplates work today when referenced by a Pod. Whereas
ResourceClaims today can only be shared between Pods by name, this will allow
ResourceClaims to be shared by Pods in the same PodGroup where the
exact name of the ResourceClaim is not known ahead of time.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

AI/ML workloads are particularly sensitive to network latency and bandwidth
between closely related Pods. Certain groups of Pods must be placed as closely
together as possible to achieve maximum performance.

["Modeling Topology and Multi-Node Logical Devices"][dra-topology-model]
describes where existing mechanisms like Pod and Node affinity break down for
these use cases and how Dynamic Resource Allocation (DRA) can fulfill those
requirements by scheduling Pods within strict topological boundaries:

- A ResourceSlice lists "devices" which represent nested topological units and
  form a tree of arbitrary depth. These units could be as small as a single
  host or as large as an entire datacenter, or perhaps even larger.
- A ResourceClaim requests one of these topological units. Pods which reference
  that same ResourceClaim are scheduled within the same topological boundary.
- Additionally, the allocation of the ResourceClaim may trigger a controller to
  reprogram the datacenter fabric to match the selected topological unit.

Large-scale workloads orchestrated by specialized APIs like JobSet and
LeaderWorkerSet cannot currently practically express granular topological
constraints with the current Kubernetes APIs. Today, ResourceClaims to be
shared by multiple Pods must be created one by one, and referenced by name in
the Pod spec. Those APIs which define replicable groups of Pods are left to
manage shared ResourceClaims themselves.

Moreover, the current limit of 256 entries in a ResourceClaim's
`status.reservedFor` list limits a device to being shared by up to that number
of Pods. Production-scale workloads require larger numbers of Pods to share a
single claim.

The Workload API defines a common representation of these related sets of Pods
as a PodGroup. Associating ResourceClaimTemplates with PodGroups allows
Kubernetes to manage the lifecycle of the generated ResourceClaims generically
for all types implementing the Workload API.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

- Allow users to express sets of DRA resources to be replicated for each
  PodGroup, and shared by each Pod in the PodGroup.
- Automatically create and delete PodGroups' ResourceClaims as needed.
- Reduce the burden of each true workload controller implementing
  ResourceClaim generation separately (e.g. JobSet, LWS).
- Allow claims to be allocated for more than 256 Pods.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

- Associate ResourceClaims or ResourceClaimTemplates with Workload objects
  (future work).
- Influence how Pods are placed onto Nodes based on the ResourceClaimTemplates and ResourceClaims
  associated with a PodGroup or Pod (See
  [KEP-5732](https://kep.k8s.io/5732)).

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

### User Stories

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Sharing a ResourceClaim among many Pods

As a workload author administering large deployments, I want to be able to share
a single ResourceClaim among more than 256 Pods. That opens up the possibility
for DRA to orchestrate scheduling large groups of Pods that all share a large
device, such as a virtual device representing a topological domain.

#### Shareable and replicable ResourceClaims

As a workload author administering a deployment composed of multiple groups of
Pods, I want to be able to express DRA resources which are replicated once for
each group and can be shared by all of the Pods within a particular group.

Currently, ResourceClaims generated for an individual Pod from
ResourceClaimTemplate cannot be declaratively shared among other Pods, and
standalone ResourceClaims would need to be managed separately from the rest of
the workload.

#### Integrating DRA with high-level APIs

As a maintainer of a high-level workload API like LWS or JobSet, I want to
manage the lifecycle of ResourceClaims associated with the groups of Pods
defined by my API.

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

#### Higher memory usage by the device_taint_eviction controller

The device_taint_eviction controller will need to keep an index of which Pods
are referenced from each ResourceClaim, so it can evict the correct Pods when
devices are tainted. This will require some additional memory.

#### The number of Pods that can share a ResourceClaim will not be unlimited

Removing this limit does not mean that the number of Pods that can share a
ResourceClaim will be unlimited. New scale tests will determine how many Pods
can practically share a single ResourceClaim.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Background

The `status.reservedFor` field ResourceClaims is currently used for two
purposes:

#### Deallocation

Devices are allocated to a ResourceClaim when the first Pod referencing the
claim is scheduled. Other Pods can also share the ResourceClaim in which case
they share the devices. Once no Pods are consuming the claim, the devices should
be deallocated to they can be allocated to other claims. The
`status.reservedFor` list is used to keep track of Pods consuming a
ResourceClaim. Pods are added to the list by the DRA scheduler plugin during
scheduling and removed from the list by the ResourceClaim controller when Pods
are deleted or finish running. An empty list means there are no current
consumers of the claim and it can be deallocated.

#### Finding Pods Using a ResourceClaim

`status.reservedFor` is read by the DRA scheduler plugin, the kubelet, and the
device_taint_eviction controller to find Pods that are using a ResourceClaim:

1. The kubelet uses this to make sure it only runs Pods where the claims
   have been allocated to the Pod. It can verify this by checking that the Pod
   is listed in the `status.reservedFor` list.

1. The DRA scheduler plugin uses the list to find claims that have zero or only
   a single Pod using it, and is therefore a candidate for deallocation in the
   `PostFilter` function.

1. The device_taint_eviction controller uses the `ReservedFor` list to find the
   Pods that need to be evicted when one or more of the devices allocated to a
   ResourceClaim is tainted (and the ResourceClaim does not have a toleration).

So the solution needs to:

- Give the ResourceClaim controller a way to know when there are no more
  consumers of a ResourceClaim so it can be deallocated.
- Give controllers a way to list the Pods consuming or referencing a
  ResourceClaim.

### API

The following API changes will be made:

- The Workload and PodGroup APIs will be updated to include references to
  ResourceClaims and ResourceClaimTemplates, like Pods.
- The Pod API will be updated to include references to claims listed in its
  PodGroup.

#### Workload

The Workload API changes are modeled after the existing Pod API to reference
ResourceClaims, adding a `spec.podGroupTemplates[].resourceClaims` field:

```go
type PodGroupTemplate struct {
	...

	// ResourceClaims defines which ResourceClaims may be shared among Pods in
	// the group. Pods must reference these claims in order to consume the
	// allocated devices.
	//
	// This is an alpha-level field and requires that the
	// WorkloadPodGroupResourceClaimTemplate feature gate is enabled.
	//
	// This field is immutable.
	//
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	// +listType=map
	// +listMapKey=name
	// +featureGate=WorkloadPodGroupResourceClaimTemplate
	// +optional
	ResourceClaims []PodGroupResourceClaim `json:"resourceClaims,omitempty"`
}

// PodGroupResourceClaim references exactly one ResourceClaim, either directly
// or by naming a ResourceClaimTemplate which is then turned into a ResourceClaim
// for the PodGroup.
//
// It adds a name to it that uniquely identifies the ResourceClaim inside the PodGroup.
// Pods that need access to the ResourceClaim reference it with this name.
type PodGroupResourceClaim struct {
	// Name uniquely identifies this resource claim inside the PodGroup.
	// This must be a DNS_LABEL.
	Name string `json:"name"`

	// ResourceClaimName is the name of a ResourceClaim object in the same
	// namespace as this PodGroup. The ResourceClaim will be reserved for the
	// PodGroup instead of its individual pods.
	//
	// Exactly one of ResourceClaimName and ResourceClaimTemplateName must
	// be set.
	ResourceClaimName *string `json:"resourceClaimName,omitempty"`

	// ResourceClaimTemplateName is the name of a ResourceClaimTemplate
	// object in the same namespace as this PodGroup.
	//
	// The template will be used to create a new ResourceClaim, which will
	// be bound to this PodGroup. When this PodGroup is deleted, the ResourceClaim
	// will also be deleted. The PodGroup name and resource name, along with a
	// generated component, will be used to form a unique name for the
	// ResourceClaim, which will be recorded in pod.status.resourceClaimStatuses.
	//
	// This field is immutable and no changes will be made to the
	// corresponding ResourceClaim by the control plane after creating the
	// ResourceClaim.
	//
	// Exactly one of ResourceClaimName and ResourceClaimTemplateName must
	// be set.
	ResourceClaimTemplateName *string `json:"resourceClaimTemplateName,omitempty"`
}
```

#### PodGroup

The PodGroup API will be updated similarly to contain the ResourceClaim
references from its template defined in the Workload:

```go
type PodGroupSpec struct {
	...

	// ResourceClaims defines which ResourceClaims may be shared among Pods in
	// the group. Pods must reference these claims in order to consume the
	// allocated devices.
	//
	// This is an alpha-level field and requires that the
	// WorkloadPodGroupResourceClaimTemplate feature gate is enabled.
	//
	// This field is immutable.
	//
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	// +listType=map
	// +listMapKey=name
	// +featureGate=WorkloadPodGroupResourceClaimTemplate
	// +optional
	ResourceClaims []PodGroupResourceClaim `json:"resourceClaims,omitempty"`
}
```

#### Pod

When a PodGroup includes claims, the `name` of a claim in the
PodGroup can be used on Pods in the group to associate the PodGroup's dedicated
ResourceClaim. This complements existing references to ResourceClaims and
ResourceClaimTemplates.

```go
// PodResourceClaim references exactly one ResourceClaim, either directly,
// by naming a ResourceClaimTemplate which is then turned into a ResourceClaim
// for the pod, or by naming a claim made for a PodGroup.
//
// It adds a name to it that uniquely identifies the ResourceClaim inside the Pod.
// Containers that need access to the ResourceClaim reference it with this name.
type PodResourceClaim struct {
	...

	// PodGroupResourceClaim refers to the name of a claim associated
	// with this pod's PodGroup.
	//
	// Exactly one of ResourceClaimName, ResourceClaimTemplateName,
	// or PodGroupResourceClaim must be set.
	PodGroupResourceClaim *string `json:"podGroupResourceClaim,omitempty"`
}
```

#### Example

The following example demonstrates the relationships between the new fields. It
describes the more common case where some higher level true workload controller
(e.g. LWS, JobSet) is orchestrating the Workload and PodGroup objects vs. the
user managing those directly.

Here, a user defines a high-level workload with two logical groups of Pods. Each
of the two groups of Pods also request one device to be shared by the Pods in
its group.

The user creates the following objects to request DRA devices which will
be referenced by Pods through their PodGroup:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaimTemplate
metadata:
  name: pg-claim-template
  namespace: default
spec:
  spec:
    devices:
      requests:
      - name: my-device
        exactly:
          deviceClassName: example
---
apiVersion: example.com/v1
kind: MyWorkload
metadata:
  name: my-workload
  namespace: default
spec:
  ...
```

The true workload API defines how ResourceClaims and ResourceClaimTemplates
relate to groups of Pods. If the user is responsible for defining the Pods'
`spec.resourceClaims` in a Pod template, then the PodGroups'
`spec.resourceClaims[].name`s must be deterministic for the user to be able to
reference them in the Pod spec.

The true workload controller then creates the following Workload API resources
based on the true workload's definition:

```yaml
apiVersion: scheduling.k8s.io/v1alpha1
kind: Workload
metadata:
  name: my-workload
  namespace: default
spec:
  podGroupTemplates:
  - name: group-1
    policy:
      basic: {}
    resourceClaims:
    - name: pg-claim
      resourceClaimTemplateName: pg-claim-template
  - name: group-2
    policy:
      basic: {}
    resourceClaims:
    - name: pg-claim
      resourceClaimTemplateName: pg-claim-template
---
apiVersion: scheduling.k8s.io/v1alpha1
kind: PodGroup
metadata:
  name: my-podgroup-1
  namespace: default
spec:
  workloadRef:
    name: my-workload
  podGroupTemplateRef:
    name: group-1
  resourceClaims:
  - name: pg-claim
    resourceClaimTemplateName: pg-claim-template
---
apiVersion: scheduling.k8s.io/v1alpha1
kind: PodGroup
metadata:
  name: my-podgroup-2
  namespace: default
spec:
  workloadRef:
    name: my-workload
  podGroupTemplateRef:
    name: group-2
  resourceClaims:
  - name: pg-claim
    resourceClaimTemplateName: pg-claim-template
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wl-claim-example-1
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: wl-claim-example-1
  template:
    metadata:
      labels:
        app: wl-claim-example-1
    spec:
      containers:
      - name: pause
        image: "registry.k8s.io/pause:3.6"
        resources:
          claims:
          - name: resource
      resourceClaims:
      - name: resource
        podGroupResourceClaim: pg-claim
      workloadRef:
        name: my-workload
        podGroupName: my-podgroup-1
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: wl-claim-example-2
  namespace: default
spec:
  replicas: 2
  selector:
    matchLabels:
      app: wl-claim-example-2
  template:
    metadata:
      labels:
        app: wl-claim-example-2
    spec:
      containers:
      - name: pause
        image: "registry.k8s.io/pause:3.6"
        resources:
          claims:
          - name: resource
      resourceClaims:
      - name: resource
        podGroupResourceClaim: pg-claim
      workloadRef:
        name: my-workload
        podGroupName: my-podgroup-2
```

Here, a Workload organizes Pods managed by two different Deployments into two
different PodGroups.
Each group refers to the same ResourceClaimTemplate,
`pg-claim-template`. This single ResourceClaimTemplate forms the basis of two
different ResourceClaims which will be created by the ResourceClaim controller:
one for each PodGroup. The Pod templates in the Deployments include a reference
to the claim listed for the PodGroup, which ultimately resolves to its
PodGroup's ResourceClaim. The result is that with a single
ResourceClaimTemplate, Pods in the same group all share the exact same allocated
device, while Pods in the other group use an equivalent, but separately
allocated, device.

### ResourceClaim Lifecycle

The DynamicResources scheduler plugin and the ResourceClaim controller will
cooperate to manage key points in the life of a ResourceClaim or
ResourceClaimTemplate claimed by a PodGroup. Referenced ResourceClaimTemplates
will replicate into one ResourceClaim per PodGroup. Those generated
ResourceClaims and ResourceClaims referenced by name by a PodGroup will be
allocated and deallocated by the Kubernetes control plane.

#### Create

When a PodGroup is created which references a ResourceClaimTemplate, the
ResourceClaim controller will create a ResourceClaim from that template if one
does not already exist for that PodGroup. Generated ResourceClaims will be owned (through
`metadata.ownerReferences`) by the PodGroup and annotated with
`resource.kubernetes.io/podgroup-claim-name` where the value is the name of the
claim from the PodGroup's `spec.resourceClaims[].name` to facilitate mapping a
single PodGroup claim to the ResourceClaim generated for its PodGroup. When a
Pod is created which requests a claim from its PodGroup, the name of the
ResourceClaim generated for the PodGroup's claim
will be recorded in the Pod's `status.resourceClaimStatuses` like
ResourceClaims generated for Pods. Like the
`resource.kubernetes.io/podgroup-claim-name` annotation,
`resource.kubernetes.io/podgroup-claim-name` is only to be used by the
controller and will not be documented as part of the public API.

#### Delete

The `resource.kubernetes.io/delete-protection` finalizer added to a generated
ResourceClaim by kube-scheduler serves the same purpose as for other
ResourceClaims, preventing the ResourceClaim from being deleted until it is
deallocated. Like other generated ResourceClaims, the ResourceClaim controller
will unlock deletion of PodGroup-owned claims by removing the finalizer when
they become deallocated. The garbage collector will then be responsible for
deleting the ResourceClaim once its owning PodGroup is deleted.

#### Allocate

Generated and standalone ResourceClaims referenced by a PodGroup remain
unallocated until kube-scheduler allocates the ResourceClaim by setting
`status.allocation` for the first Pod in the PodGroup that references the
PodGroup's claim. When a Pod's claim is requested through
`podGroupResourceClaim`, the ResourceClaim's `status.reservedFor` list will
reference the PodGroup instead of each individual Pod.

#### Deallocate

The ResourceClaim controller will continue to deallocate claims when there are
no entries in the ResourceClaim's `status.reservedFor`. References to PodGroups
in `status.reservedFor` are removed after the PodGroup is deleted. Since
PodGroups can only be deleted when all of their Pods have been deleted,
ResourceClaims reserved for a PodGroup will therefore not be deleted before any
of the Pods in the group which are actively using it. The entity which creates a
PodGroup is responsible for deleting it when no more Pods in the group are
expected to run.

### Determining Allowed Pods for a ResourceClaim

Currently, any Pod allowed to utilize a ResourceClaim is listed explicitly in
the claim's `status.reservedFor`. When the list instead references a PodGroup,
only the name in the reference must match a Pod's
`spec.workloadRef.podGroupName`. Since a finalizer will protect a PodGroup from
being deleted before any of its Pods, a reference to the name of a PodGroup in a
Pod will always refer to the exact same PodGroup, i.e. the PodGroup cannot be
deleted and recreated with the same name without all of its Pods also being
deleted in the meantime or if its finalizer is manually removed.

### Finding Pods Using a ResourceClaim

If the reference in the `status.reservedFor` list is to a PodGroup,
controllers can no longer use the list to directly find all Pods consuming the
ResourceClaim. Instead they will look up all Pods referencing the
PodGroup, which can be done by using a watch on Pods and maintaining an index of
PodGroup to Pods referencing it. This can be done using the informer
cache.

The list of Pods making up a PodGroup for which a ResourceClaim is
reserved is not exactly the same as the list of Pods consuming a ResourceClaim.
The `status.reservedFor` list only references Pods, or Pods'
PodGroups, that have been processed by the DRA scheduler plugin and
are scheduled to use the ResourceClaim. It is possible to have Pods that
reference a PodGroup that has been allocated a claim, but haven't
yet been scheduled. This distinction is important for some of the usages of the
`status.reservedFor` list described above:

<!-- TBD if status.allocation.reservedForAnyPod will be used
1. If the kubelet sees that the `status.allocation.ReservedForAnyPod` is set, it
   will skip the check that the Pod is listed in the `ReservedFor` list and just
   run the pod.
-->

1. If the DRA scheduler plugin is trying to find candidates for deallocation in
   the `PostFilter` function and sees a ResourceClaim with a non-Pod reference,
   it will not attempt to deallocate. The plugin has no way to know how many
   Pods are actually consuming the ResourceClaim without the explicit list in
   `status.reservedFor` list and therefore it will not be safe to deallocate.

1. The device_taint_eviction controller will use the list of Pods referencing
   the PodGroup to determine the list of pods that needs to be
   evicted. In this situation, it is ok if the list includes pods that haven't
   yet been scheduled.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

None needed.

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `k8s.io/dynamic-resource-allocation/resourceclaim`: `2026-01-29` - `89.3%`
- `k8s.io/kubernetes/pkg/apis/core/v1`: `2026-01-29` - `79.0%`
- `k8s.io/kubernetes/pkg/apis/core/validation`: `2026-01-29` - `85.3%`
- `k8s.io/kubernetes/pkg/apis/scheduling/v1alpha1`: `2026-01-29` - `83.3%`
- `k8s.io/kubernetes/pkg/apis/scheduling/validation`: `2026-01-29` - `96.6%`
- `k8s.io/kubernetes/pkg/controller/devicetainteviction`: `2026-01-29` - `86.7%`
- `k8s.io/kubernetes/pkg/controller/resourceclaim`: `2026-01-29` - `74.6%`
- `k8s.io/kubernetes/pkg/kubelet/cm/dra`: `2026-01-29` - `83.6%`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: `2026-01-29` - `79.2%`

##### Integration tests

<!--
Integration tests are contained in https://git.k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
For more details, see https://github.com/kubernetes/community/blob/master/contributors/devel/sig-testing/testing-strategy.md

If integration tests are not necessary or useful, explain why.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically https://testgrid.k8s.io/sig-release-master-blocking#integration-master), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)
-->

New integration tests will verify:
- New API fields in Pod and PodGroup are persisted or rejected correctly
  depending on the value of the `WorkloadPodGroupResourceClaimTemplate` feature
  gate.
- ResourceClaimTemplates specified for PodGroups result in the correct
  ResourceClaims being allocated for the correct Pods.
- No inconsistent state is reached when PodGroups rapidly come and go.
    - ResourceClaims should continue to be created and deleted with their
      owning PodGroups such that Pods still schedule and no ResourceClaims are
      orphaned.
    - At most one generated ResourceClaim should exist for a claim made by a
      PodGroup at any given time.

Additionally, scheduler_perf tests will be added, aiming for the same thresholds
as existing DRA tests.

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, document that tests have been written,
have been executed regularly, and have been stable.
This can be done with:
- permalinks to the GitHub source code
- links to the periodic job (typically a job owned by the SIG responsible for the feature), filtered by the test name
- a search in the Kubernetes bug triage tool (https://storage.googleapis.com/k8s-triage/index.html)

We expect no non-infra related flakes in the last month as a GA graduation criteria.
If e2e tests are not necessary or useful, explain why.

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)
-->

New e2e tests will verify correct behavior at key points in the lifecycle of a
PodGroup.

- When a PodGroup referencing a ResourceClaimTemplate is created, a
  ResourceClaim is generated and remains unallocated.
- When the first Pod is created for the PodGroup, the ResourceClaim is
  allocated.
- When subsequent Pods in the PodGroup are created, no additional ResourceClaims
  are generated and the Pods are all allocated the same existing ResourceClaim.
- When all Pods in the PodGroup are deleted, the ResourceClaim is not
  deleted and remains allocated.
- When the PodGroup has been deleted, then the ResourceClaim
  is deallocated, and eventually deleted.


### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].
-->

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

<!--
**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified
-->

#### GA

- Integration with at least 2 widely used APIs for complex workload
  orchestration (e.g. Jobset, LeaderWorkerSet)
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

<!--
**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
-->

<!--
#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
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

The feature will no longer work if downgrading to a release without support for
it. The API server will no longer accept the new fields and the other components
will not know what to do with them. So the result is that the
`status.reservedFor` list will only have references to Pod resources like today.

Any ResourceClaims that have already been allocated when the feature was active
will have PodGroup references in the `status.reservedFor` list after a
downgrade, but the controllers will not know how to handle it. There are two
problems that will arise as a result of this:

- The ResourceClaim controller will also have been downgraded, meaning that it
  will not remove references to PodGroups from the `status.reservedFor` list,
  thus leading to a situation where the claim will never be deallocated.

- For new Pods that get scheduled, the scheduler will add Pod references in the
  `status.reservedFor` list, despite there being a PodGroup reference here. So
  it ends up with both Pod and PodGroup references in the list. We can manage
  both Pod and PodGroup references in the list by
  adding the PodGroup reference even if Pod references exist and
  making sure that the ResourceClaim controller removes Pod references even if
  there are PodGroup references in the list. Deallocation is only safe
  when no Pods are consuming the claim, so both PodGroup and Pod reference
  should be removed once that is true.

We will also provide explicit recommendations for how users can manage
downgrades or disabling this feature. This means manually updating the
`status.reservedFor` list to reference only Pods and not PodGroups. We don't
plan on providing automation for this.

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

If the kubelet is on a version that doesn't support the feature but the rest of
the components are, Pods referencing a PodGroup will be scheduled, but the
kubelet will refuse to run those Pods since it will still check whether the
Pods are referenced in the `status.reservedFor` list.

If the API server is on a version that supports the feature, but the scheduler
is not, the scheduler will not know about the new fields added, so it will put
the reference to the Pod in the `status.reservedFor` list rather than the
PodGroup. It will do this even if there is already a PodGroup reference in the
`status.reservedFor` list. This leads to the challenge described in the previous
section.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

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

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
-->

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: WorkloadPodGroupResourceClaimTemplate
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager
    - kube-scheduler
    - kubelet

<!--
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?
-->

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

If the kubelet restarts with the feature disabled, existing containers continue
to run with all of their allocated devices, including those from claims made by
their PodGroup when the feature was enabled.

If a DRA device is allocated to a ResourceClaim reserved for a PodGroup and the
feature is disabled, the PodGroup will continue to be listed in the
`status.reservedFor` of the ResourceClaim and will not be deallocated.

###### What happens if we reenable the feature if it was previously rolled back?

If the kubelet restarts with the feature enabled, then containers similarly
continue to run with all of the devices with which they were first started.

Since no other state is lost when the feature is disabled, other components
once again operate as described.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

Unit and integration tests will verify behavior both when the feature is enabled
and when it is disabled. They will also exercise cases where the feature is
toggled.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
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
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

- kube-controller-manager will list and watch PodGroup resources.
- kubelet will `GET` the PodGroup for a Pod when the Pod references a PodGroup
  claim made through the PodGroup's `resourceClaimName`.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

This feature adds a new `spec.resourceClaims` list to the PodGroup API. It will
have the same limits as the Pod API's `spec.resourceClaims`.

The Pod API adds a new `spec.resourceClaims[].podGroupResourceClaim` field which
is mutually exclusive with its sibling `resourceClaimName` and
`resourceClaimTemplate` fields so it will not meaningfully impact the size of a
Pod.

The size of a ResourceClaim's `spec.reservedFor` list will be reduced
significantly when many Pods sharing the same claim make that claim through a
common PodGroup.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

- kube-controller-manager will run a new informer for PodGroup resources and
  index them by ResourceClaims they reference.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
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
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

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

- 2025-12-12: KEP first draft published for review
- 2026-01-28: Combined with [KEP-5194]

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

This complicates the allocation and deallocation logic somewhat as there will be
two separate ways to manage the allocation and deallocation process for
ResourceClaims.

It also leads to additional work for the device_taint_eviction controller since
it needs to maintain an index to find all Pods using a ResourceClaim rather than
just looking at the list of Pods in the `status.reservedFor` list.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Increase the size limit on the `status.reservedFor` field

To allow more Pods to share a single claim, the simplest solution would be to
increase the size limit on the `status.reservedFor` field. Having a large
list of Pod references is not a good way to handle it and could at least in
theory run into the size limit of Kubernetes resources. Also, we would need to
have some limit on the size, and whatever number we choose might still be too
small for the largest workloads.

### Allow ResourceClaims to be reserved for any object

[KEP-5194] originally described the addition of new `spec.reservedFor` and
`status.reservedForAnyPod` fields for ResourceClaims, to enable references to
arbitrary objects in `status.reservedFor`. This approach shifts the
responsibility to remove non-Pod objects from the `status.reservedFor` list to
each true workload controller supporting DRA.

With the addition of the Workload and PodGroup APIs, the ResourceClaim API no
longer needs to be as flexible since true workloads can integrate with those
common APIs. In order to integrate with this feature, true workload controllers
create and delete PodGroup objects (which will also provide many additional
features) and don't have to explicitly manage ResourceClaims.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->


[KEP-4671]: https://kep.k8s.io/4671
[KEP-5832]: https://kep.k8s.io/5832
[KEP-5194]: https://kep.k8s.io/5194
[dra-topology-model]: https://docs.google.com/document/d/1Fg9ughIRMtt1HmDqiGWV-w9OKdrcKf_PsH4TjuP8Y40/edit?usp=sharing
