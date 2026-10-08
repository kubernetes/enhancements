# KEP-5259: DRA Variables in CEL

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [CEL Environment and Semantics](#cel-environment-and-semantics)
  - [Validation](#validation)
  - [Compilation and Runtime Evaluation](#compilation-and-runtime-evaluation)
  - [Cost Limits](#cost-limits)
  - [Feature Gate](#feature-gate)
  - [Test Plan](#test-plan)
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
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
  - Tracking issue: [kubernetes/enhancements#5259](https://github.com/kubernetes/enhancements/issues/5259)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Add named CEL variables to `CELDeviceSelector` and `DerivedAttribute`, mirroring the variable composition model already used in
ValidatingAdmissionPolicy (VAP) and MutatingAdmissionPolicy (MAP). Variables let authors define reusable
sub-expressions once and reference them from the main `expression` as `variables.<name>`, improving
readability of complex device selection and derived-attribute logic in `DeviceClass`, `ResourceClaim`, and `ResourceClaimTemplate`.

This KEP covers **CEL device selectors** and **derived attribute CEL expressions** ([KEP-6080](https://github.com/kubernetes/enhancements/tree/master/keps/sig-scheduling/6080-dra-derived-attributes)). It is tracked separately from the broader
[DRA CEL usability improvements](https://github.com/kubernetes/kubernetes/issues/125826) umbrella.

## Motivation

DRA device selectors and derived attributes are CEL expressions evaluated once per candidate device during scheduling and allocation.
As those expressions grow—multiple attribute domains, capacity checks, driver preconditions, topology mapping—they become hard to read
and maintain. Authors currently duplicate sub-expressions or rely on `cel.bind()`:

```yaml
expression: |
  cel.bind(dra, device.attributes["dra.example.com"],
    dra.model == "A100" && device.capacity["dra.example.com"].memory >= quantity("40Gi"))
```

VAP and MAP already solve this with a `variables` list: each variable is a named CEL expression, compiled in
order, exposed as `variables.<name>`, and referenced from the main expression. The device-management working group agreed this pattern is useful for DRA as well
([kubernetes#125826](https://github.com/kubernetes/kubernetes/issues/125826),
[enhancements#5259](https://github.com/kubernetes/enhancements/issues/5259)).

This work does **not** block promotion of existing DRA features. It is a usability enhancement with low
adoption pressure today because most users do not author CEL selectors directly.

### Goals

- Add an optional `variables` field to `CELDeviceSelector` and `DerivedAttribute` with VAP-compatible semantics (naming, ordering, acyclic references, lazy evaluation).
- Compile and evaluate variables in the existing DRA CEL environment (`device` input, same libraries as today).
- Enforce the same expression length and cost limits, accounting for variable evaluation cost.
- Gate the API and behavior behind a dedicated feature gate for safe rollout.
- Cover all API surfaces that embed these types today (`DeviceClass`, device request selectors in claims/templates, and `derivedAttributes` on claim/template requests).

### Non-Goals

- **CEL usability improvements outside variables** from [#125826](https://github.com/kubernetes/kubernetes/issues/125826):
  AST-based detection of invalid attribute/capacity domain names, status/event reporting for runtime CEL errors,
  and offline CEL debugging tools. Those may get separate KEPs or issues.
- **Variables in future CEL match expressions** (e.g. device constraints using CEL). The design should not
  preclude reuse, but this KEP only changes `CELDeviceSelector` and `DerivedAttribute`.
- Replacing or deprecating `cel.bind()`' both mechanisms may coexist.
- Changing the `device` CEL type, cost limit constants, or scheduler allocation algorithm beyond what is
  required to evaluate composed expressions.

## Proposal

Extend `CELDeviceSelector` and `DerivedAttribute` with an optional list of variables. The main `expression` continues to be required.
For selectors it must evaluate to `bool`' for derived attributes it must evaluate to a scalar (or list of scalars) per existing KEP-6080 rules.

### User Stories

#### Story 1: Readable GPU selection

As a platform engineer authoring a `DeviceClass`, I want to name intermediate checks (driver attributes,
capacity thresholds) so my team can review selector logic without parsing one long expression.

#### Story 2: Consistent selectors across templates

As a cluster user, I want to define the same attribute/capacity sub-expressions once in a
`ResourceClaimTemplate` selector and combine them in the final expression.

#### Story 3: Readable derived attributes

As a cluster user authoring a `ResourceClaim`, I want to name intermediate CEL sub-expressions in a
derived attribute so topology-mapping expressions stay reviewable.

### Notes/Constraints/Caveats

- **Existing expressions unchanged**: Objects with only `expression` behave exactly as today when the feature
  gate is enabled' `variables` is optional on both selectors and derived attributes.
- **`cel.bind()` remains available** (since Kubernetes 1.31 in the DRA CEL environment). Variables are
  preferable when multiple sub-expressions are reused or when VAP-style `variables.foo` reads clearer than nested binds.
- **Stored API objects**: Validation uses the stored CEL environment for persisted resources, consistent with
  current `validateCELSelector` behavior (see [kubernetes#125826](https://github.com/kubernetes/kubernetes/issues/125826)
  on backward compatibility for stricter validation).

### Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Higher runtime cost from evaluating variables per device | Lazy evaluation of variables (same as VAP)' enforce combined cost limit at admission time' keep default max variable count small |
| Divergence from VAP variable semantics | Reuse the same `Variable` shape and document any intentional differences |
| Cache key collisions in scheduler CEL cache | Cache on canonical serialized selector or derived attribute (expression + variables), not expression string alone |
| API churn if variable limits are wrong | Start conservative (align with VAP limits where applicable)' relax in beta if needed |

## Design Details

### API Changes

Add to `CELDeviceSelector` and `DerivedAttribute` in `pkg/apis/resource/types.go` (and published `resource.k8s.io` API versions):

```go
// CELDeviceSelector contains a CEL expression for selecting a device.
type CELDeviceSelector struct {
    // Variables contain named sub-expressions that can be referenced from Expression.
    //
    // Each variable is compiled in order. A variable may reference the device input and
    // variables declared earlier in the list. The list must be acyclic.
    //
    // In later variables and in Expression, variables are available as variables.<name>,
    // for example variables.attributes for a variable named "attributes".
    // Bare names are not in scope.
    //
    // +optional
    // +listType=map
    // +listMapKey=name
    // +featureGate=DRAVariablesInCEL
    Variables []CELSelectorVariable

    // Expression is unchanged (required). For selectors it must evaluate to bool.
    Expression string
}

// DerivedAttribute also gains the same optional Variables field.
type DerivedAttribute struct {
    Name QualifiedName

    // +optional
    // +listType=map
    // +listMapKey=name
    // +featureGate=DRAVariablesInCEL
    Variables []CELSelectorVariable

    // Expression remains required and must evaluate to a scalar or list of scalars
    // per KEP-6080. It may reference variables.<name>.
    Expression string
}

// CELSelectorVariable is a named CEL sub-expression for use in a CELDeviceSelector
// or DerivedAttribute.
// +structType=atomic
type CELSelectorVariable struct {
    // Name must be a valid CEL identifier and unique among variables in this
    // CELDeviceSelector or DerivedAttribute.
    Name string

    // Expression is evaluated in the DRA device CEL environment. It may refer to device
    // and to variables.<earlierName>. The result type is unconstrained (any CEL type).
    Expression string
}
```

**Example** (from [kubernetes#125826](https://github.com/kubernetes/kubernetes/issues/125826)):

```yaml
selectors:
- cel:
    variables:
    - name: attrs
      expression: device.attributes["gpu.example.com"]
    - name: memOK
      expression: device.capacity["gpu.example.com"].memory >= quantity("40Gi")
    expression: device.driver == "gpu.example.com" && variables.attrs.model == "A100" && variables.memOK
```

**Derived attribute example:**

```yaml
derivedAttributes:
- name: numa
  variables:
  - name: attrs
    expression: device.attributes["gpu.example.com"]
  expression: variables.attrs.numa
```

**Resources affected:**

- `DeviceClass.spec.selectors` (`CELDeviceSelector`)
- `ResourceClaim.spec.devices.requests[].selectors` (`CELDeviceSelector`)
- `ResourceClaimTemplate.spec.spec.devices.requests[].selectors` (`CELDeviceSelector`)
- `ResourceClaim` / `ResourceClaimTemplate` request `derivedAttributes` (`DerivedAttribute`)

**Limits** (proposed' adjust during review):

| Limit | Value | Rationale |
|-------|-------|-----------|
| Max variables per selector or derived attribute | 32 | Matches `DeviceSelectorsMaxSize`' same order of magnitude as practical selector count |
| Max variable name length | 63 | Align with VAP identifier constraints |
| Max per-variable expression length | 10 KiB | Same as `CELSelectorExpressionMaxLength` |
| Max main expression length | 10 KiB | Unchanged |
| Combined compile-time cost | `CELSelectorExpressionMaxCost` | Sum of variable + main expression estimated costs |

OpenAPI/markdown docs for `CELDeviceSelector` and `DerivedAttribute` should cross-link VAP variable documentation for authors
already familiar with admission policies.

### CEL Environment and Semantics

Semantics match VAP variables unless noted:

1. **Input**: Only the existing `device` binding is in scope (driver, attributes, capacity, optional
   `allowMultipleAllocations` when `DRAConsumableCapacity` is enabled).
2. **Access**: Variable `foo` is referenced as `variables.foo` in later variables and in `expression`.
3. **Order**: Variables are compiled and type-checked in list order' forward references are invalid.
4. **Acyclicity**: Enforced at compile time (same as VAP).
5. **Evaluation**: **Lazy**, same as VAP. A variable is evaluated only when first referenced by a later
   variable or by the main `expression`. Result and any error are memoized for that per-device evaluation
   and count once toward runtime cost. Unused variables are never evaluated, so a lookup of a non-existent
   attribute in an unused variable does not fail the selector or derived attribute. Errors from a referenced
   variable surface on the referring expression. Combined with CEL short-circuit of `&&` / `||`, a missing
   attribute in `variables.gpuAttrs` does not fail
   `device.driver == "gpu.example.com" && variables.gpuAttrs.model == "A100"` when the driver check is already
   false. Eager evaluation of all variables in advance is **not** used: it would fail allocation on candidates
   that never need that variable.
6. **Main expression**: Selectors must still compile to `bool` (or acceptable unknown types per existing DRA rules).
   Derived attributes must compile to a scalar or list of scalars (per KEP-6080).
7. **Variable result types**: Any CEL type' only the main expression is type-constrained.

Introduce a `variables` object in the CEL environment, implemented similarly to
`k8s.io/apiserver/pkg/admission/plugin/cel` `CompositedCompiler` (`variablesTypeName = "kubernetes.variables"`).

### Validation

Update `validateCELSelector` and derived-attribute validation in `pkg/apis/resource/validation/validation.go`:

1. When `DRAVariablesInCEL` is disabled: reject non-empty `variables` (strict validation on create/update).
2. When enabled:
   - Validate variable names (non-empty, valid CEL identifier, unique).
   - Use a compositing compiler: compile variables in order, then compile `expression` with `variables` in scope.
   - Reject if combined estimated cost exceeds the existing per-expression cost limit (`CELSelectorExpressionMaxCost` for selectors' KEP-6080 budgets for derived attributes, including the claim-level combined derived-attribute cost cap).
   - Apply length limits per field.

For **stored** resources (`stored == true`), use `environment.StoredExpressions` for both variables and
expression, matching current selector and derived-attribute validation.

### Compilation and Runtime Evaluation

**Package**: `k8s.io/dynamic-resource-allocation/cel`

Today `compiler.CompileCELExpression` compiles a single string and `CompilationResult.DeviceMatches` binds
only `device`. Planned changes:

1. Add `CompileCELSelector` / equivalent for derived attributes that compiles variables + expression together.
2. Extend evaluation to resolve `variables` lazily (VAP model) with cost tracking' do **not** evaluate all
   variables in advance.
3. Update `Cache.GetOrCompile` / `Check` to key on the full selector or derived-attribute content
   (serialize variables + expression).

**Call sites** to update:

- `pkg/apis/resource/validation` (admission)
- `pkg/scheduler/framework/plugins/noderesources` (filter/score paths using CEL cache)
- `k8s.io/dynamic-resource-allocation/structured/...` (allocator tests and allocation logic)

### Cost Limits

Admission validation already rejects expressions whose **estimated** max cost exceeds
`CELSelectorExpressionMaxCost` (currently tied to scheduler safety). With variables:

- Estimate cost for each variable expression and add to a running total.
- Estimate the main `expression` cost in an environment that includes compiled variable types.
- Reject the object if the total exceeds the limit.

At runtime, actual cost may be lower (short-circuiting). Runtime evaluation continues to use the per-evaluation
CEL cost limit configured on the program (unchanged).

### Feature Gate

- **Name**: `DRAVariablesInCEL`
- **Default**: `false` (Alpha)
- **Lock to default**: `false` in Alpha/Beta' `true` in GA
- **Requires**: `DynamicResourceAllocation` (parent DRA gate)

Register in `pkg/features/kube_features.go` and `staging/src/k8s.io/api/resource/...` feature tags.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests.

##### Prerequisite testing updates

None required beyond new unit/integration coverage for variable compilation.

##### Unit tests

- `staging/src/k8s.io/dynamic-resource-allocation/cel`: compile/order/forward-reference/cost aggregation'
  `DeviceMatches` with variables' cache key behavior.
- `pkg/apis/resource/validation`: API validation cases mirroring VAP variable tests (empty name, bad identifier,
  cyclic reference, over cost limit, gate disabled).
- Scheduler plugin unit tests where CEL selectors are exercised.

##### Integration tests

- API server rejects invalid variable selectors' accepts valid ones with gate enabled.
- Stored-version compatibility: old selectors without variables continue to work.

##### e2e tests

- DRA e2e (`test/e2e/dra` or `test/e2e_dra`): `DeviceClass` + claim template using variables selects
  expected devices' negative test with forward reference fails at create time.

### Graduation Criteria

#### Alpha

- Feature gate `DRAVariablesInCEL` implemented end-to-end (API, validation, scheduler evaluation).
- Unit + integration tests as above' at least one e2e happy path.
- KEP marked implementable' enhancement issue linked.

#### Beta

- At least one release of Alpha feedback addressed.
- Complete e2e coverage for claim + class selectors' flake-free testgrid signal for two weeks.
- Documentation on kubernetes.io for authoring variables in device selectors.
- Prod-readiness review completed.

#### GA

- Two releases at Beta with no major API semantic changes.
- Feature gate locked to enabled.

### Upgrade / Downgrade Strategy

- **Upgrade**: Enable gate → authors may start setting `variables`. No migration of existing objects required.
- **Downgrade**: Disable gate → API server rejects create/update with `variables`' existing stored objects with
  variables should be treated like other gated fields (validation on update). Operators should remove `variables`
  from manifests before downgrade or leave objects unchanged if the apiserver preserves unknown fields (document
  recommended cleanup).

### Version Skew Strategy

- **N apiserver / N-1 scheduler**: New selectors with variables are rejected at admission if gate off' with gate on,
  skewed scheduler without support could mis-evaluate—follow standard Kubernetes requirement: enable gate only when
  control plane and scheduler are updated.
- **CEL stored vs new environment**: Handled by existing stored-expression validation path.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a running cluster?

- Feature gate: `DRAVariablesInCEL`
- Requires `DynamicResourceAllocation=true`

###### Does enabling the feature change any default behavior?

No. Until users add `variables` to a selector, behavior is identical.

###### Can the feature be disabled once it has been enabled?

Yes, via feature gate. Selectors that already contain `variables` may fail validation on update while disabled.

### Rollout, Upgrade and Rollback Planning

Low risk: opt-in API field. Rollback is gate disable plus manifest cleanup for objects using variables.

### Monitoring Requirements

###### How can an operator determine if the feature is in use?

```bash
kubectl get deviceclasses,resourceclaimtemplates --all-namespaces -o json \
  | jq '[.. | objects | select(has("cel")) | .cel | select(.variables != null and (.variables | length) > 0)] | length'
```

(Approximate' adjust for your API version.)

No new metrics required for Alpha. Optional: count compilation failures in apiserver audit logs.

### Dependencies

- `DynamicResourceAllocation` feature gate
- `k8s.io/dynamic-resource-allocation/cel` and apiserver CEL libraries

### Scalability

Variables add per-device evaluation work proportional to the number and complexity of variables. Cost limits
and lazy evaluation bound worst-case impact. Scheduler CEL cache must include variables in cache keys to avoid
incorrect reuse.

### Troubleshooting

Invalid variables surface at **resource create/update** with CEL compile errors on the `variables[i].expression`
or `expression` field—same as today for broken selectors. Runtime errors (e.g. missing attribute keys) behavior
is unchanged from the [kubernetes#125826](https://github.com/kubernetes/kubernetes/issues/125826) repro discussion.

## Implementation History

- 2025-04-24: Enhancement issue [kubernetes/enhancements#5259](https://github.com/kubernetes/enhancements/issues/5259) filed
- 2025-04-29: Device-management WG: variables work does not block other DRA promotions ([kubernetes#125826](https://github.com/kubernetes/kubernetes/issues/125826))
- TBD: KEP PR to kubernetes/enhancements
- TBD: Alpha implementation in kubernetes/kubernetes

## Drawbacks

- Additional API surface and implementation complexity in DRA CEL for a niche authoring path.
- Another feature gate for cluster operators to track.
- Potential confusion between `cel.bind()` and `variables`—documentation must explain when to use each.

## Alternatives

### Alternative 1: Document `cel.bind()` only

No API change' improve docs and examples. Rejected: does not match VAP ergonomics' nested binds scale poorly
when reusing multiple sub-expressions.

### Alternative 2: Share `admissionregistration.Variable` type

Reuse the API type from admissionregistration. Rejected: couples resource API to admissionregistration' a
DRA-local `CELSelectorVariable` with identical fields is clearer for OpenAPI and versioning.

### Alternative 3: Macro expansion at admission time

Expand variables into a single expression string before compile. Rejected: loses separate compile errors per
variable, breaks cost estimation per sub-expression, and complicates stored-expression versioning.

## Alternatives considered for feature gate name

`DRACELVariables` was proposed early but avoided to prevent awkward `DRACEL` capitalization
([enhancements#5259](https://github.com/kubernetes/enhancements/issues/5259)).
