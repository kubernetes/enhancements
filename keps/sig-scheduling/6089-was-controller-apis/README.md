# KEP-6089: Workload Aware Scheduling Controller APIs
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Reusable API Building Blocks](#reusable-api-building-blocks)
  - [Shared workloadbuilder Library](#shared-workloadbuilder-library)
  - [Integration Recommendations &amp; Controller Autonomy](#integration-recommendations--controller-autonomy)
  - [Job Integration - API Usage Examples](#job-integration---api-usage-examples)
    - [Example 1: Job with Gang Scheduling, Zone Topology, and Atomic Disruption](#example-1-job-with-gang-scheduling-zone-topology-and-atomic-disruption)
    - [Example 2: Backward Compatibility and Sane Defaulting (Implicit Opt-Out)](#example-2-backward-compatibility-and-sane-defaulting-implicit-opt-out)
  - [User Stories](#user-stories)
    - [Story 1: The End-User](#story-1-the-end-user)
    - [Story 2: The Controller Maintainer](#story-2-the-controller-maintainer)
    - [Story 3: The Multi-Level Controller Maintainer](#story-3-the-multi-level-controller-maintainer)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Core Principles &amp; Assumptions](#core-principles--assumptions)
  - [Standardized Building Blocks Definitions (<code>scheduling.k8s.io</code>)](#standardized-building-blocks-definitions-schedulingk8sio)
  - [Job Integration (batch/v1)](#job-integration-batchv1)
    - [API Changes](#api-changes)
  - [Shared workloadbuilder Go Translation Library](#shared-workloadbuilder-go-translation-library)
    - [1. Design &amp; Architecture](#1-design--architecture)
    - [2. Controller Opt-In for New Scheduling Capabilities](#2-controller-opt-in-for-new-scheduling-capabilities)
    - [3. Library API Definition](#3-library-api-definition)
    - [4. Library Usage Example (Job)](#4-library-usage-example-job)
  - [Reference Integration Examples: JobSet (Multi-Level)](#reference-integration-examples-jobset-multi-level)
    - [1. Option A: Template Delegation Model (Nested Configuration)](#1-option-a-template-delegation-model-nested-configuration)
      - [Example YAML Manifest](#example-yaml-manifest)
    - [2. Option B: Centralized 'Targeted Policies' Model (Root-only Configuration)](#2-option-b-centralized-targeted-policies-model-root-only-configuration)
      - [Example YAML Manifest](#example-yaml-manifest-1)
    - [3. Controller Integration and workloadbuilder Mapping Go Code](#3-controller-integration-and-workloadbuilder-mapping-go-code)
  - [Recommendations for Multi-Level Composite Controllers](#recommendations-for-multi-level-composite-controllers)
    - [1. Runtime PodGroup and CompositePodGroup Lifecycle Management](#1-runtime-podgroup-and-compositepodgroup-lifecycle-management)
    - [2. Downward Template and Parent Mapping via Well-Known Annotations](#2-downward-template-and-parent-mapping-via-well-known-annotations)
      - [The Solution: Downward Mapping Annotations](#the-solution-downward-mapping-annotations)
  - [Go Package Placement &amp; Graduation Strategy](#go-package-placement--graduation-strategy)
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
  - [1. Implementation Complexity &amp; The &quot;Transitive Capability Leak&quot;](#1-implementation-complexity--the-transitive-capability-leak)
  - [2. The Upstream Dependency Bottleneck](#2-the-upstream-dependency-bottleneck)
  - [The Chosen Solution: Autonomous Composed Configurations &amp; Conscious Trade-offs](#the-chosen-solution-autonomous-composed-configurations--conscious-trade-offs)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP proposes a standardized set of reusable API building blocks (`scheduling.k8s.io`),
integration guidelines, and shared libraries to simplify how workload controllers (e.g., `JobSet`,
`TrainJob`, `RayJob`, `LWS`, as well as core workloads like `Job`) integrate with Workload-aware
Scheduling (WAS).

By providing common API primitives (such as topology constraints and disruption policies) and a
shared library to handle boilerplate resource generation, we enable controller developers to
easily expose WAS features natively within their APIs without reinventing the wheel, while
ensuring a consistent user experience across the Kubernetes ecosystem.

## Motivation

The Kubernetes ecosystem has steadily evolved its scheduling capabilities from a strictly
pod-centric model towards a more robust, workload-centric approach. This transition successfully
established foundational features in the recent v1.36 release, such as Gang Scheduling,
Topology-aware Scheduling (TAS), and Workload-aware Preemption (WAP).

However, the `Workload` and `PodGroup` resources backing these features were designed primarily as
intermediate, scheduler-facing APIs. We have not yet addressed how end-users of higher-level
workload controllers (such as `Job`, `LWS`, `JobSet`, or `RayJob`) should express their scheduling
requirements to utilize these features.

For example, in the first alpha release of [KEP-5547] (Job Integration), we intentionally bypassed the user-facing
API design challenge. Instead, the integration automatically creates a `PodGroup` with a hardcoded
Gang policy under specific conditions (e.g., for fully parallel static indexed Jobs). While this
unblocked initial adoption, it is fundamentally insufficient. Users have diverse use cases and
require the ability to express explicit intent—such as opting in or out of gang scheduling,
requesting specific topologies, or configuring disruption policies for their workloads.

Currently, there is no standardized way for workload controllers to expose these user intents, nor
is there a standard mechanism for controllers to translate user intent into underlying scheduling
objects. If every controller authors its own user-facing API structs and custom logic to manage
scheduling objects, the ecosystem will suffer from inconsistent UX, duplicate effort, and varied
levels of WAS support.

We need a standardized toolkit that provides common scheduling API structures, handles the
boilerplate compilation, and establishes architectural guidelines to solve common integration
challenges across the ecosystem. This proposal aims to fill these gaps, providing shared tooling
and best practices while still allowing controller owners the flexibility to design their root
APIs natively.


### Goals

- Define reusable API primitives (e.g., Scheduling Policies, Topology Constraints, Disruption
  Modes) under `scheduling.k8s.io` to be consumed by real-workload controllers.

- Provide a shared library (workloadbuilder) to handle the boilerplate of constructing underlying
  scheduling objects (`Workload`, `PodGroup`, or `CompositePodGroup`) from controller-specific intents.

- Establish architectural guidelines for workload controllers to expose WAS features consistently.

- Integrate these building blocks and the translation library with the core `Job` API (`batch/v1`)
  to ensure we are not designing in a vacuum. Standard `Job` is the natural candidate to "blaze
  the path" for other workload controllers; it initially integrated with WAS in v1.36 in alpha,
  but intentionally bypassed the user-facing scheduling API aspect. Under this KEP, the core `Job`
  integration remains in **Alpha** in v1.37, but is enriched to give users the ability to express
  explicit scheduling intent, resolving usability gaps from the initial v1.36 alpha.

- Provide reference integration examples demonstrating how complex, multi-level composite
  controllers (such as `JobSet`) can adopt WAS Controller APIs. Since standard `Job` serves as the
  production single-level implementation, we focus our reference designs purely on demonstrating
  multi-level hierarchical patterns.


### Non-Goals

- Define a single, mandatory and rigid scheduling API struct for all Kubernetes workload
  controllers.

- Implement the actual integration of these new API blocks into other complex composite
  controllers (such as `JobSet`, `LeaderWorkerSet`, or Kubeflow `TrainJob`) as part of this KEP.
  While this KEP establishes the design guidelines and shared library for their integration, the
  implementation PRs for these out-of-tree controllers will be pursued independently in their
  respective repositories.

- Create or manage the lifecycle of `Workload`, `PodGroup`, or `CompositePodGroup`.

## Proposal

This proposal builds on the enhancements that have been recently introduced in the workload-aware
scheduling space. We assume that the reader is already acquainted with the following KEPs:

- [KEP-4671: Gang Scheduling using Workload Object](https://kep.k8s.io/4671)
- [KEP-5710: Workload-aware preemption](https://kep.k8s.io/5710)
- [KEP-5732: Topology-aware workload scheduling](https://kep.k8s.io/5732)
- [KEP-6012: CompositePodGroup API](https://kep.k8s.io/6012)
- [KEP-5547: Integrate Workload APIs with Job Controller](https://kep.k8s.io/5547)

### Reusable API Building Blocks

We propose introducing a set of standard, reusable structs in the `scheduling.k8s.io` API group.
Controller developers can embed these structs directly into their native APIs. This ensures that
when a user configures a `TopologyConstraint` on a `RayJob`, it uses the exact same schema and
semantics as a `TopologyConstraint` on a `TrainJob`.

### Shared workloadbuilder Library

To prevent every controller from writing custom logic to translate these API blocks into
underlying scheduling resources, we will provide a shared Go library. Controller developers will
map their custom API surface to an intermediate representation, and the library will handle:

- Generating the correct `Workload`, `PodGroup`, or `CompositePodGroup` hierarchies.
- Applying sane scheduling defaults based on the controller's semantic purpose (e.g., defaulting
  to standard pod-by-pod scheduling for a core `Job` to explicitly prevent breaking existing CI/CD
  pipelines).
- Handling standard validation logic.


### Integration Recommendations & Controller Autonomy

Instead of forcing a one-size-fits-all API shape, we provide recommendations on how these building
blocks can be exposed, leaving the final design decisions to the controller owners. This approach
prioritizes local consistency over global uniformity. While this may introduce a degree of API
fragmentation across the ecosystem, it is a necessary and acceptable trade-off to ensure each
controller's API remains idiomatic and intuitive for its specific users.

This autonomy is particularly crucial for complex, multi-level controllers that rely on resource
composition. If we mandated a strict, unified API shape that relied on downward API propagation,
we would introduce severe upstream dependency bottlenecks. For example, `TrainJob` relies on `JobSet`,
which in turn relies on the core `Job` API. Requiring bottom-up integration would block `TrainJob`
users for months while waiting for the underlying components to adopt the standard. By granting
controllers autonomy, they can implement workarounds native to their architecture—such as `JobSet`
using its established targetReplicatedJobs pattern to apply scheduling constraints to underlying
Jobs—delivering value to users immediately without waiting for the entire dependency chain to
resolve.

### Job Integration - API Usage Examples

This KEP proposes enriching the core `Job` API to allow users to express their scheduling intents
through a composed scheduling configuration. The following examples show how this API represents
different Workload-aware Scheduling intents:

#### Example 1: Job with Gang Scheduling, Zone Topology, and Atomic Disruption
A batch ML training `Job` where all 4 pods must schedule together atomically (All-or-Nothing),
must co-locate within the same availability zone, and must be treated as a single unit for
disruptions (meaning if one pod is preempted, the entire group is disrupted together):

```yaml
apiVersion: batch/v1
kind: Job
spec:
  parallelism: 4
  completions: 4
  scheduling: # New API field - scheduling intent
    schedulingPolicy:
      gang: {} # MinCount is omitted: Job defaults MinCount = parallelism (4)
    schedulingConstraints:
      topology:
        - level: "topology.kubernetes.io/zone"
    disruptionMode:
      all: {} # DisruptionMode resolves to All (entire group must be disrupted together)
  template:
    spec:
      containers:
        - name: train-node
          image: training-image:v1
```

#### Example 2: Backward Compatibility and Sane Defaulting (Implicit Opt-Out)
A standard Job manifest where the `scheduling` block is omitted entirely. This natively defaults
to standard Kubernetes pod-by-pod scheduling (`Basic` mode), ensuring 100% backward compatibility
and eliminating the need for an explicit opt-out mechanism:

```yaml
apiVersion: batch/v1
kind: Job
spec:
  parallelism: 10
  completions: 10
  # The scheduling block is completely omitted (which defaults to Basic scheduling
  # and single disruption).
  # This effectively acts as an implicit opt-out from gang scheduling in the Job integration.
  template:
    spec:
      containers:
        - name: processor
          image: processor-image:v1
```

### User Stories

#### Story 1: The End-User

As a ML engineer submitting distributed training workloads to a cluster, I want to explicitly
define my scheduling requirements — such as requesting that all worker Pods are scheduled together
(gang scheduling) and placed within the same network rack (topology constraint) — directly within
my workload's YAML manifest. I expect these scheduling configurations to be intuitive,
well-documented, and to use a similar structure and vocabulary whether I am submitting a `JobSet`,
a `LWS` resource, or a company-internal batch job.

#### Story 2: The Controller Maintainer

As a maintainer of a single-level workload controller, such as the core `Job` API, I want to add
Workload-aware Scheduling capabilities to my API without having to design custom struct fields
from scratch or write reconciliation logic to manage scheduler-specific objects like `PodGroup`. By
importing standard API primitives from `scheduling.k8s.io` into my API schema and using a shared
builder library in my controller's reconcile loop, I can easily expose features like gang
scheduling to my users while ensuring consistency with the rest of the ecosystem.

#### Story 3: The Multi-Level Controller Maintainer

As a maintainer of a multi-level composite controller (e.g., `JobSet` which creates Jobs, or a
custom training operator composing `LWS`), I want to integrate WAS features using the same standard
API primitives. Furthermore, because my controller relies on composing other Kubernetes resources,
I expect this KEP to provide clear architectural guidelines on how to handle nested scheduling
intent. For example, I need recommendations on whether my parent controller should generate the
`PodGroup` directly, or if it should delegate that creation to the underlying child controllers.

### Risks and Mitigations

* **API Fragmentation and Inconsistent UX:** Because this proposal grants controller owners the
  autonomy to design and integrate their own API schemas to avoid upstream dependency bottlenecks,
  there is a risk that different controllers expose Workload-aware Scheduling (WAS) features
  differently, leading to a fragmented user experience across the ecosystem.
  * *Mitigation:* This is a conscious and deliberate trade-off: we prioritize rapid out-of-tree
    ecosystem adoption and native local consistency over delayed global uniformity (`local
    consistency > global uniformity/fragmentation`). To minimize fragmentation, we provide
    strongly-typed, reusable building blocks (like `SchedulingConstraints`, `DisruptionMode`,
    `SchedulingMode`) in the `scheduling.k8s.io` API group. By following our design
    recommendations and using these building blocks, controller owners ensure that the JSON/YAML
    schema shapes remain highly consistent and intuitive for users.

* **Split-Brain Configurations:** Because we preserve controller autonomy, a situation can arise
  where a composite wrapper controller (such as `JobSet` or `TrainJob`) implements its own custom
  wrapper-level fields or conventions to expose WAS features. In the meantime, the underlying
  child resource (such as the core `Job` API) officially integrates with WAS and introduces its
  own scheduling fields. This creates a "split-brain" configuration problem where a user of
  `JobSet` can configure scheduling directives in two parallel, potentially conflicting ways: at
  the wrapper level, or directly inside the child's nested template (e.g.,
  `spec.replicatedJobs[*].template.spec.scheduling`).
  * *Mitigation:* The composite controller remains in full control of its API and the
    translation/propagation of its templates. Since the parent controller is the sole "compiler"
    of the workload tree, it has several flexible options to resolve this duplication without
    breaking backward compatibility:
    1. **API Translation and Mapping:** The parent controller can map its existing wrapper-level
       fields to the compiled `Workload` resource, while explicitly stripping or ignoring the
       child's nested scheduling fields in the generated templates before applying them to prevent
       conflicts.
    2. **Gradual Deprecation:** The parent controller can choose to gradually deprecate its custom
       duplicate wrapper-level fields over several minor releases in favor of the child's native
       embedded fields, guiding users to a unified configuration path.
    3. **Conflict Validation:** The parent controller's validating webhooks can reject requests
       where a user attempts to populate *both* wrapper-level and child-template-level scheduling
       fields for the same workload, preventing ambiguous configurations.

## Design Details

### Core Principles & Assumptions

Integration of Workload-aware Scheduling (WAS) into workload controllers is guided by the
following design principles:

* **The Root Controller as the Compiler:** Regardless of whether a workload is a simple,
  single-level resource (like a core `Job`) or a complex, multi-level composite resource (like
  `JobSet` or `TrainJob`), the low-level scheduler-facing `Workload` resource is **always**
  compiled, created, and managed strictly by the root-most controller (the **Root Controller**):
  * **Full Context Visibility:** Only the root-most controller has the complete, high-level view
    of the entire workload structure and its logical orchestration (e.g., `JobSet` knows all its
    `replicatedJobs` and their parallelism, whereas a single child `Job` only knows its own pods).
  * **Ownership & Skip Logic:** Child controllers (like standard `Job`) observe their
    `OwnerReference` pointing to a registered parent workload and explicitly **bypass** creating
    any `Workload` objects. This prevents duplicate resource creation and guarantees a single
    source of truth. However, because `PodGroup` is the runtime representation of the `Workload`
    blueprint, child controllers may still be responsible for instantiating the corresponding
    `PodGroup` objects themselves (or delegating this to the root controller depending on the
    integration design).
* **Separation of Structure and Policy:** The integration strictly separates real-workload
  structure from scheduling policies:
  * **The Controller API owns the Structure:** The true workload API definition (e.g., `JobSet`
    or `LWS` schemas) fully defines its own shape, hierarchy, and replication mechanics. The user
    does not need to manually repeat this structure to the scheduler.
  * **The User owns the Policy:** The user knows *how* they want the workload to be scheduled
    based on their specific environment (e.g., "I want gang scheduling", "I need these workers
    colocated on the same network rack").
  * **The Controller acts as a Translator:** The real-workload controller consumes the user's
    high-level policy intent, combines it with its own structural knowledge, and acts as a
    compiler to generate the low-level `Workload` objects for the scheduler.
* **Universal Representation:** Legacy, standard pod-by-pod scheduling is represented natively as
  a first-class citizen (`Basic` mode). Controllers always generate the underlying `Workload`
  objects, using basic scheduling as the backward-compatible default for true workloads.
* **Sane Defaults and Escape Hatches:** Controllers balance their native orchestration purpose
  with backward compatibility by providing sensible defaults (e.g. standard `Job` defaulting to
  `Basic`, `LWS` defaulting to a Set of Gangs). Integrated Controllers must provide explicit
  escape hatches allowing users to override these default templates (e.g., opting out of LWS's
  default local gang back to `Basic`).

### Standardized Building Blocks Definitions (`scheduling.k8s.io`)

Following the structure of the `PodGroup` and `CompositePodGroup` APIs under development, the shared
building block primitives are categorized into distinct levels representing the layers of the
workload tree:
1. **Leaf Level (`PodGroup`):** Prefixed with `WorkloadPodGroup...`. These primitives group pods
   directly and represent standard execution boundaries.
2. **Composite Level (`CompositePodGroup`):** Prefixed with `WorkloadCompositePodGroup...`. These
   primitives coordinate groups of workloads.

This level-specific categorization allows independent API evolution. As a general design
philosophy, when a structure represents a concrete, physical "real-world" scheduling concept used
verbatim by the scheduling stack (such as `TopologyConstraint` from [KEP-5732]), we reuse it
directly across all levels. For higher-level policy abstractions introduced by this WAS layer, we
define distinct level-specific types (such as `WorkloadPodGroupSchedulingPolicy`) to ensure they
can evolve independently at each hierarchy level.

The `WorkloadPodGroup` and `WorkloadCompositePodGroup` prefixes are used to avoid name collisions
with other scheduling field structures defined directly in the `scheduling.k8s.io` group
(e.g., [KEP-5732]'s `PodGroup` structures).

To keep this specification concise and focused, we only define the detailed Go API structs for
the leaf-level `PodGroup` specific types. An analogous set of types prefixed with
`WorkloadCompositePodGroup...` is provided under the same API group.

The Go definitions are structured as follows. The shipped types carry declarative-validation (DV) markers and are declared as `+union` where exactly one member must be set:

```go
// API Group: scheduling.k8s.io/v1alpha3

// WorkloadPodGroupSchedulingConstraints defines leaf-level scheduling constraints, such as topology.
type WorkloadPodGroupSchedulingConstraints struct {
    // Topology specifies desired topological placements for all pods
    // within the scheduling group.
    // +optional
    Topology []TopologyConstraint `json:"topology,omitempty"`
}

// WorkloadPodGroupDisruptionMode defines how individual pods within a group can be disrupted.
// Exactly one mode must be set.
type WorkloadPodGroupDisruptionMode struct {
    // Single specifies that pods can be disrupted independently from each other.
    // +optional
    Single *WorkloadPodGroupSingleDisruptionMode `json:"single,omitempty"`

    // All specifies that all pods in the group must be disrupted together.
    // +optional
    All *WorkloadPodGroupAllDisruptionMode `json:"all,omitempty"`
}

// WorkloadPodGroupSingleDisruptionMode indicates that individual pods can be disrupted independently.
type WorkloadPodGroupSingleDisruptionMode struct {
    // Intentionally empty for now.
}

// WorkloadPodGroupAllDisruptionMode indicates that all pods in the group must be disrupted together.
type WorkloadPodGroupAllDisruptionMode struct {
    // Intentionally empty for now.
}

// WorkloadPodGroupSchedulingPolicy defines the scheduling policy for a group of pods.
// Exactly one policy must be set.
type WorkloadPodGroupSchedulingPolicy struct {
    // Basic specifies that standard, pod-by-pod Kubernetes scheduling behavior should be used.
    // +optional
    Basic *WorkloadPodGroupBasicSchedulingPolicy `json:"basic,omitempty"`

    // Gang specifies all-or-nothing scheduling semantics.
    // +optional
    Gang *WorkloadPodGroupGangSchedulingPolicy `json:"gang,omitempty"`
}

// WorkloadPodGroupBasicSchedulingPolicy indicates standard Kubernetes scheduling behavior.
type WorkloadPodGroupBasicSchedulingPolicy struct {
    // Intentionally empty for now.
}

// WorkloadPodGroupGangSchedulingPolicy defines the parameters for gang (all-or-nothing) scheduling.
type WorkloadPodGroupGangSchedulingPolicy struct {
    // MinCount is the minimum number of pods that must be scheduled
    // at the same time for the scheduler to admit the entire group. must be >= 1 when set
    // If omitted, the controller should inject a context-specific sane default.
    // +optional
    MinCount *int32 `json:"minCount,omitempty"`
}

// WorkloadPodGroupResourceClaim references dynamic resource claims for the group.
// Exactly one of ResourceClaimName or ResourceClaimTemplateName must be set.
type WorkloadPodGroupResourceClaim struct {
    // Name uniquely identifies this resource claim inside the group.
    Name string `json:"name"`

    // ResourceClaimName is the name of a ResourceClaim object in the same namespace.
    // +optional
    ResourceClaimName *string `json:"resourceClaimName,omitempty"`

    // ResourceClaimTemplateName is the name of a ResourceClaimTemplate object.
    // +optional
    ResourceClaimTemplateName *string `json:"resourceClaimTemplateName,omitempty"`
}
```

The composite level provides the analogous `WorkloadCompositePodGroup...` set under the same API
group, following the same shapes. Its policy building block
(`WorkloadCompositePodGroupSchedulingPolicy`) uses `minGroupCount` (the minimum number of child
groups, not pods) in place of the leaf's `minCount`.

**Embedding the building blocks & immutability.**
These structs are meant to be embedded verbatim into a controller's own API next to its other
scheduling fields (for example, in a Job's `spec.scheduling`).

The building blocks themselves are not marked `+k8s:immutable` because this would force that decision 
on every embedder, but different controllers make different choices (one may recreate the 
`Workload`/`PodGroup` on change; another may freeze it). Instead. The scheduling **policy** is a 
special case as the variant (`basic`/`gang`) must be frozen while 
`gang.minCount`/`gang.minGroupCount` stays mutable (to support scaling).

### Job Integration (batch/v1)

To deliver native, typed Workload-aware Scheduling support in core Kubernetes, we propose
integrating the standardized building blocks directly into the core `Job` API (`batch/v1`).

The new fields in the `Job` API follow the standard process to graduate to a stable type:
the new fields are gated behind a feature gate and progress through the usual Alpha → Beta → Stable
maturity levels, with the field cleared on write and ignored on read while the gate is disabled.

This integration serves as the foundational implementation ("blazing the path") that demonstrates
the viability of these building blocks before out-of-tree controllers adopt them. More design 
details are covered in [KEP-5547].

#### API Changes

We will introduce a new `Scheduling` field inside `JobSpec`. This field embeds a curated, composed
structure consisting of the standardized building blocks:

```go
// API Group: batch/v1

// JobSpec defines the desired state of a Job.
type JobSpec struct {
    // ... existing fields ...

    // Scheduling defines the Workload-aware Scheduling configuration for this Job.
    // This field is alpha-gated by the WorkloadWithJob feature gate.
    // +optional
    Scheduling *JobSchedulingConfiguration `json:"scheduling,omitempty"`
}

// JobSchedulingConfiguration composes the reusable WAS building blocks.
type JobSchedulingConfiguration struct {
    // SchedulingPolicy defines the gang or basic scheduling rules for this Job.
    // +optional
    SchedulingPolicy *schedulingv1alpha3.WorkloadPodGroupSchedulingPolicy `json:"schedulingPolicy,omitempty"`

    // SchedulingConstraints defines topology co-location constraints for the Job's pods.
    // +optional
    SchedulingConstraints *schedulingv1alpha3.WorkloadPodGroupSchedulingConstraints `json:"schedulingConstraints,omitempty"`

    // DisruptionMode specifies how the pods in this Job should be disrupted (Single vs All).
    // +optional
    DisruptionMode *schedulingv1alpha3.WorkloadPodGroupDisruptionMode `json:"disruptionMode,omitempty"`

    // ResourceClaims specifies dynamic resource claims shared across the Job's pods.
    // +optional
    ResourceClaims []schedulingv1alpha3.WorkloadPodGroupResourceClaim `json:"resourceClaims,omitempty"`
}
```

### Shared workloadbuilder Go Translation Library

To prevent every workload controller (both core and out-of-tree) from writing custom, translation
and validation logic, we propose providing a shared Go library: `workloadbuilder`.

**Package placement:** The library ships from staging as the `workloadbuilder` package under
`k8s.io/component-helpers/scheduling/schedulingv1/workloadbuilder`. It is scoped as helpers shared by multiple
core binaries, keeps a minimal dependency surface (no external deps), and is meant for this 
kind of scheduling-API translation. `k8s.io/kube-scheduler` was considered but
carries heavier dependencies and is a less natural import for out-of-tree controllers.

#### 1. Design & Architecture

This library utilizes an **Intermediate Representation (IR)** tree pattern. The architecture adopts a
**Polymorphic Bridge Pattern** to reconcile the level-specific K8s API structures (leaf-level
`PodGroup` vs. composite-level `CompositePodGroup`) with a single, uniform tree definition inside
the library:

* **Hierarchy-Agnostic Library IR:** The library defines its own internal, polymorphic structures
  (`workloadbuilder.SchedulingConfig`, `workloadbuilder.SchedulingPolicy`, etc.) that represent
  scheduling configurations in a hierarchy-agnostic way.
* **Standard Mapping:** To prevent controllers from writing custom translation boilerplate to bridge
  K8s API types to the library IR, the library maps the public, level-specific building blocks into
  its polymorphic IR internally. A controller records its user's intent on each node's `Input`: a
  `WorkloadInput` that pairs each leaf-level building block with the field path (`PathElements`)
  where it lives in the controller's API, so validation errors are reported at the exact field. The
  composite level provides the analogous input for `CompositePodGroup` building blocks.

Controller authors construct a logical tree using `WorkloadItem` representing their workload
structure, populate each node's `DefaultConfig` (the controller's sane defaults) and `Input` (the
user's intent), and invoke the builder.

The library encapsulates the following logic:
1. **Policy Resolution:** Merges default configurations with user-provided overrides (e.g.,
   resolving escape hatches uniformly across the ecosystem) into each node's `ResolvedConfig`,
   then applies that node's `Callbacks` so controllers can post-process the resolved
   configuration (e.g. defaulting gang `MinCount`).
2. **Structural Resolution:** Maps the logical tree hierarchy to the corresponding technical
   structures in the low-level scheduler `Workload` API, abstracting version variations (e.g. flat
   templates vs. nested sub-group templates).
3. **Centralized Validation:** Rejects invalid configurations early (e.g. ensuring a nested leaf
   group does not declare a conflicting disruption mode not supported by its parent).

#### 2. Controller Opt-In for New Scheduling Capabilities

Because the building-block types under `scheduling.k8s.io` are shared across all controllers, new
scheduling options may be added in future releases (e.g. a new scheduling policy or disruption mode)
that do not make sense for every controller. For example, a new policy added in v1.3x might
be valid for `JobSet` but not for `Job`.

To prevent new options from silently leaking into controllers that have not been updated to support
them, the `workloadbuilder` library adopts an **allow-list** (opt-in) validation approach rather
than a deny-list (opt-out). Controllers declare the specific set of policies and modes they support,
and the library's validation helpers reject anything not explicitly allowed. This means new
additions to the building-block API are **denied by default** until a controller explicitly updates
its allow-list.

A controller declares the policies and modes it supports on the builder's `BuildOptions`;
`Builder.Validate` resolves the config and rejects anything outside the allow-list, reporting at the
offending block's field path:

```go
// In Job's API validation (pkg/apis/batch/validation):
builder := workloadbuilder.NewBuilder(item, workloadbuilder.BuildOptions{
    AllowedPolicies:        []workloadbuilder.SchedulingPolicyOption{workloadbuilder.BasicPolicy, workloadbuilder.GangPolicy},
    AllowedDisruptionModes: []workloadbuilder.DisruptionModeOption{workloadbuilder.SingleMode, workloadbuilder.AllMode},
    // The apiserver already runs declarative validation on the building blocks,
    // so in-tree Validate performs only the complex controller-policy checks.
    DisableDeclarativeValidation: true,
})
allErrs = append(allErrs, builder.Validate(ctx, fldPath, workloadbuilder.ValidationInput{})...)
```

This gives controllers opt-in semantics: when a new policy is introduced in a future release,
existing controllers (including `Job`) will reject it until their allow-list is explicitly updated
to include the new option. Out-of-tree controllers get the same guarantee by updating their vendored
library version and extending their allow-list.

Long-term, this pattern can migrate to Declarative Validation (DV) using `+k8s:subfield` markers,
eliminating the need for hand-written allow-list calls while preserving the same opt-in semantics:

```go
type JobSpec struct {
    // ...
    // +k8s:subfield(disruptionMode)=+k8s:allowed=single,all
    Scheduling *JobSchedulingConfiguration
}
```

The structural building-block constraints are already generated as DV validators; until subfield
allow-list markers land in DV, the builder's allow-list checks in `Validate` serve as a lightweight,
defensive bridge that keeps the overhead minimal for controller integrators.

#### 3. Library API Definition

```go
package workloadbuilder

import (
    "context"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/util/validation/field"
    schedulingv1alpha3 "k8s.io/api/scheduling/v1alpha3"
)

// SchedulingConfig is the polymorphic, hierarchy-agnostic IR model of the PodGroup/CompositePodGroup.
type SchedulingConfig struct {
    Constraints    *SchedulingConstraints
    DisruptionMode *DisruptionMode
    Policy         *SchedulingPolicy
    ResourceClaims []ResourceClaim

    // PriorityClassName is copied onto the compiled PodGroupTemplate so a PodGroup
    // materialized from it inherits the group's priority.
    PriorityClassName string
}

type SchedulingConstraints struct {
    Topology []schedulingv1alpha3.TopologyConstraint
}

type DisruptionMode struct {
    Single *SingleDisruptionMode
    All    *AllDisruptionMode
}

type SingleDisruptionMode struct {
    // Intentionally empty for now.
}

type AllDisruptionMode struct {
    // Intentionally empty for now.
}

type SchedulingPolicy struct {
    Basic *BasicSchedulingPolicy
    Gang  *GangSchedulingPolicy
}

type BasicSchedulingPolicy struct {
    // Intentionally empty for now.
}

type GangSchedulingPolicy struct {
    MinCount *int32
}

type ResourceClaim struct {
    Name                      string
    ResourceClaimName         *string
    ResourceClaimTemplateName *string
}

// SchedulingConfigFunc post-processes the merged SchedulingConfig after the
// default/user merge. Controllers use it for defaulting (e.g. gang MinCount) but
// it is general-purpose and may perform any controller-specific adjustment.
type SchedulingConfigFunc func(*SchedulingConfig)

// WorkloadItem represents a logical component of a workload (e.g., the whole JobSet,
// a specific ReplicatedJob role, or a single standalone Job).
type WorkloadItem struct {
    // Name is the logical identifier of this component; it becomes the
    // PodGroupTemplate name and must be non-empty.
    Name string

    // DefaultConfig is the controller's "sane defaults" for any field the user
    // left unset, based on its orchestration domain logic.
    DefaultConfig *SchedulingConfig

    // Input holds the user's intent for this node as the versioned building blocks
    // paired with the field paths where the controller embeds them. Nil sub-fields
    // fall back to DefaultConfig field-by-field.
    Input WorkloadInput

    // Callbacks run in order against the resolved config after the default/user merge.
    Callbacks []SchedulingConfigFunc

    // Children contains the logical sub-components of this workload.
    // - If len(Children) > 0, the node is a structural group (CompositePodGroupTemplate).
    // - If len(Children) == 0, the node is a leaf (PodGroup).
    //
    // Multi-level compilation (children -> CompositePodGroupTemplate) is the designed
    // composite extension; the alpha library compiles a single leaf node into one
    // PodGroupTemplate and gains composite-tree compilation in a fast-follow (see the
    // note below).
    Children []*WorkloadItem
}

// WorkloadInput bundles the leaf-level building blocks a controller embeds in its
// own API, each paired with its field path. The zero value means "nothing set", so
// callers only populate the blocks they care about. The composite level provides an
// analogous input for the WorkloadCompositePodGroup* building blocks.
type WorkloadInput struct {
    Policy         PolicyInput
    Constraints    ConstraintsInput
    DisruptionMode DisruptionModeInput
    ResourceClaims ResourceClaimsInput
}

// PolicyInput pairs the scheduling policy building block with its field path.
// PathElements is the path, relative to the WorkloadItem's rootPath, at which the
// block is embedded: e.g. a rootPath of `spec.scheduling` with PathElements
// []string{"schedulingPolicy"} reports errors at `spec.scheduling.schedulingPolicy`.
type PolicyInput struct {
    PodGroupData *schedulingv1alpha3.WorkloadPodGroupSchedulingPolicy
    PathElements []string
}

type ConstraintsInput struct {
    PodGroupData *schedulingv1alpha3.WorkloadPodGroupSchedulingConstraints
    PathElements []string
}

type DisruptionModeInput struct {
    PodGroupData *schedulingv1alpha3.WorkloadPodGroupDisruptionMode
    PathElements []string
}

type ResourceClaimsInput struct {
    PodGroupData []schedulingv1alpha3.WorkloadPodGroupResourceClaim
    PathElements []string
}

// SchedulingPolicyOption and DisruptionModeOption enumerate the policies and modes
// a controller opts into. Validate rejects anything outside the allow-list.
type SchedulingPolicyOption int

const (
    BasicPolicy SchedulingPolicyOption = iota
    GangPolicy
)

type DisruptionModeOption int

const (
    SingleMode DisruptionModeOption = iota
    AllMode
)

// BuildOptions carries the identity of the compiled Workload plus the controller's
// scheduling allow-lists.
type BuildOptions struct {
    Name      string
    Namespace string
    // Owner becomes the Workload's controllerRef, used for discovery and GC.
    Owner *metav1.OwnerReference

    AllowedPolicies        []SchedulingPolicyOption
    AllowedDisruptionModes []DisruptionModeOption

    // DisableDeclarativeValidation skips declarative validation on the input blocks.
    // In-tree controllers set this because the apiserver already runs DV.
    DisableDeclarativeValidation bool
}

// Builder turns a controller's WorkloadItem tree into scheduler-facing objects.
// Construct it with NewBuilder (or NewBuilderFromExistingWorkload), then call
// Validate, BuildWorkload, and NewPodGroup.
type Builder struct { /* unexported fields */ }

// NewBuilder returns a Builder for the given WorkloadItem tree and options.
func NewBuilder(root *WorkloadItem, opts BuildOptions) *Builder

// NewBuilderFromExistingWorkload returns a Builder that materializes PodGroups
// from an already-persisted Workload rather than compiling one from a WorkloadItem
// tree. Use it when a parent controller (or a hand-authored template) owns and
// compiled the Workload. BuildWorkload is refused and Validate is a no-op; only
// NewPodGroup is meaningful, using opts.Owner for the PodGroup's controller ownerRef.
func NewBuilderFromExistingWorkload(workload *schedulingv1alpha3.Workload, opts BuildOptions) *Builder

// ValidationInput carries the parameters Validate only consults when declarative
// validation is enabled. Its zero value means a create with no previous object,
// which is the common case; a caller running with DisableDeclarativeValidation can
// always pass the zero value.
type ValidationInput struct {
    // OldRoot is the previously persisted WorkloadItem. It is nil for a create and
    // non-nil for an update. Validate infers the operation from it, so the
    // update-time declarative checks run exactly when OldRoot is set. Only
    // OldRoot.Name (to correlate it with the new root) and OldRoot.Input (the
    // previous versioned data) are consulted; DefaultConfig, Callbacks, and
    // Input.*.PathElements are ignored because the resolved config and error paths
    // always come from the new root.
    OldRoot *WorkloadItem
}

// Validate runs declarative validation on the input blocks (unless disabled) plus
// the controller-policy checks DV cannot express (allow-lists, and cross-field rules
// such as rejecting `all` disruption with the Basic policy), reporting errors at each
// block's field path. For a create, pass the zero ValidationInput; for an update, set
// OldRoot to the previous tree, and Validate runs the update-time checks. A root name
// mismatch between OldRoot and the new root is an invocation-contract bug and is
// reported as an InternalError.
func (b *Builder) Validate(ctx context.Context, rootPath *field.Path, input ValidationInput) field.ErrorList

// BuildWorkload compiles the tree into a Workload, sets its identity and
// controllerRef, and caches the result so PodGroups can be materialized from it.
func (b *Builder) BuildWorkload() (*schedulingv1alpha3.Workload, error)

// NewPodGroup materializes a runtime PodGroup from the named PodGroupTemplate of the
// Builder's Workload (compiled via BuildWorkload or supplied via
// NewBuilderFromExistingWorkload).
func (b *Builder) NewPodGroup(podGroupName, templateName string) (*schedulingv1alpha3.PodGroup, error)
```

The translation from the versioned building blocks (recorded in each node's `Input`) into the
polymorphic IR is performed internally by the library while resolving the tree. A leaf node's 
`WorkloadInput` maps the `WorkloadPodGroup...` blocks into the IR, and a composite node maps its analogous `WorkloadCompositePodGroup...` blocks.
Keeping the mapping internal means there is a single, consistent translation path (exercised by both
`Validate` and `BuildWorkload`) rather than a public helper each controller must call correctly.


#### 4. Library Usage Example (Job)

This is how the core `Job` controller integrates with the `workloadbuilder` library to compile its
flat `Workload` structure:

```go
import (
    batch "k8s.io/api/batch/v1"
    schedulingv1alpha3 "k8s.io/api/scheduling/v1alpha3"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    workloadbuilder "k8s.io/component-helpers/scheduling/schedulingv1/workloadbuilder"
    "k8s.io/utils/ptr"
)

// mapSchedulingInput translates the Job's user-facing spec.scheduling block into
// a WorkloadInput, pairing each building block with the spec.scheduling sub-path
// where it lives. It returns the zero value for a nil block so the builder falls
// back to the controller's default config.
func mapSchedulingInput(cfg *batch.JobSchedulingConfiguration) workloadbuilder.WorkloadInput {
    if cfg == nil {
        return workloadbuilder.WorkloadInput{}
    }
    return workloadbuilder.WorkloadInput{
        Policy: workloadbuilder.PolicyInput{
            PodGroupData: cfg.SchedulingPolicy,
            PathElements: []string{"schedulingPolicy"},
        },
        Constraints: workloadbuilder.ConstraintsInput{
            PodGroupData: cfg.SchedulingConstraints,
            PathElements: []string{"schedulingConstraints"},
        },
        DisruptionMode: workloadbuilder.DisruptionModeInput{
            PodGroupData: cfg.DisruptionMode,
            PathElements: []string{"disruptionMode"},
        },
        ResourceClaims: workloadbuilder.ResourceClaimsInput{
            PodGroupData: cfg.ResourceClaims,
            PathElements: []string{"resourceClaims"},
        },
    }
}

// defaultMinCountForJob returns a callback that defaults an unset gang MinCount
// to the Job's parallelism, clamped to a minimum of 1 (parallelism may be 0 for
// a suspended Job, but MinCount must be positive). It mutates only the resolved
// config, never writing the derived value back onto the Job's spec.
func defaultMinCountForJob(job *batch.Job) workloadbuilder.SchedulingConfigFunc {
    return func(cfg *workloadbuilder.SchedulingConfig) {
        if cfg == nil || cfg.Policy == nil || cfg.Policy.Gang == nil {
            return
        }
        if cfg.Policy.Gang.MinCount == nil {
            cfg.Policy.Gang.MinCount = new(max(int32(1), ptr.Deref(job.Spec.Parallelism, 1)))
        }
    }
}

// multiplyMinCountForAdjustedJob is an example of a non-defaulting adjustment: callbacks
// are free to implement arbitrary, controller-specific logic when needed.
func multiplyMinCountForAdjustedJob(job *batch.Job) workloadbuilder.SchedulingConfigFunc {
    return func(cfg *workloadbuilder.SchedulingConfig) {
      if job.Annotations["isAdjustedJob.example.com"] == "true" {  
        if cfg.Policy.Gang != nil {  
            cfg.Policy.Gang.MinCount *= 42  
        }  
      }  
    }
}

// buildWorkloadItem assembles the single-node logical workload tree for a Job.
// The Job defaults to Basic scheduling; the user's spec.scheduling overrides it.
func buildWorkloadItem(job *batch.Job) *workloadbuilder.WorkloadItem {
    return &workloadbuilder.WorkloadItem{
        Name: podGroupTemplateName(job),
        DefaultConfig: &workloadbuilder.SchedulingConfig{
            Policy:            &workloadbuilder.SchedulingPolicy{Basic: &workloadbuilder.BasicSchedulingPolicy{}},
            PriorityClassName: job.Spec.Template.Spec.PriorityClassName,
        },
        Input:     mapSchedulingInput(job.Spec.Scheduling),
        Callbacks: []workloadbuilder.SchedulingConfigFunc{defaultMinCountForJob(job)},
    }
}

// generateWorkload compiles the Job's spec.scheduling into a Workload via the
// shared workloadbuilder library. BuildWorkload also sets the controller
// ownerReference and spec.controllerRef pointing at the Job.
func (jm *Controller) generateWorkload(job *batch.Job) (*schedulingv1alpha3.Workload, error) {
    return workloadbuilder.NewBuilder(buildWorkloadItem(job), workloadbuilder.BuildOptions{
        Name:      computeWorkloadName(job),
        Namespace: job.Namespace,
        Owner:     metav1.NewControllerRef(job, controllerKind),
    }).BuildWorkload()
}
```

`Callbacks` are ordinary functions the controller sets on a node; `defaultMinCountForJob` is the
defaulting one used here, but because a callback receives the resolved config it can apply any
controller-specific adjustment. Note the allow-lists (`AllowedPolicies`/`AllowedDisruptionModes`)
are not set on this compile path, they are an apiserver-validation concern applied where the Job's
validation calls `Validate`.

### Reference Integration Examples: JobSet (Multi-Level)

This section provides **non-normative reference examples** demonstrating how a complex,
multi-level composite controller (such as `JobSet`) can integrate with the Workload-aware
Scheduling (WAS) building blocks and the `workloadbuilder` library.

These examples prove the viability and flexibility of the library for hierarchical workloads. The
final API design and integration details remain at the sole discretion of the `JobSet` project
maintainers.

We explore two different API representation options that `JobSet` could choose to adopt.

#### 1. Option A: Template Delegation Model (Nested Configuration)
In this model, `JobSet` defines scheduling directives globally at the root
(`JobSet.spec.scheduling`) for policies that apply to the entire group. For leaf-level scheduling
(individual `ReplicatedJobs`), it directly leverages the nested scheduling fields already present
inside the embedded `JobTemplateSpec` (e.g., `spec.replicatedJobs[*].template.spec.scheduling`).

##### Example YAML Manifest

```yaml
apiVersion: jobset.x-k8s.io/v1alpha2
kind: JobSet
spec:
  scheduling: # Global policy: applies to the entire JobSet
    schedulingPolicy:
      basic: {} # ESCAPE HATCH: Disable global "gang of gangs" so components start independently
  replicatedJobs:
    - name: driver
      replicas: 1
      template:
        spec:
          # Defaults to Basic (pod-by-pod) scheduling
          containers:
            - name: main
              image: driver-image
    - name: workers
      replicas: 16
      template:
        spec:
          scheduling: # Leaf-level policy declared inside the nested Job template
            schedulingConstraints:
              topology:
                - level: "topology.kubernetes.io/rack" # Co-locate workers on same rack
          containers:
            - name: worker
              image: worker-image
```

#### 2. Option B: Centralized 'Targeted Policies' Model (Root-only Configuration)
In this model, `JobSet` does not expose or use the nested child template fields. Instead, all
scheduling configurations—both global and local—are declared centrally inside a single root-level
`spec.scheduling` block. It uses a "shadow tree" pattern to map scheduling policies to specific
`ReplicatedJobs` by name (which directly follows the established `targetReplicatedJob` convention
already used in `JobSet` features like `FailurePolicyRule`).

##### Example YAML Manifest

```yaml
apiVersion: jobset.x-k8s.io/v1alpha2
kind: JobSet
spec:
  scheduling: # All scheduling policies are defined here at the root
    schedulingPolicy:
      basic: {} # Global policy: components schedule independently
    replicatedJobPolicies:
      - targetReplicatedJob: "workers" # Policy target
        schedulingConstraints:
          topology:
            - level: "topology.kubernetes.io/rack" # Co-locate workers on same rack
  replicatedJobs:
    - name: driver
      replicas: 1
      template:
        spec:
          containers:
            - name: main
              image: driver-image
    - name: workers
      replicas: 16
      template:
        spec:
          # Templates remain completely clean of scheduling directives
          containers:
            - name: worker
              image: worker-image
```

---

#### 3. Controller Integration and workloadbuilder Mapping Go Code

Regardless of which API model `JobSet` adopts, the controller maps its structural spec into a
`workloadbuilder.WorkloadItem` tree: the root is a composite node (its `Children` are the
`ReplicatedJob` roles) carrying the composite scheduling policy, and each child leaf carries the
leaf building blocks. The builder then compiles the tree into a `Workload` with a
`CompositePodGroupTemplate` over the child `PodGroupTemplate`s:

```go
root := &workloadbuilder.WorkloadItem{
    Name: "jobset-root",
    // A composite node: the group-of-groups policy lives in the IR just like a leaf's.
    DefaultConfig: &workloadbuilder.SchedulingConfig{
        Policy: &workloadbuilder.SchedulingPolicy{Gang: &workloadbuilder.GangSchedulingPolicy{}},
    },
    // Input maps the JobSet's composite WorkloadCompositePodGroup* building block(s).
    Input: mapJobSetSchedulingInput(jobSet),
    Children: []*workloadbuilder.WorkloadItem{
        leafItemForReplicatedJob(jobSet, "driver"),
        leafItemForReplicatedJob(jobSet, "workers"),
    },
}
workload, err := workloadbuilder.NewBuilder(root, opts).BuildWorkload()
```

### Recommendations for Multi-Level Composite Controllers

Integrating Workload-aware Scheduling (WAS) into multi-level composite controllers (where
controllers orchestrate other controllers, such as `JobSet` creating core `Jobs`, or a Kubeflow
`TrainJob` composing a `JobSet`) introduces unique coordination challenges. Composite controllers
should adhere to the following guidelines:

#### 1. Runtime PodGroup and CompositePodGroup Lifecycle Management

For single-level controllers (e.g., standard `Job`), the ownership boundaries are straightforward:
the Job controller manages both the static `Workload` resource and the corresponding runtime
`PodGroup` objects.

For multi-level composite controllers, two distinct lifecycle management strategies are available:
* **Centralized Management:** The root controller (e.g., `JobSet`) compiles the `Workload` and is
  also fully responsible for creating and managing all runtime `PodGroup` or `CompositePodGroup`
  objects.
* **Delegated Management:** The root controller only compiles and creates the n-level `Workload`
  resource, and delegates the creation and management of individual runtime `PodGroup` objects to
  its child execution controllers (e.g., delegating to standard `Job` controllers).

**Alpha Phase Strategy:** For this initial alpha phase, we intentionally **do not mandate** a
single recommended lifecycle management strategy for multi-level controllers. Controller
maintainers and ecosystem integrators are encouraged to experiment with both centralized and
delegated management patterns. The authors of this KEP will observe these patterns in the wild,
gather user and operator feedback, and generalize these best practices into a standardized,
unified lifecycle convention in a subsequent phase.

#### 2. Downward Template and Parent Mapping via Well-Known Annotations

If a composite controller delegates runtime `PodGroup` management to child execution controllers,
we must solve a crucial multi-level coordination problem. The child controller needs two distinct
pieces of information to construct and place its runtime scheduling objects correctly:

1. **Template Mapping:** Which `PodGroupTemplate` or `CompositePodGroupTemplate` inside the parent's
   compiled `Workload` corresponds to this child's pods (enabling correct policy/constraint
   compilation).
2. **Parent Instance Linkage:** Which specific runtime `CompositePodGroup` instance name in the
   namespace this newly created child must attach to (under its "parentRef"). This linkage is
   especially critical in multi-instantiated environments (such as `LeaderWorkerSet` / LWS), where a
   composite controller may instantiate multiple separate `CompositePodGroup` objects from the exact
   same template (one per replica).

##### The Solution: Downward Mapping Annotations

To resolve this template and hierarchy mapping without structural API schema changes, the root and
intermediate orchestrators must propagate these linkages downwards by injecting two well-known
metadata annotations directly into the created child objects (for example, the `JobSet`
controller sets these annotations on each standard `Job` resource it creates):

* **Template Linkage Annotation:**
  * **Annotation Key:** `scheduling.k8s.io/group-template-name`
  * **Value:** The unique name of the target `PodGroupTemplate` or `CompositePodGroupTemplate`
    defined inside the parent `Workload` resource (ensuring direct mapping, as all template
    names inside a Workload are guaranteed to be unique). For example, in a
    `TrainJob -> JobSet -> Job` hierarchy this tells the child `Job` which template from the root
    `Workload` to use to create its `PodGroup`.
* **Parent Instance Linkage Annotation:**
  * **Annotation Key:** `scheduling.k8s.io/parent-compositepodgroup`
  * **Value:** The exact resource name of the parent `CompositePodGroup` object in the same
    namespace that the child's newly created group must attach to. This is required only in the
    delegated lifecycle model (the parent creates the `Workload` and `CompositePodGroup`, but the
    child creates its own `PodGroup`); when the parent centrally manages both the `CompositePodGroup`
    and the `PodGroup`s, this annotation is unnecessary.

We strictly use **unstructured metadata annotations** rather than introducing new structural fields
in the child's API schemas for this coordination. These mappings are transient, internal, and
automatically managed by composite operators during compilation, not user-configurable scheduling
intents.

### Go Package Placement & Graduation Strategy

Embedding reusable building block Go structures (defined in a pre-stable package like
`scheduling.k8s.io/v1alpha3`) directly into a stable GA type (like `batch/v1.JobSpec`) during its
Alpha phase introduces package dependency and graduation challenges.

In the Go language, changing the import path of an embedded field inside a GA struct constitutes a
breaking change in client libraries. To solve this graduation compatibility trap without forcing
identical structure duplication across different apiGroups, we adopt the following approved
transition pattern:

* **Alpha Phase:** The shared building blocks are defined in the pre-stable
  `scheduling.k8s.io/v1alpha3` package. The standard Kubernetes import rules allow stable GA
  groups (`batch/v1`) to import pre-stable packages as long as the field itself remains gated in
  Alpha.
* **Graduation to Beta/GA:** When the composed field is promoted to default-enabled (Beta/GA in
  the `v1` type), we bypass the intermediate `v1beta1` package version entirely (since wire-format
  compatibility is already committed at the `v1` resource level). We graduate the building block
  structs straight into the stable `scheduling.k8s.io/v1` package and update the field inside
  `batch/v1.JobSpec` to reference the `v1` type.
* **Go Type Aliasing for Compatibility:** To prevent breaking third-party Go controllers that
  still import the older alpha package, we replace the physical structures in `v1alpha3` with **Go
  Type Aliases (`=`)** pointing to the new stable `v1` types. This is a well-established, approved
  Kubernetes API pattern (previously used in the `admissionregistration` API group) that allows
  external codebases to compile seamlessly while gradually transitioning their imports over
  multiple releases.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

Job-specific test plans are tracked in [KEP-5547].

##### Unit tests

- Add tests that verify:
  - `workloadbuilder` compiles a `Basic` policy into the expected `Workload`/`PodGroup`
  - `workloadbuilder` compiles a `Gang` policy into the expected `Workload`/`PodGroup`
  - `workloadbuilder` correctly maps topology constraints, disruption mode, and resourceClaims
  - `workloadbuilder` merges controller defaults with user overrides (e.g. user `Gang` overrides
    controller default `Basic`)
  - `workloadbuilder` runs node `Callbacks` after merging config, and a defaulting callback
    fills `gang.minCount` (e.g. from a Job's parallelism) when omitted
  - `workloadbuilder` `Validate` rejects semantically invalid configurations
  - Single-level `WorkloadItem` (flat, no children) produces a leaf `PodGroup` only
  - the library correctly translates the leaf building blocks recorded in a `WorkloadItem.Input`
    (`WorkloadInput`) into the library IR
  - Multi-level `WorkloadItem` tree (with children) produces a `CompositePodGroup` with correct 
    parent–child structure, translating the composite `WorkloadCompositePodGroupSchedulingPolicy` block into the IR
- Reference integration tests for multi-level controllers (e.g. `JobSet`) verify that the 
  `workloadbuilder` produces the expected `CompositePodGroup` and child `PodGroup` objects from 
  a composite `WorkloadItem` tree.

##### Integration tests

- Verify that a single-level controller (Job) can create the correct `Workload`/`PodGroup` via
  `workloadbuilder` — covered in [KEP-5547]
- Verify that a multi-level controller (e.g. `JobSet`) can produce a `CompositePodGroup` with
  multiple child `PodGroups` via the `workloadbuilder` library
- Verify that updating `gang.minCount` triggers recompilation of the `Workload` and re-sync of
  the `PodGroup`

##### e2e tests

- Gang scheduling end-to-end: all pods scheduled together or none via `workloadbuilder`-compiled
  `Workload`/`PodGroup`
- Mixed workloads: gang and basic Jobs coexist without interference

### Graduation Criteria

#### Alpha

- Reusable scheduling API building blocks (`SchedulingConstraints`, `DisruptionMode`,
  `SchedulingMode`, `ResourceClaim`) introduced under the `scheduling.k8s.io` API group.
- The shared `workloadbuilder` Go translation library implemented in the `k8s.io/component-helpers`
  staging repository.
- Comprehensive unit and integration tests added for the `workloadbuilder` library to verify
  correct resource translation and default-overriding logic.
- Core `Job` API (batch/v1) integrated with the standardized WAS building blocks and validated in
  the alpha phase.

#### Beta

- At least one multi-level composite workload controller (such as `JobSet`, `LeaderWorkerSet`, or
  Kubeflow `TrainJob`) successfully integrated using the standardized building blocks and the
  `workloadbuilder` library.
- Clear recommendations on runtime `PodGroup` / `CompositePodGroup` creation and lifecycle
  management for multi-level composite controllers finalized and validated in practice.
- User feedback gathered on usability, confirming that the proposed approach provides a natural
  and cohesive UX.

#### GA

- TBD once the KEP promoted to beta

### Upgrade / Downgrade Strategy

The API building blocks are not top-level objects, and thus are not exposed directly 
by kube-apiserver. Update/Downgrade strategy for top-level APIs (like `Job`) are 
described in detail in corresponding KEPs.

### Version Skew Strategy

`workloadbuilder` adds no version-skew constraints of its own, each component vendors a fixed
version at build time. Skew applies only to the runtime components of each integration.

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

This KEP itself is a code-level change and a building block for few KEPs. 
The API building blocks and libraries can't really be disabled. The integration 
with those however is handled by feature gates dedicated to integrations (e.g. 
`WorkloadWithJob` for the integration with Job API and job-controller). This 
can be disabled and it's described in detail in corresponding KEPs (e.g. 
[KEP-5547] for Job integration).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

The building blocks and libraries themselves aren't gated, so there is nothing to roll back at this
level. Rollback is a property of each integration, disabling that integration's dedicated gate (e.g.
`WorkloadWithJob`) is what stops the building-block fields from being served.

###### What happens if we reenable the feature if it was previously rolled back?

This is likewise governed by the integration rather than the building blocks. Reenabling an
integration's gate is handled according to each controller's enablement/disablement strategy.

###### Are there any tests for feature enablement/disablement?

Since the building blocks aren't gated, enablement/disablement tests live with each 
integration and are described in the corresponding KEPs. This KEP is covered by library 
and unit tests for the building blocks themselves.

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

###### Will enabling / using this feature result in any new API calls?

Enabling an integration's gate adds no new API calls by itself, it only allows the integration's
building-block fields to be persisted. The new calls (creating `Workload`/`PodGroup` objects) are
made by integrating controllers when a user opts in. Workloads that omit the `scheduling` block
generate no new API calls.

###### Will enabling / using this feature result in introducing new API types?

Yes, the building-block field types under `scheduling.k8s.io/v1alpha3`, embedded into integrating
APIs.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, but only when a user opts in. This KEP itself only adds optional building-block fields
to integrating workload objects. The larger effect — creating `Workload` and `PodGroup` (or
`CompositePodGroup`) objects (~500 bytes each, typically one per opted-in workload) — is performed
by integrating controllers and quantified per integration.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

This KEP itself adds only in-process API translation (negligible CPU) in controllers that vendor
the library. The user-visible effects come from the integrations and the scheduler and 
apply only to opted-in workloads. Workloads that omit the `scheduling` block are unaffected.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

This KEP itself adds only the building-block fields and the build-time translation library
(negligible CPU).

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature operates entirely at the control-plane/API level and does not consume any node-level resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2026-06-03: KEP Created for alpha release
- 2026-07-15: Synced the KEP with the implementation of the `workloadbuilder` library, the Job
  building-block fields and the `workloadbuilder` API (`Validate` with a
  `ValidationInput` and the `NewBuilderFromExistingWorkload` constructor).
- 2026-07-18: Synced the `CompositePodGroup` design with the codebase and review feedback from the
  building-blocks library.

## Drawbacks

  * **Reduced global uniformity / API fragmentation:** Because each controller composes
    its own user-facing scheduling API from the shared building blocks rather than a
    single unified schema, the exact shape and vocabulary of the `scheduling`
    configuration can differ between controllers.
  * **Shared-library coupling and version skew:** Out-of-tree controllers that adopt the
    `workloadbuilder` library take on a dependency whose translation/defaulting logic
    must stay compatible across controller and library versions. Skew between a
    controller's vendored library version and the cluster's `scheduling.k8s.io` API
    version can lead to subtle behavioral differences.
  * **Additional API surface to maintain:** The standardized building blocks add new
    types under `scheduling.k8s.io` that must evolve carefully to remain
    backward-compatible across the many controllers that embed them.

## Alternatives

During the design phase, we initially pursued a highly unified, top-down compiler vision outlined
in the [[Public] API Design for WAS Controller
Integration](https://docs.google.com/document/d/1VG7Zto9JYuPG4Anb01WMRryJlfV6met0jgob3T2NjZ4/edit?tab=t.str8vvikk64z).

However, as we analyzed the implementation details, we discovered two fatal architectural and
logistical challenges documented in [[Public] WAS Controller API - challenges and potential
alternatives](https://docs.google.com/document/d/13EIkSvj7bPeD9NaORLrJWAuvZ-zSfIe3l6cScGHNHoM/edit?tab=t.9eobkyll7zgq)
that made the original unified API vision unfeasible within a reasonable timeframe:

### 1. Implementation Complexity & The "Transitive Capability Leak"
As detailed in [[Public] The "capability leak" in
go/was-controller-api](https://docs.google.com/document/d/1bOn210d7FL0fl5T8RjEgq1Sfk2GRWzFAyYnyYMoU-Io),
because composite workloads (such as `JobSet` or `TrainJob`) natively wrap child templates (like
standard `JobTemplateSpec`), any new scheduling field introduced at the child level transitively
propagates ("leaks") up the schema stack. Handling these nested configurations requires massive,
complex boilerplate inside every intermediate controller (e.g., reconcilers dynamically checking
if they are the root compiler, managing owner references, and validating nested fields), making
the unified compiler pattern highly cumbersome and fragile.

### 2. The Upstream Dependency Bottleneck
The most critical issue with the original unified API design is the strict **Controller
Integration Dependency chain**. Under a monolithic, cascading rollout, integrating a new
scheduling feature into a top-level out-of-tree controller (such as `TrainJob` or `RayJob`) was
strictly blocked by the successful integration of all intermediate child controllers (waiting
first for core `Job` and then `JobSet`). This dependency chain would delay crucial Workload-aware
Scheduling features for quarters or years, which is completely unacceptable when the user demand
in the AI/ML space is immediate.

### The Chosen Solution: Autonomous Composed Configurations & Conscious Trade-offs

Rather than delaying critical features, this KEP embraces **Controller Autonomy**. Sponsoring
out-of-tree controllers have full authority to design their own composed configurations using the
standard `scheduling.k8s.io` building blocks and the `workloadbuilder` library.

This represents a conscious and deliberate architectural trade-off:

* **Local Consistency > Global Uniformity/Fragmentation:** We prioritize native, idiomatic
  consistency within each controller's local API over a globally unified, rigid schema. Enabling
  `JobSet` to utilize its established `targetReplicatedJobs` convention is far more intuitive for
  its users than forcing a single, shared structure across the entire ecosystem.
* **Time-to-Market > Perfect API:** In the fast-paced AI and machine learning landscape, workload
  requirements change from month to month. Users need working scheduling capabilities today, not
  an idealized but delayed API a year from now. A "prettier" global API structure is not an
  acceptable justification for blocking immediate ecosystem adoption.




[KEP-5547]: https://kep.k8s.io/5547
[KEP-5732]: https://kep.k8s.io/5732
[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website
