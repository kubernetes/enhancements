# KEP-6276: Workload-Aware Scheduling for Deployments

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Deployment Integration - API Usage Examples](#deployment-integration---api-usage-examples)
    - [Example 1: Gang scheduling with zone topology and atomic disruption](#example-1-gang-scheduling-with-zone-topology-and-atomic-disruption)
    - [Example 2: Gang scheduling with user-defined minCount](#example-2-gang-scheduling-with-user-defined-mincount)
    - [Example 3: Gang with template-backed ResourceClaims](#example-3-gang-with-template-backed-resourceclaims)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Feature Gate and RBAC](#feature-gate-and-rbac)
  - [Controller Changes](#controller-changes)
    - [Creation Ordering](#creation-ordering)
    - [workloadbuilder Integration](#workloadbuilder-integration)
    - [EqualIgnoreHash](#equalignorehash)
    - [Scaling and HPA](#scaling-and-hpa)
  - [Mutability and Validation](#mutability-and-validation)
  - [Test Plan](#test-plan)
    - [Unit Tests](#unit-tests)
    - [Integration Tests](#integration-tests)
    - [E2E Tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
  - [Upgrade Strategy](#upgrade-strategy)
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
<!-- /toc -->

## Summary

This KEP integrates the Workload-aware Scheduling (WAS) APIs (`Workload` and `PodGroup`) into
`apps/v1` Deployments through a user-facing `spec.scheduling` field, allowing users to express
scheduling intent such as gang scheduling, topology placement, disruption handling, and DRA
resource claims. The Deployment controller creates one stable Deployment-owned Workload, while
the ReplicaSet controller materializes one ReplicaSet-owned PodGroup per revision via the shared
`workloadbuilder` library ([KEP-6089]), adapting the controller-as-compiler pattern established by
the Job + WAS integration ([KEP-5547]) to the Deployment/ReplicaSet rollout, scaling, and
revision-history lifecycle.

## Motivation

Long-running inference services (multi-GPU model servers, disaggregated prefill/decode
pipelines) commonly run as `apps/v1.Deployment` objects and require all replicas co-located
within the same topology domain or placed atomically to avoid wasting accelerator capacity on
partially placed groups.

Today the only path to gang-schedule or topology-schedule a Deployment is to manually create a
`PodGroup` and inject `pod.spec.schedulingGroup.podGroupName` into the pod template. This approach
is fragile: a pod created before its referenced PodGroup exists hangs silently in Pending with no
event or error. It also places the entire burden of naming, ownership, garbage collection, and
scale-time reconciliation on the user, none of which composes cleanly with rolling updates,
revision history, or HPA-driven scaling. Alternatively, users can turn to external solutions like
Volcano, Kueue, KAI or Coscheduling plugin.

### Goals

- Enable users to apply Workload-Aware Scheduling to Deployments without manually creating or
  managing Workloads, PodGroups, or their lifecycle.
- Allow users to optionally set gang `minCount`. When unset, the controller derives it from
  `spec.replicas`. Validation ensures `minCount` does not exceed `replicas`.
- Support gang scheduling with `Recreate` strategy.
- Support horizontal scaling and HPA natively through the Deployment `/scale` subresource.
- Support topology-constrained Deployments whose replicas must be co-located within a requested
  topology domain.
- Support shared DRA resource claims for Deployment replicas, including claims backed by
  `ResourceClaimTemplate` objects.

### Non-Goals

- Automatic in-tree recovery or rescheduling of a replacement pod stuck on a saturated topology
  domain. Left to out-of-tree queue managers.
- `RollingUpdate` with gang scheduling. For Alpha, only `Recreate` is supported. Rolling
  update semantics with gang scheduling are re-evaluated for Beta.
- Supporting mutable `spec.scheduling` post-creation (toggle on/off, flip gang to basic, or change
  topology constraints). Scheduling configuration is immutable for Alpha.
- Elastic gang semantics (multiple PodGroups per ReplicaSet for `minCount < replicas`).
  Deferred to Beta.
- Multi-level or nested composite (`CompositePodGroup`) structures; this KEP covers single-level
  Deployment → ReplicaSet workloads only.
- Exclusive access to DRA claims. Any pod on the same node can reference a PodGroup's claim by
  name and share the device. Claim isolation is a DRA-layer property; this KEP does not add access
  control beyond what DRA provides.

## Proposal

This proposal builds on the recently introduced Workload-aware Scheduling enhancements. We assume
the reader is acquainted with the following KEPs:

- [KEP-4671]: Gang Scheduling.
- [KEP-5710]: Workload-aware preemption.
- [KEP-5732]: Topology-aware workload scheduling.
- [KEP-6089]: WAS Controller APIs.

The Deployment controller compiles the user's scheduling intent into a single `Workload`, created
once per Deployment. For each ReplicaSet revision, the ReplicaSet controller materializes one
ReplicaSet-owned `PodGroup` from the Workload's `PodGroupTemplate`, carrying the scheduling policy,
topology constraints, and resource claims into a per-revision runtime context. The intent is
expressed through a new `spec.scheduling` field.

The key design principles:

- One `Workload` per Deployment serves as the shared scheduling template. Each ReplicaSet revision
  gets its own `PodGroup` stamped from that template, with an independent scheduling context
  (topology domain, gang quorum).
- Scheduling intent (gang semantics, topology constraints, disruption mode) is expressed through
  `spec.scheduling` and is orthogonal to the Deployment's rollout strategy (`spec.strategy`).
  When `spec.scheduling` is omitted, no scheduling objects are created.
- Gang `minCount` defaults to `spec.replicas` when unset. Users may set `minCount` explicitly,
  but validation rejects values exceeding `replicas`.
- All `spec.scheduling` fields are immutable after creation.
- Deployments own the `Workload` for its lifecycle, and ReplicaSets own individual `PodGroups`
  for revision-specific scheduling. Each PodGroup also carries a non-controller ownerReference to
  the Workload for discoverability, matching the Job pattern ([KEP-5547]). Supported by
  deterministic naming and reconciliation, creation is fully idempotent so that controllers
  recover missing objects after crashes while garbage collection handles cleanup.

### Deployment Integration - API Usage Examples

#### Example 1: Gang scheduling with zone topology and atomic disruption

A multi-GPU inference service whose 3 replicas must schedule together, co-locate within the same
availability zone, and be disrupted atomically:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: inference-server
  namespace: ml-serving
spec:
  replicas: 3
  strategy:
    type: Recreate
  scheduling:
    schedulingPolicy:
      gang: {}
    schedulingConstraints:
      topology:
        - key: "topology.kubernetes.io/zone"
    disruptionMode:
      all: {}
  selector:
    matchLabels:
      app: inference-server
  template:
    metadata:
      labels:
        app: inference-server
    spec:
      containers:
      - name: server
        image: inference-server:v1
        resources:
          limits:
            nvidia.com/gpu: 1
```

The Deployment controller compiles this intent into a `Workload` owned by the Deployment. The
ReplicaSet controller creates and owns the revision-specific `PodGroup`. The Workload remains
stable and is reused across rollouts:

```yaml
apiVersion: scheduling.k8s.io/v1beta1
kind: Workload
metadata:
  name: inference-server
  namespace: ml-serving
  ownerReferences:
  - apiVersion: apps/v1
    kind: Deployment
    name: inference-server
    uid: <deployment-uid>
    controller: true
spec:
  controllerRef:
    apiVersion: apps/v1
    kind: Deployment
    name: inference-server
  podGroupTemplates:
  - name: inference-server
    schedulingPolicy:
      gang:
        minCount: 3
    schedulingConstraints:
      topology:
        - key: "topology.kubernetes.io/zone"
    disruptionMode:
      all: {}
---
apiVersion: scheduling.k8s.io/v1beta1
kind: PodGroup
metadata:
  name: inference-server-<podTemplateHash>
  namespace: ml-serving
  ownerReferences:
  - apiVersion: apps/v1
    kind: ReplicaSet
    name: inference-server-<podTemplateHash>
    uid: <rs-uid>
    controller: true
  - apiVersion: scheduling.k8s.io/v1beta1
    kind: Workload
    name: inference-server
    uid: <workload-uid>
spec:
  schedulingPolicy:
    gang:
      minCount: 3
  schedulingConstraints:
    topology:
      - key: "topology.kubernetes.io/zone"
  disruptionMode:
    all: {}
```

#### Example 2: Gang scheduling with user-defined minCount

Gang scheduling with a user-defined `minCount` lower than `replicas`. The gang is satisfiable
with 3 out of 4 pods, allowing partial placement:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: prefill-workers
spec:
  replicas: 4
  strategy:
    type: Recreate
  scheduling:
    schedulingPolicy:
      gang:
        minCount: 3
  selector:
    matchLabels:
      app: prefill-workers
  template:
    metadata:
      labels:
        app: prefill-workers
    spec:
      containers:
      - name: worker
        image: prefill:v2
        resources:
          limits:
            nvidia.com/gpu: 2
```

#### Example 3: Gang with template-backed ResourceClaims

A gang Deployment that requests a shared DRA device allocated once per PodGroup:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: model-server
spec:
  replicas: 3
  strategy:
    type: Recreate
  scheduling:
    schedulingPolicy:
      gang: {}
    resourceClaims:
    - name: gpu-pool
      resourceClaimTemplateName: gpu-template
  selector:
    matchLabels:
      app: model-server
  template:
    metadata:
      labels:
        app: model-server
    spec:
      containers:
      - name: server
        image: model-server:v1
        resources:
          claims:
          - name: gpu-pool
```

### User Stories

**Distributed inference server.** A platform team runs a tensor-parallel inference service as a
Deployment. All replicas must be scheduled together or not at all, because a partially placed set
wastes accelerator capacity without serving traffic. The team sets
`spec.scheduling.schedulingPolicy.gang: {}`. On initial creation, the scheduler places the full
gang atomically or leaves all pods pending. On scale-up, existing pods continue running while new
pods wait for the updated gang quorum to be satisfiable.

**Rack-local worker pool.** A latency-sensitive service needs all its pods co-located within one
rack. The team adds a topology constraint on `topology.kubernetes.io/rack`. The scheduler places
the whole gang in a best-fit rack. If no single rack can satisfy the request, the Deployment
reports unavailable replicas through its standard status conditions.

### Notes/Constraints/Caveats

- Users may set `gang.minCount`. Validation rejects values exceeding `replicas`. When
  `minCount < replicas`, the gang is satisfiable with fewer pods than the full replica count,
  but each ReplicaSet still has a single PodGroup. Multiple PodGroups per ReplicaSet for
  elastic gang semantics are deferred to Beta.
- Gang scheduling requires `Recreate` strategy. `RollingUpdate` with gang is rejected at
  admission.
- With `Recreate` strategy, the old ReplicaSet scales to zero and its PodGroup and
  ResourceClaims are released before the new ReplicaSet is created. A transient overlap
  of old and new claims is possible due to async cleanup but is unlikely in practice.
- `revisionHistoryLimit` retains old ReplicaSets for rollback metadata only. PodGroups are
  explicitly deleted at scale-to-zero, so they are already gone before the ReplicaSet becomes a
  history entry. If explicit deletion is missed (e.g., crash), owner-reference GC removes the
  PodGroup when the ReplicaSet is pruned, and the `podgroup-protection` finalizer ensures
  referencing pods drain first.
- Named ResourceClaims pin all revisions to the same node (the node where the device is allocated).
  If the node lacks capacity for the gang's pods and their requested resources, the update may
  remain Pending.
- Topology binding is permanent per PodGroup. A replacement pod stuck on a full domain will not
  automatically reschedule to a different domain.
- At `replicas=0`, the Deployment-owned Workload is retained. If the user set `minCount`
  explicitly, that value is preserved. Otherwise the controller defaults to `minCount=1`.
  The ReplicaSet-owned PodGroup is deleted at zero replicas and recreated when the
  Deployment scales positive again.
- Workload and PodGroup informers and listers are used by the Deployment and ReplicaSet
  controllers to reconcile scheduling objects.
- When `DRAWorkloadResourceClaims` gate is off, `spec.scheduling.resourceClaims` is stored on the
  Deployment but silently stripped from the PodGroup by the apiserver. Pods fall back to per-pod
  claims instead of shared PodGroup-level claims. Alpha gap: rejection deferred to Beta.

### Risks and Mitigations

- **Named ResourceClaim deadlocks rollouts on tight nodes.** A named claim pins revisions to one
  node. If that node lacks capacity for the gang's pods and their requested resources, the rollout
  hangs. *Mitigation:* document the capacity requirement and prefer template-backed claims when
  independent per-revision allocation is needed.

## Design Details

### API Changes

A new optional field is added to `DeploymentSpec` in both the internal (`pkg/apis/apps`) and
external (`apps/v1`) types:

```go
// Scheduling, if set, opts this Deployment into Workload-Aware Scheduling.
// The controller compiles one Workload per Deployment and one PodGroup per
// ReplicaSet. Gang minCount defaults to replicas when unset by the user.
//
// +featureGate=WorkloadWithDeployment
// +optional
// +k8s:ifDisabled(WorkloadWithDeployment)=+k8s:forbidden
// +k8s:optional
// +k8s:update=NoSet
// +k8s:update=NoUnset
Scheduling *DeploymentSchedulingConfiguration `json:"scheduling,omitempty"`
```

`DeploymentSchedulingConfiguration` mirrors `batch/v1.JobSchedulingConfiguration`, reusing the
`scheduling.k8s.io/v1alpha3` building-block types directly. The generated `Workload` and
`PodGroup` resources use the served `scheduling.k8s.io/v1beta1` API:

```go
type DeploymentSchedulingConfiguration struct {
    // SchedulingPolicy selects the scheduling mode. Defaults to Basic when gang
    // is not specified. The user may set gang.minCount explicitly. When unset,
    // the controller derives minCount from replicas.
    // +optional
    // +k8s:optional
    // +k8s:update=NoSet
    // +k8s:update=NoUnset
    SchedulingPolicy *WorkloadPodGroupSchedulingPolicy

    // SchedulingConstraints carries topology placement rules.
    // +optional
    // +k8s:optional
    // +k8s:immutable
    SchedulingConstraints *WorkloadPodGroupSchedulingConstraints

    // DisruptionMode (single | all) is passed through to the PodGroup
    // and consumed by scheduler preemption logic.
    // +optional
    // +k8s:optional
    // +k8s:immutable
    DisruptionMode *WorkloadPodGroupDisruptionMode

    // ResourceClaims declares DRA ResourceClaims shared across all pods
    // of the gang (allocated once to the PodGroup, not per-pod). Max 4
    // entries. Immutable after creation.
    // +optional
    // +listType=map
    // +listMapKey=name
    // +k8s:maxItems=4
    // +k8s:immutable
    ResourceClaims []WorkloadPodGroupResourceClaim
}
```

### Feature Gate and RBAC

The `Scheduling` field is gated by `WorkloadWithDeployment` (Alpha, default off). The gate depends
on `GenericWorkload` being enabled. When `WorkloadWithDeployment` is disabled, requests that set
`spec.scheduling` are rejected.

The `deployment-controller` ClusterRole grants `get`, `list`, `watch`, `create`, `update`, and
`patch` on `scheduling.k8s.io/workloads`.

The `replicaset-controller` ClusterRole grants:

- `get`, `list`, and `watch` on `scheduling.k8s.io/workloads`.
- `get`, `list`, `watch`, `create`, `update`, `patch`, and `delete` on
  `scheduling.k8s.io/podgroups`.


### Controller Changes

#### Creation Ordering

For each scheduling-enabled ReplicaSet revision:

1. **Deterministic naming.** The Workload is named `<deployment.Name>` (one per Deployment).
   The PodGroup is named `<deployment.Name>-<podTemplateHash>` (one per ReplicaSet revision).
   The pod-template hash makes PodGroup naming stable across controller restarts and identical
   for the same revision.

2. **Template injection.** Set
   `pod.spec.schedulingGroup.podGroupName` on the ReplicaSet pod template so every pod created
   by the ReplicaSet references the correct PodGroup.

3. **Workload creation.** Call `ensureWorkloadForDeployment` (get-or-create) to instantiate the
   Deployment-owned Workload. The Workload's scheduling configuration is derived from the
   Deployment. If the user set gang `minCount`, that value is used. Otherwise `minCount` is
   derived from the replica count, defaulting to 1 at zero replicas.

4. **ReplicaSet creation.** Create or update the ReplicaSet with the injected scheduling-group
   reference.

5. **PodGroup creation.** Before creating pods, the ReplicaSet controller calls
   `ensurePodGroupForReplicaSet` (get-or-create) to instantiate the ReplicaSet-owned PodGroup
   from the Workload's sole PodGroupTemplate. The resulting `gang.minCount` is inherited from
   the Workload template and does not necessarily match the ReplicaSet's desired replica count.

6. **Scale-to-zero cleanup.** When the ReplicaSet has zero desired replicas,
   `ensurePodGroupForReplicaSet` calls `deletePodGroupForReplicaSet` to remove the PodGroup.
   When the ReplicaSet scales positive again, the PodGroup is recreated
   before creating pods.

#### workloadbuilder Integration

The Deployment controller uses the shared `workloadbuilder` library, also used by Job, to compile
the Deployment's `spec.scheduling` into one Workload with one PodGroupTemplate. The builder
preserves the configured scheduling constraints, disruption mode, and resource claims while
using the user's explicit `minCount` when set, or deriving it from the Deployment's replica count.

The ReplicaSet controller uses the same builder to materialize one PodGroup from the Workload's
sole PodGroupTemplate. The runtime PodGroup inherits its gang `minCount` from the Workload's
PodGroupTemplate. At zero replicas, the PodGroup is deleted because no pods exist to schedule.

#### EqualIgnoreHash

The controller injects `SchedulingGroup` into the ReplicaSet pod template, but it is absent from
the Deployment template. Without excluding this field from the template-equality check, every
reconcile would misdetect a template drift, driving endless collisionCount / ReplicaSet / PodGroup
churn. This exclusion is unconditional (not gated), because a stored ReplicaSet may carry the
field even after the gate is turned off.

#### Scaling and HPA

Scaling through the `/scale` subresource updates the Deployment's desired replica count. For a
positive replica count, the Deployment controller first reconciles the Workload's PodGroupTemplate
so its gang `minCount` reflects the resolved value (user-set or derived from replicas). It then
scales the relevant ReplicaSets.

The ReplicaSet controller reconciles each ReplicaSet-owned PodGroup before managing its pods. For a
positive ReplicaSet size, it creates or updates the PodGroup and inherits its runtime gang
`minCount` from the Workload's PodGroupTemplate. When a ReplicaSet reaches zero replicas, its
PodGroup is deleted because no pods exist to schedule. When it scales positive again, the PodGroup is
recreated before new pods are created.

Scaling a positive ReplicaSet does not delete its existing PodGroup. The controller updates its
quorum before creating additional pods. If the new gang cannot be scheduled, the new pods remain
pending while existing pods continue running.

### Mutability and Validation

`spec.scheduling` is validated in three complementary layers:

1. **Declarative validation (DV) on the building blocks** owns the structural rules and most of
   the immutability. Because the Deployment API embeds the versioned
   `scheduling.k8s.io/v1alpha3` building blocks directly, their DV markers apply unchanged.
2. **Hand-written Deployment validation** covers the cross-cutting rules DV cannot express:
   - **`gang.minCount` must not exceed `replicas`.** If the user sets `minCount` and it exceeds
     the current replica count, the request is rejected.
   - **`RollingUpdate` with gang is rejected.** Gang scheduling requires `Recreate` strategy
     for Alpha.
3. **`workloadbuilder` semantic validation** owns the consistency rules that must stay identical
   to what the controller compiles. Validation builds the same `WorkloadItem` tree the controller
   does and calls `NewBuilder(...).Validate()`, running the builder's allow-list checks. In-tree
   it is constructed with `BuildOptions{DisableDeclarativeValidation: true}` because the
   API server already ran DV on the versioned building blocks.

### Test Plan

#### Unit Tests

- Building the Deployment-owned Workload: scheduling constraints, disruption mode, resource claims,
  owner reference, and positive gang `minCount` are compiled correctly.
- ReplicaSet scheduling reconciliation: the ReplicaSet-owned PodGroup is created from the Workload's
  sole PodGroupTemplate, its runtime `minCount` comes from the Workload's PodGroupTemplate,
  and it is reconciled before pod creation.
- ReplicaSet scale-to-zero behavior: the PodGroup is deleted at zero replicas and recreated before
  pods are created when the ReplicaSet scales positive.
- Validation: `minCount > replicas` rejected, `RollingUpdate` with gang rejected,
  immutability violations rejected, and resource-claim structural violations rejected.
- Idempotent reconciliation: missing Workloads and PodGroups are recreated without duplicates.

#### Integration Tests

- Create a gang Deployment and verify one Deployment-owned Workload and one ReplicaSet-owned
  PodGroup exist, with the resolved `minCount` and the ReplicaSet template carrying the correct
  `schedulingGroup.podGroupName`.
- Verify creation ordering: the Workload exists before the ReplicaSet is created, and the PodGroup
  exists before the ReplicaSet creates pods.
- Scale up and down and verify the Workload template and active PodGroup receive the correct
  positive `minCount`.
- Scale to zero and verify the Workload retains the user's `minCount` (or defaults to 1) while
  the PodGroup is deleted. Scale positive again and verify the Workload is patched and the
  PodGroup is recreated before pods are created.
- Delete the Deployment and verify the Workload and ReplicaSet-owned PodGroup are eventually
  garbage-collected.
- Verify old PodGroups and template-backed ResourceClaims are released when old ReplicaSets reach
  zero.

#### E2E Tests

- Gang Deployment with `Recreate`: all replicas bind atomically or none bind.
- Admission rejection for `RollingUpdate` with gang and `minCount > replicas`.
- Topology placement: gang lands in a single domain.
- `disruptionMode.single` versus `all`: preemption behavior differs as expected.
- ResourceClaims: one template-backed claim per PodGroup, released when the PodGroup is deleted;
  named claims remain pinned to their allocated node.
- Controller crash recovery: deterministic naming yields idempotent recovery without duplicates.
- Scale-to-zero-and-back: the Workload retains the user's `minCount` (or defaults to 1).
  The PodGroup is deleted and recreated.
- Scaling up does not disturb existing running pods: only new pods wait for the updated gang quorum.
- Single-pod replacement: a deleted pod is replaced without recreating the PodGroup.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `WorkloadWithDeployment` feature gate (default: disabled).
- The Deployment controller creates one Workload per Deployment, and the ReplicaSet controller
  creates one PodGroup per ReplicaSet when the gate is enabled and `spec.scheduling` is set.
- ReplicaSet-owned PodGroups are created before the ReplicaSet creates pods.
- Gang scheduling with user-settable `minCount`, topology constraints, disruption mode, and
  resourceClaims are wired end-to-end.
- Admission validation rejects `minCount > replicas`, `RollingUpdate` with gang, immutability
  violations, and resourceClaims structural violations.
- Unit, integration, and E2E tests cover creation ordering, validation, scaling, scale-to-zero
  cleanup, and garbage collection.

#### Beta

- Promote `WorkloadWithDeployment` to enabled by default.
- Improve observability of scheduling failures through Deployment conditions and dedicated
  metrics (e.g., Workload/PodGroup creation latency).
- Investigate scheduler-side failure reporting for stuck topology domains, dependent on
  sig-scheduling exposing standardized signals distinguishing terminal from transient
  unschedulability.
- Re-evaluate whether elastic gang semantics (multiple PodGroups per ReplicaSet for
  `minCount < replicas`) should be supported.
- Re-evaluate `RollingUpdate` support with gang scheduling.
- Re-evaluate Deployment-side admission rejection of `spec.scheduling.resourceClaims` when
  `DRAWorkloadResourceClaims` is off (currently a silent semantic downgrade).
- E2E test coverage for the full scenario matrix.

#### GA

TBD

### Upgrade Strategy

With `WorkloadWithDeployment` disabled, new requests that set `spec.scheduling` are rejected, and
Deployments without scheduling configuration behave as they do today.

Disabling the gate after scheduling-enabled Deployments already exist does not remove their stored
Workload, PodGroup, ReplicaSet template, or pod references. Existing Workloads and PodGroups remain
owned by their Deployment and ReplicaSet, respectively, and are cleaned up when those owners are
deleted. Existing pods retain the scheduling-group reference they were created with. Re-enabling
the gate allows the controllers to resume normal scheduling-object reconciliation.

### Version Skew Strategy

The feature requires `GenericWorkload` and the `scheduling.k8s.io` API versions to be active on
the API server. If the API server does not serve these resources, the controller's create/patch
calls fail and the Deployment sync retries with backoff. No scheduling objects are compiled until
the API server is upgraded. Pods created without `schedulingGroup` schedule normally through the
default path.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `WorkloadWithDeployment`
  - Components depending on the feature gate:
    - kube-controller-manager
    - kube-apiserver

###### Does enabling the feature change any default behavior?

No. The feature is opt-in via `spec.scheduling`. Deployments without `spec.scheduling` are
unaffected. No scheduling objects are created unless the user explicitly sets the field.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. With the gate disabled on kube-apiserver, new requests that set `spec.scheduling` are rejected; with the
gate disabled on kube-controller-manager, the controllers stop reconciling scheduling objects.
Existing PodGroups remain until their owning ReplicaSet is garbage-collected; the Workload remains
until the Deployment is deleted.

###### What happens if we reenable the feature if it was previously rolled back?

When the feature is re-enabled:
- Deployments with a stored `spec.scheduling` value resume reconciliation on their next sync.
- Existing Workload/PodGroup objects are discovered via deterministic naming and reused.
- If only a partial set exists (e.g., Workload but no PodGroup from a crash mid-creation), the
  controller completes the missing object on its next sync.

###### Are there any tests for feature enablement/disablement?

Yes. Unit and integration tests cover feature gate on/off behavior.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

- If the API server doesn't serve the Workload and PodGroup APIs, the Deployment and ReplicaSet
  controllers cannot persist scheduling objects and requeue with backoff until the APIs are available.
- Already running Deployments are not affected by enabling the feature; pods already scheduled
  continue to run.
- Disabling the gate does not remove existing Workloads, PodGroups, ReplicaSet template references,
  or pod references. Existing ReplicaSets continue using their stored pod template; re-enable the
  gate to resume scheduling-object reconciliation.

###### What specific metrics should inform a rollback?

- `deployment_sync_duration_seconds`: significant increase may indicate issues with Workload/PodGroup
  creation.
- Increased error rate in deployment-controller logs for `scheduling.k8s.io` API calls.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested manually as part of alpha release.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- `kubectl get workloads -A` will show Workload objects created by the Deployment controller.
- `kubectl get podgroups -A` will show PodGroup objects created by the ReplicaSet controller for
  each active Deployment revision.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Condition name: `Available=False` with reason `MinimumReplicasUnavailable` when required replicas
    are unavailable; `Progressing=False` with reason `ProgressDeadlineExceeded` when placement does
    not make progress before the deadline.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

TBD

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

TBD

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A dedicated metric for Workload/PodGroup creation latency per Deployment would be useful for Beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

Yes. The `scheduling.k8s.io` API group must be served (requires `GenericWorkload` feature gate
enabled on the API server).

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. The controllers use Workload and PodGroup informers and listers for cached reads. API writes
include:
- Creating one Workload for each scheduling-enabled Deployment.
- Patching the Deployment's Workload when a positive replica count changes.
- Creating one PodGroup for each active ReplicaSet revision, or recreating it after scale-up from
  zero.
- Patching a ReplicaSet-owned PodGroup when its owner or runtime `minCount` needs reconciliation.
- Deleting the PodGroup when its ReplicaSet reaches zero replicas.

###### Will enabling / using this feature result in introducing new API types?

No. Workload and PodGroup are introduced by [KEP-6089]; this KEP only creates instances.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. Each Deployment with `spec.scheduling` creates 1 Workload per Deployment (~500 bytes) and
1 PodGroup per ReplicaSet (~500 bytes), and each Pod gains a `schedulingGroup` field (~100 bytes).

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Scheduling-enabled Deployments and ReplicaSets may incur additional reconciliation work while
Workload and PodGroup objects are created or updated. Informer-backed reads limit the steady-state
overhead, while ordinary Deployments and ReplicaSets are unaffected. The impact should be measured
during Alpha.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

The feature is opt-in in Alpha, so the additional overhead is limited to scheduling-enabled
Deployments. Each such Deployment adds one long-lived Workload, one PodGroup for each active
ReplicaSet revision, informer cache entries for Workloads and PodGroups, and reconciliation work
in both controllers.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature is purely control-plane and does not affect node resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

- Deployment and ReplicaSet controllers cannot create Workloads or PodGroups.
- Retries with exponential backoff when kube-apiserver recovers.
- Existing Deployments with scheduling objects continue to run.

###### What are other known failure modes?

- Gang cannot place due to insufficient cluster resources: Deployment reports
  `Available=False` / `ProgressDeadlineExceeded`. No automatic recovery.

###### What steps should be taken if SLOs are not being met to determine the problem?

- Verify `WorkloadWithDeployment` and `GenericWorkload` are enabled on all control plane
  components.
- Check controller-manager logs for errors related to Workload/PodGroup creation.
- Check resource constraints since gang scheduling may fail if the cluster doesn't have
  sufficient resources.

## Implementation History

- 2026-08: KEP created for Alpha targeting v1.38.

## Drawbacks

- `Recreate` strategy terminates all old pods before creating new ones, causing downtime
  during rollouts.

- Permanent topology-domain binding can strand replacement pods in Pending with no automatic
  recovery. The in-tree remedy is triggering a new rollout revision.

- Post-bind runtime failures (e.g., bad image) retain node capacity while individual pods crash;
  the gang holds its reservations even though no useful work is happening.

- With `Recreate` strategy, template-backed ResourceClaims are released when the old ReplicaSet
  reaches zero and its PodGroup is deleted. `revisionHistoryLimit` does not retain the claim.

- Named ResourceClaims pin all revisions to one node, risking deadlock when node capacity is tight
  during rollouts.

## Alternatives

**Delete the Workload when replicas reach zero.** Instead of retaining the Deployment-owned
Workload at zero replicas (with the user's explicit `minCount` or a default of 1), the controller
would delete the Workload and recreate it when the Deployment scales positive again.

*Advantages:* No Workload exists while the Deployment has zero replicas. The Workload and its
PodGroupTemplate are recreated from the current Deployment configuration on scale-up.

*Tradeoffs:* Deleting and recreating the Workload adds API operations and creates another
scale-to-zero/scale-up lifecycle transition. The current design retains one stable
Deployment-owned Workload and deletes only the ReplicaSet-owned runtime PodGroup, avoiding
Workload churn while the retained `minCount` has no runtime effect at zero replicas.

**One PodGroup per Deployment.** Rejected: topology binding is permanent per PodGroup, so a stuck
gang cannot re-place in a different domain without a new revision. Additionally, old and new
rollout gangs would collide within a single PodGroup.

**User-managed PodGroups (status quo).** Rejected: fragile ordering (pods created before PodGroup
hang silently), no garbage collection, and no integration with scaling or rolling updates.

**Controller-derived `minCount` only (no user override).** The controller always sets
`minCount = replicas` and rejects user-provided values. This was the original Alpha design.
It was changed to allow user-set `minCount` (validated to not exceed `replicas`) based on
feedback that tying `minCount` to `replicas` is too restrictive for workloads that can
tolerate partial placement.

**Deployment controller creates PodGroups.** The Deployment controller would create both the
Deployment-owned Workload and each revision's PodGroup, then the ReplicaSet controller would only
create pods.

*Advantages:* The ReplicaSet controller remains unaware of Workload and PodGroup APIs, requiring no
additional scheduling informers or RBAC permissions. Scheduling-object creation stays centralized
in the Deployment controller.

*Tradeoffs:* The Deployment controller must create a PodGroup before the ReplicaSet has a UID, so
the PodGroup requires temporary Deployment ownership and later ownership transfer. This adds an
ownership-transfer step and requires recovery if that transfer is interrupted. It also couples
Deployment reconciliation to revision-specific PodGroup lifecycle.

[KEP-4671]: https://kep.k8s.io/4671
[KEP-5547]: https://kep.k8s.io/5547
[KEP-5710]: https://kep.k8s.io/5710
[KEP-5732]: https://kep.k8s.io/5732
[KEP-6089]: https://kep.k8s.io/6089
