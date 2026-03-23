# KEP-5975: Declarative API Definitions

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Strategy DSL](#strategy-dsl)
  - [Spec/Status Accessors](#specstatus-accessors)
  - [Feature-Gate Field Dropping Generator](#feature-gate-field-dropping-generator)
  - [Warning Generator](#warning-generator)
- [Examples](#examples)
  - [Adding a New API Resource](#adding-a-new-api-resource)
  - [Adding a New Field to an Existing API](#adding-a-new-field-to-an-existing-api)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
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

For example, instead of 100+ lines of strategy code, an API following best
practices might have only:

```go
var Strategy = strategy.New(strategy.Config{
	// Identify the Kind
    Object:      &widgets.Widget{},
    Scheme:      legacyscheme.Scheme,
	
	// Define the resource
    Namespaced:  true,
})
```

Which elides all the code for:

- Validation and defaulting - Handled declaratively when declarative validation/defaulting are used.
- Subresource field wiping / resetting - Highly standardized and can be implemented generically.
- Generation management - Highly standardized and can be implemented generically.
- Field dropping for new API fields guarded by a feature gate - Highly standardized and can be generated.
- Warnings - Expected to become declarative once validation-gen generates
  warning-producing code.

Much of the remaining API behavior is driven by tags on the type definition
itself rather than hand-written strategy code. For example, feature-gated fields use
`+k8s:featureGate=<GateName>` to drive both validation behavior and automatic
field-dropping code generation. The strategy consumes these generated
artifacts without any per-resource wiring.

It still remains possible to customize the strategy. For example, a resource
with custom warnings simply provides a function:

```go
var Strategy = strategy.New(strategy.Config{
    Object:     &widgets.Widget{},
    Scheme:     legacyscheme.Scheme,
    Namespaced: true,
    WarningsOnCreate: func(ctx context.Context, obj runtime.Object) []string {
        return networkPolicyWarnings(obj.(*networking.NetworkPolicy))
    },
    WarningsOnUpdate: func(ctx context.Context, obj, old runtime.Object) []string {
        return networkPolicyWarnings(obj.(*networking.NetworkPolicy))
    },
})
```

## Motivation

Today, the standard pattern is reimplemented from scratch in every API's
strategy file. A resource that follows all conventions still requires ~100
lines of method implementations identical to every other resource. The
resource-specific decisions — a handful of flags and function references —
are buried in the noise. This makes it harder to spot the non-standard hooks
that reviewers should pay attention to.

Eliminating this boilerplate reduces toil for API authors and reviewers,
and prevents copy-paste errors which are
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
lines. A detailed audit is in [ANALYSIS.md](ANALYSIS.md).

### Goals

- Eliminate the need to hand-write strategy boilerplate for resources that follow
  standard conventions while preserving the customization that is available
  today.

### Non-Goals

- A large-scale migration. This is intended to provide convenience to the
  existing API definition framework in a backward compatible way and will
  be adopted as-needed.

## Proposal

### Strategy DSL

Introduce a strategy config struct following the same pattern as `registry.Store`. This will
be introduced in a fully backward-compatible way.

Core capabilities:

- **Automatic status/spec handling.** Types with `Status` and `Spec` fields
  get automatic status clearing on create and update, generation bumping when
  spec changes, and a status substrategy.
- **Automatic validation.** Declarative validation is registered in the scheme.
  The strategy always invokes it automatically — no configuration needed.
  Hand-written validation can be provided additionally via a `Validate`
  function value; results are merged with DV results transparently.
- **Feature-gate field dropping.** A `DropDisabledFields` hook called at the
  correct lifecycle point for both create and update.
- **Behavioral overrides.** Function-valued fields for custom normalization,
  warnings, generation tracking, and status-specific behavior.

### Spec/Status Accessors

The strategy needs to copy, clear, and compare `Spec` and `Status` fields
without knowing their concrete types. Rather than relying on reflection, we would like to
introduce two interfaces:

```go
type SpecAccessor interface {
    GetSpec() any
    SetSpec(any)
}

type StatusAccessor interface {
    SpecAccessor
    GetStatus() any
    SetStatus(any)
}
```

These could be generated by either a new generator or by deepcopy-gen, which 
already walks the internal type graph and knows which types have `Spec` and `Status`
fields.

### Feature-Gate Field Dropping Generator

A `+featureGate` tag is already used in the Kubernetes APIs for documentation
purposes today. A `+k8s:featureGate=<GateName>` tag is being added as part of
declarative validation. This tag could also serve as the source of truth for
field dropping.

The field dropping code can be generated automatically as a small addition to
validation-gen. Generated field dropping code would be called from the
strategy in the same way declarative validation code is.

### Warning Generator

A small extension to validation-gen to generate warning-producing code.

In the future, validation errors and warnings may be output in a single
validation pass, but that is not a goal for this KEP.

## Examples

### Adding a New API Resource

Consider a namespaced `Widget` with Spec and Status. Full before/after
comparisons are in [BEFORE.md](BEFORE.md) and [AFTER.md](AFTER.md).

With DV and the strategy DSL, the strategy file is:

```go
var Strategy = strategy.New(strategy.Config{
    Object:      &widgets.Widget{},
    Scheme:      legacyscheme.Scheme,
    Namespaced:  true,
})
```

### Adding a New Field to an Existing API

Adding a new field to an API requires only adding the field to the type with
the requisite validation and feature gate:

```go
type WidgetSpec struct {
    // +k8s:featureGate=WidgetPriority
    // +k8s:optional
    // +k8s:minimum=0
    // +k8s:maximum=1000
    Priority *int32 `json:"priority,omitempty"`
}
```

The appropriate field dropping is generated automatically.

Declarative validation automatically handles proper validation of the field
when the feature gate is off (i.e. in-use detection and ratcheting).

## Design Details

Detailed analysis including the full `Config` struct field table,
feature-gate dropping pattern catalog, and alternatives comparison is in
[ANALYSIS.md](ANALYSIS.md). Before/after code comparisons for adding a new
API resource are in [BEFORE.md](BEFORE.md) and [AFTER.md](AFTER.md).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None. Existing per-resource tests serve as the compatibility gate.

##### Unit tests

- `k8s.io/apiserver/pkg/registry/rest/strategy/`: unit tests covering all
  Config fields, Status/Spec detection, generation bumping, field dropping
  ordering, and validation wrapping.

##### Integration tests

- [x] Subresource field wiping: https://github.com/kubernetes/kubernetes/pull/137689
- [ ] TODO: Feature gated field dropping
- [ ] TODO: Generation management

##### e2e tests

Existing e2e coverage for migrated resources validates correctness.

### Graduation Criteria

This is an internal refactoring with no user-visible behavior change.
There are no feature gates and no alpha/beta/stable transitions. Milestones
are tracked as implementation progress:

- Strategy DSL package implemented in `k8s.io/apiserver`
- At least 5 resources migrated
- All existing tests pass
- Feature-gate field dropping generator prototype for at least one resource

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

**Declarative tags** for all strategy behavior breaks down for
complex validation options, cross-field generation tracking, custom warnings,
and injected dependencies. Tags are appropriate for feature-gate field
dropping, validation, warnings, and defaulting, but for strategy
and storage definitions, there are few benefits and keeping the definitions in
Go provides type safety.

## Implementation History

- 2026-03-23: Initial KEP filed
