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
    - [NominatedNodeName impact on filtering performance](#nominatednodename-impact-on-filtering-performance)
- [Design Details](#design-details)
  - [Naming](#naming)
  - [Associating Pod into PodGroups](#associating-pod-into-podgroups)
  - [API](#api)
    - [Basic Policy Extension](#basic-policy-extension)
  - [Scheduler Changes](#scheduler-changes)
    - [North Star Vision](#north-star-vision)
    - [GangScheduling Plugin](#gangscheduling-plugin)
    - [Future plans](#future-plans)
  - [Scheduler Changes for v1.36](#scheduler-changes-for-v136)
    - [The Workload Scheduling Cycle](#the-workload-scheduling-cycle)
    - [Queuing and Ordering](#queuing-and-ordering)
    - [Scheduling Algorithm](#scheduling-algorithm)
    - [Algorithm Limitations](#algorithm-limitations)
    - [Interaction with Basic Policy](#interaction-with-basic-policy)
    - [Delayed Preemption](#delayed-preemption)
    - [Workload-aware Preemption](#workload-aware-preemption)
    - [Failure Handling](#failure-handling)
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
  - [API](#api-1)
  - [Pod group queueing in scheduler](#pod-group-queueing-in-scheduler)
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
- Implement first version of gang-scheduling in kube-scheduler supporting (potentially in non-optimal way)
  all existing scheduling features.
- Provide full backward compatibility for all existing scheduling features

### Non-Goals

- Take away responsibility to create pods from controllers.
- Bring fairness or multiple workload queues in kube-scheduler. Kueue and Volcano.sh will continue to provide this.
- Map all the declarative state and behaviors into `Workload` object. It is focused only on scheduling-related parts.

The following are non-goals for this KEP but will probably soon appear to be goals for follow-up KEPs:

- Integrate cluster autoscaling with gang scheduling.
- Introduce a concept of `Reservation` that can be later consumed by pods.
- Workload-level preemption.
- Address resource contention between different schedulers (including possible deadlocks).
- Address the problem of premature preemptions in case the higher priority workloads does not
  eventually schedule.

See [Future plans](#future-plans) for more details.

## Proposal

The `spec.workloadRef` field will be added to the Pod resource.  A sample pod with this new field looks like this:
```yaml
apiVersion: v1
kind: Pod
spec:
  ...
  workloadRef:
    name: job-1
    podGroup: pg1
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
      workloadRef:
        name: job-1
        podGroup: pg1
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
apiVersion: scheduling.k8s.io/v1alpha1
kind: Workload
metadata:
  namespace: ns-1
  name: job-1
spec:
  podGroups:
    - name: "pg1"
      policy:
        gang:
          minCount: 100
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

#### NominatedNodeName impact on filtering performance

Using `.status.nominatedNodeName` as an output of the Workload Scheduling Cycle
can impact the performance of the standard pod-by-pod scheduling cycle.
Whenever the scheduler filters a node, it must temporarily add nominated pods
(with equal or higher priority) to the cached NodeInfo. In large clusters,
the number of such operations multiplied by the scheduling throughput can yield to a visible overhead.
If the latency between the end of the Workload Scheduling Cycle
and the actual processing of those pods is high, the number of unrelated pods
having to consider such nomination also increases.

However, this impact is mitigated by several factors:
* Nominations are temporary. As soon as workload-scheduled pods pass
  their individual scheduling cycle and are assumed, what cleans the in-memory nominations.
* For the workload pods themselves, the performance impact is negligible.
  They will typically only execute filters for the single node they are nominated to,
  rather than evaluating the entire cluster.
* These pods are expected to be retried quickly after the Workload Scheduling Cycle because
  their initial timestamps are preserved. This places them near the head of the active queue,
  minimizing the duration they remain in the "nominated but not assumed" state.
* While higher-priority or long-standing (equal priority) pods might interleave and be scheduled before the gang pods,
  the overall window of time where these nominations are active is expected to be short enough
  to prevent severe degradation.

The real impact will be verified through scalability tests (scheduler-perf benchmark).

## Design Details

### Naming

* `Workload` is the resource Kind.
* `scheduling.k8s.io` is the ApiGroup.
* `spec.workloadRef` is the name of the new field in pod.
* Within a Workload there is a list of groups of pods. Each group represents a top-level division of pods within a Workload.  Each group can be independently gang scheduled (or not use gang scheduling). This group is named `PodGroup`.
* In a future , we expect that this group can optionally specify further subdivision into sub groups.  Each sub-group can have an index.  The indexes go from 0 to N, without repeats or gaps. These subgroups are called `PodSubGroup`.
* In subsequent KEPs, we expect that a sub-group can optionally specify further subdivision into pod equivalence classes.  All pods in a pod equivalence class have the same values for all fields that affect scheduling feasibility.  These pod equivalence classes are called `PodSet`.

### Associating Pod into PodGroups

When a `Workload` consists of a single group of pods needing Gang Scheduling, it is clear which pods belong to the group from the `spec.workloadRef.name` field of the pod.  However `Workload` supports listing multiple list items, and a list item can represent a single group, or a set of identical replica groups.
In these cases, there needs to be additional information to indicate which group a pod belongs to.

We proposed to extend the newly introduced `pod.spec.workloadRef` field with additional information
to include that information. More specifically, the `pod.spec.workloadRef` field is of type `WorkloadReference`
and is defined as following:

```go
type PodSpec struct {
	...
	// WorkloadRef provides a reference to the Workload object that this Pod belongs to.
	// This field is used by the scheduler to identify the PodGroup and apply the
	// correct group scheduling policies. The Workload object referenced
	// by this field may not exist at the time the Pod is created.
	// This field is immutable, but a Workload object with the same name
	// may be recreated with different policies. Doing this during pod scheduling
	// may result in the placement not conforming to the expected policies.
	//
	// +featureGate=GenericWorkload
	// +optional
	WorkloadRef *WorkloadReference
}

// WorkloadReference identifies the Workload object and PodGroup membership
// that a Pod belongs to. The scheduler uses this information to apply
// workload-aware scheduling semantics.
type WorkloadReference struct {
	// Name defines the name of the Workload object this Pod belongs to.
	// Workload must be in the same namespace as the Pod.
	// If it doesn't match any existing Workload, the Pod will remain unschedulable
	// until a Workload object is created and observed by the kube-scheduler.
	// It must be a DNS subdomain.
	//
	// +required
	Name string

	// PodGroup is the name of the PodGroup within the Workload that this Pod
	// belongs to. If it doesn't match any existing PodGroup within the Workload,
	// the Pod will remain unschedulable until the Workload object is recreated
	// and observed by the kube-scheduler. It must be a DNS label.
	//
	// +required
	PodGroup string

	// PodGroupReplicaKey specifies the replica key of the PodGroup to which this
	// Pod belongs. It is used to distinguish pods belonging to different replicas
	// of the same pod group. The pod group policy is applied separately to each replica.
	// When set, it must be a DNS label.
	//
	// +optional
	PodGroupReplicaKey string
}
```

At least for Alpha, we start with `WorkloadReference` to be immutable field in the Pod.
In further phases, we may decide to relax validation and allow for setting some of the fields later.
Moreover, the visibility into issues (debuggability) will depend on [#5501], but we don't
treat it as a blocker.

[#5501]: https://github.com/kubernetes/enhancements/pull/5501

The example below shows how this could look like for with the following `Workload` object:

```yaml
apiVersion: scheduling.k8s.io/v1alpha1
kind: Workload
metadata:
  name: jobset
spec:
  podGroups:
    - name: "job-1"
      policy:
        gang:
          minCount: 100
```

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: jobset-job-1-abc123
spec:
  ...
  workloadRef:
    name: jobset
    podGroup: job-1
    podGroupReplicaKey: key-2
  ...
```

We decided for this option because it is more succinct and makes the role of a pod clear just
from inspecting the pod (and simple/efficient to group).
We acknowledge the fact that this option may require additional minor changes in the controllers
to adopt this pattern (e.g. for LeaderWorkerSet we will need to populate the pod template
similarly that we currently populate the labels).

The primary alternative we consider was to introduce the `PodGroupSelector` on each `PodGroup`
to identify pods belonging to it. However, with this pattern:
- there are additional corner cases (e.g. a pod links to a workload but none of its PodGroups match
  that pod)
- for replicated gang, we can't use the full label selector, but rather support specifying only the
  label key, similar to `MatchLabelKeys` in pod affinity


### API

The `Workload` type will be defined with the following structure:
```go
// Workload allows for expressing scheduling constraints that should be used
// when managing lifecycle of workloads from scheduling perspective,
// including scheduling, preemption, eviction and other phases.
type Workload struct {
	metav1.TypeMeta
	// Standard object's metadata.
	// Name must be a DNS subdomain.
	//
	// +optional
	metav1.ObjectMeta

	// Spec defines the desired behavior of a Workload.
	//
	// +required
	Spec WorkloadSpec
}

// WorkloadMaxPodGroups is the maximum number of pod groups per Workload.
const WorkloadMaxPodGroups = 8

// WorkloadSpec defines the desired state of a Workload.
type WorkloadSpec struct {
	// ControllerRef is an optional reference to the controlling object, such as a
	// Deployment or Job. This field is intended for use by tools like CLIs
	// to provide a link back to the original workload definition.
	// When set, it cannot be changed.
	//
	// +optional
	ControllerRef *TypedLocalObjectReference

	// PodGroups is the list of pod groups that make up the Workload.
	// The maximum number of pod groups is 8. This field is immutable.
	//
	// +required
	// +listType=map
	// +listMapKey=name
	PodGroups []PodGroup
}

// TypedLocalObjectReference allows to reference typed object inside the same namespace.
type TypedLocalObjectReference struct {
	// APIGroup is the group for the resource being referenced.
	// If APIGroup is empty, the specified Kind must be in the core API group.
	// For any other third-party types, setting APIGroup is required.
	// It must be a DNS subdomain.
	//
	// +optional
	APIGroup string
	// Kind is the type of resource being referenced.
	// It must be a path segment name.
	//
	// +required
	Kind string
	// Name is the name of resource being referenced.
	// It must be a path segment name.
	//
	// +required
	Name string
}

// PodGroup represents a set of pods with a common scheduling policy.
type PodGroup struct {
	// Name is a unique identifier for the PodGroup within the Workload.
	// It must be a DNS label. This field is immutable.
	//
	// +required
	Name string

	// Policy defines the scheduling policy for this PodGroup.
	//
	// +required
	Policy PodGroupPolicy
}

// PodGroupPolicy defines the scheduling configuration for a PodGroup.
type PodGroupPolicy struct {
	// Basic specifies that the pods in this group should be scheduled using
	// standard Kubernetes scheduling behavior.
	//
	// +optional
	// +oneOf=PolicySelection
	Basic *BasicSchedulingPolicy

	// Gang specifies that the pods in this group should be scheduled using
	// all-or-nothing semantics.
	//
	// +optional
	// +oneOf=PolicySelection
	Gang *GangSchedulingPolicy
}

// BasicSchedulingPolicy indicates that standard Kubernetes
// scheduling behavior should be used.
type BasicSchedulingPolicy struct {
	// This is intentionally empty. Its presence indicates that the basic
	// scheduling policy should be applied. In the future, new fields may appear,
	// describing such constraints on a pod group level without "all or nothing"
	// (gang) scheduling.
}

// GangSchedulingPolicy defines the parameters for gang scheduling.
type GangSchedulingPolicy struct {
	// MinCount is the minimum number of pods that must be schedulable or scheduled
	// at the same time for the scheduler to admit the entire group.
	// It must be a positive integer.
	//
	// +required
	MinCount int32
}
```

The individual `PodGroups` and `PodGroup` replicas are treated as independent gangs. As an example, if one of
the groups can be scheduled and the other can't be - this is exactly what will happen. If the underlying
user intention was to have either both of them or none of them running, they should form a single group and
not be split into two. A `LeaderWorkerSet` is a good example of it, where a single `PodGroup` replica consists
of a single leader and `N` workers and that forms a scheduling (and runtime unit), but workload as a whole
may consist of a number of such replicas.

#### Basic Policy Extension

While Gang Scheduling focuses on atomic, all-or-nothing scheduling, there is a significant class
of workloads that requires best-effort optimization without the strict blocking semantics of a gang.

In the first alpha version of the Workload API, the `Basic` policy was a no-op.
We propose extending the `Basic` policy to accept a `desiredCount` field.
This feature will be gated behind a separate
feature gate (`WorkloadBasicPolicyDesiredCount`) to decouple it from the core Gang Scheduling graduation path.

```go
// BasicSchedulingPolicy indicates that standard Kubernetes
// scheduling behavior should be used.
type BasicSchedulingPolicy struct {
	// DesiredCount is the expected number of pods that will belong to this
	// PodGroup. This field is a hint to the scheduler to help it make better
	// placement decisions for the group as a whole.
	//
	// Unlike gang's minCount, this field does not block scheduling. If the number
	// of available pods is less than desiredCount, the scheduler can still attempt
	// to schedule the available pods, but will optimistically try to select a
	// placement that can accommodate the future pods.
	//
	// +optional
	DesiredCount *int32
}
```

This field allows users to express their "true" workloads more easily
and enables the scheduler to optimize the placement of such pod groups by taking the desired state
into account. Ideally, the scheduler should prefer placements that can accommodate
the full `desiredCount`, even if not all pods are created yet.
When `desiredCount` is specified, the scheduler can delay scheduling the first Pod it sees
for a short amount of time in order to wait for more Pods to be observed.

### Scheduler Changes

The kube-scheduler will be watching for `Workload` objects (using informers) and will use them to map pods
to and from their `Workload` objects.

In the initial implementation, we expect users to create the `Workload` objects. In the next steps controllers
will be updated to create an appropriate `Workload` objects themselves whenever they can appropriately infer
the intention from the desired state.
Note that given scheduling options are stored in the `Workload` object, pods linked to the `Workload`
object will not be scheduled until this `Workload` object is created and observed by the kube-scheduler.

#### North Star Vision

The north star vision for gang scheduling implementation should satisfy the following requirements:

1. Ensure that pods being part of a gang are not bound if all pods belonging to it can't be scheduled.
2. Provide the "optimal enough" placement by considering all pods from a gang together.
3. Avoid deadlock and livelock scenario when multiple workloads are being scheduled at the same time by kube-scheduler.
4. Avoid deadlock and livelock scenario when multiple workloads are being scheduled at the same time by different
   schedulers.
5. Avoid premature preemptions of already running pods in case a higher priority gang will be rejected.
6. Support gang-level (or workload-level in general) level preemption (if pods form a gang also
   from a runtime perspective, they can't be preempted individually).
7. Updating workload status and triggering rescheduling when a gang failed binding in the all-or-nothing
   fashion.
8. Support gang-scheduling even if part of the infrastructure needs to be provisioned (by Cluster
   Autoscaler, Karpenter or other solutions).

Addressing all these requirements in a single shot would be a huge change, so as part ot this KEP we
will only focus on a subset of those. However, we very briefly sketch the path towards the vision to
ensure that this KEP is moving in the right direction.

#### GangScheduling Plugin

For `Alpha`, we are focusing on introducing the concept of the `Workload` and plumbing it into
kube-scheduler in the simplest possible way. We will implement a new plugin implementing the following
hooks:
- PreEnqueue - used as a barrier to wait for the `Workload` object and all the necessary pods to be
  observed by the scheduler before even considering them for actual scheduling
- WaitOnPermit - used as a barrier to wait for the pods to be assigned to the nodes before initiating
  potential preemptions and their bindings

This seems to be the simplest possible implementation to address the requirement (1). We are consciously
ignoring the rest of the requirements for `Alpha` phase.

#### Future plans

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

Addressing requirement (8) is the biggest effort as it requires much closer integration between
scheduler and autoscaling components. So in the initial steps we will only focus on mitigating
this problem with existing mechanisms (e.g. reserving resources via NominatedNodeName).

However, approval for this KEP is NOT an approval for this vision. We only sketch it to show that
we see a viable path forward from the proposed design that will not require significant rework.

### Scheduler Changes for v1.36

For the `Alpha` phase in v1.35, we focused on plumbing the `Workload` API and implementing
the `GangScheduling` plugin using simple barriers (`PreEnqueue` and `Permit`).
While this satisfied the correctness requirement for "all-or-nothing" scheduling,
it did not address performance or efficiency at scale, scheduling livelocks,
nor did it solve the problem of partial preemption application.

For v1.36, we propose introducing a **Workload Scheduling Cycle**.
This mechanism processes all Pods belonging to a single `PodGroup` in one batch,
rather than attempting to schedule them individually in isolation using the
traditional pod-by-pod approach. While introduction of this phase itself won't
fully address the "optimal enough" part of requirement (2),
it provides the necessary foundation for applying workload scheduling algorithms
to process the entire gang together.
The single scheduling cycle, together with blocking resources using nomination,
will address requirement (3).

We will also introduce delayed preemption (described in [KEP-5710](https://kep.k8s.io/5711)).
Together with the introduction of a dedicated Workload Scheduling Cycle,
this will address requirement (5).

#### The Workload Scheduling Cycle

We introduce a new phase in the main scheduling loop (`scheduleOne`). In the
end-to-end Pod scheduling flow, it is planned to place this new phase *before*
the standard pod-by-pod scheduling cycle. When the loop pops a `PodGroup` from
the active queue, it initiates the Workload Scheduling Cycle.

Since the `PodGroup` instance (defined by the group name and replica key)
is the effective scheduling unit, the Workload Scheduling Cycle will operate
at the `PodGroup` instance level, i.e., each instance will be scheduled separately
in its own cycle.

If new Pods belonging to an already scheduled `PodGroup` instance
(i.e., one that already passed `WaitOnPemit`) appear,
they are also processed via the Workload Scheduling Cycle, which takes the previously
scheduled Pods into consideration. This is done for safety reasons to ensure
the PodGroup-level constraints are still satisfied. However, if the `PodGroup` is being processed,
these new Pods must wait for the ongoing pod group scheduling to be finished (pass `WaitOnPermit`),
before being considered.

The cycle proceeds as follows:

1. The scheduler takes pod group from the scheduling queue.
   If the pod group is unscheduled (even partially), it temporarily removes
   all group's pods from the queue for processing. The order of processing
   is determined by the queueing mechanism (see *Queuing and Ordering* below).
   
2. A single cluster state snapshot is taken for the entire group operation
   to ensure consistency during the cycle.

3. The scheduler runs a specialized algorithm (detailed below)
   to find placements for the group.

4. Outcome:
   * If the group (i.e., at least `minCount` Pods) can be placed,
     these Pods have the `.status.nominatedNodeName` set.
     They are then effectively "reserved" on those nodes in the
     scheduler's internal cache. Pods are then pushed to the
     active queue (restoring their original timestamps to ensure fairness)
     to pass through the standard scheduling and binding cycle,
     which will consider and follow the nomination.
   * If `minCount` cannot be met (even after calculating potential
     preemptions), the scheduler considers the `PodGroup` unschedulable. Standard backoff
     logic applies (see *Failure Handling*), and Pods are returned to
     the scheduling queue.

#### Queuing and Ordering

Workload-aware preemption (an `Alpha` effort in [KEP-5710](https://github.com/kubernetes/enhancements/pull/5711))
will introduce a specific scheduling priority for a workload.
Having that in mind, it is beneficial to design a queueing mechanism open
for taking a workload's scheduling priority into account.
However, as we need to support ordering before that feature can be enabled,
we also need to derive the priority from the pod group's pods.
One such formula can be to set it to the lowest priority found within the pod group,
what will be effectively the weakest link to determine if the whole pod group is schedulable
and reduce unnecessary preemption attempts.

To ensure that we process the `PodGroup` instance at an appropriate time and
don't starve other pods (including gang pods in the pod-by-pod scheduling phase)
from being scheduled, we need to have a good queueing mechanism for pod groups.

We have decided to make the scheduling queue explicitly workload-aware.
The queue will support queuing `PodGroup` instances alongside individual Pods.

1. When Pods belonging to a `PodGroup` are added to the scheduler and pass the `PreEnqueue`,
   they are initially stored in a dedicated internal data structure (tentatively named `workloadPods`)
   rather than the standard active queue.

2. Once the number of accumulated Pods meets the scheduling requirements (e.g., `minCount`),
   a `QueuedPodGroupInfo` object (analogous to `QueuedPodInfo`) is created
   and injected into the main scheduling queue.

3. The `scheduleOne` loop will pop the highest-priority item from the queue,
   which may now be either a single Pod (triggering the standard cycle)
   or a `PodGroup` (triggering the Workload Scheduling Cycle).

4. During a Workload Scheduling Cycle, all member Pods are retrieved from `workloadPods`.
   Based on the cycle's outcome:
   * **Success:** Pods are moved to the standard `activeQ` (with nominations set)
     to proceed to the pod-by-pod scheduling soon.
   * **Failure/Preemption:** Pods are returned to `workloadPods` or the unschedulable queue.
     The `PodGroup` enters a backoff state and is eligible for retry only when
     a relevant cluster event wakes up at least one of its member pods.

While this represents a significant architectural change to the scheduling
queue and `scheduleOne` loop, it provides a clean separation of concerns and
establishes a necessary foundation for future Workload Aware Scheduling features.

#### Scheduling Algorithm

*Note: The algorithm described below is a simplified default version based on baseline scheduling logic.
It is expected to evolve to more effectively handle complex scenarios and specific features
in future iterations.*

The internal algorithm for placing the group utilizes the optimization defined
in *Opportunistic Batching* ([KEP-5598](https://kep.k8s.io/5598)) for improved performance.
The approach described below allows mitigating some restrictions of that feature, e.g.,
by sorting the Pods appropriately by their signatures. In case Opportunistic Batching
is disabled or not applicable, this falls back to non-optimized filtering and scoring for each Pod.
The list and configuration of plugins used by this algorithm will be the same as in the pod-by-pod cycle.

1. The scheduler iterates through the retrieved Pods and groups
   them into homogeneous sub-groups (using the signatures defined in
   [KEP-5598](https://kep.k8s.io/5598)).
   *This aggregation can be done in the scheduler's cache earlier to optimize performance.*

2. These sub-groups are sorted. Initially, we sort by the highest priority
   of the sub-group (assuming homogeneity enforces uniform sub-group priority).
   In the future, sorting may use the size of the sub-group (larger groups first) to
   tackle the hardest placement problems early. Crucially, the ordering should be deterministic
   and saable if the pod group state doesn't change
   *This sorting can be done in the scheduler's cache earlier to optimize performance.*

3. The scheduler iterates through the sorted sub-groups. It finds a feasible node
   for each pod from a sub-group using standard filtering and scoring phases.
   It also utilizes the Opportunistic Batching feature where possible,
   reducing overall scheduling time.

   * If a pod fits, it is temporarily assumed and reserved on the selected node.
  
   * If a pod cannot fit, the scheduler tries preemption by running
     the `PostFilter` extension point.
     *Note: With workload-aware preemption this phase will be replaced by a workload-level algorithm
     that will be run after trying to schedule all pod group's pods.*

     * If calculated preemption is successful, the pod is temporarily assumed and reserved on the selected node.
       Victim pods are not preempted yet, but just marked as nominated for removal.
       Subsequent pods from this group won't see victims on the nodes in this workload cycle.
       [Delayed Preemption](#delayed-preemption) feature is used to delay the actuation
       until after all group's pods are considered.

     * If preemption fails, the pod is considered unscheduled for this cycle.
       However, the scheduling of subsequent pods continues as long as
       the `minCount` constraint remains satisfiable. The processing can also be
       optimized by rejecting all subsequent pods from the same
       homogeneous sub-group, as their failed scheduling outcome will be the same.

   The phase can effectively stop once `minCount` pods have a placement,
   though attempting to schedule the full group is preferred to maximize utilization.

4. The scheduler checks if the number of schedulable (including those after delayed preemption)
   Pods meets the `minCount`.

   * If `schedulableCount >= minCount`, the cycle succeeds.
  
     * If preemptions are needed: The removal of all nominated victims is actuated
       as described in [Delayed Preemption](#delayed-preemption).
       The pods are nominated to their chosen nodes but are moved to the unschedulable queue,
       waiting for victim removal to complete. They can be moved back to the active queue
       and retried even before the victims are fully removed, but they must pass through
       the Workload Scheduling Cycle again. Crucially, initiating *new* preemptions
       will be forbidden during this retry. This ensures that the pod group
       can be scheduled in a different location if resources become available earlier,
       but cannot cause additional disruption to do so.

     * If preemptions are not needed: Pods are nominated to their chosen nodes,
       pushed directly to the active queue, and will soon attempt to be scheduled
       on their nominated nodes in their own, pod-by-pod cycles.

     Pod will be restricted to its nominated node during the individual cycle.
     If the node is unavailable, the pod will remain unschedulable and the `WaitOnPermit` gate will take that
     into consideration. The `minCount` check can consider the number of pods that have passed
     the Workload Scheduling Cycle to ensure that Pods are not waiting unnecessarily when some have been rejected
     but other new pods have been added to the cluster.

     In the pod-by-pod cycle, preemption initiated by the workload pods will be forbidden.
     Allowing it would complicate reasoning about the consistency of the
     Workload Scheduling Cycle and Workload-Aware Preemption. If preemption is necessary
     (e.g., the nominated node is no longer valid), the gang will either time out
     or be instantly rejected (when the `minCount` cannot be satisfied) at `WaitOnPermit` and all necessary preemptions
     will be simulated again in the next Workload Scheduling Cycle.

   * If `schedulableCount < minCount`, the cycle fails. Preemptions computed but not actuated
     during this cycle are discarded. Pods go through traditional failure handlers
     and nominations for them are cleared to ensure the other workloads (pod groups)
     can be attempted on that place. See *Failure Handling*.

   Gang Scheduling is currently implemented as a plugin, meaning the `minCount` constraint
   is enforced at the plugin level. However, the proposed Workload Scheduling Cycle algorithm
   needs to know if this constraint is met to decide whether to commit the results.
   To verify this, a new extension point will be introduced, allowing plugins to validate the group's
   scheduled pods. This will function similarly to a `Permit` check (likely requiring `Reserve` state)
   but without the suspension (`WaitOnPermit`) gate. Crucially, this extension should support two checks:

   * Validation: Check whether the currently scheduled pods meet the requirements,
     e.g., if the `minCount` pods from a pod group was successfully scheduled.

   * Feasibility: Given the number of pods that have already failed scheduling in this cycle,
     check whether is it still *possible* to meet the constraint. If not, the cycle should abort early
     to save time.
  
While this algorithm might be suboptimal, it is a solid first step for ensuring we have
a single-cycle workload scheduling phase. As long as PodGroups consist of homogeneous pods,
opportunistic batching itself will provide significant improvements.
Future features like Topology Aware Scheduling can further improve other subsets of use cases.

#### Algorithm Limitations

Default algorithm proposed above relies on specific sorting and may fail to find
a valid placement that could have been discovered by processing the group's pods
in a different order. While resolving this limitation could be desirable,
implementing a generalized solver for arbitrary constraints would introduce excessive complexity
for the default implementation. The current proposal addresses the vast majority of standard use cases
(specifically homogeneous workloads). Future improvements for this should be delivered
via specialized algorithms based on specific pod group constraints,
such as Topology Aware Scheduling (TAS).

Since the scheduler cannot exhaustively analyze all possible placement permutations,
we will advise users via documentation regarding which pod group types
are well-supported and which scenarios are handled on a
best-effort basis (where a successful placement is not guaranteed, even if
one theoretically exists).

In particular:
* For basic **homogeneous** pod groups without inter-pod dependencies, this
  algorithm is expected to find a placement whenever one exists.
* For **heterogeneous** pod groups, finding a valid placement is not guaranteed.
* For pod groups with **inter-pod dependencies** (e.g., affinity/anti-affinity
  or topology spreading rules), finding a valid placement is not guaranteed.

Moreover, if a pod using these features is rejected by the Workload Scheduling Cycle,
its rejection message (exposed via Pod status) will explicitly indicate
that the rejection may be due to the use of features for which finding an existing
placement cannot be guaranteed, distinguishing it from a generic `Unschedulable` reason.

#### Interaction with Basic Policy

For pod groups using the `Basic` policy, the Workload Scheduling Cycle is
optional. In the v1.36 timeframe, this cycle will be applied to
`Basic` pod groups to leverage the batching performance benefits, but the
"all-or-nothing" (`minCount`) checks will be skipped; i.e., we will try to
schedule as many pods from such PodGroup as possible.

If the `Basic` policy has `desiredCount` set, the Workload Scheduling Cycle
may utilize this value to simulate the full group size during feasibility checks.
Note that the implementation of this specific logic might follow in a Beta stage
of this API field.

#### Delayed Preemption

A critical requirement for moving Gang Scheduling to Beta is the integration with *Delayed Preemption*,
which allows the scheduler to avoid unnecessary preemptions. However, the current model of preemption,
when preemption is triggered immediately
after the victims are decided (in `PostFilter`), doesn't achieve this goal. The reason for that is
that the proposed placement (nomination) can actually appear to be invalid and not proceed.
In such cases, we will not even proceed to binding and the preemption will be completely unnecessary
disruption.

Note that this problem already exists in the current gang scheduling implementation. A given gang may
not proceed with binding if the `minCount` pods from it can't be scheduled. But, the preemptions are
currently triggered immediately after choosing a place for individual pods. So similarly as above,
we may end up with completely unnecessary disruptions.

We will address it with what we call *delayed preemption* mechanism as following:

1. We will modify the `DefaultPreemption` plugin to just compute preemptions, without actuating them.
   We advise maintainers of custom `PostFilter` implementations to do the same.

2. We will extend the `PostFilterResult` to include a set of victims (in addition to the existing
   `NominationInfo`). This will allow us to clearly decouple the computation from actuation.

   We believe that while custom plugins may want to provide their custom preemption logic,
   the actuation logic can actually be standardized and implemented directly as part of the framework.
   If that proves incorrect, we will introduce a new plugin extension point (tentatively called
   `Preempt`) that will be responsible for actuation. However, for now we don't see evidence for this
   being needed.

3. For individual pods (not being part of a workload), we will adjust the scheduling framework
   implementation of `schedulingCycle` to actuate preemptions of returned victims if calling
   `PostFilter` plugins resulted in finding a feasible placement.

4. For pods being part of a workload, we will rely on the Workload Scheduling Cycle.
   We still have two subcases here:

   1. In the legacy case (without workload-aware preemption), we call `PostFilter` individually for
      every pod from a PodGroup. However, the victims computed for already the already processed
      pods may affect placement decisions for the next pods.
      To accommodate for that, if a set of victims was returned from a `PostFilter` in addition
      to keeping them for further actuation, we will additionally store them in `CycleState`.
      More precisely, the `CycleState` will store a new entry containing a map from
      a `nodeName` to a list of victims that were already chosen.
      With that, the `DefaultPreemption` plugin will be extended to remove all already chosen
      victims from a given node before processing that node.

   2. In the target case (with workload-aware preemption), we will have no longer be processing
      pods individually, so the additional mutations of `CycleState` should not be needed.

5. In both above cases, we will introduce an additional step to the scheduling algorithm at the
   end. If we managed to find a feasible placement for the PodGroup, we will simply take all
   the victims and actuate their preemption. If a feasible placement was not found, the victims
   will be dropped. In both cases, the scheduling of the whole PodGroup (all its pods)
   will be marked as unschedulable and got back to the scheduling queue.

6. To reduce the number of unnessary preemptions, in case a preemption has already been triggerred
   and the already nominated placement remains valid, no new preemptions can be triggerred.
   In other words, a different placement can be chosen in a subsequent (workload) scheduling cycles only if
   it doesn't require additional preemptions or the previously chosen placement is no longer
   feasible (e.g. because higher priority pods were scheduled in the meantime).
   This can be done by ignoring the pods with `deletionTimestamp` set in these preemption attempts
   (when the previous preemption is ongoing for the preemptor).

The rationale behind the above design is to maintain the current scheduling property where preemption
doesn't result in a commitment for a particular placement. If a different possible placement appears
in the meantime (e.g. due to other pods terminating or new nodes appearing), subsequent scheduling
attempts may pick it up, improving the end-to-end scheduling latency. Returning pods to scheduling
queue if these need to wait for preemption to become schedulable maintains that property.

We acknowledge the two limitations of the above approach: (a) dependency on the introduction of
Workload Scheduling Cycle (delayed preemption will not work if workload pods will not be processed
by Workload Scheduling Cycle) and (b) the fact that the placement computed in
Workload Scheduling Cycle may be invalidated in pod-by-pod scheduling later.
However, those features should be used together,
and the simplicity of the approach and target architecture outweigh these limitations.

#### Workload-aware Preemption

Workload-aware preemption ([KEP-5710](https://kep.k8s.io/5710)) aims to
enable preemption for a whole pod group at once. In the context of this cycle,
it means that if the cycle determines preemption for a single pod is necessary,
it won't run the `PostFilter` phase, but defer that to the end of the workload scheduling phase,
running a new, single workload-aware preemption step.

Read more about the proposal in
[KEP-5710: Workload Aware Preemption](https://github.com/kubernetes/enhancements/pull/5711) PR.

#### Failure Handling

If a Workload Scheduling Cycle fails (e.g., `minCount` is not met, preemption fails,
or a timeout occurs), the scheduler must handle the failure efficiently.

1. Rejection

When the cycle fails, the scheduler rejects the entire group.
* All Pods in the group are moved back to the scheduling queue.
  Their status is updated the event with failure reason is sent.
* Crucially, any `.status.nominatedNodeName` entries set during the failed attempt
  (or from previous cycles) must be cleared. This ensures that the resources
  tentatively reserved for this gang are immediately released for other workloads.

2. Backoff strategy

Backoff mechanism has to be applied for a pod group similarly as we do for individual pods.
Initially, we will apply the standard Pod backoff logic to the group.

At the same time, we should consider increasing the maximum backoff duration for pod groups
or potentially scaling it based on the number of pods within the group.
The current default of 10 seconds has proven insufficient in large clusters,
so this might be the case for workloads. Crucially, because the Workload Scheduling Cycle
can be computationally expensive, retrying it too frequently risks starving individual pods.
Moreover, retries triggered by the Delayed Preemption feature may further strengthen the problem.

3. Retries

We rely on the existing Queueing Hints mechanism to determine when to retry the gang.
It is considered for a retry when *at least one* member Pod receives a `Queue` hint
(indicating a relevant cluster event, such as a Node addition or Pod deletion,
has made that specific Pod potentially schedulable).

While checking a single Pod does not guarantee the *whole* gang can fit,
calculating gang-level schedulability inside the event handler can be difficult at the moment.
Therefore, we optimistically retry the Workload Scheduling Cycle if any member's condition improves.

It might be beneficial to retry the pod group without being triggered by any cluster event,
because single Workload Scheduling Cycle cannot determine the placement doesn't really exists,
especially for heterogeneous workloads or inter-pod dependencies.
To avoid introducing subtle errors in the initial implementation,
we can start by skipping the Queueing Hints mechanism and relying solely on the backoff time.


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

- `k8s.io/kubernetes/pkg/apis/scheduling/v1alpha1`: `2025-10-02` - 62.7%
- `k8s.io/kubernetes/pkg/apis/scheduling/validation`: `2025-10-02` - 97.8%
- `k8s.io/kubernetes/pkg/scheduler`: `2025-10-02` - 81.7%
- `k8s.io/kubernetes/pkg/scheduler/backend/queue`: `2025-10-02` - 91.4%
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

- [test name](https://github.com/kubernetes/kubernetes/blob/2334b8469e1983c525c0c6382125710093a25883/test/integration/...): [integration master](https://testgrid.k8s.io/sig-release-master-blocking#integration-master?include-filter-by-regex=MyCoolFeature), [triage search](https://storage.googleapis.com/k8s-triage/index.html?test=MyCoolFeature)
-->

Initially, we created integration tests to ensure the basic functionalities of gang scheduling including:

- Pods linked to the non-existing workload are not scheduled
- Pods get unblocked when workload is created and observed by scheduler
- Pods are not scheduled if there is no space for the whole gang
  
With Workload Scheduling Cycle and Delayed Preemption features, we will significantly expand test coverage to verify:

- Pods referencing a `Workload` (both gang and basic policies) are correctly processed via the Workload Scheduling Cycle.
- `PodGroup` queuing ensures that all available members are retrieved and processed correctly.
- Deadlocks and livelocks do not occur when multiple gangs compete for resources or interleave with standard pods.
- Delayed Preemption works correctly for pod-by-pod (non-workload) scheduling.
- Delayed Preemption ensures atomicity, i.e., victims are deleted only if the scheduler determines the entire gang can fit,
  otherwise, the cycle aborts with zero disruption.
- Failed pod groups are requeued correctly and retry successfully when resources become available.

We will also benchmark the performance impact of these changes to measure:

- The scheduling throughput of the workload scheduling, including gang and basic policies and preemptions.
- The performance impact on standard pod scheduling when there are many nominated pods,
  for scenarios mentioned in the [NominatedNodeName impact on filtering performance](#nominatednodename-impact-on-filtering-performance).

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

We will add basic API tests for the the new `Workload` API, that will later be
promoted to the conformance.

### Graduation Criteria

#### Alpha

- Workload API is introduced behind GenericWorkload feature flag
- API tests for Workload API (that will be promoted to conformance in GA release)
- kube-scheduler implements first version of gang-scheduling based on groups defined in the Workload object

#### Beta

- Providing "optimal enough" placement by considering all pods from a gang together
- Avoiding livelock scenario when multiple workloads are being scheduled at the same time
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
  explicitly specifying their new `spec.workloadRef` field
- scheduler automatically uses the new extensions and tries to schedule all pods from a given
  gang in a scheduling group based on the defined `Workload` objects

When user downgrades the cluster to the version that no longer supports these two features:
- the `Workload` objects can no longer be created (the existing ones are not removed though)
- the `spec.workloadRef` field can no longer be set on the Pods (the already set fields continue
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
  - Feature gate name: GenericWorkload (alternatives: NativeWorkload/Workload)
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-scheduler
  - Feature gate name: GangScheduling
  - Components depending on the feature gate:
    - kube-scheduler
  - Feature gate name: DelayedPreemption
  - Components depending on the feature gate:
    - kube-scheduler
  - Feature gate name: WorkloadBasicPolicyDesiredCount
  - Components depending on the feature gate:
    - kube-apiserver
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
If additionally the API changes needs to be disabled, the GenericWorkload feature gate needs to
also be disabled. However, the content of `spec.workloadRef` fields in Pod objects will not be
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

Yes. New field (spec.workloadRef) is added to the Pod API:
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
However:
- the other implementations don't address all the issues (e.g. different kinds of
  races/deadlocks) that this proposal paves the way for addressing
- the introduced concepts are fundamental enough in AI era, that we believe that
  our users shouldn't need to install any extensions to have them addressed


## Alternatives

### API

The longer version of this design describing the whole thought process of choosing the
above described approach can be found in the [extended proposal] document.

[extended proposal]: https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?

It's maybe worth noting that we started the KEP with a different API definition of
`PodGroup`, but based on the community discussions and feedback decided to change it.
The original API definition for `PodGroup` was as following:

```go
type GangMode string
const (
	// GangModeOff means that all pods in this PodGroup do not need to be scheduled as a gang.
	GangModeOff GangMode = "Off"

	// GangModeSingle means that all pods in this PodGroup need to be scheduled as one gang.
	GangModeSingle GangMode = "Single"

	// GangModeReplicated means that there is a variable number of identical copies of this PodGroup,
    //  as specified in Replicas, and each copy needs to be independently gang scheduled.
	GangModeReplicated GangMode = "Replicated"
)

// GangSchedulingPolicy holds options that affect how gang scheduling of one PodGroup is handled by the scheduler.
type GangSchedulingPolicy struct {
    // SchedulingTimeoutSeconds defines the timeout for the scheduling logic.
    // Namely it's timeout from the moment when the first  pod show up in
    // PreEnqueue, until those pods are observed in WaitOnPermit - for context
    // see https://kubernetes.io/docs/concepts/scheduling-eviction/scheduling-framework/#interfaces
    // If the timeout is hit, we reject all the waiting pods, free the resources
    // they were reserving and put all of them back to scheduling queue.
    //
    // We decided to drop the field for Alpha because:
    // 1) it won't be obvious for majority of users how to set it
    // 2) it's usefulness after Beta is unclear - see:
    //   https://github.com/kubernetes/enhancements/pull/5558#discussion_r2400876903
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
    GangMode *GangMode // default is "Off"

    // Optional when GangMode = "ReplicatedGang".
    // Forbidden otherwise.
    Replicas int

    // GangSchedulingPolicy defines the options applying to all pods in this gang.
    // Forbidden if GangMode is set to "Off".
    GangSchedulingPolicy GangSchedulingPolicy
}
```

### Pod group queueing in scheduler

In selecting the optimal pod group queuing mechanism, we evaluated several alternatives:

Alternative 0 (Keep current queueing and ordering):

We can minimize changes by retaining the current queueing and ordering logic.
When a Pod is popped, the scheduler can check if it belongs to a `PodGroup`
requiring a Workload Scheduling Cycle. As we add scheduling priorities
for pod groups later, this alternative naturally evolves into Alternative 1.
* *Pros:* Fits the current architecture. Retains current reasoning about the
  scheduling queue. Minimizes implementation effort.
* *Cons:* Might be problematic when some of the pod groups's pods are in the backoffQ
  or unschedulablePods and need to be retrieved efficiently.
  Makes it hard to further evolve the Workload Scheduling Cycle.
  Observability, currently suited for pod-by-pod scheduling, may not
  accurately reflect the state of the queue (e.g., pending gangs).
  Likely harder to support future extensions and won't work well
  if `PodGroup` becomes a separate top-level resource.
  The pod group will be likely scheduled based on the highest priority member,
  meaning the latter pod-by-pod cycles might be visibly delayed for lower priority Pods.

Alternative 1 (Modify sorting logic):

Modify the sorting logic within the existing `PriorityQueue` to put all pods
from a pod group one after another.
* *Pros:* Fits the current architecture.
* *Cons:* Might be problematic when some of the pod groups's pods are in the
  backoffQ or unschedulablePods and need to be retrieved efficiently.
  Makes it hard to further evolve the Workload Scheduling Cycle.
  Would need to inject the workload priority into each of the Pods
  or somehow apply the lowest pod's priority to the rest of the group.

Alternative 2 (Store a PodGroup instance):

Modify the scheduling queue's data structures to accept `QueuedPodGroupInfo` alongside `QueuedPodInfo`.
This allows reusing existing queue logic while extending it to `PodGroups`.
All queued members would be stored in a new data structure
and retrieved for the Workload Cycle when the `PodGroup` is popped.
* *Pros:* Makes it easier to obtain all pods in a group and reduces queue size.
  Reuses current logic for popping, enforcing backoff, and processing unschedulable entities.
* *Cons:* Requires adapting the scheduling queue to handle `PodGroups` as
  queueable entities, which is non-trivial and might clutter the code.

Alternative 3 (Dedicated PodGroup queue):

Introduce a completely separate queue for PodGroups alongside the `activeQ` for Pods.
The scheduler would pop the item (Pod or PodGroup) with the highest priority/earliest timestamp.
Pods belonging to an enqueued PodGroup won't be allowed in the `activeQ`.
* *Pros:* Clean separation of concerns. Can easily use the Workload scheduling priority.
  Can report dedicated logs and metrics with less confusion to the user.
* *Cons:* Significant and non-trivial architectural change to the scheduling queue
  and `scheduleOne` loop.

Ultimately, Alternative 3 (Dedicated PodGroup queue) was chosen as the best long-term solution.

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->

[^1]: The Kubernetes community uses the term "gang scheduling" to mean "all-or-nothing scheduling of a set of pods" [1,2,3,4,5,6,7,8,9,10,11,12,13]. In the Kubernetes context, it does not imply time-multiplexing (in contrast to prior academic work such as [Feitelson and Rudolph](https://doi.org/10.1016/0743-7315(92)90014-E), and in contrast to [Slurm Gang Scheduling](https://slurm.schedmd.com/gang_scheduling.html)).  

[^2]: [API Design for Gang and Workload-Aware Scheduling](https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?pli=1&tab=t.0)

[^3]: Volcano.sh, Co-scheduling plugin, Preferred Networks Plugin, and Kueue all implement gang scheduling outside of kube-scheduler.  Additionally, two previous proposals have been made on this KEP's issue.  These alternatives are compared in detail in the [Background tab of the API Design for Gang Scheduling](https://docs.google.com/document/d/1ulO5eUnAsBWzqJdk_o5L-qdq5DIVwGcE7gWzCQ80SCM/edit?pli=1&tab=t.3zjbiyx2yldg).

