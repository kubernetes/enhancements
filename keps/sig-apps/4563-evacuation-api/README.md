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
# KEP-4563: Evacuation API

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
  - [Evacuation Instigator](#evacuation-instigator)
  - [Evacuee and Evacuator](#evacuee-and-evacuator)
  - [Evacuation controller](#evacuation-controller)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Disruptive Eviction](#disruptive-eviction)
- [Design Details](#design-details)
  - [Evacuation](#evacuation)
  - [Evacuation Instigator](#evacuation-instigator-1)
    - [Evacuation Instigator Finalizer](#evacuation-instigator-finalizer)
  - [Evacuator](#evacuator)
  - [Evacuation controller](#evacuation-controller-1)
  - [Evacuation API](#evacuation-api)
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
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Pod API](#pod-api)
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

This KEP proposes to add an Evacuation API to manage the evacuation of pods. Its mission is to
allow for cooperative evacuation (removal) of a pod, usually in order to run the pod on another
node. If the owner of the pod does not cooperate, the evacuation will try to resort to pod eviction
(API initiated Eviction). This API can be used to implement additional capabilities around node
draining, pod descheduling, or as a general interface between applications and/or controllers.

## Motivation

Many of today's solutions rely on eviction (API-initiated Eviction) as the goto-safe way to remove
a pod from a node (kubectl drain, descheduler, cluster autoscaler, partially scheduler preemption).
Unfortunately, this is done in an application agnostic way and can cause many problems.

From an application owner or developer perspective, the only standard tool they have to protect
them against eviction is a PodDisruptionBudget. This is sufficient in a basic scenario with a simple
multi-replica application. The edge case applications, where this does not work are very important
to the cluster admin or controllers managing workload distribution on nodes, as they can for
example block the node drain. And, in turn, very important to the application owner, as the admin
can then override the pod disruption budget and disrupt their sensitive application anyway.

This KEP is a prerequisite for the [Declarative Node Maintenance KEP](https://github.com/kubernetes/enhancements/pull/4213),
which describes other issues and consequences that would be solved by the Evacuation API.

The major issues are:

1. Without extra manual effort, an application running with a single replica has to settle for
   experiencing application downtime during the node drain. They cannot use PDBs with
   `minAvailable: 1` or `maxUnavailable: 0`, or they will block node maintenance. Not every user
   needs high availability either, due to a preference for a simpler deployment model, lack of
   application support for HA, or to minimize compute costs. Also, any automated solution needs
   to edit the PDB to account for the additional pod that needs to be spun to move the workload
   from one node to another. This has been discussed in issue [kubernetes/kubernetes#66811](https://github.com/kubernetes/kubernetes/issues/66811)
   and in issue [kubernetes/kubernetes#114877](https://github.com/kubernetes/kubernetes/issues/114877).
2. Similar to the first point, it is difficult to use PDBs for applications that can have a variable
   number of pods; for example applications with a configured horizontal pod autoscaler (HPA). These
   applications cannot be disrupted during a low load when they have only pod. However, it is
   possible to disrupt the pods during a high load without experiencing application downtime. If
   the minimum number of pods is 1, PDBs cannot be used without blocking the node drain. This has
   been discussed in issue [kubernetes/kubernetes#93476](https://github.com/kubernetes/kubernetes/issues/93476).
3. Graceful deletion of DaemonSet pods is currently only supported as part of (Linux) graceful node
   shutdown. The length of the shutdown is again not application specific and is set cluster-wide
   (optionally by priority) by the cluster admin. This does not take into account
   `.spec.terminationGracePeriodSeconds` of each pod and may cause premature termination of
   the application. This has been discussed in issue [kubernetes/kubernetes#75482](https://github.com/kubernetes/kubernetes/issues/75482)
   and in issue [kubernetes-sigs/cluster-api#6158](https://github.com/kubernetes-sigs/cluster-api/issues/6158).
4. Descheduler does not allow postponing eviction for applications that are unable to be evicted
   immediately. This can result in descheduling of incorrect set of pods. This is outlined in the
   KEP [kubernetes-sigs/descheduler#1354](https://github.com/kubernetes-sigs/descheduler/pull/1354).

### Goals

- Introduce new `evacuation.coordination.k8s.io` API.
- Introduce evacuation controller.

### Non-Goals

- Synchronizing of the evacuation status to the pod status.
- Introduce the evacuation concept even for types other than pods.

## Proposal

We will introduce a new term called evacuation. This is a contract between the evacuation instigator,
the evacuee, and the evacuator. The contract is enforced by the API and an evacuation controller.
We can think of evacuation as a managed and safer alternative to eviction.

### Evacuation Instigator

The evacuation instigator can be any entity in the system: node maintenance controller, descheduler,
cluster autoscaler, or any application/controller interfacing with the affected application/pods
(evacuee).

The instigator's responsibility is to communicate an intent to a pod that it should be evacuated,
according to the instigator's own internal rules. It should reconcile its intent in case the intent
is removed by a third party. And it should remove its intent when the evacuation is no longer necessary.

Example evacuation triggers:
- Node maintenance controller: node maintenance triggered by an admin.
- Descheduler: descheduling triggered by a descheduling rule.
- Cluster autoscaler: node downscaling triggered by a low node utilization.

It is understood that multiple evacuation instigators may request evacuation of the same pod at the
same time. The instigators should coordinate their intent and not remove the evacuation until all
instigators have dropped their intent.

### Evacuee and Evacuator

The evacuee can be any pod. The owner/controller of the pod is the evacuator. In a
special case, the evacuee can be its own evacuator. The evacuator should decide what action to take
when it observes an evacuation intent:
1. It can reject the evacuation and wait for the pod to be evicted by the evacuation controller.
2. It can do nothing and wait for the pod to be evacuated either by another evacuator or evicted by
   the evacuation controller.
3. It can start the evacuation and periodically respond to the evacuation intent to signal that the
   evacuation is in progress and not stuck. Evacuation is at the discretion of the evacuator and
   can take many forms:
   - Migration of data (both persistent and ephemeral) from one node to another.
   - Waiting for a cleanup and release of important resources held by the pod.
   - Waiting for important computations to complete.
   - Non-graceful deletion of the pod (`gracePeriodSeconds=0`).
   - Deletion of a pod that is covered by a blocking PodDisruptionBudget. The controller of the
     application should have additional logic to distinguish whether a disruption of a particular
     pod will disrupt the application as a whole.

In the end, the evacuation should always end with a pod being deleted by either the evacuator or
via an eviction by the evacuation controller.

We should discourage the creation of preventive evacuations, so that they do not end up as
another PDB. So we should design the API appropriately and also not allow behaviors that do not
conform to the evacuation contract.

### Evacuation controller

In order to fully enforce the evacuation contract and prevent code duplication among evacuation
instigators, we will introduce a new controller called the evacuation controller.

Its responsibility is to observe evacuation requests from instigators and periodically check that
evacuators are making progress in evacuating evacuees. It is important to see a consistent effort
by the evacuators to reconcile the progress of the evacuation. This is important to prevent stuck
evacuations that could bring node maintenance to a halt. If the evacuation controller detects
that the evacuation progress updates have stopped (or never started), it will resort to pod
eviction by calling the eviction API (taking PodDisruptionBudgets into consideration).

It is also responsible for garbage collection/deletion of existing evacuations whose pods have
already been deleted.

### User Stories (Optional)

#### Story 1

As a cluster admin I want high-level components like node maintenance (planned replacement of
kubectl drain), scheduler, descheduler to use the Evacuation API to gracefully remove pods from a
set of nodes. I also want to see the progress of ongoing evacuations and be able to debug them if
something goes wrong. This means to:
- Easily identify pods that have accepted evacuation and are making progress. If possible to be
  able to see evacuation's ETA.
- Identify pods that should be evicted instead of evacuated and to distinguish pods that are
  failing eviction.
- See additional debug information from the evacuator and to be able to identify the evacuator.

#### Story 2

As an application owner, I want to run single replica applications without disruptions and have the
ability to easily migrate the workload pods from one node to another. This also applies to
applications with larger number of replicas that prefer to surge (upscale) pods first instead of
downscaling.

### Notes/Constraints/Caveats (Optional)

DaemonSet pods and mirror/static pods can potentially be evacuated, but cannot be evicted because
they can run critical services.
- The daemonset controller can be made Evacuation aware and graceful terminate pods on nodes under
  NodeMaintenance (proposed feature in [Declarative Node Maintenance KEP](https://github.com/kubernetes/enhancements/pull/4213)).
  The NodeMaintenance would be used by the daemonset controller as a scheduling hint to not
  restart the pods on a node.
- An application with access to a node's filesystem could observe the Evacuation of a mirror pod
  and remove the static pod manifest from the node, resulting in static pod termination.
- Kubelet could be made aware of the static pods Evacuations and allow their termination in
  certain scenarios (e.g., during the final phase of node maintenance).

### Risks and Mitigations

<<[UNRESOLVED TBD]>>
How much extra risk is being added here?
<<[/UNRESOLVED]>>

#### Disruptive Eviction

When using kubectl drain, pods without owning controller and pods with local storage
(having `emptyDir` volumes) are not evicted by default. We have decided to evict most
of the pods (except DaemonSet and mirror pods) by default. In the motivation section of the
[Declarative Node Maintenance KEP](https://github.com/kubernetes/enhancements/pull/4213),
we can see many administrators override these default settings and many components evict all pods
indiscriminately. There are also many ways that users use to prevent the eviction;
PodDisruptionBudgets, validating admission webhooks, or just plain HA. Users who want to protect
their applications in today's clusters should already be aware that they should be able to handle
sudden evictions. Therefore, it should be okay for the evacuation controller to evict these pods.

To mitigate the sudden eviction problem, users should use PodDisruptionBudgets or HA.

## Design Details

### Evacuation

We will introduce a new type called `evacuation.coordination.k8s.io`  to enforce the contract
between the evacuation instigators and the evacuators. This type is a bit similar to
`leases.coordination.k8s.io` in that it requires multiple actors to synchronize the state. Which in
our case is the progress of the evacuation.

### Evacuation Instigator
[Evacuation Instigator](#evacuation-instigator) section provides a general overview.

There can be many evacuation instigators for a single Evacuation. 

When an instigator decides that a pod needs to be evacuated, it should create an Evacuation:
- With a predictable name in the following format: `${POD_UID}-${POD_NAME_PREFIX}`. The pod name
  might be only partial as the full eviction name is limited to 253 characters.
- `.spec.podRef` should be set to fully identify the pod. Similar to the name, the UID should be
  specified to ensure that we do not evacuate a pod with the same name that appears immediately
  after the previous one is removed.
- `.spec.acceptDeadlineSeconds` should be set to a reasonable value. For example, 600 (10m) to
  allow reasonably fast feedback, but still give the potential evacuator sufficient time to begin
  the evacuation.
- `.spec.progressDeadlineSeconds` should be set to a reasonable value. For example, 3600 (1h) to
  allow for potential evacuator disruptions.
- The pod labels should be copied to the Evacuation labels to allow for custom label selectors when
  observing the evacuations.

It should also add itself to the Evacuation finalizers upon creation. If the evacuation already
exists, the instigator should still add itself to the finalizers. The finalizers are used for:
- Tracking the instigators of this evacuation intent.
- Processing the evacuation result by the instigator once the evacuation is complete.
- Ensure the evacuations are not stomped over and periodically deleted. This means that an
  Evacuation with a DeletionTimestamp is still a valid evacuation.

If the evacuation is no longer needed, the instigator should remove itself from the finalizers.
The evacuation will then be deleted by the evacuation controller.

#### Evacuation Instigator Finalizer
To distinguish between instigator and other finalizers, instigators should use the
`evacuation-instigator` value. For example, the node maintenance instigator would use the
`maintenance-a.nodemaintenance.kubernetes.io/evacuation-instigator` finalizer.

Instigator finalizers are automatically removed upon pod removal by evacuation controller. If
the instigator needs to perform additional final tasks before the Evacuation object deletion, it
should create a second finalizer without the `evacuation-instigator` value.

### Evacuator

[Evacuee and Evacuator](#evacuee-and-evacuator) section provides a general overview.

The evacuator should observe the evacuation objects matching the pods that the evacuator manages.

If the evacuator is not interested in evacuating the pod, it should set `.status.evacuatorRef` and
`.status.accepted=false`. It can also ignore the evacuation object if it thinks there is another
evacuator that should start the evacuation. If there is none, the `.spec.acceptDeadlineSeconds`
will timeout and the pod will get evicted by the evacuation controller.

If the evacuator is interested in evacuating the pod it should:
- Set `.status.evacuatorRef` with as much self-identifying information as possible.
- Set `.status.accepted=true` to signal that the evacuation is commencing.
- Set `.status.evacuationProgressTimestamp` to the present time to signal that the evacuation is
  not stuck.
- Set `.status.expectedEvacuationFinishTime` if a reasonable estimation can be made of how long
  the evacuation will take. This can be modified during the evacuation to change the estimate.
- Set `.status.message` to inform about the progress of the evacuation in human-readable form.
- Optionally, `.status.conditions` can be set for additional details about the evacuation.
- Optionally, an event can be emitted to inform about the start of the evacuation.

After starting the pod evacuation, the evacuator should take a look at
`.spec.progressDeadlineSeconds` and make sure to update the status periodically in a less time than
that duration. For example, if `.spec.progressDeadlineSeconds` is 3600 (60m), it may update
the status every 5 minutes. The status updates should look as follows:
- Set `.status.evacuationProgressTimestamp` to the present time to signal that the evacuation is
  not stuck.
- Set `.status.expectedEvacuationFinishTime` if a better estimation can be made of how long
  the evacuation will take.
- Set `.status.message` to inform about the progress of the evacuation in human-readable form.
- Optionally, `.status.conditions` can be set for additional details about the evacuation.
- Optionally, an event can be emitted to inform about the progress of the evacuation. Or lack
  thereof, if the evacuation is blocked. The evacuator should ensure that an appropriate number of
  events is emitted.

The end of the evacuation is communicated by the pod deletion.

If the evacuator does not wish the evacuation to never be canceled, it should also become the
evacuation instigator and set a finalizer on the evacuation to ensure that the evacuation completes
atomically (see [Evacuation Instigator Finalizer](#evacuation-instigator-finalizer)).
This is done either for tracking purposes by the evacuator or to make sure that there are no
multiple evacuations of the same objects in a row, as restarting the evacuation can be expensive.

### Evacuation controller

[Evacuation controller](#evacuation-controller) section provides a general overview.

The evacuation controller will observe pod evacuations and evict pods that cannot be evacuated by
calling the eviction API. 

Pods that cannot be evacuated are:
- Evacuation's `.status.accepted` field is empty, and `.spec.acceptDeadlineSeconds` has elapsed
  since `.metadata.creationTimestamp`.
- Evacuation's `.status.accepted` field is `true` and `.spec.progressDeadlineSeconds` has elapsed
  since `.status.evacuationProgressTimestamp`.
- Evacuation's `.status.accepted` field is `false`, indicating that the evacuator has rejected the
  evacuation.

Eviction of DaemonSet pods and mirror pods is not supported. However, the Evacuation can still be
used to evacuate them by other means. Terminating pods will not attempt eviction either.

If the pod eviction fails, e.g. due to a blocking PodDisruptionBudget, the
`.status.failedEvictionCounter` is incremented and the pod is added back to the queue with
exponential backoff (maximum approx. 15 minutes). If there is a positive progress update in the
`.status` of the Evacuation, it will cancel the eviction.

See [Evacuation Instigator Finalizer](#evacuation-instigator-finalizer) for how to distinguish
between instigator and other finalizers.

The controller deletes the evacuation object if:
- There are no instigator finalizers (instigators have canceled their intent to evacuate the pod).
- The referenced pod no longer exists (has been deleted from etcd), signaling a successful evacuation.
  For convenience, we will also remove instigator finalizers when the evacuation task is complete.
  Other finalizers will still block deletion.

### Evacuation API

```golang

// Evacuation defines an evacuation.
type Evacuation struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// The labels of the evacuation object will be copied from pod's .metadata.labels.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Spec defines the evacuation.
	// https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec EvacuationSpec `json:"spec,omitempty" protobuf:"bytes,2,opt,name=spec"`

	// Status represents the most recently observed status of the evacuation.
	// Populated by the current evacuator.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Status EvacuationStatus `json:"status,omitempty" protobuf:"bytes,3,opt,name=status"`
}

// EvacuationSpec is a specification of an Evacuation.
type EvacuationSpec struct {
	// PodRef references a pod that is subject to evacuation.
	// This field is required and immutable.
	// +optional
	PodRef *LocalPodReference `json:"podRef,omitempty" protobuf:"bytes,1,opt,name=podRef"`

	// AcceptDeadlineSeconds is a maximum amount of time an evacuator should take to respond to an
	// evacuation. It should respond by setting .status.accepted to true or false, and providing
	// .status.evacuatorRef and .status.evacuationProgressTimestamp. The evacuator should then
	// periodically update .status.evacuationProgressTimestamp.
	// If the .status.accepted is not updated within the duration of .spec.acceptDeadlineSeconds,
	// the evacuated pod will be evicted using the Eviction API.
	//
	// The minimum value is 60 (1m) and the maximum value is 3600 (1h).
	// The value must be less than or equal to .spec.progressDeadlineSeconds.
	// This field is required and immutable.
	AcceptDeadlineSeconds int32 `json:"acceptDeadlineSeconds" protobuf:"varint,2,opt,name=acceptDeadlineSeconds"`
	
	// ProgressDeadlineSeconds is a maximum amount of time an evacuator should take to report on a
	// progress by updating the .status.evacuationProgressTimestamp.
	// If the .status.evacuationProgressTimestamp is not updated within the duration of
	// .spec.progressDeadlineSeconds, the evacuated pod will be evicted using the Eviction API.
	//
	// The minimum value is 60 (1m) and the maximum value is 43200 (12h).
	// The value must be greater than or equal to .spec.acceptDeadlineSeconds.
	// This field is required and immutable.
	ProgressDeadlineSeconds int32 `json:"progressDeadlineSeconds" protobuf:"varint,3,opt,name=progressDeadlineSeconds"`
}

// LocalPodReference contains enough information to locate the referenced pod inside the same namespace.
type LocalPodReference struct {
	// Name of the pod.
	// This field is required.
	Name string `json:"name" protobuf:"bytes,1,opt,name=name"`
	// UID of the pod.
	// This field is required.
	UID string `json:"uid" protobuf:"bytes,2,opt,name=uid"`
}

// EvacuationStatus represents the most recently observed status of the evacuation.
type EvacuationStatus struct {
	// EvacuatorRef references the evacuator of the current evacuation. 
	// This field should be set when Accepted field is set.
	// +optional
	EvacuatorRef *EvacuatorReference `json:"evacuatorRef,omitempty" protobuf:"bytes,1,opt,name=evacuatorRef"`

	// If set to true, EvacuationProgressTimestamp should also be set, and then updated periodically.
	// If set to false, the pod will be evicted immediately.
	// This field cannot be changed once set.
	// +optional
	Accepted *bool `json:"accepted,omitempty" protobuf:"varint,2,opt,name=accepted"`

	// EvacuationProgressTimestamp is the time at which the evacuation was reported to be in progress by the evacuator.
	// Cannot be set to the future time (after taking time skew into account).
	// This field must be set when the Accepted field is set to true.
	// +optional
	EvacuationProgressTimestamp *metav1.Time `json:"evacuationProgressTimestamp,omitempty" protobuf:"bytes,3,opt,name=evacuationProgressTimestamp"`

	// ExpectedEvacuationFinishTime is the time at which the evacuation is expected to end.
	// May be empty if no estimate can be made.
	// +optional
	ExpectedEvacuationFinishTime *metav1.Time `json:"expectedEvacuationFinishTime,omitempty" protobuf:"bytes,4,opt,name=expectedEvacuationFinishTime"`
	
	// The number of unsuccessful attempts to evict the referenced pod, e.g. due to a PDB.
	// This field is required.
	FailedEvictionCounter int32 `json:"failedEvictionCounter" protobuf:"varint,5,opt,name=failedEvictionCounter"`
	
	// Message is a human readable message indicating details about the evacuation.
	// This may be an empty string.
	// +required
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MaxLength=32768
	Message string `json:"message" protobuf:"bytes,6,opt,name=message"`

	// Conditions can be used by evacuators to share additional information about the evacuation.
	// +optional    
	// +patchMergeKey=type    
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,7,rep,name=conditions"`
}

// EvacuatorReference contains information that allows you to locate the evacuator responding to
// this evacuation. If the evacuator is not represented by a Kubernetes object, only the name or
// UID can be used.
type EvacuatorReference struct {
	// APIGroup of the evacuator.
	// +optional
	APIGroup *string `json:"apiGroup,omitempty" protobuf:"bytes,1,opt,name=apiGroup"`
	// Kind of the evacuator.
	// +optional
	Kind *string `json:"kind,omitempty" protobuf:"bytes,2,opt,name=kind"`
	// Name of the evacuator.
	// Name or UID is required.
	// +optional
	Name *string `json:"name,omitempty" protobuf:"bytes,3,opt,name=name"`
	// Namespace of the evacuator.
	// +optional
	Namespace *string `json:"namespace,omitempty" protobuf:"bytes,4,opt,name=namespace"`
	// UID of the evacuator.
	// Name or UID is required.
	// +optional
	UID *string `json:"uid,omitempty" protobuf:"bytes,5,opt,name=uid"`
}
```


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
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

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
- Does this enhancement involve coordinating behavior in the control plane and nodes?
- How does an n-3 kubelet or kube-proxy without this feature available behave when this feature is used?
- How does an n-1 kube-controller-manager or kube-scheduler without this feature available behave when this feature is used?
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
    of a node?

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
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

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->


### Pod API

The original proposal in the [Declarative Node Maintenance KEP](https://github.com/kubernetes/enhancements/pull/4213)
was to use pod conditions to communicate the intent of Evacuation between the evacuation
instigator and the evacuator.

Two new condition types would be introduced:
1. `EvacuationRequested` condition should be set by a controller (e.g. node maintenance controller)
   on the pod to signal a request to evacuate the pod from the node. A reason should be given to
   identify the requester, in our case `EvacuationByNodeMaintenance` (similar to how `DisruptionTarget`
   condition behaves). The requester has the ability to withdraw the request by removing the
   condition or setting the condition status to `False`. Other controllers can also use this
   condition to request evacuation. For example, a descheduler could set this condition to `True`
   and give a `EvacuationByDescheduler` reason. Such a controller should not overwrite an  existing
   request and should wait for either the pod deletion or removal of the evacuation request. The
   owning controller of the pod should observe the pod's conditions and respond to the
   `EvacuationRequested` by accepting it and setting an `EvacuationInitiated` condition to `True` in
   the pod conditions.
2. `EvacuationInitiated` condition should be set by the owning controller to signal that work is
   being done to either remove or evacuate/migrate the pod to another node. The draining
   process/controller should wait a reasonable amount of time (3 minutes) to observe the appearance
   of the condition or change of the condition status to `True`. The draining process should then
   skip such a pod and leave its management to the owning controller. If `EvacuationInitiated`
   condition does not appear after 3 minutes, the draining process will begin evicting or deleting
   the pod. If the owning controller is unable to remove or migrate the pod, it should set the
   `EvacuationInitiated` condition status back to `False` to give the eviction a chance to start.

```golang

type PodConditionType string

const (
    ...
    EvacuationRequested PodConditionType = "EvacuationRequested"
    EvacuationInitiated PodConditionType = "EvacuationInitiated"
)

const (
    ...
    PodReasonNodeMaintenance = "NodeMaintenance"
)

```

There are multiple issues with this approach:
- Multiple evacuations instigators can try to evacuate a pod at the same time and there is no
  simple way to track this in a single condition. It is much easier to handle concurrency on a
  standalone resource.
- The observability of the evacuation progress, evacuator reference and other useful information
  is lacking as we can see requested in the [User Stories](#user-stories-optional).

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
