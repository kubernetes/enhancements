# KEP-5547: Integrate Workload APIs with Job Controller

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Job Integration - API Usage Examples](#job-integration---api-usage-examples)
    - [Example 1: Gang scheduling with zone topology and atomic disruption](#example-1-gang-scheduling-with-zone-topology-and-atomic-disruption)
    - [Example 2: Backward Compatibility and Defaulting (Implicit Opt-Out)](#example-2-backward-compatibility-and-defaulting-implicit-opt-out)
    - [Example 3: CronJob with Gang Scheduling](#example-3-cronjob-with-gang-scheduling)
  - [User Stories](#user-stories)
    - [ML Training Job with Gang Scheduling](#ml-training-job-with-gang-scheduling)
    - [Backward-Compatible Standard Batch Job](#backward-compatible-standard-batch-job)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
    - [Constraints](#constraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Core Principles &amp; Assumptions](#core-principles--assumptions)
  - [Job API Changes](#job-api-changes)
    - [Go Package Placement &amp; Graduation](#go-package-placement--graduation)
  - [Integration with the workloadbuilder Library](#integration-with-the-workloadbuilder-library)
    - [Library Dependency and Packaging](#library-dependency-and-packaging)
    - [Building the Logical Tree and Compiling the <code>Workload</code>](#building-the-logical-tree-and-compiling-the-workload)
    - [API Validation via the <code>workloadbuilder</code> Library](#api-validation-via-the-workloadbuilder-library)
    - [Instantiating the runtime <code>PodGroup</code>](#instantiating-the-runtime-podgroup)
    - [Reconcile Integration and Error Handling](#reconcile-integration-and-error-handling)
  - [Job Controller Changes](#job-controller-changes)
    - [Workload and PodGroup Discovery](#workload-and-podgroup-discovery)
    - [Controller Workflow](#controller-workflow)
    - [OwnerReferences Relationship](#ownerreferences-relationship)
    - [Defaulting Rules](#defaulting-rules)
    - [Object Creation Order](#object-creation-order)
    - [Handling Updates and Mutability](#handling-updates-and-mutability)
    - [Reconciliation Flow upon Updates](#reconciliation-flow-upon-updates)
    - [Suspend and Resume](#suspend-and-resume)
  - [Interaction with a BYO PodGroup](#interaction-with-a-byo-podgroup)
  - [Naming Conventions](#naming-conventions)
  - [Deletion and Garbage Collection](#deletion-and-garbage-collection)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.36)](#alpha-v136)
    - [Alpha (v1.37)](#alpha-v137)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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
  - [Bring-your-own Workload](#bring-your-own-workload)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [x] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP integrates the Workload-aware Scheduling (WAS) APIs (`Workload` and `PodGroup`) into the
`batch/v1` Job by adding a user-facing `spec.scheduling` field, allowing users to express explicit
scheduling intent such as gang scheduling[^1], topology co-location, and disruption policies.

The feature has gone through two alpha releases:

- **v1alpha1 (v1.36)** intentionally bypassed a user-facing API: the controller inferred a
  hardcoded `Gang` policy from the Job's type (parallel Jobs with indexed completion mode), with
  `minCount` fixed to `parallelism`.
- **v1alpha2 (v1.37)** replaced that automatic, controller-inferred model with an explicit,
  user-driven design that separates scheduling policy from workload structure. It introduced 
  the `spec.scheduling` field on the Job API, built on the reusable scheduling building blocks 
  and the shared `workloadbuilder` translation library defined in [KEP-6089], keeping the Job 
  integration consistent with the rest of the ecosystem rather than reinventing bespoke logic.

The Job controller acts as a translator, compiling `spec.scheduling` into the underlying
`Workload`/`PodGroup` objects. When `spec.scheduling` is omitted, it defaults to `Basic` 
scheduling, so the scheduling outcome of existing Jobs is preserved.

For v1.38 the integration is promoted to Beta and the `WorkloadWithJob` feature gate is enabled 
by default. Because the `spec.scheduling` field is embedded in the GA `batch/v1` API, its building 
blocks graduate straight into `scheduling.k8s.io/v1` (with Go type aliases left in `v1alpha3`) as 
described in [KEP-6089]. The Beta revision also resolves the design questions the v1alpha2 left
open and the controller exposes metrics for the `Workload`/`PodGroup` lifecycle. Default enablement
is conditional on the `GenericWorkload` gate ([KEP-4671]) being enabled by default in the same
release, since `WorkloadWithJob` depends on it.

## Motivation

The Kubernetes Job Controller historically created pods independently without workload-aware
scheduling constraints. This is a challenge for parallel applications (i.e., AI/ML training
workloads, MPI jobs) that require all pods to be scheduled and run together or none (gang
scheduling[^1]). The v1.36 alpha brought gang scheduling to the Job controller, but it did so by
inferring a hardcoded `Gang` policy from the Job's type rather than from explicit user intent.

Users have diverse use cases and require the ability to express explicit intent, such as opting
in or out of gang scheduling, requesting specific topologies, or configuring disruption policies
for their workloads. [KEP-6089] standardizes the reusable scheduling building blocks
(introduced as `scheduling.k8s.io/v1alpha3`, graduating to `scheduling.k8s.io/v1` in v1.38) and a
shared `workloadbuilder` translation library so that workload controllers can expose these
features consistently. This KEP integrates those building blocks into the core `Job` API. The same 
building blocks have been adopted by out-of-tree controllers (JobSet, LeaderWorkerSet, Kubeflow 
TrainJob, kubeRay) and proposed for `Deployment` and `StatefulSet`, which is the ecosystem feedback 
the alpha was meant to collect.

### Goals

- Add a user-facing `spec.scheduling` (`JobSchedulingConfiguration`) field to the `batch/v1` Job,
  embedding the `scheduling.k8s.io` building blocks (`schedulingPolicy`, `schedulingConstraints`, 
  `disruptionMode`, `resourceClaims`) so users can express explicit scheduling intent.
- Default to `Basic` scheduling when `spec.scheduling` is omitted, so the observable scheduling 
  outcome of existing Jobs is preserved (no all-or-nothing gate, and any number of schedulable 
  pods proceed to binding). Following the [KEP-6089], the controller still materializes a `Basic`
  `Workload`/`PodGroup`, which routes these pods through the Workload Scheduling Cycle 
  (batched scheduling and workload-aware preemption) without enforcing minCount.
- Let users opt in to `Gang` scheduling, with `minCount` defaulting to `parallelism` when omitted.
- Compile `spec.scheduling` into `Workload`/`PodGroup` objects via the shared `workloadbuilder`
  library instead of bespoke controller logic.
- Support mutable `spec.scheduling.schedulingPolicy.gang.minCount` for elastic scaling, while keeping all
  other `spec.scheduling` fields immutable after creation. This relies on [KEP-4671] that makes 
  `minCount` in `PodGroup`/`PodGroupTemplate` mutable in v1.37 to support workload scaling.
- When the Job is not the root of the workload tree (the `OwnerReference` refers to a parent
  controller that compiles and owns the `Workload`), defer `Workload` management to that parent,
  preserving the root-controller-as-compiler principle. A parent may own the `Workload` while still
  delegating `PodGroup` management to the Job (e.g., a `Job` running under a `TrainJob` that does not
  know about Jobs). The parent signals this split via [KEP-6089]'s downward-mapping annotations, so 
  a non-root controller can still create and manage the `PodGroup` for its own pods.
- Ensure proper ordering of `Workload` → `PodGroup` → `Pod` creation.
- Release scheduler-side resources while a Job is suspended by deleting the runtime `PodGroup`
  and recreating it on resume, keeping the `Workload` template in place.

### Non-Goals

- Multi-level / nested composite (`CompositePodGroup`) structures, since this KEP covers 
single-level, flat `Job` workloads only.
- Implementing the integration in composite controllers (`JobSet`, `LWS`, `TrainJob`). Those 
  are pursued independently in their own repositories.
- Defining the `scheduling.k8s.io` building-block API or the `workloadbuilder` library 
  itself (owned by [KEP-6089] and consumed here).
- Supporting a user pre-created `Workload` for a standalone Job ("bring your own `Workload`").
  With `spec.scheduling` covering every shape a Job-owned `Workload` can have, a second source of
  truth adds reconciliation complexity without new expressiveness. See
  [Alternatives](#alternatives). A user pre-created `PodGroup` referenced through
  `spec.template.spec.schedulingGroup` remains supported.

## Proposal

This proposal builds on the recently introduced Workload-aware Scheduling enhancements. We assume
the reader is acquainted with the following KEPs:

- [KEP-4671]: Gang Scheduling.
- [KEP-5710]: Workload-aware preemption.
- [KEP-5732]: Topology-aware workload scheduling.
- [KEP-6089]: WAS Controller APIs.

The Job controller is extended to compile the user's scheduling intent into `Workload` and
`PodGroup` objects as part of its pod-management lifecycle, so that pods belonging to a Job are
scheduled according to the requested policy before they are created. The intent is expressed
through a new `spec.scheduling` field.

The key design principles are:

- One `Job` maps to one `PodGroup` representing a single group of pods. The `PodGroup` 
always links to a `Workload` via a `PodGroupTemplate`:
  * For a root Job it links to the `Workload` the controller compiles itself
  * For a non-root Job it links to the parent-owned `Workload`
  * The `PodGroup` links to a parent `CompositePodGroup` instance only when the parent 
  supplies the `scheduling.k8s.io/parent-compositepodgroup` annotation
- The scheduling policy comes from the user's `spec.scheduling`, not from the Job's type. When
  `spec.scheduling` is omitted, the controller defaults to the `Basic` policy.
- Following the [KEP-6089], the controller always materializes scheduling objects (a `Workload`
  and/or `PodGroup`) for an *eligible* Job — a Job the controller is responsible for when the gate is
  on: a standalone/root Job, or a non-root Job whose parent delegates the `PodGroup` (a Job whose
  parent owns both objects is skipped). This holds even for the `Basic` scheduling policy.
- For `Gang`, an omitted `minCount` defaults to the Job's `parallelism`. `minCount` is mutable to support elastic scaling; all other
  `spec.scheduling` fields are immutable after creation.
- The Job controller does not create a `Workload` when the Job carries an `OwnerReference` to a
  parent controller that compiles and owns the `Workload` (e.g., `JobSet`). Such controllers set
  this `OwnerReference` when they create the Job. Whether the Job controller also skips `PodGroup`
  creation depends on what the parent delegates: if the parent injects the annotation 
  ([KEP-6089]), the Job controller still creates and manages the runtime `PodGroup` for its own 
  pods, mapping them to the parent's named `PodGroupTemplate` and attaching to the parent instance. 
  If no such annotation is present, the parent owns both objects and the Job controller skips both.
- Jobs created by `CronJob` are standalone (no parent-workload `OwnerReference`); the Job controller
  creates one `Workload` and one `PodGroup` per Job for them based on each Job's `spec.scheduling`.
- The controller only manages `Workload`/`PodGroup` objects it created itself, identified by a
  controller `ownerReference` to the Job. The `Workload` is the durable scheduling template and
  lives as long as the Job. The runtime `PodGroup` is deleted while the Job is suspended and
  recreated on resume.

The `spec.scheduling` field embeds the building blocks from `scheduling.k8s.io/v1`, while the 
runtime `Workload`/`PodGroup` objects the controller creates are served from 
`scheduling.k8s.io/v1beta1` ([KEP-4671]). The examples below use those versions.

### Job Integration - API Usage Examples

#### Example 1: Gang scheduling with zone topology and atomic disruption

A distributed training `Job` whose 4 pods must schedule together (all-or-nothing), co-locate within
the same availability zone, and be disrupted together (if one pod is preempted, the whole group is):

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: <job-name>
  namespace: training
spec:
  parallelism: 4
  completions: 4
  completionMode: Indexed
  scheduling:               # New API field - scheduling intent
    schedulingPolicy:
      gang: {}              # minCount omitted -> defaults to parallelism (4)
    schedulingConstraints:
      topology:
        - level: "topology.kubernetes.io/zone"
    disruptionMode:
      all: {}               # entire group must be disrupted together
  template:
    spec:
      containers:
      - name: trainer
        image: training-image:latest
        resources:
          limits:
            nvidia.com/gpu: 1
```

The Job controller compiles this intent into a `Workload` and its runtime `PodGroup`:

```yaml
apiVersion: scheduling.k8s.io/v1beta1
kind: Workload
metadata:
  name: <job-name>-<hash>
  namespace: training
  ownerReferences:
  - apiVersion: batch/v1
    kind: Job
    name: <job-name>
    uid: <job-uid>
    controller: true
spec:
  controllerRef:
    apiVersion: batch/v1
    kind: Job
    name: <job-name>
  podGroupTemplates:
  - name: <podGroupTemplateName>
    schedulingPolicy:
      gang:
        minCount: 4         # defaulted from Job.spec.parallelism
    schedulingConstraints:
      topology:
        - level: "topology.kubernetes.io/zone"
    disruptionMode:
      all: {}
---
apiVersion: scheduling.k8s.io/v1beta1
kind: PodGroup
metadata:
  name: <workload-name>-<podGroup-template-name>-<hash>
  namespace: training
  ownerReferences:
  - apiVersion: batch/v1
    kind: Job
    name: <job-name>
    uid: <job-uid>
    controller: true
  - apiVersion: scheduling.k8s.io/v1beta1
    kind: Workload
    name: <workload-name>
    uid: <workload-uid>
spec:
  podGroupTemplateRef:
    workload:
      workloadName: <workload-name>
      podGroupTemplateName: <podGroup-template-name>
  schedulingPolicy:
    gang:
      minCount: 4
```

#### Example 2: Backward Compatibility and Defaulting (Implicit Opt-Out)

A standard Job that omits the `scheduling` block. It defaults to `Basic` scheduling. Per 
the [KEP-6089], the controller does not impose all-or-nothing, so the scheduling 
outcome matches a standard Job (batched scheduling cycle, no minCount enforcement):

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: <job-name>
  namespace: batch
spec:
  parallelism: 10
  completions: 10
  # The scheduling block is omitted, which defaults to Basic scheduling. 
  # This acts as an implicit opt-out from gang scheduling.
  template:
    spec:
      containers:
      - name: processor
        image: processor-image:v1
```

This compiles into a `Basic` scheduling policy:

```yaml
apiVersion: scheduling.k8s.io/v1beta1
kind: Workload
metadata:
  name: <job-name>-<hash>
  namespace: batch
  ownerReferences:
  - apiVersion: batch/v1
    kind: Job
    name: <job-name>
    uid: <job-uid>
    controller: true
spec:
  controllerRef:
    apiVersion: batch/v1
    kind: Job
    name: <job-name>
  podGroupTemplates:
  - name: <podGroup-template-name>
    schedulingPolicy:
      basic: {}
---
apiVersion: scheduling.k8s.io/v1beta1
kind: PodGroup
metadata:
  name: <workload-name>-<podGroup-template-name>-<hash>
  namespace: batch
  ownerReferences:
  - apiVersion: batch/v1
    kind: Job
    name: <job-name>
    uid: <job-uid>
    controller: true
  - apiVersion: scheduling.k8s.io/v1beta1
    kind: Workload
    name: <workload-name>
    uid: <workload-uid>
spec:
  podGroupTemplateRef:
    workload:
      workloadName: <workload-name>
      podGroupTemplateName: <podGroup-template-name>
  schedulingPolicy:
    basic: {}
```

#### Example 3: CronJob with Gang Scheduling

A `CronJob` that periodically runs a gang-scheduled training Job. Each `Job` created by the
`CronJob` is treated as standalone. `CronJob` does not create or manage `Workload` objects,
the Job controller creates a separate `Workload` and `PodGroup` per `Job`. These objects are
garbage-collected when each `Job` completes or is deleted.

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: periodic-training
  namespace: training
spec:
  schedule: "0 */6 * * *"
  jobTemplate:
    spec:
      parallelism: 4
      completions: 4
      completionMode: Indexed
      scheduling:
        schedulingPolicy:
          gang: {}            # minCount defaults to parallelism (4) per Job
      template:
        spec:
          containers:
          - name: trainer
            image: training-image:latest
            resources:
              limits:
                nvidia.com/gpu: 1
```

Each `Job` created by this `CronJob` produces its own `Workload` and `PodGroup`, compiled from
the Job's `spec.scheduling`.
If the `CronJob`'s `jobTemplate` omits the `scheduling` block, each `Job` defaults to `Basic`.

In all cases, the Job controller then creates the pods and sets the `schedulingGroup` field so the
scheduler can associate each pod with its `PodGroup`:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: <job-name>-<random-suffix>
  namespace: <namespace>
  ownerReferences:
  - apiVersion: batch/v1
    kind: Job
    name: <job-name>
    uid: <job-uid>
    controller: true
  - apiVersion: scheduling.k8s.io/v1beta1
    kind: PodGroup
    name: <podGroup-name>
    uid: <podGroup-uid>
spec:
  schedulingGroup:
    podGroupName: <workload-name>-<podGroup-template-name>-<hash>
  containers:
  - name: ...
```

### User Stories

#### ML Training Job with Gang Scheduling

As a machine learning engineer, I want to run a distributed training job with 8 workers that must
all be scheduled together. I set `spec.scheduling.schedulingPolicy.gang` on the Job (optionally with a
topology constraint to co-locate the workers), so that if only 7 workers can be scheduled, no pods
start and no resources are wasted. I do not have to set `parallelism` and `completions`
in a specific way to "qualify" for gang scheduling; I declare my intent explicitly.

#### Backward-Compatible Standard Batch Job

As a data engineer, I want to run a batch processing job that processes files independently without
gang scheduling requirements. I omit `spec.scheduling` entirely (or set `spec.scheduling.schedulingPolicy.basic`
explicitly for the same effect), so the Job defaults to `Basic` scheduling. The observable scheduling
outcome matches a standard Job, while a `Basic` `Workload`/`PodGroup` is still materialized, giving me consistent objects to observe its scheduling state.

### Notes/Constraints/Caveats

#### Constraints

- The integration targets single-level `Job` workloads: one `Job` maps to one `PodGroup`, and all
  pods in the `Job` share a single scheduling policy. A Job-owned `Workload` therefore always has
  exactly one `PodGroupTemplate`. There is no Job use case we know of that needs more than one 
  `PodGroupTemplate`, heterogeneous groups belong in a composite controller.
- `spec.scheduling.schedulingPolicy.gang.minCount` is mutable to support elastic scaling ([KEP-4671]); all 
  other `spec.scheduling` fields are immutable after creation.
- The Job controller creates `Workload`/`PodGroup` objects for every eligible Job, including
  `Basic` ones. The only way to avoid the objects entirely is to disable the feature gate. 
  By default, an end user gets the original scheduling outcome even though a `Basic` 
  `Workload`/`PodGroup` is still created.
- The controller manages only objects it created (controller `ownerReference` to the Job). A
  pre-created `Workload` is not adopted. The supported bring-your-own path is a `PodGroup`
  referenced from `spec.template.spec.schedulingGroup`, in which case the controller creates
  nothing.

### Risks and Mitigations

- **Split-brain configuration.** A composite wrapper controller (such as `JobSet` or `TrainJob`)
  may expose its own scheduling fields while the child `Job` now also has native `spec.scheduling`
  fields, letting a user configure scheduling in two conflicting places. 
  * *Mitigation:* the parent controller remains the sole compiler of the workload tree and can map 
  its own fields onto the compiled `Workload`, strip/ignore the child's nested scheduling fields, 
  or reject requests that populate both. The Job controller cooperates by deferring `Workload`
  ownership whenever the Job carries an `OwnerReference` to a registered parent workload (replacing
  the v1.36 `spec.template.spec.schedulingGroup`-based opt-out). The parent then decides whether 
  the Job also defers `PodGroup` creation or manages its own `PodGroup` mapped to the parent's 
  `PodGroupTemplate`.

- **Increased object count.** Because the controller now materializes a `Workload`/`PodGroup` for
  every eligible Job, the number of scheduling objects grows relative to the v1.36 alpha, which 
  only created objects for inferred gang Jobs. With the gate enabled by default from v1.38, this 
  applies to every cluster, not only to those that opted in.
  * *Mitigation:* objects are small and garbage-collected with the Job; the Scalability section 
  quantifies the impact, the feature can still be disabled with the `WorkloadWithJob` gate, and performance tests
  covering Job creation throughput with the gate on are a Beta requirement.

- **Behavior change between alphas.** Jobs that were automatically gang-scheduled in v1.36 default
  to `Basic` in v1.37 unless the user sets `spec.scheduling.schedulingPolicy.gang`. 
  * *Mitigation:* gang is now an explicit opt-in; this is documented in the 
  [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy) and release notes.

- **Suspended Jobs and resource release.** In v1alpha2 the controller relied only on GC, which does not
  release resources (e.g., DRA claims, scheduler reservations) while a Job is suspended.
  * *Mitigation:* from Beta the controller deletes the runtime `PodGroup` once the suspended
  Job's pods are gone and recreates it on resume, so resources are released and the scheduler
  makes a fresh placement decision. The `Workload` template is retained. See
  [Suspend and Resume](#suspend-and-resume).

- **Silent loss of gang semantics on fallback.** When the controller finds more than one
  controller-owned `Workload` or `PodGroup` for a Job, or a Job-owned `Workload` whose shape no
  longer matches what the controller compiles (external mutation), it falls back to default
  pod-by-pod scheduling. For a `Gang` Job this means the all-or-nothing guarantee the user asked
  for is not enforced.
  * *Mitigation:* both cases require an external actor writing to controller-owned objects, and
  both are surfaced with a Warning event on the Job (`UnsupportedWorkloadStructure`) and counted 
  in `job_scheduling_object_syncs_total{result="error"}`.
  Blocking pod creation instead was considered and rejected for Beta because it turns an
  observability problem into a stuck Job. The decision will be revisited for GA based on Beta feedback.

- **Pod creation latency for gangs.** The Job controller creates pods in slow-start batches
  (1, 2, 4, ...), so a gang of `minCount` pods is fully created only after `ceil(log2(minCount))`
  rounds and the scheduler cannot admit the gang before then.
  * *Mitigation:* the scheduler, not the controller, enforces all-or-nothing, so the batching adds
  bounded latency rather than incorrect behavior. Batching stays unchanged for Beta. Performance tests 
  measure time-to-first-schedule for gang Jobs so the trade-off can be revisited with data.

## Design Details

### Core Principles & Assumptions

The integration follows the Workload-aware Scheduling design principles from [KEP-6089] for the 
single-level `Job`:

- **The Root Controller is the Compiler.** For a standalone `Job`, the Job controller is the
  root-most controller and is responsible for compiling, creating, and managing the
  scheduler-facing `Workload`. When a `Job` instead carries an `OwnerReference` to a parent 
  controller that compiles the `Workload` (e.g., `JobSet`), the Job controller observes 
  that linkage and *bypasses* compiling the `Workload`, so the parent remains the single 
  source of truth for workload structure and policy. Ownership of the runtime `PodGroup` 
  is decided separately and is not necessarily transferred with the `Workload`. Only in 
  the delegated case the Job controller creates and manages the `PodGroup` for its own 
  pods even though it does not own the `Workload`.
- **Universal Representation.** Standard pod-by-pod scheduling is a first-class policy (`Basic`).
  The controller always emits a `Workload`/`PodGroup` for an eligible Job, using `Basic` as the 
  backward-compatible default. `Basic` keeps the standard scheduling outcome, 
  while still participating in the Workload Scheduling Cycle, without enforcing minCount.
- **Sane Defaults and Escape Hatches.** A `Job` defaults to `Basic`.

### Job API Changes

To deliver native, typed Workload-aware Scheduling on core Kubernetes, we add a new `Scheduling`
field to `JobSpec`. This integration is the foundational, single-level implementation that
demonstrates the building blocks before out-of-tree controllers adopt them.

We introduce a new optional `Scheduling` field in `JobSpec` that embeds a curated composition of
the standardized building blocks:

```go
// API Group: batch/v1

// JobSpec defines the desired state of a Job.
type JobSpec struct {
    // ... existing fields ...

    // Scheduling defines the Workload-aware Scheduling configuration for this Job.
    // This field is beta-gated by the WorkloadWithJob feature gate (enabled by
    // default since v1.38).
    // +optional
    Scheduling *JobSchedulingConfiguration `json:"scheduling,omitempty"`
}

// JobSchedulingConfiguration composes the reusable WAS building blocks. The
// field names mirror the scheduler-facing PodGroupSpec 1:1 rather than being
// shortened, so the Job surface reads the same as the compiled PodGroup.
type JobSchedulingConfiguration struct {
    // SchedulingPolicy defines the gang or basic scheduling rules for this Job.
    // Exactly one of Basic or Gang must be set. Immutable after creation: the
    // policy may not be added or removed, and the basic/gang variant may not be
    // switched. Only gang.minCount may change.
    // +optional
    SchedulingPolicy *schedulingv1.WorkloadPodGroupSchedulingPolicy `json:"schedulingPolicy,omitempty"`

    // SchedulingConstraints defines topology co-location constraints for the
    // Job's pods. Immutable after creation.
    // +optional
    SchedulingConstraints *schedulingv1.WorkloadPodGroupSchedulingConstraints `json:"schedulingConstraints,omitempty"`

    // DisruptionMode specifies how the pods in this Job should be disrupted
    // (Single vs All). Immutable after creation.
    // +optional
    DisruptionMode *schedulingv1.WorkloadPodGroupDisruptionMode `json:"disruptionMode,omitempty"`

    // ResourceClaims specifies dynamic resource claims shared across the Job's
    // pods. Immutable after creation.
    // +optional
    ResourceClaims []schedulingv1.WorkloadPodGroupResourceClaim `json:"resourceClaims,omitempty"`
}
```

The building blocks are the versioned `scheduling.k8s.io/v1` types, embedded directly by 
both the internal and versioned `batch` API. There are no internal `batch`-owned copies 
of the building blocks; the internal `JobSchedulingConfiguration` references the `schedulingv1` 
types as-is, so a single `workloadbuilder` mapping serves both the API validation and the 
controller.

The `Scheduling` field is gated by the existing `WorkloadWithJob` feature gate. Standard
field-gating semantics apply: when the gate is disabled, the API server clears `spec.scheduling` on
create and ignores it on update (preserving an already-set value on the stored object), and the
field's validation and the controller's compilation (including its compile-time resolution of unset
fields; the api-server does not default `spec.scheduling`) only run when the gate is enabled. From
v1.38 the gate is enabled by default, so this path is only taken by clusters that opt out.

This typed, user-facing field replaces the v1.36 alpha's implicit mechanisms: the type-based
automatic policy inference and the `spec.template.spec.schedulingGroup`-based opt-out are no longer
how users express or suppress scheduling intent.

#### Go Package Placement & Graduation

Embedding a pre-stable `scheduling.k8s.io/v1alpha3` type inside the GA `batch/v1.JobSpec` was
permitted while the `Scheduling` field itself remained alpha-gated. Promoting the field to
default-enabled commits its wire format at the `v1` resource level, and API conventions draw no
distinction between Beta and GA for a struct embedded in a `v1` resource. Following the transition
pattern in [KEP-6089], v1.38 therefore:

- updates the `batch/v1.JobSpec` field to reference the `v1` types
- leaves Go type aliases (`type WorkloadPodGroupSchedulingPolicy = schedulingv1.WorkloadPodGroupSchedulingPolicy`,
  etc.) in `k8s.io/api/scheduling/v1alpha3` so third-party controllers that still import the alpha
  package continue to compile and serialize identically.

### Integration with the workloadbuilder Library

The Job controller compiles `spec.scheduling` into a `Workload` using the `workloadbuilder` library.

#### Library Dependency and Packaging

The building-block types live in the API staging repo (`k8s.io/api/scheduling/v1`). The `workloadbuilder`
library lives separately in `k8s.io/component-helpers/scheduling/schedulingv1/workloadbuilder`, so it can 
be vendored by both in-tree and out-of-tree controllers. The Job controller consumes these entrypoints:

- `NewBuilder(root *WorkloadItem, opts BuildOptions)` and `(*Builder).BuildWorkload()` to compile
  the `Workload` from `spec.scheduling`
- `NewBuilderFromExistingWorkload(workload, opts)` to recompile against a persisted `Workload`
  (elastic resize, delegated `PodGroup`, resume after suspend)
- `(*Builder).NewPodGroup(podGroupName, templateName)` to instantiate the runtime `PodGroup` from
  a template
- `(*Builder).Validate(ctx, ValidationInput)` from API validation

If the library API shifts, the Job integration tracks those changes through the shared dependency
rather than maintaining its own copy.

#### Building the Logical Tree and Compiling the `Workload`

The controller assembles the logical tree and compiles it in these steps: 
  1. Build a `workloadbuilder.WorkloadInput` from `spec.scheduling` via `jobutil.WorkloadInput`,
    mapping `schedulingPolicy`/`schedulingConstraints`/`disruptionMode`/`resourceClaims` onto the
    library's per-block inputs (with their field paths for error reporting).
  2. Assemble a single-node `WorkloadItem` via `jobutil.WorkloadItemForJob`, whose `DefaultConfig`
    resolves an absent policy to `Basic` and whose callback defaults an unset gang `minCount` to
    `spec.parallelism` (clamped to `>= 1`).
  3. Invoke `NewBuilder(item, BuildOptions{Owner: <controllerRef to Job>, AllowedPolicies: ...,
    AllowedDisruptionModes: ...}).BuildWorkload()`. The controller-owner `OwnerReference` makes
    the emitted `Workload` GC with the Job.

For a delegated non-root `PodGroup`, an update against a controller-owned `Workload`, or a
`PodGroup` recreated on resume, the controller uses
`NewBuilderFromExistingWorkload(workload, BuildOptions{...})` so it recompiles against the
already-persisted template instead of re-deriving it from scratch.

#### API Validation via the `workloadbuilder` Library

`spec.scheduling` is validated in three complementary layers, all self-contained:

1. **Declarative validation (DV) on the building blocks** owns the structural rules and most of the
   immutability. Because the `batch` API embeds the versioned `scheduling.k8s.io/v1` building
   blocks directly, their DV markers apply unchanged.
2. **Hand-written `batch` validation** covers the two cross-cutting rules DV cannot express:
   * `validateGangMinCount` rejects a gang `minCount` greater than `spec.parallelism`. A gang larger
     than the pod count can never be satisfied and the Job would stall with pending pods, so it is
     rejected at admission rather than surfacing only at runtime. It runs on create and update and
     for Jobs embedded in a `CronJob` `jobTemplate`. An elastic scale-up that raises
     `spec.parallelism` and `gang.minCount` in the same request is validated against the final state.
   * `validateJobSchedulingUpdate` freezes the basic/gang policy after creation. DV keeps
     the policy present but cannot forbid an in-place switch between `basic` and `gang`, so only
     `gang.minCount` may change.
3. **`workloadbuilder` semantic validation** owns the consistency rules that must stay identical to
   what the controller compiles. Validation builds the same `WorkloadItem` tree the controller does 
   and calls `NewBuilder(...).Validate()`. This runs the builder's allow-list checks. In-tree 
   it is constructed with `BuildOptions{DisableDeclarativeValidation: true}` because the api-server already ran DV on the
   versioned building blocks, so `Validate` here neither re-runs DV, compiles the `Workload`, nor
   needs an owner.

#### Instantiating the runtime `PodGroup`

`BuildWorkload` returns only the `Workload` (the scheduling template); it does not create the runtime
`PodGroup`. After the `Workload` exists on the API server, the Job controller instantiates the
`PodGroup` from the `Workload`'s single `PodGroupTemplate`. For a Job there is exactly one template,
so the controller creates one `PodGroup` that references the template and carries two
ownerReferences — a controller ref to the `Job` (so it is GC'd with the Job) and a non-controller
ref to the `Workload`. The same `NewPodGroup` path is used when the `PodGroup` is recreated on
resume, with the retained `Workload` as input.

Pods are then created by the existing Job pod-management logic with
`pod.Spec.SchedulingGroup.PodGroupName` set to the `PodGroup`'s name, which is what the scheduler
keys on for gang/topology behavior.

#### Reconcile Integration and Error Handling

The scheduling reconcile path is invoked from the Job reconcile loop, gated on the
  `WorkloadWithJob` feature gate, before pod management runs. Objects are created only while the
  Job is new or resuming from suspension, at any other time the path is discover-only. The 
  integration is designed to be idempotent and crash-safe:

- **Discovery first:** the controller looks up an existing `Workload`/`PodGroup` (via
  `spec.controllerRef` / `spec.podGroupTemplateRef`) before compiling. A restart between 
  creating the `Workload` and the `PodGroup` therefore does not produce duplicates.
- **Ordering:** `Workload` is created (or found) before the `PodGroup`, and both before pods, so
  references always resolve.
- **Errors are retryable:** a compilation error from `BuildWorkload` is terminal for that spec and
  is surfaced with a `FailedWorkloadCompilation` Warning event (the user must fix
  `spec.scheduling`); an API error creating the `Workload`/`PodGroup` emits `FailedWorkloadCreate`
  and requeues the Job with backoff, blocking pod creation until it succeeds.
- **Fallback with a Warning event:** if discovery finds more than one controller-owned `Workload`
  or `PodGroup`, or a controller-owned `Workload` whose shape no longer matches what the controller 
  compiles (`UnsupportedWorkloadStructure`), the controller falls back to default pod-by-pod scheduling 
  for that Job and pods are created without `schedulingGroup`. The condition is also counted in
  `job_scheduling_object_syncs_total{result="error"}`.
- **Updates:** on a `gang.minCount` (or `parallelism`-driven) change the controller recompiles the
  desired `Workload` (`NewBuilderFromExistingWorkload(...).BuildWorkload`) and patches the delta to
  the existing object, then patches the size onto the runtime `PodGroup`.
- **Suspend/resume:** once a suspended Job's pods are gone the controller deletes the runtime
  `PodGroup` (`PodGroupDeleted` event) and recreates it from the retained `Workload` on resume
  (`PodGroupCreated`).

### Job Controller Changes

The Job controller reconciliation loop that processes each Job will be extended to ensure `Workload`
and `PodGroup` objects exist before creating pods.

Because a standalone `Job` is a single-level workload, the Job controller is solely responsible for
both objects: it creates and owns the `Workload` and its corresponding runtime `PodGroup`, and 
garbage-collects them when the `Job` is deleted.

Before doing any work, the controller classifies each Job into one of three management modes,
which determines how much of the scheduling tree it owns. The supported scenarios are:

- **`manageBoth`**: the Job has no controller `ownerReference`. The controller compiles and owns
  both the `Workload` and the runtime `PodGroup` from `spec.scheduling`.
- **`managePodGroupOnly`**: the Job has a controller `ownerReference` to a parent and carries the
  `scheduling.k8s.io/group-template-name` annotation. The parent owns the `Workload`, so the
  controller discovers it, creates and owns only the `PodGroup` that is mapped to the named
  `PodGroupTemplate`.
- **`manageNone`**: the controller creates nothing and only stamps pods with whatever
  `schedulingGroup` the pod template already carries. This mode is selected when any of the
  following holds:
  - `spec.template.spec.schedulingGroup` is set (bring-your-own `PodGroup`): the user manages the
    objects.
  - The Job has a controller `ownerReference` to a parent but no `scheduling.k8s.io/group-template-name` 
  annotation: the parent (e.g. `JobSet`) owns both the `Workload` and the `PodGroup` and has set 
  `schedulingGroup` on the pod template itself.
  - The `WorkloadWithJob` feature gate is disabled.

#### Workload and PodGroup Discovery

Discovery uses indexers on `workload.spec.controllerRef` and `podGroup.spec.podGroupTemplateRef`
to find candidates cheaply, and then keeps only objects that carry a controller `ownerReference` 
to the Job. It is what makes the object garbage-collected with the Job, and it is what an external actor cannot plausibly set by accident. No separate `managed-by` annotation is needed.

A `Workload` is considered the Workload for this Job object if:
- it is in the Job's namespace
- its `spec.controllerRef` points to this Job, and
- it has a controller `ownerReference` to this Job

Similarly, a `PodGroup` is considered the `PodGroup` for this Job if:
- it is in the Job's namespace
- its `spec.podGroupTemplateRef.workload.workloadName` names the `Workload` for this Job (the
  Job-owned one for a root Job, the parent-owned one for a delegated Job), and
- it has a controller `ownerReference` to this Job

Objects that match the reference but not the ownership rule (a `Workload` pre-created by a user, a
`PodGroup` created by a parent) are ignored by discovery. This is what lets the parent-managed
scenario coexist with the delegated one without the Job controller ever adopting or mutating an
object it did not create.

#### Controller Workflow

The Job controller creates `Workload` and `PodGroup` objects only for a Job that has never
started, or that is resuming from suspension. A Job counts as never started when it has no pods
(active or terminal) owned by it, no `status.startTime`, zero `status.succeeded`/`status.failed`,
and no `JobSuspended` condition. Each signal alone has a gap, so the controller requires all of 
them.

The controller discovers or creates `Workload` and `PodGroup` as follows:

1. If the Job carries an `OwnerReference` to a parent controller that owns the `Workload` 
(i.e., `JobSet`), the Job controller does not create a `Workload` (skip step 3 and step 4). 
It then branches on whether the parent delegates `PodGroup` management, detected via the 
`scheduling.k8s.io/group-template-name` annotation on the Job:
   - **Annotation present (PodGroup delegated):** the parent owns the `Workload` but expects the Job
     to manage its own runtime `PodGroup`. Proceed to step 5, creating the `PodGroup` linked to the
     parent-owned `Workload` via the parent's named `PodGroupTemplate` (the
     `scheduling.k8s.io/group-template-name` value) and, when the parent also sets the
     `scheduling.k8s.io/parent-compositepodgroup` annotation, additionally link it to that parent 
     `CompositePodGroup` instance. The `PodGroup` gets a controller `ownerReference` to the Job.
   - **Annotation absent (both managed by the parent):** the parent owns both the `Workload` and
     the `PodGroup` and has already set `schedulingGroup` on the Job's pod template. The Job
     controller creates nothing (`manageNone`).
2. If the Job is neither new nor resuming, skip creation and only discover existing controller-owned 
  objects.
3. Look up controller-owned `Workload`(s) for this Job:
  - If none found and the Job is new, compile a `Workload` from the Job's `spec.scheduling` and
    create it with a controller `ownerReference` and `spec.controllerRef` pointing to this Job.
  - If exactly one, validate its shape: a Job-owned `Workload` must have exactly one
    `PodGroupTemplate`. If it does not, an external actor changed it; emit
    `UnsupportedWorkloadStructure` and fall back to default scheduling for this Job.
  - If more than one, emit `UnsupportedWorkloadStructure` and fall back to default scheduling.
4. When creating a new `Workload`, the controller derives the scheduling policy from the Job's
   `spec.scheduling` rather than from the Job's type. It maps `spec.scheduling` into the
   `workloadbuilder` library, which applies the defaulting rules (defaulting to `Basic`, defaulting
   `Gang.minCount` to `parallelism`) and compiles the `Workload`. This happens for every eligible Job, including those that default to `Basic`.
5. Look up controller-owned `PodGroup`(s) whose `spec.podGroupTemplateRef` targets the
   `PodGroupTemplate` for this Job. For a root Job that template lives in the Job-owned `Workload`;
   for a delegated non-root Job (step 1, annotation present) it is the parent's `PodGroupTemplate`:
  - If none found and the Job is new or resuming, create a `PodGroup` with a controller
    `ownerReference` to the `Job`, linked to the Job-owned `Workload` for a root Job or to the 
    parent-owned `Workload` for a delegated Job. When the `scheduling.k8s.io/parent-compositepodgroup` annotation is present, link it to that parent `CompositePodGroup` instance.
  - If exactly one, that is the `PodGroup` for this Job.
  - If more than one, emit `UnsupportedWorkloadStructure` and fall back to default scheduling.
6. Execute the existing pod-management logic to create pods, including `schedulingGroup.podGroupName`
   in the pod spec to associate pods with the `PodGroup`. In the fallback cases pods are created
   without `schedulingGroup`.

Note that the controller does not update the `Workload` or `PodGroup` objects at this point if they
already exist.

The controller requires informers, listers and indexers for `Workload` and `PodGroup` objects.
Both `Workload` and `PodGroup` are automatically garbage collected when they were created by the Job
controller and the corresponding Job is deleted.

#### OwnerReferences Relationship

The ownerReferences relationship between `Job`, `Workload`, `PodGroup`, and `Pod` is as follows:

```mermaid
flowchart BT
    Pod[Pod]
    PodGroup[PodGroup]
    Workload[Workload]
    Job[Job]

    Pod -->|ownerRef| PodGroup
    Pod -->|ownerRef| Job
    PodGroup -->|ownerRef| Job
    PodGroup -->|ownerRef <br/> (root Job only)| Workload
    Workload -->|ownerRef| Job

    PodGroup -.->|via <br/> podGroupTemplateRef| Workload

    linkStyle 5 stroke:#888,color:#888
```

- The `Workload` object has an ownerReference to the `Job` object with `controller: true` in case 
  it was created by the Job controller.
- The `PodGroup` object links to a `Workload` via `spec.podGroupTemplateRef`. When created 
by the Job controller it carries a controller ownerReference to the `Job`. A parent-owned 
`Workload` is never given an ownerReference from the `PodGroup`.
- The `Pod` object has an ownerReference to the `Job` object with `controller: true` and another 
  ownerReference to the `PodGroup` object

By this ownerReferences relationship, garbage collection will remove objects accordingly that avoids orphaned Pods with a stale PodGroup reference.

#### Defaulting Rules

These rules are applied by the controller when it compiles the
`Workload`/`PodGroup`; they are not api-server field defaulting. 
This is a controller-side resolution that is required to resolve the unset case anyway.

- **`Scheduling` unset → `Basic`.** Existing and non-WAS Jobs carry no `spec.scheduling`, the
  controller resolves the absent policy to `Basic`, preserving their behavior.
- **`Scheduling` set but `SchedulingPolicy` nil → `Basic`.** `WorkloadPodGroupSchedulingPolicy` is a 
  discriminated union for which the compiled `PodGroup` must carry exactly one concrete policy, so a
  nil `SchedulingPolicy` is resolved to `Basic`.
- **`Gang` with `MinCount` unset → `MinCount = parallelism`.** Done controller-side only 
without persisting the derived value back onto the Job spec because writing it back would make a 
user-set `minCount` indistinguishable from the default on later updates, where `minCount` is mutable.
The controller clamps this default to a minimum of 1 (a suspended Job may have `parallelism = 0`,
but a gang `minCount` must be positive).

Optional modifiers (`DisruptionMode`, `SchedulingConstraints`, `ResourceClaims`) are deliberately not
defaulted. Unlike `SchedulingPolicy`, these are optional fields whose absence is a defined state. A nil `DisruptionMode` resolves to standard per-pod (`Single`) disruption, a nil `SchedulingConstraints`
means no topology co-location, and a nil `ResourceClaims` means no shared claims.

#### Object Creation Order

The Job controller creates objects in the following order so that references point to existing objects and to satisfy any API validation that `Workload` exists before `PodGroup` is created. The order is as follows:
1. `Workload` object which will reference the `Job`.
2. `PodGroup` object which will reference the `Workload` and the `Job`.
3. `Pod` objects which will reference `PodGroup`.

The kube-scheduler waits for `PodGroup` when Pods have `schedulingGroup`, so scheduling does not depend on this order, the order is for consistency and API validity.

#### Handling Updates and Mutability

To support dynamic scaling of gang-scheduled workloads (Elastic Jobs), the Job API allows in-flight
updates to `spec.scheduling.schedulingPolicy.gang.minCount`; all other `spec.scheduling` fields are immutable
after Job creation, and updates that change them are rejected by API validation. This replaces the
v1.36 validation that rejected `spec.parallelism` updates for gang Jobs: because the gang size is
now driven by the mutable `minCount` ([KEP-4671]), `spec.parallelism` is no longer frozen for gang
Jobs, restoring support for [Elastic Indexed
Jobs](https://kubernetes.io/docs/concepts/workloads/controllers/job/#elastic-indexed-jobs). API
validation reuses the `workloadbuilder` library where possible so the accepted configurations stay
consistent with what the controller actually compiles. The update-validation rules change as follows:

  * `spec.parallelism` becomes *mutable* again: the v1.36 rule that rejected `spec.parallelism`
  updates for gang Jobs is removed, restoring Elastic Indexed Jobs.
  * `spec.scheduling.schedulingPolicy.gang.minCount` is *mutable* in-flight: on change the controller
  recompiles the `Workload` and re-syncs the `PodGroup` size.
  * All other `spec.scheduling` fields remain *immutable* after creation, enforced by api-server
  validation, since changing the policy, topology, disruption mode, or resourceClaims would require
  recreating the `Workload`/`PodGroup`.

When `minCount` is omitted it follows `spec.parallelism`, so a `parallelism` update is a valid way to
scale the gang without ever touching `spec.scheduling`.

#### Reconciliation Flow upon Updates

A user can change the target gang size in one of two ways:

- by setting `spec.scheduling.schedulingPolicy.gang.minCount` directly, when it is set explicitly
- by setting `spec.parallelism`, when `minCount` is unset.

In either case, the Job controller reconciles the change as follows:

1. **Detection:** the Job controller's reconcile loop detects the change and fetches the existing
   `Workload` resource from the API server.
2. **Workload Compilation:** it recompiles the desired `Workload` from the updated
   `spec.parallelism`/`minCount` via `NewBuilderFromExistingWorkload(...).BuildWorkload`, reusing the
   persisted template so only the gang size changes.
3. **Workload Update:** the controller applies the delta to the existing resource with a
   strategic-merge `Patch` rather than a full replace.
4. **PodGroup Sync:** the controller patches the updated size onto the runtime `PodGroup` so the
   scheduler targets the newly scaled size. In the delegated (`managePodGroupOnly`) mode the size
   instead follows the parent's `PodGroupTemplate`.

`minCount` is enforced only during scheduling: per [KEP-4671], updates do not affect
already-scheduled pods and apply only to pods evaluated in future scheduling cycles. The scheduler
also operates on an eventually consistent view, so an update may not take effect until the next
scheduling cycle.

#### Suspend and Resume

Suspending a Job deletes its pods and is expected to release the resources they held. In the
v1.37 alpha, the `Workload` and `PodGroup` were left in place during suspension. That is a
problem for the `PodGroup`, because the scheduler and the DRA layer attach group-level reservations 
and claims to it. So, a `PodGroup` that outlives its pods can keep resources pinned to a Job that 
is not running.

For Beta, the controller handles the two objects differently:

- The **`Workload`** is the compiled scheduling template. It holds no scheduler state, costs
  nothing to keep, and keeping it means the `PodGroup` can be recreated by
  `NewBuilderFromExistingWorkload` with the same name, template reference and `minCount` without
  recompiling from `spec.scheduling`. It is retained across suspension.
- The **runtime `PodGroup`** is the object the scheduler acts on. Therefore, it will be deleted 
  on suspend and recreated on resume, so the scheduler makes a fresh all-or-nothing placement 
  decision for the resumed Job instead of reasoning about a group whose members all disappeared.

The sequence on suspend is:

1. The existing suspend path deletes the Job's active pods and sets the `JobSuspended` condition.
2. Once the Job has no pods left (all pod finalizers removed), the controller deletes the
   controller-owned `PodGroup` and emits a `PodGroupDeleted` event. Deleting only after the pods
   are gone avoids racing the scheduler on a `PodGroup` that still has members, and respects any
   deletion protection adds to `PodGroup`. In `managePodGroupOnly` mode the delegated `PodGroup` 
   is handled the same way, while in `manageNone` mode nothing is deleted.
3. The `Workload` stays. `gang.minCount` changes made while suspended are still patched onto the
   `Workload` so the resumed `PodGroup` picks them up.

The sequence on resume is:

1. The `JobSuspended` condition transitions to `False`.
2. The controller recognizes the resume carve-out (Job has no pods, condition just cleared),
   discovers the retained `Workload`, and creates a new `PodGroup` from its template. The 
   new `PodGroup` has a new UID and, because of the hash suffix, may have a new name.
3. Pods are created and associated with the new `PodGroup`.

For a Job that is created suspended (`spec.suspend: true` at creation) the controller creates the
`Workload` immediately but defers the `PodGroup` to the first resume, so a suspended Job never owns
a runtime `PodGroup`.

### Interaction with a BYO PodGroup

The reconciliation above applies only to a `PodGroup` that the Job controller created and owns.
A user or a higher-level controller can instead manage the `PodGroup` themselves and wire the Job's
pods to it by setting `spec.template.spec.schedulingGroup.podGroupName` on the pod template. This
is the pattern `JobSet` uses for the Jobs it creates.

In this case the controller is in `manageNone` mode:

- It creates no `Workload` and no `PodGroup`, and does not add an `ownerReference` to, mutate, or
  delete the user's `PodGroup`.
- `spec.scheduling` is not translated into the user's `PodGroup` and `gang.minCount` is not synced
  into it. Setting both `spec.scheduling` and `schedulingGroup` is accepted by validation, but 
  `spec.scheduling` has no effect and the user's `PodGroup` is authoritative. This avoids a 
  split-brain where the controller would fight the object's owner.
- Pods are created with the user-provided `schedulingGroup` as-is.

A pre-created `Workload` (with `spec.controllerRef` pointing at the Job but no controller
`ownerReference`) is not a supported input. The discovery ignores it and the controller compiles its
own `Workload` from `spec.scheduling`.

### Naming Conventions

We will not use naming for discovery due to limitations related to naming. Naming is for human readability 
and logical linking between Job, `Workload`, and `PodGroup`. Because discovery does not depend on it, the 
naming pattern can be changed in later releases if needed. 

Following prior-art in [Deployment](https://github.com/kubernetes/kubernetes/blob/f42571572d241a2cdeffa3962c0ccf1f59180113/pkg/controller/deployment/sync.go#L560-L568), the naming convention can be as follows:

**1. Workload**
  - Pattern: `<(truncated-if-needed)job-name>-<hash>`
  - Truncation of the Job name is applied when necessary to respect object name length limits.
  - The hash is used for collision avoidance (implementation may use a generated suffix or a hash of relevant identity).
  - Object type (`Workload` vs `PodGroup`) is identified by other metadata (`ownerReferences[].kind`), not by the name pattern.

**2. PodGroup**
  - Pattern: `<(truncated-if-needed)workload-name>-<(truncated-if-needed)podGroup-template-name>-<hash>`
  - Truncation of workload name and podGroup name is applied when necessary to respect name length limits.
  - The hash allows multiple PodGroups within a `Workload` and `PodGroupTemplate` to have distinct names. The controller creates a single `PodGroup` per Job at a time, but the hash also gives a `PodGroup` recreated after suspension a distinct name from its predecessor.

### Deletion and Garbage Collection

Deletion of the scheduling objects happens on two paths:

- **Job deletion** relies on garbage collection. Controller-created `Workload` and `PodGroup`
  objects carry a controller `ownerReference` to the Job, so they are removed when the Job is
  deleted. The `PodGroup` additionally carries a non-controller `ownerReference` to the 
  `Workload` for a root Job. Pods carry a non-controller `ownerReference` to the `PodGroup` with 
  `blockOwnerDeletion` unset, so deleting a `PodGroup` never waits on pod termination.
- **Job suspension** is the one case where the controller deletes explicitly. The runtime
  `PodGroup` is deleted once the suspended Job has no pods. The `Workload` is never deleted 
  by the controller.

The controller does not add or adopt `ownerReferences` on objects it did not create. Only objects
with a controller `ownerReference` to the Job are considered controller-created. Since objects
can have at most one controller `ownerReference`, this identification is unambiguous.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

##### Unit tests

- `k8s.io/kubernetes/pkg/controller/job`: `2026-09-10` - `91.2%`
- `k8s.io/kubernetes/pkg/apis/batch/validation`: `2026-09-10` - `88.8%`
- `k8s.io/kubernetes/pkg/registry/batch/job`: `2026-09-10` - `95.4%`
- Add tests that verify:
  - An omitted `spec.scheduling` is resolved by the controller to the `Basic` 
  policy, and a `Gang` policy with a nil `MinCount` is resolved to `MinCount = parallelism` 
  without the api-server writing these values back into the Job's `spec.scheduling`.
  - `workloadbuilder` compilation: `Basic` vs `Gang` policy, and that topology constraints,
    disruption mode (single/all), and resourceClaims are mapped into the generated `Workload`/
    `PodGroup`; that a `Job` builds a flat single-node tree via the shared
    `WorkloadInput`/`WorkloadItemForJob` helpers.
  - A `Basic` `Workload`/`PodGroup` is created for a Job with `spec.scheduling` omitted.
  - pod creation includes the correct `schedulingGroup`.
  - Mutability/validation: updates to `spec.scheduling.schedulingPolicy.gang.minCount` are allowed; updates to
    any other `spec.scheduling` field are rejected.
  - `gang.minCount > spec.parallelism` is rejected on both create and update. A single 
  request that raises `spec.parallelism` and `gang.minCount` together is accepted.
  - Feature gate disabled: `spec.scheduling` is dropped on create and no `Workload`/`PodGroup` is
    created.
  - Parent-owned `Workload`, both delegated: a Job with an `OwnerReference` to a parent workload and
    no annotation creates neither `Workload` nor `PodGroup`.
  - Parent-owned `Workload`, `PodGroup` delegated: a Job with an `OwnerReference` to a parent
    workload and the annotation present does not create a `Workload`, but does create a `PodGroup` 
    linked to the parent-owned `Workload`.
  - Job deletion cascades to `Workload` and `PodGroup` deletion.
  - ownerReferences on controller-created objects match the expected structure:
    - Root Job: `Workload` has a controller ownerRef to the Job; `PodGroup` has a controller ownerRef
      to the Job and a non-controller ownerRef to the `Workload`.
    - Delegated Job: `PodGroup` has a controller ownerRef to the Job and links to the parent-owned 
    `Workload`/`CompositePodGroup` (no Job-owned `Workload` exists).
  - Naming abbreviations for `Workload` and `PodGroup`.
  - Discovery only matches objects with a controller `ownerReference` to the Job: a `Workload` with
    `spec.controllerRef` to the Job but no controller `ownerReference` is ignored and the
    controller compiles its own.
  - Ambiguity: two controller-owned `Workload`s (or `PodGroup`s) for one Job produce an
    `UnsupportedWorkloadStructure` Warning event, pods are created without `schedulingGroup`, and
    `job_scheduling_object_syncs_total{result="error"}` is incremented.
  - Drift: a controller-owned `Workload` whose `PodGroupTemplates` count is not 1 produces an
    `UnsupportedWorkloadStructure` Warning event and the same fallback.
  - Suspend/resume: the `PodGroup` is deleted only after the suspended Job has no pods
    (`PodGroupDeleted`), the `Workload` is retained, a `gang.minCount` change while suspended is
    applied to the `Workload`, and a new `PodGroup` is created on resume with the current
    `minCount`. A Job created with `spec.suspend: true` gets a `Workload` but no `PodGroup` until
    first resume.

##### Integration tests

Existing tests in `test/integration/job/job_test.go` (v1.37):
- `TestJobGangScheduling`: lifecycle for `Basic` and `Gang` Jobs (create, discover, delete; `Workload`
  and `PodGroup` are materialized, pods carry `schedulingGroup`, deletion cascades), passthrough of
  topology constraints, disruption mode and resourceClaims, and the multiple-object fallback.
- `TestJobGangSchedulingElasticScaling`: updating `spec.scheduling.schedulingPolicy.gang.minCount`
  (or `spec.parallelism` when `minCount` is unset) updates the `Workload` and the runtime `PodGroup`.
- `TestJobGangSchedulingSuspendResume`: suspend/resume behavior (updated for Beta, see above).
- `TestJobDelegatedPodGroup`: a Job owned by a parent workload skips `Workload` creation, skips
  `PodGroup` creation when no annotation is set, and creates a `PodGroup` mapped to the parent's
  `PodGroupTemplate` when the annotation is present.
- Feature gate disabled: `spec.scheduling` is dropped and no `Workload`/`PodGroup` is created.

Tests added for Beta:
- Suspend deletes the `PodGroup` after pods are gone and keeps the `Workload`; resume creates a new
  `PodGroup` referencing the same `Workload`; pods created after resume reference the new
  `PodGroup` name.
- Bring-your-own `PodGroup` via `spec.template.spec.schedulingGroup`: the controller creates no
  `Workload`/`PodGroup`, does not add an `ownerReference` to the user's `PodGroup`, and leaves its
  `minCount` untouched on a `gang.minCount` update.
- Ambiguous and drifted controller-owned objects: Warning events are emitted and pods are created
  without `schedulingGroup`.

##### e2e tests

Existing tests (`test/e2e/apps/job.go`):
- `should compile a Workload and PodGroup for a gang-scheduled Job and wire its pods to the PodGroup`
- `should propagate an elastic gang minCount change to the Workload and PodGroup`
- `should create Workload and PodGroup for gang-eligible Job` (also gated on `GenericWorkload`;
  verifies `PodGroup` owner references and pod `schedulingGroup` wiring)

Tests added for Beta:
- Gang scheduling behavior: with insufficient capacity for `minCount` pods, no pod of the gang is
  bound; once capacity is available all are bound.
- `Basic` scheduling policy: pods of a Job without `spec.scheduling` are scheduled pod-by-pod and
  the Job completes, with a `Basic` `Workload`/`PodGroup` present.
- Suspend/resume: suspending a gang Job removes its `PodGroup`; resuming creates a new one and the
  Job completes.
- CronJob with gang scheduling: each Job created by the CronJob gets its own `Workload`/`PodGroup`,
  and completed Jobs clean up their scheduling objects via GC.

Performance: a scale test in `kubernetes/perf-tests` compares Job creation throughput and
`job_sync_duration_seconds` with the gate on and off, and records time-to-first-bind for gang Jobs.

### Graduation Criteria

#### Alpha (v1.36)

The first alpha (the automatic, type-based model) delivered:
- [x] Feature implemented behind the `WorkloadWithJob` feature gate (default: disabled).
- [x] Job controller creates `Workload`/`PodGroup` objects when the feature gate is enabled.
- [x] Gang scheduling policy applied to indexed parallel Jobs (`parallelism > 1`, `completions = parallelism`, `completionMode: Indexed`).
- [x] Non-gang scheduling Jobs do not have `Workload`/`PodGroup` objects created.
- [x] Jobs managed by higher-level controllers skip `Workload`/`PodGroup` creation.
- [x] API validation rejects updates that change `spec.parallelism` for gang scheduling Jobs.
- [x] Unit and integration tests for the `Workload`/`PodGroup` creation flow.

#### Alpha (v1.37)

The second alpha replaced the automatic model with the user-facing API:
- [x] New `spec.scheduling` (`JobSchedulingConfiguration`) field added to `batch/v1`, gated by the
  existing `WorkloadWithJob` feature gate (still default-disabled).
- [x] The Job controller compiles `spec.scheduling` into `Workload`/`PodGroup` via the shared
  `workloadbuilder` library, defaulting to `Basic` and materializing a `Workload`/`PodGroup` for
  every eligible Job.
- [x] `Gang` opt-in with `minCount` defaulting to `parallelism`, plus support for mutable `minCount`
  (elastic scaling) and passthrough of topology constraints, disruption mode, and resourceClaims.
- [x] API validation makes `spec.scheduling` fields immutable except `gang.minCount`; the v1.36
  `spec.parallelism`-rejection validation is removed.
- [x] Jobs owned by a higher-level controller (via `OwnerReference`) defer `Workload` ownership to the
  parent; they manage their own `PodGroup` when the parent delegates it via the annotation, and skip both objects otherwise.
- [x] Unit and integration tests for the new API, defaulting, mutability, and `workloadbuilder`
  compilation; user-facing documentation for the new API.

#### Beta

- [ ] `WorkloadWithJob` enabled by default, together with `GenericWorkload` ([KEP-4671]). If
  `GenericWorkload` does not ship default-on in v1.38, `WorkloadWithJob` graduates to Beta as
  default-disabled.
- [ ] `batch/v1.JobSpec.Scheduling` repointed to the `scheduling.k8s.io/v1` building blocks
  ([KEP-6089]).
- [ ] `PodGroup` deleted on suspend and recreated on resume.
- [ ] `UnsupportedWorkloadStructure` Warning event and fallback to default scheduling.
- [ ] Metrics `job_scheduling_object_syncs_total` and `job_scheduling_object_sync_duration_seconds`.
- [ ] E2E tests passed as designed in the [Test Plan](#test-plan) and linked in Testgrid.
- [ ] Performance test in `kubernetes/perf-tests` showing no regression in Job creation throughput.
- [ ] Address all issues reported by users during alpha.
- [ ] User-facing documentation updated.

#### GA

- [ ] `GenericWorkload` ([KEP-4671]) is GA.
- [ ] Two releases of Beta with no changes to the `spec.scheduling` shape or defaulting.
- [ ] Address all issues reported by users during beta, including feedback on the fallback behavior
  for ambiguous/drifted objects.
- [ ] e2e tests for `spec.scheduling` promoted to conformance.
- [ ] `WorkloadWithJob` locked to enabled; the gate is removed after the deprecation window.

#### Deprecation

N/A. Promotion to Beta deprecates nothing; the `v1alpha3` building-block package remains
importable through type aliases.

### Upgrade / Downgrade Strategy

This KEP is additive and falls back to the original behavior on downgrade.

#### Upgrade

kube-apiserver is upgraded first, then kube-controller-manager. From that point every eligible Job
that has not started gets a `Workload`/`PodGroup` compiled from its `spec.scheduling` (defaulting
to `Basic`). Jobs that already started before the upgrade are left alone. Operators who do not want
the scheduling objects can set `WorkloadWithJob=false` on kube-apiserver and kube-controller-manager;
disabling `GenericWorkload` requires disabling `WorkloadWithJob` as well.

Clusters upgrading from the v1.36 alpha should note that indexed fully-parallel Jobs are no longer
gang-scheduled automatically; since v1.37 gang scheduling requires `spec.scheduling.schedulingPolicy.gang`.

#### Downgrade

Downgrading kube-controller-manager (or disabling the gate) stops compiling `Workload`/`PodGroup`,
stops setting `schedulingGroup` on new pods, and stops deleting `PodGroup`s on suspend. Downgrading
kube-apiserver clears `spec.scheduling` on create and ignores it on update; values already stored on
a Job are preserved and served again once the gate is re-enabled. Existing `Workload`/`PodGroup`
objects remain in etcd, are still honored by the scheduler for pods that reference them while
`GenericWorkload` is enabled there, and are garbage collected with their Job.

### Version Skew Strategy

This feature is limited to the control plane, so version skew with kubelets does not matter.

kube-apiserver can be one minor version ahead of kube-controller-manager and kube-scheduler. An
older kube-controller-manager ignores `spec.scheduling` and compiles no `Workload`/`PodGroup`, and
an older kube-scheduler ignores `schedulingGroup` on pods. In both cases Jobs schedule pod-by-pod
as if the feature were absent until the component is upgraded. Users should not rely on gang
scheduling until all control-plane instances are upgraded.

In HA control planes only one kube-controller-manager and one kube-scheduler are leader at a time,
so replica skew does not matter.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `WorkloadWithJob`
  - Components depending on the feature gate:
    - kube-controller-manager
    - kube-apiserver
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

Yes. With the gate enabled, the Job controller creates and manages a `Workload` and a `PodGroup`
for every eligible Job that has not started, including Jobs without `spec.scheduling`, and sets 
`spec.schedulingGroup.podGroupName` on the pods it creates.

Note this differs from the v1.36 alpha, where indexed fully-parallel Jobs were gang-scheduled
automatically. Since v1.37 gang scheduling is an explicit opt-in via `spec.scheduling`.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. With the gate disabled on kube-apiserver, it clears `spec.scheduling` on create and ignores it
on update. With the gate disabled on kube-controller-manager, the Job controller stops compiling 
`Workload`/`PodGroup`, stops setting `schedulingGroup` on new pods, and stops deleting `PodGroup`s 
on suspend.

###### What happens if we reenable the feature if it was previously rolled back?

Jobs that have not started get `Workload`/`PodGroup` compiled from their stored `spec.scheduling`
(defaulting to `Basic`) on their next sync. Jobs that already have these objects from before the
rollback reuse them; a partial set (e.g. `Workload` without `PodGroup`) is completed if the Job has
no pods yet. Jobs with running pods are not affected.

###### Are there any tests for feature enablement/disablement?

Yes.
- [strategy_test.go](https://github.com/kubernetes/kubernetes/blob/master/pkg/registry/batch/job/strategy_test.go)
- [job_scheduling_manager_test.go](https://github.com/kubernetes/kubernetes/blob/master/pkg/controller/job/job_scheduling_manager_test.go)
- [job_test.go](https://github.com/kubernetes/kubernetes/blob/master/test/integration/job/job_test.go)

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Already running pods are not affected. The Job controller does not touch existing pods and the
scheduler does not revisit bound pods.

If `WorkloadWithJob` is enabled on kube-controller-manager while `GenericWorkload` is still
disabled on kube-apiserver (e.g. kube-controller-manager upgraded first), `Workload`/`PodGroup`
creates fail and new Jobs are requeued with backoff without creating pods until the API is served.
Upgrading kube-apiserver first avoids this.

On rollback, new pods of in-flight gang Jobs are created without `schedulingGroup`, so a partially
created gang finishes scheduling pod-by-pod. Existing `PodGroup`s are garbage collected with their
Job.

###### What specific metrics should inform a rollback?

- `job_scheduling_object_syncs_total{result="error"}` increasing: the controller cannot write
  `Workload`/`PodGroup` objects and Jobs are stuck before pod creation.
- `job_syncs_total{result="error"}` or p99 `job_sync_duration_seconds` increasing after enablement
  compared to the pre-upgrade baseline.
- `scheduler_pending_pods{queue="gated"}` growing for Jobs that used to schedule ([KEP-4671]).

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The gate enable -> disable -> re-enable sequence is covered by integration tests. The
upgrade -> downgrade -> upgrade path will be tested manually before the release using the following
sequence:

1. Start a v1.37 cluster with `WorkloadWithJob` and `GenericWorkload` disabled (default). Create a
   Job with `spec.scheduling.schedulingPolicy.gang`; verify the field is dropped and the Job runs
   pod-by-pod with no `Workload`/`PodGroup`.
2. Upgrade to v1.38 (gates enabled by default). Create a gang Job; verify a `Workload` and
   `PodGroup` are created, pods carry `schedulingGroup`, and pods are bound together. Suspend the
   Job; verify the `PodGroup` is deleted and the `Workload` retained. Resume; verify a new
   `PodGroup` is created and the Job completes.
3. Downgrade to v1.37. Verify existing `Workload`/`PodGroup` objects remain, a new Job gets no
   scheduling objects, and its pods schedule pod-by-pod.
4. Upgrade to v1.38 again. Verify the Jobs created in step 2 reuse their existing objects and a new
   gang Job is gang-scheduled.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check the `job_scheduling_object_syncs_total{kind="Workload",action="create",result="success"}`
metric. If it is non-zero, the Job controller has created scheduling objects and the feature is
in use. For a specific Job, `kubectl get workloads,podgroups -n <ns>` lists the objects owned by it.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `WorkloadCreated`, `PodGroupCreated`, `PodGroupDeleted` (Normal);
    `FailedWorkloadCompilation`, `FailedWorkloadCreate`, `UnsupportedWorkloadStructure` (Warning)
- [x] API .spec
  - Other field:
    - `.spec.schedulingGroup.podGroupName` on the Job's pods
    - a `Workload` and a `PodGroup` with a controller `ownerReference` to the Job

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

99th percentile over a day of `job_sync_duration_seconds` with the feature enabled stays within 10%
of the value with the feature disabled for the same workload. 99% of `Workload`/`PodGroup` writes
made by the Job controller per day succeed.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name:
    - `job_scheduling_object_syncs_total` (new), with labels `kind`, `action` and `result`.
      Incremented for each `Workload`/`PodGroup` create/update/delete made by the Job controller.
    - `job_scheduling_object_sync_duration_seconds` (new), with labels `kind` and `action`.
      Latency of those calls.
    - `job_sync_duration_seconds` and `job_syncs_total` (existing), compared against the
      pre-enablement baseline.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A gauge of Jobs per scheduling policy (`basic`/`gang`) would show adoption directly. However, it is not
added because the same information is available by listing `PodGroup` objects.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No dependencies other than the components where the feature is implemented (kube-apiserver and
kube-controller-manager) and the `GenericWorkload` feature gate being enabled on kube-apiserver 
and kube-scheduler.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes. The Job controller adds two informers (`Workload`, `PodGroup`), i.e. two LIST+WATCH streams
against kube-apiserver. Per Job, with the gate on (the default from v1.38), for every root Job
including `Basic` Jobs:
- `CREATE Workload`: 1 per Job creation.
- `CREATE PodGroup`: 1 per Job creation, plus 1 per resume from suspension.
- `DELETE PodGroup`: 1 per suspension (after the pods are gone).
- `PATCH Workload` and `PATCH PodGroup`: 1 each per `gang.minCount` (or `parallelism`-driven)
  elastic resize.

A non-root Job whose parent owns the `Workload` but delegates the `PodGroup` makes only the
`PodGroup` calls. A non-root Job where the parent owns both objects, or a Job with a BYO
`PodGroup`, makes none of these calls.

###### Will enabling / using this feature result in introducing new API types?

No new top-level types. `spec.scheduling` (`JobSchedulingConfiguration`) is a new field on the
existing `batch/v1` Job that embeds the `scheduling.k8s.io/v1` building blocks from [KEP-6089].
The `Workload`/`PodGroup` resources are defined by [KEP-4671].

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes. Because of Universal Representation, every root Job (both `Gang` and `Basic`) creates 1 
`Workload` (~500 bytes) and 1 `PodGroup` (~500 bytes), and each Pod gains a `schedulingGroup` 
field (~100 bytes) plus one `ownerReference` (~150 bytes). A delegated non-root Job adds only a
`PodGroup`, and a fully-delegated Job (parent owns both objects) adds neither. `Job` objects
themselves grow only when the user sets `spec.scheduling` (a few hundred bytes at most).

For a cluster with 10,000 live root Jobs, this adds approximately:
- 10,000 `Workload` objects
- 10,000 `PodGroup` objects
- ~10MB additional etcd storage for the objects, plus ~250 bytes per Job pod

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Job sync latency (`job_sync_duration_seconds`) for the first sync of a new Job increases by two
sequential API writes before pod creation. The Beta performance test compares the gate on and off
with a target of p99 within 10% of the baseline.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Yes, proportional to the number of live Jobs.
- kube-controller-manager: `Workload`/`PodGroup` informer caches (~1-2KB per Job, about 20MB for
  10,000 Jobs).
- kube-scheduler: memory for the same `Workload`/`PodGroup` caches, accounted for in [KEP-4671].
- etcd: storage for `Workload`/`PodGroup` objects (~1KB per Job, about 10MB for 10,000 Jobs).
- kube-apiserver: two additional watch streams and two extra writes per Job creation. Minimal CPU
  impact.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature is purely control-plane and does not affect node resources.

### Troubleshooting


###### How does this feature react if the API server and/or etcd is unavailable?

The Job controller cannot create or delete `Workload`/`PodGroup` objects and requeues the affected
Jobs with backoff, as it does for any other write. New Jobs make no progress until the API server
recovers, the same as for pod creation. Running pods are unaffected.

###### What are other known failure modes?

- `Workload`/`PodGroup` create rejected by the API server (RBAC, admission webhook,
  `GenericWorkload` disabled).
  - Detection: `FailedWorkloadCreate` Warning event on the Job;
    `job_scheduling_object_syncs_total{action="create",result="error"}` increasing.
  - Mitigations: fix the rejecting component; the controller retries with backoff. If the resources
    are intentionally unavailable, disable `WorkloadWithJob` on kube-controller-manager.
  - Diagnostics: kube-controller-manager logs include the API error.
  - Testing: unit tests with a fake-client reactor.
- Duplicate or externally mutated controller-owned objects.
  - Detection: `UnsupportedWorkloadStructure` Warning event on the Job and its pods have no
    `schedulingGroup`.
  - Mitigations: delete the extra or modified object. The controller does not recreate a
    `Workload` for a Job that has already started, so recreate the Job if gang semantics are
    required.
  - Diagnostics: audit log for writers on `workloads`/`podgroups`.
  - Testing: unit and integration tests for the fallback path.

###### What steps should be taken if SLOs are not being met to determine the problem?

- Confirm `WorkloadWithJob` and `GenericWorkload` are enabled consistently on kube-apiserver,
  kube-controller-manager and kube-scheduler.
- Check `job_scheduling_object_syncs_total{result="error"}`, and the Warning events on the 
  affected Jobs.
- Compare `job_scheduling_object_sync_duration_seconds` with `apiserver_request_duration_seconds`
  for `workloads`/`podgroups` to tell controller-side from API-server-side latency.
- For gang Jobs whose pods stay `Pending` with objects present, follow the scheduler
  troubleshooting steps in [KEP-4671].

## Implementation History
- 2026-01-29: KEP created
- 2026-02-10: KEP updated according to final API design for `Workload` and `PodGroup`
- 2026-06-03: KEP reworked for a second alpha (v1.37) to replace the automatic, type-based gang
  selection with the explicit user-facing `spec.scheduling` API from [KEP-6089], adopting the
  shared `workloadbuilder` library, Universal Representation, and mutable `gang.minCount` for
  elastic scaling.
- 2026-09-10: KEP updated for Beta (v1.38).

## Drawbacks

## Alternatives

### Bring-your-own Workload

The v1.37 alpha design allowed a user to pre-create a `Workload` whose `spec.controllerRef` points
at a Job and have the Job controller instantiate the runtime `PodGroup` from it. This was dropped
during implementation review (kubernetes/kubernetes#140188) and the KEP now matches the code:

- With `spec.scheduling` available, every shape a Job-owned `Workload` can have (one
  `PodGroupTemplate` with a policy, constraints, disruption mode and claims) is expressible on the
  Job itself. A pre-created `Workload` adds no expressiveness.
- It creates a second source of truth the controller must reconcile against `spec.scheduling`,
  which is exactly the split-brain the design otherwise avoids.
- Telling a user-created `Workload` apart from a controller-created one required an extra
  `managed-by` marker and made discovery ownership-agnostic, complicating both the controller and
  the garbage-collection story.

Users who need full control over the scheduling objects keep the bring-your-own `PodGroup` path,
which also serves parent controllers such as `JobSet`.

## Infrastructure Needed (Optional)

[^1]: The Kubernetes community uses the term "gang scheduling" to mean "all-or-nothing 
scheduling of a set of pods" [1,2,3,4,5,6,7,8,9,10,11,12,13]. In the Kubernetes context, 
it does not imply time-multiplexing (in contrast to prior academic work such as 
[Feitelson and Rudolph](https://doi.org/10.1016/0743-7315(92)90014-E), and in contrast 
to [Slurm Gang Scheduling](https://slurm.schedmd.com/gang_scheduling.html).

[KEP-4671]: https://kep.k8s.io/4671
[KEP-5710]: https://kep.k8s.io/5710
[KEP-5732]: https://kep.k8s.io/5732
[KEP-6089]: https://kep.k8s.io/6089
