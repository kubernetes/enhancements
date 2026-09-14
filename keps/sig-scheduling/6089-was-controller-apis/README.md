# KEP-6089: Workload Aware Scheduling Controller APIs
<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Ecosystem Momentum and Controller Adoption](#ecosystem-momentum-and-controller-adoption)
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
    - [1. Option A: Centralized 'Targeted Policies' Model (Root-only Configuration)](#1-option-a-centralized-targeted-policies-model-root-only-configuration)
      - [Example YAML Manifest](#example-yaml-manifest)
    - [2. Option B: Template Delegation Model (Nested Configuration)](#2-option-b-template-delegation-model-nested-configuration)
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
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [x] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP proposes a standardized set of reusable API building blocks (`scheduling.k8s.io`),
integration guidelines, and shared libraries to simplify how workload controllers (e.g., `JobSet`,
`TrainJob`, `RayJob`, `LWS`, `SparkApplication`, as well as core workloads like `Job`, `Deployment`, and `StatefulSet`)
integrate with Workload-aware Scheduling (WAS).

In v1.37, this KEP and the reusable building blocks entered Alpha alongside the core `Job` integration ([KEP-5547]).
For v1.38, this KEP is promoted to **Beta**. To support embedding into GA `v1` workload APIs (`batch/v1`, `apps/v1`),
the reusable building block structs graduate directly into `scheduling.k8s.io/v1` (with Go type aliases maintained in
`scheduling.k8s.io/v1alpha3` for backward compatibility), formalizing stable primitives for the entire Kubernetes ecosystem.

By providing common API primitives (such as topology constraints and disruption policies) and a
shared library (`workloadbuilder`) to handle boilerplate resource generation, we enable controller developers to
easily expose WAS features natively within their APIs without reinventing the wheel, while
ensuring a consistent user experience across the Kubernetes ecosystem.

## Motivation

The Kubernetes ecosystem has steadily evolved its scheduling capabilities from a strictly
pod-centric model towards a more robust, workload-centric approach. This transition successfully
established foundational features in the recent v1.36 release, such as Gang Scheduling,
Topology-aware Scheduling (TAS), and Workload-aware Preemption (WAP).

However, the `Workload`, `PodGroup`, and `CompositePodGroup` resources backing these features were designed primarily as
intermediate, scheduler-facing APIs. We have not yet addressed how end-users of higher-level
workload controllers (such as `Job`, `LWS`, `JobSet`, or `RayJob`) should express their scheduling
requirements to utilize these features.

For example, in the first alpha release of [KEP-5547] (Job Integration), in v1.36 — before this KEP (KEP-6089) was established — we intentionally bypassed the user-facing
API design challenge. Instead, the integration automatically created a `PodGroup` with a hardcoded
Gang policy under specific conditions (e.g., for fully parallel static indexed Jobs). While this
unblocked initial adoption, it was fundamentally insufficient. Users have diverse use cases and
require the ability to express explicit intent—such as opting in or out of gang scheduling,
requesting specific topologies, or configuring disruption policies for their workloads.

Without this KEP, there is no standardized way for workload controllers to expose these user intents, nor
is there a standard mechanism for controllers to translate user intent into underlying scheduling
objects. If every controller authors its own user-facing API structs and custom logic to manage
scheduling objects, the ecosystem will suffer from inconsistent UX, duplicate effort, and varied
levels of WAS support.

We need a standardized toolkit that provides common scheduling API structures, handles the
boilerplate compilation, and establishes architectural guidelines to solve common integration
challenges across the ecosystem. This proposal aims to fill these gaps, providing shared tooling
and best practices while still allowing controller owners the flexibility to design their root
APIs natively.

In v1.37, this KEP (KEP-6089) addressed this gap by introducing standardized API building blocks under `scheduling.k8s.io/v1alpha3`
and the shared `workloadbuilder` library in `k8s.io/component-helpers`, validated via the core `Job` API (`batch/v1`).

### Ecosystem Momentum and Controller Adoption

Since the v1.37 alpha release, the Kubernetes ecosystem has enthusiastically embraced these building blocks
and the `workloadbuilder` library. Seven controllers spanning batch, distributed ML training, inference serving,
and core workloads have designed or implemented native integrations:

1. **JobSet** (out-of-tree): [KEP-969](https://github.com/kubernetes-sigs/jobset/blob/main/keps/969-WAS-integration/README.md)
   and implementation ([kubernetes-sigs/jobset#1250](https://github.com/kubernetes-sigs/jobset/pull/1250)) adopt the
   `workloadbuilder` library to compile hierarchical `CompositePodGroup` and `PodGroup` resources using the centralized
   targeted-policies pattern.
2. **LeaderWorkerSet (LWS)** (out-of-tree): [KEP-666](https://github.com/kubernetes-sigs/lws/pull/979) embeds the building
   blocks directly to provide gang scheduling for replica groups in multi-host AI/ML training and inference.
3. **Kubeflow Trainer (TrainJob)** (out-of-tree): [KEP-3015](https://github.com/kubeflow/trainer/pull/3219) adopts KEP-6089's
   targeted-policy model to create `Workload` blueprints and `PodGroup` objects for complex multi-node training runtimes.
4. **Spark Operator (SparkApplication)** (out-of-tree): [KEP-2962](https://github.com/kubeflow/spark-operator/pull/3154)
   embeds the leaf building blocks and vendors `workloadbuilder` from `k8s.io/component-helpers` to compile `Workload` and
   attempt-scoped `PodGroup` resources for Spark executor gangs.
5. **Core Deployment** (in-tree): [KEP-6276](https://github.com/kubernetes/enhancements/pull/6295) embeds the building
   blocks directly into `apps/v1.DeploymentSpec` to enable gang scheduling and topology constraints for long-running services.
6. **Core StatefulSet** (in-tree): [KEP-6277](https://github.com/kubernetes/enhancements/pull/6298) embeds the building
   blocks directly into `apps/v1.StatefulSetSpec` to support coordinated gang scheduling and topology placement for stateful sets.
7. **KubeRay** (out-of-tree): [ray-project/kuberay#4962](https://github.com/ray-project/kuberay/pull/4962) explores Workload
   Aware Scheduling integration for `RayJob` and `RayCluster`. Promoting the building blocks to `v1` provides a stable API
   target for aligning its batch scheduling provider with upstream WAS standards.

### Goals

- Define reusable API primitives (e.g., Scheduling Policies, Topology Constraints, Disruption
  Modes) under `scheduling.k8s.io` to be consumed by real-workload controllers.

- Provide a shared library (`workloadbuilder`) to handle the boilerplate of constructing underlying
  scheduling objects (`Workload`, `PodGroup`, or `CompositePodGroup`) from controller-specific intents.

- Establish architectural guidelines for workload controllers to expose WAS features consistently.

- Validate the building blocks and translation library against real controllers to ensure we are not designing
  in a vacuum:
  - Standard `Job` (`batch/v1`) serves as the reference in-tree implementation for single-level workloads (targeting **Beta** promotion in v1.38 in [KEP-5547], unblocked by this KEP).
  - `JobSet` serves as the reference out-of-tree composite workload ([JobSet KEP-969](https://github.com/kubernetes-sigs/jobset/blob/main/keps/969-WAS-integration/README.md)).

- Provide reference integration examples demonstrating how workload controllers can adopt WAS Controller APIs,
  covering both single-level workloads (standard `Job`) and multi-level composite workloads (`JobSet`).

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
- [KEP-6276: Workload-Aware Scheduling for Deployments](https://kep.k8s.io/6276)
- [KEP-6277: Workload API Integration with StatefulSet](https://kep.k8s.io/6277)

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
controllers autonomy, they can implement patterns native to their architecture—such as `JobSet`
using its established `targetReplicatedJobs` pattern to apply scheduling constraints to underlying
Jobs—delivering value to users immediately without waiting for the entire dependency chain to
resolve. This pattern was subsequently formalized and successfully materialized in practice in
JobSet ([KEP-969](https://github.com/kubernetes-sigs/jobset/blob/main/keps/969-WAS-integration/README.md))
and Kubeflow Trainer / TrainJob ([KEP-3015](https://github.com/kubeflow/trainer/pull/3219)).

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
    In practice, real-world data across 7 integrating controllers (Job, JobSet, LWS, TrainJob,
    Spark Operator, Deployment, StatefulSet) demonstrates that this risk did not materialize, as
    the ecosystem consistently converged on the standardized building blocks and `workloadbuilder`.

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

    In practice, composite controllers (such as `JobSet` and `TrainJob`) introduce their own
    wrapper-level fields and avoid split-brain ambiguity by enforcing validation that prevents users
    from configuring inner controllers' scheduling fields or annotations in nested child templates.

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

This section provides **reference integration examples** demonstrating how a complex,
multi-level composite controller (`JobSet`) integrates with the Workload-aware
Scheduling (WAS) building blocks and the `workloadbuilder` library.

During the alpha phase of this KEP, two architectural models were explored:
- **Option A (Recommended & Adopted):** Centralized 'Targeted Policies' Model (Root-only Configuration).
- **Option B (Alternative / Rejected):** Template Delegation Model (Nested Configuration).

Following community review, the JobSet maintainers officially approved Option A in [JobSet KEP-969: Workload-Aware Scheduling Integration](https://github.com/kubernetes-sigs/jobset/blob/main/keps/969-WAS-integration/README.md)
(implemented in [kubernetes-sigs/jobset#1250](https://github.com/kubernetes-sigs/jobset/pull/1250)). Both patterns are
documented below to contrast the architectural trade-offs, with Option A serving as the production reference.

#### 1. Option A: Centralized 'Targeted Policies' Model (Root-only Configuration)
In this model—officially adopted by JobSet in [KEP-969](https://github.com/kubernetes-sigs/jobset/blob/main/keps/969-WAS-integration/README.md)—all
scheduling configurations are declared centrally inside a single root-level `spec.scheduling` block.
The nested child templates (`replicatedJobs[*].template`) remain completely free of scheduling directives,
eliminating schema pollution, duplicate definitions, and split-brain ambiguity.

JobSet structures this centralized scheduling across three distinct levels:
1. **JobSet Level (Root):** Global policy and constraints applied across the entire JobSet (e.g. basic scheduling across components or an overarching group gang).
2. **ReplicatedJob Level:** Target-specific policies mapping to groups of replicated jobs using `targetReplicatedJobs` (e.g. gang scheduling all replicas of the worker role, or zone-level topology).
3. **Job Replica Level (`job:`):** Specific policies applying to each individual Job replica, such as per-job gang scheduling, rack-level topology constraints, disruption mode, and shared DRA `resourceClaims`.

##### Example YAML Manifest

```yaml
apiVersion: jobset.x-k8s.io/v1alpha2
kind: JobSet
metadata:
  name: distributed-training
spec:
  scheduling: # Root-level centralized scheduling
    # 1. JobSet Level: Global policy across all replicated jobs
    schedulingPolicy:
      basic: {}
    replicatedJobs:
      # 2. ReplicatedJob Level: Targeting specific ReplicatedJob roles
      - targetReplicatedJobs: [worker]
        schedulingPolicy:
          gang: {} # Gang across all worker replicas
        schedulingConstraints:
          topology:
            - level: "topology.kubernetes.io/zone"
        # 3. Job Replica Level: Per-job policies and shared DRA resource claims
        job:
          schedulingPolicy:
            gang: {}
          schedulingConstraints:
            topology:
              - level: "topology.kubernetes.io/rack" # Co-locate pods of each worker replica on same rack
          disruptionMode:
            all: {}
          resourceClaims:
            - name: shared-imex-channel
              resourceClaimTemplateName: imex-channel-template
  replicatedJobs:
    - name: driver
      replicas: 1
      template:
        spec:
          containers:
            - name: main
              image: driver-image:v1
    - name: worker
      replicas: 4
      template:
        spec:
          # Templates remain completely clean of scheduling directives
          containers:
            - name: worker
              image: worker-image:v1
```

#### 2. Option B: Template Delegation Model (Nested Configuration)

In this alternative model (which was evaluated but rejected in favor of Option A), leaf-level scheduling
policies would be declared inside nested child templates (e.g., `spec.replicatedJobs[*].template.spec.scheduling`),
while global policies (if any) would be declared at the root level (`JobSet.spec.scheduling`).

This model was rejected because:
1. It requires users to drill down into deeply nested child templates to define workload-level scheduling policies,
   fragmenting policy definition across multiple levels of the specification instead of presenting a clean, unified
   policy declaration at the top level.
2. It tightly couples the parent controller's scheduling configuration to child template schemas.
3. In multi-tier hierarchies (such as `TrainJob -> JobSet -> Job`), passing scheduling configuration down through
   multiple nested templates becomes cumbersome and fragile.

In contrast, Option A keeps all scheduling declarations centralized at the root level (`JobSet.spec.scheduling`)
while keeping child templates completely clean.

---

#### 3. Controller Integration and workloadbuilder Mapping Go Code

Under Option A, the `JobSet` controller maps its centralized scheduling policies and structural spec
into a `workloadbuilder.WorkloadItem` tree. In the initial alpha implementation (MVP), JobSet compiles
the `Workload` resource and runtime `PodGroup` objects. When multi-level composite scheduling
(`CompositePodGroup`) is enabled, the controller structures the root as a composite node whose `Children`
are the `ReplicatedJob` roles, compiling a `Workload` with a `CompositePodGroupTemplate` over the child
`PodGroupTemplate`s:

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
* **Centralized Management:** The parent controller compiles the `Workload` and is also fully
  responsible for directly creating and managing all runtime `CompositePodGroup` and `PodGroup` objects.
* **Delegated Management:** The root controller compiles the `Workload` blueprint (and potentially its top-level
  `CompositePodGroup`), but delegates the creation and management of individual runtime `PodGroup` objects to
  intermediate or child execution controllers via downward annotations.

**Architectural Guidance & Applicability:**

It is crucial to distinguish between the depth of the **scheduling hierarchy** (the number
of nested `CompositePodGroup` and `PodGroup` levels) and the depth of the **controller
actuation chain** (whether a single controller directly creates child `Job` resources or
acts through intermediate controllers):

- **Centralized Management (Direct Controller Orchestration):**
  When a controller directly creates and manages child workload resources (such as `JobSet`
  directly orchestrating core `Job` resources), centralized management is natural and
  effective, regardless of how many levels exist in the scheduling hierarchy. For example,
  even though `JobSet` models a 3-level scheduling hierarchy (`JobSet` -> `ReplicatedJob` ->
  `Job`), the single `JobSet` controller directly stamps out child `Job` specs. It can
  therefore directly instantiate all runtime `CompositePodGroup` and `PodGroup` objects and
  set `job.spec.template.spec.schedulingGroup.podGroupName` pointing to its own created groups
  without coordination overhead or race conditions.

- **Delegated Management (Nested / Chained Controllers):**
  Delegated management is necessary when there is a chain of intermediate controllers across
  component boundaries (such as Kubeflow Trainer / `TrainJob` composing a `JobSet`, which in
  turn creates standard `Jobs`). `TrainJob` only constructs the top-level `JobSet` custom
  resource; it does not construct `Job` objects directly.
  
  Crucially, `TrainJob` has no mechanism to inject individual `PodGroup` references into the
  pod templates of the Jobs generated by `JobSet` across the component boundary (unlike
  `JobSet`, which directly constructs child `Job` objects and can set their pod template
  fields). Attempting centralized management across intermediate controllers would cause a
  severe abstraction leak and break controller encapsulation.

  Therefore, a wrapper controller like `TrainJob` cannot directly bind pods to centrally
  created leaf `PodGroup`s. Instead, it must **delegate** runtime group creation downward:
  `TrainJob` compiles the `Workload` blueprint (and potentially creates its top-level
  `CompositePodGroup`), and injects well-known downward annotations
  (`scheduling.k8s.io/group-template-name` and `scheduling.k8s.io/parent-compositepodgroup`)
  into the `JobSet` metadata. The intermediate `JobSet` controller then reads these
  annotations, materializes the runtime `PodGroup`s from the referenced template in the
  parent `Workload`, attaches them to the parent `CompositePodGroup`, and injects the resulting
  `PodGroup` names into the child `Job` pod templates.

Both patterns are first-class and fully supported by the `workloadbuilder` library and
conventions. Formal rules and detailed recommendations on when to apply each pattern will be
finalized for GA based on production feedback from ecosystem adopters.

#### 2. Downward Template and Parent Mapping via Well-Known Annotations

If a composite controller delegates runtime `PodGroup` management to an intermediate or child controller (such as
in the `TrainJob -> JobSet -> Job` multi-tier pattern where the parent cannot inject pod-level scheduling references
across the intermediary abstraction boundary), we must solve a crucial coordination problem. The downstream
controller needs two distinct pieces of information to construct and place its runtime scheduling objects correctly:

1. **Template Mapping:** Which `PodGroupTemplate` or `CompositePodGroupTemplate` inside the parent's
   compiled `Workload` corresponds to this child's pods (enabling the downstream controller to materialize
   or compile the correct policy and constraints).
2. **Parent Instance Linkage:** Which specific runtime `CompositePodGroup` instance name in the
   namespace this newly created group must attach to (under its `spec.parentRef`). This linkage is
   especially critical in multi-instantiated environments (such as `LeaderWorkerSet` / LWS), where a
   composite controller may instantiate multiple separate `CompositePodGroup` objects from the exact
   same template (one per replica).

##### The Solution: Downward Mapping Annotations

To resolve this template and hierarchy mapping without structural API schema changes, orchestrators
operating in delegated mode propagate these linkages downwards by injecting two well-known metadata annotations
directly into the created child objects (for example, `TrainJob` sets these annotations on the `JobSet` objects it creates):

* **Template Linkage Annotation:**
  * **Annotation Key:** `scheduling.k8s.io/group-template-name`
  * **Value:** The unique name of the target `PodGroupTemplate` or `CompositePodGroupTemplate`
    defined inside the parent `Workload` resource (ensuring direct mapping, as all template
    names inside a Workload are guaranteed to be unique). For example, in a
    `TrainJob -> JobSet -> Job` hierarchy, this tells `JobSet` which template from the root
    `Workload` to use to materialize its `PodGroup`s and configure its child `Job`s.
* **Parent Instance Linkage Annotation:**
  * **Annotation Key:** `scheduling.k8s.io/parent-compositepodgroup`
  * **Value:** The exact resource name of the parent `CompositePodGroup` object in the same
    namespace that the downstream controller's newly created groups must attach to (via `spec.parentRef`). This is required
    only in the delegated lifecycle model (the parent creates the `Workload` and `CompositePodGroup`, but runtime
    `PodGroup` creation is delegated); when the parent centrally manages both the `CompositePodGroup`
    and the `PodGroup`s (as `JobSet` does for core `Job`s), this annotation is unnecessary.

We strictly use **unstructured metadata annotations** rather than introducing new structural fields
in the child's API schemas for this coordination. These mappings are transient, internal, and
automatically managed by composite operators during compilation, not user-configurable scheduling
intents.

### Go Package Placement & Graduation Strategy

Embedding reusable building block Go structures (defined in a pre-stable package like
`scheduling.k8s.io/v1alpha3`) directly into a stable GA type (like `batch/v1.JobSpec` or `apps/v1.DeploymentSpec`)
during its Alpha phase introduces package dependency and graduation challenges.

In the Go language, changing the import path of an embedded field inside a GA struct constitutes a
breaking change in client libraries. Furthermore, as highlighted during API reviews,
when building blocks are embedded into GA `v1` resources, promoting the embedding feature gate to Beta
(enabled by default) commits wire format compatibility at the `v1` resource level. In Kubernetes API
conventions, there is no semantic difference between Beta and GA for embedded structs in `v1` resources.

To solve this graduation compatibility trap without forcing identical structure duplication across different
apiGroups, we adopt the following transition pattern:

* **Alpha Phase (v1.37):** The shared building blocks were defined in the pre-stable
  `scheduling.k8s.io/v1alpha3` package. Standard Kubernetes import rules allowed stable GA
  groups (`batch/v1`) to import pre-stable packages as long as the field itself remained gated in
  Alpha.
* **Graduation to Beta (v1.38) via Direct-to-v1 Promotion:** When the composed fields are promoted to Beta
  (enabled by default in `v1` types like `batch/v1.JobSpec` under [KEP-5547]), we bypass the intermediate
  `v1beta1` package version entirely (since wire-format compatibility is already committed at the `v1` resource level).
  We graduate the building block structs straight into the stable `scheduling.k8s.io/v1` package and update the
  fields inside `batch/v1.JobSpec`, `apps/v1.DeploymentSpec`, and `apps/v1.StatefulSetSpec` to reference the `v1` types.
* **Go Type Aliasing for Backward Compatibility:** To prevent breaking third-party Go controllers that
  still import the older alpha package (`k8s.io/api/scheduling/v1alpha3`), we replace the physical structures in
  `v1alpha3` with **Go Type Aliases (`=`)** pointing to the new stable `v1` types:
  ```go
  // In k8s.io/api/scheduling/v1alpha3/types.go:
  type WorkloadPodGroupSchedulingConstraints = schedulingv1.WorkloadPodGroupSchedulingConstraints
  type WorkloadPodGroupDisruptionMode = schedulingv1.WorkloadPodGroupDisruptionMode
  type WorkloadPodGroupSchedulingPolicy = schedulingv1.WorkloadPodGroupSchedulingPolicy
  type WorkloadPodGroupResourceClaim = schedulingv1.WorkloadPodGroupResourceClaim
  // Analogously for composite WorkloadCompositePodGroup* types
  ```
  This is a well-established, approved Kubernetes API pattern (previously used in the `admissionregistration`
  API group) that allows external codebases to compile seamlessly while gradually transitioning their imports.

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
  `SchedulingMode`, `ResourceClaim`) introduced under the `scheduling.k8s.io` API group (`v1alpha3`).
- The shared `workloadbuilder` Go translation library implemented in the `k8s.io/component-helpers`
  staging repository (`k8s.io/component-helpers/scheduling/schedulingv1/workloadbuilder`).
- Comprehensive unit and integration tests added for the `workloadbuilder` library to verify
  correct resource translation and default-overriding logic.
- Core `Job` API (`batch/v1`) integrated with the standardized WAS building blocks and validated in
  the alpha phase ([KEP-5547]).

#### Beta

- Reusable building blocks graduated directly into `scheduling.k8s.io/v1`, establishing wire-format
  stability for embedding into `v1` workload APIs (`batch/v1`, `apps/v1`), with Go type aliases in
  `scheduling.k8s.io/v1alpha3` preserving backward compatibility.
- Widespread ecosystem adoption validated: 7 controllers across the ecosystem
  (JobSet, LeaderWorkerSet, Kubeflow TrainJob, Spark Operator, Core Deployment, Core StatefulSet,
  and lessons learned from KubeRay) are actively designing and implementing integrations based
  on these building blocks and `workloadbuilder`, providing strong confirmation that the
  building-block schemas and library abstractions are sound and ready for Beta.
- User and ecosystem feedback gathered on usability, confirming that the composed configuration
  and `workloadbuilder` approach provides a natural and cohesive UX.

#### GA

- Architectural and lifecycle recommendations for runtime `PodGroup` and `CompositePodGroup`
  objects (formalizing rules for Centralized vs Delegated Management) finalized based on
  production feedback from ecosystem adopters.
- The `workloadbuilder` library in `k8s.io/component-helpers` matured and expanded based on use
  cases emerging from active integrations.
- At least two out-of-tree controllers (e.g., JobSet, LeaderWorkerSet, TrainJob, or
  SparkApplication) running in production clusters with the building blocks and
  `workloadbuilder`, with positive operator feedback.

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

###### How can a rollout or rollback fail? Can it impact already running workloads?

This KEP provides reusable Go structs (`scheduling.k8s.io`) and a shared compilation library
(`workloadbuilder`). These building blocks are **not top-level API resources** served directly by
`kube-apiserver`. Instead, they are embedded into the specifications of integrating workloads
(such as `batch/v1.Job`).

- **Rollout / Beta Promotion:**
  - When integrating workloads (such as `Job` in [KEP-5547]) graduate their scheduling
    integrations and migrate their embedded fields from `scheduling.k8s.io/v1alpha3` to
    `scheduling.k8s.io/v1`, the transition is completely seamless and safe because of Go type
    aliases (`type Foo = schedulingv1.Foo`).
  - Go type aliases preserve identical Go type identity, in-memory representation, and
    JSON/protobuf serialization tags across the packages. As a result, there is zero wire-format
    change, ensuring complete backward compatibility and zero impact on running workloads or
    existing client payloads.
  - Controllers vendoring `workloadbuilder` consume it as a compile-time Go library dependency;
    translation and compilation logic is validated via extensive unit and integration tests.
  - Running workloads (Pods, Jobs, etc.) are **not impacted** by rollout. Running Pods remain
    scheduled and bound to their assigned nodes. Existing runtime `Workload` and `PodGroup`
    objects in etcd remain active and untouched.

- **Rollback:**
  - Rollback is managed strictly at the level of each integrating controller's feature gate
    (e.g., `WorkloadWithJob` in [KEP-5547]). Disabling an integration's gate stops that controller
    from accepting or compiling scheduling fields for new workloads.
  - Existing running workloads continue running without disruption. Existing `Workload` and
    `PodGroup` objects in etcd remain intact and are cleaned up according to standard garbage
    collection when their parent workload is deleted.

###### What specific metrics should inform a rollback?

As this KEP provides Go structs and a client library rather than an active standalone controller
or top-level API endpoint, there are no KEP-specific metrics for rollback.

Rollback criteria, alert thresholds, and operational metrics are defined by the respective
integrating controller KEPs (e.g., `job_sync_duration_seconds` and `job_sync_total{result="error"}`
in [KEP-5547] for Job integration, and corresponding metrics in [KEP-6276] for Deployment).

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Upgrade and downgrade testing for runtime controller behavior is owned and conducted by individual
controller integration KEPs (e.g. [KEP-5547] for Job).

For this KEP, testing verifies:
- Go type alias compatibility between `v1alpha3` and `v1` (serialization and deserialization
  equivalence).
- Comprehensive unit and integration test coverage of the `workloadbuilder` compilation library
  across all supported scheduling configurations.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. The graduation of building blocks to `scheduling.k8s.io/v1` preserves
`scheduling.k8s.io/v1alpha3` via Go type aliases. No fields, APIs, or flags are deprecated or
removed.

### Monitoring Requirements

As this KEP provides reusable Go structs and a client compilation library rather than an active
standalone controller or top-level API resource, there are no direct runtime metrics or
monitoring requirements for the building blocks themselves. Monitoring, observability, SLOs, and
SLIs are owned and defined by each integrating controller's KEP (e.g., [KEP-5547] for Job).

###### How can an operator determine if the feature is in use by workloads?

N/A directly for the building blocks. Because this KEP provides reusable Go structs and a
compilation helper library rather than a standalone controller, detecting whether the feature is
in use is done by inspecting workloads managed by integrating controllers (e.g., checking for
`.spec.scheduling` on `Job` or `JobSet` objects) or checking for generated runtime objects
(`Workload`, `PodGroup`, `CompositePodGroup`). Detailed procedures and query commands belong to
each controller's integration KEP (e.g., [KEP-5547] for Job).

###### How can someone using this feature know that it is working for their instance?

N/A directly for the building blocks. As building blocks and a client library, there are no runtime
instances or standalone components to monitor directly. Runtime events (e.g., `WorkloadCreated`,
`PodGroupCreated`), status conditions, and pod-level references (such as
`.spec.schedulingGroup.podGroupName`) are emitted and managed by the integrating controllers and
kube-scheduler. These signals are detailed in the respective controller KEPs (e.g., [KEP-5547] for
Job).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A for the building blocks and compilation library. In-process compilation latency of
`workloadbuilder` is sub-millisecond (< 1 ms per workload compilation). System-level and
controller-level SLOs (such as controller sync duration or scheduler binding latency) belong to and
are measured under the respective integrating controller KEPs (such as [KEP-5547]).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A for this KEP. Because this KEP provides Go structs and a client library rather than an active
controller or standalone service, operational SLIs and health indicators belong to the integrating
controllers and kube-scheduler, and are specified in their dedicated KEPs (e.g., [KEP-5547] for
Job, [KEP-6276] for Deployment).

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

N/A. Observability metrics tracking workload adoption across controllers belong to each respective
integration KEP.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

None directly. This KEP provides Go structs and a compilation library. Integrating controllers
depend on standard core Kubernetes components (`kube-apiserver`, `kube-controller-manager`, and
`kube-scheduler` with WAS plugins enabled), as detailed in their respective KEPs.

### Scalability

###### Will enabling / using this feature result in any new API calls?

The API building blocks and `workloadbuilder` library execute in-process and do not make any API
calls directly. API calls to create or manage `Workload` and `PodGroup` objects are made by
integrating controllers when users opt in, and are quantified in each controller's KEP (e.g.,
[KEP-5547]).

###### Will enabling / using this feature result in introducing new API types?

Yes, but they are not top-level API types. This KEP introduces reusable building-block Go field
types under `scheduling.k8s.io/v1` (with Go type aliases in `scheduling.k8s.io/v1alpha3`) that are
embedded directly into the specifications of integrating workload APIs (such as `batch/v1.Job`).

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

This KEP only defines optional embedded building-block fields. Object count and size increases
(such as runtime `Workload` and `PodGroup` objects) are created by integrating controllers and
quantified in their respective KEPs.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

This KEP adds only in-process Go translation (negligible CPU, < 1 ms) in controllers vendoring the
library. Any operational latency impact is governed by the integrating controllers and
kube-scheduler, as evaluated in their KEPs.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The building blocks and library execute in-process with negligible resource overhead.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. This feature operates entirely at the API and library compilation level and does not consume
any node-level resources.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The building blocks and `workloadbuilder` library execute strictly in-process within workload
controllers and carry no network I/O or state of their own. Controller failure behavior during API
server unavailability is governed by the host controller's reconciliation loop, as detailed in the
corresponding integration KEPs.

###### What are other known failure modes?

Failure modes related to scheduling execution (e.g., unfulfillable gang scheduling requirements,
missing downward template annotations) are handled and diagnosed within the integrating controllers
and kube-scheduler, as described in their respective KEPs.

###### What steps should be taken if SLOs are not being met to determine the problem?

Refer to the troubleshooting guide of the relevant integrating controller KEP (e.g., [KEP-5547]
for Job integration) to diagnose controller reconciliation, admission, or scheduling issues.

## Implementation History

- 2026-06-03: KEP Created for alpha release
- 2026-07-15: Synced the KEP with the implementation of the `workloadbuilder` library, the Job
  building-block fields and the `workloadbuilder` API (`Validate` with a
  `ValidationInput` and the `NewBuilderFromExistingWorkload` constructor).
- 2026-07-18: Synced the `CompositePodGroup` design with the codebase and review feedback from the
  building-blocks library.
- 2026-09-10: Promoted KEP to Beta for v1.38 milestone. Reflected adoption by 7 controllers across
  the ecosystem (JobSet, LWS, TrainJob, Spark Operator, Deployment, StatefulSet). Promoted reusable
  building blocks directly to `scheduling.k8s.io/v1` with `v1alpha3` Go type aliases per API review guidance.
  Completed Beta PRR questionnaire.

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
