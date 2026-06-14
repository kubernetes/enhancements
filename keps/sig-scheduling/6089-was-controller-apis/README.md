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
    - [1. API Changes](#1-api-changes)
    - [2. Defaulting Rules](#2-defaulting-rules)
  - [Shared workloadbuilder Go Translation Library](#shared-workloadbuilder-go-translation-library)
    - [1. Design &amp; Architecture](#1-design--architecture)
    - [2. Controller Opt-In for New Scheduling Capabilities](#2-controller-opt-in-for-new-scheduling-capabilities)
    - [3. Library API Definition](#3-library-api-definition)
    - [4. Library Usage Example (Job)](#4-library-usage-example-job)
  - [Job Integration - Handling Updates and Mutability](#job-integration---handling-updates-and-mutability)
    - [Gang MinCount Defaulting &amp; Scaling Behavior](#gang-mincount-defaulting--scaling-behavior)
    - [Reconciliation Flow upon Updates](#reconciliation-flow-upon-updates)
    - [API Validation via the workloadbuilder Library](#api-validation-via-the-workloadbuilder-library)
      - [Allowing Mutability of spec.parallelism](#allowing-mutability-of-specparallelism)
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
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
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
    policy:
      gang: {} # MinCount is omitted: Job defaults MinCount = parallelism (4)
    constraints:
      topology:
        - level: "topology.kubernetes.io/zone"
    disruption:
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

The Go definitions are structured as follows:

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
// Exactly one mode can be set.
//
// +union
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
// +union
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
    // at the same time for the scheduler to admit the entire group.
    // If omitted, the controller should inject a context-specific sane default.
    // +optional
    MinCount *int32 `json:"minCount,omitempty"`
}

// WorkloadPodGroupResourceClaim references dynamic resource claims for the group.
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
### Job Integration (batch/v1)

To deliver native, typed Workload-aware Scheduling support in core Kubernetes, we propose
integrating the standardized building blocks directly into the core `Job` API (`batch/v1`).

This integration serves as the foundational implementation ("blazing the path") that demonstrates
the viability of these building blocks before out-of-tree controllers adopt them. More design 
details are covered in [KEP-5547].

#### 1. API Changes
We will introduce a new `Scheduling` field inside `JobSpec`. This field embeds a curated, composed
structure consisting of the standardized building blocks:

```go
// API Group: batch/v1

// JobSpec defines the desired state of a Job.
type JobSpec struct {
    // ... existing fields ...

    // Scheduling defines the Workload-aware Scheduling configuration for this Job.
    // +optional
    Scheduling *JobSchedulingConfiguration `json:"scheduling,omitempty"`
}

// JobSchedulingConfiguration composes the reusable WAS building blocks.
type JobSchedulingConfiguration struct {
    // Policy defines the gang or basic scheduling rules for this Job.
    // +optional
    Policy *schedulingv1alpha3.WorkloadPodGroupSchedulingPolicy `json:"policy,omitempty"`

    // Constraints defines topology co-location constraints for the Job's pods.
    // +optional
    Constraints *schedulingv1alpha3.WorkloadPodGroupSchedulingConstraints `json:"constraints,omitempty"`

    // DisruptionMode specifies how the pods in this Job should be disrupted (Single vs All).
    // +optional
    DisruptionMode *schedulingv1alpha3.WorkloadPodGroupDisruptionMode `json:"disruptionMode,omitempty"`

    // ResourceClaims specifies dynamic resource claims that are shared across the Job's pods.
    // +optional
    ResourceClaims []schedulingv1alpha3.WorkloadPodGroupResourceClaim `json:"resourceClaims,omitempty"`
}
```

#### 2. Defaulting Rules
To ensure 100% backward compatibility and prevent breaking existing CI/CD pipelines:
* If `Scheduling` is unset or `Policy` is nil, the Job controller defaults to the
  `Basic` scheduling policy (meaning standard Kubernetes pod-by-pod scheduling).
* If a user configures `Policy` to use `Gang` scheduling, but leaves `MinCount` unset,
  the Job controller automatically injects a context-aware sane default: `MinCount = parallelism`
  (or `MinCount = completions` if parallelism is unset).
* Users can explicitly configure an escape hatch (e.g., opting in only to topology constraints
  while maintaining standard `Basic` pod-by-pod scheduling).

### Shared workloadbuilder Go Translation Library

To prevent every workload controller (both core and out-of-tree) from writing custom, translation
and validation logic, we propose providing a shared Go library: `workloadbuilder`.

#### 1. Design & Architecture

This library utilizes an **Intermediate Representation (IR)** tree pattern. The architecture adopts a
**Polymorphic Bridge Pattern** to reconcile the level-specific K8s API structures (leaf-level
`PodGroup` vs. composite-level `CompositePodGroup`) with a single, uniform tree definition inside
the library:

* **Hierarchy-Agnostic Library IR:** The library defines its own internal, polymorphic structures
  (`workloadbuilder.SchedulingConfig`, `workloadbuilder.SchedulingPolicy`, etc.) that represent
  scheduling configurations in a hierarchy-agnostic way.
* **Standard Mapping Helpers:** To prevent controllers from writing custom translation boilerplate
  to bridge K8s API types to the library IR, the library provides standard, built-in conversion
  functions (`MapPodGroupConfig` and `MapCompositeGroupConfig`). These helper adapters cleanly
  translate public, level-specific schemas into polymorphic IR models at runtime.

Controller authors construct a logical tree using `WorkloadNode` representing their workload
structure, populate `DefaultConfig` and the user's `UserConfig` (using the standard mapping
helpers), and invoke the builder.

The library encapsulates the following logic:
1. **Policy Resolution:** Merges default configurations with user-provided overrides (e.g.,
   resolving escape hatches uniformly across the ecosystem).
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

The library provides per-field validation helpers that accept the supported options as arguments:

```go
// In Job's API validation (pkg/apis/batch/validation):
allErrs = append(allErrs,
    workloadbuilder.ValidateSchedulingPolicy(
        spec.Scheduling.Policy, fldPath.Child("policy"),
        workloadbuilder.BasicPolicy, workloadbuilder.GangPolicy))
allErrs = append(allErrs,
    workloadbuilder.ValidateDisruptionMode(
        spec.Scheduling.DisruptionMode, fldPath.Child("disruptionMode"),
        workloadbuilder.SingleMode, workloadbuilder.AllMode))
```

This gives controllers opt-in semantics: when a new policy is introduced in a future release,
existing controllers (including `Job`) will reject it until their validation is explicitly updated
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

Until DV support is available, the library-provided validation helpers serve as a lightweight,
defensive bridge that keeps the overhead minimal for controller integrators.

#### 3. Library API Definition

```go
package workloadbuilder

import (
    "context"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    schedulingv1alpha3 "k8s.io/api/scheduling/v1alpha3"
)

// SchedulingConfig is the polymorphic, hierarchy-agnostic IR model of the PodGroup/CompositePodGroup.
type SchedulingConfig struct {
    Constraints    *SchedulingConstraints
    DisruptionMode *DisruptionMode
    Policy         *SchedulingPolicy
    ResourceClaims []ResourceClaim
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

// WorkloadNode represents a logical component of a workload (e.g., the whole JobSet,
// a specific ReplicatedJob role, or a single standalone Job).
type WorkloadNode struct {
    // Name is the logical identifier of this component (e.g., "jobset-root", "driver").
    Name string

    // DefaultConfig defines the complete set of "sane defaults" assigned by the controller
    // based on its specific orchestration domain logic.
    DefaultConfig *SchedulingConfig

    // DefaultGangMinCount provides fallback gang size if user selects Gang but leaves MinCount nil.
    DefaultGangMinCount *int32

    // UserConfig is the exact policy intent configured by the user at this specific level.
    // Can be nil if the user left the scheduling block unconfigured.
    UserConfig *SchedulingConfig

    // Children contains the logical sub-components of this workload.
    // - If len(Children) > 0, the node is inferred as a structural group
    //   (i.e., represents a CompositePodGroupTemplate).
    // - If len(Children) == 0, the node is inferred as a leaf (i.e. represents a PodGroup)
    Children []*WorkloadNode
}

// WorkloadBuilder translates the logical WorkloadNode tree into a scheduler Workload object.
type WorkloadBuilder interface {
    // Build translates the tree, merges defaults, validates policies,
    // and generates the Workload resource.
    Build(
        ctx context.Context,
        name, namespace string,
        owner *metav1.OwnerReference,
    ) (*schedulingv1alpha3.Workload, error)
}

// NewBuilder initializes a builder with a specific root node.
func NewBuilder(root *WorkloadNode) WorkloadBuilder {
    return &builderImpl{root: root}
}

// MapPodGroupConfig translates standard leaf building blocks into the library's polymorphic IR.
func MapPodGroupConfig(
    policy *schedulingv1alpha3.WorkloadPodGroupSchedulingPolicy,
    constraints *schedulingv1alpha3.WorkloadPodGroupSchedulingConstraints,
    disruption *schedulingv1alpha3.WorkloadPodGroupDisruptionMode,
    claims []schedulingv1alpha3.WorkloadPodGroupResourceClaim,
) *SchedulingConfig

// MapCompositeGroupConfig translates standard composite building blocks into the library's polymorphic IR.
func MapCompositeGroupConfig(
    policy *schedulingv1alpha3.WorkloadCompositePodGroupSchedulingPolicy,
    constraints *schedulingv1alpha3.WorkloadCompositePodGroupSchedulingConstraints,
    disruption *schedulingv1alpha3.WorkloadCompositePodGroupDisruptionMode,
) *SchedulingConfig
```

#### 4. Library Usage Example (Job)

This example demonstrates how the core `Job` controller integrates with the `workloadbuilder`
library to compile its flat `Workload` structure:

```go
import (
    "context"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    batchv1 "k8s.io/api/batch/v1"
    schedulingv1alpha3 "k8s.io/api/scheduling/v1alpha3"
    "k8s.io/utils/ptr"
)

func (r *JobReconciler) generateWorkload(
    job *batchv1.Job,
) (*schedulingv1alpha3.Workload, error) {
    // A Job's context-aware sane default is Basic scheduling (standard Kubernetes pod-by-pod)
    defaultConfig := &workloadbuilder.SchedulingConfig{
        Policy: &workloadbuilder.SchedulingPolicy{
            Basic: &workloadbuilder.BasicSchedulingPolicy{},
        },
    }

    // 2. Map the public Job.Spec.Scheduling wrapper directly using the library helper
    var userConfig *workloadbuilder.SchedulingConfig
    if job.Spec.Scheduling != nil {
        userConfig = workloadbuilder.MapPodGroupConfig(
            job.Spec.Scheduling.Policy,
            job.Spec.Scheduling.Constraints,
            job.Spec.Scheduling.DisruptionMode,
            job.Spec.Scheduling.ResourceClaims,
        )
    }

    // 3. Create the flat logical tree for Job (root node representing a single PodGroup)
    rootNode := &workloadbuilder.WorkloadNode{
        Name:                "job-root",
        DefaultConfig:       defaultConfig,
        DefaultGangMinCount: ptr.To(job.Spec.Parallelism), // Defaults to parallelism if MinCount is nil
        UserConfig:          userConfig,
    }

    // 4. Let the workloadbuilder compile and generate the Workload object
    builder := workloadbuilder.NewBuilder(rootNode)
    workloadObj, err := builder.Build(
        context.Background(),
        job.Name,
        job.Namespace,
        metav1.NewControllerRef(job, jobKind),
    )
    if err != nil {
        return nil, err
    }

    return workloadObj, nil
}
```

### Job Integration - Handling Updates and Mutability

To support the dynamic scaling of gang-scheduled workloads (Elastic Jobs) in v1.37+, the Job API
allows in-flight updates to `spec.scheduling.policy.gang.minCount`. All other scheduling
fields in `spec.scheduling` are strictly immutable upon Job creation. API
validation reuses the `workloadbuilder` library where possible so the accepted configurations stay
consistent with what the controller actually compiles.

#### Gang MinCount Defaulting & Scaling Behavior

If `minCount` is set explicitly, it has full precedence and defines the target gang size; changes 
to `spec.parallelism` are then ignored for gang-size calculations. If `minCount` is omitted 
(nil), the gang size dynamically defaults to and follows `spec.parallelism`, so changing 
`spec.parallelism` automatically changes the target gang size.

#### Reconciliation Flow upon Updates

When `spec.parallelism` (with unset `minCount`) or explicit `minCount` is changed:

1. **Detection:** The Job controller's reconcile loop detects the change and fetches the existing
   `Workload` resource from the API server.
2. **Workload Compilation:** It creates a fresh logical `WorkloadNode` tree representing the Job
   using the updated `spec.parallelism` and `minCount` value. It then passes this node to
   the `workloadbuilder` library to compile a fresh `Workload` API object.
3. **Workload Update:** The Job controller performs an API Update to apply this newly compiled
   `Workload` spec to the active resource on the API server.
4. **PodGroup Sync:** The Job controller propagates the updated size to the corresponding
   runtime `PodGroup` to ensure the scheduler immediately targets the newly scaled size.

*Note on Alternative Propagation Semantics:*
Alternative update propagation semantics are possible. For example, for composite controllers (such as
`JobSet`), scaling might not propagate down to existing `PodGroups` at all. Instead, it only applies
to newly spawned children (e.g., new `Jobs` created in-flight) via `Workload` change, while active,
running child `PodGroups` remain untouched to avoid unnecessary disruption.

#### API Validation via the workloadbuilder Library

Validation of `spec.scheduling` is layered. The integrating controller owns the *structural* and
*mutability* checks (e.g. exactly one policy is set, and every `spec.scheduling` field is immutable
except `gang.minCount`), since these are specific to its API types. The `workloadbuilder` library
owns the *semantic* checks: the api-server calls the library's `Validate` entrypoint, which runs the
same resolution and policy validation as `Build`, so the server only accepts configurations the
controller can compile into a valid `Workload`. Because this resolution reads only the incoming
object and never cluster state, it is safe to call from the api-server.

##### Allowing Mutability of spec.parallelism

The v1.36 integration hardcoded the gang size to `spec.parallelism` and therefore had to reject
`spec.parallelism` updates on gang Jobs, which disabled [Elastic Indexed
Jobs](https://kubernetes.io/docs/concepts/workloads/controllers/job/#elastic-indexed-jobs). Because
the gang size is now driven by the mutable `spec.scheduling.policy.gang.minCount` ([KEP-4671]), 
while all other `spec.scheduling` fields stay immutable. The concrete `Job` validation rules 
are specified in [KEP-5547].

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
    policy:
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
            constraints:
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
    policy:
      basic: {} # Global policy: components schedule independently
    replicatedJobPolicies:
      - targetReplicatedJob: "workers" # Policy target
        constraints:
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

Regardless of which API model `JobSet` adopts, the controller can easily map its structural spec
into the `workloadbuilder` logical tree.

The example below demonstrates the integration flow under **Option A**. Notice how the library
automatically handles hierarchical composition:

```go
import (
    "context"
    "fmt"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    jobsetv1alpha2 "k8s.io/api/jobset/v1alpha2"
    schedulingv1alpha3 "k8s.io/api/scheduling/v1alpha3"
    "k8s.io/utils/ptr"
)

func (r *JobSetReconciler) generateWorkload(
    js *jobsetv1alpha2.JobSet,
) (*schedulingv1alpha3.Workload, error) {
    // 1. Map composite JobSet-level user config to the library's IR
    var rootUserConfig *workloadbuilder.SchedulingConfig
    if js.Spec.Scheduling != nil {
        rootUserConfig = workloadbuilder.MapCompositeGroupConfig(
            js.Spec.Scheduling.Policy,
            js.Spec.Scheduling.Constraints,
            js.Spec.Scheduling.DisruptionMode,
        )
    }

    // 2. Define the Root node representing the entire JobSet (CPG Level 1).
    // The default configuration at the root is Gang scheduling, with size defaulted
    // to the count of child ReplicatedJob roles.
    rootNode := &workloadbuilder.WorkloadNode{
        Name: js.Name,
        DefaultConfig: &workloadbuilder.SchedulingConfig{
            Policy: &workloadbuilder.SchedulingPolicy{
                Gang: &workloadbuilder.GangSchedulingPolicy{},
            },
        },
        DefaultGangMinCount: ptr.To(int32(len(js.Spec.ReplicatedJobs))),
        UserConfig:          rootUserConfig,
    }

    // 3. Build the intermediate (Level 2) and leaf (Level 3) nodes representing hierarchy
    for _, rJob := range js.Spec.ReplicatedJobs {
        // Create intermediate CompositePodGroup node representing the ReplicatedJob role
        repJobNode := &workloadbuilder.WorkloadNode{
            Name: rJob.Name,
            DefaultConfig: &workloadbuilder.SchedulingConfig{
                Policy: &workloadbuilder.SchedulingPolicy{
                    Gang: &workloadbuilder.GangSchedulingPolicy{},
                },
            },
            DefaultGangMinCount: ptr.To(rJob.Replicas),
        }

        // Under each ReplicatedJob role, create leaf nodes for each Job replica instance
        for i := int32(0); i < rJob.Replicas; i++ {
            var leafUserConfig *workloadbuilder.SchedulingConfig
            if rJob.Template.Spec.Scheduling != nil {
                leafUserConfig = workloadbuilder.MapPodGroupConfig(
                    rJob.Template.Spec.Scheduling.Policy,
                    rJob.Template.Spec.Scheduling.Constraints,
                    rJob.Template.Spec.Scheduling.DisruptionMode,
                    rJob.Template.Spec.Scheduling.ResourceClaims,
                )
            }

            leafNode := &workloadbuilder.WorkloadNode{
                Name: fmt.Sprintf("%s-%s-%d", js.Name, rJob.Name, i),
                DefaultConfig: &workloadbuilder.SchedulingConfig{
                    Policy: &workloadbuilder.SchedulingPolicy{
                        Basic: &workloadbuilder.BasicSchedulingPolicy{},
                    },
                },
                // In this example, we assume a Sane default for a ReplicatedJob leaf is Gang scheduling.
                DefaultGangMinCount: ptr.To(rJob.Template.Spec.Parallelism),
                UserConfig:          leafUserConfig,
            }
            repJobNode.Children = append(repJobNode.Children, leafNode)
        }

        rootNode.Children = append(rootNode.Children, repJobNode)
    }

    // 4. Let the workloadbuilder library compile and generate the n-level Workload
    builder := workloadbuilder.NewBuilder(rootNode)
    workloadObj, err := builder.Build(
        context.Background(),
        js.Name,
        js.Namespace,
        metav1.NewControllerRef(js, jsKind),
    )
    if err != nil {
        return nil, err
    }

    return workloadObj, nil
}
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
  * **Annotation Key:** `scheduling.k8s.io/podgroup-template`
  * **Value:** The unique name of the target `PodGroupTemplate` or `CompositePodGroupTemplate`
    defined inside the parent `Workload` resource (ensuring direct mapping, as all template
    names inside a Workload are guaranteed to be unique).
* **Parent Instance Linkage Annotation:**
  * **Annotation Key:** `scheduling.k8s.io/parent-composite-podgroup`
  * **Value:** The exact resource name of the parent `CompositePodGroup` object in the same
    namespace that the child's newly created group must connect to.

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
  - `workloadbuilder` defaults `gang.minCount` from `DefaultGangMinCount` when omitted
  - `workloadbuilder` `Validate` rejects semantically invalid configurations
  - Single-level `WorkloadNode` (flat, no children) produces a leaf `PodGroup` only
  - Multi-level `WorkloadNode` tree (with children) produces a `CompositePodGroup` with correct
    parent–child structure
  - `MapPodGroupConfig` and `MapCompositeGroupConfig` correctly translate API types into the
    library IR
- Reference integration tests for multi-level controllers (e.g. `JobSet`) verify that the
  `workloadbuilder` produces the expected `CompositePodGroup` and child `PodGroup` objects from
  a composite `WorkloadNode` tree.

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
- The shared `workloadbuilder` Go translation library implemented in the Kubernetes repository.
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

### Version Skew Strategy

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: "WorkloadAwareIntegration"
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No. Enabling the feature only exposes the new opt-in `scheduling` API fields. When
the `scheduling` block is omitted, the controller defaults to `Basic` scheduling policy (pod-by-pod), 
so existing workloads behave exactly as before. Behavior changes only when a 
user explicitly sets a scheduling policy (e.g. Gang or a topology constraint).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the `WorkloadAwareIntegration` feature gate stops the controller from
honoring the new `scheduling` fields, so workloads fall back to the `Basic`
scheduling policy (pod-by-pod). The fields are ignored while the gate is off. 
However, the content of the `scheduling` fields in objects will not be cleared,
and existing `Workload` objects will not be deleted.

###### What happens if we reenable the feature if it was previously rolled back?

The feature starts working again, and the controller resumes honoring the
`scheduling` fields. Since these fields are stored in etcd while the gate is off,
objects that already had them set will have their scheduling policies applied again
on the next reconciliation.

###### Are there any tests for feature enablement/disablement?

Yes. For the newly introduced API fields, dedicated
enablement/disablement tests at the kube-apiserver registry layer will be added in
Alpha, including tests exercising the feature-gate `switch`

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

Only when a user opts in by setting the `scheduling` fields. In that case the
integrating controller (e.g. the Job controller) creates the corresponding
scheduling objects.

Workloads that omit the `scheduling` block generate no new API calls.

###### Will enabling / using this feature result in introducing new API types?

Yes,the building-block field types under `scheduling.k8s.io/v1alpha3`. No new top-level/standalone API resource kinds are
introduced by this KEP.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, but only when a user opts in. Setting the `scheduling` fields adds a small
field to the workload object and causes the controller to create the corresponding `Workload` and
`PodGroup` (or `CompositePodGroup`) objects (~500 bytes each), typically one per
workload that opts in. Workloads that omit the `scheduling` block are unaffected.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Only for workloads that opt in. For those, there may be an increase in workload sync
duration due to creating the `Workload`/`PodGroup` objects, plus a potential
increase in scheduling latency / Pod Startup SLO. This KEP itself only adds API translation in the
controller. Workloads that omit the `scheduling` block are unaffected.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Only for workloads that opt in, and the increase comes from the `Workload`/`PodGroup` objects:
  * kube-controller-manager: additional memory for `Workload`/`PodGroup` informers and
  the (negligible) CPU for translating the `scheduling` fields.
  * kube-scheduler: additional memory for `Workload`/`PodGroup` caches.
  * etcd: additional storage for the `Workload`/`PodGroup` objects.
  * kube-apiserver: additional watches for `Workload`/`PodGroup` resources, with minimal CPU impact.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature operates entirely at the control-plane/API level and does not consume any node-level resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

- 2026-06-03: KEP Created for alpha release

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
