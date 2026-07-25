# KEP-6228: JSON Pointer extraction for Secret and ConfigMap data

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
    - [Story 3](#story-3)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API changes](#api-changes)
  - [Validation](#validation)
  - [Failure behavior](#failure-behavior)
  - [Feature gate](#feature-gate)
  - [Performance](#performance)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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

## Summary

This KEP proposes adding an optional `jsonPointer` field to Secret and ConfigMap
key references so that a consumer receives a selected JSON subtree of a key's
value instead of the whole key. Selection is performed with an
[RFC 6901 JSON Pointer](https://datatracker.ietf.org/doc/html/rfc6901).

The initial scope covers two consumption surfaces in one capability:

- Environment variables sourced through `secretKeyRef` / `configMapKeyRef`
  (resolved by the kubelet before CRI `CreateContainer`).
- Secret and ConfigMap volume items, including projected volume sources, via
  `KeyToPath` (the selected subtree is serialized to compact JSON and written to
  the projected file).

The feature is gated by the `SecretConfigMapJSONPointer` feature gate. It is
limited to selecting one JSON subtree: JSON Pointer can address by object key or
array index and returns exactly one subtree or fails. It does not transform data
(no jq, CEL, templating, regex, scripting, base64 decoding, or string
splitting).

## Motivation

Applications frequently require only a small portion of a structured Secret or
ConfigMap value. Today they must either receive the entire JSON document, which
can expose unrelated credentials to the container, or operators must duplicate
individual fields into separate derived Secrets.

A common pattern is maintaining multiple derived Secrets such as
`docker-ghcr-auth`, `docker-ecr-auth`, and `docker-gcr-auth`, all sourced from
one structured registry credential Secret. This adds operational overhead,
creates drift risk, and weakens least privilege by exposing unrelated
credentials to containers that need only one.

### Goals

- Allow a Secret or ConfigMap key reference to select a JSON subtree via RFC 6901
  JSON Pointer, for both environment variables and projected volume items.
- Re-serialize the selected subtree to compact JSON for volume items; for
  environment variables, decode JSON strings and compact-serialize all other JSON
  types.
- Keep the feature declarative and consistent with existing Secret/ConfigMap
  update and kubelet atomic-writer semantics.
- Ship behind a feature gate; alpha is opt-in.

### Non-Goals

- General data transformation or parsing formats other than JSON; this KEP only
  selects JSON subtrees.
- A JSONPath engine (JSONPath is intentionally not selected; see Alternatives).
- Sub-document handling for `imagePullSecrets` / `kubernetes.io/dockerconfigjson`
  consumed by the kubelet image puller. That is a separate code path that reads
  such Secrets directly and is unaffected by this KEP.
- Producing multiple output keys from one input key.
- Mutation of the source Secret or ConfigMap.

## Proposal

Add an optional `jsonPointer` sub-object to three existing types:

- `SecretKeySelector` and `ConfigMapKeySelector` (used by
  `env[].valueFrom.secretKeyRef` / `configMapKeyRef`).
- `KeyToPath` (used by `secret.items[]`, `configMap.items[]`, and projected
  volume sources `projected.sources[].secret.items[]` /
  `projected.sources[].configMap.items[]`).

The `jsonPointer` sub-object carries both the source key and the pointer
expression. It is mutually exclusive with the existing `key` field: a key
reference uses either `key` (deliver the whole key, current behavior) or
`jsonPointer` (deliver the selected subtree), never both. This mutual exclusion
is enforced at validation and is the foundation of the fail-closed version
skew story (see Version Skew Strategy). The tradeoff is a slightly more complex
API shape than a flat string field, but it prevents older kubelets from
silently delivering the whole document.

Example — environment variable:

```yaml
env:
  - name: REGISTRY_TOKEN
    valueFrom:
      secretKeyRef:
        name: app-config
        jsonPointer:
          key: config.json
          pointer: /registries/ghcr/token
```

Example — projected volume item:

```yaml
volumes:
  - name: config
    projected:
      sources:
        - secret:
            name: app-config
            items:
              - path: token
                jsonPointer:
                  key: config.json
                  pointer: /registries/ghcr/token
```

When `jsonPointer` is set on a key reference:

1. The kubelet decodes the referenced key's bytes as exactly one JSON value
   using `encoding/json` with `Decoder.UseNumber`, so large integer values are
   preserved exactly through decode/re-serialize (no `float64` coercion).
   Trailing non-whitespace data and multiple top-level JSON values are rejected.
2. It evaluates the non-empty RFC 6901 JSON Pointer against the decoded
   document.
3. The value at the pointer is delivered to the consumer:
   - For volume items: the subtree is re-serialized to compact JSON and written
     to the projected `path`, replacing what would otherwise be the raw key
     bytes. File mode/ownership semantics are unchanged.
   - For environment variables: JSON string targets become the decoded string;
     number, boolean, null, object, and array targets become their compact JSON
     representation (`123`, `true`, `null`, `{"host":"db"}`, `[1,2]`).

A pointer whose target does not exist is a hard failure (see
Notes/Constraints). A key that is not exactly one valid JSON value when
`jsonPointer` is set is a hard failure. A `null` target value is delivered
literally (JSON `null` to a file; the string `"null"` to an env var).

### User Stories (Optional)

#### Story 1

An application expects its full registry auth config — host, scope, and token —
as a JSON string in one environment variable. The operator stores credentials
for multiple registries in one structured Secret. With this KEP, one env var
selects the right registry's config object and delivers it as a compact JSON
string, without duplicating data into a separate Secret and without exposing
sibling credentials.

#### Story 2

An operator maintains one structured registry credential Secret and today must
fan it out into `docker-ghcr-auth`, `docker-ecr-auth`, and `docker-gcr-auth` so
that different workloads each receive only their registry's entry as a projected
file. With this KEP, each workload selects its subtree directly from the single
source Secret, eliminating the derived Secrets and their drift.

#### Story 3

An identity provider is configured with a list of OAuth/OIDC connectors (for
example, Google, GitHub, an internal IdP). Each connector's config is a JSON
object with fields like `clientID`, `clientSecret`, and `redirectURI`. Instead
of maintaining one ConfigMap per connector, the operator stores them all in one
ConfigMap key as a JSON object keyed by connector name. Each connector instance
selects its config via `jsonPointer` (for example, `/google`) and receives just
its own config as a projected file.

### Notes/Constraints/Caveats (Optional)

- JSON Pointer is pure selection. It returns exactly one subtree (an object,
  array, or scalar) or fails. It cannot express wildcard matching, filtering, or
  iteration. This is a deliberate safety property, not a limitation to remove
  later.
- The RFC 6901 root pointer (`""`) is intentionally unsupported. It selects the
  whole document, which existing `key` references already provide without this
  feature.
- The selected value is re-serialized, so formatting (whitespace, key order) may
  differ from the source document. Consumers must parse JSON, not byte-compare.
- The feature helps only when the desired value exists as its own JSON node. A
  document whose only field is a base64(`user:pass`) `auth` blob cannot yield
  the split `username`/`password` via selection alone; that requires decode and
  split, which is out of scope.

### Risks and Mitigations

- **Scope creep toward a transformation engine.** JSON Pointer is chosen
  precisely because it cannot express transforms. Follow-ups requesting filtering
  or renaming should be redirected to a sidecar (see Alternatives and #30716).
- **Silent whole-document leak on misconfiguration or version skew.** Two
  mitigations: (1) an unresolved pointer never falls back to delivering the
  whole document, since the whole document is what least-privilege aims to
  withhold; (2) the nested selector makes `key` and `jsonPointer` mutually
  exclusive, so an older kubelet that ignores the unknown `jsonPointer` field
  sees an empty `key` and fails rather than delivering the full document.
- **JSON Pointer escaping confusion** (`~0`, `~1`, leading `/`). Mitigated by
  static pointer-syntax validation at admission.
- **Security.** Net positive: the flagship use case reduces secret exposure.
  JSON Pointer is a pure function of the pointer string over the decoded tree;
  it cannot reference out-of-document data, perform I/O, or cause injection.

## Design Details

### API changes

Add a new `JSONPointerSelector` type and an optional field using it to
`SecretKeySelector`, `ConfigMapKeySelector`, and `KeyToPath`:

```go
// JSONPointerSelector identifies a key and an RFC 6901 JSON Pointer that
// selects a subtree of the JSON document stored in that key.
type JSONPointerSelector struct {
    // key is the key of the Secret or ConfigMap whose value is the JSON
    // document to select from.
    Key string `json:"key" protobuf:"bytes,1,opt,name=key"`

    // pointer is a non-empty RFC 6901 JSON Pointer expression that selects a
    // subtree of the JSON document stored in the key. The selected subtree is
    // delivered to the consumer instead of the raw key bytes. The key must
    // contain valid JSON.
    Pointer string `json:"pointer" protobuf:"bytes,2,opt,name=pointer"`
}
```

On each of `SecretKeySelector`, `ConfigMapKeySelector`, and `KeyToPath`:

```go
// When set, the selected JSON subtree is delivered instead of the raw key
// bytes. Mutually exclusive with key. Feature gate: SecretConfigMapJSONPointer.
// +optional
JSONPointer *JSONPointerSelector `json:"jsonPointer,omitempty" protobuf:"bytes,N,opt,name=jsonPointer"`
```

The existing `key` field becomes optional when `jsonPointer` is set. No new
object types are introduced beyond `JSONPointerSelector`.

### Validation

Added to `pkg/apis/core/validation`:

- Exactly one of `key` or `jsonPointer` must be set on each selector. Setting
  both is rejected; setting neither is rejected. `optional` only affects runtime
  absence of the referenced object or key; it does not waive selector-shape
  validation.
- If `jsonPointer` is set, `jsonPointer.key` is required and must be a valid
  Secret/ConfigMap key name, and `jsonPointer.pointer` is required, non-empty,
  and must be a syntactically valid RFC 6901 pointer (grammar:
  `^(/([^~/]|~[01])*)+$`, with `~0` and `~1` escaping). Pointer validation is
  static and does not require the data. The RFC 6901 root pointer (`""`) is
  rejected; use `key` for whole-key delivery.
- Validation applies to `SecretKeySelector`, `ConfigMapKeySelector`, and
  `KeyToPath`.

### Failure behavior

All failures surface through existing pod-condition/event machinery
(`FailedMount` / `CreateContainerError`), analogous to missing-key errors:

| Condition | Behavior |
| --- | --- |
| Key absent from object | Existing behavior, governed by `optional`. Unchanged. |
| `jsonPointer` set, key present, not valid JSON | Fail; event names the key and states "not valid JSON". |
| `jsonPointer` set, pointer target missing | Fail closed; event states the pointer did not resolve. Never fall back to the whole document. |
| Pointer resolves to `null` | Deliver literal JSON `null`. Not an error. |
| Pointer resolves to scalar/array/object (volume) | Serialize to compact JSON and write the file. |
| Pointer resolves to any type (env var) | Strings become the decoded string; numbers, booleans, null, objects, and arrays become compact JSON. |
| Older kubelet (no `jsonPointer` support) | `key` is empty on the wire; non-optional refs fail with `FailedMount` / `CreateContainerError`; optional env vars or volume items are omitted. Never delivers the whole document. |
| `jsonPointer` set while apiserver gate disabled | Rejected at admission with a clear message. |
| `jsonPointer` set while kubelet gate disabled | Kubelet treats the selector as unsupported and fails/omits as above. It must not evaluate `jsonPointer` while its gate is off. |

### Feature gate

`SecretConfigMapJSONPointer` (alpha -> beta -> GA), on `kube-apiserver` and
`kubelet`.

- Alpha: gated, disabled by default, covering both env var and volume items.
- Beta: enabled by default after at least one release of clean alpha.
- GA: gate removed.

`jsonPointer` set while the apiserver gate is disabled is rejected at admission
with a clear message. A kubelet with the gate disabled treats `jsonPointer` as
unsupported and fails closed (or omits optional env vars / volume items), so a
spec cannot silently degrade to delivering the whole document.

### Performance

- One `json.Decoder` (with `UseNumber`) + pointer evaluation (+ one
  `json.Marshal` for volume items and for non-string env var targets) per
  referenced key that sets `jsonPointer`, on each kubelet refresh of the
  payload. Bounded by existing refresh cadence and the 1 MiB Secret/ConfigMap
  size limit.
- JSON Pointer evaluation is O(pointer length); no regex, no backtracking.
- Peak transient memory is ~3x document size (decoded tree + encoded output +
  input). For a 1 MiB Secret this is ~3 MiB, negligible against existing
  atomic-writer tmp usage.
- No new goroutines, watchers, or network hops.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None identified. The touched packages already have coverage for env var
resolution and projected volume assembly.

##### Unit tests

Core packages to be touched and tested:

- `pkg/volume/projected`: projected volume assembly with `jsonPointer` on
  `KeyToPath` items.
- `pkg/kubelet` (env resolution): `secretKeyRef` / `configMapKeyRef` with
  `jsonPointer`.
- `pkg/apis/core/validation`: pointer-syntax validation; rejection when the gate
  is disabled.
- Shared JSON Pointer parse/evaluate path, fuzzed.

Test matrix: pointer grammar validity; empty/root pointer rejected; `~0`/`~1`
escaping; missing target (fail closed); invalid JSON (fail, including trailing
non-whitespace data and multiple top-level values); scalars/arrays/objects/null;
array index out of range; mode/ownership parity with non-pointer items; large
integer preservation (`9007199254740993` round-trips exactly); mutual exclusion
of `key` and `jsonPointer` at validation; kubelet gate-off fail-closed behavior.

Coverage to be captured at alpha:

- `pkg/volume/projected`: `<date>` - `<coverage>`
- `pkg/kubelet`: `<date>` - `<coverage>`
- `pkg/apis/core/validation`: `<date>` - `<coverage>`

##### Integration tests

Integration tests in `test/integration` covering admission validation of
`jsonPointer` (valid and invalid pointers, apiserver gate-disabled rejection)
and pod creation acceptance with the field set.

##### e2e tests

- Projected Secret with a `dockerconfigjson`-style key; select a registry
  subtree; assert the container sees only that subtree.
- ConfigMap variant.
- Environment variable variant selecting a scalar.
- Negative: bad pointer -> pod fails to mount with the documented reason.
- Negative: non-JSON key with pointer set -> fails.
- Apiserver feature gate disabled -> spec rejected.
- Kubelet feature gate disabled or older kubelet receiving `jsonPointer` spec ->
  non-optional mount/env fails, optional env var/file is omitted, and no
  whole-document delivery occurs.

### Graduation Criteria

#### Alpha

- Fields added to the three types, validated, and implemented in the projected
  volume plugin and kubelet env resolution.
- Shared JSON Pointer parse/evaluate path, fuzzed.
- Unit, integration, and at least one e2e test exercising a real registry
  selection scenario.
- Documentation updated.

#### Beta

- Enabled by default.
- No unresolved correctness or security issues from alpha.
- Benchmark confirming negligible overhead at 1 MiB documents.

#### GA

- At least two releases at beta with no regressions.
- At least one widely-used adoption signal documented.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

### Upgrade / Downgrade Strategy

- The new field is optional and additive. Existing specs that use `key` are
  byte-identical in behavior. Specs that use `jsonPointer` are mutually
  exclusive with `key` (validation enforces exactly one).
- Older clients ignore the unknown `jsonPointer` field on read; the apiserver
  preserves it.
- No etcd migration, no RBAC change, no change to existing Secret/ConfigMap
  write paths.
- Downgrade: remove `jsonPointer` from specs and restore `key` before or during
  rollback. A spec that still has `jsonPointer` but no `key` is rejected once
  the gate is off at admission, and already-admitted workloads fail closed if
  restarted on kubelets without the feature or with the kubelet gate disabled.

### Version Skew Strategy

The nested selector design is fail-closed under version skew by construction:

- **Newer apiserver, older kubelet.** The apiserver accepts the pod spec (gate
  on). The older kubelet deserializes the selector and ignores the unknown
  `jsonPointer` field. Because `key` and `jsonPointer` are mutually exclusive,
  `key` is empty on the wire. The kubelet looks up the empty key in the
  Secret/ConfigMap data and does not find it. Non-optional env vars fail with
  `CreateContainerError`, non-optional volume items fail with `FailedMount`,
  and optional env vars or volume items are omitted. In no case does the older
  kubelet deliver the whole document. This preserves the least-privilege
  guarantee during rollout.
- **Newer apiserver, newer kubelet with kubelet gate disabled.** The kubelet
  sees the typed `jsonPointer` field but must treat it as unsupported while its
  gate is off. It fails or omits the reference with the same behavior as the
  older-kubelet case above, and must not evaluate `jsonPointer` while disabled.
- **Newer apiserver, older kubelet, gate disabled on apiserver.** The spec is
  rejected at admission, so the older kubelet never sees it.
- **No other node components** (CRI, CNI, CSI) are involved: the kubelet
  resolves env vars and writes projected files before invoking the CRI.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `SecretConfigMapJSONPointer`
  - Components depending on the feature gate: `kube-apiserver`, `kubelet`

###### Does enabling the feature change any default behavior?

No. The field is optional and defaults to "deliver the whole key", which is the
current behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, but rollback is not transparent for workloads using `jsonPointer`. Those
workloads must be updated to remove `jsonPointer` and restore `key` before or
during rollback. Setting the gate to `false` rejects new specs that set
`jsonPointer`; already-admitted workloads fail closed if restarted on kubelets
without the feature or with the kubelet gate disabled.

###### What happens if we reenable the feature if it was previously rolled back?

Specs resume selecting subtrees as configured. No persisted state depends on the
gate.

###### Are there any tests for feature enablement/disablement?

Unit tests covering admission validation with the gate on and off, kubelet
gate-disabled fail-closed behavior, and conversion tests for the new optional
field.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A partial rollout (gate on in apiserver, off or unsupported on some kubelets)
means pods scheduled to those kubelets fail closed. Older kubelets see an empty
`key` and cannot find the referenced data; newer kubelets with the gate off must
treat `jsonPointer` as unsupported. Non-optional refs surface
`CreateContainerError` / `FailedMount`; optional env vars or volume items are
omitted. The whole document is never delivered to a kubelet that does not have
the feature enabled. The mitigation is to complete the kubelet upgrade and gate
enablement before relying on the feature.

###### What specific metrics should inform a rollback?

An increase in `CreateContainerError` / `FailedMount` events mentioning JSON
Pointer resolution failures.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be tested before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check for Pod specs that set `jsonPointer` on `SecretKeySelector`,
`ConfigMapKeySelector`, or `KeyToPath` (visible via `kubectl describe` or the
API). A dedicated metric may be added at beta if adoption warrants.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `FailedMount` / `CreateContainerError` on failure; success is
    the env var or file containing exactly the selected subtree.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No new SLOs. Env var resolution and projected volume assembly are within
existing container-start SLOs; the added decode/evaluate work is negligible.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details: existing container start / mount failure metrics; no new metric at
    alpha.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A counter for JSON Pointer resolution failures may be useful at beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. Uses only stdlib `encoding/json` (with `Decoder.UseNumber` for
number-preserving decode).

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Marginal: one optional `JSONPointerSelector` sub-object (two short strings)
per key reference that uses it.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Negligible added time in env resolution and projected volume assembly.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Transient ~3x document size memory per extracted key during refresh; bounded by
the 1 MiB Secret/ConfigMap limit.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Identically to today: the kubelet cannot fetch the Secret/ConfigMap, so the
volume mount or env resolution fails as it currently does.

###### What are other known failure modes?

- Invalid JSON in a key referenced by `jsonPointer` -> mount/env failure
  naming the key.
- Pointer does not resolve -> mount/env failure stating the pointer did not
  resolve; no whole-document fallback.
- Older kubelet, or kubelet with the gate disabled, receiving a `jsonPointer`
  spec -> non-optional mount/env failure or optional env/file omission; no
  whole-document delivery. Mitigation: complete the kubelet upgrade and gate
  enablement.
- Detection: pod events. Mitigation: correct the pointer or the source data.

###### What steps should be taken if SLOs are not being met to determine the problem?

Update affected workloads to remove `jsonPointer` and restore `key`, then
disable the feature gate and restart affected components. Disabling the gate
alone rejects new `jsonPointer` specs and causes already-admitted workloads to
fail closed if restarted on unsupported kubelets or kubelets with the gate off.

## Implementation History

- 2026-07-10: KEP drafted (provisional). Enhancement tracking issue filed as
  [#6228](https://github.com/kubernetes/enhancements/issues/6228).

## Drawbacks

- Adds one field and one code path to the kubelet env resolution and projected
  volume hot paths.
- Re-serialization normalizes formatting; apps that byte-compare against the
  source formatting will see a diff.
- JSON Pointer cannot address "all children" (for example, projecting every
  registry as separate files). Such cases remain on existing workarounds. This is
  intentional.

## Alternatives

- **Do nothing.** Leaves the least-privilege gap and forces app changes or
  duplicated Secrets. Rejected.
- **General transform (jq / CEL / templates).** Powerful but crosses the line
  drawn in [#30716](https://github.com/kubernetes/kubernetes/issues/30716) and
  massively expands kubelet blast radius and attack surface. Out of scope.
- **JSONPath selector.** Not a single standard; can return multiple matches,
  creating serialization ambiguity; its grammar drifts toward a query language.
  Rejected in favor of RFC 6901 JSON Pointer, which returns exactly one subtree
  or fails.
- **External Secrets Operator / initContainer / sidecar.** Valid and already work
  today, and remain the right answer for general transforms. This KEP only covers
  the narrow selection case natively to close the least-privilege gap and avoid
  app changes.
- **A new projection source type.** Larger API surface for the same outcome;
  `KeyToPath` and the key selectors already exist exactly for "how do these
  bytes reach the consumer". Rejected.
- **A flat `jsonPointer` string field alongside `key`.** Rejected: old kubelets
  would still see the populated `key` and deliver the whole document.

## Infrastructure Needed (Optional)

None beyond a normal KEP PR and implementation PRs. No new CI jobs; e2e tests run
in existing sig-storage and sig-node suites.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website
