# KEP-6277: Workload API Integration with StatefulSet

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: All-or-Nothing Placement for a Quorum-Based Database](#story-1-all-or-nothing-placement-for-a-quorum-based-database)
    - [Story 2: Atomic Preemption for a Workload With No Partial Value (<code>disruptionMode: all</code>)](#story-2-atomic-preemption-for-a-workload-with-no-partial-value-disruptionmode-all)
    - [Story 3: Rack-Local Placement for Distributed Training (<code>schedulingConstraints.topology</code>)](#story-3-rack-local-placement-for-distributed-training-schedulingconstraintstopology)
    - [Story 4: Sharing One ResourceClaim Across the Whole StatefulSet](#story-4-sharing-one-resourceclaim-across-the-whole-statefulset)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Workload and PodGroup Discovery](#workload-and-podgroup-discovery)
  - [Controller Workflow](#controller-workflow)
  - [Composition by Higher-Level Controllers](#composition-by-higher-level-controllers)
  - [OwnerReferences Relationship](#ownerreferences-relationship)
  - [Pod Management Policy Constraints](#pod-management-policy-constraints)
  - [Field Mapping](#field-mapping)
  - [StatefulSet with Parallel Pod Management](#statefulset-with-parallel-pod-management)
  - [StatefulSet with OrderedReady](#statefulset-with-orderedready)
  - [RollingUpdate.Partition (deferred to Beta)](#rollingupdatepartition-deferred-to-beta)
  - [Lifecycle Management](#lifecycle-management)
    - [Initial Creation Lifecycle](#initial-creation-lifecycle)
    - [Scale Lifecycle](#scale-lifecycle)
  - [Opting into Workload-Aware Scheduling](#opting-into-workload-aware-scheduling)
  - [Admission Validation](#admission-validation)
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
  - [Alternative 1: Explicit gangScheduling API Field on StatefulSet](#alternative-1-explicit-gangscheduling-api-field-on-statefulset)
  - [Alternative 2: Create Workload by Default for All StatefulSets](#alternative-2-create-workload-by-default-for-all-statefulsets)
- [Infrastructure Needed](#infrastructure-needed)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements]
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP brings Workload-Aware Scheduling (WAS) to stateful workloads by adding a `scheduling`
field to `StatefulSetSpec`. The field carries the PodGroup-level capabilities the scheduler already
implements: the scheduling policy
([KEP-4671](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling)) —
`gang` for all-or-nothing placement, or `basic`, the default, for today's pod-by-pod scheduling —
the disruption mode
([KEP-5710](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5710-workload-aware-preemption)),
topology constraints
([KEP-5732](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5732-topology-aware-workload-scheduling)),
and shared ResourceClaims
([KEP-5729](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5729-resourceclaim-support-for-workloads)).
What this KEP adds is the ability to ask for them declaratively on the StatefulSet. The
[Motivation](#motivation) and [User Stories](#user-stories) below take each capability in turn.

The field exposes the reusable scheduling building blocks introduced by
[KEP-6089](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis).
The StatefulSet controller compiles it — via the shared `workloadbuilder` library — into one
`Workload` (static template) and one `PodGroup` (runtime scheduling unit) in
`scheduling.k8s.io/v1beta1`, and stamps `spec.schedulingGroup.podGroupName` onto every pod it
creates so the scheduler can act on the group as a whole. The controller owns the lifecycle of both
objects and keeps the pod-to-group mapping correct across scale, rollout, and pod recreation.

**The catch: gang scheduling requires `podManagementPolicy: Parallel`.** The default `OrderedReady`
creates pods one at a time and waits for each to become ready, which can never satisfy a gang — so
a StatefulSet combining `OrderedReady` with `schedulingPolicy.gang` is rejected by validation. Every
other capability is unaffected: `OrderedReady` StatefulSets may still opt into `spec.scheduling`
with the `basic` policy and use disruption mode, topology constraints, and shared ResourceClaims.

The integration follows the patterns established by the Job controller integration
([KEP-5547](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/5547-integrate-workload-with-job)),
so the StatefulSet surface reads the same as the Job surface.

## Motivation

StatefulSets are the primary Kubernetes primitive for deploying stateful applications—distributed
databases (etcd, CockroachDB, Cassandra), coordination services (ZooKeeper), message queues
(Kafka), and increasingly stateful AI/ML inference serving systems. These workloads have a
fundamental property that the current scheduler does not account for: their pods are
interdependent and a partial deployment is often worse than no deployment at all.

Today, when a StatefulSet with `podManagementPolicy: Parallel` is created, all replica pods are
submitted to the scheduler independently. The scheduler places pods one at a time as resources
become available. This leads to several problems:

1. **Split-brain and quorum failures**: A distributed database configured for 5 replicas may have
   only 3 pods scheduled, which is enough to form a quorum but not enough for the intended
   fault-tolerance level. The 2 remaining pods may be stuck pending indefinitely, leaving the
   cluster in a degraded state that operators may not detect until a failure occurs.

2. **Resource waste from partial placement**: If only some pods of a tightly-coupled stateful
   workload can be placed, those running pods consume resources without providing useful work,
   since the application cannot function until all members are present.

3. **Deadlock scenarios**: In a resource-constrained cluster, two StatefulSets may each get partial
   placement, with neither able to acquire enough resources to run. Gang scheduling prevents this
   by ensuring resources are committed atomically.

4. **No way to declare the preemption unit**: The scheduler always preempts StatefulSet pods
   individually. That is the right default for a redundant ensemble built to survive losing a
   member, but wrong for a StatefulSet with no partial value — a model sharded across every
   replica, say — where preempting one pod frees too little for the preemptor while the rest keep
   their resources and produce nothing. There is no way to tell the scheduler which shape a given
   StatefulSet is.

5. **No group-level placement control**: `podAffinity`/`topologySpreadConstraints` are per-pod
   hints evaluated pod-by-pod; they cannot express "place all replicas of this StatefulSet inside
   one rack" as an atomic requirement, which is what latency-sensitive replication and
   shard-to-shard traffic actually need.

6. **No group-level device sharing**: A StatefulSet whose replicas must share one DRA device (a
   pooled accelerator, a virtual device representing a topological domain, shared memory) has no
   way to say so. Each pod either gets its own claim, or the user hand-manages a ResourceClaim
   outside the StatefulSet and runs into the 256-entry `status.reservedFor` limit.

All six are addressed by the same mechanism: describing the StatefulSet's pods to the scheduler as
a `PodGroup` and attaching group-level policy to it.

The Workload API was introduced in
[KEP-4671](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling)
to provide a standardized mechanism for describing groups of pods that must be scheduled together.
[KEP-5832](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5832-decouple-podgroup-api)
decoupled `PodGroup` as a standalone runtime object, and
[KEP-6089](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis)
introduced the reusable `scheduling.k8s.io/v1alpha3` building blocks and the shared
`workloadbuilder` library for controller integration. Together these provide the `Workload` (static
template) and `PodGroup` (runtime scheduling unit) objects, and the scheduling policy, disruption
mode, topology constraint, and shared-ResourceClaim building blocks that this KEP consumes.

This KEP focuses specifically on the StatefulSet controller integration, following the patterns
established by the Job controller integration in KEP-5547 and aligning with the broader effort to
integrate all workload controllers with the Workload API.

### Goals

- Expose the Workload-Aware Scheduling API surface — `schedulingPolicy`, `schedulingConstraints`,
  `disruptionMode`, and `resourceClaims` — on `StatefulSetSpec` via a new `scheduling` field, using
  the reusable building blocks from KEP-6089 rather than StatefulSet-specific field definitions.

- Support gang scheduling for StatefulSets using `podManagementPolicy: Parallel`, where `minCount`
  pods must be schedulable before any are bound to nodes.

- Support the `Basic` policy — including for `podManagementPolicy: OrderedReady` — so a StatefulSet
  can use disruption mode, topology constraints, and shared ResourceClaims, plus workload-level
  observability, without changing its sequential scheduling behavior.

- Enable per-PodGroup ResourceClaim sharing via the `resourceClaims` field: the controller
  populates `PodGroupTemplate.resourceClaims` so that the ResourceClaim controller and scheduler
  handle generation, allocation, and reservation at the PodGroup level rather than per pod (per
  [KEP-5729](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5729-resourceclaim-support-for-workloads)).

- Validate `spec.scheduling` at admission, on three levels: structural rules via declarative
  validation on the building blocks themselves, semantic rules via `workloadbuilder`, and the
  StatefulSet-specific cross-field rules — notably rejecting `schedulingPolicy.gang` combined with
  `podManagementPolicy: OrderedReady`.

- Compile `spec.scheduling` into exactly one `Workload` and one `PodGroup` per StatefulSet using
  the shared `workloadbuilder` library, ensuring consistency with other controller integrations
  (Job, JobSet).

- Populate `pod.spec.schedulingGroup.podGroupName` on all pods created by the StatefulSet
  controller, linking each pod to its PodGroup for scheduler consumption.

- Handle the StatefulSet lifecycle correctly — initial creation, scaling up and down, and deletion
  — with proper Workload/PodGroup creation, in-place update, and cleanup.

- Follow the ownership model defined in the Workload API: one Workload and one PodGroup per
  StatefulSet, both owned via `ownerReference` with `controller=true` and
  `blockOwnerDeletion=true`.

### Non-Goals

- **Changes to scheduling or preemption logic.** Gang scheduling (KEP-4671), workload-aware
  preemption (KEP-5710), topology-aware scheduling (KEP-5732), and PodGroup-level ResourceClaim
  lifecycle (KEP-5729) are implemented in kube-scheduler and the ResourceClaim controller. This KEP
  adds no scheduling logic of its own.

- Introducing changes to the Workload API itself (`scheduling.k8s.io`). This KEP consumes the
  building blocks as defined in KEP-4671 and KEP-6089; any API changes are tracked separately.

- Modifying the `OrderedReady` pod management policy to support gang scheduling. Its sequential
  creation semantics are fundamentally incompatible with a gang.

- Composing StatefulSets into larger scheduling structures *through the StatefulSet API*. A
  StatefulSet compiles to exactly one leaf `PodGroup` in its own namespace; there is no field for
  cross-namespace Workload references or for KEP-6012's `CompositePodGroup`. A higher-level
  controller may still own the Workload and attach the StatefulSet's PodGroup to its own structure
  via the KEP-6089 downward-mapping annotations; see
  [Composition by Higher-Level Controllers](#composition-by-higher-level-controllers).

- Automatic detection of whether a StatefulSet "should" use Workload-Aware Scheduling via
  heuristics. The user must explicitly opt in by setting `spec.scheduling` on the StatefulSet.

- Per-replica ResourceClaims analogous to `volumeClaimTemplates`, where each replica gets its own
  dedicated claim. Only per-PodGroup ResourceClaims — one claim shared across the group — are
  supported; per-replica semantics are deferred to future work.

- Switching a StatefulSet's scheduling *shape* after creation: adding or removing
  `spec.scheduling`, flipping between `gang` and `basic`, or changing `schedulingConstraints`,
  `disruptionMode`, or `resourceClaims`. Each would require recreating the Workload and PodGroup
  under a running StatefulSet and — since `pod.spec.schedulingGroup.podGroupName` is immutable —
  its pods. Out of scope for Alpha, Beta, and GA alike; Alpha enforces it with a `+k8s:immutable`
  marker on the whole field (see [Admission Validation](#admission-validation)).

- User-configurable `gang.minCount`. For Alpha the controller always derives it from
  `spec.replicas`. Making it configurable — allowing `minCount < replicas` for partial-gang
  semantics — and mutable in flight is planned for Beta, and is the one part of the immutability
  above expected to be relaxed. See [Beta graduation criteria](#beta).

- `updateStrategy.rollingUpdate.partition` (canary rollouts) combined with `spec.scheduling`,
  rejected by validation for Alpha. Whether a partitioned StatefulSet should stay a single PodGroup
  across revisions, rather than splitting into one group per partition, is an open design question
  deferred to Beta. See [RollingUpdate.Partition](#rollingupdatepartition-deferred-to-beta).

## Proposal

The StatefulSet controller in `kube-controller-manager` will be extended to create and manage
`Workload` and `PodGroup` objects based on the StatefulSet's configuration. A new `scheduling`
field is added to `StatefulSetSpec`, embedding the reusable `scheduling.k8s.io/v1alpha3` building
blocks (scheduling policy, scheduling constraints, disruption mode, shared ResourceClaims).

Following the decoupled model from KEP-5832 and the controller pattern from KEP-6089, the
integration uses the `workloadbuilder` library:

```go
// Scheduling, if set, opts this StatefulSet into Workload-Aware Scheduling:
// the controller compiles it into one Workload and one PodGroup, and every pod
// the StatefulSet creates references that PodGroup.
// An empty value (scheduling: {}) is valid and resolves to the Basic policy
// with no constraints, no disruption mode, and no shared claims.
// Gang scheduling requires podManagementPolicy: Parallel.
// This field is immutable for Alpha; see the Non-Goals for what is expected to
// be relaxed in Beta.
//
// +featureGate=WorkloadWithStatefulSet
// +optional
// +k8s:ifDisabled(WorkloadWithStatefulSet)=+k8s:forbidden
// +k8s:optional
// +k8s:immutable
Scheduling *StatefulSetSchedulingConfiguration `json:"scheduling,omitempty"`
```

Marking the whole `Scheduling` field `+k8s:immutable` for Alpha is deliberate: no sub-field is
user-mutable in Alpha, so a single marker on the parent is simpler and stricter than per-field
`NoSet`/`NoUnset` plus hand-written rules, and it can be relaxed field-by-field later (relaxing
validation is a compatible change; tightening it is not). When `gang.minCount` becomes
user-configurable in Beta, the marker moves down to the individual sub-fields.

`StatefulSetSchedulingConfiguration` reuses the `scheduling.k8s.io/v1alpha3` building-block types:

```go
type StatefulSetSchedulingConfiguration struct {
    // SchedulingPolicy defines the scheduling policy for this StatefulSet.
    // Exactly one of Basic or Gang must be set. When SchedulingPolicy is nil,
    // the controller resolves it to Basic.
    // Gang requires podManagementPolicy: Parallel; combining Gang with
    // OrderedReady is rejected at admission.
    // For Alpha, the controller derives gang minCount from spec.replicas and a
    // user-supplied minCount is rejected at admission. User-configurable
    // minCount is planned for Beta.
    //
    // +optional
    // +k8s:optional
    SchedulingPolicy *schedulingv1alpha3.WorkloadPodGroupSchedulingPolicy `json:"schedulingPolicy,omitempty" protobuf:"bytes,1,opt,name=schedulingPolicy"`

    // SchedulingConstraints defines scheduling constraints (e.g. topology)
    // for the StatefulSet's pods. Setting schedulingConstraints.topology
    // requires every pod of the StatefulSet to land in a single instance of
    // the named topology domain (KEP-5732).
    //
    // +optional
    // +k8s:optional
    SchedulingConstraints *schedulingv1alpha3.WorkloadPodGroupSchedulingConstraints `json:"schedulingConstraints,omitempty" protobuf:"bytes,2,opt,name=schedulingConstraints"`

    // DisruptionMode defines the mode in which the StatefulSet's pods can be
    // disrupted. Exactly one of Single or All must be set.
    // Single (the scheduler default when unset) treats each pod as an
    // independent preemption victim; All makes the whole StatefulSet a single
    // atomic preemption unit.
    // All is only valid with the Gang policy: workloadbuilder rejects
    // `all` combined with `basic`, since a Basic group is scheduled
    // independently and all-or-nothing disruption is meaningless for it.
    //
    // +optional
    // +k8s:optional
    DisruptionMode *schedulingv1alpha3.WorkloadPodGroupDisruptionMode `json:"disruptionMode,omitempty" protobuf:"bytes,3,opt,name=disruptionMode"`

    // ResourceClaims defines which ResourceClaims may be shared among Pods in
    // the StatefulSet. Pods consume the devices allocated to a PodGroup's
    // claim by defining a claim in its own Spec.ResourceClaims that matches
    // the PodGroup's claim exactly. The claim must have the same name and
    // refer to the same ResourceClaim or ResourceClaimTemplate.
    // At most 4 claims may be set, matching the limit on the resulting
    // PodGroup.
    //
    // This field requires the DRAWorkloadResourceClaims feature gate
    // (KEP-5729) to be enabled in kube-apiserver, kube-controller-manager,
    // kube-scheduler, and kubelet.
    //
    // +optional
    // +patchMergeKey=name
    // +patchStrategy=merge
    // +listType=map
    // +listMapKey=name
    // +k8s:optional
    // +k8s:listType=map
    // +k8s:listMapKey=name
    // +k8s:maxItems=4
    // +featureGate=DRAWorkloadResourceClaims
    ResourceClaims []schedulingv1alpha3.WorkloadPodGroupResourceClaim `json:"resourceClaims,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,4,rep,name=resourceClaims"`
}
```

When `spec.scheduling` is non-nil, the StatefulSet controller:

1. Discovers or creates the `Workload` — the static template that defines the PodGroup template.
2. Discovers or creates the `PodGroup` — the runtime scheduling unit instantiated from that
   template.
3. Creates pods with `spec.schedulingGroup.podGroupName` referencing the PodGroup.

The scheduler then places those pods as a single PodGroup, applying its `schedulingPolicy`,
`schedulingConstraints`, `disruptionMode`, and `resourceClaims`. Discovering rather than blindly
creating is what keeps steps 1 and 2 safe across controller restarts and under composition by a
higher-level controller; see
[Workload and PodGroup Discovery](#workload-and-podgroup-discovery) for how the objects are found
and [Initial Creation Lifecycle](#initial-creation-lifecycle) for ordering.

**Interaction with `podManagementPolicy`.** Gang scheduling requires `Parallel`. A StatefulSet that
combines `schedulingPolicy.gang` with `OrderedReady` — or leaves `podManagementPolicy` unset, since
`OrderedReady` is the default — is rejected at admission: `OrderedReady` creates pods one at a time
and waits for each to become Ready, so a gang of more than one pod can never be satisfied.
`OrderedReady` StatefulSets may still set `spec.scheduling` with `basic`, gaining topology
constraints, shared ResourceClaims, and workload-level observability without changing their
sequential creation behavior.

### User Stories

#### Story 1: All-or-Nothing Placement for a Quorum-Based Database

As a platform engineer deploying a 5-node CockroachDB cluster (or a 3-node ZooKeeper ensemble — the
shape is the same), I want all replicas to be scheduled
atomically so that the cluster starts at full replication factor and never enters a degraded state
due to partial scheduling. Without gang scheduling, 3 of 5 pods might get scheduled — enough to form
a quorum but with reduced fault tolerance — while the remaining 2 stay Pending indefinitely due to
resource fragmentation, consuming resources without providing the intended durability.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: cockroachdb
spec:
  serviceName: cockroachdb
  replicas: 5
  podManagementPolicy: Parallel
  scheduling:
    schedulingPolicy:
      gang: {}          # minCount derived from spec.replicas (5)
  template:
    spec:
      containers:
      - name: cockroachdb
        image: cockroachdb/cockroach:v23.2
        resources:
          requests:
            cpu: "4"
            memory: 8Gi
```

#### Story 2: Atomic Preemption for a Workload With No Partial Value (`disruptionMode: all`)

As an ML engineer serving one large model sharded across a StatefulSet's 8 replicas, my pods are
not interchangeable copies — each holds a distinct shard, and the service produces no output
unless all 8 are running. Preempting a single pod frees one pod's worth of capacity, usually not
enough for the preemptor, and takes the whole service down while the other 7 keep their GPUs and
produce nothing. I want the StatefulSet to be one preemption unit: either the whole group goes and
its capacity is actually returned, or none of it does.

`disruptionMode: all` expresses exactly this, and requires the `gang` policy:

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: llm-shards
spec:
  serviceName: llm-shards
  replicas: 8
  podManagementPolicy: Parallel
  scheduling:
    schedulingPolicy:
      gang: {}
    disruptionMode:
      all: {}           # the whole StatefulSet is one atomic preemption unit
  template:
    spec:
      containers:
      - name: shard
        image: my-org/llm-inference:latest
        resources:
          requests:
            nvidia.com/gpu: 1
```

In practice this shape usually reaches a cluster through a higher-level controller rather than a
hand-written StatefulSet — LeaderWorkerSet is purpose-built for it and creates StatefulSets as its
building blocks; see
[Composition by Higher-Level Controllers](#composition-by-higher-level-controllers).

**`all` is the exception, not the default for stateful workloads.** "Preemption breaks my quorum"
is a tempting but usually wrong reason to reach for it: a 5-replica CockroachDB is built on
redundancy precisely so that losing one member is survivable, and `all` would turn that survivable
eviction into a full outage. For a redundant ensemble the right answer is `single` (the default)
plus a PodDisruptionBudget. `all` earns its place only where partial capacity has genuinely zero
value — a sharded model, a tightly-coupled MPI-style computation, a pipeline whose stages are all
required.

Omitting `disruptionMode` (or setting `single: {}`) keeps today's behavior: each pod is an
independent preemption victim. The preemption machinery itself is
[KEP-5710](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5710-workload-aware-preemption);
this KEP only lets a StatefulSet author ask for it.

#### Story 3: Rack-Local Placement for Distributed Training (`schedulingConstraints.topology`)

As an ML engineer deploying a distributed training job as a StatefulSet where each pod holds a
model shard, I need every pod placed inside the *same* rack so that gradient exchange stays on the
top-of-rack fabric instead of crossing the spine. `podAffinity` and `topologySpreadConstraints`
cannot express this: they are evaluated pod-by-pod, so the first pods can land in a rack that
cannot fit the rest, and the group ends up split. A topology constraint on the PodGroup makes
"all replicas in one rack instance" a property of the group, evaluated as a unit.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: training-shards
spec:
  serviceName: training-shards
  replicas: 8
  podManagementPolicy: Parallel
  scheduling:
    schedulingPolicy:
      gang: {}
    schedulingConstraints:
      topology:
      - key: topology.kubernetes.io/rack
  template:
    spec:
      containers:
      - name: trainer
        image: my-org/distributed-trainer:latest
        resources:
          requests:
            nvidia.com/gpu: 1
```

The topology semantics — which domains exist, how the scheduler searches them, and how it falls
back — are owned by
[KEP-5732](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5732-topology-aware-workload-scheduling).
This KEP surfaces the field on StatefulSet, validates it, and compiles it into the PodGroup.

#### Story 4: Sharing One ResourceClaim Across the Whole StatefulSet

As a workload author running a StatefulSet whose replicas all need access to the *same* DRA device
— a pooled hardware accelerator, a shared-memory device, or any device meant to be consumed
collectively rather than per pod — I want one `ResourceClaim` allocated once and shared by every
pod in the group. Today I either get one claim per pod (wrong: the device is shared, not
replicated), or I create the claim by hand outside the StatefulSet, which puts its lifecycle out of
sync with the StatefulSet's and runs into the 256-entry `status.reservedFor` limit once the
StatefulSet grows past 256 replicas.

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaimTemplate
metadata:
  name: shared-accelerator
spec:
  spec:
    devices:
      requests:
      - name: device
        exactly:
          deviceClassName: shared-accelerator.example.com
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: shard-server
spec:
  serviceName: shard-server
  replicas: 4
  podManagementPolicy: Parallel
  scheduling:
    schedulingPolicy:
      gang: {}
    resourceClaims:
    - name: shared
      resourceClaimTemplateName: shared-accelerator
  template:
    spec:
      containers:
      - name: server
        image: my-org/shard-server:latest
        resources:
          claims:
          - name: shared
      resourceClaims:
      - name: shared
        resourceClaimTemplateName: shared-accelerator
```

The StatefulSet controller puts a `resourceClaims` entry on the PodGroup referencing the
`shared-accelerator` ResourceClaimTemplate. The ResourceClaim controller generates **one**
ResourceClaim for the PodGroup (not one per pod), the scheduler allocates the device once, and the
claim is reserved for the PodGroup rather than for each individual pod, so the `reservedFor` limit
does not scale with `spec.replicas`. Each pod's `spec.resourceClaims` entry must match the
PodGroup's claim by name and source to consume it.

### Notes/Constraints/Caveats

1. **OrderedReady incompatibility**: Gang scheduling is fundamentally incompatible with
   `podManagementPolicy: OrderedReady` because `OrderedReady` creates pods sequentially (waiting
   for each to become Ready before creating the next), which contradicts gang scheduling's
   requirement that all pods be created and evaluated simultaneously. Admission validation MUST
   reject a StatefulSet that specifies both `OrderedReady` and a gang scheduling policy in
   `spec.scheduling.schedulingPolicy.gang`.

2. **User-set `gang.minCount` is rejected for Alpha**: For Alpha, `minCount` is always derived
   from `spec.replicas` by the controller, and a user-supplied `minCount` value is rejected at
   admission. A gang is satisfiable once `minCount` pods can be scheduled, so allowing
   `minCount < replicas` would enable useful partial-gang semantics (e.g., a 5-replica database
   that can start with 3 members). Making `minCount` user-configurable is planned for Beta
   (see [Beta graduation criteria](#beta)).

3. **schedulingGroup immutability on pods**: The `spec.schedulingGroup.podGroupName` field on pods
   is immutable after creation, so a running pod can never be reassigned to a different PodGroup.
   The design in this KEP avoids ever needing to: a StatefulSet always maps to exactly **one**
   PodGroup, whose name is derived from the StatefulSet's name and does not change over the
   StatefulSet's lifetime. This is the main reason `rollingUpdate.partition` is deferred (see
   [RollingUpdate.Partition](#rollingupdatepartition-deferred-to-beta)) — any design that splits a
   StatefulSet across multiple PodGroups would force pod recreation purely to move pods between
   scheduling groups, which is not acceptable.

4. **Finalizer requirement**: The PodGroup must have a finalizer to prevent premature deletion
   while pods still reference it (KEP-5832 deletion protection). Without this, deleting a
   PodGroup would orphan its pods, leaving them pointing to a non-existent PodGroup and causing
   scheduler issues. The finalizer is added at PodGroup creation and removed only when all
   referencing pods reach a terminal phase.

5. **Feature gate dependencies**: This feature depends on the `GenericWorkload` feature gate
   (introduced in KEP-4671, consolidated in 1.37) being enabled in both `kube-controller-manager`
   and `kube-scheduler`. The `resourceClaims` field additionally requires
   `DRAWorkloadResourceClaims` (KEP-5729) in kube-apiserver, kube-controller-manager,
   kube-scheduler, and kubelet; with that gate off the API server clears the field before it is
   stored — as StatefulSet already does for its other feature-gated fields — so the controller
   never sees it.

6. **ResourceClaim lifecycle tied to PodGroup**: ResourceClaims generated from
   ResourceClaimTemplates referenced by a PodGroup are owned by the PodGroup (via
   `ownerReferences`). When the PodGroup is deleted, the generated ResourceClaim is deallocated
   and garbage-collected. Because a StatefulSet keeps a single PodGroup for its whole lifetime,
   generated claims survive scaling and rolling updates and are removed only when the StatefulSet
   itself is deleted.

7. **Maximum 4 ResourceClaims per PodGroup**: KEP-5729 limits the `resourceClaims` list to 4
   entries per PodGroupTemplate/PodGroup. This limit applies to the StatefulSet's
   `spec.scheduling.resourceClaims` field as well.

8. **Higher-level controllers composing StatefulSets**: A StatefulSet created by a parent
   controller (LeaderWorkerSet, for example) can set `spec.scheduling` and still let the parent own
   the `Workload` and the `CompositePodGroup` structure; the StatefulSet controller creates one only
   when no parent already owns it, following the KEP-6089 downward-mapping annotations. See
   [Composition by Higher-Level Controllers](#composition-by-higher-level-controllers).

9. **Discovery is by reference, not by name**: Following
   [KEP-5547](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/5547-integrate-workload-with-job#workload-and-podgroup-discovery),
   the controller finds its Workload via `workload.spec.controllerRef` and its PodGroup via
   `podGroup.spec.podGroupTemplateRef` — never by reconstructing a name. Names are for human
   readability only, so the naming scheme can change in a later release without breaking
   reconciliation. A consequence worth stating: the controller never blindly creates, it discovers
   first and creates only what is missing, which is what makes crash recovery and re-enablement
   after a rollback safe.

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| `rollingUpdate.partition` cannot be used with `spec.scheduling` in Alpha | Partitioned canary rollouts are unavailable to any StatefulSet using WAS, and immutability means one that already opted in cannot opt out to run one | Rejected by validation on create and update with a clear message. See [RollingUpdate.Partition](#rollingupdatepartition-deferred-to-beta) |
| Rolling update of a gang-scheduled StatefulSet stalls | A pod recreated with a revision that no longer fits (bigger requests, new topology domain) cannot be placed, and the gang stays unsatisfied | Visible in PodGroup status and Pending-pod signals; rolling-update semantics under gang are a Beta design item |
| Scale-up makes the gang unsatisfiable | `minCount` rises to the new replica count, so the new pods stay Pending and the group is never satisfied. In Alpha the only exit is scaling `replicas` back down, since `minCount` is not user-settable | Raising `minCount` never evicts bound pods — the StatefulSet keeps serving at its previous size. User-configurable `minCount` in Beta lets a StatefulSet set a floor below `replicas` |
| Volume bindings conflict with group placement | PVCs bind to PVs that may carry node affinity (local or zonal storage), so a `schedulingConstraints.topology` constraint can be unsatisfiable against volumes already bound elsewhere, and each bound volume narrows where the group can be placed on reschedule | Surfaces as an unsatisfied PodGroup, not a partial placement. Topology constraints are intended for StatefulSets whose storage is topology-agnostic or provisioned in the same domain; volume-aware group placement is a Beta design item |
| Parent controller's `PodGroupTemplate` disagrees with the inner StatefulSet's `spec.scheduling` | The delegated PodGroup is instantiated from the parent's template, so an inner StatefulSet asking for e.g. `disruptionMode: all` when the parent's template does not may silently not get it | Alpha open question: parents that own the Workload usually own the policy too, so the conflict should be rare. Alpha will either define precedence or reject the conflict at admission. See [Composition by Higher-Level Controllers](#composition-by-higher-level-controllers) |
| Scheduler missing `GenericWorkload` while the controller has `WorkloadWithStatefulSet` | Pods reference the PodGroup but the scheduler ignores it: they are placed individually and the gang guarantee is silently absent, so a quorum-based StatefulSet can come up partially placed | Nothing fails and no object is corrupted — the exposure is a missing guarantee, not an error. A PodGroup recording no `scheduler_podgroup_schedule_attempts_total` is the signal; see [Version Skew Strategy](#version-skew-strategy) |
| `DRAWorkloadResourceClaims` gate disabled but `resourceClaims` set | The API server silently clears the field, per the convention for feature-gated fields, so pods are created without the shared claim and may only fail once they try to use the device | The gate dependency is documented in the field godoc and in [Notes/Constraints/Caveats](#notesconstraintscaveats); the drop is observable by comparing the applied manifest against the stored object |

## Design Details

### Workload and PodGroup Discovery

Discovery is by **reference, not by name and not by ownership**, exactly as specified in
[KEP-5547](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/5547-integrate-workload-with-job#workload-and-podgroup-discovery).
This KEP adopts that model unchanged; the rules below are restated only for readability.

A `Workload` is the Workload for a given StatefulSet if:
- it is in the StatefulSet's namespace, and
- its `spec.controllerRef` identifies that StatefulSet (`apiGroup: apps`, `kind: StatefulSet`,
  matching `name`).

A `PodGroup` is the PodGroup for that StatefulSet if:
- it is in the StatefulSet's namespace, and
- its `spec.podGroupTemplateRef.workloadName` is the name of that Workload.

`ownerReference` is used only so that objects the controller created are garbage-collected with the
StatefulSet — it is not a discovery mechanism, because a Workload created by a user or by a parent
controller may legitimately carry no ownerReference to the StatefulSet.

Names are for human readability and carry no semantics, so the scheme can change in a later release
without breaking reconciliation. Following the Job convention:

- **Workload**: `<truncated-statefulset-name>-<hash>`
- **PodGroup**: `<truncated-workload-name>-<truncated-podgrouptemplate-name>-<hash>`

To distinguish objects it created from user-created objects that happen to carry the same
ownerReferences, the controller marks the objects it creates with a `managed-by` annotation, as
the Job integration does.

### Controller Workflow

The controller attempts to *create* Workload and PodGroup only when the StatefulSet has no pods
associated with it yet. Once pods exist, it only discovers and uses whatever is already there. This
is what makes a restart mid-workflow — Workload created but PodGroup or pods not — recoverable: on
the next sync the existing objects are found via the listers and the workflow continues from there.

1. If the StatefulSet carries an `OwnerReference` to a parent controller that owns the Workload,
   the controller does not create a Workload. It then branches on the downward-mapping annotations
   described in [Composition by Higher-Level Controllers](#composition-by-higher-level-controllers).
2. If the StatefulSet already has pods (active or terminal, owned by this StatefulSet), skip
   creation and only discover.
3. Look up the Workload by `spec.controllerRef`. If none exists, compile one from
   `spec.scheduling` via `workloadbuilder` and create it with a controller `ownerReference` and a
   `spec.controllerRef` pointing at the StatefulSet. If more than one is found, treat it as
   ambiguous: create nothing, mutate nothing, and surface an event.
4. Look up the PodGroup by `spec.podGroupTemplateRef` against the target `PodGroupTemplate`. If
   none exists, instantiate it from the template. If more than one is found, fall back — multiple
   PodGroups per StatefulSet are not supported.
5. Run the existing pod-management logic, setting `spec.schedulingGroup.podGroupName` on each pod.

The controller does not update an existing Workload or PodGroup during this discovery path. The
only ongoing reconciliation it performs on them is the in-place `minCount` update on scale (see
[Scale Lifecycle](#scale-lifecycle)).

If the Workload was created by another actor — a user pre-creating one, or a parent controller —
the StatefulSet controller respects and uses it, adds no ownerReference, never mutates it, and
never deletes it. It falls back (ignoring the discovered objects and raising an event) when the
discovered Workload has a shape it cannot use — for Alpha, when its `podGroupTemplates` count is
not 1.

### Composition by Higher-Level Controllers

Controllers such as LeaderWorkerSet create StatefulSets as their building blocks, and a parent may
want to keep the `Workload` and the `CompositePodGroup` structure under its own control while the
inner StatefulSet still carries its own scheduling requirements. Setting `spec.scheduling` on an
inner StatefulSet therefore does not imply that the StatefulSet controller manages a standalone
Workload of its own. The coordination uses the well-known downward-mapping annotations from
[KEP-6089](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis#the-solution-downward-mapping-annotations),
which the parent injects onto each StatefulSet it creates:

| Parent's ownerReference on the StatefulSet | `scheduling.k8s.io/group-template-name` | StatefulSet controller behavior |
|---|---|---|
| Absent | — | **Root case.** The controller compiles and owns both the Workload and the PodGroup, as described above. |
| Present (parent owns the Workload) | Present | **PodGroup delegated.** The controller creates no Workload. It creates its own runtime PodGroup from the parent's named `PodGroupTemplate`, and — when `scheduling.k8s.io/parent-compositepodgroup` is also set — links that PodGroup to the named parent `CompositePodGroup` instance. The PodGroup gets a controller `ownerReference` to the StatefulSet. |
| Present | Absent | **Both delegated.** The parent owns the Workload and the PodGroup. The controller creates neither; it discovers the existing objects and uses them when stamping `spec.schedulingGroup.podGroupName` onto pods. |

The annotations are transient coordination metadata set by controllers, not user-facing scheduling
intent, which is why they are annotations rather than API fields. The exact keys are owned by
KEP-6089; this KEP consumes them.

This is also what allows a composed StatefulSet to contribute a `CompositePodGroup` member without
the StatefulSet controller fighting the parent for ownership, which the earlier revision of this
KEP incorrectly ruled out by telling parent controllers not to set `spec.scheduling` at all.

### OwnerReferences Relationship

This KEP uses the relationship defined by
[KEP-5547](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/5547-integrate-workload-with-job#ownerreferences-relationship)
with `StatefulSet` substituted for `Job`:

```mermaid
flowchart BT
    Pod[Pod]
    PodGroup[PodGroup]
    Workload[Workload]
    StatefulSet[StatefulSet]

    Pod -->|ownerRef| PodGroup
    Pod -->|ownerRef| StatefulSet
    PodGroup -->|ownerRef| StatefulSet
    PodGroup -->|ownerRef <br/> (root StatefulSet only)| Workload
    Workload -->|ownerRef| StatefulSet

    PodGroup -.->|via <br/> podGroupTemplateRef| Workload

    linkStyle 5 stroke:#888,color:#888
```

- The `Workload` carries an ownerReference to the StatefulSet with `controller: true`, when the
  StatefulSet controller created it.
- The `PodGroup` links to its Workload via `spec.podGroupTemplateRef` and carries a controller
  ownerReference to the StatefulSet when the controller created it. A **parent-owned** Workload is
  never given an ownerReference from the PodGroup.
- The `Pod` carries a controller ownerReference to the StatefulSet plus an ownerReference to the
  PodGroup, so garbage collection never leaves a pod pointing at a PodGroup that is gone.
- `OwnerReference.UID` must match the StatefulSet's `metadata.UID`, so a recreated StatefulSet of
  the same name does not adopt the previous one's objects.
- All objects are in the StatefulSet's namespace; ownerReferences cannot cross namespaces.
- The PodGroup additionally carries the KEP-5832 deletion-protection finalizer, removed only once
  every referencing pod has reached a terminal phase.

The controller does not explicitly delete the Workload or PodGroup; ownerReferences and garbage
collection handle it. Objects it did not create are never adopted, so they are never collected with
the StatefulSet either.

### Pod Management Policy Constraints

| `podManagementPolicy` | `schedulingPolicy.gang` | `schedulingPolicy.basic` (or omitted) |
|---|---|---|
| `Parallel` | Accepted — compiles to a Gang PodGroup with `minCount = spec.replicas` | Accepted — compiles to a Basic PodGroup |
| `OrderedReady` (default) | **Rejected at admission** | Accepted — compiles to a Basic PodGroup |

Gang scheduling for StatefulSet **requires** `podManagementPolicy: Parallel`. The `OrderedReady`
policy creates pods sequentially, waiting for each to become Ready before creating the next, which
fundamentally conflicts with gang scheduling's requirement that all pods exist simultaneously for
the scheduler to evaluate them as a group. Rather than silently downgrading such a StatefulSet to
`Basic` — which would give the user a materially different behavior from what they asked for —
admission validation **rejects** it:

```
Error: spec.scheduling.schedulingPolicy.gang: Forbidden: gang scheduling requires
podManagementPolicy: Parallel; OrderedReady creates pods sequentially, which can never
satisfy a gang
```

`podManagementPolicy` is already immutable on StatefulSets, so this decision is locked at creation
time and cannot be reached by a later update.

An `OrderedReady` StatefulSet may still set `spec.scheduling` with `basic: {}` (or omit
`schedulingPolicy`, which resolves to `Basic`): it gets a Workload and PodGroup, keeps its existing
sequential scheduling behavior, and can use `schedulingConstraints.topology` and `resourceClaims`.

**DisruptionMode with Basic policy**: `disruptionMode: all` requires the `Gang` policy, and the
combination with `Basic` is rejected rather than ignored. The reason is that a preemption unit
cannot be larger than the scheduling unit, so
[KEP-5710](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5710-workload-aware-preemption#preemption-unit)
specifies validation preventing `All` on a PodGroup with a Basic policy; StatefulSet inherits it by
running the same `workloadbuilder` validation at admission (see
[Admission Validation](#admission-validation)). `disruptionMode: single` is valid with either
policy and matches today's behavior.

### Field Mapping

The StatefulSet controller maps StatefulSet fields to Workload API fields as follows:

| StatefulSet Field | Workload Field | Mapping Logic |
|-------------------|---------------|---------------|
| `metadata.name` | `spec.controllerRef.name` | Direct reference |
| `"apps"` | `spec.controllerRef.apiGroup` | Constant |
| `"StatefulSet"` | `spec.controllerRef.kind` | Constant |
| `spec.replicas` | `podGroupTemplates[0].schedulingPolicy.gang.minCount` | Derived from replicas (Alpha); user-configurable in Beta |
| `spec.scheduling.schedulingConstraints` | `podGroupTemplates[0].schedulingConstraints` | Pass-through |
| `spec.scheduling.disruptionMode` | `podGroupTemplates[0].disruptionMode` | Pass-through |
| `spec.scheduling.resourceClaims` | `podGroupTemplates[0].resourceClaims` | Pass-through; each entry maps to a `PodGroupResourceClaim` |

### StatefulSet with Parallel Pod Management

When `podManagementPolicy` is set to `Parallel`, the StatefulSet controller creates all replica
pods simultaneously without waiting for sequential readiness. This aligns naturally with gang
scheduling because all pods are created together and should be scheduled together.

The controller creates a Workload (static template), a PodGroup (runtime scheduling unit), and
then the pods. The MinCount equals the replica count, ensuring all StatefulSet pods are scheduled
atomically.

```yaml
# Workload (static template)
apiVersion: scheduling.k8s.io/v1beta1
kind: Workload
metadata:
  name: my-statefulset-a1b2c3      # <sts-name>-<hash>; not used for discovery
  annotations:
    scheduling.k8s.io/managed-by: statefulset-controller
  ownerReferences:
  - apiVersion: apps/v1
    kind: StatefulSet
    name: my-statefulset
    uid: <statefulset-uid>
    controller: true
    blockOwnerDeletion: true
spec:
  controllerRef:                   # this is what discovery matches on
    apiGroup: apps
    kind: StatefulSet
    name: my-statefulset
  podGroupTemplates:
  - name: my-statefulset
    schedulingPolicy:
      gang:
        minCount: 3  # Matches spec.replicas
    # Present only when spec.scheduling.resourceClaims is non-empty
    resourceClaims:            # (requires DRAWorkloadResourceClaims gate)
    - name: shared-device
      resourceClaimTemplateName: my-device-template
---
# PodGroup (runtime scheduling unit)
apiVersion: scheduling.k8s.io/v1beta1
kind: PodGroup
metadata:
  name: my-statefulset-a1b2c3-my-statefulset-d4e5f6
  annotations:
    scheduling.k8s.io/managed-by: statefulset-controller
  ownerReferences:
  - apiVersion: apps/v1
    kind: StatefulSet
    name: my-statefulset
    uid: <statefulset-uid>
    controller: true
    blockOwnerDeletion: true
  # Second ownerRef to the Workload, for the root case only: a parent-owned
  # Workload is never given an ownerReference from the PodGroup.
  - apiVersion: scheduling.k8s.io/v1beta1
    kind: Workload
    name: my-statefulset-a1b2c3
    uid: <workload-uid>
    blockOwnerDeletion: true
  finalizers:
  - scheduling.k8s.io/podgroup-protection
spec:
  podGroupTemplateRef:             # this is what discovery matches on
    workloadName: my-statefulset-a1b2c3
    podGroupTemplateName: my-statefulset
  schedulingPolicy:  # Snapshot from Workload template
    gang:
      minCount: 3
  resourceClaims:              # Snapshot from Workload template
  - name: shared-device
    resourceClaimTemplateName: my-device-template
```

Each pod created by the StatefulSet receives a scheduling group reference and, when
`spec.scheduling.resourceClaims` is configured, matching ResourceClaim entries:

```yaml
# Pod: ownerRef to the StatefulSet (controller) and to the PodGroup, so GC never
# leaves a pod pointing at a PodGroup that is gone.
metadata:
  ownerReferences:
  - apiVersion: apps/v1
    kind: StatefulSet
    name: my-statefulset
    uid: <statefulset-uid>
    controller: true
    blockOwnerDeletion: true
  - apiVersion: scheduling.k8s.io/v1beta1
    kind: PodGroup
    name: my-statefulset-a1b2c3-my-statefulset-d4e5f6
    uid: <podgroup-uid>
spec:
  schedulingGroup:
    podGroupName: my-statefulset-a1b2c3-my-statefulset-d4e5f6
  # Present only when spec.scheduling.resourceClaims is non-empty.
  # Each entry matches a PodGroup claim by name and source, so the
  # ResourceClaim is reserved for the PodGroup, not each individual pod.
  resourceClaims:
  - name: shared-device
    resourceClaimTemplateName: my-device-template
  containers:
  - name: my-container
    resources:
      claims:
      - name: shared-device
```

The controller builds a single-node `WorkloadItem` and compiles it with `workloadbuilder`, exactly
as the Job controller does:

```go
item := &workloadbuilder.WorkloadItem{
    Name: sts.Name,
    Path: field.NewPath("spec", "scheduling"),
    // DefaultConfig supplies anything the user left unset; a nil policy
    // resolves to Basic.
    DefaultConfig: &workloadbuilder.SchedulingConfig{
        Policy: &workloadbuilder.SchedulingPolicy{
            Basic: &workloadbuilder.BasicSchedulingPolicy{},
        },
    },
    // Input carries the user's intent as the versioned building blocks,
    // together with their field paths for error reporting.
    Input: stsutil.WorkloadInput(sts.Spec.Scheduling),
    // Callbacks run against the resolved config after the default/user merge.
    Callbacks: []workloadbuilder.SchedulingConfigFunc{defaultMinCountForStatefulSet(sts)},
}

builder := workloadbuilder.NewBuilder(item, workloadbuilder.BuildOptions{
    Owner:                  controllerRef(sts),
    AllowedPolicies:        []workloadbuilder.PolicyType{workloadbuilder.BasicPolicy, workloadbuilder.GangPolicy},
    AllowedDisruptionModes: []workloadbuilder.DisruptionModeType{workloadbuilder.SingleMode, workloadbuilder.AllMode},
})

workload, err := builder.BuildWorkload()            // *schedulingv1beta1.Workload
podGroup, err := builder.NewPodGroup(sts.Name, sts.Name) // *schedulingv1beta1.PodGroup
```

`BuildWorkload` returns only the `Workload`; the controller creates it first, then instantiates the
single `PodGroup` from its one `PodGroupTemplate` via `NewPodGroup`. On subsequent syncs the
controller uses `NewBuilderFromExistingWorkload` so it recompiles against the already-persisted
template rather than re-deriving it.

The `defaultMinCountForStatefulSet` callback sets `minCount = *sts.Spec.Replicas` when the resolved
policy is gang, clamped to a minimum of 1. A StatefulSet may legitimately be scaled to
`replicas: 0`, but `minCount` is required and must be positive — it is specified as "a positive
integer" on the PodGroup in
[KEP-4671](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling#api)
and as ">= 1 when set" on the building block in
[KEP-6089](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis) —
so `minCount: 0` would be rejected by validation. The PodGroup of a scaled-to-zero StatefulSet
therefore keeps `minCount: 1`, which has no scheduling effect while no pods reference the group,
until the StatefulSet is scaled back up. This matches the clamp the Job integration applies for
suspended Jobs with `parallelism: 0`.

As in [KEP-5547](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/5547-integrate-workload-with-job#defaulting-rules),
the derived value is used controller-side only and is never written back to
`spec.scheduling.schedulingPolicy.gang.minCount`. Persisting it would make a user-supplied
`minCount` indistinguishable from a derived one on later updates — which matters in Beta, when
`minCount` becomes user-configurable and mutable.

For Alpha, a user-supplied `minCount` is rejected at admission, so the callback always applies. In
Beta the callback becomes a default that only fills in an omitted `minCount`.

When `spec.scheduling.resourceClaims` is non-empty (and the `DRAWorkloadResourceClaims` feature
gate is enabled), the controller populates `PodGroupTemplate.resourceClaims` on the Workload
object. The PodGroup snapshots these entries into its own `spec.resourceClaims`. Pods created by
the StatefulSet controller include matching `spec.resourceClaims` entries — same `name` and same
`resourceClaimName` or `resourceClaimTemplateName` — so that the DRA scheduler plugin and
ResourceClaim controller treat these claims as PodGroup-level rather than per-pod.

### StatefulSet with OrderedReady

For StatefulSets using the default `OrderedReady` policy that have `spec.scheduling` configured,
the controller creates a Workload with `Basic` policy — `gang` is the one capability these
StatefulSets cannot have.

It is worth being explicit that this is not a consolation prize. Being restricted to `Basic` costs
an `OrderedReady` StatefulSet only all-or-nothing placement; the rest of the PodGroup surface is
fully available and delivers value on its own from day one:

- `schedulingConstraints.topology` — the pods are still co-located in a single rack or zone
  instance, even though they are created and bound one at a time.
- `resourceClaims` — the pods still share one PodGroup-level DRA claim rather than one claim each.
- Workload-level observability over the group.

The only thing `Basic` changes relative to `Gang` is that pods are admitted individually instead of
as a unit. `disruptionMode: all` is rejected here, since atomic preemption of a group that was
never atomically scheduled is not meaningful.

```yaml
apiVersion: scheduling.k8s.io/v1beta1
kind: Workload
metadata:
  name: my-ordered-statefulset-9f8e7d
  annotations:
    scheduling.k8s.io/managed-by: statefulset-controller
  ownerReferences:
  - apiVersion: apps/v1
    kind: StatefulSet
    name: my-ordered-statefulset
    uid: <statefulset-uid>
    controller: true
    blockOwnerDeletion: true
spec:
  controllerRef:
    apiGroup: apps
    kind: StatefulSet
    name: my-ordered-statefulset
  podGroupTemplates:
  - name: my-ordered-statefulset
    schedulingPolicy:
      basic: {}
```

### RollingUpdate.Partition (deferred to Beta)

**For Alpha, `updateStrategy.rollingUpdate.partition > 0` is rejected when `spec.scheduling` is
set.** A StatefulSet always compiles to exactly one PodGroup, whose name never changes.

An earlier revision of this KEP proposed splitting a partitioned StatefulSet into two PodGroups
(`partition-old` / `partition-new`) sized to the ordinal ranges on either side of the partition.
That approach was dropped for three reasons:

1. **It forces pod recreation purely to reassign a scheduling group.** `spec.schedulingGroup.podGroupName`
   is immutable on pods, so every time the partition moves — which is the entire point of a canary
   rollout — pods would have to be deleted and recreated just to change which PodGroup they belong
   to. Recreating a running stateful pod for a scheduling bookkeeping reason is not an acceptable
   cost, and it is not something the user asked for by shifting a partition.

2. **A partition is not a scheduling boundary.** A `PodGroup` is the unit of *scheduling
   configuration* — quorum size, topology domain, disruption unit, shared claims. A partition is a
   *revision rollout stage*. Conflating the two gives a StatefulSet two half-sized gangs that each
   individually satisfy `minCount` while the application's actual quorum requirement spans both,
   and it makes topology constraints ambiguous (must both halves land in the same rack?).

3. **The single-PodGroup model does not need it.** Pod count does not change during a rolling
   update, so `minCount` does not change either. Pods keep the same `podGroupName` across revisions
   and the group's scheduling configuration is stable throughout.

What still needs design before partitions can be supported (Beta):

- **Gang semantics during a rolling update.** A gang is satisfied at bind time; a rolling update
  deletes and recreates one pod at a time. The recreated pod must rejoin an already-placed gang.
  If the new revision does not fit — larger requests, a different topology domain, a new device
  class — the group can stall part-way through the rollout. Whether the scheduler should treat the
  replacement as an incremental admission against the existing group, or the whole group should be
  re-gang-scheduled, is a KEP-4671/KEP-5710 level question.
- **Canary semantics under a group constraint.** With `disruptionMode: all` or a topology
  constraint, "update only pods ≥ partition" and "the group is one unit" pull in opposite
  directions. The intended interaction needs to be specified rather than inferred.

Until that is settled, rejecting the combination is the honest option: it is a strictly relaxable
restriction, and it avoids shipping semantics in Alpha that we would have to change later.

**Known cost of this decision**: users who rely on partitioned canary rollouts cannot opt a
StatefulSet into WAS in Alpha, and — because `spec.scheduling` is immutable in Alpha — cannot
temporarily opt out to perform one. This is called out in
[Risks and Mitigations](#risks-and-mitigations) and in the Beta graduation criteria.

Note that the underlying tension is not created by partitions: even a plain
`RollingUpdate` with `partition: 0` recreates gang members one at a time. Alpha allows this
(rejecting rolling updates outright would make the feature unusable) and documents the stall risk;
Beta is where the semantics get pinned down.

### Lifecycle Management

#### Initial Creation Lifecycle

When a Parallel StatefulSet is created with gang scheduling enabled:

1. The StatefulSet controller detects `spec.scheduling` is non-nil and that the StatefulSet has no
   pods yet.
2. It discovers, or creates, the `Workload` object (static template with PodGroup template).
3. It discovers, or creates, the `PodGroup` object (runtime unit, instantiated from the Workload's
   template), and proceeds on the synchronous response to that call — there is no wait on the
   informer cache.
4. It creates all pods with `spec.schedulingGroup.podGroupName` pointing to the PodGroup.
5. The scheduler schedules the pods as a single PodGroup.

Object creation order matters only for reference validity — a PodGroup must be able to point at an
existing Workload — not for scheduling correctness, since the scheduler holds a pod that references
a PodGroup it has not seen yet.

```
StatefulSet Created (podManagementPolicy: Parallel, schedulingPolicy.gang)
  │
  ├─→ Controller discovers/creates Workload (static template, includes resourceClaims if configured)
  │
  ├─→ Controller discovers/creates PodGroup (runtime, minCount = replicas, resourceClaims included)
  │
  ├─→ ResourceClaim controller generates ResourceClaims from PodGroup's
  │   ResourceClaimTemplate references (one per template per PodGroup)
  │
  ├─→ Controller creates all Pods simultaneously (sets spec.schedulingGroup.podGroupName;
  │   pods include matching spec.resourceClaims entries from the pod template)
  │
  └─→ Scheduler schedules the pods as a single PodGroup
```

#### Scale Lifecycle

Per [KEP-4671](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling),
`minCount` is mutable on both `PodGroupTemplate` and the standalone `PodGroup`, which significantly
simplifies scaling: a scale is an in-place update, never a PodGroup recreation.

**Scale Up** (e.g., replicas 3 → 5):
1. The controller updates the PodGroup's `schedulingPolicy.gang.minCount` from 3 to 5.
2. The controller creates the 2 new pods with `spec.schedulingGroup.podGroupName` referencing the
   existing PodGroup.
3. The scheduler re-evaluates the gang with the updated minCount.
4. Once all 5 pods are schedulable, the scheduler binds them (existing pods are already bound;
   new pods are bound simultaneously).

**Scale Down** (e.g., replicas 5 → 3):
1. Delete pods with ordinals ≥ 3.
2. Update the PodGroup's `schedulingPolicy.gang.minCount` from 5 to 3.

**Scale to zero**: `replicas: 0` is a normal StatefulSet operation, but a gang `minCount` must be
positive. The controller deletes all pods and leaves the PodGroup in place with `minCount: 1` (the
clamped floor described in
[StatefulSet with Parallel Pod Management](#statefulset-with-parallel-pod-management)) rather than
attempting an invalid `minCount: 0` update or deleting the PodGroup. Since no pods reference it,
the value has no scheduling effect until the StatefulSet is scaled back up, at which point the
normal scale-up path applies.

**Note on Alpha behavior**: The examples above reflect Alpha, where `minCount` is always derived
from `spec.replicas` — scaling replicas automatically updates `minCount` to match. In Beta, when
`minCount` becomes user-configurable, scaling `replicas` will not automatically change a
user-supplied `minCount`. For example, a user may set `replicas: 5` with `minCount: 3` and later
scale to `replicas: 7` while keeping `minCount: 3`.

**ResourceClaims during scaling**: ResourceClaims shared at the PodGroup level are unaffected by
simple scaling operations. Scale-up adds new pods that reference the existing PodGroup and its
already-allocated ResourceClaims — no new ResourceClaims are generated. Scale-down deletes pods
but the PodGroup's ResourceClaims remain allocated as long as the PodGroup exists.

**Note**: There is no operation in this design that recreates the PodGroup. A StatefulSet keeps one
PodGroup, with a stable identity, for its entire lifetime; scaling mutates `minCount` in place
and rolling updates leave the PodGroup untouched. The PodGroup and its generated ResourceClaims are
deleted only when the StatefulSet itself is deleted.

### Opting into Workload-Aware Scheduling

A StatefulSet opts in by setting `spec.scheduling`, following the controller-as-compiler pattern
from [KEP-6089](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis).
Every sub-field is independently optional; the example below sets all four, but each of them is
useful on its own (see the [User Stories](#user-stories)):

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: my-db
spec:
  replicas: 3
  podManagementPolicy: Parallel
  scheduling:
    schedulingPolicy:
      gang: {}          # minCount derived from spec.replicas (3)
    schedulingConstraints:
      topology:
      - key: topology.kubernetes.io/zone   # all 3 replicas in one zone
    disruptionMode:
      all: {}           # preempt the whole StatefulSet or none of it
    resourceClaims:     # optional; requires DRAWorkloadResourceClaims gate
    - name: shared-device
      resourceClaimTemplateName: shared-device-template
  template:
    spec:
      containers:
      - name: db
        image: my-db:latest
        resources:
          claims:
          - name: shared-device
      resourceClaims:
      - name: shared-device
        resourceClaimTemplateName: shared-device-template
```

The minimal opt-in is `scheduling: {}`, which resolves to `basic: {}` with no constraints, no
disruption mode, and no shared claims: the StatefulSet gets a Workload and a PodGroup, and
workload-level observability with them, while keeping today's scheduling behavior exactly. It
remains a creation-time choice in Alpha — because `spec.scheduling` is immutable, a StatefulSet
created with `{}` cannot turn capabilities on later, and one created without the field cannot opt
in at all.

The user sets `gang: {}` — the controller derives `minCount = *spec.replicas`. For Alpha, a
user-supplied `minCount` is rejected at admission (this restriction will be relaxed in Beta).
The Workload and PodGroup the controller creates are named for readability
(`<statefulset-name>-<hash>` and `<workload-name>-<template-name>-<hash>`), but the controller
never relies on those names to find them again — see
[Workload and PodGroup Discovery](#workload-and-podgroup-discovery).

When `resourceClaims` is specified, the controller passes these entries through to the
PodGroupTemplate and PodGroup. The pod template must include matching `spec.resourceClaims`
entries — the controller injects `spec.schedulingGroup.podGroupName` but does **not** auto-inject
`spec.resourceClaims` into the pod template; the user must declare them in the StatefulSet's
`spec.template.spec.resourceClaims` to opt each pod into the shared claim. This mirrors how pods
currently opt into ResourceClaims and keeps the pod template explicit.

### Admission Validation

Validation follows the same three-layer approach used across WAS-integrated controllers
([KEP-5547](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/5547-integrate-workload-with-job)),
with StatefulSet-specific rules for `podManagementPolicy` and partitioned rollouts:

1. **Declarative validation (DV) on the building blocks** — the embedded
   `scheduling.k8s.io/v1alpha3` types carry their own structural rules and immutability markers.
   These apply unchanged and require no StatefulSet-specific code.

2. **Hand-written StatefulSet validation** — cross-cutting rules that span fields DV cannot
   reason about:
   - **OrderedReady + Gang conflict**: Reject if `spec.scheduling.schedulingPolicy.gang` is
     non-nil and `podManagementPolicy` is `OrderedReady` (or unset, since `OrderedReady` is the
     default). This prevents a structurally guaranteed deadlock — OrderedReady waits
     for each pod to become Ready before creating the next, so gang can never be satisfied.
     ```
     spec.scheduling.schedulingPolicy.gang: Forbidden: gang scheduling requires
     podManagementPolicy: Parallel; OrderedReady creates pods sequentially, which can
     never satisfy a gang
     ```
   - **`rollingUpdate.partition` conflict (Alpha)**: Reject if `spec.scheduling` is set and
     `spec.updateStrategy.rollingUpdate.partition` is greater than 0, on both create and update.
     See [RollingUpdate.Partition](#rollingupdatepartition-deferred-to-beta) for why. Because
     `spec.scheduling` is immutable in Alpha, this also means a StatefulSet that opted into WAS
     cannot later start a partitioned rollout; the error is raised on the update that sets
     `partition`.
     ```
     spec.updateStrategy.rollingUpdate.partition: Forbidden: partitioned rollouts are not
     supported for StatefulSets with spec.scheduling set
     ```
   - **User-set `gang.minCount` is forbidden (Alpha)**: If the gang policy carries a non-nil
     `MinCount`, the request is rejected — the value is derived from `spec.replicas` for Alpha.
     This restriction will be relaxed in Beta to allow user-configurable `minCount`.
   - **`podManagementPolicy` immutability**: `podManagementPolicy` is already immutable on
     StatefulSets. This means a user cannot create a StatefulSet with OrderedReady, then later
     flip to Parallel to enable gang — the scheduling decision is locked at creation time.
   - **Pod template must match PodGroup claims**: If `spec.scheduling.resourceClaims` is set,
     validation warns (but does not reject) if the pod template's `spec.resourceClaims` does
     not contain matching entries for each PodGroup-level claim. Without matching entries in
     the pod template, pods will not consume the PodGroup's shared ResourceClaims.

   Note that `spec.scheduling` immutability itself needs no hand-written rule in Alpha: the
   `+k8s:immutable` marker on the whole field covers add, remove, and any in-place change,
   including the basic/gang switch and the `resourceClaims` list. The hand-written policy-freeze
   rule the Job integration needs (`validateJobSchedulingUpdate`) only becomes necessary here when
   the marker is relaxed in Beta.

3. **`workloadbuilder` semantic validation** — the same `WorkloadItem` tree the controller
   compiles is built during validation and checked via `NewBuilder(...).Validate()`, constructed
   with `BuildOptions{DisableDeclarativeValidation: true}` since the API server already ran DV on
   the versioned building blocks. This owns the rules that must stay identical to what the
   controller compiles, including the allow-list checks and the cross-field rule that
   `disruptionMode: all` is not valid with the `basic` policy.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests
to make this code solid enough prior to implement this enhancement.

##### Prerequisite testing updates

Existing StatefulSet controller tests must be verified to pass with the `GenericWorkload` feature gate
both enabled and disabled, ensuring no regressions to current behavior.

##### Unit tests

- `pkg/controller/statefulset`: Coverage for Workload creation, field mapping, lifecycle
  management (create, scale, delete), and error handling.
  - Workload and PodGroup are created before pods when `spec.scheduling` is non-nil
  - Neither Workload nor PodGroup is created when `spec.scheduling` is nil
  - Correct field mapping from StatefulSet fields to Workload/PodGroup fields
  - `scheduling: {}` resolves to a `Basic` PodGroup with no constraints, disruption mode, or claims
  - `disruptionMode` and `schedulingConstraints.topology` are passed through to the PodGroup
    unchanged for both `Basic` and `Gang` policies
  - Pods receive correct `spec.schedulingGroup.podGroupName` references
  - PodGroup minCount updated in-place on replica scaling
  - `minCount` is clamped to 1 when `spec.replicas` is 0: the PodGroup is neither deleted nor
    patched to an invalid `minCount: 0`
  - The derived `minCount` is never written back to the StatefulSet —
    `spec.scheduling.schedulingPolicy.gang.minCount` is still nil after a sync
  - PodGroup name and identity are stable across a rolling update (no recreation)
  - Workload and PodGroup cleanup on StatefulSet deletion
  - OwnerReferences and finalizers are set correctly on all three object kinds: Workload →
    StatefulSet; PodGroup → StatefulSet and (root case only) Workload; Pod → StatefulSet and
    PodGroup
  - Discovery by reference: an existing Workload is matched by `spec.controllerRef` and an
    existing PodGroup by `spec.podGroupTemplateRef`, regardless of their names
  - No duplicate creation when a Workload exists but its PodGroup does not (crash-recovery path)
  - No creation attempted once the StatefulSet already has pods; existing objects are discovered
    and used
  - Two Workloads matching the same StatefulSet are treated as ambiguous: nothing is created or
    mutated and an event is emitted
  - A Workload or PodGroup not created by the controller is used as-is: no ownerReference added,
    no mutation, no deletion
  - A discovered Workload with a `podGroupTemplates` count other than 1 is ignored, with an event
  - Composition: with a parent ownerReference and `scheduling.k8s.io/group-template-name` set, the
    controller creates no Workload and instantiates its PodGroup from the parent's named template
  - Composition: with `scheduling.k8s.io/parent-compositepodgroup` also set, the created PodGroup
    links to that parent CompositePodGroup instance
  - Composition: with a parent ownerReference and no `group-template-name` annotation, the
    controller creates neither object and only stamps `podGroupName` onto pods
  - ResourceClaims from `spec.scheduling.resourceClaims` are passed through to PodGroupTemplate
    and PodGroup `spec.resourceClaims`
  - Pods receive matching `spec.resourceClaims` entries when the pod template includes them
  - ResourceClaims field is omitted from PodGroupTemplate when `DRAWorkloadResourceClaims` gate
    is disabled
  - ResourceClaims field is omitted when `spec.scheduling.resourceClaims` is empty

- `pkg/apis/apps/validation`: Coverage for admission validation rules.
  - Reject OrderedReady + `scheduling.schedulingPolicy.gang` combination
  - Reject user-set `gang.minCount` (Alpha only — must be nil, derived from replicas)
  - Reject `rollingUpdate.partition > 0` together with `spec.scheduling`, on create and on update
  - Reject `disruptionMode: all` with the `basic` policy (delegated to `workloadbuilder`)
  - Accept valid Parallel + `scheduling.schedulingPolicy.gang` combinations
  - Accept OrderedReady + `scheduling.schedulingPolicy.basic`, including with topology constraints
    and shared ResourceClaims
  - Accept `scheduling: {}`
  - Reject `spec.scheduling` mutations of any kind (whole-field immutability), including
    add-after-create, unset, basic↔gang switch, and `resourceClaims` list edits
  - Accept valid `spec.scheduling.resourceClaims` entries (one of name/template set)
  - Reject `spec.scheduling.resourceClaims` entries with both or neither source set
  - Reject more than 4 entries in `spec.scheduling.resourceClaims`

##### Integration tests

- StatefulSet controller creates Workload and PodGroup before pods; pods reference PodGroup correctly
- Gang StatefulSet: no pod is bound until the whole group can be placed; all are bound once it can
- `disruptionMode: all`: a higher-priority pod preempts the whole StatefulSet rather than a subset
- `schedulingConstraints.topology`: all pods land in a single instance of the named domain
- Scaling updates PodGroup minCount in-place (no recreation needed)
- Rolling update (`partition: 0`) keeps the same PodGroup; pods are recreated with the same
  `podGroupName`
- StatefulSet deletion cascades to Workload, PodGroup, and Pod deletion via OwnerReference
- Controller restart between Workload creation and PodGroup creation: the next sync discovers the
  Workload and creates only the missing PodGroup — no duplicate Workload
- A StatefulSet composed by a parent controller (parent-owned Workload + downward-mapping
  annotations) gets a PodGroup attached to the parent's CompositePodGroup, and the StatefulSet
  controller creates no Workload of its own
- Feature gate disabled: no Workload/PodGroup objects created, standard behavior preserved
- ResourceClaims from `spec.scheduling.resourceClaims` propagated to PodGroup and shared by all pods
- ResourceClaimTemplate generates one ResourceClaim per PodGroup, not per pod
- ResourceClaim is reserved for PodGroup (not individual pods) in `status.reservedFor`
- PodGroup deletion cascades to generated ResourceClaim deletion

##### e2e tests

- End-to-end all-or-nothing scheduling of a Parallel StatefulSet
- End-to-end gang preemption with `disruptionMode: all`
- End-to-end topology co-location with `schedulingConstraints.topology`
- Scale-up with in-place PodGroup minCount update: verify new pods join existing PodGroup
- Scale-down with in-place PodGroup minCount update: verify correct pod termination and PodGroup
  minCount reduced to match new replica count
- Scale to zero and back up: all pods are removed, the PodGroup survives with `minCount: 1`, and
  the gang re-forms on scale-up against the same PodGroup
- OrderedReady StatefulSet with Basic Workload: verify sequential scheduling preserved
- Failure scenario: insufficient resources prevent gang formation; no pod of the group is bound
- StatefulSet deletion: verify Workload, PodGroup, and all pods cleaned up
- Feature gate toggle: verify behavior change on enable/disable
- End-to-end ResourceClaim sharing: StatefulSet with `resourceClaims` creates PodGroup with
  shared claim; all pods reference same allocated ResourceClaim
- ResourceClaimTemplate generates one claim per PodGroup; scaling up adds pods referencing the
  existing claim without generating new claims

### Graduation Criteria

#### Alpha

- Feature implemented behind the `WorkloadWithStatefulSet` feature flag (depends on
  `GenericWorkload` feature gate)
- StatefulSet controller compiles `spec.scheduling` into one Workload and one PodGroup, covering
  all four capabilities: `schedulingPolicy`, `schedulingConstraints`, `disruptionMode`,
  `resourceClaims`
- Reference-based discovery of the Workload and PodGroup, matching the Job integration, so that a
  controller restart mid-workflow never produces duplicates
- Support for composition by higher-level controllers via the KEP-6089 downward-mapping
  annotations (parent-owned Workload, optionally parent-owned PodGroup)
- Basic lifecycle management: create, delete, scale (in-place PodGroup minCount update)
- Admission validation for the OrderedReady + Gang conflict and the `partition` + `scheduling`
  conflict
- Initial unit and integration tests completed and enabled
- Documentation of the feature and its Alpha limitations (immutable `spec.scheduling`, no
  partitioned rollouts, derived `minCount`)

#### Beta

- Gather feedback from early adopters, in particular on rolling updates of gang-scheduled
  StatefulSets and on demand for partitioned rollouts
- Support user-configurable `gang.minCount` to allow partial-gang semantics
  (`minCount < replicas`), enabling use cases like quorum-based systems that can start with a
  subset of replicas; relax the field-level immutability accordingly
- Specify and implement rolling-update semantics under a gang policy (how a replacement pod
  rejoins an already-placed group, and what happens when the new revision does not fit)
- On that basis, decide and implement `rollingUpdate.partition` support — without splitting a
  StatefulSet across multiple PodGroups and without recreating pods to reassign scheduling groups
- Controller crash-recovery edge cases addressed
- Metrics for Workload/PodGroup creation and update latency
- E2e tests in CI, linked in TestGrid
- Performance testing to verify acceptable scheduler overhead with StatefulSet Workloads
- Documentation updated with best practices for common stateful workloads (databases,
  coordination services)

#### GA

- At least 2 releases in Beta with no critical bugs
- Real-world usage validation from distributed database and stateful AI/ML operators
- Conformance tests covering core Workload-Aware Scheduling behavior for StatefulSets
- All open questions resolved (rolling updates under gang, partitioned rollouts)
- Stable metrics and monitoring documentation

### Upgrade / Downgrade Strategy

With the gate disabled or `spec.scheduling` unset, StatefulSets behave exactly as they do today.

The StatefulSet controller owns Workload and PodGroup objects directly. This means:

- **On downgrade**: Existing Workload and PodGroup objects remain in the cluster, protected by
  their `podgroup-protection` finalizers. The StatefulSet controller stops reconciling them, but
  they are **not** inert from the scheduler's point of view: as long as `GenericWorkload` is
  enabled in kube-scheduler and pods reference the PodGroups, the scheduler keeps enforcing their
  gang, topology, and disruption semantics. They are cleaned up only when the owning StatefulSet is
  deleted. Cluster admins who want the semantics to actually stop must delete the StatefulSet or
  manually remove the PodGroup — see the desynchronization warning under
  [Can the feature be disabled...](#can-the-feature-be-disabled-once-it-has-been-enabled-ie-can-we-roll-back-the-enablement).
- **Pod-level references**: Pods created while the gate was enabled carry
  `spec.schedulingGroup.podGroupName`, and the scheduler continues to honor it while the referenced
  PodGroup exists and kube-scheduler still has `GenericWorkload`. Pods created after downgrade will
  not carry the field, so a StatefulSet can end up with a mix of pods inside and outside its
  PodGroup.
- **PVCs are unaffected**: `volumeClaimTemplates`-managed PVCs are independent of the scheduling
  objects and are retained per the StatefulSet's `persistentVolumeClaimRetentionPolicy`.
- **ResourceClaims**: ResourceClaims generated from ResourceClaimTemplates for PodGroups are
  owned by the PodGroup via `ownerReferences`. If the PodGroup persists after downgrade (as
  described above), its ResourceClaims also persist. If the `DRAWorkloadResourceClaims` gate
  is disabled independently, the ResourceClaim controller stops managing PodGroup-level claims,
  but already-allocated claims remain functional for running pods. See KEP-5729 for detailed
  downgrade behavior of PodGroup-level ResourceClaims.

### Version Skew Strategy

The feature requires `GenericWorkload` and the `scheduling.k8s.io/v1beta1` API to be served by
the API server. Two skew scenarios are relevant:

1. **API server lacks the scheduling API**: The controller's create/patch calls for Workload and
   PodGroup return 404. The StatefulSet sync retries with backoff, but pod creation is blocked
   for gang-configured StatefulSets (creating pods without a PodGroup would bypass gang
   guarantees). This resolves once the API server is upgraded.

2. **Scheduler lacks `GenericWorkload`**: The controller creates Workload and PodGroup objects
   successfully, and pods carry `schedulingGroup.podGroupName`, but the scheduler ignores the
   gang semantics. Pods are scheduled individually — equivalent to pre-feature behavior. This is
   a degraded-but-safe mode; it does not cause failures.

The StatefulSet directly owns the Workload and PodGroup — scheduling objects have no independent
lifecycle and exist exactly as long as the StatefulSet does.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `WorkloadWithStatefulSet`
  - Components depending on the feature gate:
    - kube-controller-manager
    - kube-apiserver

###### Does enabling the feature change any default behavior?

No. The feature is opt-in: only StatefulSets with an explicit `spec.scheduling` field produce
scheduling objects. Existing StatefulSets — including those with `podManagementPolicy: Parallel` —
are completely unaffected. There is no heuristic or automatic opt-in.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, with an important caveat about what "disabled" means for objects that already exist.

- **kube-apiserver**: Clears `spec.scheduling` on new creates; preserves the stored value on
  existing objects (standard alpha field-gating).
- **kube-controller-manager**: Stops compiling Workload and PodGroup objects for StatefulSets. It
  neither deletes nor updates the ones it already created.

**Existing PodGroups do not become inert.** `WorkloadWithStatefulSet` gates the *controller-side
compilation*, not the scheduler. As long as `GenericWorkload` is enabled in kube-scheduler, any
`PodGroup` still present in the cluster continues to be enforced for every pod that references it
via `spec.schedulingGroup.podGroupName` — gang, topology constraints, and disruption mode all keep
applying. Disabling the gate therefore does not roll back the *behavior*; it only stops the
StatefulSet controller from keeping the objects in sync, which introduces a real desynchronization
hazard:

- Scaling `spec.replicas` no longer updates the PodGroup's `minCount`. Scaling a gang-scheduled
  StatefulSet from 3 to 5 leaves `minCount: 3`, so the 2 new pods are scheduled against a group
  whose declared size no longer matches reality; scaling down to 2 leaves `minCount: 3`, which can
  never be satisfied by the remaining pods.
- New pods created after rollback are created *without* `schedulingGroup`, so a single StatefulSet
  can end up with some pods inside its PodGroup and some outside it.

To genuinely roll the behavior back, an operator must additionally remove the PodGroup objects
(deleting the StatefulSet, or deleting the PodGroups directly), or disable `GenericWorkload` in
kube-scheduler. Running pods are never terminated by any of this — they continue executing
regardless of whether their `schedulingGroup` reference resolves.

###### What happens if we reenable the feature if it was previously rolled back?

On reenable, the controller resumes compilation for any StatefulSet whose stored
`spec.scheduling` survived the rollback (the API server preserves already-set values even with
the gate off). Specifically:

- If the Workload and PodGroup still exist (they were not manually deleted), the controller
  discovers them by reference — `workload.spec.controllerRef` and
  `podGroup.spec.podGroupTemplateRef` — and reuses them. This works regardless of how they were
  named.
- If only a partial set exists (e.g., Workload but no PodGroup from a crash during the original
  enablement), the controller creates the missing object on its next sync.
- StatefulSets with `OrderedReady` and `spec.scheduling` (non-gang, Basic policy) also resume
  compilation — they produce a Workload with `basic: {}` for observability.

###### Are there any tests for feature enablement/disablement?

Yes — integration tests cover:
- Gate disabled: no Workload/PodGroup objects created, even for StatefulSets with `spec.scheduling`.
- Gate enabled: Workload and PodGroup created for qualifying StatefulSets (both Parallel+Gang and
  OrderedReady+Basic configurations).
- Gate toggled off then on: existing scheduling objects discovered and reused; no duplicates.
- Gate toggled off: PodGroup `minCount` is no longer updated on scale, and the pre-existing
  PodGroup is still honored by the scheduler (the documented desynchronization behavior above).
- Running pods unaffected by gate toggle.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout can fail in two ways:

- **API server does not serve `scheduling.k8s.io`**: The controller fails to create Workload and
  PodGroup objects. For gang-configured StatefulSets, pod creation is blocked (pods must not be
  created without a PodGroup, or gang guarantees are lost). The controller retries with backoff.
  StatefulSets without `spec.scheduling` are unaffected.
- **Scheduler lacks `GenericWorkload`**: Workloads and PodGroups are created successfully, but
  the scheduler ignores them. Pods are scheduled individually — equivalent to pre-feature
  behavior. This is degraded but safe.

Rollback impact: Running pods are unaffected — StatefulSet pods with stable identities continue
executing. Workload and PodGroup objects persist in the cluster until the StatefulSet is deleted,
and continue to be enforced by the scheduler for the pods that reference them; only the
controller's syncing of those objects stops. New pods created after rollback lack
`schedulingGroup` and schedule individually, so a StatefulSet can end up split across
group-scheduled and individually-scheduled pods, and its PodGroup's `minCount` can drift from
`spec.replicas`. For stateful systems with quorum requirements, operators should verify all members
are running before rolling back, and should delete the PodGroups if they want the group semantics
to actually stop.

###### What specific metrics should inform a rollback?

Two classes of signal should drive the decision to disable `WorkloadWithStatefulSet`:

- **The controller cannot produce scheduling objects**:
  `statefulset_workload_creation_errors_total` rising and not draining. For gang-configured
  StatefulSets this blocks pod creation entirely, so it is the one symptom that stalls a rollout
  rather than merely degrading it.
- **Groups are created but cannot be placed**: the KEP-4671 scheduler metrics —
  `scheduler_podgroup_schedule_attempts_total` with `result=unschedulable` growing for PodGroups
  owned by StatefulSets, or `scheduler_podgroup_scheduling_attempt_duration_seconds` regressing
  against the pre-enablement baseline — indicating the gang or topology constraint is tighter than
  the cluster can satisfy.

One signal that should *not* lead to a rollback: a PodGroup that exists while
`scheduler_podgroup_schedule_attempts_total` records no attempts for it. That means the scheduler is
not enforcing the group at all (missing `GenericWorkload`), so the fix is to enable the gate, not to
disable this feature.

Rolling back does not by itself stop the behavior being rolled back — as described above, existing
PodGroups keep being enforced until they are deleted or `GenericWorkload` is disabled in the
scheduler.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested as part of Beta graduation criteria.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- Check for `Workload` objects with `spec.controllerRef.kind: StatefulSet`:
  ```
  kubectl get workloads -A -o json | jq '.items[] | select(.spec.controllerRef.kind == "StatefulSet")'
  ```

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `WorkloadCreated` — emitted on the StatefulSet when a Workload is successfully
    created.
  - Event Reason: `PodGroupCreated` — emitted when the runtime PodGroup is successfully created.
  - Event Reason: `WorkloadCreationFailed` — emitted when Workload or PodGroup creation fails.
- [x] API .status
  - Condition name: `PodGroupInitiallyScheduled`, on the PodGroup, defined by
    [KEP-4671](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling#api).
    `True` means the group was placed; a reason of `Unschedulable` names why it was not. This is
    where the feature's outcome is observable, and the StatefulSet's owning reference leads to it.
  - This KEP adds no condition to `StatefulSet.status` in Alpha. The `status.conditions` field
    exists on StatefulSet but has never carried any condition type, so populating it is an API
    surface change in its own right — worth considering for Beta, once there is feedback on whether
    following the ownership chain to the PodGroup is too indirect in practice.
- [x] Other
  - The `Workload` object itself is visible via `kubectl get workloads` and shows PodGroup status.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No numeric SLO is proposed for Alpha. The feature adds no new serving path — it adds object
compilation to the StatefulSet sync loop and hands placement to the scheduler's existing PodGroup
handling — so the objectives that matter are regression bounds rather than new guarantees:

- StatefulSets *without* `spec.scheduling` see no change in controller sync latency; the compile
  step is skipped entirely for them.
- For StatefulSets *with* `spec.scheduling`, the added work is one discover-then-create of two
  objects per StatefulSet, not per replica, and it happens once rather than on every sync.
- Gang placement does not regress time-to-all-pods-running relative to individual scheduling for a
  StatefulSet that fits in the cluster. When it does not fit, the group deliberately waits, so no
  time-based objective applies.

Numbers will be set at Beta, once the creation and update latency metrics named in the
[Beta](#beta) graduation criteria exist and performance testing has established a baseline.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `statefulset_workload_creation_errors_total`
    - Aggregation method: rate, over a sustained window — a flat counter is healthy. A non-zero rate
      means StatefulSets with `spec.scheduling` are not getting their objects, and gang-configured
      ones are not creating pods at all.
    - Components exposing the metric: kube-controller-manager
  - Metric name: `scheduler_podgroup_schedule_attempts_total` (KEP-4671)
    - Aggregation method: rate by `result`, for PodGroups owned by StatefulSets. Healthy means
      non-zero and dominated by `scheduled`. A sustained `unschedulable` rate means the group does
      not fit; *no attempts at all* means the scheduler is not enforcing the group, which is a
      version-skew problem rather than a capacity one (see
      [Version Skew Strategy](#version-skew-strategy)).
    - Components exposing the metric: kube-scheduler
  - Metric name: `scheduler_podgroup_scheduling_attempt_duration_seconds` (KEP-4671)
    - Aggregation method: histogram quantiles, compared against the same cluster's pod-level
      scheduling latency
    - Components exposing the metric: kube-scheduler
- [x] Other
  - Per-object, the `PodGroupInitiallyScheduled` condition on the PodGroup ([KEP-4671](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling#api))
    answers "did this StatefulSet's group ever get placed", and a reason of `Unschedulable` names
    why it did not. This is the signal an application owner can read without cluster-wide metrics
    access.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Yes. Alpha ships only the error counter above; the following are proposed for Beta, matching the
[Beta](#beta) graduation criteria:

- Latency from a StatefulSet becoming eligible (created or updated with `spec.scheduling`) to its
  Workload and PodGroup existing. This is the one number the SLOs above are currently unable to
  state, and it is what would show compilation falling behind under load.
- A gauge of StatefulSet-owned Workloads, split by resolved policy (gang vs basic), so adoption and
  the mix of policies are visible without listing objects.

Not needed: a metric for "pods blocked awaiting a PodGroup". That state is already visible as
Pending pods with the existing scheduler queue metrics, and the blocking itself is intentional.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- **Workload API (`scheduling.k8s.io/v1beta1`)**: The built-in `Workload` and `PodGroup` types must
  be served by the API server. This is provided by KEP-4671 and gated behind the `GenericWorkload`
  feature gate. The `scheduling.k8s.io/v1alpha3` building-block types must also be available, since
  `StatefulSetSpec` embeds them.
- **kube-scheduler with `GenericWorkload` enabled**: The scheduler must be able to act on PodGroups.
  Without it, Workloads and PodGroups are created but have no effect on scheduling.
- **DRAWorkloadResourceClaims feature gate** (optional, for ResourceClaim support): The
  `DRAWorkloadResourceClaims` feature gate (KEP-5729) must be enabled in kube-apiserver,
  kube-controller-manager, kube-scheduler, and kubelet for the `resourceClaims` field on
  `StatefulSetSchedulingConfiguration` to take effect. Without it, the field is ignored and
  ResourceClaims are not shared at the PodGroup level.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. For each StatefulSet with `spec.scheduling` set, the controller uses informers for Workload
and PodGroup objects and makes the following API calls:
- `CREATE Workload` — 1 per StatefulSet creation
- `CREATE PodGroup` — 1 per StatefulSet creation
- `PATCH PodGroup` — on scale (minCount update)

There is no PodGroup recreation path: the PodGroup is created once and mutated in place for the
lifetime of the StatefulSet.

###### Will enabling / using this feature result in introducing new API types?

No—the `Workload` type is introduced by KEP-4671. This KEP only creates instances of it.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

- One additional `Workload` object + one `PodGroup` object per StatefulSet that opts in (small
  objects, ~1-2 KB each).
- Each pod gains a `spec.schedulingGroup` field (~50 bytes of additional pod spec).
- When `resourceClaims` is configured: up to 4 additional `PodGroupResourceClaim` entries on the
  Workload and PodGroup (~100 bytes each), plus one generated `ResourceClaim` object per
  ResourceClaimTemplate reference per PodGroup (~500 bytes each).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

There is an expected increase in StatefulSet sync duration due to creating Workload and PodGroup
objects. Pod start-up latency for a gang-scheduled StatefulSet can increase when resources are
constrained, since no pod is bound until the whole group can be placed — that is the point of the
feature, not a regression. Impact to be measured during Alpha.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Minimal (opt-in only). Per-StatefulSet overhead is 2 additional small objects in etcd (~500 bytes
each), informer watches for Workload and PodGroup types, and CREATE API calls per new StatefulSet.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature operates entirely at the control plane level and does not affect node resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The StatefulSet controller cannot create Workload objects. StatefulSet reconciliation will retry
with standard exponential backoff. No pods are created until the Workload is successfully created
(for StatefulSets with gang scheduling enabled). This is the correct behavior—creating pods
without a Workload would bypass gang scheduling guarantees.

###### What are other known failure modes?

| Failure Mode | Description | Detection | Mitigation |
|-------------|-------------|-----------|------------|
| Workload creation fails | API server rejects Workload (e.g., quota, webhook) | `WorkloadCreationFailed` event on StatefulSet; `statefulset_workload_creation_errors_total` metric | Controller retries with backoff; admin resolves underlying issue |
| Workload exists but scheduler doesn't process it | Scheduler feature gate disabled or plugin not loaded | Pods are scheduled individually instead of as a group — they do *not* stay Pending, so the loss of the gang guarantee is silent. The signal is a PodGroup that exists while `scheduler_podgroup_schedule_attempts_total` records no attempts for it, and pods binding one at a time | Enable `GenericWorkload` feature gate in scheduler |
| Gang cannot be satisfied | Cluster lacks resources for MinCount pods | Pods stay Pending; PodGroup status reports the unsatisfied group | Scale cluster or reduce replicas (which reduces MinCount in Alpha) |
| Stale Workload after controller crash | Controller crashed between creating Workload and creating pods | Workload exists but no pods reference it | Controller reconciliation detects and resolves on restart |
| PodGroup out of sync with replicas | `WorkloadWithStatefulSet` was disabled in kube-controller-manager while PodGroups still exist and are enforced by the scheduler | PodGroup `minCount` differs from `spec.replicas`; pods created after the rollback lack `spec.schedulingGroup` | Re-enable the gate so the controller resumes syncing, or delete the PodGroups to stop enforcement |

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `statefulset_workload_creation_errors_total` for persistent Workload creation failures.
2. Verify both `WorkloadWithStatefulSet` and `GenericWorkload` feature gates are enabled.
3. Inspect the Workload and PodGroup: `kubectl get workloads,podgroups -n <namespace>` — verify
   they exist, that `minCount` matches `spec.replicas`, and check the PodGroup's status conditions.
4. Check the KEP-4671 scheduler metrics (`scheduler_podgroup_schedule_attempts_total`,
   `scheduler_podgroup_scheduling_attempt_duration_seconds`) for group-level scheduling delays.
5. Check scheduler logs for messages about the PodGroup.
6. If the issue is resource-related, check node capacity vs. the group's aggregate requirements
   (and, with a topology constraint, the capacity of a single domain).

## Implementation History

- 2026-08-25: Initial KEP draft created (OCPNODE-4667)
- 2026-09-21: Addressed review comments — adopted KEP-5547's ownership and discovery model,
  deferred `rollingUpdate.partition` to Beta, reworked the Risks and Mitigations table, corrected
  the `minCount` and `disruptionMode` semantics, and filled in the previously TBD PRR monitoring
  answers.

## Drawbacks

1. **No partitioned rollouts in Alpha**: A StatefulSet with `spec.scheduling` set cannot use
   `rollingUpdate.partition`, and — because `spec.scheduling` is immutable in Alpha — cannot opt
   out to perform one. Users who depend on partitioned canary rollouts must wait for Beta.

2. **Rolling updates under a gang policy are not fully specified in Alpha**: pods are recreated one
   at a time, and a replacement that no longer fits can stall the group part-way through a
   rollout.

3. **Increased complexity in the StatefulSet controller**: The controller gains significant new
   responsibility for Workload and PodGroup lifecycle management, increasing its code complexity
   and the surface area for bugs. The `workloadbuilder` library mitigates this by encapsulating
   common patterns.

4. **OrderedReady limitation**: The most common StatefulSet configuration (OrderedReady, which is
   the default) cannot use gang scheduling. Users must explicitly switch to Parallel policy,
   which changes pod startup behavior and may not be suitable for all applications. The other WAS
   capabilities remain available to OrderedReady StatefulSets under the `Basic` policy.

5. **Additional API objects**: Each opted-in StatefulSet creates an additional Workload and
   PodGroup object, increasing the total object count in the cluster and adding to API server and
   etcd load.

## Alternatives

### Alternative 1: Explicit gangScheduling API Field on StatefulSet

Add a dedicated `gangScheduling` field to the StatefulSet spec:

```yaml
apiVersion: apps/v1
kind: StatefulSet
spec:
  gangScheduling:
    enabled: true
    reschedulingPolicy: TerminateAll
```

**Pros:**
- Explicit user intent with no ambiguity
- Full schema validation at admission time with workload-specific constraints
- Discoverable via `kubectl explain statefulset.spec.gangScheduling`

**Cons:**
- Requires API changes to `apps/v1`, which has a high bar for new fields
- Each workload API evolves independently (version skew between apps/v1 and scheduling API)
- Duplicates field definition across multiple APIs (Job, StatefulSet, Deployment, etc.)

This was not chosen because the `spec.scheduling` approach (using reusable `scheduling.k8s.io`
building blocks) provides a consistent pattern across all workload controllers (Job, Deployment,
StatefulSet) without duplicating field definitions in each controller's API.

### Alternative 2: Create Workload by Default for All StatefulSets

Automatically create a Workload for every StatefulSet instance, using Basic policy by default and
Gang policy when heuristics indicate it's appropriate (e.g., Parallel + replicas > 1).

**Pros:**
- Uniform model: every workload has a Workload object
- Consistent observability across all workloads
- No user action required

**Cons:**
- Resource overhead for StatefulSets that don't need scheduling coordination
- Scheduler processes more Workload objects
- Users may not understand why gang scheduling is enabled
- Heuristics may incorrectly enable gang for independent workloads
- Existing clusters accumulate Workload objects for all StatefulSets
- Difficult to opt out when the heuristic is wrong

This was not chosen because the opt-in approach avoids resource waste and incorrect heuristic
decisions, while allowing users who need gang scheduling to explicitly request it.

## Infrastructure Needed

No additional infrastructure is needed beyond what is provided by KEP-4671 (Workload API).

## References

- [KEP-4671: Gang Scheduling](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling) — Workload API definition and scheduler gang scheduling plugin
- [KEP-5547: Integrate Workload with Job](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/5547-integrate-workload-with-job) — Reference implementation for Job controller integration
- [KEP-5832: Decouple PodGroup API](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5832-decouple-podgroup-api) — Standalone PodGroup as the runtime scheduling unit
- [KEP-6089: WAS Controller APIs](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis) — Reusable building blocks and `workloadbuilder` library
- [KEP-5710: Workload-Aware Preemption](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5710-workload-aware-preemption)
- [KEP-5729: DRA ResourceClaim Support for Workloads](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5729-resourceclaim-support-for-workloads) — PodGroup-level ResourceClaim sharing
- [KEP-5732: Topology-Aware Workload Scheduling](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5732-topology-aware-workload-scheduling) — `schedulingConstraints.topology`
- [KEP-6012: Composite PodGroup API](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6012-composite-podgroup-api) — group-of-groups; not exposed on StatefulSet
