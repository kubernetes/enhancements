<!--
**Note:** When your KEP is complete, all of these comment blocks should be removed.

Follow the guidelines of the [documentation style guide].
In particular, wrap lines to a reasonable length, to make it
easier for reviewers to cite specific portions, and to minimize diff churn on
updates.

[documentation style guide]: https://github.com/kubernetes/community/blob/master/contributors/guide/style-guide.md

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
# KEP-5982: Pod Deletion Preference API

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
    - [Story 1: HPA scales down the idle pods](#story-1-hpa-scales-down-the-idle-pods)
    - [Story 2: Model inference scales down on KV cache utilization](#story-2-model-inference-scales-down-on-kv-cache-utilization)
    - [Story 3: Node autoscaler consolidates a node](#story-3-node-autoscaler-consolidates-a-node)
    - [Story 4: Two actors, one answer](#story-4-two-actors-one-answer)
    - [Story 5: Custom workload controller as a consumer](#story-5-custom-workload-controller-as-a-consumer)
    - [Story 6: Zone-balanced scale-down](#story-6-zone-balanced-scale-down)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Overview](#overview)
  - [API](#api)
    - [Shared types](#shared-types)
    - [PodDeletionPreference (input)](#poddeletionpreference-input)
    - [PodDeletionPreferenceSlice (input scores)](#poddeletionpreferenceslice-input-scores)
    - [PodDeletionOrder (output)](#poddeletionorder-output)
    - [Example](#example)
    - [Validation](#validation)
  - [Scores and aggregation semantics](#scores-and-aggregation-semantics)
  - [Slices](#slices)
  - [Binding a preference to its workload](#binding-a-preference-to-its-workload)
  - [The aggregator](#the-aggregator)
  - [The ReplicaSet controller as consumer](#the-replicaset-controller-as-consumer)
  - [Coexistence with pod-deletion-cost](#coexistence-with-pod-deletion-cost)
  - [The HPA as writer](#the-hpa-as-writer)
    - [Converting metrics to scores](#converting-metrics-to-scores)
  - [Out-of-tree writers](#out-of-tree-writers)
  - [Hub mode](#hub-mode)
  - [Consistency model](#consistency-model)
  - [Permissions](#permissions)
  - [Feature gates](#feature-gates)
  - [Future Work](#future-work)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
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
  - [Per-actor objects, consumer aggregates](#per-actor-objects-consumer-aggregates)
  - [Shared object per workload, consumer aggregates through a library](#shared-object-per-workload-consumer-aggregates-through-a-library)
  - [Shared object per workload, controller writes status](#shared-object-per-workload-controller-writes-status)
  - [Field on the Scale subresource](#field-on-the-scale-subresource)
  - [Pod annotation only](#pod-annotation-only)
  - [Aggregated API server](#aggregated-api-server)
  - [On-demand request and response](#on-demand-request-and-response)
  - [Other ways to bind a preference to its workload](#other-ways-to-bind-a-preference-to-its-workload)
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
Several controllers have good reasons to say which pods should go first
when a workload scales down. The HorizontalPodAutoscaler (HPA) knows
which pods are idle. A node autoscaler such as Karpenter knows which
pods sit on a node it is about to remove. Today the only way to say
this is the `controller.kubernetes.io/pod-deletion-cost` annotation. It
holds one number per pod, so two controllers can only overwrite each
other, and changing it means one PATCH per pod.

This KEP adds an API in the `lifecycle.k8s.io` group:

- `PodDeletionPreference`: one object per writer per workload. It says
  who the writer is and which workload it is about.
- `PodDeletionPreferenceSlice`: the writer's scores, in chunks of at
  most 200 pods, owned by the preference.
- `PodDeletionOrder`: one object per workload, written by a new
  controller in kube-controller-manager (the aggregator). It holds the
  summed and sorted answer.

The ReplicaSet controller reads one `PodDeletionOrder` from its cache
and uses it as one step in its existing deletion sort. The HPA is the
first in-tree writer. Preferences are advisory. Nothing waits on them.

## Motivation

<!--
This section is for explicitly listing the motivation, goals, and non-goals of
this KEP.  Describe why the change is important and the benefits to users. The
motivation section can optionally provide links to [experience reports] to
demonstrate the interest in a KEP within the wider Kubernetes community.

[experience reports]: https://github.com/golang/go/wiki/ExperienceReports
-->
Today, the ReplicaSet controller picks pods to delete with a fixed sort:
unscheduled first, then by phase, readiness, pod-deletion-cost,
co-location, time ready, restarts and age. That sort has no input from
the controllers that know most about the pods.

The HPA computes per-pod metrics every sync. It can tell that one pod
serves 0 requests and another serves 500, but it can only lower the
replica count. The ReplicaSet then removes the youngest ready pod,
which may be the busy one. Scale-down then costs more than it should.

Node autoscalers face the same wall from the other side. Karpenter
wants the pods on a node it is consolidating to be the ones that go
when the owner scales down. Today it does this with pod-deletion-cost,
one PATCH per pod, and it collides with anyone else using the same
annotation.

A second motivation is Graceful Node Shutdown. During a shutdown the
kubelet terminates pods tier by tier and, from v1.39 ([KEP-2000](https://github.com/kubernetes/enhancements/issues/2000)), admits
new pods only above the tier being terminated. The DaemonSet controller
cannot see the tier, so it keeps creating pods the kubelet rejects
([KEP-6249](https://github.com/kubernetes/enhancements/issues/6249)). A kubelet-written preference with a rule such as "pods
below priority X" is the planned fix. Rules and node-targeted
preferences are not in v1alpha1, but the API is shaped so they can be
added. See [Future work](#future-work).

### Goals

<!--
List the specific goals of the KEP. What is it trying to achieve? How will we
know that this has succeeded?
-->
- Let several independent controllers say which pods should be deleted
  first, without overwriting each other and without touching pods.
- Give the ReplicaSet controller one object to read, with the answer
  already summed and sorted, so it does no aggregation.
- Keep preferences advisory. Writers and readers never wait for each
  other. Scale-down stays synchronous.
- Define how the new API coexists with the
  `controller.kubernetes.io/pod-deletion-cost` annotation.
- Make the HPA the first in-tree writer, behind its own gate.

### Non-Goals

<!--
What is out of scope for this KEP? Listing non-goals helps to focus discussion
and make progress.
-->
- Changing how many pods are deleted, or when. Only which ones.
- StatefulSet, DaemonSet or Job consumers. The ReplicaSet controller is
  the first consumer. Others can follow with the same contract.
- Rules that name pods by a property instead of by name, node-targeted
  preferences, and the kubelet as a writer. These are follow-ups; see
  [Future Work](#future-work).
- Weighting actors against each other, or negative scores that say
  "keep this pod". All actors count equally and scores only say "delete
  first".
- Deletion plans that depend on the number or combination of pods
  removed, such as topology-aware or gang-aware scale-down. This KEP
  adds no zone-aware writer and does not guarantee balanced zones
  ([kubernetes/kubernetes#124149](https://github.com/kubernetes/kubernetes/issues/124149)).
  An actor can approximate it with per-pod scores; see
  [Story 6](#story-6-zone-balanced-scale-down).
- Deprecating `controller.kubernetes.io/pod-deletion-cost`.

## Proposal

<!--
This is where we get down to the specifics of what the proposal actually is.
This should have enough detail that reviewers can understand exactly what
you're proposing, but should not include things like API designs or
implementation. What is the desired outcome and how do we measure success?.
The "Design Details" section below is for the real
nitty-gritty.
-->
Each writer, called an actor, owns one `PodDeletionPreference` per pod
owner it has an opinion about. The pod owner is the object named in
the pods' owner reference with `controller: true`, which for a
Deployment is a ReplicaSet. The actor copies the owner's kind, name and
UID from the pod, so it never reads the owner. The preference is
written once. The scores go into `PodDeletionPreferenceSlice` objects
owned by the preference, each holding at most 200 pods.

A new controller in kube-controller-manager, the aggregator, watches
preferences and slices. It groups preferences by owner UID, merges
their slices, adds the scores per pod, sorts, and writes one
`PodDeletionOrder` per owner.
The ReplicaSet controller, when it must delete pods, reads its
`PodDeletionOrder`, drops entries whose UID does not match a pod it
owns, and uses the remaining scores as one step in its sort. Higher
score means delete first. Pods without a score are neutral. If there
is no order, the sort is the one from today.





### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1: HPA scales down the idle pods

As an application owner, I run a Deployment behind an HPA that scales on
requests per second. When traffic drops, I want the HPA to remove the
pods that serve the fewest requests, not the youngest ones, so in-flight
work is not cut.

#### Story 2: Model inference scales down on KV cache utilization

As an ML platform owner, I run an LLM inference server 
as a Deployment behind an HPA that scales on KV cache utilization,
exposed as a pods metric. Each replica holds the KV cache of its active
sequences, so removing a busy replica aborts long generations and throws
away warm cache.

#### Story 3: Node autoscaler consolidates a node

As a cluster operator, I run a node autoscaler that consolidates
under-utilized nodes. When a workload scales down, I want the pods on
the node being consolidated to be the ones removed, so the node drains
on its own.

#### Story 4: Two actors, one answer

As a platform team, I run both the HPA and a node autoscaler against the
same Deployment. I want both opinions to count, so a pod that is both
idle and on a draining node is the first to go.

#### Story 5: Custom workload controller as a consumer

As the author of a controller that owns its own pods, I want the same
signals the ReplicaSet controller gets, without reimplementing the
aggregation so I could implement the scale down logic myself.

#### Story 6: Zone-balanced scale-down

As an application owner, I spread a Deployment across zones. I want a
controller to score the pods in the fullest zones, so scale-down keeps
the zones even.

### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

- **The order can be one sync behind.** The HPA writes its slices and
  then patches Scale in the same sync. The ReplicaSet controller reacts
  to the Scale patch within milliseconds, while the new slice still has
  to reach the aggregator, be recomputed, written, and arrive in the
  ReplicaSet controller's cache. So the ReplicaSet deletes by the order
  computed from the previous sync. In practice this is mitigated by the
  HPA's [scale-down stabilization window](https://kubernetes.io/docs/concepts/workloads/autoscaling/horizontal-pod-autoscale/#stabilization-window),
  300 seconds by default: the HPA writes scores every sync (15 seconds
  by default) but lowers replicas only once every recommendation in the
  window agrees, so the candidates have been scored and aggregated for
  many cycles before the Scale patch lands. Making the HPA wait for the
  order before it patches Scale would break the goal that writers and
  readers never wait for each other, and is not needed.
- **Actors target the direct owner.** A preference names the object in
  the pods' owner reference with `controller: true`. For a Deployment
  that is the ReplicaSet, so during a rollout an actor writes two
  preferences and deletes the old one when it has no pods left. A
  controller that owns pods through intermediate objects consumes at
  the direct owner level, or runs as a hub.
- **No time-based expiry.** An entry counts until its actor removes it
  or the object it lives in is garbage collected. `lastUpdateTime` is
  for humans and tools.
- **Namespaced only.** No cross-namespace reference in v1alpha1.
- **Pods the consumer does not own are ignored.** An entry whose UID
  does not match one of the consumer's pods is skipped, so a stale
  entry never applies to a new pod that reuses the name.

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

- **Users read the order as a guarantee.** It is a hint, like
  pod-deletion-cost. Unscheduled, not running and not ready pods still
  go first, and there may be no order at all. Documentation and the API
  field comments say "advisory".

- **Write amplification by a chatty actor.** Every slice is capped at
  200 entries, every input at 10 slices, and every output at 2000
  entries. The HPA rounds scores to coarse buckets, scores only the pods
  it would remove plus a margin, and writes only when that changes.
  kube-controller-manager API clients default to 50 QPS with a burst of
  100, applied separately to each controller's client, so the HPA and
  the aggregator cannot flood the API server. For out-of-tree actors,
  [API Priority and Fairness](https://kubernetes.io/docs/concepts/cluster-administration/flow-control/)
  lets an operator isolate a misbehaving writer in its own FlowSchema.

- **Large output rewritten whole.** A `PodDeletionOrder` at the cap is
  about 240 KB. It has one watcher, the owner's controller, so the cost
  is bounded. If that changes the output can be served by an aggregated
  API server without changing the consumer contract.

- **Two outputs for one owner.** The aggregator checks its index for an
  existing output, claimed or not, before creating one. If a consumer
  still finds more than one, it uses the oldest and records an event.

- **Create and delete loop with garbage collection.** When an owner is
  deleted, GC removes its output while the inputs may still exist. The
  aggregator does not recreate it and marks the inputs `Accepted=False`
  with reason `TargetGone`.

- **Unprivileged writers steering deletions.** Anyone who can create a
  `PodDeletionPreference` in a namespace can influence which pods of
  that namespace are deleted first. That is the same power anyone with
  patch on pods has today through pod-deletion-cost, and it changes
  neither how many pods are deleted nor when. Only the aggregator can
  write `PodDeletionOrder`. The bootstrap RBAC changes will be reviewed
  with SIG Auth.

- **Stale input from a dead actor.** An input stays until its owner
  reference is collected. Inputs owned by the actor's object (the HPA)
  go with it; inputs owned by the target (Karpenter) go with the target.
  Pod UIDs are checked, so a stale entry only affects a pod that still
  exists.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Overview

Three kinds, one controller, one consumer change and one writer change:

1. Actors write `PodDeletionPreference` (identity, written once) and
   `PodDeletionPreferenceSlice` (scores, written often).
2. The aggregator, a controller in kube-controller-manager, groups
   preferences by target UID, merges slices, sums and sorts, and writes
   one `PodDeletionOrder` per target.
3. The ReplicaSet controller reads the `PodDeletionOrder` for its own
   UID and uses it as one step in its deletion sort.
4. The HPA writes preferences and slices for the pods behind its scale
   target.

```
 HPA ───────┐  write                                  writes
            ├─► PodDeletionPreference + Slices ─► Aggregator ─► PodDeletionOrder
 Karpenter ─┘  (one per actor per owner)                       (one per owner)
                                                                      │ reads
                                                                      ▼
                                                            ReplicaSet controller
```

### API

All kinds are namespaced and live in the `lifecycle.k8s.io` API group,
version `v1alpha1`.

#### Shared types

```go
// PodDeletionScore is one actor's opinion about one pod, or one
// aggregated entry in a PodDeletionOrder.
type PodDeletionScore struct {
	// PodName is the name of the pod within the same namespace. List key.
	// +k8s:format=k8s-long-name
	PodName string `json:"podName"`

	// UID of the pod, as in .metadata.uid. An entry whose UID does not match
	// the live pod is ignored, so a stale entry never applies to a new pod
	// that reuses the name.
	// +k8s:format=k8s-uuid
	UID types.UID `json:"uid"`

	// Score: higher = delete first. 1 is the weakest opinion.
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=1000
	Score int32 `json:"score"`
}

// TargetReference names the object a preference or order is about. The
// target is in the same namespace as the referencing object, or is
// cluster-scoped. There is no namespace field, so a cross-namespace
// reference cannot be expressed.
type TargetReference struct {
	APIVersion string    `json:"apiVersion"`
	Kind       string    `json:"kind"`
	Name       string    `json:"name"`
	UID        types.UID `json:"uid"`
}
```

#### PodDeletionPreference (input)

One object per actor per pods' controller owner. The spec is the actor's
identity and is immutable.

```go
type PodDeletionPreference struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodDeletionPreferenceSpec   `json:"spec"`
	Status PodDeletionPreferenceStatus `json:"status,omitempty"`
}

type PodDeletionPreferenceSpec struct {
	// Actor identifies the writer. Domain-qualified, e.g.
	// "horizontalpodautoscaler.autoscaling.k8s.io" or "karpenter.sh".
	// +k8s:immutable
	Actor string `json:"actor"`

	// Target says what this preference is about. Exactly one field is set,
	// and each field has its own contract. v1alpha1 has one field,
	// controllerRef. selector, nodeName and podGroupName are reserved for
	// later versions.
	// +k8s:immutable
	Target PodDeletionPreferenceTarget `json:"target"`
}

// +kubebuilder:validation:XValidation:rule="[has(self.controllerRef)].filter(x, x).size() == 1",message="exactly one target field must be set"
type PodDeletionPreferenceTarget struct {
	// ControllerRef is the pods' controller owner: apiVersion, kind, name
	// and UID copied from the pods' owner reference with controller: true.
	// Used for scale-down. The aggregator groups inputs by its UID and
	// writes one PodDeletionOrder per owner. Nobody reads the owner itself.
	// A listed pod that is not owned by it is ignored by the consumer.
	// +optional
	ControllerRef *TargetReference `json:"controllerRef,omitempty"`
}

type PodDeletionPreferenceStatus struct {
	// Conditions are written by the aggregator. Accepted: True once the
	// aggregator used this input; False with reason TargetGone or
	// TargetConflict. Nothing else lives here in v1alpha1.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

#### PodDeletionPreferenceSlice (input scores)

Written only by the actor that owns the parent preference. 
A slice belongs to its preference through an owner reference with
`controller: true`, which carries the parent's name and UID and drives
garbage collection, and it carries the parent's name in the label
`lifecycle.k8s.io/preference-name` for label selectors and kubectl, as
EndpointSlice does with `kubernetes.io/service-name`. 
Readers index slices by their controller owner UID.
A slice whose controller owner UID does not match a live parent is ignored.

```go
type PodDeletionPreferenceSlice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec PodDeletionPreferenceSliceSpec `json:"spec"`
}

type PodDeletionPreferenceSliceSpec struct {
	// LastUpdateTime is informational; there is no time-based expiry.
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// Scores lists the pods this actor has an opinion about. A pod should
	// appear in one slice; readers merge slices by pod UID and keep the
	// higher score if it appears in two.
	// +optional
	// +listType=map
	// +listMapKey=podName
	// +kubebuilder:validation:MaxItems=200
	Scores []PodDeletionScore `json:"scores,omitempty"`
}
```

#### PodDeletionOrder (output)

One object per target. Created and written only by the aggregator that
owns it. The name is the writer's choice and is not part of the
contract; consumers find the object through `spec.targetRef` (indexed
on UID) or through the target labels.

```go
type PodDeletionOrder struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PodDeletionOrderSpec   `json:"spec"`
	Status PodDeletionOrderStatus `json:"status,omitempty"`
}

type PodDeletionOrderSpec struct {
	// TargetRef names the object whose controller consumes this order,
	// including its UID. Copied from the inputs' controllerRef. Consumers
	// look the order up by this field and check kind and UID against
	// themselves. The writer also copies kind, name and UID into the labels
	// lifecycle.k8s.io/target-kind, lifecycle.k8s.io/target-name and
	// lifecycle.k8s.io/target-uid so clients without an index can use a
	// label selector.
	// +k8s:immutable
	TargetRef TargetReference `json:"targetRef"`

	// AggregatorName selects the controller that owns this object and its
	// status. Empty means the in-tree aggregator. Any other value tells the
	// in-tree aggregator to leave this object alone.
	// +optional
	AggregatorName string `json:"aggregatorName,omitempty"`
}

type PodDeletionOrderStatus struct {
	// Inputs is the number of inputs that produced this order.
	// +optional
	Inputs int32 `json:"inputs,omitempty"`

	// LastUpdateTime is when the aggregator last recomputed this order.
	// Informational.
	// +optional
	LastUpdateTime *metav1.Time `json:"lastUpdateTime,omitempty"`

	// DeletionOrder is the aggregated answer: one entry per pod named by any
	// input, score = sum of that pod's scores, sorted by score descending,
	// then by podName.
	// +optional
	// +listType=atomic
	// +kubebuilder:validation:MaxItems=2000
	DeletionOrder []PodDeletionScore `json:"deletionOrder,omitempty"`

	// Conditions: Aggregated; Truncated when the 2000 cap was hit and the
	// lowest scores were dropped.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

#### Example

A Deployment `my-app` in namespace `prod` with one ReplicaSet,
`my-app-7d4f9c8b6`. The HPA scores two idle pods. Karpenter scores one
pod on a node it is consolidating. The aggregator writes one order.

```yaml
# Input owned by the HPA. Spec is written once.
apiVersion: lifecycle.k8s.io/v1alpha1
kind: PodDeletionPreference
metadata:
  name: hpa-my-app-7d4f9c8b6          # HPA name + owner name
  namespace: prod
  uid: 6c1d2e3f-4a5b-4c6d-9e7f-8a9b0c1d2e3f
  labels:
    autoscaling.k8s.io/hpa: my-app
  ownerReferences:
  - {apiVersion: autoscaling/v2, kind: HorizontalPodAutoscaler, name: my-app, uid: 9a0b1c2d-3e4f-4a5b-8c6d-0e1f2a3b4c5d}
spec:
  actor: horizontalpodautoscaler.autoscaling.k8s.io
  target:
    controllerRef: {apiVersion: apps/v1, kind: ReplicaSet, name: my-app-7d4f9c8b6, uid: 2b7e1510-9c2a-4d8b-a4f1-6e2c9b1d0f3a}
status:
  conditions:
  - {type: Accepted, status: "True", reason: Aggregated}
---
# The HPA's scores. One slice is enough here.
apiVersion: lifecycle.k8s.io/v1alpha1
kind: PodDeletionPreferenceSlice
metadata:
  name: hpa-my-app-7d4f9c8b6-0        # writer's convention
  namespace: prod
  labels:
    lifecycle.k8s.io/preference-name: hpa-my-app-7d4f9c8b6
  ownerReferences:
  - {apiVersion: lifecycle.k8s.io/v1alpha1, kind: PodDeletionPreference, name: hpa-my-app-7d4f9c8b6, uid: 6c1d2e3f-4a5b-4c6d-9e7f-8a9b0c1d2e3f, controller: true}
spec:
  lastUpdateTime: "2026-10-26T14:30:00Z"
  scores:
  - {podName: my-app-7d4f9c8b6-abc12, uid: 8f7e6d5c-4b3a-4291-8a7b-11aa22bb33c1, score: 100}
  - {podName: my-app-7d4f9c8b6-def34, uid: 2b9c8d7e-6f5a-4b4c-9d3e-44dd55ee66f2, score: 80}
---
# Input owned by Karpenter.
apiVersion: lifecycle.k8s.io/v1alpha1
kind: PodDeletionPreference
metadata:
  name: karpenter-my-app-7d4f9c8b6
  namespace: prod
  uid: 7d2e3f4a-5b6c-4d7e-8f9a-0b1c2d3e4f5a
  ownerReferences:
  - {apiVersion: apps/v1, kind: ReplicaSet, name: my-app-7d4f9c8b6, uid: 2b7e1510-9c2a-4d8b-a4f1-6e2c9b1d0f3a}
spec:
  actor: karpenter.sh
  target:
    controllerRef: {apiVersion: apps/v1, kind: ReplicaSet, name: my-app-7d4f9c8b6, uid: 2b7e1510-9c2a-4d8b-a4f1-6e2c9b1d0f3a}
---
apiVersion: lifecycle.k8s.io/v1alpha1
kind: PodDeletionPreferenceSlice
metadata:
  name: karpenter-my-app-7d4f9c8b6-0
  namespace: prod
  labels:
    lifecycle.k8s.io/preference-name: karpenter-my-app-7d4f9c8b6
  ownerReferences:
  - {apiVersion: lifecycle.k8s.io/v1alpha1, kind: PodDeletionPreference, name: karpenter-my-app-7d4f9c8b6, uid: 7d2e3f4a-5b6c-4d7e-8f9a-0b1c2d3e4f5a, controller: true}
spec:
  lastUpdateTime: "2026-10-26T14:29:40Z"
  scores:
  - {podName: my-app-7d4f9c8b6-ghi56, uid: 5e4f3a2b-1c0d-4e9f-8a7b-77aa88bb99c3, score: 500}
---
# Output, created and written only by the aggregator. Kind, name and UID
# are copied from the inputs' controllerRef into spec.targetRef, the labels
# and the owner reference.
apiVersion: lifecycle.k8s.io/v1alpha1
kind: PodDeletionOrder
metadata:
  name: replicaset-my-app-7d4f9c8b6-x7k2q
  namespace: prod
  labels:
    lifecycle.k8s.io/target-kind: ReplicaSet
    lifecycle.k8s.io/target-name: my-app-7d4f9c8b6
    lifecycle.k8s.io/target-uid: 2b7e1510-9c2a-4d8b-a4f1-6e2c9b1d0f3a
  ownerReferences:
  - {apiVersion: apps/v1, kind: ReplicaSet, name: my-app-7d4f9c8b6, uid: 2b7e1510-9c2a-4d8b-a4f1-6e2c9b1d0f3a}
spec:
  targetRef: {apiVersion: apps/v1, kind: ReplicaSet, name: my-app-7d4f9c8b6, uid: 2b7e1510-9c2a-4d8b-a4f1-6e2c9b1d0f3a}
status:
  inputs: 2
  lastUpdateTime: "2026-10-26T14:30:01Z"
  deletionOrder:
  - {podName: my-app-7d4f9c8b6-ghi56, uid: 5e4f3a2b-1c0d-4e9f-8a7b-77aa88bb99c3, score: 500}
  - {podName: my-app-7d4f9c8b6-abc12, uid: 8f7e6d5c-4b3a-4291-8a7b-11aa22bb33c1, score: 100}
  - {podName: my-app-7d4f9c8b6-def34, uid: 2b9c8d7e-6f5a-4b4c-9d3e-44dd55ee66f2, score: 80}
  conditions:
  - {type: Aggregated, status: "True"}
```

#### Validation

- `PodDeletionPreference`: `spec.actor` is required and must be a
  domain-qualified name. `spec.target` must have exactly one field set.
  `spec` is immutable after creation.
- `PodDeletionPreferenceSlice`: at most 200 entries in `spec.scores`,
  keyed by `podName`. Each entry has a valid pod name, a UID and a score
  in `[1, 1000]`. The number of slices per preference is not validated
  by the API server; the aggregator reads at most 10 slices per input
  and ignores the rest.
- `PodDeletionOrder`: `spec.targetRef` is required and immutable. At
  most 2000 entries in `status.deletionOrder`.

### Scores and aggregation semantics

- A score is a pod reference (name and UID) and a number from 1 to
  1000. Higher means "delete this pod first". 1 is the weakest
  opinion.
- A pod that is not listed means the actor has no opinion about it.
  Actors list only the pods they want deleted first, usually their top
  candidates.
- Entries carry both the pod name and the pod UID, as `EvictionRequest`
  does. The name is the list key and is what humans read. The UID makes
  sure a stale entry never matches a new pod that reuses the name.
  An entry whose UID does not match the live pod is ignored.
- Within one actor, if a pod appears in two slices the higher score
  counts.
- Across actors, scores for the same pod UID are added. The result is
  sorted by score descending, then by pod name ascending. All actors
  count equally.
- If more than 2000 pods have a score, the aggregator keeps the 2000
  highest and sets the `Truncated` condition on the output.

The aggregated order is a deterministic function of the inputs. Every
aggregator version produces the same answer for the same inputs, and
every consumer produces the same deletion order for the same answer and
pod set.

### Slices

Scores are never stored inline in the object that identifies an actor.
They live in slice objects owned by it, each holding at most 200 scores.
A small workload has one slice. A large one has up to 10, so 2000
scores per input. Every actor may score a different set of pods, so the
output can hold up to the union of all inputs, which is why the output
cap is also 2000.

Slices are independent. A reader merges all slices of one parent by pod
UID and, if a pod appears in two slices, keeps the higher score. A pod
missing for one informer tick while a writer rebalances is a pod with
no opinion for that tick, which is what advisory means. Writers should
keep a pod in the slice it was first placed in, so most updates are one
in-place write; that is good practice, not something readers depend on.
No parent object is written on a score update.

### Binding a preference to its workload

A preference has to say which pods it is about, and a consumer has to
find the preferences that are about its pods. `v1alpha1` binds through a
typed reference to the pods' controller owner: `spec.target.controllerRef`
on the input and `spec.targetRef` on the output carry the apiVersion,
kind, name and UID of the object in the pods' owner reference with
`controller: true`, of which a pod has at most one.

`spec.target` is a union with one field per use case so that other
bindings can be added later with their own written contracts: a label
selector (one input for a whole Deployment, with the aggregator doing
the per-owner split), a node name (kubelet, node autoscalers) and a pod
group name (gang workloads). See [Future Work](#future-work) and
[Alternatives](#other-ways-to-bind-a-preference-to-its-workload).

### The aggregator

A new controller in kube-controller-manager. It watches
`PodDeletionPreference` and `PodDeletionPreferenceSlice` and writes
`PodDeletionOrder`.

**Grouping by owner.** Inputs are indexed by `spec.target.controllerRef.uid`,
which is unique. Kind and name are taken from the inputs and copied
into the output's `spec.targetRef` and labels. If inputs for one UID
disagree on kind or name, that is a writer bug: the first wins and the
others get `Accepted=False` with reason `TargetConflict`. When an input
or a slice changes, the aggregator recomputes every input with the same
UID.

**Reading slices.** Slices are indexed by their controller owner UID.
For each input the aggregator merges its slices by pod UID, keeping the
higher score if a pod appears twice. It reads at most 10 slices per
input and ignores the rest.

**Writing.** It adds up the scores per pod UID and sorts them, highest
first, ties by pod name. It keeps the top 2000 and sets `Truncated=True`
if it dropped any. It writes one `PodDeletionOrder` per owner. Before
creating one, it checks its own index for an output with that owner
UID, claimed or not, so there is never a second one. It creates the
object with `generateName`, for example `replicaset-my-app-7d4f9c8b6-`,
so the name is readable and never collides; the API server shortens a
long prefix on its own. This naming is the aggregator's convention, not
an API rule. It copies kind, name and UID from the inputs into
`spec.targetRef`, into the target labels and into an owner reference to
the target. It writes the sorted list into `status.deletionOrder`, sets
`status.inputs`, `status.lastUpdateTime` and the `Aggregated` condition,
and sets `Accepted=True` on each input it used, once. When the last
input for an owner is gone, it deletes the `PodDeletionOrder`.

**Outputs it does not own.** The in-tree aggregator owns exactly the
outputs whose `spec.aggregatorName` is empty. Claimed outputs, and the
inputs and slices for their target, are ignored until the field is
cleared, after which the owner is recomputed.

**When an owner disappears.** If an unclaimed output disappears while
its inputs still exist, the owner was deleted and garbage collection
removed the output. The aggregator does not recreate it and marks the
inputs `Accepted=False` with reason `TargetGone`, until an input
changes. Without this rule the aggregator and garbage collection would
create and delete the output in a loop. A Deployment recreated with the
same name is not affected: its new ReplicaSet has a new UID, so the
stale inputs never reach it.

**Rate.** The aggregator recomputes one owner per input or slice
event, with the usual workqueue deduplication, so a burst of slice
writes for one owner collapses into one output write.

### The ReplicaSet controller as consumer

The ReplicaSet controller gains an informer on `PodDeletionOrder` with
an index on `spec.targetRef.uid`, the same way it already indexes pods
and ReplicaSets by controller UID.

When it has to scale down, it looks up the `PodDeletionOrder` for its
own UID. It checks that `spec.targetRef.kind` is `ReplicaSet`; if not,
it ignores the object. If it finds more than one, it uses the oldest and
records an event with reason `MultiplePodDeletionOrders`. It takes the
scores from `status.deletionOrder`, skips entries whose UID does not
match one of its pods, and uses the rest as one step in its sort order.
Pods with no score are neutral. If there is no `PodDeletionOrder`, the
sort is the one from today.

The feature does not change how many pods are deleted, or when. It only
changes which ones.

A custom controller that owns its own pods does exactly the same with
its own UID. A controller without an index lists by the
`lifecycle.k8s.io/target-uid` label instead.

### Coexistence with pod-deletion-cost

The ReplicaSet controller's current sort, from first deleted to last:

1. Unscheduled.
2. Pending, then unknown, then running.
3. Not ready.
4. Lower `controller.kubernetes.io/pod-deletion-cost`.
5. More co-located pods on the same node.
6. More recently ready.
7. Higher restart count.
8. More recently created.

The aggregated score becomes a new step, higher score first. There are
two places it can go:

- **Annotation outranks.** New step between 4 and 5. The preference
  only decides among pods with equal deletion cost.
- **API outranks.** New step between 3 and 4. The annotation only breaks
  ties among equal preference scores.

Pod-deletion-cost is not deprecated by this KEP. It stays the way for an
application to express a per-pod preference about itself without a
controller.

### The HPA as writer

The HPA controller becomes the first in-tree actor, with actor name
`horizontalpodautoscaler.autoscaling.k8s.io`, behind the
`HPAPodDeletionPreference` gate.

**Scoring, every sync.** The HPA already lists the pods behind its scale
target, using the selector from the Scale subresource and its pod
informer cache, and already has per-pod metrics for them. It turns
those metrics into a score for each candidate pod as described in
[Converting metrics to scores](#converting-metrics-to-scores). It then
groups the candidates by their controller owner reference. For a
Deployment that is the ReplicaSet. The set of owners that currently
have pods is the set of inputs the HPA should have.

**Owner tracking and cleanup.** For each owner it makes sure one
`PodDeletionPreference` exists, named after the HPA and the owner (for
example `hpa-my-app-7d4f9c8b6`), with `controllerRef` copied from the
pod's owner reference, a label with the HPA's name, and an owner
reference to the HPA. The input is written once. Then it lists its own
inputs by that label and deletes every one whose `controllerRef` is an
owner that has no pods any more.

**Writing scores.** The HPA does not score every pod, only the
candidates it would remove if it scaled down now. A write happens only
when the candidate set or a pod's bucket changed. That bounds the write
rate: at most one slice write per HPA per sync, and in practice far
fewer, since a bucketed ranking of the bottom few pods is stable. A pod
stays in the slice it was first placed in, so a changed score or a pod
added or dropped is one in-place write to one slice. New slices are
created when the existing ones are full; empty slices are deleted. The
candidate set is capped at 2000, the input limit.

**Order of operations.** The HPA writes its slices before it patches
Scale. The ReplicaSet controller reacts to the replica change within
milliseconds, while the new slices still have to pass through the
aggregator. So the ReplicaSet scales down on the order completed before
this sync, about one sync period old. The stabilization window closes
most of that gap. The HPA scores the pods on every sync, 15 seconds by
default, but with the default scale-down stabilization window of 300
seconds it lowers the replica count only after every recommendation in
the window has been at or below the new value. By the time Scale is
patched, the bottom candidates have been scored, aggregated and visible
to the ReplicaSet controller for many cycles, and the previous sync's
order already names them. Only a change in the last 15 seconds is
missed, and the metrics behind the scores are older than that anyway.
Users who shorten the window to 0 fall back to the one-sync lag, which
is still acceptable because preferences are advisory.

#### Converting metrics to scores

The HPA already fetches metrics for every pod to decide how many
replicas it wants. It uses the same metrics to score pods. It does not
fetch anything extra.

**Step 1: one ratio per pod, per metric.** For each metric in the HPA
spec, the HPA computes how close each pod is to the target. This is the
pod's ratio. A ratio below 1.0 means the pod is below the target. A
ratio above 1.0 means the pod is above the target.

| Source            | Target type           | Per-pod ratio                                                       |
|-------------------|-----------------------|---------------------------------------------------------------------|
| Resource          | Utilization           | (pod usage / pod requests) divided by target utilization            |
| Resource          | AverageValue          | pod usage / target average value                                    |
| ContainerResource | Utilization           | (container usage / container request) divided by target utilization |
| ContainerResource | AverageValue          | container usage / target average value                              |
| Pods              | AverageValue          | pod metric value / target average value                             |
| Object            | Value or AverageValue | none. One number for the whole object.                              |
| External          | Value or AverageValue | none. One number from outside the cluster.                          |

Object and External metrics give one number for the whole workload,
not one per pod. They cannot say which pod is busier. An HPA that has
only these metrics writes no preference.

**Step 2: one ratio per pod.** If the HPA has more than one metric, a
pod has more than one ratio. The HPA keeps the highest one. This is the
pod's busiest side. A pod that is idle on CPU but busy on requests per
second is a busy pod. It should not be deleted first. Taking the
highest ratio also keeps the result stable. If the HPA used only the
metric that is driving the replica count, the scores would jump each
time a different metric takes the lead. A pod with no ratio at all, for
example because its metrics are missing, gets no score.

**Step 3: pick the candidates.** The HPA sorts the pods by ratio, lowest
first. It then takes the pods from the top of the list. How many it
takes is the largest of these three numbers:

- current replicas minus the replicas the HPA wants after its scale-down
  policies
- 10% of current replicas
- 2

Only pods with a ratio below 1.0 can be candidates. A pod that is at or
above the target is never scored by the HPA.

**Step 4: turn the ratio into a score.** The ratio is split into ten
buckets of 0.1. Each bucket maps to a score. The HPA only uses scores
from 1 to 499. This leaves the upper half of the range, 500 to 1000, for
actors with a stronger reason, such as a node that is about to be
removed. One such score beats any HPA score on its own.

| Ratio            | Score   |
|------------------|---------|
| below 0.1        | 499     |
| 0.1 to 0.2       | 449     |
| each further 0.1 | 50 less |
| 0.9 to 1.0       | 49      |

A pod does not change bucket as soon as its ratio crosses a line. It
has to move 0.05 past the line first. This stops small noise around a
line from causing writes. The same rule applies to the candidate list:
a pod has to move half a step past the cut-off before it joins or
leaves.

**Step 5: write.** The HPA writes one entry per candidate: pod name,
pod UID and score. It writes each entry into the slice where the pod
already is. It writes only when the candidate list changed or a pod
moved to a different bucket.

### Out-of-tree writers

Using this API design, any controller can be an actor. The contract is:

- Pick a domain-qualified actor name and use it in every input.
- Group the pods you have an opinion about by their controller owner
  reference. Write one `PodDeletionPreference` per owner, with
  `controllerRef` copied from the pod. Write it once.
- Put the scores in `PodDeletionPreferenceSlice` objects owned by the
  preference with `controller: true`, labelled with its name, at most
  200 scores per slice and at most 10 slices.
- Write only when your opinion changes. Keep a pod in the slice it was
  first placed in.
- Delete the input when you have nothing more to say about the owner.
  Give the input an owner reference to your own object if you have one,
  otherwise to the target, so garbage collection cleans up after you.

### Hub mode

A controller that wants its own combining logic for some owners claims
their `PodDeletionOrder` by setting `spec.aggregatorName` to its own
name. It finds the output by target UID like a consumer does, and
creates one only when none exists, with any name it likes. From then on
the in-tree aggregator leaves that object alone; what the hub writes
there, and how it computes it, is the hub's business.

### Consistency model

The API is advisory and eventually consistent. Actors publish on their
own cadence. The aggregator and consumers act on the most recent state
in their informer caches. No write is synchronized with a scale event,
and no consumer waits for a fresher preference.

### Permissions

Each object has one writer for its data:

| Role | PodDeletionPreference | PodDeletionPreferenceSlice | PodDeletionOrder |
|---|---|---|---|
| Actor (HPA controller, out-of-tree) | create, delete, list, watch | create, update, delete, list, watch | none |
| Aggregator | list, watch, update `status` | list, watch | create, update `status`, delete, list, watch |
| Consumer (ReplicaSet controller) | none | none | list, watch |

Bootstrap RBAC: the `system:controller:horizontal-pod-autoscaler` role
gains the actor verbs. A new `system:controller:pod-deletion-order-controller`
role is added for the aggregator. The `system:controller:replicaset-controller`
role gains list and watch on `poddeletionorders`. Actors cannot write
outputs, so they cannot fake the answer. The aggregator cannot change
the inputs.

### Feature gates

Two gates, both alpha and off by default:

- `PodDeletionPreference` (kube-apiserver, kube-controller-manager).
  Serves the three kinds, runs the aggregator, and makes the ReplicaSet
  controller read `PodDeletionOrder`.
- `HPAPodDeletionPreference` (kube-controller-manager). Makes the HPA
  controller write preferences and slices. Depends on
  `PodDeletionPreference`; with the API gate off the HPA writes nothing.

The HPA writer has its own gate so the API and its first consumer can
be used with out-of-tree actors alone, and so the HPA behaviour can be
turned off without losing the API.

### Future Work

- **Rules.** A rule names pods by a property instead of listing them,
  as a union with exactly one match field. The first is "priority
  below N", for the kubelet during Graceful Node Shutdown.
- **Node-targeted preferences and the kubelet.** A `nodeName` target
  field. The kubelet would write one preference per node during
  Graceful Node Shutdown with a priority rule, so the DaemonSet
  controller can hold back templates the kubelet would reject
  ([KEP-6249](https://github.com/kubernetes/enhancements/issues/6249)). This needs a home for a node-scoped object in a
  namespaced kind (the `kube-node-lease` namespace by convention, or a
  cluster-scoped sibling kind) and node authorizer and NodeRestriction
  rules.
- **Common evaluation library.** A shared library that resolves a set
  of inputs (slices, rules) into a final per-pod score, so every
  consumer evaluates preferences the same way.
- **Per-actor weights.** Let users control how much each actor counts.
  Today all actors are weighted equally and summed.
- **Signed scores.** Let an actor say "prefer to keep this pod".
- **Workload- and batch-aware deletion planning.** Per-pod scores
  cannot represent decisions that depend on the number or combination
  of pods deleted. A future integration with the Workload API, or a
  separate deletion-plan API, could support topology-, scheduling- and
  batch-aware decisions while this API stays focused on per-pod
  preferences.
- **Other consumers.** StatefulSet, DaemonSet, Job and out-of-tree
  workload controllers, each with the same read-one-object contract.

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

None identified. The ReplicaSet controller's pod ranking and the HPA
controller's replica calculation already have unit tests that the new
behaviour extends.

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

New packages:

- `k8s.io/kubernetes/pkg/apis/lifecycle`: the API types, validation and
  defaulting.
- `k8s.io/kubernetes/pkg/registry/lifecycle`: storage strategies for the
  three kinds.
- `k8s.io/kubernetes/pkg/controller/poddeletionorder`: the aggregator.

Changed packages:

- `k8s.io/kubernetes/pkg/controller/replicaset`: `2026-11-08` - `<fill from testgrid>`
- `k8s.io/kubernetes/pkg/controller/podautoscaler`: `2026-11-08` - `<fill from testgrid>`

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
-->

Added under `test/integration/` for alpha:

- Write two preferences with slices for one ReplicaSet from two actors;
  the aggregator produces one order with the summed, sorted list and
  sets `Accepted` on both inputs.
- Scale a ReplicaSet down; the pods with the highest aggregated score
  are the ones deleted. With no order, the deleted pods are the ones the
  existing sort picks.
- Pod-deletion-cost and an order on the same ReplicaSet; the agreed
  sort position is honoured.
- Delete the ReplicaSet; the order is garbage collected and the inputs
  get `TargetGone` and are not recreated.
- Delete an HPA; its inputs and slices are garbage collected.
- HPA writer: create a Deployment and an HPA, drive metrics so that
  some pods are idle, scale down, and check the idle pods were removed.
  Roll the Deployment and check there is one input per ReplicaSet and
  the old one is removed when it has no pods.
- Claim an order with `spec.aggregatorName`; the in-tree aggregator
  stops writing it; clear the field and it resumes.
- Feature gate off in kube-controller-manager with objects present; the
  ReplicaSet controller scales down as today.

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
-->

Alpha: e2e test behind the `PodDeletionPreference` feature label
that creates a Deployment, writes a preference and slice for its
ReplicaSet as an out-of-tree actor would, scales the Deployment down,
and checks the scored pods were the ones deleted.
- [test name](https://github.com/kubernetes/kubernetes/blob/master/test/e2e/...): [SIG Autoscaling](https://testgrid.k8s.io/sig-autoscaling?include-filter-by-regex=PodDeletionPreference), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=PodDeletionPreference)

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

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

**Note:** Beta criteria must include all functional, security, monitoring, and testing requirements along with resolving all issues and gaps identified

#### GA

- N examples of real-world usage
- N installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

**Note:** GA criteria must not include any functional, security, monitoring, or testing requirements.  Those must be beta requirements.

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

<!--
- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

#### Alpha

- Feature gates disabled by default.
- `lifecycle.k8s.io/v1alpha1` with the three kinds.
- Aggregator, ReplicaSet consumer and HPA writer implemented.
- Unit, enablement/disablement, e2e, and integration tests implemented
  and passing.

#### Beta

- Feature gates enabled by default.
- `lifecycle.k8s.io/v1beta1`.
- API group and sort position relative to pod-deletion-cost settled.
- At least one out-of-tree actor writes the API and its feedback is
  addressed.
- Metrics listed in [Monitoring Requirements](#monitoring-requirements)
  added.
- E2e and integration tests are in Testgrid and linked in the KEP.
- Scalability test with the caps filled (2000 scores per input and per
  output).
- If the HPA writer and the API need different graduation schedules,
  the HPA writer moves to its own KEP before either graduates.

#### GA

- Every bug report is fixed.
- Confirm the stability of e2e and integration tests.
- Conformance tests for the API endpoints.
- `PodDeletionPreference` and `HPAPodDeletionPreference` feature gates
  are locked to enabled.

#### Deprecation

Not applicable. `controller.kubernetes.io/pod-deletion-cost` is not
deprecated.

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

**Upgrade.** Enabling `PodDeletionPreference` serves the API, starts the
aggregator and makes the ReplicaSet controller read orders. Until an
actor writes something, nothing changes: there are no orders and the
sort is today's. Enabling `HPAPodDeletionPreference` makes the HPA start
writing on its next sync.

**Downgrade.** Disabling the gates stops the aggregator, makes the
ReplicaSet controller ignore orders, and makes the HPA stop writing.
Stored objects remain in etcd but are not served while the API gate is
off. Scale-down returns to today's behaviour. No cleanup is needed.

**Re-upgrade.** The stored objects are served again. The aggregator
recomputes every target from its inputs. Entries whose pod UID no
longer exists are ignored, so stale data is harmless. The HPA resumes
and deletes inputs for owners that no longer have pods.

Nothing about the feature is required to keep previous behaviour on
upgrade. Nothing has to be configured to use it beyond the gates, for
the HPA path. Out-of-tree actors need the RBAC described under
[Permissions](#permissions).

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

We need to consider the version skew between kube-controller-manager and
the apiserver.

If the feature is enabled on the apiserver, but not in the
kube-controller-manager, then `PodDeletionPreference` and
`PodDeletionPreferenceSlice` objects can be created by out-of-tree
actors, but no `PodDeletionOrder` is produced and scale-down is
unchanged.

If the feature is not enabled on the apiserver, and it is enabled in the
kube-controller-manager, then

- The HPA cannot write preferences and skips that step.
- The aggregator has nothing to watch and does not run.
- The ReplicaSet controller finds no `PodDeletionOrder` and scales down
  as today. It does not block on the missing informer.

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

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `PodDeletionPreference`
  - Components depending on the feature gate: kube-apiserver,
    kube-controller-manager
  - Feature gate name: `HPAPodDeletionPreference`
  - Components depending on the feature gate: kube-controller-manager

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

Enabling `PodDeletionPreference` alone changes nothing until an actor
writes a preference. There are no orders, and the ReplicaSet controller
sorts as today.

Enabling `HPAPodDeletionPreference` as well changes which pods the
ReplicaSet controller deletes when an HPA scales a workload down: the
pods the HPA scored as idle are deleted first instead of the pods the
existing sort would pick. It does not change how many pods are deleted
or when.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

Yes. Disabling the gates and restarting the components stops the
aggregator, makes the ReplicaSet controller ignore orders, and stops
the HPA from writing. Scale-down returns to today's behaviour
immediately. Stored objects stay in etcd and are not served; they do
not affect running workloads.

###### What happens if we reenable the feature if it was previously rolled back?

The stored objects are served again. The aggregator recomputes every
target from its inputs and slices. Entries for pods that no longer
exist, or whose UID changed, are ignored by the consumer. The HPA
resumes writing and deletes inputs for owners that no longer have pods.

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

Yes:

- Registry strategy unit tests for the three kinds with the gate on and
  off.
- ReplicaSet controller unit tests that an order present in the cache is
  ignored with the gate off and used with the gate on.
- HPA controller unit tests that nothing is written with
  `HPAPodDeletionPreference` off, and with `PodDeletionPreference` off.

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

A rollout cannot affect a running pod. The feature only chooses which
pods are deleted when a scale-down already happens.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

New metrics added by this KEP, exposed by kube-controller-manager:

- `poddeletionpreference_inputs_rejected_total` rising: inputs rejected
  with `TargetConflict` or `TargetGone`, a writer bug or a GC loop.
- `poddeletionpreference_aggregator_sync_duration_seconds` p99 growing:
  the aggregator is falling behind.
- `horizontal_pod_autoscaler_controller_deletion_preference_writes_total`
  climbing faster than one per HPA per sync: the HPA writes too often.

Existing metrics:

- `apiserver_request_total` for the `lifecycle.k8s.io` group rising
  beyond the expected rate.
- `workqueue_depth` for the aggregator growing.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

Not yet. Will be tested manually before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

No.

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

Existing metric, exposed by kube-apiserver:

- `apiserver_storage_objects{resource="poddeletionorders.lifecycle.k8s.io"}`
  greater than zero means at least one workload has an order.

New metrics added by this KEP, exposed by kube-controller-manager:

- `replicaset_controller_scale_down_deletion_order_total`: scale-downs
  where the ReplicaSet controller found and used an order.
- `horizontal_pod_autoscaler_controller_deletion_preference_writes_total`:
  slice writes by the HPA controller.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [x] Events
  - Event Reason: `MultiplePodDeletionOrders` on the ReplicaSet when
    more than one `PodDeletionOrder` targets it.
- [x] API .status
  - `PodDeletionPreference.status`
    - Condition name: `Accepted`. `True` once the aggregator used the
      input; `False` with reason `TargetGone` or `TargetConflict`.
  - `PodDeletionOrder.status`
    - Condition name: `Aggregated`, and `Truncated` when the 2000 cap
      dropped entries.
    - Other field: `deletionOrder`, the pods in the order they will be
      preferred for deletion with their summed score; `inputs`, how
      many actors contributed; `lastUpdateTime`.

A user can list `poddeletionorders` with the label
`lifecycle.k8s.io/target-name=<replicaset>` and compare
`status.deletionOrder` with the pods deleted on the last scale-down.

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

Scale-down latency of the ReplicaSet controller does not change
measurably. Reading the order is one cached lookup and one more sort
key, so the existing ReplicaSet controller sync duration stays within
its current distribution.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [x] Metrics
  - Metric name: `poddeletionpreference_aggregator_sync_duration_seconds`
    - Components exposing the metric: kube-controller-manager, the
      aggregator (pod deletion order controller)
  - Metric name: `poddeletionpreference_outputs_truncated_total`
    - Components exposing the metric: kube-controller-manager, the
      aggregator
  - Metric name: `poddeletionpreference_inputs_rejected_total`
    - [Optional] Aggregation method: by `reason`
    - Components exposing the metric: kube-controller-manager, the
      aggregator
  - Metric name: `replicaset_controller_scale_down_deletion_order_total`
    - [Optional] Aggregation method: by whether an order was found
    - Components exposing the metric: kube-controller-manager, the
      ReplicaSet controller
  - Metric name: `horizontal_pod_autoscaler_controller_deletion_preference_writes_total`
    - [Optional] Aggregation method: by result
    - Components exposing the metric: kube-controller-manager, the HPA
      controller

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

A metric for the age of the order used at scale-down (time between
`status.lastUpdateTime` and the deletion) would show how stale the
opinion is in practice. it may be added at beta.

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

No. The aggregator and the consumers run in kube-controller-manager.
The HPA writer depends on the metrics the HPA already uses; if metrics
are unavailable the HPA does not scale and does not write.

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

Yes.

- HPA controller: CREATE and DELETE `PodDeletionPreference`, once per
  ReplicaSet that gains or loses pods (rollouts); UPDATE
  `PodDeletionPreferenceSlice`, at most once per HPA per sync period
  (15 s by default) and only when the rounded ranking changed.
- Aggregator: LIST and WATCH `PodDeletionPreference` and
  `PodDeletionPreferenceSlice`; CREATE, UPDATE status and DELETE
  `PodDeletionOrder`, at most one status update per slice write after
  workqueue deduplication; one status update per input the first time
  it is accepted or rejected.
- ReplicaSet controller: LIST and WATCH `PodDeletionOrder`. No new
  writes.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

Yes, three namespaced kinds in `lifecycle.k8s.io/v1alpha1`:

- `PodDeletionPreference`: one per actor per ReplicaSet with pods.
- `PodDeletionPreferenceSlice`: 1 to 10 per preference, at most 200.
- `PodDeletionOrder`: one per ReplicaSet with at least one input, at
  most 2000 entries.

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

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

No measurable change is expected. The ReplicaSet controller does one
cached index lookup and uses the result as one more sort key. The HPA
does one grouping of pods it already has and at most one write per
sync.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

kube-controller-manager holds three more informer caches: inputs and
slices for the aggregator, orders for the ReplicaSet controller. At the
cluster sizes above that is in the low tens of MB. The aggregator's
recompute is a merge and sort over at most a few thousand entries per
target. kube-apiserver and etcd store the new objects, bounded by the
counts above.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

No. Nothing runs on nodes in v1alpha1.

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

The same as the rest of kube-controller-manager: no scale-down happens,
no preferences are written, no orders are computed. When the API server
returns, the controllers resume from their caches and the aggregator
recomputes from the stored inputs. Nothing is lost that matters: the
next sync rewrites the HPA's opinion anyway.

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

- **Aggregator falling behind.**
  - Detection: `poddeletionpreference_aggregator_sync_duration_seconds`
    p99 or the aggregator's `workqueue_depth` growing; stale
    `status.lastUpdateTime` on orders.
  - Mitigations: scale-down keeps working with the last order or
    today's sort. Disable `PodDeletionPreference` to stop the
    controller.
  - Diagnostics: aggregator logs at verbosity 2 per output write and
    per rejected input.
  - Testing: unit tests for every rejection path.

- **Input rejected (`TargetConflict`, `TargetGone`).**
  - Detection: `poddeletionpreference_inputs_rejected_total` by reason;
    `Accepted=False` on the input.
  - Mitigations: `TargetConflict` is a writer bug, delete the wrong
    input. `TargetGone` means the owner was deleted; the input goes with
    its owner reference. No effect on scale-down.
  - Testing: unit and integration tests.

- **Scale-down ignores the preference.**
  - Detection: `replicaset_controller_scale_down_deletion_order_total`
    reports no order found, or the deleted pods were not the scored
    ones.
  - Mitigations: check the gate in kube-controller-manager; check the
    order targets the ReplicaSet's UID, not its name; check the pod UIDs
    match live pods; remember that unscheduled, not running, not ready
    pods and pod-deletion-cost come first in the sort.
  - Testing: unit and integration tests for the sort position.

###### What steps should be taken if SLOs are not being met to determine the problem?

- Check `poddeletionpreference_aggregator_sync_duration_seconds` and
  the aggregator's `workqueue_depth`. If high, find the chatty actor
  through `apiserver_request_total` by user agent.
- Check `poddeletionpreference_inputs_rejected_total` by reason.
- If the aggregator is the problem, disable `PodDeletionPreference`. If
  one actor is the problem, isolate it with API Priority and Fairness
  or stop it; the other actors keep working.

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

- 2026-11-08: KEP created, based on the design document
  ["PodDeletionPreference API Design"](https://docs.google.com/document/d/1i7ShORMRcgeHbFrgh-qSTmqqhJTj2OOi02n6pR920-A/edit?usp=sharing).

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

- Three kinds and a new controller for a feature that is, from the
  consumer's point of view, one more sort key. The alternatives with
  one kind either make the consumer do the aggregation (which SIG Apps
  asked to avoid), put several writers on one object, or have no home
  for a hub and for node-targeted preferences.
- Every actor update is two writes (slice, then output) and one hop
  before the consumer sees it. For the HPA a scale-down is ordered by
  the previous sync's opinion, which the default 300 second scale-down
  stabilization window makes negligible.
- Every actor must target the pods' direct owner and keep one input per
  owner, two during a rollout. It is a loop in every actor rather than
  once in the aggregator. A selector target would move that work to
  the aggregator at the price of a pod informer; it is reserved for a
  later version.
- Small workloads pay one input, one slice and one output for what
  could be a few bytes.
- The output is rewritten whole on every recompute, up to about 240 KB
  at the cap. Acceptable while it has one watcher.
- A controller that owns pods through intermediate objects is targeted
  at the direct owner and must consume at that level or run as a hub.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

The design space was explored along several axes: how preferences are
delivered (standing objects, computed on read, on demand), object
topology (per-actor, shared, per-actor inputs plus an output), where
aggregation happens, how a preference finds its workload, data shape,
relation to pod-deletion-cost, and scope. The approaches below are the
combinations that were worked out in detail.

### Per-actor objects, consumer aggregates

Each actor owns a `PodDeletionPreference` with a label selector and
slices with scores. The ReplicaSet controller lists the preferences in
its namespace, keeps the ones whose selector matches its pods, follows
their slices and sums.

Pros: single-writer objects with clean ownership and garbage
collection; small watch events; a selector spans both ReplicaSets of a
rollout so the actor does nothing special.

Cons: the consumer does everything (list, selector match, join, sum),
which SIG Apps asked to avoid; selector discovery is O(preferences ×
pods) per sync unless indexed; overlapping selectors; many objects for
one answer; no way to mark a preference as "input for a hub, not for
direct consumption", so a hub would be double counted.

### Shared object per workload, consumer aggregates through a library

One `PodDeletionPreference` per ReplicaSet, with the same name. Every
actor writes its own entry into `spec.preferences` with Server-Side
Apply. No status, no controller. The ReplicaSet controller reads the
object with its own name and sums the entries through a shared Go
library.

Pros: one kind, one object per workload, found by name; no extra
controller, no extra hop; the math is a small pure function.

Cons: the ReplicaSet controller does the math, which SIG Apps did not
want; consumers in other languages must reimplement the library;
every write by any actor sends the whole object to every watcher;
several actors apply to one list, so the first create, the owner
reference and cleanup of a dead actor's entry conflict; a deleted
actor's entry stays until the ReplicaSet is replaced; no hub mode; no
home for a node-targeted preference, since NodeRestriction cannot
scope the kubelet to one Server-Side Apply entry.

### Shared object per workload, controller writes status

As above, but an in-tree aggregator computes `status.deletionOrder` from
`spec.preferences`, and `spec.aggregatorName` lets a hub take over the
status. The ReplicaSet controller reads only the status.

Pros: one kind, one object per workload; the consumer reads only
status; the math lives in one place; permissions split between spec
(actors) and status (aggregator); hub mode for free; no slices, no
selectors, no discovery.

Cons: every actor write produces two whole-object events; one extra
hop; several controllers write the same spec and RBAC cannot stop one
actor from editing another's entry; the object size is the sum of all
actors; a deleted actor's entry lingers; no home for a node-targeted
preference. The multi-writer spec was judged the main API review risk.

This was the closest alternative. The selected design keeps its
consumer contract (read one object, written by one aggregator, hub
through `aggregatorName`) and fixes the multi-writer spec, the unbounded
object size and the lingering entries by giving each actor its own
input and slices.

### Field on the Scale subresource

The original proposal. Together with the replica count, the HPA writes
a list of pod names into the Scale subresource in one PATCH. Each
scalable kind stores the list in its own spec and its controller deletes
the listed pods first.

Rejected: Scale is not a stored object, so every scalable kind and
every CRD must add a field and map it, and a kind that does not silently
drops it; and it has one writer, so the HPA, Karpenter and others
overwrite each other.

### Pod annotation only

The status quo. Actors set `controller.kubernetes.io/pod-deletion-cost`
on each pod and the ReplicaSet controller sorts by it.

Pros: exists today, beta since 1.22, no new API.

Cons: N pod patches per update; one number per pod with no actor
identity, so two actors overwrite each other; fast-changing metrics
would cause constant pod writes, so it cannot carry the HPA's signal.

### Aggregated API server

Serve `PodDeletionOrder` from an extension API server, computed on read
and never stored, the way metrics-server serves PodMetrics. Actors
write inputs and slices to kube-apiserver as in the selected design; the
extension server watches them and sums on each LIST.

The saving is one write per actor update, since the output is not
stored. The cost is that a resource computed on read has no Watch. The
aggregation layer only forwards requests, and a computed answer has
nothing to watch, which is why metrics-server has none. Every consumer
and client would have to implement its own polling protocol against the
extension server, including the ReplicaSet controller, which would lose
its informer and send a LIST to a remote server on every scale-down. We
want to keep the ReplicaSet controller simple: one cached lookup. If
the extension server is down, every preference is silently lost. Giving
consumers a cache back means implementing Watch in the server, which
rebuilds the aggregator out of tree and loses the saved write.

On top of that: a new component to build, secure and operate outside
the kubernetes repo; an unavailable APIService breaks discovery and can
stall garbage collection and namespace deletion; managed clusters would
not ship it on day one. A pull model, where the server fetches scores
from actors over HTTP, was also rejected because every actor would need
an authenticated endpoint and the HPA runs in kube-controller-manager,
which serves nothing.

The selected design keeps the door open: the output contract is the
same whether it is stored or computed on read.

### On-demand request and response

Nothing stands. When the ReplicaSet controller has to remove n pods, it
creates a request naming itself, the count, the candidate pods and a
deadline. Registered responders score the candidates against the
request; an aggregator sums the answers and marks it complete; the
ReplicaSet controller reads the order on its next sync and deletes the
request. The shape is EvictionRequest's, asking "which n pods should
go" instead of "may this pod go".

Rejected for v1alpha1 because it breaks the design goal that writers
and readers never wait for each other, in the one controller where that
matters most, and because responders have to be known in advance or
every request waits for the full deadline.

### Other ways to bind a preference to its workload

- **Same name as the consumer.** One cached read by own name and no
  search. But ReplicaSet names change on every rollout so writers must
  know the current ReplicaSets, two kinds with one name collide, a
  target recreated with the same name has a new UID, and it puts a
  naming rule in the API. Dropped for the output in favour of the typed
  reference.
- **Label selector.** Flexible, used by PodDisruptionBudget and
  Service, and the HPA already has one. But someone has to map matched
  pods to their owner: the consumer on every sync, or the aggregator
  with a pod informer. Reserved as a future target field.
- **PodGroup.** Lines up with the Workload API for gang workloads, but
  only the Job controller creates PodGroups today and ReplicaSets have
  none. A future target field.
- **Node name.** Karpenter's natural unit, but it mixes two kinds of
  keys, someone has to expand node to pods, and it cannot carry the
  HPA's per-pod signal. Fits better as the first declarative rule.
- **Reverse reference on the workload.** A field on the ReplicaSet or
  Deployment pointing at the preference. Explicit opt-in, but it
  changes every workload API including CRDs (the Scale problem again)
  and has actors writing into user-owned specs.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

None.
