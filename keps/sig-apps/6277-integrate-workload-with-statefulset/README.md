# KEP-6277: Workload API Integration with StatefulSet

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Distributed Database with Quorum Requirements](#story-1-distributed-database-with-quorum-requirements)
    - [Story 2: Distributed Coordination Service](#story-2-distributed-coordination-service)
    - [Story 3: Stateful AI/ML Inference Serving](#story-3-stateful-aiml-inference-serving)
    - [Story 4: Topology-Aware GPU Placement for Distributed Training](#story-4-topology-aware-gpu-placement-for-distributed-training)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Workload Ownership Model](#workload-ownership-model)
  - [Pod Management Policy Constraints](#pod-management-policy-constraints)
  - [Field Mapping](#field-mapping)
  - [StatefulSet with Parallel Pod Management](#statefulset-with-parallel-pod-management)
  - [StatefulSet with OrderedReady](#statefulset-with-orderedready)
  - [StatefulSet with RollingUpdate.Partition](#statefulset-with-rollingupdatepartition)
  - [Lifecycle Management](#lifecycle-management)
    - [Initial Creation Lifecycle](#initial-creation-lifecycle)
    - [Scale Lifecycle](#scale-lifecycle)
  - [Enabling Gang Scheduling](#enabling-gang-scheduling)
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

This KEP proposes integrating the Kubernetes Workload API (`scheduling.k8s.io/v1alpha3`) with the
StatefulSet controller to enable gang scheduling for stateful workloads. The StatefulSet controller
will be extended to automatically create and manage `Workload` and `PodGroup` objects that describe
pod groups and their scheduling requirements, allowing the kube-scheduler to make gang scheduling
decisions for StatefulSet pods.

Gang scheduling ensures that all pods in a group are scheduled atomically—either all pods are placed
or none are—preventing partial deployments that could cause split-brain scenarios, inconsistent
quorum states, or resource waste in distributed stateful systems such as databases, coordination
services, and stateful AI/ML serving infrastructure.

This integration follows the patterns established by the Job controller integration ([KEP-5547](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/5547-integrate-workload-with-job))
and uses the reusable `workloadbuilder` library and `scheduling.k8s.io/v1alpha3` building blocks from ([KEP-6089](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis)).
A new `scheduling` field is added to `StatefulSetSpec` to expose scheduling policy configuration.

Gang scheduling requires `podManagementPolicy: Parallel`, as the default `OrderedReady` policy's
sequential pod creation fundamentally conflicts with gang scheduling's all-at-once model.
StatefulSets using `OrderedReady` will receive a Workload with `Basic` (non-gang) policy,
preserving existing behavior while still enabling workload-level observability.

Additionally, this KEP integrates ResourceClaim support for StatefulSet PodGroups, building on
[KEP-5729](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5729-resourceclaim-support-for-workloads).
A `resourceClaims` field on `StatefulSetSchedulingConfiguration` allows users to specify
ResourceClaims or ResourceClaimTemplates that are shared across all pods in a StatefulSet's
PodGroup. This enables use cases such as topology-aware GPU placement, where all pods in a
StatefulSet must share the same topological domain device without requiring manual ResourceClaim
management.

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

The Workload API was introduced in
[KEP-4671](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/4671-gang-scheduling)
to provide a standardized mechanism for describing groups of pods that must be scheduled together.
[KEP-5832](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5832-decouple-podgroup-api)
decoupled `PodGroup` as a standalone runtime object, and
[KEP-6089](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis)
introduced reusable `scheduling.k8s.io/v1alpha3` building blocks and a shared `workloadbuilder`
library for controller integration. Together these provide the `Workload` (static template),
`PodGroup` (runtime scheduling unit), and scheduling policies (`Gang` and `Basic`) that this KEP
consumes.

The [Workload API Integration Design](https://docs.google.com/document/d/1-lBmKIlgcCggjBWYO0VkZ8t3NmvJp-LCsfP6IB0XKRY)
document (by Heba Elayoty and Andrey Velichkevich) provides the cross-controller integration
blueprint. This KEP focuses specifically on the StatefulSet controller integration, following the
patterns established by the Job controller integration in KEP-5547 and aligning with the broader
effort to integrate all true workload controllers with the Workload API.

### Goals

- Enable the StatefulSet controller to automatically create and manage `Workload` and `PodGroup`
  objects when gang scheduling is requested via a new `scheduling` field on `StatefulSetSpec`.

- Support gang scheduling for StatefulSets using `podManagementPolicy: Parallel`, where all replica
  pods must be schedulable before any are bound to nodes.

- Provide a `Basic` (non-gang) Workload policy for StatefulSets using `podManagementPolicy:
  OrderedReady`, enabling workload-level observability without altering their sequential scheduling
  behavior.

- Handle the StatefulSet lifecycle events correctly—including initial creation, scaling (up/down),
  and `RollingUpdate` with `partition`—with proper Workload creation, recreation, and cleanup.

- Follow the ownership model defined in the Workload API: one Workload and one PodGroup per
  StatefulSet, both owned via `ownerReference` with `controller=true` and
  `blockOwnerDeletion=true`.

- Populate `pod.spec.schedulingGroup.podGroupName` on all pods created by the StatefulSet
  controller, linking each pod to its PodGroup for scheduler consumption.

- Use the shared `workloadbuilder` library from KEP-6089 for Workload and PodGroup construction,
  ensuring consistency with other controller integrations (Job, JobSet).

- Enable per-PodGroup ResourceClaim sharing via the `resourceClaims` field on
  `StatefulSetSchedulingConfiguration`, allowing users to specify ResourceClaims or
  ResourceClaimTemplates that are shared across all pods in the StatefulSet's PodGroup. The
  controller populates `PodGroupTemplate.resourceClaims` and creates pods with matching
  `spec.resourceClaims` entries so that the ResourceClaim controller and scheduler handle
  allocation and reservation at the PodGroup level (per
  [KEP-5729](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5729-resourceclaim-support-for-workloads)).

### Non-Goals

- Modifying the `OrderedReady` pod management policy to support gang scheduling. The sequential
  creation semantics of `OrderedReady` are fundamentally incompatible with gang scheduling, and
  changing them is out of scope.

- Introducing changes to the Workload API itself (`scheduling.k8s.io`). This KEP consumes the
  API as defined in KEP-4671; any API changes are tracked separately.

- Supporting cross-namespace or cross-controller Workload ownership. The Workload is always
  co-located in the same namespace as the StatefulSet and owned exclusively by it.

- Implementing workload-aware preemption for StatefulSets. Preemption behavior is handled by
  [KEP-5710](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5710-workload-aware-preemption)
  and applies uniformly to all Workload-aware controllers.

- Topology-aware scheduling for StatefulSet pods. This is covered by
  [KEP-5732](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5732-topology-aware-workload-scheduling).

- Automatic detection of whether a StatefulSet "should" use gang scheduling via heuristics. The
  user must explicitly opt in by setting `spec.scheduling` on the StatefulSet.

- Mutable `spec.scheduling` post-creation (toggle on/off, flip gang/basic, or change topology
  constraints); scheduling configuration is immutable for Alpha. For Beta, `gang.minCount` will
  become user-configurable (and mutable), while the scheduling policy type and topology
  constraints remain immutable.

- User-configurable `gang.minCount` — for Alpha, `minCount` is always derived from
  `spec.replicas` to keep the initial implementation simple. Making `minCount` configurable
  (allowing `minCount < replicas` for partial-gang semantics) is planned for Beta as a future
  extension (see [Beta graduation criteria](#beta)).

- Per-replica ResourceClaims analogous to `volumeClaimTemplates`, where each StatefulSet replica
  gets its own dedicated ResourceClaim. This KEP only supports per-PodGroup ResourceClaims
  (shared across all pods in a PodGroup). Per-replica ResourceClaim semantics would require
  additional design and are deferred to future work.

## Proposal

The StatefulSet controller in `kube-controller-manager` will be extended to create and manage
`Workload` and `PodGroup` objects based on the StatefulSet's configuration. A new `scheduling`
field is added to `StatefulSetSpec`, embedding the reusable `scheduling.k8s.io/v1alpha3` building
blocks (scheduling policy, topology constraints, disruption mode).

Following the decoupled model from KEP-5832 and the controller pattern from KEP-6089, the
integration uses the `workloadbuilder` library:

```go
// Scheduling, if set, opts this StatefulSet into Workload-Aware Scheduling.
// The controller compiles one Workload and one PodGroup per StatefulSet with
// gang minCount == replicas (Alpha; user-configurable in Beta).
// Requires podManagementPolicy: Parallel for gang;
// OrderedReady StatefulSets receive Basic (non-gang) policy.
//
// +featureGate=WorkloadWithStatefulSet
// +optional
// +k8s:ifDisabled(WorkloadWithStatefulSet)=+k8s:forbidden
// +k8s:optional
// +k8s:update=NoSet
// +k8s:update=NoUnset
Scheduling *StatefulSetSchedulingConfiguration `json:"scheduling,omitempty"`
```

`StatefulSetSchedulingConfiguration` reuses the `scheduling.k8s.io/v1alpha3` building-block types:

```go
type StatefulSetSchedulingConfiguration struct {
    // SchedulingPolicy defines the scheduling policy for this StatefulSet.
    // Exactly one of Basic or Gang must be set.
    // For Parallel StatefulSets the user sets gang: {}; for OrderedReady,
    // only basic: {} is valid. For Alpha, the controller derives minCount
    // from spec.replicas and a user-supplied minCount is rejected at
    // admission. User-configurable minCount is planned for Beta.
    // This field is immutable after creation: the policy may not be added or
    // removed. The policy variant (basic/gang) is frozen by hand-written
    // validation; only schedulingPolicy.gang.minCount may be changed.
    //
    // +optional
    // +k8s:optional
    // +k8s:update=NoSet
    // +k8s:update=NoUnset
    SchedulingPolicy *schedulingv1alpha3.WorkloadPodGroupSchedulingPolicy `json:"schedulingPolicy,omitempty" protobuf:"bytes,1,opt,name=schedulingPolicy"`

    // SchedulingConstraints defines scheduling constraints (e.g. topology)
    // for the StatefulSet's pods.
    // This field is immutable after creation.
    //
    // +optional
    // +k8s:optional
    // +k8s:immutable
    SchedulingConstraints *schedulingv1alpha3.WorkloadPodGroupSchedulingConstraints `json:"schedulingConstraints,omitempty" protobuf:"bytes,2,opt,name=schedulingConstraints"`

    // DisruptionMode defines the mode in which the StatefulSet's pods can be
    // disrupted. One of Single, All.
    // Only meaningful with Gang policy — for Basic policy this field is a
    // no-op (pods are always disrupted independently).
    // This field is immutable after creation: it may not be added or removed,
    // and the selected mode may not be changed.
    //
    // +optional
    // +k8s:optional
    // +k8s:immutable
    DisruptionMode *schedulingv1alpha3.WorkloadPodGroupDisruptionMode `json:"disruptionMode,omitempty" protobuf:"bytes,3,opt,name=disruptionMode"`

    // ResourceClaims defines which ResourceClaims may be shared among Pods in
    // the StatefulSet. Pods consume the devices allocated to a PodGroup's
    // claim by defining a claim in its own Spec.ResourceClaims that matches
    // the PodGroup's claim exactly. The claim must have the same name and
    // refer to the same ResourceClaim or ResourceClaimTemplate.
    // At most 4 claims may be set, matching the limit on the resulting
    // PodGroup.
    // This list is immutable after creation: entries may neither be added,
    // removed, nor modified.
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
    // +k8s:immutable
    // +featureGate=DRAWorkloadResourceClaims
    ResourceClaims []schedulingv1alpha3.WorkloadPodGroupResourceClaim `json:"resourceClaims,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,4,rep,name=resourceClaims"`
}
```

When the StatefulSet controller detects that `spec.scheduling` is non-nil, it:

1. Creates a `Workload` object (static template defining the PodGroup template).
2. Creates a `PodGroup` object (runtime scheduling unit, snapshot of the Workload's template).
3. Waits for the PodGroup to be observed in the informer cache.
4. Creates pods with `spec.schedulingGroup.podGroupName` referencing the PodGroup.
5. The scheduler uses the PodGroup's gang policy to hold pods in the `Permit` phase until
   `MinCount` pods are schedulable, then binds them simultaneously.

For StatefulSets with `podManagementPolicy: OrderedReady`, the controller creates a Workload and
PodGroup with `Basic` policy (no gang semantics), preserving the existing sequential creation
behavior while enabling workload-level observability and future scheduling optimizations.

### User Stories

#### Story 1: Distributed Database with Quorum Requirements

As a platform engineer deploying a 5-node CockroachDB cluster on Kubernetes using a StatefulSet, I
want all 5 database pods to be scheduled atomically so that the cluster starts with full
replication factor and never enters a degraded state due to partial scheduling. Without gang
scheduling, 3 of 5 pods might get scheduled, forming a quorum but with reduced fault tolerance,
while the remaining 2 pods stay pending indefinitely due to resource fragmentation.

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
    disruptionMode:
      all: {}
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

#### Story 2: Distributed Coordination Service

As a cluster administrator running a ZooKeeper ensemble, I need all 3 ZooKeeper pods to start
together to form a quorum. A ZooKeeper ensemble with only 1 or 2 out of 3 members running cannot
elect a leader and is completely non-functional. Gang scheduling prevents wasting resources on a
partial deployment that cannot serve any requests.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: zookeeper
spec:
  serviceName: zk-headless
  replicas: 3
  podManagementPolicy: Parallel
  scheduling:
    schedulingPolicy:
      gang: {}          # minCount derived from spec.replicas (3)
  template:
    spec:
      containers:
      - name: zookeeper
        image: zookeeper:3.9
```

#### Story 3: Stateful AI/ML Inference Serving

As an ML engineer deploying a sharded model inference service where each shard holds a portion of a
large language model, I need all shards to be co-scheduled because the model cannot serve inference
requests until every shard is available. Partial deployment wastes expensive GPU resources on shards
that cannot function independently.

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: llm-inference-shards
spec:
  serviceName: llm-shards
  replicas: 8
  podManagementPolicy: Parallel
  scheduling:
    schedulingPolicy:
      gang: {}          # minCount derived from spec.replicas (8)
  template:
    spec:
      containers:
      - name: inference-shard
        image: my-org/llm-inference:latest
        resources:
          requests:
            nvidia.com/gpu: 1
```

#### Story 4: Topology-Aware GPU Placement for Distributed Training

As an ML engineer deploying a distributed training workload using a StatefulSet where each pod
holds a model shard, I need all pods to be placed within the same network topology domain (e.g.,
under the same spine switch) to minimize inter-shard communication latency. I want to express this
as a shared ResourceClaim that represents the topological domain, so that all pods in my
StatefulSet are co-located without managing ResourceClaims separately from the StatefulSet.

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaimTemplate
metadata:
  name: topology-domain
spec:
  spec:
    devices:
      requests:
      - name: domain
        exactly:
          deviceClassName: network-topology
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: training-shards
spec:
  serviceName: training-shards
  replicas: 4
  podManagementPolicy: Parallel
  scheduling:
    schedulingPolicy:
      gang: {}          # minCount derived from spec.replicas (4)
    resourceClaims:
    - name: topo
      resourceClaimTemplateName: topology-domain
  template:
    spec:
      containers:
      - name: trainer
        image: my-org/distributed-trainer:latest
        resources:
          requests:
            nvidia.com/gpu: 1
          claims:
          - name: topo
      resourceClaims:
      - name: topo
        resourceClaimTemplateName: topology-domain
```

The StatefulSet controller creates a PodGroup with a `resourceClaims` entry referencing the
`topology-domain` ResourceClaimTemplate. The ResourceClaim controller generates one ResourceClaim
for the PodGroup, and the scheduler allocates a topological domain device once, shared by all 4
pods. Each pod's `spec.resourceClaims` matches the PodGroup's claim, so the ResourceClaim is
reserved for the PodGroup (not per-pod), and all pods are placed within the same domain.

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

3. **Mutable minCount on PodGroup**: Per KEP-5832, `minCount` on the standalone `PodGroup` object
   is mutable. This means simple scaling operations (changing `spec.replicas`) can be handled by
   updating the PodGroup's `minCount` in place. However, operations that change the PodGroup
   structure (e.g., partition changes creating new PodGroups) still require PodGroup recreation.

4. **schedulingGroup immutability on pods**: The `spec.schedulingGroup.podGroupName` field on pods
   is immutable after creation. This means operations like removing a `RollingUpdate.partition`
   that change which PodGroup a pod belongs to require pod recreation—there is no way to reassign
   a running pod to a different PodGroup.

5. **Finalizer requirement**: The PodGroup must have a finalizer to prevent premature deletion
   while pods still reference it (KEP-5832 deletion protection). Without this, deleting a
   PodGroup would orphan its pods, leaving them pointing to a non-existent PodGroup and causing
   scheduler issues. The finalizer is added at PodGroup creation and removed only when all
   referencing pods reach a terminal phase.

6. **Feature gate dependencies**: This feature depends on the `GenericWorkload` feature gate
   (introduced in KEP-4671, consolidated in 1.37) being enabled in both `kube-controller-manager`
   and `kube-scheduler`. The `resourceClaims` field on `StatefulSetSchedulingConfiguration`
   additionally requires the `DRAWorkloadResourceClaims` feature gate (KEP-5729) to be enabled
   in kube-apiserver, kube-controller-manager, kube-scheduler, and kubelet. When
   `DRAWorkloadResourceClaims` is disabled, the `resourceClaims` field is ignored and the
   StatefulSet controller does not populate `resourceClaims` on the PodGroupTemplate or PodGroup.

7. **ResourceClaim lifecycle tied to PodGroup**: ResourceClaims generated from
   ResourceClaimTemplates referenced by a PodGroup are owned by the PodGroup (via
   `ownerReferences`). When the PodGroup is deleted, the generated ResourceClaim is deallocated
   and garbage-collected. For StatefulSets, this means that PodGroup recreation (e.g., due to
   partition changes) causes the old ResourceClaim to be deleted and a new one generated for the
   new PodGroup.

8. **Maximum 4 ResourceClaims per PodGroup**: KEP-5729 limits the `resourceClaims` list to 4
   entries per PodGroupTemplate/PodGroup. This limit applies to the StatefulSet's
   `spec.scheduling.resourceClaims` field as well.

9. **Higher-level controllers composing StatefulSets**: Controllers such as LeaderWorkerSet (LWS)
   create and manage StatefulSets as their building blocks. The StatefulSet controller only
   creates Workload and PodGroup objects when `spec.scheduling` is explicitly set — there is no
   heuristic or automatic opt-in. This means StatefulSets created by higher-level controllers
   will not produce scheduling objects unless the composing controller explicitly sets
   `spec.scheduling` on the inner StatefulSet specs. Higher-level controllers that manage their
   own gang scheduling across multiple StatefulSets should not set `spec.scheduling` on the
   composed StatefulSets to avoid conflicting or redundant PodGroups.

### Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Partition changes require PodGroup recreation | Changing `RollingUpdate.partition` requires PodGroup recreation and pod recreation due to immutable `schedulingGroup` on pods | Document this behavior clearly; partition changes are infrequent and already involve rolling pod updates |
| Workload object leak if StatefulSet controller crashes mid-creation | Orphaned Workload objects consume scheduler resources | OwnerReference with `blockOwnerDeletion=true` ensures garbage collection; finalizer prevents premature cleanup |
| Users configure gang scheduling with `OrderedReady` policy | Unclear failure mode, pods created sequentially cannot satisfy gang | Admission validation rejects this combination at creation time with a clear error message |
| Scheduler overhead from additional Workload objects | Increased memory and processing in kube-scheduler | Workloads are only created when explicitly requested via `spec.scheduling`; no Workload objects for StatefulSets that don't opt in |
| Version skew between controller-manager and scheduler | Controller creates Workloads that scheduler doesn't understand | Workload objects are ignored by schedulers without the feature gate; pods fall back to standard scheduling |
| ResourceClaim lost on PodGroup recreation | Partition changes recreate PodGroups; generated ResourceClaims are owned by the PodGroup and deleted with it | Document that partition changes cause ResourceClaim reallocation; for pre-existing named ResourceClaims (not templates), the claim persists and the new PodGroup references the same claim |
| `DRAWorkloadResourceClaims` gate disabled but `resourceClaims` set | Controller silently ignores the field; pods are created without shared claims and may fail to access expected devices | Admission warns when `resourceClaims` is set but `DRAWorkloadResourceClaims` is disabled; document feature gate dependency |

## Design Details

### Workload Ownership Model

Following the ownership hierarchy from KEP-5832, the StatefulSet controller directly owns the
Workload, PodGroup, and Pods:

```
StatefulSet (apps/v1)
  ├── creates & owns → Workload (scheduling.k8s.io/v1alpha3) [static template]
  ├── creates & owns → PodGroup (scheduling.k8s.io/v1alpha3)  [runtime, references Workload template]
  └── creates & owns → Pods (reference PodGroup via spec.schedulingGroup.podGroupName)
```

- The `OwnerReference.UID` must match the StatefulSet's `metadata.UID` to prevent stale references
  after owner recreation.
- All objects reside in the same namespace as the StatefulSet (OwnerReferences cannot cross
  namespace boundaries).
- When the StatefulSet is deleted, garbage collection automatically deletes the Workload and
  PodGroup via foreground cascading deletion.
- Setting `blockOwnerDeletion=true` ensures the StatefulSet's deletion blocks until the Workload
  and PodGroup are fully removed, maintaining consistent cleanup ordering.
- The PodGroup is created with a finalizer (KEP-5832 deletion protection) that is removed only
  when all referencing pods reach a terminal phase.

### Pod Management Policy Constraints

| Pod Management Policy | Workload Policy | Gang Scheduling | Rationale |
|----------------------|-----------------|-----------------|-----------|
| `Parallel` | `Gang` | Yes | All pods created simultaneously—natural fit for gang semantics |
| `OrderedReady` | `Basic` | No | Sequential pod creation conflicts with gang's all-at-once model |

Gang scheduling for StatefulSet **requires** `podManagementPolicy: Parallel`. The `OrderedReady`
policy creates pods sequentially, waiting for each to become Ready before creating the next, which
fundamentally conflicts with gang scheduling's requirement that all pods exist simultaneously for
the scheduler to evaluate them as a group.

**DisruptionMode with Basic policy**: `DisruptionMode` is only meaningful with Gang policy, where
it controls whether preemption evicts a single pod or the entire gang. For OrderedReady
StatefulSets with Basic policy, `DisruptionMode` is a no-op — pods are always disrupted
independently regardless of the value set, since there is no gang to atomically disrupt.

Admission validation enforces this constraint:
- If `spec.scheduling.schedulingPolicy.gang` is non-nil and `podManagementPolicy` is
  `OrderedReady` (or unset, since `OrderedReady` is the default), the request is rejected with a
  descriptive error.

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
apiVersion: scheduling.k8s.io/v1alpha3
kind: Workload
metadata:
  name: my-statefulset-workload
  ownerReferences:
  - apiVersion: apps/v1
    kind: StatefulSet
    name: my-statefulset
    uid: <statefulset-uid>
    controller: true
    blockOwnerDeletion: true
spec:
  controllerRef:
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
apiVersion: scheduling.k8s.io/v1alpha3
kind: PodGroup
metadata:
  name: my-statefulset-podgroup
  ownerReferences:
  - apiVersion: apps/v1
    kind: StatefulSet
    name: my-statefulset
    uid: <statefulset-uid>
    controller: true
    blockOwnerDeletion: true
  finalizers:
  - scheduling.k8s.io/podgroup-protection
spec:
  podGroupTemplateRef:
    workloadName: my-statefulset-workload
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
# Pod spec with schedulingGroup and ResourceClaim references
spec:
  schedulingGroup:
    podGroupName: my-statefulset-podgroup
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

The `workloadbuilder` library constructs these objects using a callback pattern:

```go
rootNode := &workloadbuilder.WorkloadItem{
    Name:          sts.Name,
    DefaultConfig: defaultSchedulingConfig(sts),
    UserConfig:    sts.Spec.Scheduling,
    Callbacks:     []workloadbuilder.WorkloadItemFunc{defaultMinCountForStatefulSet(sts)},
}
builder := workloadbuilder.NewBuilder(rootNode)
workload, podGroups, _ := builder.Build(ctx, sts.Name, sts.Namespace, ownerRef)
```

The `defaultMinCountForStatefulSet` callback sets `minCount = *sts.Spec.Replicas` when gang policy
is requested. For Alpha, a user-supplied `minCount` is rejected at admission; the callback always
applies. In Beta, this callback will serve as the default when `minCount` is omitted, while
user-supplied values will be accepted.

When `spec.scheduling.resourceClaims` is non-empty (and the `DRAWorkloadResourceClaims` feature
gate is enabled), the controller populates `PodGroupTemplate.resourceClaims` on the Workload
object. The PodGroup snapshots these entries into its own `spec.resourceClaims`. Pods created by
the StatefulSet controller include matching `spec.resourceClaims` entries — same `name` and same
`resourceClaimName` or `resourceClaimTemplateName` — so that the DRA scheduler plugin and
ResourceClaim controller treat these claims as PodGroup-level rather than per-pod.

### StatefulSet with OrderedReady

For StatefulSets using the default `OrderedReady` policy that have `spec.scheduling`
configured (without gang scheduling), the controller creates a Workload with `Basic` policy. This
provides workload-level observability and metadata without altering the sequential scheduling
behavior.

```yaml
apiVersion: scheduling.k8s.io/v1alpha3
kind: Workload
metadata:
  name: my-ordered-statefulset-workload
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

### StatefulSet with RollingUpdate.Partition

For StatefulSets using `updateStrategy.rollingUpdate.partition`, the controller creates separate
PodGroups for partitioned and non-partitioned pods. This respects the partition semantics where
pods with an ordinal greater than or equal to the partition value are updated while pods below the
partition retain the old revision.

```yaml
# StatefulSet with partition=2 and replicas=5
# Pods 0,1 use old revision (partition group), pods 2,3,4 use new revision
apiVersion: scheduling.k8s.io/v1alpha3
kind: Workload
metadata:
  name: my-partitioned-statefulset-workload
spec:
  controllerRef:
    apiGroup: apps
    kind: StatefulSet
    name: my-partitioned-statefulset
  podGroupTemplates:
  - name: partition-old
    schedulingPolicy:
      gang:
        minCount: 2  # Pods below partition
  - name: partition-new
    schedulingPolicy:
      gang:
        minCount: 3  # Pods at or above partition
```

**Trade-offs:**

| Pros | Cons |
|------|------|
| Respects partition semantics | More complex Workload mapping |
| Allows rolling updates with gang scheduling | Partition changes require Workload recreation |

**Open question**: Removing a partition after it has been set requires updating
`spec.schedulingGroup.podGroupName` on existing pods, which is currently immutable. The only known
solution is pod recreation.

### Lifecycle Management

#### Initial Creation Lifecycle

When a Parallel StatefulSet is created with gang scheduling enabled:

1. The StatefulSet controller detects `spec.scheduling` is non-nil.
2. The controller creates the `Workload` object (static template with PodGroup template).
3. The controller creates the `PodGroup` object (runtime unit, snapshot of Workload template).
4. The controller waits for the PodGroup to be observed in the informer cache.
5. The controller creates all pod templates with `spec.schedulingGroup.podGroupName` pointing to
   the PodGroup.
6. Pods enter the scheduler queue.
7. The scheduler's `PreFilter` plugin looks up each pod's PodGroup to determine gang requirements.
8. Pods are held in the `Permit` phase until `MinCount` pods are schedulable.
9. Once MinCount is satisfied, the scheduler binds all gang members simultaneously.

```
StatefulSet Created (podManagementPolicy: Parallel, scheduling.policy.gang)
  │
  ├─→ Controller creates Workload (static template, includes resourceClaims if configured)
  │
  ├─→ Controller creates PodGroup (runtime, minCount = replicas, resourceClaims snapshotted)
  │
  ├─→ ResourceClaim controller generates ResourceClaims from PodGroup's
  │   ResourceClaimTemplate references (one per template per PodGroup)
  │
  ├─→ Controller waits for PodGroup in informer cache
  │
  ├─→ Controller creates all Pods simultaneously (sets spec.schedulingGroup.podGroupName;
  │   pods include matching spec.resourceClaims entries from the pod template)
  │
  ├─→ Pods enter scheduler queue
  │
  ├─→ Scheduler PreFilter validates gang (PodGroup → minCount)
  │
  ├─→ DRA scheduler plugin allocates ResourceClaims for PodGroup (reserved for PodGroup,
  │   not individual pods)
  │
  ├─→ Permit gate: hold until minCount pods are schedulable
  │
  └─→ All pods bound simultaneously
```

#### Scale Lifecycle

Per KEP-5832, `minCount` on the standalone `PodGroup` object is mutable, which significantly
simplifies scaling.

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

This non-disruptive scaling is enabled by the mutable `minCount` on the standalone PodGroup
(KEP-5832).

**Note on Alpha behavior**: The examples above reflect Alpha, where `minCount` is always derived
from `spec.replicas` — scaling replicas automatically updates `minCount` to match. In Beta, when
`minCount` becomes user-configurable, scaling `replicas` will not automatically change a
user-supplied `minCount`. For example, a user may set `replicas: 5` with `minCount: 3` and later
scale to `replicas: 7` while keeping `minCount: 3`.

**ResourceClaims during scaling**: ResourceClaims shared at the PodGroup level are unaffected by
simple scaling operations. Scale-up adds new pods that reference the existing PodGroup and its
already-allocated ResourceClaims — no new ResourceClaims are generated. Scale-down deletes pods
but the PodGroup's ResourceClaims remain allocated as long as the PodGroup exists.

**Note**: Operations that change the PodGroup *structure* (e.g., RollingUpdate.partition changes
that require new PodGroups) still require PodGroup recreation and pod recreation due to the
immutability of `spec.schedulingGroup.podGroupName` on pods. When a PodGroup is recreated,
any ResourceClaims generated from ResourceClaimTemplates for the old PodGroup are deleted (via
`ownerReferences` garbage collection), and new ResourceClaims are generated for the new PodGroup.

### Enabling Gang Scheduling

Gang scheduling is enabled via the `spec.scheduling` field on the StatefulSet, following the
controller-as-compiler pattern from [KEP-6089](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis):

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
        - key: "topology.kubernetes.io/zone"
    disruptionMode:
      all: {}
    resourceClaims:             # optional; requires DRAWorkloadResourceClaims gate
    - name: shared-topo
      resourceClaimTemplateName: topo-domain-template
  template:
    spec:
      containers:
      - name: db
        image: my-db:latest
        resources:
          claims:
          - name: shared-topo
      resourceClaims:
      - name: shared-topo
        resourceClaimTemplateName: topo-domain-template
```

The user sets `gang: {}` — the controller derives `minCount = *spec.replicas`. For Alpha, a
user-supplied `minCount` is rejected at admission (this restriction will be relaxed in Beta).
Workload and PodGroup names are deterministic:
`<statefulset-name>` for both, matching the StatefulSet's own name.

When `resourceClaims` is specified, the controller passes these entries through to the
PodGroupTemplate and PodGroup. The pod template must include matching `spec.resourceClaims`
entries — the controller injects `spec.schedulingGroup.podGroupName` but does **not** auto-inject
`spec.resourceClaims` into the pod template; the user must declare them in the StatefulSet's
`spec.template.spec.resourceClaims` to opt each pod into the shared claim. This mirrors how pods
currently opt into ResourceClaims and keeps the pod template explicit.

### Admission Validation

Validation follows the same three-layer approach used across WAS-integrated controllers
([KEP-5547](https://github.com/kubernetes/enhancements/tree/master/keps/sig-apps/5547-integrate-workload-with-job)), but the StatefulSet layer adds rules specific to `podManagementPolicy`
and partition semantics:

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
     Error: gang scheduling requires podManagementPolicy: Parallel;
     OrderedReady creates pods sequentially which conflicts with gang semantics
     ```
   - **User-set `gang.minCount` is forbidden (Alpha)**: If the gang policy carries a non-nil
     `MinCount`, the request is rejected — the value is derived from `spec.replicas` for Alpha.
     This restriction will be relaxed in Beta to allow user-configurable `minCount`.
   - **`spec.scheduling` immutability**: The field cannot be set after creation or unset once set
     (`+k8s:update=NoSet`, `+k8s:update=NoUnset`).
   - **`podManagementPolicy` immutability**: `podManagementPolicy` is already immutable on
     StatefulSets. This means a user cannot create a StatefulSet with OrderedReady, then later
     flip to Parallel to enable gang — the scheduling decision is locked at creation time.
   - **`resourceClaims` consistency**: If `spec.scheduling.resourceClaims` is non-empty, each
     entry must have exactly one of `resourceClaimName` or `resourceClaimTemplateName` set.
     The list is limited to 4 entries (matching the PodGroupTemplate limit from KEP-5729).
     The field is immutable after creation.
   - **Pod template must match PodGroup claims**: If `spec.scheduling.resourceClaims` is set,
     validation warns (but does not reject) if the pod template's `spec.resourceClaims` does
     not contain matching entries for each PodGroup-level claim. Without matching entries in
     the pod template, pods will not consume the PodGroup's shared ResourceClaims.

3. **`workloadbuilder` semantic validation** — the same `WorkloadItem` tree the controller
   compiles is built during validation and checked via `NewBuilder(...).Validate()`. This
   catches structural inconsistencies (e.g., topology constraints on a Basic-policy workload)
   that would otherwise surface only at Workload creation time.

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
  - Correct PodGroup policy selection based on `podManagementPolicy`
  - Pods receive correct `spec.schedulingGroup.podGroupName` references
  - PodGroup minCount updated in-place on replica scaling
  - PodGroup recreation on partition changes
  - Workload and PodGroup cleanup on StatefulSet deletion
  - OwnerReferences and finalizers are set correctly on both Workload and PodGroup
  - ResourceClaims from `spec.scheduling.resourceClaims` are passed through to PodGroupTemplate
    and PodGroup `spec.resourceClaims`
  - Pods receive matching `spec.resourceClaims` entries when the pod template includes them
  - ResourceClaims field is omitted from PodGroupTemplate when `DRAWorkloadResourceClaims` gate
    is disabled
  - ResourceClaims field is omitted when `spec.scheduling.resourceClaims` is empty

- `pkg/apis/apps/validation`: Coverage for admission validation rules.
  - Reject OrderedReady + `scheduling.schedulingPolicy.gang` combination
  - Reject user-set `gang.minCount` (Alpha only — must be nil, derived from replicas)
  - Accept valid Parallel + `scheduling.schedulingPolicy.gang` combinations
  - Accept OrderedReady + `scheduling.schedulingPolicy.basic`
  - Reject `spec.scheduling` mutations (immutability)
  - Accept valid `spec.scheduling.resourceClaims` entries (one of name/template set)
  - Reject `spec.scheduling.resourceClaims` entries with both or neither source set
  - Reject more than 4 entries in `spec.scheduling.resourceClaims`
  - Reject mutations to `spec.scheduling.resourceClaims` after creation

##### Integration tests

- StatefulSet controller creates Workload and PodGroup before pods; pods reference PodGroup correctly
- Scheduler holds pods in Permit phase until MinCount is satisfied
- Scaling updates PodGroup minCount in-place (no recreation needed)
- StatefulSet deletion cascades to Workload and PodGroup deletion via OwnerReference
- Feature gate disabled: no Workload/PodGroup objects created, standard behavior preserved
- Partition update triggers PodGroup recreation with updated structure
- ResourceClaims from `spec.scheduling.resourceClaims` propagated to PodGroup and shared by all pods
- ResourceClaimTemplate generates one ResourceClaim per PodGroup, not per pod
- ResourceClaim is reserved for PodGroup (not individual pods) in `status.reservedFor`
- PodGroup deletion cascades to generated ResourceClaim deletion

##### e2e tests

- End-to-end gang scheduling of a Parallel StatefulSet: all pods scheduled simultaneously
- Scale-up with in-place PodGroup minCount update: verify new pods join existing PodGroup and
  scheduler re-evaluates gang with updated minCount
- Scale-down with in-place PodGroup minCount update: verify correct pod termination and PodGroup
  minCount reduced to match new replica count
- OrderedReady StatefulSet with Basic Workload: verify sequential scheduling preserved
- Failure scenario: insufficient resources prevent gang formation, pods remain in Permit
- StatefulSet deletion: verify Workload and all pods cleaned up
- Feature gate toggle: verify behavior change on enable/disable
- End-to-end ResourceClaim sharing: StatefulSet with `resourceClaims` creates PodGroup with
  shared claim; all pods reference same allocated ResourceClaim
- ResourceClaimTemplate generates one claim per PodGroup; scaling up adds pods referencing the
  existing claim without generating new claims
- Partition change recreates PodGroup and generates a new ResourceClaim for the new PodGroup

### Graduation Criteria

#### Alpha

- Feature implemented behind the `WorkloadWithStatefulSet` feature flag (depends on
  `GenericWorkload` feature gate)
- StatefulSet controller creates Workload and PodGroup objects for Parallel StatefulSets with
  `spec.scheduling` configured
- Basic lifecycle management: create, delete, scale (in-place PodGroup minCount update)
- Admission validation for OrderedReady + Gang conflict
- Initial unit and integration tests completed and enabled
- Documentation of the feature and its limitations (PodGroup immutability, scale disruption)

#### Beta

- Gather feedback from early adopters on scaling behavior and partition-related PodGroup recreation
- Support user-configurable `gang.minCount` to allow partial-gang semantics
  (`minCount < replicas`), enabling use cases like quorum-based systems that can start with a
  subset of replicas
- RollingUpdate.Partition support with multiple PodGroups
- All known edge cases addressed (partition removal, controller crash recovery)
- Metrics for Workload creation/deletion/recreation latency
- E2e tests in CI, linked in TestGrid
- Performance testing to verify acceptable scheduler overhead with StatefulSet Workloads
- Documentation updated with best practices for common stateful workloads (databases,
  coordination services)

#### GA

- At least 2 releases in Beta with no critical bugs
- Real-world usage validation from distributed database and stateful AI/ML operators
- Conformance tests covering core gang scheduling behavior for StatefulSets
- All open questions resolved (partition removal, mutable PodGroups)
- Stable metrics and monitoring documentation

### Upgrade / Downgrade Strategy

With the gate disabled or `spec.scheduling` unset, StatefulSets behave exactly as they do today.

The StatefulSet controller owns Workload and PodGroup objects directly. This means:

- **On downgrade**: Existing Workload and PodGroup objects remain in the cluster, protected by
  their `podgroup-protection` finalizers. They are inert (the controller no longer reconciles
  them) and are cleaned up only when the owning StatefulSet is deleted — there is no
  revision-history pruning equivalent. Cluster admins who want immediate cleanup can delete the
  StatefulSet or manually remove the orphaned objects.
- **Pod-level references**: Pods created while the gate was enabled carry
  `spec.schedulingGroup.podGroupName`. These references are harmless if the scheduler no longer
  processes them — the pods schedule normally via the default path. Pods created after downgrade
  will not carry the field.
- **PVCs are unaffected**: `volumeClaimTemplates`-managed PVCs are independent of the scheduling
  objects and are retained per the StatefulSet's `persistentVolumeClaimRetentionPolicy`.
- **ResourceClaims**: ResourceClaims generated from ResourceClaimTemplates for PodGroups are
  owned by the PodGroup via `ownerReferences`. If the PodGroup persists after downgrade (as
  described above), its ResourceClaims also persist. If the `DRAWorkloadResourceClaims` gate
  is disabled independently, the ResourceClaim controller stops managing PodGroup-level claims,
  but already-allocated claims remain functional for running pods. See KEP-5729 for detailed
  downgrade behavior of PodGroup-level ResourceClaims.

### Version Skew Strategy

The feature requires `GenericWorkload` and the `scheduling.k8s.io/v1alpha3` API to be served by
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

Yes. Disabling the gate has two effects depending on the component:

- **kube-apiserver**: Clears `spec.scheduling` on new creates; preserves the stored value on
  existing objects (standard alpha field-gating).
- **kube-controller-manager**: Stops compiling Workload and PodGroup objects. Existing scheduling
  objects become inert but remain in the cluster because they are owned by the StatefulSet.
  They are cleaned up when the StatefulSet is deleted, or they can be manually removed.

Running pods are not affected — they continue executing regardless of whether their
`schedulingGroup` reference points to an active PodGroup. New pods created after rollback are
created without `schedulingGroup` and schedule individually.

###### What happens if we reenable the feature if it was previously rolled back?

On reenable, the controller resumes compilation for any StatefulSet whose stored
`spec.scheduling` survived the rollback (the API server preserves already-set values even with
the gate off). Specifically:

- If the Workload and PodGroup still exist (they were not manually deleted), the controller
  discovers them via deterministic naming (`<statefulset-name>`) and reuses them.
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
- Running pods unaffected by gate toggle.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Rollout can fail in two ways:

- **API server does not serve `scheduling.k8s.io`**: The controller fails to create Workload and
  PodGroup objects. For gang-configured StatefulSets, pod creation is blocked (pods must not be
  created without a PodGroup, or gang guarantees are lost). The controller retries with backoff.
  StatefulSets without `spec.scheduling` are unaffected.
- **Scheduler lacks `GenericWorkload`**: Workloads and PodGroups are created successfully, but
  the scheduler ignores gang semantics. Pods are scheduled individually — equivalent to
  pre-feature behavior. This is degraded but safe.

Rollback impact: Running pods are unaffected — StatefulSet pods with stable identities continue
executing. Workload and PodGroup objects become inert but persist in the cluster until the
StatefulSet is deleted. New pods created after rollback lack
`schedulingGroup` and schedule individually, which may lead to partial placement for workloads
that depend on atomic scheduling. For stateful systems with quorum requirements, operators should
verify all members are running before rolling back.

###### What specific metrics should inform a rollback?

TBD

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
  - Event Reason: `WorkloadRecreated` — emitted when a Workload is recreated due to partition changes.
  - Event Reason: `WorkloadCreationFailed` — emitted when Workload creation fails.
- [x] API .status
  - Condition name: `WorkloadReady` — indicates whether the Workload is created and ready for
    scheduling.
- [x] Other
  - The `Workload` object itself is visible via `kubectl get workloads` and shows PodGroup status.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

TBD

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

TBD

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

TBD

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- **Workload API (`scheduling.k8s.io/v1alpha3`)**: The CRD or built-in API type must be registered
  in the API server. This is provided by KEP-4671 and gated behind the `GenericWorkload` feature gate.
- **kube-scheduler with gang scheduling plugin**: The scheduler must have the gang scheduling
  plugin enabled to act on Workload objects. Without it, Workloads are created but have no effect
  on scheduling.
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
- During partition changes: `DELETE` + `CREATE` for PodGroup recreation

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
objects. The gang scheduling Permit phase may add additional latency
if resources are constrained, but this is by design (waiting for all pods to be schedulable).
Impact to be measured during Alpha.

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
| Workload exists but scheduler doesn't process it | Scheduler feature gate disabled or plugin not loaded | Pods remain Pending without gang-related scheduler events | Enable `GenericWorkload` feature gate in scheduler |
| Gang cannot be satisfied | Cluster lacks resources for MinCount pods | Pods remain in Permit phase; `scheduler_permit_wait_duration_seconds` increases | Scale cluster or reduce replicas (which reduces MinCount in Alpha) |
| Stale Workload after controller crash | Controller crashed between creating Workload and creating pods | Workload exists but no pods reference it | Controller reconciliation detects and resolves on restart |

###### What steps should be taken if SLOs are not being met to determine the problem?

1. Check `statefulset_workload_creation_errors_total` for persistent Workload creation failures.
2. Check `scheduler_permit_wait_duration_seconds` for gang satisfaction delays.
3. Verify both `WorkloadWithStatefulSet` and `GenericWorkload` feature gates are enabled.
4. Inspect Workload objects: `kubectl get workloads -n <namespace>` to verify they exist and have
   correct PodGroup configurations.
5. Check scheduler logs for gang-related messages.
6. If the issue is resource-related, check node capacity vs. gang MinCount requirements.

## Implementation History

- 2026-08-25: Initial KEP draft created (OCPNODE-4667)
- 2026-08-27: Added per-PodGroup ResourceClaim support (KEP-5729 integration)

## Drawbacks

1. **Partition-related disruption**: While simple replica scaling is non-disruptive (mutable
   minCount on PodGroup), operations that change PodGroup structure (e.g., partition changes)
   still require PodGroup recreation and pod recreation due to immutable `schedulingGroup` on
   pods.

2. **Increased complexity in the StatefulSet controller**: The controller gains significant new
   responsibility for Workload and PodGroup lifecycle management, increasing its code complexity
   and the surface area for bugs. The `workloadbuilder` library mitigates this by encapsulating
   common patterns.

3. **OrderedReady limitation**: The most common StatefulSet configuration (OrderedReady, which is
   the default) cannot use gang scheduling. Users must explicitly switch to Parallel policy,
   which changes pod startup behavior and may not be suitable for all applications.

4. **Additional API objects**: Each gang-scheduled StatefulSet creates an additional Workload
   object, increasing the total object count in the cluster and adding to API server and etcd load.

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
- [KEP-5832: Decouple PodGroup API](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5832-decouple-podgroup-api) — Standalone mutable PodGroup
- [KEP-6089: WAS Controller APIs](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6089-was-controller-apis) — Reusable building blocks and `workloadbuilder` library
- [KEP-5710: Workload-Aware Preemption](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5710-workload-aware-preemption)
- [KEP-5729: DRA ResourceClaim Support for Workloads](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5729-resourceclaim-support-for-workloads) — PodGroup-level ResourceClaim sharing
- [KEP-5732: Topology-Aware Workload Scheduling](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/5732-topology-aware-workload-scheduling)
- [Workload API Integration Design Document](https://docs.google.com/document/d/1-lBmKIlgcCggjBWYO0VkZ8t3NmvJp-LCsfP6IB0XKRY) — Cross-controller integration blueprint by Heba Elayoty and Andrey Velichkevich
