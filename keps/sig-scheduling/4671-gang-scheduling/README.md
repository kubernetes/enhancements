# KEP-4671: Gang Scheduling using Workload Object


<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1: Gang-scheduling of a Job](#story-1-gang-scheduling-of-a-job)
    - [Story 2: Gang-scheduling of a custom workload](#story-2-gang-scheduling-of-a-custom-workload)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [The API needs to be extended in an unpredictable way](#the-api-needs-to-be-extended-in-an-unpredictable-way)
- [Design Details](#design-details)
  - [Naming](#naming)
  - [Associating Pod into PodGroups](#associating-pod-into-podgroups)
  - [API](#api)
  - [Scheduler Changes](#scheduler-changes)
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

In this KEP, kube-scheduler is modified to support gang scheduling[^1]. We focus on framework support and building blocks, not the ideal gang-scheduling algorithm - it can come as a follow-up. We start with simpler implementation of gang scheduling, kube-scheduler identifies pods that are in a group and waits until all pods reach the same stage of the scheduling/binding cycle before allowing any pods from the group to advance past that point.  If not all pods can reach that point before a timeout expires, then the scheduler stops trying to schedule that group, and all pods release all their resources.  This allows other workloads to try to allocate those resources.

A new core type called `Workload` is introduced to tell the kube-scheduler that a group of pods should be scheduled together and any policy options related to gang scheduling. Pods have an object reference in their spec to their `Workload`, if any. The `Workload` object is intended to evolve[^2] via future KEPs to support additional kube-scheduler improvements, such as topology-aware scheduling.

## Motivation

Parallel applications can require communication between every pod in order to begin execution, and then ongoing communication between all pods (such as barrier or all-reduce operations) in order to make progress.  Starting all pods as close to the same time is necessary to run these workloads.  Otherwise, either expensive compute resources are idle, or the application may fail due to an application-level communication timeout.

Gang scheduling has been implemented outside of kube-scheduler at least 4 times[^3].  Some controllers are starting to support multiple Gang Schedulers in order to be portable across different clusters.  Moving support into kube-scheduler makes gang scheduling support available in all Kubernetes distributions and eventually may allow workload controllers to rely on a standard interface to request gang scheduling from the standard or custom schedulers. A standard API may also allow other components to understand workload needs better (such as cluster autoscalers).

Workloads that require gang scheduling often also need all members of the gang to be as topologically "close" to one another as possible, in order to perform adequately. Existing Pod affinity rules influence pod placement, but they do not consider the gang as a unit of scheduling and they do not cause the scheduler to efficiently try multiple mutually exclusive placement options for a set of pods. The design of the Workload object introduced in this KEP anticipates how Gang Scheduling support can evolve over subsequent KEPs into full Topology-aware scheduling support in kube-scheduler.

The `Workload` object will allow kube-scheduler to be aware that pods are part of workloads with complex internal structure.  Those workloads include builtins like `Job` and `StatefulSet`, and custom workloads, like `JobSet`, `LeaderWorkerSet`, `MPIJob` and `TrainJob`. All of these workload types are used for AI training and inference use cases.


### Goals
- Introduce a concept of a `Workload` as a primary building block for workload-aware scheduling vision
- Implement the first version of `Workload` API necessary for defining a Gang
- Ensuring that we can extend `Workload` API in backward compatible way toward north-star API
- Ensuring that `Workload` API will be usable for both built-in and third-party workload controllers and APIs
- Implement first version of gang-scheduling in kube-scheduler
- Provide full backward compatibility for all existing scheduling features

### Non-Goals

- Take away responsibility to create pods from controllers.
- Bring fairness or multiple workload queues in kube-scheduler. Kueue and Volcano.sh will continue to provide this.
- Map all the declarative state and behaviors into `Workload` object. It is focused only on scheduling-related parts.

The following are non-goals for this KEP but will probably soon appear to be goals for follow-up KEPs:

- Introduce a concept of `Reservation` that can be later consumed by pods.
- Workload-level preemption.
- Address resource contention between different schedulers (including possible deadlocks).
- Address the problem of premature preemptions in case the higher priority workloads does not
  eventually schedule.


## Proposal

The `spec.workload` field will be added to the Pod resource.  A sample pod with this new field looks like this:
```yaml
apiVersion: v1
kind: Pod
spec:
  ...
  workload:
    name: job-1
  ...
```

The above pod might be one of several pods created by a `Job` like this.
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: job-1
spec:
  completions: 100
  parallelism: 100
  completionMode: Indexed
  template:
    spec:
      workload:
        name: job-1
      restartPolicy: OnFailure
      containers:
      - name: ml-worker
        image: awesome-training-program:v1 
        command: ["python", "train.py"]
        resources:
          limits:
            nvidia.com/gpu: 1
        env:
        - name: JOB_COMPLETION_INDEX
          valueFrom:
            fieldRef:
              fieldPath:
               "metadata.annotations['batch.kubernetes.io/job-completion-index']"
```

The `Workload` core resource will be introduced. A `Workload` does not create any pods. It just describes what pods the scheduler should expect to see, and how to treat them.   

 It does not affect pod creation by Job or any other controller.  A sample resource looks like this:
```yaml
apiVersion: scheduling/v1alpha1   
kind: Workload
metadata:
  namespace: ns-1
  name: job-1
spec:
  podGroups:   # or gangGroups -- TBD
    - name: "pg1"
      gangMode: Single
      gangSchedulingPolicy:
        minCount: 100
        schedulingTimeoutSeconds: 60
```


### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1: Gang-scheduling of a Job

I have a tightly-coupled job and I want its pods to be scheduled and run only when the
resources for all of them can be found in the cluster.

#### Story 2: Gang-scheduling of a custom workload

I have my own workload definition (CRD) and controller managing its lifecycle. I would
like to be able to easily benefit of gang-scheduling feature supported by the core
Kubernetes without extensive changes to my custom controller.


### Risks and Mitigations

#### The API needs to be extended in an unpredictable way

We try to mitigate it by an extensive analysis of usecases and already sketching
how we envision the direction in which the API will need to evolve to support further
usecases. You can read more about it in the [extended proposal] document.


## Design Details

### Naming

* `Workload` is the resource Kind.
* `scheduling` is the ApiGroup.
* `spec.workload` is the name of the new field in pod.
* Within a Workload there is a list of groups of pods. Each group represents a top-level division of pods within a Workload.  Each group can be independently gang scheduled (or not use gang scheduling). This group is named
  <<[UNRESOLVED community feedback requested]>> `PodGroup` or `GangGroup` for the top level. <<[/UNRESOLVED]>>.
* In a future , we expect that this group can optionally specify further subdivision into sub groups.  Each sub-group can have an index.  The indexes go from 0 to N, without repeats or gaps. These subgroups are called
  <<[UNRESOLVED depending on previous unresolved item]>> `PodSubGroup` if `PodGroup` is chosen, or else `RankedGroup` if `GangGroup` is chosen<<[/UNRESOLVED]>>.
* In subsequent KEPs, we expect that a sub-group can optionally specify further subdivision into pod equivalence classes.  All pods in a pod equivalence class have the same values for all fields that affect scheduling feasibility.  These pod equivalence classes are called
  <<[UNRESOLVED depending on a previous unresolved item]>> `PodSet` if `PodGroup` is chosen, or else `EqGroup` if `GangGroup` is chosen<<[/UNRESOLVED]>>.

### Associating Pod into PodGroups

When a `Workload` consists of a single group of pods needing Gang Scheduling, it is clear which pods belong to the group from the `spec.workload.name` field of the pod.  However `Workload` supports listing multiple list items, and a list item can represent a single group, or a set of identical replica groups.
In these cases, there needs to be additional information to indicate which group a pod belongs to.

We proposed to extend the newly introduced `pod.spec.workload` field with additional information
to include that information. More specifically, the `pod.spec.workload` field is of type `PodWorkload`
and is defined as following:

```go
// WorkloadReference identifies the Workload object and PodGroup membership
// that a Pod belongs to. The scheduler uses this information to enforce
// gang scheduling semantics.
type WorkloadReference struct {
    // Workload defines the name of the Workload object this pod belongs to.
    Workload string

    // PodGroup defines the name of the PodGroup within a Workload this pod belongs to.
    PodGroup string
    // PodGroupReplicaIndex is the replica index of the PodGroup that this pod
    // belong to when the workload is running ReplicatedGangMode. In this mode,
    // a workload may create multiple identical PodGroups.
    // For workload in a different mode, this field is unset.
    PodGroupReplicaIndex string
}
```

At least for Alpha, we start with `WorkloadReference` to be immutable field in the Pod.
In further phases, we may decide to relax validation and allow for setting some of the fields later.
Moreover, the visibility into issues (debuggability) will depend on [#5510], but we don't
treat it as a blocker.

[#5510]: https://github.com/kubernetes/enhancements/pull/5510

The example below shows how this could look like for with the following `Workload` object:

```yaml
apiVersion: scheduling/v1alpha1   
kind: Workload
metadata:
  name: jobset
spec:
  podGroups:   # or gangGroups -- TBD
    - name: "job-1"
      gangMode: Replicated
      replicas: 4
      gangSchedulingPolicy:
        minCount: 100
        schedulingTimeoutSeconds: 60
```

```yaml
apiVersion: v1
kind: Pod
name:
  jobset-job-1-abc123
spec:
  ...
  workload:
    name: jobset
    podGroup: job-1
    podGroupReplica: 2
  ...

```

We decided for this option because it is more succint and makes the role of a pod clear just
from inspecting the pod (and simple/efficient to group).
We acknowledge the fact that this option may require additional minor changes in the controllers
to adopt this pattern (e.g. for LeaderWorkerSet we will need to populate the pod template
similarly that we currently populate the labels).

The primary alternative we consider was to introduce the the `PodGroupSelector` on each `PodGroup`
to identify pods belonging to it. However, with this pattern:
- there are additional corner cases (e.g. a pod links to a workload but none of its PodGroups matching
  that pod)
- for replicated gang, we can't use the full label selector, but rather support specifying only the
  label key, similar to `MatchLabelKeys` in pod affinity


### API

The `Workload` type will be defined with the following structure:
```go
type Workload struct {
	metav1.TypeMeta
	metav1.ObjectMeta
	Spec WorkloadSpec
	Status WorkloadStatus
}

// WorkloadSpec describes a workload in a portable way that scheduler and related
// tools can understand.  
type WorkloadSpec struct {
    // ControllerRef points to the true workload, e.g. Deployment.
    // It is optional to set and is intended to make this mapping easier for
    // things like CLI tools.
    // This field is immutable.
    ControllerRef *v1.ObjectReference

    // PodGroups is a list of groups of pods.
    // Each group may request gang scheduling.
    PodGroups []PodGroup 
}

type GangMode string
const (
	// GangModeSingle means that all pods in this PodGroup need to be scheduled as one gang.
	GangModeSingle GangMode = "Single"

	// GangModeOff means that all pods in this PodGroup do not need to be scheduled as a gang.
	GangModeOff GangMode = "Off"

	// GangModeReplicatedGang means that there is a variable number of identical copies of this PodGroup,
    //  as specified in Replicas, and each copy needs to be independently gang scheduled.
	GangModeReplicated GangMode = "Replicated"
)

// GangSchedulingPolicy holds options that affect how gang scheduling of one PodGroup is handled by the scheduler.
type GangSchedulingPolicy struct {
    // SchedulingTimeoutSeconds defines the timeout for the scheduling logic.
    // Namely it's timeout from the moment when `minCount` pods show up in
    // PreEnqueue, until those pods are observed in WaitOnPermit - for context
    // see https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#interfaces
	SchedulingTimeoutSeconds *int
	MinCount *int
}

// PodGroup is a group of pods that may contain multiple shapes (EqGroups) and may contain
// multiple dense indexes (RankedGroups) and which can optionally be replicated in a variable
// number of identical copies.
//
// TODO: Decide on the naming: PodGroup vs GangGroup.
type PodGroup struct {
    Name *string
    GangMode *GangMode // default is "Single"

    // Optional when GangMode = "ReplicatedGang".
    // Forbidden otherwise.
    Replicas int

    // GangSchedulingPolicy defines the options applying to all pods in this gang.
    // Forbidden if GangMode is set to "Off".
    GangSchedulingPolicy GangSchedulingPolicy
}


type WorkloadStatus struct {
  // Necessary status fields TBD.
}
```

The individual `PodGroups` and `PodGroup` replicas are treated as independent gangs. As an example, if one of
the groups can be scheduled and the other can't be - this is exactly what will happen. If the underlying
user intention was to have either both of them or none of them running, they should form a single group and
not be split into two. A `LeaderWorkerSet` is a good example of it, where a single `PodGroup` replica consists
of a single leader and `N` workers and that forms a scheduling (and runtime unit), but workload as a whole
may consist of a number of such replicas.

### Scheduler Changes

The kube-scheduler will be watching for `Workload` objects (using informers) and will use them to map pods
to and from their `Workload` objects.

In the initial implementation, we expect users to create the `Workload` objects. In the next steps controllers
will be updated to create an appropriate `Workload` objects themselves whenever they can appropriately infer
the intention from the desired state.
Note that given scheduling options are stored in the `Workload` object, pods linked to the `Workload`
object will not be scheduled until this `Workload` object is created and observed by the kube-scheduler.

The north star vision for gang scheduling implementation should satisfy the following requirements:

1. Ensure that pods being part of a gang are not bound if all pods belonging to it can't be scheduled.
2. Provide the "optimal enough" placement by considering all pods from a gang together.
3. Avoid deadlock scenario when multiple workloads are being scheduled at the same time by kube-scheduler.
4. Avoid deadlock scenario when multiple workloads are being scheduled at the same time by different
   schedulers.
5. Avoid premature preemptions of already running pods in case a higher priority gang will be rejected.
6. Support gang-level (or workload-level in general) level preemption (if pods form a gang also
   from a runtime perspective, they can't be preempted individually).
7. Updating workload status and triggering rescheduling when a gang failed binding in the all-or-nothing
   fashion.

Addressing all these requirements in a single shot would be a huge change, so as part ot this KEP we
will only focus on a subset of those. However, we very briefly sketch the path towards the vision to
ensure that this KEP is moving in the right direction.

For `Alpha`, we are focusing on introducing the concept of the `Workload` and plumbing it into
kube-scheduler in the simplest possible way. We will implement a new plugin implementing the following
hooks:
- PreEnqueue - used as a barrier to wait for the `Workload` object and all the necessary pods to be
  observed by the scheduler before even considering them for actual scheduling
- WaitOnPermit - used as a barrier to wait for the pods to be assigned to the nodes before initiating
  potential preemptions and their bindings

This seems to be the simplest possible implementation to address the requirement (1). We are consciously
ignoring the rest of the requirements for `Alpha` phase.


For `Beta`, we want to also touch requirements (2) and (3) by extending the scheduling framework with
a new dedicated phase (tentatively called Workload). In that phase,
kube-scheduler will be looking at all pods from a gang (part of `Workload`) and compute the placement
for all of these pods in a single scheduling cycle. Those placements will be stored only in-memory and
block the required resources from scheduling. Tentively we plan to use `NominatedNodeName` field for it.
After that, pods will go through regular pod-by-pod scheduling phases (including Filter and Score)
with a nomination as a form of validation the proposed placement and execution of this placement decision.
Therefore we expect the order of processing pods won't ever be important, but all-or-nothing nature of
gangs will be preserved while advancing through the further steps of the binding process.

While we will not target addressing "optimal enough" part of requirement (2), we will assure that we
can process all gang pods together. The single scheduling cycle and blocking resources in beta
will address the requirement (3).

We will also introduce delayed preemption by moving it after `WaitOnPermit` phase. Together with
introduction of a dedicated phase for scheduling all pods in a single scheduling cycle this
will address the requirement (5). If accompanied with blocking the resources in-memory as
mentioned above, this basically mitigates the problem.

More detail about scheduler changes is described in [this document](https://docs.google.com/document/d/1lMYkDuGqEoZWfE2b8vjQx0vHieOMyfmi6VHUef5-5is/edit?tab=t.0#heading=h.1p88ilpefnb).


We will continue with further improvements on top of it with follow-up KEPs. We are planning to
introduce the concept of `Reservation` that will allow to treat distributed subset of resources as
a single unit from scheduling perspective. With that, the proposed placement being a result of
the scheduling decision of the `Workload` phase will become a `Reservation`. This will become the
coordination point and a mechanism for multiple schedulers to share the underlying infrastructure
addressing the requirement (4). This will also be a critical building block for workload-level
preemption and addressing requirement (6). Finally, this will allow to address the few remaining
corner cases around unnecessary preemption - requirement (5), such as blocking DRA resources
(which we can't solve with NominatedNodeName). Further extensions to `Reservation` with different
states (e.g. not yet block resources) will help with improving the scheduling accuracy.
Finally making the binding process aware of gangs will allow to make sure the process is either
successful or triggers workload rescheduling satisfying requirement (7).

The workload-aware preemption is tightly coupled, but separate feature that will also be designed
in a dedicated KEP. The current vision includes introducing a dedicated preemption policy (that
will result in pods no longer being treated individually for preemption purposes) which makes it
an additive feature. However, having a next level of details is required to ensure that we really
have a feasible backward-compatible plan before promoting this feature to Beta.

However, approval for this KEP is NOT an approval for this vision. We only sketch it to show that
we see a viable path forward from the proposed design that will not require significant rework.


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

N/A

##### Unit tests

- `k8s.io/kubernetes/pkg/scheduler`: `2025-10-02` - 81.7%
- `k8s.io/kubernetes/pkg/scheduler/backend/queue`: `2025-10-02` - 91.4%
- `k8s.io/kubernetes/pkg/scheduler/framework`: `2025-10-02` - 81.7%
- `k8s.io/kubernetes/pkg/scheduler/framework`: `2025-10-02` - 81.7%
- `k8s.io/kubernetes/pkg/scheduler/framework/preemption`: `2025-10-02` - 64.2%
- `k8s.io/kubernetes/pkg/scheduler/framework/util/assumecache`: `2025-10-02` - 86.2%

##### Integration tests

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

We will create integration test(s) to ensure basic functionalities of gang-scheduling including:
- Pods linked to the non-existing workload are not scheduled
- Pods get unblocked when workload is created and observed by scheduler
- Pods are not scheduled if there is no space for the whole gang

In Beta, we will add tests to verify that deadlocks are not happening.

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

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

We will add basic API tests for the the new `Workload` API, that will later be
promoted to the conformance.

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/e2e/...): [SIG ...](https://testgrid.k8s.io/sig-...?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)

### Graduation Criteria

#### Alpha

- Workload API is introduced behind Workload feature flag
- API tests for Workload API (that will be promoted to conformance in GA release)
- kube-scheduler implements first version of gang-scheduling based on groups defined in the Workload object

#### Beta

- Providing "optimal enough" placement by considering all pods from a gang together
- Avoiding deadlock scenario when multiple workloads are being scheduled at the same time
  by kube-scheduler
- Implementing delayed preemption to avoid premature preemptions
- Workload-aware preemption design to ensure we won't break backward compatibility with it.

#### GA

- TBD in for Beta release


### Upgrade / Downgrade Strategy

This KEP is completely additive and can safely fallback to the original behavior on downgrade.

This KEP effectively boils down to two separate functionalities:
- the Workload API and new field in Pod API that allows linking Pods to Workloads
- scheduler changes implementing the gang scheduling functionality

When user upgrades the cluster to the version that supports these two features:
- they can start using the new API by creating Workload objects and linking pods to it via
  explicitly specifying their new `spec.workload` field
- scheduler automatically uses the new extensions and tries to schedule all pods from a given
  gang in a scheduling group based on the defined `Workload` objects

When user downgrades the cluster to the version that no longer supports these two features:
- the `Workload` objects can no longer be created (the existing ones are not removed though)
- the `spec.workload` field can no longer be set on the Pods (the already set fields continue
  to be set though)
- scheduler reverts to the original behavior of scheduling one pod at a time ignoring
  existence of `Workload` objects and pods being linked to them


### Version Skew Strategy

The feature is limited to the control plane, so the version skew with nodes (kubelets) doesn't matter.

For the API changes (introduction of Workload API and the new field in Pod API), the old version of
components (in particular kube-apiserver) may not handle those. Thus, users should not set those
fields before confirming all control-plane instances were upgraded to the version supporting those.

For the gang-scheduling itself, this is purely kube-scheduler in-memory feature, so the skew doesn't
really matter (as there is always only single kube-scheduler instance being a leader).


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

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: Workload/GenericWorkload/NativeWorkload
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
  - Feature gate name: GangScheduling
  - Components depending on the feature gate:
    - kube-scheduler
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No. Gang scheduling is triggerred purely via existence of Workload objects and
those are not yet created automatically behind the scenes.


###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. The GangScheduling features gate need to be switched off to disabled gang scheduling
functionality.
If additionally the API changes needs to be disabled, the Workload feature gate needs to
also be disabled. However, the content of `spec.workload` fields in Pod objects will not be
cleared, as well as the existing Workload objects will not be deleted.


###### What happens if we reenable the feature if it was previously rolled back?

The feature should start working again.
However, the user need to remember that some Workload objects could already be stored
in etcd and may affect the behavior of some of the existing workloads.


###### Are there any tests for feature enablement/disablement?

No.
The enablement/disablement for the new field in Pod API will be added similarly to this PR:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282

Note that gang-scheduling itself is purely in-memory feature, so feature themselves are enough.


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

###### Does this feature depend on any specific services running in the cluster?

No dependendies other than the components where the feature is implemented
(kube-apiserver and kube-scheduler).

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes:

Watching for workloads:
  - API call type: LIST+WATCH Workloads
  - estimated throughput: < XX/s
  - originating component: kube-scheduler, kube-controller-manager (GC)

Status updates (potentially not in Alpha):
  - API call type: PUT/PATCH Workloads
  - estimated throughput < XX/s
  - originating component: kube-scheduler

###### Will enabling / using this feature result in introducing new API types?

Yes:
  - API type: Workload
  - Supported number of objects per cluster: XX,000
  - Supported number of objects per namespace: XX,000

The above numbers should eventually match the numbers for built-in workload APIs
(e.g. Deployments, Jobs, StatefulSets, ...).

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. New field (spec.workload) is added to the Pod API:
  - API type: Pod
  - Estimated increase in size: XX-XXX bytes per object (depending on the final choice described
    in the Associating Pod into PodGroups section above).


###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Pod startup SLI/SLO may be affected and should be adjusted appropriately.
The reason is that scheduling a pod being part of a gang will now be blocked on all pods
from a gang to be created and observed by the scheduler (which from large gangs can take
non-negligible amount of time).


###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The increase of CPU/MEM consumption of kube-apiserver and kube-scheduler should be negligible
percentage of the current resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

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

## Drawbacks

There are already multiple implementations of gang scheduling in the ecosystem.

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

The longer version of this design describing the whole thought process of choosing the
above described approach can be found in the [extended proposal] document.

[extended proposal]: https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?


## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

[^1]: The Kubernetes community uses the term "gang scheduling" to mean "all-or-nothing scheduling of a set of pods" [1,2,3,4,5,6,7,8,9,10,11,12,13]. In the Kubernetes context, it does not imply time-multiplexing (in contrast to prior academic work such as [Feitelson and Rudolph](https://doi.org/10.1016/0743-7315(92)90014-E), and in contrast to [Slurm Gang Scheduling](https://slurm.schedmd.com/gang_scheduling.html)).  

[^2]: [API Design for Gang and Workload-Aware Scheduling](https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?pli=1&tab=t.0)

[^3]: Volcano.sh, Co-scheduling plugin, Preferred Networks Plugin, and Kueue all implement gang scheduling outside of kube-scheduler.  Additionally, two previous proposals have been made on this KEP's issue.  These alternatives are compared in detail in the [Background tab of the API Design for Gang Scheduling](https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?pli=1&tab=t.3zjbiyx2yldg).

