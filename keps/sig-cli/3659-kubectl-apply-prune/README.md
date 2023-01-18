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
# KEP-3659: ApplySet: kubectl apply --prune redesign and graduation strategy

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
- [Background](#background)
  - [Use case](#use-case)
  - [Feature history](#feature-history)
  - [Current implementation](#current-implementation)
  - [Problems with the current implementation](#problems-with-the-current-implementation)
    - [Correctness: object leakage](#correctness-object-leakage)
    - [Scalability](#scalability)
    - [UX: easy to trigger inadvertent over-selection](#ux-easy-to-trigger-inadvertent-over-selection)
    - [UX: flag changes affect correctness](#ux-flag-changes-affect-correctness)
    - [UX: difficult to use with custom resources](#ux-difficult-to-use-with-custom-resources)
    - [Sustainability: incompatibility with server-side apply](#sustainability-incompatibility-with-server-side-apply)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability-1)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

When creating objects with `kubectl apply`, it is frequently desired to make changes to the config that remove objects and then re-apply and have those objects deleted. Since Kubernetes v1.5, an alpha-stage `--prune` flag exists to support this workflow: it deletes objects previously applied that no longer exist in the source config. However, the current implementation has fundamental design flaws that limit its performance and lead to surprising behaviours. This KEP proposes a safer and more performant implementation for this feature as a second, independent alpha. The new implementation is based on a low-level concept called "ApplySet" that other, higher-level ecosystem tooling can build on top of to enhance their own higher-level object groupings with improved interoperability.

## Motivation

### Goals

- MUST use a pruning set identification algorithm that remains accurate regardless of what has changed between the previous and current sets
- MUST use a pruning set identification algorithm that scales to thousands of resources across hundreds of types
- MUST natively support custom resources
- MUST provide a way to accurately preview which objects will be deleted
- MUST support namespaced and non-namespaced resources; SHOULD support them within the same operation
- SHOULD use a low-level "plumbing" object grouping mechanism over which more sophisticated abstractions can be built by "porceline" tooling.
<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->

### Non-Goals

- MUST NOT formalize the grouping of objects under management as an "application" or other high-level construct
- MUST NOT require users to manually/independently construct the grouping, which would be a significant reduction in UX compared to the current alpha
- MUST NOT require any particular CRDs to be installed
- MAY still have limited performance when used to manage thousands of resources of hundreds of types in a single operation (MUST NOT be expected to overcome performance limitations of issuing many individual deletion requests, for example)

## Background

### Use case

The pruning feature enables kubectl to automatically clean up previously applied objects that have been removed from the current configuration set.

Adding the `--prune` flag to kubectl apply adds a deletion step after objects are applied, removing all objects that were previously applied AND are not currently being applied: `{objects to prune (delete)} = {previously applied objects} - {currently applied objects}`.

In the illustration below, we initially apply a configuration set containing two objects: Object A and Object B. Then, we remove Object A from our configuration and add Object C. When we re-apply our configuration with pruning enabled, we expect Object A to be deleted (pruned), Object B to be updated, and Object C to be created. This basic use case works as expected today.

<img src=initial-apply.png width=500px>
<img src=subsequent-apply.png width=500px>

### Feature history

The `--prune` flag (and dependent `--prune-whitelist` and `--all` flags) were added to `kubectl apply` back in [Kubernetes v1.5](https://github.com/kubernetes/kubernetes/commit/56a22f925f6f1fd774ad1ae9e04bcf8d75bbde63). Twenty releases later, this feature is still in alpha, as documented in `kubectl apply -h` (though interestingly not on the flag doc string itself, or during usage):

<details>
<summary>Relevant portion of `kubectl apply -h`</summary>

```shell
$ kubectl version --client --short
Client Version: v1.25.2

$ kubectl apply -h
Apply a configuration to a resource by file name or stdin. The resource name must be specified. This resource will be
created if it doesn't exist yet. To use 'apply', always create the resource initially with either 'apply' or 'create
--save-config'.

 JSON and YAML formats are accepted.

 Alpha Disclaimer: the --prune functionality is not yet complete. Do not use unless you are aware of what the current
state is. See https://issues.k8s.io/34274.

Examples:
  # Note: --prune is still in Alpha
  # Apply the configuration in manifest.yaml that matches label app=nginx and delete all other resources that are not in
the file and match label app=nginx
  kubectl apply --prune -f manifest.yaml -l app=nginx

  # Apply the configuration in manifest.yaml and delete all the other config maps that are not in the file
  kubectl apply --prune -f manifest.yaml --all --prune-whitelist=core/v1/ConfigMap

Options:
    --all=false:
	Select all resources in the namespace of the specified resource types.
    --prune=false:
	Automatically delete resource objects, that do not appear in the configs and are created by either apply or
	create --save-config. Should be used with either -l or --all.
    --prune-whitelist=[]:
	Overwrite the default whitelist with <group/version/kind> for --prune
    -l, --selector='':
	Selector (label query) to filter on, supports '=', '==', and '!='.(e.g. -l key1=value1,key2=value2). Matching
	objects must satisfy all of the specified label constraints.
```
</details>

The reason for this stagnation is that the implementation has fundamental limitations that limit performance and cause unexpected behaviours.

Acknowledging that pruning could not be progressed out of alpha in its current form, SIG CLI created a proof of concept for an alternative implmentation in the [cli-utils](https://github.com/kubernetes-sigs/cli-utils) repo in 2019 (initially [moved over](https://github.com/kubernetes-sigs/cli-utils/pull/1) from [cli-experimental#13](https://github.com/kubernetes-sigs/cli-experimental/pull/13)). This implementation was proposed in [KEP 810](https://github.com/kubernetes/enhancements/pull/810/files), which did not reach consensus and was ultimately closed. In the subsequent three years, work continued on the proof of concept, and other ecosystem tools (notably `kpt live apply`) have been using it successfully while the canoncial implementation in k/k has continued to stagnate.

At Kubecon NA 2022, @seans3 and @KnVerey led a session discussing the limitations of the prune approach in kubectl apply. The consensus was:
- `kubectl apply --prune` is very difficult to use correctly.
- Any changes to existing behavior are likely to break existing users.  - Although `--prune` is technically in alpha, breaking existing workflows is likely to be unpopular. If the new solution is independent of the existing alpha, the alpha will need to be deprecated using a beta (at minimum) timeline, given how long it has existed.
- There are several solutions in the community that have broadly evolved to follow the label pattern, and typically store the label and the list of GVKs on a parent object.  Some solutions store a complete list of objects.
- We could likely standardize and support the existing approaches, so that they could be more interoperable.  kubernetes would define the “plumbing” layer, and leave nice user-facing “porcelain” to tooling such as helm.
- By defining a common plumbing layer, tools such as kubectl could list existing “applysets”, regardless of the tooling used to install them.
- `kubectl apply --prune` could use this plumbing layer as a new pruning implementation that would address many of the existing challenges, but would also simplify adoption of tooling such as Helm or Carvel.

### Current implementation

The implementation of this feature is not as simple as the illustration above might suggest at first glance. The core of the reason is that the previously applied set is not specifically encoded anywhere by the previous apply operation, and therefore that set needs to be dynamically discovered.

Several different factors are used to select the set of objects to be pruned:

1. **GVK allowlist**: A user-provided ( via `--prune-whitelist` until v1.26, `--prune-allowlist` in v1.26+) or defaulted list of GVK strings identifying which resources kubectl will consider for pruning. The default list is hardcoded. [[code](https://github.com/kubernetes/kubernetes/blob/e39a0af5ce0a836b30bd3cce237778fb4557f0cb/staging/src/k8s.io/kubectl/pkg/util/prune/prune.go#L28-L50)]
1. **namespace** (for namespaced resources): `kubectl` keeps track of which namespaces it has "visited" during the apply operation and considers both them and the objects they contain for pruning. [[code](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kubectl/pkg/cmd/apply/prune.go#L78)]
1. **the `kubectl.kubernetes.io/last-applied-configuration` annotation**: kubectl uses this as the signal that the object was created with `apply` as opposed to by another kubectl command or entity. Only objects created by apply are considered for pruning. [[code](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kubectl/pkg/cmd/apply/prune.go#L117-L120)]
1. **labels**: pruning forces users to specify either `--all` or `-l/--selector`, and in the latter case, the query selecting resources for pruning will be constrained by the provided labels (note that this flag also constrains the resources applied in the main operation) [[code](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/kubectl/pkg/cmd/apply/prune.go#L99)]

For a more detailed walkthrough of the implementation along with examples, please see [kubectl apply/prune: implementation and limitations](https://docs.google.com/document/d/1y747qL3GYMDDYHR8SqJ-6iWjQEdCJwK_n0z_KHcwa3I/edit#) by @seans3.

### Problems with the current implementation

#### Correctness: object leakage

 If an object is supposed to be pruned, but it is not, then it is leaked. This situation occurs when the set of previously applied objects selected is incomplete. There are two main ways this can happen:
 - **GVK allowlist mismatch**: the allowlist is hardcoded (either by kubectl or by the user) and as such it is not tied in any way to the list of kinds we actually need to manage to prune effectively. For example, the default allowlist will never prune PDBs, regardless of whether current or previous operations created them.
 - **namespace mismatch**: the namespace list is constructed dynamically from the _current_ set of objects, which causes object leakage when the current operation touches fewer namespaces than the previous one did. For example, if the initial operation touched namespaces A and B, and the second touched only B, nothing in namespace A will be pruned.

Related issues:
- https://github.com/kubernetes/kubernetes/issues/106284
- https://github.com/kubernetes/kubernetes/issues/57916
- https://github.com/kubernetes/kubernetes/issues/40635
- https://github.com/kubernetes/kubernetes/issues/66430
- https://github.com/kubernetes/kubectl/issues/555

#### UX: flag changes affect correctness

If the user changes the `--prune-allowlist` or `--selector` flags used with the apply command, this may radically change the scoping of the pruning operation, causing it over- or under-select resources. For example, if they add a new label to all their resources and adjust the `--selector` accordingly, this will have the side-effect of leaking ALL resources that should have been deleted during the operation (nothing will be pruned). On the contrary, if `--prune-allowlist` is expanded to include additional types or `--selector` is made more general, any objects that have been manually applied by other actors in the system may automatically get scoped in.

There was also a previous bad interaction with the `--force` flag, which was worked around by disabling that flag combination at the flag parsing stage.

Related issues:
- https://github.com/kubernetes/kubernetes/issues/89322
- https://github.com/kubernetes/kubectl/issues/1239

#### Scalability

To discover the set of resources to be pruned, kubectl makes a LIST query for each Group-Version-Resource (GVR) on the allowlist, for every namespace (if applicable): `GVR(namespaced)*Ns + GVR(global)`. For example, with the default list and one target namespace, this is 14 requests; with the default list and two namespaces, it jumps to 26. An obvious fix for some of the correctness issues described would be to get the full list of GVRs from discovery and query ALL of them, ensuring all previous objects are discovered. Indeed some tools do this, and pass the resulting list to kubectl's allowlist. But this strategy is clearly not performant, and many of the additional queries are wasted, as the GVRs in question are extremely unlikely to have resources managed via kubectl.

A related issue is that the identifier of ownership for pruning is the last-applied annotation, which is not something that can be queried on. This means we cannot avoid retrieving irrelevant resources in the LIST requests we make.

#### UX: easy to trigger inadvertent over-selection

The default allowlist, in addition to being incomplete, is unintuitive. Notably, it includes the cluster-scoped Namespace and PersistentVolume resources, and will prune these resources even if the `--namespace` flag is used. Given that Namespace deletion cascades to all the contents of the namespaces, this is particularly catastropic.

Because every `apply` operation uses the same identity for the purposes of pruning (i.e. has the same last-applied annotation), it is easy to make a small change to the scoping of the command that will inadvertantly cover resources managed by other operations, with potentially disasterous effects.

Related issues:
- https://github.com/kubernetes/kubectl/issues/1272
- https://github.com/kubernetes/kubernetes/issues/110905
- https://github.com/kubernetes/kubernetes/issues/108161
- https://github.com/kubernetes/kubernetes/issues/74414

#### UX: difficult to use with custom resources

Because the default allowlist is hard-coded in the kubectl codebase, it inherently does not include any custom resources. Users who want to prune custom resources necessarily need to specify their own allowlist and keep it up to date.

Related issues:
- https://github.com/kubernetes/kubectl/issues/1310

#### Sustainability: incompatibility with server-side apply

While it is not disabled, pruning does not work correctly with server-side apply today. If the objects being managed were created with server-side apply, or were migrated to server-side apply using a custom field manager, they will never be pruned. If they were created with client-side apply and migrated to server-side using the default field manager, they will be pruned as needed. The worst case is that the managed set includes objects in multiple of these states, leading to inconsistent behaviour.

One solution to this would be to use the presence of the current field manager as the indicator of eligibility for pruning. However, field managers cannot be queried on any more than annotations can, so are not a great for an identifier we want to select on. It can also be considered problematic that the default state for server-side applied objects includes at least two field managers, which are then all taken to be object owners for the purposes of pruning, regardless of their intent to use this power. In other words, we end up introducing the possibilty of multiple owners without the possiblity of conflict detection.

Related issues:
- https://github.com/kubernetes/kubernetes/issues/110893

### Related solutions in the ecosystem

The following popular tools have mechanisms for managing sets of objects, which are described briefly below. An ideal solution for kubectl's pruning feature would allow tools like these to "rebase" these mechanisms over the new "plumbing" layer. This possibilty could increase ecosystem coherence and interoperability, as well as provide a cleaner bridge from the baseline capabilities offered in kubectl to these more advanced tools.

#### Helm

**Pattern**: list of Group-Version-Kind-Namespace-Name (GVKNN) (from secret) + labels

Each helm chart installation is represented by a Secret object in the cluster.  The `type` field of the Secret is set to `helm.sh/release.v1`.  Objects that are part of the helm chart installation get annotations `meta.helm.sh/release-name` and `meta.helm.sh/release-namespace`, but the link to the “parent” Secret is somewhat obscure.    The list of GKs in use can be derived from the data encoded in the secret, but this data actually includes the full manifest.

#### Carvel kapp

**Pattern**: list of Group-Kinds (GK) (from configmap) + labels

Each kapp installation is represented by a ConfigMap object in the cluster.  The ConfigMap has a label `kapp.k14s.io/is-app: "”`.  Objects that are part of the kapp have two labels: `kapp.k14s.io/app=<number>` and `kapp.k14s.io/association=<string>`.  Getting from the parent ConfigMap to these values is again somewhat obscure.  The `app` label is encoded in a JSON blob in the “spec” value of the ConfigMap.  The `association` object includes an MD5 hash of the object identity, and varies across objects in a kapp.  The list of GKs in use is encoded as JSON in the “spec” value of the ConfigMap.

#### kpt

**Pattern**: list of Group-Kind-Namespace-Name (GKNN) (from ResourceGroup)

Kpt uses a ResourceGroup CRD, and can register that CRD automatically.    The ResourceGroup contains a full list of GKNNs for all managed objects.  Kpt calls this full list of objects - including the names and namespaces - an “inventory”.  Each object gets an annotation `config.k8s.io/owning-inventory`, where that annotation corresponds to a label on the ResourceGroup `cli-utils.sigs.k8s.io/inventory-id`

#### Google ConfigSync

**Pattern**: list of Group-Version-Kind-Namespace-Name (GVKNN) (from ResourceGroup)

Distinct sets of synchronized resources are represented by RootSync / RepoSyncs, along with a ResourceGroup that has the full inventory.  Each object has some annotations that define membership, including the same `config.k8s.io/owning-inventory` as is used by kpt.    As with other solutions, following the “chain” from RootSync/RepoSync to managed objects is somewhat obscure.


## Proposal

A v2-prunable "apply set" is associated with an object on the cluster.   We define a set of standardized labels and annotations that identify the “parent object” of the apply set and the “member objects” of that parent.  We operate at the plumbing layer; we aim to support:

- listing the parent objects efficiently (porcelain may expose this as listing groups of objects as managed by the various tools)
- listing the member objects for a specific parent object efficiently (porcelain may use this for advanced diffing and pruning, and for presenting objects grouped by their higher-level aggregation)
- basic apply-with-prune operations, where it creates or reuses a Secret in the cluster as the parent object.

### Apply Set

"Apply set" refers to a group of resources that are applied to the cluster by a tool. An apply set has a “parent” object of the tool’s preference. This “parent” object can be implemented using a ConfigMap, Secret, or a CRD of the tool’s choice.

“ApplySet” is used to refer to the parent object in this design document, though the actual concrete resource on the cluster will typically be of a different Kind.  We might think of ApplySet as a “duck-type” based on the `applyset.k8s.io/id` label proposed here.

### Member Object Labels

Objects that are part of an ApplySet should carry a standardized label, with a key of:

```yaml
applyset.k8s.io/part-of: <applysetkey>
```

The `<applysetkey>` can be chosen essentially arbitrarily (subject to the limits of label values).

We recommend that tooling uses a prefix that is its "tool name", e.g. `kubectl.` or `helm.`,
followed by some tooling-specific unique value.  We can investigate maintaining a formal registry,
but we initially reserve the following prefixes:

* `kubectl.` for kubectl
* `helm.` for helm
* `kpt.` for kpt
* `uid.` for where the suffix is the UID of the applyset object (e.g. `uid.e049464e-4583-4642-9649-93dcb0e96bd4`)
* `id.` for where the suffix is the group-kind followed by the namespace followed by the name, such as
  `id.configmaps.ns1.parent1`.  (While this is a little tricky to parse, it should be unique because
  neither namespaces nor object names allows dots.)

### ApplySet Object Labels

ApplySets should also have a “parent object” in the cluster. This ApplySet object can (in theory) be of any type.  For performance reasons we later propose limiting to ConfigMap, Secret and custom resources with a specific label. In future we may define a common CRD, but we believe we can achieve a reasonable user experience without defining one.  Many existing tools avoid using a CRD, so that they can be used by people without the cluster-admin permissions needed to install a CRD.  (This also avoids CRD versioning problems etc).

The ApplySet object should be labeled with:

```yaml
applyset.k8s.io/id: <applyset-id>
```

Implicit in this are a few assumptions:

- An object can be part of at most one ApplySet.  This is a limitation, but seems to be a good one in that objects that are part of multiple ApplySets are complicated both conceptually for users and in terms of implementation behaviour.
- An ApplySet object can be part of another ApplySet (sub-ApplySets).

How the ApplySet object is specified is a tooling decision.  Gitops based tooling may choose to make the ApplySet object explicit in the source git repo.  Other tooling may choose to leverage their existing concepts, for example mapping to a Secret or ConfigMap that they are creating already.  The tooling is responsible for consistently specifying the ApplySet object across apply invocations, so that pruning can be done consistently.


For kubectl specifically, we propose supporting but not requiring explicitly
provided parent objects, with automatic object creation in the latter case.
This is explained in more detail below.



### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

#### Story 2

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

### Efficient Listing of ApplySets

In order to support listing all the applysets in theory we would need to query all GKs with a label selector.  However, we can reduce the set of GKs that need to be queried with two optimizations:

* For built-in types, we limit to ConfigMaps and Secrets.

* For custom resources, we require that CRDs that define types that can be used as ApplySet objects be labeled with a label with a name of `applyset.k8s.io/role/applyset`.

The value is currently ignored, but implementors should set an empty value to
be forwards-compatible with future evolution of this convention.

A `kubectl apply list-applysets -n ns` command would therefore do the following queries:

```bash
kubectl get secret -n ns -l applyset.k8s.io/id # --only-partial-object-metadata
kubectl get configmap -n ns -l applyset.k8s.io/id # --only-partial-object-metadata

for crd in $(kubectl get crd -l applyset.k8s.io/role/applyset); do
kubectl get $crd -n ns -l  applyset.k8s.io/id  # --only-partial-object-metadata
done
```

Optimizations are possible here. For example we can likely cache the list of CRDs.  However, while the number of management tools may grow, the number of management ecosystems is relatively small, and we would expect a given cluster to use only a fraction of the management ecosystems.  So the number of queries here is likely to be small.  Moreover these queries can be executed in parallel, we can now rely on priority-and-fairness to throttle these appropriately without needing to self-throttle client-side.

In future, we may define additional “index” mechanisms here to further optimize this (controllers or webhooks that watch these labels and populate an annotation on the namespace, or support in kube-apiserver for cross-object querying).  However the belief is that this is likely not needed at the current time.

### Efficient Listing of ApplySet Contents

We want to support efficient listing of the objects that belong to a particular applyset.  In theory, this again requires the all-GK listing (with a label filter).  An advantage of this approach is that this remains an option: as we implement optimizations we may also periodically run a “garbage collector” to verify that our optimizations have not leaked objects, perhaps `kubectl apply fsck` or a plugin.

We already know the label selector for a given applyset, by convention: we take the id from the value of the `applyset.k8s.io/id` label, and that becomes the required value of the `applyset.k8s.io/part-of` label.

In order to narrow the list of Group-Kinds (GKs), we require the applyset object to define the list of GKs in use.  The plumbing tooling can optimize selection of the objects in this applyset based on this list.

“Porcelain” tooling can still perform tooling-specific GK identification. Tooling generally can use their existing mechanisms, be they more efficient or more powerful or just easier to continue to support.  However, by using the standardized labels proposed here, they can interoperate with other tooling and enjoy protection against their resources being changed by another tool (such as kubectl).  Tooling is not required to implement these labels, and we are not introducing new behaviours for “unenlightened” objects.

To identify the GKs in use, kubectl and applyset-compatible tooling shall add an annotation `applyset.k8s.io/contains-group-kinds`; we use an annotation instead of a label because the annotation can be larger, and because we do not need to select on this value.  The value of the annotation shall be a comma separated list of the group-kinds, in the fully-qualified name format, i.e. `<resourcename>.<group>`.  An example annotation value might therefore look like: `certificates.cert-manager.io,configmaps,deployments.apps,secrets,services`.  Note that we do not include the version; formatting is a different concern from “applyset membership”.

To avoid spurious updates and conflicts, the list must be sorted alphabetically.  The list may include GKs where there are no resources actually labeled with the applyset-id, but to avoid churn this should be avoided and ideally only be a transitional step during an apply or prune operation.

We may in future define additional mechanisms, such as supporting a field
selector on the CRD that identifies a strongly-typed list; we do not plan
to do this in the alpha.

Where no list of GKs can be determined the tooling should warn that we are performing a full-GK scan.  As discussed in the interoperability section, tooling should not populate the annotation unless it believes itself to be the manager of an applyset.

In pseudo-code, to discover the existing members of an applyset:

```bash
for-each gk in $(split group-kind-annotation); do
kubectl get $gk -n ns -l  applyset.k8s.io/id  # --only-partial-object-metadata
done
```

### Cluster-scoped ApplySets

We need to support ApplySets that are cluster-scoped, for example ApplySets that include installation of CRDs (such as cert-manager).  The mechanisms we have defined here work for cluster-scoped ApplySets.  Today’s tooling will create an managing object in a namespace, and likely a cluster-scoped CRD would be more intuitive than a namespace-scoped resource. However, no additional explicit support for cluster-scoped ApplySets is required or proposed at the current time (but cf the next section for cross-namespace considerations).

### Cross-Namespace ApplySet Contents

When querying for ApplySet contents, an ApplySet could contain cluster-scoped resources or could contain resources in other namespaces.  Querying for this content is generally going to require more permissions and be slower, so we would like to avoid over-querying here.

Best practice is likely to avoid ApplySets spanning namespaces.  However, sometimes this is unavoidable - particularly when managing cluster-scoped objects - and the “plumbing” tooling cannot enforce this restriction.

Where a GK is known to be part of the ApplySet and is cluster-scoped, we should naturally query for those objects at cluster scope; any permission problems here should be surfaced as errors.  When a tool cannot determine the list of GKs for an ApplySet, it may support “discovery”, likely warning that the ApplySet does not define a list of GKs, and then attempt to perform cluster-scoped queries, likely warning if there are insufficient permissions.

For the alpha scope, this functionality will be restricted to subcommands of apply
(like the migrate functionality).  We will likely want a command similar in spirit
to `fsck`.  We will add warnings/suggestions to the main "apply" flow when we detect
problems that might require a full-scan / discovery.  We may extend this based on
user-feedback from the alpha.

For GKs that are namespace-scoped, we would normally expect those to be part of an ApplySet object in the same namespace.  We define an additional annotation however for cross-namespace ApplySets:  `applyset.k8s.io/additional-namespaces`.  The value of this annotation is a comma-separated list of the names of the namespaces (other than the ApplySet namespace) in which objects are found, for example `default,kube-system,ns1,ns2`.  Note that there is no need to include this specifically for cluster-scoped objects, as those are covered by the group-kind list.  We reserve the empty value.  If this annotation is present, the tooling will query namespace-scoped resources in those namespaces in addition to the namespace of the ApplySet object (if any).  If this annotation is not present on a namespace-scoped ApplySet parent object, the tooling will query namespace-scoped resources only in the same namespace as the ApplySet parent object.  If the annotation is not present on a cluster-scoped ApplySet parent object, the tooling will not query namespace-scoped resources at all and should output an error if given namespace-scoped GKs.

As with `applyset.k8s.io/contains-group-kinds`, this list of namespaces must be sorted alphabetically, and should be minimal (ideally other than during apply and prune operations).

As cross-namespace ApplySets are not particularly encouraged, we do not currently optimize this further.  In particular, we do not specify the GKs per namespace.  We can add more annotations in future should the need arise.

Where an applyset includes both cluster-scoped and namespace-scoped resources,
by reducing to the above cases.  The set of relevant resources is determined
by consulting the `applyset.k8s.io/contains-group-kinds` annotation; whether
those kinds are cluster-scoped or namespace-scoped are found using the normal
API discovery mechanisms.  Cluster-scoped resources ignore the
`applyset.k8s.io/additional-namespaces` annotation, namespace-scoped resources
combine the current namespace from the applyset object (if that is itself
namespace-scoped) with the namespaces from the annotation.

### Objects with owner references

If an object in the set retrieved for pruning has owner references,
tooling should verify that those references match the ApplySet parent.
If they do, the tool should proceed as normal. If they do not, the
tooling should consider this an ownership conflict and throw an error.

We are taking this stance initially to be conservative and ensure that
use cases related to objects bearing owner references are surfaced.
In the future, we could downgrade this stance to recommending a warning,
or to considering owner references orthogonal and ignoring them entirely.

### Versioning

The labels and annotations proposed here are not versioned.  Putting a version
into the key would forever complicate label-selection (because we would have to
query over multiple labels).  However, if we do need versioning, we can introduce
versions by including a prefix like `v2:` (and we would likely do
`v2:[...` or `v2:{...`).  Colons are not valid in namespaces nor in group-kinds,
so there is no conflict with the existing (v1) usage described here.  Labels cannot
include a `:` character, so if we needed to version a label we can use `v2.`,
however our usage of labels is primarily around matching opaque applyset-id
tokens and thus seems unlikely to need versioning.

### Kubectl Reference Implementation

We will develop a reference implementation of this specification in kubectl, with the intention of providing a supportable replacement for the current alpha `kubectl apply --prune` semantics.  Our intention is not to change the behavior of the existing `--prune` functionality, but rather to produce an alternative that users will happily and safely move to.  We can likely trigger the V2-semantics when the user specifies an applyset flag, so that this is intuitive and does not break existing prune users.

>  <<[UNRESOLVED @justinsb @KnVerey]>>
>
> Add summary of exact commands and flags proposed
>
> <<[/UNRESOLVED]>>

The proposal may evolve at the coding/PR stage, but the current plan is as follows.

We will add a flag to kubectl apply, `--applyset=<id>`.  If specified, that will change the behavior of apply and prune to include the new functionality.

We may also develop additional `kubectl apply` subcommands, such as the `fsck` functionality (perhaps `applyset-verify-integrity`), to build complementary functionality that is helpful for adoption.

We intend to treat the flag and any subcommands as alpha commands initially.  During alpha, users will need to set an environment variable (e.g. KUBECTL_APPLYSET_ALPHA) to make the flag available.

When `--applyset` is specified, kubectl will automatically create a secret named `applyset-<id>`, in the targeted namespace. kubectl will populate the labels/annotations on applied objects as described here.

>  <<[UNRESOLVED @justinsb @KnVerey]>>
>
> Will we not support cluster-scoped applysets initially? There is no
> obvious choice of cluster-scoped built-in resource to use for them.
>
> <<[/UNRESOLVED]>>

In future, we may support a applyset object being provided as part of
the input resources, but we will do so in response to user demand and
user feedback, and do not have existing plans to do so in the alpha
scope.

When pruning with `--applyset`, kubectl will delete objects that are labeled as part of the applyset of objects, but are not in the list of objects being applied.  We expect to reuse the existing prune logic and behavior here, except that we will select objects differently (although as existing prune is also based on label selection, we may be able to reuse the bulk of the label-selection logic also).  Dry-run will be supported, as will `kubectl diff --applyset=id`.

We will not support all the combinations of flags that apply and prune currently does with the new `--applyset` flag; this is not a breaking change and our goal is to support the existing safe workflows, not the full permutations of all flags.  We can add support for additional flag combinations based on user feedback, in many cases the meaning is ambiguous and we will need to collaborate with kubectl users to understand their true intent and determine how best to support it.  In particular, we will not support the `--selector` flag in combination with `--applyset` until we understand the intent of users here. Flags associated with the prune alpha (`--prune`, `--prune-allowlist`, and `--all`) will also specifically be excluded.

We will detect “overlapping” applysets where objects already have a different applyset label, and initially treat this an error (we may add “adopt” or “force” functionality later).

During implementation of the alpha we will explore to what extent we can
optimize this overlap discovery, particularly in conjunction with
server-side-apply which does not require an object read before applying.
A richer apply tooling than kubectl does will likely establish watches
on the objects before applying them, to monitor object health and status.
However, this is out of scope for kubectl and thus we will likely have to
optimize differently for kubectl.  In the worst case, we will have to fetch
the objects before applying (with a set of label-filtered LIST requests),
we will explore to what extent that can be amortized over other kubectl
operations in alpha.  One interesting option may be to use the fieldManager,
choosing a fieldManager that includes the applyset ID to automatically
detect conflicts (by _not_ specifying force); we intend to explore
how this looks in practice and whether other options present themselves.

We differentiate between "adoption" (taking over management of a set of
objects created by another tool), vs "migration" (taking over management of
a set of objects created with the existing pruning mechanism).

We will not support "adoption" of existing applysets initially, other than
by re-applying "over the top".  Based on user feedback, we may require a flag
to adopt existing objects / applysets.

In the alpha scope, we will explore suitable "migration" tooling for moving
from existing `--prune` objects.  Note that migration is not trivial, in that
different users may expect different behaviors with regard to the GKs selected
or the treatment of objects having/lacking the `last-application-configuration`
annotation.  We intend to create `migrate` as an explicit subcommand of `apply`,
rather than trying to overload the "normal flow" apply command.

### Tooling Interoperability

There is a rich ecosystem of existing tooling that we hope will adopt these labels and annotations.  So that different tooling can interoperate smoothly, we define some requirements for safe interoperability here.

For read operations, we expect that using different tooling shall generally be safe.  As these labels do not collide with existing tooling, we would expect that objects installed with existing tooling would be invisible to the porcelain tooling until they had been updated to include the labels.  We do not propose to implement “bridges” to existing tooling, rather as the proposal here is lightweight and small, it makes more sense to update the existing tooling.  We may add warnings such as “applysets using an old version of X detected, upgrade to v123 of X to work with those applysets”.

For write operations, we need to be more careful.  Deleting an applyset using the “wrong tool” should be safe, but we will likely include a confirmation if deleting an applyset using the “wrong tool”, particularly unknown tools.  We expect that porcelain tools may define richer behavior on delete, so this is the equivalent of pulling the power cable on an applyset instead of performing a clean shutdown.

We do not believe that update operations are safe if using the “wrong tool”, because that tooling may have additional metadata that would then not be updated.  Tooling should generally reject applying on top of unknown applysets.  Porcelain tooling may choose to recognize other tooling and implement specific logic there; in particular this may be useful for moving between different major versions of the same tooling.  We may implement a `--force` flag, but this would likely be logically equivalent in outcome to a full applyset deletion and recreation, though with the potential (but not the guarantee) to be less disruptive.

In order to identify usage of the "wrong tool", we define a further annotation
`applyset.k8s.io/tooling`, which tooling can set to protect their applysets.
The value should be something like `kubectl/v1.27.3` or `helm/v2.0.6`,
i.e. `<toolname>/<semver>`.  Compatible porcelain tooling should recognize that
a different tool is managing the applyset and provide an appropriate warning.
We intend to explore the trade-off between safety and user-friendly behaviour
here, during evolution of the feature in alpha and beyond.

### Security Considerations

Generally RBAC gives us the permissions we need to operate safely here.  No special permissions are granted - for example there is no “backdoor” to read objects simply because they are part of an applyset.  In order to mark an object as part of an applyset, we need permission to write to that object.  If we have permission to update an applyset object, we can “leak” objects from the optimized search, but we can support a “fsck” scan that does not optimize the search, and generally the ability to mutate the applyset carries this risk.  Using a more privileged object, such as a secret or a dedicated CRD can limit this risk.

Known Risks:
- A user without delete permission but with update permission could mark an object as part of an applyset, and then an administrator could inadvertently delete the object as part of their next apply/prune. This is also true of the current pruning implementation (by setting the last-applied-configuration annotation to any value). Mitigation: We will support the dry-run functionality for pruning.  Webhooks or future enhancements to RBAC/CEL may allow for granular permission on labels.


### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
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

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

We would like this functionality to replace the existing uses of `--prune`.  We have
chosen to take an approach that is a better and supportable evolution of the existing
label based pruning, rather than a revolutionary new approach, to try to enable migration.

At some point we might deprecate the existing `--prune` functionality, to encourage users
to migrate.  A suitable timeline would probably be to begin deprecation at beta, and to
not remove the functionality until at least applyset reaches GA + 1 version.  However, we
intend to gather feedback from early alphas here - in particular we want to discover:

* Are there `--prune` use-cases we do not cover?
* Do existing `--prune` users migrate enthusiastically (without any "nudge" from deprecation)?

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

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

## Alternatives

### Full GKNN listing

Instead of encoding a list of GKs to scope in, we could encode a the full list of GKNN object references, making the ApplySet parent object a (somewhat) human-readable inventory of the set. The reason for not choosing this approach is that we do not think it would actually allow us to further optimize the implementation in practice, and that its additional detail would make it more prone to desynchronization.

The reason it does not optimize performance in practice is that we're considering the source of truth for membership to be the `part-of` annotations on the resources themselves. This is useful for visibility and for ownership conflict avoidance, but it means we must retrieve the objects themselves to check the source of truth rather than relying on the GVKNN. Since individual GET calls are far more expensive than LISTs in the common case for pruning, in practice, we would end up extracting the GK list from any GKNN list and make the same calls we would have with just a GK list. If it is deemed worthwhile, we could indeed do this, and it would allow an additional layer of in-band drift detection via comparison of the precise list to the set of current labelled resources.

Alternatively, we could omit the `part-of` label entirely (which leaves no means of ownership conflict management), or consider the GKNN list the source of truth (which leaves a much wider vector for object leakage in practice than GK listing does, in our opinion).

### OwnerRefs

We could use ownerRefs to track applyset membership.  A significant advantage of ownerRefs is that pruning is done automatically by the kube-apiserver, which runs a garbage collection algorithm to automatically delete resources that are no longer referenced.
However today the apiserver does not support an efficient way to query by ownerRef (unlike labels, where we can specify a label selector to the kube-apiserver).  This means we can’t efficiently list the objects in an applyset, nor can we efficiently support a dry-run / preview (without listing all the objects).  Moreover, there is no support for cross-namespace ownerRefs, nor for a namespace-scoped object owning a cluster-scoped object.  These are not blockers per-se, in that as a community we control the full-stack.  However, the scoping issues are more fundamental and have meant that existing tooling such as helm has not used ownerRefs, so this would likely be a barrier to adoption by existing tooling.  We do not preclude tooling from using ownerRefs; we are simply proposing standardizing the labels to provide interoperability with existing tooling and the existing kube-apiserver.

### ManagedFields

We could use managedFields to track ownership, however again this is not standardized and the kube-apiserver does not support an efficient way to query by managedFields manager (today).  This too may be an interesting area for porcelain tooling to explore, and we should likely be defining some conventions around field manager names, but that is complementary to and out of scope of the current proposal.  It does not appear viable today to define an approach using managedFields that can be implemented efficiently and in a way that is adoptable by the existing ecosystem.


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
