# KEP-5975: Declarative API Definitions

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
  - [Background](#background)
- [Proposal](#proposal)
  - [Enforcement of Best Practices](#enforcement-of-best-practices)
  - [Code generation](#code-generation)
  - [API Declarations](#api-declarations)
  - [Validation](#validation)
  - [Warnings](#warnings)
  - [Field Wiping and Fields Resetting](#field-wiping-and-fields-resetting)
  - [Generation Management](#generation-management)
  - [Feature-Gate Field Dropping](#feature-gate-field-dropping)
- [Examples](#examples)
  - [Adding a New API Resource](#adding-a-new-api-resource)
  - [Adding a New Field to an Existing API](#adding-a-new-field-to-an-existing-api)
- [Design Details](#design-details)
  - [registry.RESTConfig](#registryrestconfig)
  - [strategy.Config](#strategyconfig)
    - [Spec/Status Accessor Interfaces](#specstatus-accessor-interfaces)
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
- [Alternatives](#alternatives)
- [Future Work](#future-work)
- [Implementation History](#implementation-history)
<!-- /toc -->

**Note:** This KEP tracks an internal code organization change that is not
visible to end users. There are no feature gates and no alpha/beta/stable
transitions. We are using the KEP process to encourage discussion and
community involvement.

## Summary

Each API's directory should only contain code that makes it different from
the standard pattern. Today, each contains hundreds of lines of boilerplate
that obscure those differences.

It should be impossible for an API author to accidentally violate standard patterns.
Deliberate violations should require an exception from an API reviewer.

Rather than registering a multitude of code generators in the `doc.go` for each API
definition package, a developer should simply provide required information in a single
declaration, e.g.:

```yaml
apiVersion: apidefinitions.k8s.io/v1alpha1
kind: APIVersion
metdata:
  name: admission.k8s.io/v1
spec:
  modelPackage: io.k8s.admission
```

Additionally, it should be easy to adhere to the standard patterns when in code. A
simple resource should need nothing more than a trivial storage registration:

```go
func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST, error) {
    return registry.NewREST(registry.RESTConfig{
        Resource: widgets.Resource("widgets"),
        Kind:     "Widget",
    }, optsGetter)
}
```

Which could also be generated in the future.

All code generation, validation/warning wiring, field wiping/resetting, generation management, field dropping 
of feature gated fields, and so on, should happen correctly by default. These behaviors should be
driven by information already available in the API definition (types.go files) such as declarative
validation and feature gate tags.

## Motivation

Today, the standard pattern is reimplemented from scratch in every API's
strategy file. A resource that follows all conventions still requires ~100
lines of method implementations identical to every other resource. The most
important resource-specific decisions are buried in the noise. This makes it
harder to spot the non-standard hooks that reviewers should pay attention to
and has resulted in mistakes which are
[challenging and risky to fix](https://github.com/kubernetes/kubernetes/pull/137715).

This is a natural follow-up to declarative validation
([KEP-4153](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/4153-declarative-validation),
[KEP-5073](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/5073-declarative-validation-with-validation-gen))
and declarative defaulting
([KEP-1929](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/1929-built-in-default)),
which have done much of the heavy lifting and can be further leveraged here.

Every strategy file in `pkg/registry/` implements the same ~12 methods:
status clearing, generation bumping, validation delegation, and feature-gate
field dropping. These follow rigid patterns across 67 files totaling ~12,000
lines.

### Goals

- Make it impossible to author APIs that violate best practices without
  being granted an exception by an API reviewer.
- Eliminate the need to hand-write strategy boilerplate for resources that
  follow standard conventions while preserving the customization that is
  available today.

### Non-Goals

- A large-scale migration. This is intended to provide convenience to the
  existing API definition framework in a backward-compatible way and will
  be adopted as needed.

### Background

This is not the first time we've done this. Back in 2016-2017 @enj did a major sweep:

- https://github.com/kubernetes/kubernetes/pull/37770
- https://github.com/kubernetes/kubernetes/pull/44779
- https://github.com/kubernetes/kubernetes/pull/46390

(thanks @liggitt for the PR links!)

## Proposal

### Enforcement of Best Practices

https://github.com/kubernetes/kubernetes/pull/137689 demonstrates using a
cross-cutting blackbox test to ensure that all APIs adhere to field wiping
conventions. We will expand approach with testing, linting, and exception
lists to cover:

- ResetObjectMetaForStatus was used to reset metadata
  - https://github.com/kubernetes/kubernetes/pull/137689 tests metadata wiping but does not ensure that ResetObjectMetaForStatus was used
- Generation management (set to 1 on create and monotomically increased when spec is changed by an update)
- Field dropping of feature gated fields
  - It should be impossible to add a new field to an existing API without a feature gate 
  - Iff a field is feature gated, field dropping must be implemented
  - Field dropping must be tested (can we offer any test conveniences/automation?)
- Default behaviors:
  - `AllowCreateOnUpdate` is `false`
  - `AllowUnconditionalUpdate` is `true`
  - `DefaultGarbageCollectionPolicy` is `DeleteDependents`
- Enablement of generated support code (DeepCopy, Declarative Validation, etc.)

It should be possible to make exceptions, but exceptions should be tracked in a file
owned exclusively by API approvers.

We don't want to have to remember to add the safety nets when adding new groups and new versions.
So we will structure them such that they're automatically added and automatically enforced.

### Code generation

Rather than independently enable individual code generators with tags, code generation will
be on-by-default for all API definitions in code where declaration files exist.

`GroupVersion` will be placed in external API definition directories, for example:

`staging/src/k8s.io/api/admission/v1/apiversion.yaml`:

```yaml
apiVersion: apidefinitions.k8s.io/v1alpha1
kind: APIVersion
metdata:
  name: admission.k8s.io/v1
spec:
  modelPackage: io.k8s.admission
```

This file defines all common properties of a group and it's presence indicates that
all external code generation, such as typed clients and openapi, should be run for this
group of resources.

A similar file will be used to define the API group in the unexported API definition directory:

`pkg/apis/admission/apigroup.yaml`:

```yaml
apiVersion: apidefinitions.k8s.io/v1alpha1
kind: APIGroup
metdata:
  name: admission.k8s.io
spec:
  # ...
```

Our long term goal is for all declarative information about an API lives either in `types.go` files
or these files and that the below "API Declaration" improvements lead to generation of all intermediate
artifacts based on these data sources.

### API Declarations

Build on existing `registry.Store` and strategy interfaces in a
backward-compatible way by introducing a "configuration" layer
that declares a desired resource definition at a high level while still
providing a level of customization and extensibility.

Custom behavior can be added as needed:

```go
func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST, error) {
    return registry.NewREST(registry.RESTConfig{
        Resource: widgets.Resource("widgets"),
        Kind:     "Widget",
        StrategyConfig: strategy.Config[*widgets.Widget]{
            WarningsOnCreate: func(ctx context.Context, obj *widgets.Widget) []string {
                return widgetWarnings(obj)
            },
        },
    }, optsGetter)
}
```

Entirely handwritten strategies can also be used:

```go
type podStrategy struct {}

func (s podStrategy) CheckGracefulDelete(
    ctx context.Context, obj runtime.Object, opts *metav1.DeleteOptions) bool {
    // Pod-specific logic
}

func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST, error) {
    return registry.NewREST(registry.RESTConfig{
        Resource: core.Resource("pods"),
        Kind:     "Pod",
        DeleteStrategy: &podStrategy{},
    }, optsGetter)
}
```

The rest of this proposal focuses on what we can offer to minimize the
amount of custom configuration needed.

### Validation

Declarative validation will be called automatically if available. If
handwritten validation is provided, all handwritten *and* declarative
validation code will be called and mismatch checking will be run.

For example,

```go
func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST, error) {
    return registry.NewREST(registry.RESTConfig{
        Resource: widgets.Resource("widgets"),
        Kind:     "Widget",
        StrategyConfig: strategy.Config[*widgets.Widget]{
            Validate: func(ctx context.Context, obj *widgets.Widget) field.ErrorList {
                // Call handwritten validation here (Declarative Validation is called and mismatch-checked automatically)
            },
        },
    }, optsGetter)
}
```

### Warnings

A small extension to validation-gen to generate warning-producing code.
In the future, validation errors and warnings may be output in a single
validation pass, but that is not a goal for this KEP.

### Field Wiping and Fields Resetting

Default behavior will follow best practices:

- **Main strategy**: clears status on create (`obj.Status = TypeStatus{}`),
  and clears status changes on update (`new.Status = old.Status`).
- **Status substrategy**: clears spec, labels, and annotations changes
  (`new.Spec = old.Spec`, `metav1.ResetObjectMetaForStatus`, etc.).

Managed fields are reset to match the fields that are wiped.

This requires [Spec/Status accessor interfaces](#specstatus-accessor-interfaces).

### Generation Management

Default behavior will follow best practice: Generation is set to 1 on create
and bumped on update when Spec changes.

This requires [Spec/Status accessor interfaces](#specstatus-accessor-interfaces).

### Feature-Gate Field Dropping

The `+k8s:featureGate=<GateName>` tag being added as part of declarative
validation can also serve as the source of truth for field dropping. The
field-dropping code will be generated by validation-gen. Generated
functions are consumed via the config's `DropDisabledFields` hook or
by the default strategies automatically.

## Examples

### Adding a New API Resource

For a resource following *all* best practices:

```go
func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST, error) {
    return registry.NewREST(registry.RESTConfig{
        Resource: widgets.Resource("widgets"),
        Kind:     "Widget",
    }, optsGetter)
}
```

### Adding a New Field to an Existing API

Adding a new field to an API requires a feature gate:

```go
type WidgetSpec struct {
    // +k8s:featureGate=WidgetPriority
    // +k8s:optional
    // +k8s:minimum=0
    // +k8s:maximum=1000
    Priority *int32 `json:"priority,omitempty"`
}
```

The appropriate field dropping is generated automatically when the feature gate is provided.

Declarative validation also automatically handles proper validation of the field
when the feature gate is off (i.e. in-use detection and ratcheting).

## Design Details

### registry.RESTConfig

Provides resource identity, naming, and optional customization of name generation, handwritten validation,
handwritten warnings, selectable fields, and printer columns.

**Required:**

| Field | Type | Purpose |
|-------|------|---------|
| `Resource` | `schema.GroupResource` | Resource identity (group + plural name), matching the existing `DefaultQualifiedResource` pattern |
| `Kind` | `string` | Kind name; the internal GVK is derived as `Resource.Group/__internal/Kind` |

**Optional properties:**

| Field | Default | Purpose |
|-------|---------|---------|
| `NameGenerator` | `SimpleNameGenerator` | Name generation for `generateName` |
| `TableConvertor` | default | Custom printer columns |
| `SelectableFields` | metadata only | Custom field selectors beyond metadata |

**Strategy overrides:**

| Field | Default | Purpose |
|-------|---------|---------|
| `StrategyConfig` | nil | Typed hooks for custom strategy behavior (see [strategy.Config](#strategyconfig)) |

When `StrategyConfig` is set, the provided configuration customizes strategy.

### strategy.Config

`strategy.Config` provides fields and hooks for custom strategy behavior.

**Hooks:**

| Hook | Purpose |
|------|---------|
| `Validate` / `ValidateUpdate` | Additional hand-written validation (merged with DV) |
| `WarningsOnCreate` / `WarningsOnUpdate` | Custom warning messages |

**Status substrategy:**

When the type has a `Status` field, a status substrategy is created
automatically. If it needs customization, a nested `Status *StatusConfig[T]`
provides hooks.

#### Spec/Status Accessor Interfaces

Our implementation needs to copy, clear, and compare `Spec` and
`Status` fields. To support this, we propose generating accessor interfaces
that provide type-safe access without reflection:

```go
type SpecAccessor[S any] interface {
    GetSpec() S
    SetSpec(S)
}

type StatusAccessor[T any] interface {
    GetStatus() T
    SetStatus(T)
}

type GenerationAccessor interface {
	GetGeneration() int64
	SetGeneratioin(value int64)
}
```

Implementations are trivial one-liners, and can be generated by a deepcopy-gen style generator (which
already walks the type graph).

### Test Plan

[x] I/we understand the owners of the involved components may require updates
to existing tests to make this code solid enough prior to committing the
changes necessary to implement this enhancement.

##### Prerequisite testing updates

None. Existing per-resource tests serve as the compatibility gate.

##### Unit tests

- Unit tests for all new framework types and workflows that are introduced.

##### Integration tests

- All above "Enforcement of Best Practices" tests are implemented

##### e2e tests

Not applicable.

### Graduation Criteria

This is an internal refactoring with no user-visible behavior change.
There are no feature gates and no alpha/beta/stable transitions.
Once the new way of defining strategies is implemented and is in
use by at least five APIs, we will mark this KEP as implemented.

### Upgrade / Downgrade Strategy

Not applicable. Internal refactoring only.

### Version Skew Strategy

Not applicable. Entirely within the kube-apiserver binary.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - Describe the mechanism: Internal refactoring, always enabled. No feature
    gate — no behavioral change.
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? No.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Not applicable — code refactoring, not a runtime feature.

###### What happens if we reenable the feature if it was previously rolled back?

Not applicable.

###### Are there any tests for feature enablement/disablement?

Not applicable — no feature gate.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Not applicable.

###### What specific metrics should inform a rollback?

Not applicable.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not applicable.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Not applicable — internal refactoring only.

###### How can someone using this feature know that it is working for their instance?

Not applicable.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Not applicable.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Not applicable.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Not applicable.

###### What are other known failure modes?

None.

###### What steps should be taken if SLOs are not being met to determine the problem?

Not applicable.

## Alternatives

**Declarative tags for all strategy behavior.** Tags break down for complex
validation options, cross-field generation tracking, custom warnings, and
injected dependencies. Tags are appropriate for field dropping, validation,
warnings, and defaulting, but for strategy and storage definitions, keeping
the definitions in Go provides type safety.

## Future Work

- **REST install registration.** The `StorageProvider` wiring (~4,200 lines)
  that maps resources to their REST storage follows a repeating pattern per
  API group. With `registry.NewREST`, much of this could be simplified.
- **etcd storage test data.** `test/integration/etcd/data.go` (~1,000 lines)
  contains a per-resource entry with a JSON stub, expected etcd path, and
  introduced version. This data could potentially be derived from the resource
  identity and scheme.
- Selectable fields will be tagged with `+k8s:selectableField` on the type
  definition. Generated functions would bewired into the Store automatically.
  This begs the question: *Should* selectable fields be codified into types.go?
- For 1:1 field-to-column mapping, fields could be tagged with
  `+k8s:printerColumn`. A generated `TableConvertor` is used by default.
  Computed columns (derived from multiple fields) override `TableConvertor`
  via `RESTConfig`.
  This also begs the question: *Should* printer columns be codified into types.go?

## Implementation History

- 2026-03-23: Initial KEP filed
