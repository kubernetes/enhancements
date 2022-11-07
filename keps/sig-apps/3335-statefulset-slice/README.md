<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
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

# KEP-3335: StatefulSet Slice

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
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [e2e/Integration tests](#e2eintegration-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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
  - [Alternative API changes](#alternative-api-changes)
  - [Alternatives without any API changes](#alternatives-without-any-api-changes)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

Both in this section and below, follow the guidelines of the [documentation
style guide]. In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md
-->

The goal of this feature is to allow a StatefulSet to be migrated (cross
namespace, cross cluster, split into segments) without disrupting the underlying
application.

A StatefulSet of `N` replicas implicitly numbers pods from ordinal `0` to `N-1`.
The end ordinal (`N-1`) can be controlled with the `replicas` field. The goal of
this feature is to allow a StatefulSet’s first ordinal to start from any natural
number `k`. This enables StatefulSet ordinals from `k` to `N+k-1`

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->

This feature is motivated by the use case of orchestrating the migration of a StatefulSet across a
namespace or a Kubernetes cluster without disruption. Existing approaches to
this problem include:

1. Back and restore: This approach takes a backup of an application (StatefulSet, underlying storage), and re-creates it in a different location. This introduces application downtime, the duration of time between old StatefulSet termination and new StatefulSet recreation.
2. Pod level migration: Using `--cascade=orphan` when deleting a StatefulSet preserves the pods. This allows an application operator to evict and reschedule pods individually. However, as pods are ephemeral, this requires the application operator to emulate the behavior of the StatefulSet, to reschedule pods as they restart, or are evicted and rescheduled.

Migrating a StatefulSet in slices allows for gradual migration of the application, as only a subset of replicas are migrated at any time. Consider the scenario of transferring pod ordinal ownership from a source StatefulSet with `N` pods to a destination StatefulSet with `0` pods. Further, to maintain application availability, no more than `d` pods should be unavailable at any time during the transfer. An orchestrator can manipulate `.spec.replicas` and `.spec.ordinals.start` to perform this migration:

 1. Validate the source StatefulSet (replicas=`N`, ordinals.start=`0`).
 2. Validate the destination StatefulSet (replicas=`0`, ordinals.start=`N`).
 3. Adjust PDBs on source and destination to distribute the budget `d`.
 4. While the destination StatefulSet has less than `N` replicas. (Source has `k` replicas, destination has `N-k` replicas).
    1. Scale down the source StatefulSet by `1`. Replicas `k` will be terminated (replicas=`k-1`, ordinals.start=`0`)
       1. This should allow the application to determine that replica `k` is no longer available.
       2. PDBs associated with each StatefulSet should be adjusted to reflect a reduced availability budget based on `d`
    2. Move any dependencies of replica `k` to the destination (cluster or namespace)
       1. This may include namespace resources (PVC, ConfigMap) or cluster scoped resources (PV)
    3. Scale up the destination StatefulSet by `1`. Replicas `k` will be started (replicas=`N-k`, ordinals.start=`k`)
       1. The replica `k` should re-advertise its new network identity, through application peer discovery, and network endpoints that reference the existing pod ordinal should be updated.
 5. The source StatefulSet should have `0` replicas and destination StatefulSet `N` replicas
 6. Clean up the source StatefulSet, and any unused resources safely.

StatefulSets are implicitly numbered starting at ordinal `0`. When pods are being deployed, they are created [sequentially in order](StatefulSet) from pod `0` to pod `N-1`. When pods are being deleted, they are terminated in reverse order from pod `N-1` to pod `0`. This behavior limits the migration scenario where an application operator wants to scale down pods in the source StatefulSet and scale up pods in the destination StatefulSet. If pod `N-1` is removed from the source StatefulSet, there is no mechanism to create only pod `N-1` in a destination StatefulSet without creating pods `[0, N-2]` as well. To do so would lead to the presence of duplicate pod ordinals (eg: pod `0` would exist in both StatefulSets).

Extending StatefulSet to start at an ordinal `k` (eg: `N-1`) would allow the destination StatefulSet to skip over pods `[0, N-2]` when creating pod `N-1`. This allows the original StatefulSet to be sliced at ordinal `k` between the source and destination StatefulSet.

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

StatefulSet controller manages pods for a slice of a StatefulSet, within the range `[k, N+k-1]`

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->

* Updating a PDB to safeguard more than one StatefulSet slice
    *   As StatefulSet slices are scaled up or down, corresponding PDBs can also be adjusted. For example, a PDB corresponding to a slice of `k` replicas could be adjusted to `MinAvailable: k-1` on scale up or down events. Providing guidance and functionality to adjust these PDBs is outside the scope of this KEP.
* Orchestrating pod movement from one StatefulSet slice to another
* Managing network connectivity between pods in different StatefulSet slices
* Orchestrating storage lifecycle of PVCs and PVs across different StatefulSet slices
  * Referenced PV/PVCs will need to be migrated in order for a new StatefulSet to reference data that was used by an existing StatefulSet. Orchestration complexity will depend on how volumes are used (RWO with `.spec.volumeClaimTemplates` on a StatefulSet, RWX with pod `.spec.volumes`). If using StatefulSet PVC Auto-Deletion ([KEP-1847](https://github.com/kubernetes/enhancements/issues/1847)), `whenDeleted` and `whenScaled` should be set to `Retain` on the existing StatefulSet prior to migration.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->

This KEP solves the problem of managing subsets of the replicas in a StatefulSet by introducing the concept of a slice. A slice consists of a start ordinal `k`, and a number of replicas `N`. To control the starting and ending ordinal of each slice, a new struct `ordinals` is introduced to `StatefulSetSpec`.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

The main motivation of this KEP is to support a more flexible StatefulSet, a building block in an ecosystem where Stateful applications can be migrated across Kubernetes clusters with more automation. Below are two high level user stories that share the problem of having a StatefulSet locked into a specific configuration. To fully automate these two scenarios, additional building blocks are needed around volume management and networking (see [Notes/Constraints/Caveats](#notesconstraintscaveats-optional)).

#### Story 1

**Migrating across namespaces**: Many organizations use namespaces for team isolation. Consider a team that is migrating a `StatefulSet` to a new namespace in a cluster. Migration could be motivated by a branding change, or a requirement to move out of a shared namespace. Consider the StatefulSet `my-app` with `replicas: 5`, running in a shared namespace.


```
name: my-app
namespace: shared
replicas: 5
-----------------------------------------------
[ nginx-0, nginx-1, nginx-2, nginx-3, nginx-4 ]
```


To move two pods, the `my-app` StatefulSet in the `shared` namespace can be scaled down to `replicas: 3, ordinals.start: 0`, and an analogous StatefulSet in the `app-team` namespace scaled up to `replicas: 2, ordinals.start: 3`. This allows for pod ordinals to be managed during migration. The application operator should manage network connectivity, volumes and slice orchestration (when to migrate and by how many replicas).


```
name: my-app						name: my-app
namespace: shared					namespace: app-team
replicas: 3						    replicas: 2
ordinals.start: 0				    ordinals.start: 3
------------------------------		---------------------
[ nginx-0, nginx-1, nginx-2 ]		[ nginx-3, nginx-4 ]
```


The `replicasStatefulSet` and `replicas` fields should be updated jointly, depending on the requirements of the migration.

#### Story 2

**Migrating across clusters**: Organizations taking a multi cluster approach may need to move workloads across clusters due to capacity constraints, infrastructure constraints, or for better application isolation. Similar to namespace migration, the application operator should manage network connectivity, volumes and slice orchestration.

#### Story 3

**Non-Zero Based Indexing:** A user may want to number their StatefulSet starting from ordinal `1`, rather than ordinal `0`. Using
`1` based numbering may be easier to reason about and conceptualize (eg: ordinal `k` is the `k`'th replica, not the `k+1`'th replica).

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

The following caveats are applicable to migrating a StatefulSet (scaling down one slice and scaling up another). The following caveats are outside the scope of this KEP, but are applicable to the User Journey of migration motivated by this feature.

**Networking:** Managing services and networking during migration is outside the scope of this proposal. Cross cluster migration can leverage [Multi-Cluster Services](https://github.com/kubernetes/enhancements/tree/master/keps/sig-multicluster/1645-multi-cluster-services-api) to establish connectivity between pods in different slices. The application operator must set up Multi-Cluster Services in the clusters, and the underlying Stateful application must be configured appropriately. Cross namespace migration can leverage a fallback domain, referring to services from both slices. Similarly this requires an application to be aware of both services.

**Storage:** StatefulSets that use `volumeClaimTemplates`, will create pods that consume per replica PVCs. PVs are cluster scoped resources, but are bound one-to-one with namespace scoped PVCs. If the underlying storage is to be re-used in the new namespace, PVs must be unbound and manipulated appropriately.

**Orchestration:** Consider migrating from namespace `A` to `B`. To preserve StatefulSet at most one semantics, pods should only be migrated when safe to do so. If migrating across namespaces, a pod with ordinal `i` should be scaled down in slice `A` before it is scaled up in slice `B`.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

This KEP proposes a new field `spec.ordinals.start` with a default value of `0`. StatefulSet will maintain current behavior, if this field is unset.

To mitigate risk, this feature will be rolled out with an alpha feature gate for experimentation. In Beta, new functionality should only take effect if the field `spec.ordinals.start` is set to a value greater than `0`.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

**StatefulSet Spec Changes**

A new struct is introduced to the `StatefulSetSpec`. In this KEP, the field only
has a single field `Start`. A struct is added (rather than putting `start`
directly into `spec`) to allow for the ordinals struct to change over time. If
future  use cases of StatefulSet require further ordinal controls (eg: ordinal
numbering based on failure domains), new fields related to the numbering and
grouping of StatefulSet ordinals can be added to this struct.

```
type StatefulSetSpec struct {
        // Ordinals controls how the stateful set creates pod and
        // persistent volume claim names.
        // The default behavior assigns a number starting with zero
        // and incremented by one for each additional replica requested.
        // +optional
        Ordinals struct {
               // Start is the number representing the
               // first index that is used to represent replica ordinals.
               // If set, replica ordinals will be numbered
               // [ordinals.start, ordinals.start + replicas)
               // If unspecified, defaults to 0
               // +optional
               Start int32
       }
}
```

**Control Loop Changes**

In the main control loop, StatefulSet will attempt to create pod replicas `[k, N+k-1]`
*   When scaling up pods: If ordinal `i` in range `[k, N+k-1]` does not exist, pod `i` will be created.
*   When scaling down pods: If ordinal `j` exists but is not in range `[k, N+k-1]`), pod `j` will be terminated.
**RollingUpdate Partition Changes**

Since `ordinals.start` changes the offset of the replica ordinals, this affects the `partition` field used for RollingUpdate. As `partition` specifies an ordinal index, the partition field must be in the range `[k, N+k-1]`, to be valid. 

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

*   `pkg/controller/statefulset/stateful_set_control_test.go - Tests that a StatefulSet slice can be created from specified starting ordinal`
*   `pkg/apis/apps/v1/defaults_test.go - Tests defaults for new fields added to StatefulSet`

##### e2e/Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

*   Pod Restart Tests: Validate that StatefulSet RollingUpdate behavior is preserved, with an replica ordinal offset starting at `ordinals.start`
*   Scaling Tests
    *   Adding `ordinals.start`: Validate that setting `ordinals.start` to `k` causes StatefulSet ordinals to be scaled (pods `[0, k-1]` are terminated, pods `[N, N+k-1]` are created)
    *   Increasing `ordinals.start`: Validate that increasing `ordinals.start` from `m` to `n` causes StatefulSet ordinals to be scaled (pods `[m, n-1]` are terminated, pods `[m+N, n+N-1]` are created)
    *   Removing `ordinals.start`: Validate that setting `ordinals.start` causes StatefulSet ordinals to be scaled (pods `[N-1, N+k-1]` are terminated, pods `[0, k-1]` are created)
    *   Decreasing `ordinals.start`: Validate that decreasing `ordinals.start` from `m` to `n` causes StatefulSet ordinals to be scaled (pods `[m+N, n+N-1]` are terminated, pods `[m, n-1]` are created)

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

In general, we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

 * Feature functionality implemented but hidden behind a feature gate
 * Add unit, functional, upgrade and downgrade tests to automated k8s test.


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

**Upgrades**: This feature adds a new field (`ordinals.start`) to the StatefulSet. The default value for the new field maintains the existing behavior of StatefulSet.

**Downgrades**: When using `ordinals.start`, downgrades are not backwards compatible. Versions of StatefulSet not implementing this feature will attempt to re-create all replicas from `[0, N-1]`, and terminate any pods of ordinal `N` or greater, where `N` is the number of replicas.

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

There are only `kube-controller-manager` changes involved (in addition to the apiserver changes for dealing with the new StatefulSet field). Node components are not involved so there is no version skew between nodes and the control plane.

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

 -  Feature gate (also fill in values in `kep.yaml`)
    *   Feature gate name: StatefulSetSlice
    *   Components depending on the feature gate:
        *   `kube-controller-manager`: Controls which replica ordinals are created
        *   `kube-apiserver`: Manages the new policy field `ordinals.start`
 -  Other
    *   Describe the mechanism:
    *   Will enabling / disabling the feature require downtime of the control plane? **No**
    *   Will enabling / disabling the feature require downtime or reprovisioning of a node? **No**

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

No, if the new StatefulSet field `.spec.ordinals.start` is unset,
StatefulSet will retain existing behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes, disabling the feature gate will cause the new field to be ignored

Note that if disabled, StatefulSet will implicitly number ordinals starting from `0`. This can cause churn of pods when disabled, if `ordinals.start` is not `0`. Care should be taken when disabling this feature on a spec that has `ordinals.start` set, and should only be done when pod churn or disruption can be tolerated.

###### What happens if we reenable the feature if it was previously rolled back?

StatefulSet will see the `ordinals.start` field and scale pods to start from this ordinal. This can cause churn of pods when enabled, if `ordinals.start` is not `0`. Care should be taken when enabling this feature on a spec that has `ordinals.start` set, and should only be done when pod churn or disruption can be tolerated.

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

Additional e2e tests will be added when targeting the Beta stage. These will
validate the behavior of the cluster when enabled and disabled and ensure that
existing behavior (eg: not specifying the new `ordinals.start` API) is
preserved.

### Rollout, Upgrade and Rollback Planning

TBD upon graduation to beta.

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

TBD upon graduation to beta.

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

TBD upon graduation to beta.

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

TBD upon graduation to beta.

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

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

TBD upon graduation to beta.

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

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

Downgrades are not gracefully supported, and are not backwards compatible. Cluster downgrades can cause a disruption to StatefulSet workloads if performed while `ordinals.start` is not `0`.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

### Alternative API changes

**ReverseOrderedReady**: A new PodManagementPolicy policy called
`ReverseOrderedReady` could be added. This would allow a StatefulSet to be
started and actuated from the highest ordinal (current default is from the
lowest ordinal). For the cross-cluster migration use case, this would allow for
a source StatefulSet to be scaled down and a target StatefulSet to be scaled in.
The downside with this API is that pod management policy is not a mutable field.
So if an orchestrator uses this behavior to scale in a StatefulSet, in a
destination cluster, and then wants to revert the PodManagementPolicy back to
default, the StatefulSet would need to be deleted, and re-created.

**KEP-3521**: [KEP-3521](https://github.com/kubernetes/enhancements/issues/3521)
proposes a Pod `.spec` level API that enables a pod to be paused at the initial
scheduling phase of pod lifecycle. This provides granular control of which pods
should be started and running (active) and which pods shouldn't be scheduled
(standby). An orchestrator can leverage control over specific pod scheduling,
without making changes to the StatefulSet controller, as the StatefulSet
controller is in control of creating pods.

If the StatefulSet controller is using OrderedReady Pod Management, pausing
scheduling can result in a pod being marked as not Ready. This will prevent
the StatefulSet controller from actuating updates to higher ordinal pods (eg:
pod `m` will not be created if pod `n` is unhealthy, where `m` > `n`). This
may increase orchestrator complexity, by requiring an orchestrator of a
migration to leverage Parallel Pod Management during a migration, and then
re-create a StatefulSet (using `--cascade=orphan`) to revert back to
`OrderedReady` if desired.

Additionally, if modifying a StatefulSet template is undesired, a webhook must
be introduced to mark Pods as paused when they are created. This adds a layer
of complexity to an orchestrator operator, since it needs both an operator
component that is capable of making changes to ApiServer, and a webhook that is
reading from a consistent migration state.

### Alternatives without any API changes

**Orphan Pods**: Users can orphan pods from a StatefulSet, migrate pods across a
namespace or cluster, and create a new StatefulSet to manage pods upon
migration. In the case of pod eviction or failure, pods will need to be manually
recreated, requiring manual intervention and constant monitoring.

**Backup/Restore**: Users can backup and restore a StatefulSet (and underlying
storage) in a new namespace or cluster. Doing so requires the existing
StatefulSet to be deleted, for underlying storage to be backed up and restored,
resulting in downtime for the stateful application.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
