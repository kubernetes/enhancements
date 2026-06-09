# KEP-6164: Eliminating Internal API Types

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Benchmarks](#benchmarks)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Migration steps](#migration-steps)
  - [Migration order](#migration-order)
    - [Commitment](#commitment)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Background](#background)
  - [Migration Step 1](#migration-step-1)
  - [Migration Step 2](#migration-step-2)
  - [Migration Step 3](#migration-step-3)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [Stable](#stable)
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
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
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

Every built-in Kubernetes API group maintains an *internal* ("`__internal`")
representation of its types in addition to the versioned types (`v1`,
`v1beta1`, …). The internal type is the conversion "hub" and also serves as the
type that handwritten validation evaluates against.

This KEP proposes to **incrementally migrate built-in APIs off the internal
type**, using **Go type aliases as the first and lowest-risk step**, and
**prioritizing high-traffic APIs**.

Eliminating needless conversions offers up to 46% reduction on peak memory for list
operations served by the kube-apiserver and has long term maintenance benefits to
the project.

Today many APIs have progressed to a stable `v1` version and the alpha and beta types are no
longer served by the API. For such APIs the internal type, the versioned type and the preferred
storage type are identical. 

As a first step, type aliases (`type Pod = corev1.Pod`) can be used to tell
go that the internal type is the *same* Go type, so conversion collapses to a no-op.

Longer term we can remove the internal types from the code entirely.

## Motivation

For list operations served by the kube-apiserver, conversion is the last
remaining non-streaming operation, as a result, peak memory utilization
of the kube-apiserver is heavily influenced by the allocations performed
by conversion.

https://github.com/kubernetes/kubernetes/issues/139026 showed that if conversion
operations are streamed we reduce peak memory by up to 46% plus eliminate the GC
churn caused by the allocations performed during conversion. By eliminating
conversion entirely we expect to do even better.

The internal type exists for a few historical reasons (see [kubernetes/kubernetes#138097][#138097]):

1. **A single conversion hub.** N versioned types convert to/from one internal
   type (2N conversion functions) instead of pairwise (N^2).
2. **A single validation target.** Hand-written validation is written once,
   against the internal type.
3. **A server-only representation** that is never exposed to clients.

All three have weakened:

1. The hub does not have to be a *separate* type. The hub type can be one of the
   versioned types CustomResourceDefinitions already operate handle CRD versions this
   way. For CRD conversion, the storage version *is* the hub.
2. [KEP-5073: Declarative Validation][kep-5073] (GA in v1.36) move validation
   onto **versioned** types via `validation-gen`, further weakening the
   rationale for an internal type.
3. No known built-in API actually requires a server-only type; in practice the
   internal type is best maintained as a exact copy or near copy of the
   newest/preferred versioned type.

Meanwhile the internal type imposes ongoing costs:

- **Per-request CPU and allocations.** Every request converts at least twice
  (decode -> internal and internal -> encode), storage adds two more
  (internal -> storage on write, storage -> internal on read), and every watch event
  converts once *per watcher*. Each conversion allocates a fresh destination
  object and runs a generated field copy.
- **Maintenance burden.** Each group maintains an extra full set of Go types
  plus generated deepcopy and conversion functions for them, and every new
  field must round-trip losslessly through the internal type across *all*
  supported versions. The drift between internal and versioned types is enough
  of a problem that there are standing proposals to *generate* internal helpers
  from versioned ones ([kubernetes/kubernetes#137731][kk-137731]) just to keep
  them in sync.

This KEP removes the first cost entirely (the extra types and their generated
code) and the second cost for migrated resources (the conversion becomes a
no-op).

### Benchmarks

https://github.com/kubernetes/kubernetes/issues/139026 shows that even without
eliminating conversion, that streaming the operation can reduce peak memory by
by up to 46%.

To validate that eliminating conversion performs as expected, we aliased the
`rbac.authorization.k8s.io` internal types to their `v1` counterparts and
measured an internal↔v1 round-trip conversion of a representative `ClusterRole`:

| | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| Before (distinct internal type) | 470.1 | 736 | 7 |
| After (internal type is an alias of v1) | 80.5 | 32 | 1 |
| **Improvement** | **5.8×** | **23×** | **7->1** |

Note that this is comparing aliasing with unsafe conversion.

### Goals

- Preserve the idea of a hub type.
- Eliminate the internal-to-versioned conversion cost for migrated resources.
- Define a simple, incremental, per-resource migration.
- Sequence by traffic to harvest the performance and scale benefits early.
- Converge on a *single* conversion pattern (a versioned hub) across all
  built-in APIs, rather than carrying two patterns indefinitely.


### Non-Goals

- It is not a goal to migrate all APIs in a single release. The migration is
  sequenced over several releases (see [Migration order](#migration-order)), but
  the end state is a single conversion pattern across *all* built-in APIs, not a
  permanent split between migrated and unmigrated groups.
- Changing the conversion-gen, deepcopy-gen, or defaulter-gen tooling contracts
  (they already handle aliases correctly; see Design Details).

## Proposal

### Migration steps

1. **Alias:** Replace the internal struct definitions with aliases to
   the storage version, reconcile any internal-only methods, and regenerate
   deepcopy/conversion.
1. **Switch to a Versioned hub:** Drop the `__internal` registration for the
   resource and set its in-memory/hub version to the storage version.
1. **Update call sites:** Migrate call sites (admission, registry, controllers)
   from the internal type to the versioned type. This follows each resource's
   hub switch promptly; it is not deferred indefinitely.
1. **Delete the internal package (can be later):** Once no call sites remain,
   delete the internal type and its generated code. This final removal can land
   well after the hub switch.

### Migration order

We sequence by traffic to harvest performance benefits early, while converging
on a single pattern for *all* built-in APIs:

Group 1:

1. **Demonstrate mechanism** Prove the mechanism on a small set of APIs, e.g.
   group `pod` (difficult in terms of scale), CRD (difficult because the internal API complex)
   and an easy group like `rbac.authorization.k8s.io` to demonstrate the
   migration mechanism.

Group 2:

1. **High-traffic easy APIs** These carry most
   apiserver traffic and alias cleanly today: `coordination/Lease` (kubelet
   heartbeats), `discovery/EndpointSlice`, `events.k8s.io/Event`,
   `core/ConfigMap`, `core/Endpoints`, `core/Node`, `core/ServiceAccount`.
1. **Structurally-identical APIs.** All other groups whose internal
   and versioned types are already identical are aliased and switched to a
   versioned hub. These are low-risk and are part of the committed migration,
   not just the high-traffic subset.

Group 3:

1. **High-traffic complex APIs** `core/Pod`,
   `core/Secret` have hand-written conversions and/or internal-only
   fields and cannot be aliased as-is. In such cases, the internal type will
   first need to be modified to match the hub type, and so these are sequenced
   after the clean cases.
1. **The long tail.** Groups that are not yet structurally identical have their
   internal type reshaped to match the hub and are then migrated.

#### Commitment

Our criteria success is the successful migration of the `pod` (most impactful)
and `CRD` (most difficult) APIs. If we fail to migate these within 2 releases we
will add back the internal APIs for all migrated APIs. If these are successful, we
 commit to completing the remaining migrations over a 3-release
(~1 year) window**.

### User Stories

N/A.

kube-apiserver runs with reduced peak load and GC churn.

### Notes/Constraints/Caveats

- **Only structurally-identical types can be aliased directly.** The internal
  type must be field-for-field identical to the versioned type.
- **Defaulting:** Defaulting already runs on the *versioned* object
  at decode time, before conversion to the hub. Aliasing and the hub switch do
  not move the defaulting boundary. However, a few groups
  (e.g. `autoscaling/v1`) intentionally defer some defaulting into the
  internal-conversion step.
- **Admission:** Admission controllers are written against internal types.

### Risks and Mitigations

- **Maintaining the hub type increases maintenance burden.** Today the hub type
  is the internal type. With this change, the hub type will be one of the
  versioned typed and will change type as the feature stabilizes and the storage
  version changes. This is a risk that the extra work of switching the hub type
  each time an API is changed will result in increased maintenance burden and/or
  will discourage API changes. Handw-ritten validation and admission controllers
  are the most likely to be affected.
    - Migiations:
      - The rollout of declarative validation significantly weakens the
        hand-written alidation case.
      - Removal of the internal type reduces the number of API types that must
        be kept in sync and reduces some of the maintenance burden.

Eliminating the cost of maintaining the internal API definitions reduces development burden for all APIs.

- **Regressions:** The existing round-trip fuzz and serialization-compatibility
  harnesses (see [Test Plan](#test-plan)) are the main defense. A migration that
  silently changed serialized output would fail the compatibility fixtures, and a
  conversion that dropped data would fail the round-trip fuzz tests.
- **Defaulting drift:** A few groups (e.g. `autoscaling/v1`) defer some defaulting
  into the internal-conversion step. Collapsing that conversion could skip such
  defaulting; these groups are identified and handled in the long-tail reshaping
  step rather than aliased blindly.
- **Reviewability:** Touching many groups risks large,
  hard-to-review PRs. We will limit PRs to a single migration each.

## Design Details

### Background

`runtime.Scheme` maps Go types to GVKs and back:
`gvkToType map[GVK]reflect.Type` (many GVKs → one type is allowed) and
`typeToGVK map[reflect.Type][]GVK` (one type → many GVKs is *also* allowed and
already used, e.g. by unversioned `metav1` types registered across every
version). Conversion is dispatched in `Scheme.convertToVersion`
(`staging/src/k8s.io/apimachinery/pkg/runtime/scheme.go`), which looks up the
candidate kinds for the source type, asks the target `GroupVersioner` which kind
is wanted, allocates a destination via `s.New(gvk)`, and runs the registered
conversion function.

conversion-gen already emits `unsafe.Pointer` casts for memory-identical
fields, so the *field copy* is cheap; the residual cost is the destination
allocation plus the dispatch and per-call scratch allocations.

## Migration Step 0

Make the internal types memory-identical to the storage version. This drops allocations for
list requests from O(n) to O(1) and early benchmarks suggest the memory reduction may be
up to 60% for large pod lists, better than our target of 46%.

The work here involves making the internal type field-for-field identical, the main complexity is:

- `core.PodSpec -> v1.PodSpec` (this is also the was we get the biggest scale impact by optimizing)
- `runtime.Object -> RawExtension`
- A few fields that are round-tripped through annotations

### Migration Step 1

Once we have structurally-identical resource, the internal type definitions can be
replaced by aliases to the storage version:

```go
// pkg/apis/rbac/types.go
package rbac

import rbacv1 "k8s.io/api/rbac/v1"

type (
	PolicyRule  = rbacv1.PolicyRule
	Role        = rbacv1.Role
	RoleBinding = rbacv1.RoleBinding
	// …
)
```

Object ownership is a bit tricky. We will review all impacted conversion calls, specificaly:

- `ConvertToVersion` / `Convert`: Already perform a deep-copy for self-conversion. This is safe. We're going to need to explicitly opt-out of deep-copy even after switching to a versioned hub.
- `UnsafeConvertToVersion`: Already shares data, the existing expectation is that owner of the converted object is the new owner, so this is low risk.
- Unsafe convert in the PATCH handler.
- Unsafe encode does a hack where is GVK stamps and then removes the stamp in a defer and will need careful review.

Also:

1. Get rid of any internal-only methods
2. Disable deepcopy gen on the internal type

### Migration Step 2

Phase 2 retires the `__internal` registration for the resource and make the
storage version the hub.

- Stop registering the resource under `runtime.APIVersionInternal` and update the
  encoding configuration (`DefaultResourceEncodingConfig`).
- Update decode target and the field-management `HubGroupVersion` to the
  versioned type.
- Adjust the round-trip fuzz harness to work primarily off the new hub version

### Migration Step 3

Delete the internal type after moving all handwritten validation and any custom
functions off of the internal type.

### Test Plan

The best tests to ensure correctness are tests we already have today:

- **Serialization compatibility fixtures.** Because a migration is
  invisible on the wire and in storage, we require the migration to incur
  zero fixture changes. These test are critical to manage upgrade/rollback risk.
- **Round-trip fuzz tests.** Harness:
  `staging/src/k8s.io/apimachinery/pkg/api/apitesting/roundtrip`. This offers some
  assurances that migration does not drop or corrupt data. It complements the
  compatibility fixture tests which offer the strongest accurance that the
  internal type removal has not changed normal codepaths.
- **Conversion benchmarks.** A per-resource conversion benchmark (see
  [Benchmarks](#benchmarks)) is included in each migration PR to confirm the
  expected allocation/latency reduction and guard against regressions.

### Graduation Criteria

This enhancement has no runtime feature gate (see PRR) so each resource
migration is unconditionally in effect the moment it merges, and there is no
higher maturity than "unconditionally on". This is unavoidable due to the nature
of the code change.

#### Alpha

Not possible for this change. SIG Leads agreed that we should communicate the nature of this
change by marking it as sable on the first release. This matches how we've handled
other internal changes.

#### Beta

Same retionale as Alpha.

#### Stable

Phase 1:

- The migration mechanism is proven when a few low risk APIs and then `pod` (most impactful) and
  `CRD` (most difficult) are migrated.
- Because the change cannot be gated, these migrations are production-impacting
  on merge.

Phase 2:

- We're committed to the migration, and the remaining structurally-identical and
  will complete all remaining migration work over a 3-release window.

Phase 3:

- All built-in APIs use a versioned hub type.
- Internal packages for migrated groups are deleted.
- All new APIs use a versioned hub type by default.

### Upgrade / Downgrade Strategy

None required. The change is purely in-memory and stored and wire encodings are
unchanged.

### Version Skew Strategy

None. The internal/hub type is server-local and never appears on the wire or in
storage, so mixed-version control planes are unaffected.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - **Describe the mechanism:** This is a non-user-facing source-code change that
    cannot be gated and cannot be disabled at runtime. It takes effect when an
    apiserver built with the migration runs.
  - **Will enabling / disabling require downtime of the control plane?** N/A. It
    cannot be enabled or disabled and in effect for any apiserver with the merged code.

###### Does enabling the feature change any default behavior?

No. This is an internal-only change.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

No, this change cannot be disabled. There is no feature gate and the migration is
a source-code change. Correctness and performance are validated by round-trip
fuzz tests and conversion benchmarks. To reverse a specific migration, the
migration PR must be reverted (and the revert would then be cherry-picked to release
branches).

###### What happens if we reenable the feature if it was previously rolled back?

Not applicable as there is no enablement toggle to reconcile. Re-landing a
reverted migration (with any needed fixes) would be how we re-enable the feature.

###### Are there any tests for feature enablement/disablement?

Compaibility fixture tests do ensure cross-version compatibility, providing insurance that the
API behavior is the same both before and after the change. This is the closest
we can hope to get to enablement/disablement testing.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A error in the migration could in theory lead to incorrect conversions that could result
data that is incorrectly served or stored. Mistakes around defaulting are risk (@deads2k
pointed this out). Since defaults are populated in the read path, a rollback could resolve
some of the potential errors this migration could cause.

###### What specific metrics should inform a rollback?

A regression in `apiserver_request_duration_seconds` or in the apiserver memory
profile for the migrated resource, or any round-trip / conformance test failure
attributable to a migration. There is no flag to flip; a rollback means
reverting the migration PR.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. 

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

It will always be in effect Kubernetes 1.37+ for migrated APIs.

###### How can someone using this feature know that it is working for their instance?

There is no user-facing changes. The performance improvements may be observable by performance
sensitive workloads.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- kube-apiserver request latency is strictly better than before
- kube-apiserver memory profile is strictly better than before.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `apiserver_request_duration_seconds`
  - Components exposing the metric: kube-apiserver.

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

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

N/A

###### What are other known failure modes?

N/A

###### What steps should be taken if SLOs are not being met to determine the problem?

Rollback of migration PRs will be considered and may be cherry-picked to stable releases.

## Implementation History

TODO

## Drawbacks

Review load of the migration.

## Alternatives

- Add streaming converion for list. We'd prefer avoid this approach
  since it adds complexity to the system. Better to eliminate conversion
  and reduce complexity from the system.