# Analysis: Hand-Written API Definition Code

This document provides the detailed audit and analysis supporting
[KEP-5975](README.md). It catalogs the categories of per-resource hand-written
code in the Kubernetes apiserver, quantifies the boilerplate, and documents
the design rationale for the proposed approach.

## Boilerplate Audit

An audit of hand-written per-resource code across the kube-apiserver codebase:

| Category | Files | Lines | Pattern |
|----------|-------|-------|---------|
| Strategy files | 67 | ~12,000 | Rigid per-method boilerplate |
| Storage files | 69 | ~8,400 | Store config wiring |
| Feature-gate field dropping | 15 | ~800 | Mechanical if/nil pattern |
| Validation wiring | 65 | ~500 | Type assertion + delegation |
| Printer/table registration | 2 | ~3,600 | Column defs + handler reg |
| REST install/registration | 28 | ~4,200 | StorageProvider wiring |
| Field selector code | 37 | ~600 | GetAttrs/Matcher/SelectableFields |
| RBAC bootstrap policy | 1 | ~200 | Per-resource role entries |
| Storage version pinning | 1 | ~70 | Resource-to-version mapping |
| Control plane instance wiring | 2 | ~50 | Dependency threading |

Estimated 60-70% is mechanical boilerplate (~18,000-21,000 lines).

## Strategy File Breakdown

Every strategy file in `pkg/registry/` implements the same ~12 methods.
For a typical resource, the method bodies consist of:

- **Status clearing** — `obj.Status = TypeStatus{}` on create. Present in ~42
  resources. Always the same pattern.
- **Status clearing on update** — `newObj.Status = oldObj.Status` to prevent
  callers from modifying status through the main endpoint. Same ~42 resources.
- **Generation bumping** — increment `Generation` when `Spec` changes. Present
  in ~40 resources. Almost always `apiequality.Semantic.DeepEqual(new.Spec,
  old.Spec)`.
- **Validation delegation** — call `validation.Validate<Kind>(obj)` and wrap
  with `ValidateDeclarativelyWithMigrationChecks`. Present in 65 of 67
  strategy files. Convention-driven by type name.
- **Feature-gate field dropping** — conditionally nil fields when a gate is
  disabled. Present in 15 strategy files with 30 drop functions. The pattern
  is entirely determined by the feature gate constant and the field path.

The storage files repeat the same `registry.Store` configuration:
`NewFunc`, `NewListFunc`, `DefaultQualifiedResource`, `SingularQualifiedResource`,
`CreateStrategy`, `UpdateStrategy`, `DeleteStrategy`, `TableConvertor`.

## Strategy DSL Config Fields

The `Config` struct fields:

| Field | Type | Purpose |
|-------|------|---------|
| `Object` | `runtime.Object` | Prototype for Status/Spec detection via accessor interface |
| `ObjectTyper` | `runtime.ObjectTyper` | Type introspection (usually `legacyscheme.Scheme`) |
| `Scheme` | `*runtime.Scheme` | Enables DV migration check wrapping |
| `Namespaced` | `bool` | Scope |
| `AllowCreateOnUpdate` | `bool` | PUT-creates behavior |
| `AllowUnconditionalUpdate` | `*bool` | Default: true |
| `DefaultGarbageCollectionPolicy` | `*GarbageCollectionPolicy` | Default: DeleteDependents |
| `Validate` | `func(ctx, obj) ErrorList` | Create validation |
| `ValidateUpdate` | `func(ctx, obj, old) ErrorList` | Update validation |
| `WarningsOnCreate/Update` | `func(ctx, obj[, old]) []string` | Warning hooks |
| `PrepareForCreate/Update` | `func(ctx, obj[, old])` | Custom normalization |
| `DropDisabledFields` | `func(obj, old)` | Feature-gate field dropping |
| `GenerationChangedFunc` | `func(obj, old) bool` | Custom generation tracking |
| `ResetFields` / `StatusResetFields` | `map[APIVersion]*Set` | Per-version reset fields |
| `StatusValidateUpdate` | `func(ctx, obj, old) ErrorList` | Status-specific validation |
| `StatusPrepareForUpdate` | `func(ctx, obj, old)` | Status-specific normalization |
| `ValidationConfigOptions` | `[]ValidationConfig` | DV options (normalization rules, enforcement) |

## Feature-Gate Field Dropping Patterns

Current patterns in the codebase:

| Pattern | Example | Generatable? |
|---------|---------|-------------|
| Nil a pointer field | CSIDriver, Job (most fields) | Yes |
| Nil multiple fields under same gate | Job (`BackoffLimitPerIndex` + `MaxFailedIndexes`) | Yes |
| Iterate slice, nil nested field | ResourceSlice (devices[].Taints) | Partially — needs custom drop |
| Filter slice elements | Job (remove FailIndex action) | No — use `+featureGate:customDrop` |
| Multi-gate requirement | ResourceSlice (two gates must both be enabled) | Yes — `+featureGate=A,B` |
| Delegate to shared helper | Deployment → `pod.DropDisabledTemplateFields` | Use `+featureGate:customDrop` |

The generator handles ~70% of patterns mechanically. The remaining 30% use
`+featureGate:customDrop` to call hand-written functions.

## Why Internal DSL Over Code Generation

A code generation approach was prototyped and validated (3 APIs migrated). The
internal DSL was chosen instead because:

1. **`registry.Store` already validates the pattern.** Storage configuration is
   already a config struct. A strategy config struct is the natural analogue.
2. **Custom logic maps to function values.** No convention-based hook discovery
   or naming magic. Resources needing custom warnings or normalization pass
   function values directly.
3. **No codegen pipeline.** No generated files to check in, no
   `hack/update-codegen.sh` wiring, no drift between annotations and output.
4. **Unlimited expressiveness.** Any custom behavior is a function value. The
   expressiveness ceiling is the same as hand-written code.
5. **Full IDE support.** Config fields are typed. Autocomplete, refactoring,
   and go-to-definition all work.

## Why Not Per-Field Declarative Tags for Everything

An approach using field-level tags for all strategy behavior (e.g.,
`+k8s:strategy:immutableAfterCreate`, `+k8s:strategy:clearOnCreate`) was
considered. This works for simple behaviors but breaks down for:

- Validation functions with complex option computation
- Cross-field generation tracking (Deployment bumps generation on Spec OR
  Annotations change)
- Custom warning logic that walks nested structures
- Injected dependencies (authorizer, clock)

The internal DSL handles all of these as function values. Field-level tags are
appropriate for feature-gate field dropping (where the `+featureGate` tag
already exists) but not for the full strategy lifecycle.

## Validation of Approach Against Real PR

Analysis of [kubernetes#137050](https://github.com/kubernetes/kubernetes/pull/137050)
(EvictionRequest API — 265-line strategy with authorization hooks, state
machine validation, and injected dependencies) confirmed:

- Resources with **injected dependencies** (authorizer, clock) work with the
  DSL via closures capturing the dependencies at construction time.
- Even complex resources benefit: the DSL handles the mechanical parts (status
  clearing, generation bumping, validation wrapping, status substrategy) while
  custom logic lives in function values.
- The PR revealed additional boilerplate categories not in the original audit:
  RBAC bootstrap policy, storage version pinning, control plane instance
  wiring, and `hack/lib/init.sh` edits.
