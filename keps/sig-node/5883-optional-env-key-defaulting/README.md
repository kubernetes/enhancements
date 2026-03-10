# KEP-5883: Optional Key in ConfigMapKeyRef and SecretKeyRef Defaults to Env Var Name

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: ConfigMap-backed env var with matching key](#story-1-configmap-backed-env-var-with-matching-key)
    - [Story 2: Secret-backed env var with matching key](#story-2-secret-backed-env-var-with-matching-key)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Defaulting Behavior](#defaulting-behavior)
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
  - [Always default key to name (no feature gate)](#always-default-key-to-name-no-feature-gate)
  - [Mutating admission to fill in key at admission time](#mutating-admission-to-fill-in-key-at-admission-time)
  - [New boolean field (e.g. <code>useEnvVarNameAsKey: true</code>)](#new-boolean-field-eg-useenvvarnameaskey-true)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Today, when a container environment variable is sourced from a ConfigMap or
Secret, users must specify both the parent `name` field and the `key` field
inside `configMapKeyRef` or `secretKeyRef` — even when those two values are
identical. This KEP proposes that when `key` is omitted and the feature gate
`EnvVarKeyDefaultsToName` is enabled, the kubelet defaults `key` to the value
of the enclosing `env[*].name`. This reduces boilerplate for the common case
where the env var name and the ConfigMap/Secret key are the same.

## Motivation

A typical env var sourced from a Secret looks like:

```yaml
env:
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: my-secret
        key: DB_PASSWORD   # identical to the parent name field
```

The `key: DB_PASSWORD` line adds no information when it matches `name`. At
scale — across many containers and manifests — this repetition adds noise and
creates an opportunity for copy-paste errors where the key and name drift out
of sync silently.

By making `key` optional and defaulting it to the enclosing `env[*].name`,
users can write:

```yaml
env:
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: my-secret
        # key defaults to "DB_PASSWORD"
```

This is fully opt-in: users who need a different key from the env var name
continue to specify `key` explicitly, and the behavior is unchanged for them.

### Goals

- Allow `key` to be omitted from `configMapKeyRef` and `secretKeyRef` when
  the `EnvVarKeyDefaultsToName` feature gate is enabled.
- When `key` is omitted, the kubelet resolves it to the value of the
  enclosing `env[*].name` at runtime.
- API server validation permits an empty `key` only when the feature gate is
  enabled, and additionally validates that `env[*].name` is a valid
  ConfigMap/Secret key in that case.

### Non-Goals

- Changing behavior when `key` is explicitly provided.
- Applying defaulting to `envFrom` sources (`configMapRef`/`secretRef`),
  which already import all keys under an optional prefix.
- Introducing defaulting for `fieldRef` or `resourceFieldRef`.
- Mutating stored API objects to fill in the `key` field at admission time
  (considered for GA; out of scope for alpha).

## Proposal

Introduce a feature gate `EnvVarKeyDefaultsToName` (default: `false` in
alpha). When the gate is enabled:

1. **API validation** (`kube-apiserver`): `key` in `ConfigMapKeySelector` and
   `SecretKeySelector` becomes optional. An empty `key` is accepted only when
   the feature gate is on. When `key` is empty, the validator additionally
   checks that `env[*].name` satisfies `IsConfigMapKey`, since it will be used
   as the lookup key.
2. **Runtime resolution** (`kubelet`): In `makeEnvironmentVariables()`, when
   resolving a `configMapKeyRef` or `secretKeyRef` whose `key` is empty, the
   kubelet substitutes `env[*].name` as the lookup key.

No changes are made to stored API object structure. The `key` field remains a
string in the API types; defaulting is a runtime behavior guarded by the
feature gate.

### User Stories

#### Story 1: ConfigMap-backed env var with matching key

A developer configures an app that reads `LOG_LEVEL` from a ConfigMap. The
ConfigMap key is also `LOG_LEVEL`. With this feature enabled:

```yaml
env:
  - name: LOG_LEVEL
    valueFrom:
      configMapKeyRef:
        name: app-config
        # key omitted — defaults to "LOG_LEVEL"
```

This is equivalent to specifying `key: LOG_LEVEL` explicitly.

#### Story 2: Secret-backed env var with matching key

An operator deploys a service with a database password stored in a Secret
under the key `DB_PASSWORD`:

```yaml
env:
  - name: DB_PASSWORD
    valueFrom:
      secretKeyRef:
        name: db-secret
        # key omitted — defaults to "DB_PASSWORD"
```

One fewer line of YAML per secret reference, with no loss of clarity.

### Notes/Constraints/Caveats

- The defaulting is applied at **runtime** by the kubelet, not at admission.
  This means a stored Pod spec may have an empty `key` field. Tooling that
  inspects stored Pod specs should treat an empty `key` as "use the env var
  name" when the feature is enabled.
- Environment variable names and ConfigMap/Secret keys have overlapping but
  not identical character sets. Validation when the gate is on will enforce
  that `env[*].name` is also a valid ConfigMap/Secret key when `key` is
  omitted.
- This feature is opt-in per pod spec. Existing manifests are unaffected.

### Risks and Mitigations

**Risk:** A Pod spec with an empty `key` is stored in etcd while the feature
gate is enabled, then the gate is disabled. The kubelet would fail to resolve
the env var since an empty key is now unexpected.

**Mitigation:** The feature gate is marked `disable-supported: true`.
Operators who disable the gate after use must audit workloads for empty `key`
fields before doing so. The kubelet emits a clear error event for pods that
fail env var resolution due to an empty key.

**Risk:** Confusion if the env var name is not a valid ConfigMap/Secret key
(e.g. it contains characters not allowed in keys).

**Mitigation:** API server validation (when the gate is on) explicitly checks
that `env[*].name` is a valid ConfigMap/Secret key when `key` is omitted, and
returns a descriptive error if not.

## Design Details

### API Changes

No new fields are introduced. The change is to the **validation** of the
existing `key` field in:

- `core/v1.ConfigMapKeySelector`
- `core/v1.SecretKeySelector`

Currently `key` is required (validated as non-empty in
`validateConfigMapKeySelector` and `validateSecretKeySelector` in
`pkg/apis/core/validation/validation.go`). Under this KEP, when
`EnvVarKeyDefaultsToName` is enabled, `key` becomes optional, and the
validator additionally ensures `env[*].name` is a valid key character set.

### Defaulting Behavior

The defaulting is applied in
`pkg/kubelet/kubelet_pods.go:makeEnvironmentVariables()`:

```go
// configMapKeyRef resolution (simplified)
key := cm.Key
if key == "" && utilfeature.DefaultFeatureGate.Enabled(features.EnvVarKeyDefaultsToName) {
    key = envVar.Name
}
runtimeVal, ok = configMap.Data[key]
```

The same pattern applies to `secretKeyRef`.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates
to existing tests to make this code solid enough prior to committing the
changes necessary to implement this enhancement.

##### Prerequisite testing updates

No prerequisite test changes are required. Existing unit tests for
`validateConfigMapKeySelector`, `validateSecretKeySelector`, and
`makeEnvironmentVariables` will be updated to cover both gate-on and gate-off
scenarios.

##### Unit tests

- `pkg/apis/core/validation`: empty `key` is rejected when gate is off.
- `pkg/apis/core/validation`: empty `key` is accepted when gate is on and
  `env[*].name` is a valid ConfigMap key.
- `pkg/apis/core/validation`: empty `key` with an invalid `env[*].name`
  (e.g. contains `=`) is rejected even when gate is on.
- `pkg/kubelet`: `makeEnvironmentVariables()` resolves an empty `key` to
  `env[*].name` when the gate is on.
- `pkg/kubelet`: `makeEnvironmentVariables()` returns an error for an empty
  `key` when the gate is off.
- `pkg/kubelet`: explicit `key` values are unaffected by the feature gate in
  both states.

##### Integration tests

Not required for alpha. The behavior is fully exercised by unit tests.

##### e2e tests

For alpha, e2e tests will verify:

- A Pod with an omitted `key` in `configMapKeyRef` starts successfully and
  the env var receives the correct value from the ConfigMap.
- A Pod with an omitted `key` in `secretKeyRef` starts successfully and
  the env var receives the correct value from the Secret.
- A Pod with an explicit `key` continues to work correctly regardless of
  feature gate state.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `EnvVarKeyDefaultsToName` feature gate
  (default: `false`).
- Unit tests covering gate-on and gate-off paths for both validation and
  kubelet resolution.
- e2e tests added and passing in CI.

#### Beta

- Feature gate default flipped to `true`.
- Feedback gathered from alpha adopters; no unresolved issues.
- Upgrade and downgrade scenarios tested.
- Documentation published on kubernetes.io.

#### GA

- Feature gate locked to `true` and scheduled for removal.
- No regressions reported over at least two minor release cycles.
- Conformance test added if applicable.

### Upgrade / Downgrade Strategy

**Upgrade:** No action required. The feature is disabled by default in alpha.
Users who want the new behavior must enable the gate and update their
manifests.

**Downgrade:** If the feature gate was enabled and Pods with empty `key`
fields were deployed, those Pods will fail env var resolution after the gate
is disabled or after downgrading to a version without gate support. Operators
should audit for empty `key` fields before downgrading.

### Version Skew Strategy

This feature spans two components: `kube-apiserver` (validation) and `kubelet`
(runtime resolution). Both must have the feature gate enabled for the feature
to work end-to-end.

- **Old apiserver + new kubelet (gate on):** The old apiserver rejects Pods
  with empty `key` at admission. The kubelet never sees them. Safe.
- **New apiserver (gate on) + old kubelet:** The apiserver accepts the Pod.
  The old kubelet does not know how to default an empty `key` and will fail
  to start the container with a clear error. Operators must enable the gate
  on both components before using the feature.
- **New apiserver (gate off) + new kubelet (gate on):** The apiserver rejects
  the empty `key` at admission. The kubelet path is never reached. Safe.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `EnvVarKeyDefaultsToName`
  - Components depending on the feature gate: `kube-apiserver`, `kubelet`

###### Does enabling the feature change any default behavior?

No. Existing Pod specs with explicit `key` values are unaffected. Only Pod
specs that deliberately omit `key` (which is currently a validation error) are
affected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate restores the previous validation behavior. Already
running Pods are not restarted. Pods with empty `key` fields that are
rescheduled after the gate is disabled will fail admission.

###### What happens if we reenable the feature if it was previously rolled back?

The feature resumes working. Pods that were blocked during the rollback will
be accepted again once the gate is re-enabled.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests cover the feature gate switch in both the validation path and
the kubelet resolution path.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Already running workloads are not affected; the kubelet only re-evaluates env
vars when a new container is started. A rollback impacts only newly scheduled
or restarted Pods that use the empty-key syntax.

###### What specific metrics should inform a rollback?

An increase in `kubelet_started_containers_errors_total` with reason related
to env var resolution failure.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested manually before beta graduation.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

An operator can inspect Pod specs stored in the API server for `configMapKeyRef`
or `secretKeyRef` entries where `key` is empty.

###### How can someone using this feature know that it is working for their instance?

The container will start successfully and the env var will contain the expected
value from the ConfigMap or Secret. If the key is not found, the container
fails to start with a clear error event naming the missing key (which will be
the env var name).

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No new SLOs. This is a syntactic convenience with no performance impact.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Existing `kubelet_started_containers_errors_total` covers failure cases.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None identified for alpha.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It depends on ConfigMaps and Secrets already being available to the
kubelet, which is the existing requirement for `configMapKeyRef` and
`secretKeyRef`.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The kubelet's ConfigMap and Secret fetch behavior is unchanged.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. Objects with an omitted `key` are slightly smaller than those with an
explicit key.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The defaulting is a single string assignment in the kubelet hot path with
negligible overhead.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No different from today. If the API server is unavailable, the Pod cannot be
admitted. If etcd is unavailable, the kubelet uses its local cache for
ConfigMap/Secret resolution, same as today.

###### What are other known failure modes?

- **Empty key + key not found in ConfigMap/Secret:** The kubelet emits a
  container start failure event with the message
  `couldn't find key <env-var-name> in ConfigMap/Secret <namespace>/<name>`.
  This is the same error as for an explicit key that is not found today.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check kubelet logs and container events for env var resolution errors. Verify
that the ConfigMap or Secret contains a key matching the env var name.

## Implementation History

- 2026-03-10: KEP created as provisional

## Drawbacks

- Adds subtle implicit behavior: the meaning of an empty `key` depends on the
  feature gate state, which could surprise users reading manifests without
  context.
- Tooling that validates or transforms Pod specs offline (without access to
  the feature gate state) may not correctly interpret an empty `key`.

## Alternatives

### Always default key to name (no feature gate)

Simpler, but changing existing validation without an escape hatch is not
appropriate for a Kubernetes API change. A feature gate is required.

### Mutating admission to fill in key at admission time

Could fill in the `key` field at admission so the stored object always has an
explicit key. This avoids runtime ambiguity of an empty stored `key`. However,
it requires coordination with the admission layer and adds complexity
disproportionate to the problem size. Can be reconsidered for GA if the
runtime defaulting approach proves problematic.

### New boolean field (e.g. `useEnvVarNameAsKey: true`)

A boolean field is more explicit but more verbose — the opposite of the goal.
Omitting `key` is the most natural expression of "use the default".
