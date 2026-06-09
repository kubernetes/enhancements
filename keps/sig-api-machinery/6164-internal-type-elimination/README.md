# KEP-6164: Eliminating Internal API Types

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Benchmarks](#benchmarks)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Phasing](#phasing)
    - [Commitment](#commitment)
  - [User Stories](#user-stories)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Phase 1: Make the internal types memory-identical to the stable served version](#phase-1-make-the-internal-types-memory-identical-to-the-stable-served-version)
    - [Step 1:](#step-1)
    - [Step 2:](#step-2)
  - [Phease 2: Remove internal types](#phease-2-remove-internal-types)
    - [Step 1](#step-1-1)
    - [Step 2](#step-2-1)
    - [Step 3](#step-3)
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

Today many APIs have progressed to a stable `v1` version and the alpha and beta
types are no longer served by the API. For such APIs the internal type, there is
no benefit to having an internal type that differs from the preferred served type.

This KEP proposes a two phrase project:

Phase 1: **make the internal types memory-identical to the preferred served
type**, resulting in O(n) -> O(1) allocation cost for conversions which
reduces peak memory utilization by up to 5x faster and 3x less memory
utilization for the conversions performed by large list operations.

Phase 2: **incrementally migrate built-in APIs off the internal type**, using
**Go type aliases as the first and lowest-risk step**. Longer term we can remove
the internal types from the code entirely. This long term maintenance benefits
to the project.


## Motivation

For list operations served by the kube-apiserver, conversion is the last
remaining non-streaming operation, as a result, peak memory utilization
of the kube-apiserver is heavily influenced by the allocations performed
by conversion.

https://github.com/kubernetes/kubernetes/issues/139026 showed that if conversion
operations are streamed we reduce peak memory by up to 46% plus eliminate the GC
churn caused by the allocations performed during conversion. By optimizing away
conversion costs we expect to do even better.

While many internal types are already memory-identical to the preferred served
type, some older types are not. Critically, the internal PodSpec type is
needless different for the v1 type, and since Go optimizes for copying data
between identical structs, these differences makes conversion expensive and
slow.

A large portion of this KEP will focus on making types memory-identical and
putting guard rails in place to keep them memory-identical to prevent needless
differences, like those in PodSpec, from creeping into the code.

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

internal <-> v1 pod conversion allocs go from O(n) -> O(1)  if the internal type is memory-identical with the versioned type.  This shows the cost of converting `PodList` of various sizes from internal to v1:

| pods | master | this PR | improvement |
|---|---|---|---|
| 1 | 1,435 ns · 4,280 B · **26** allocs | 485 ns · 1,472 B · **5** allocs | 3.0× faster · 2.9× mem |
| 100 | 109,851 ns · 403,879 B · **2,105** allocs | 19,314 ns · 123,072 B · **5** allocs | 5.7× faster · 3.3× mem |
| 1000 | 859,427 ns · 4.00 MB · **21,005** allocs | 148,795 ns · 1.20 MB · **5** allocs | **5.8× faster · 3.3× mem** |

### Goals

- Optimize away internal-to-versioned conversion costs.
- Ensure that the internal types stay memory-identical to the stable served type
  unless there is a good reason for the difference.
- Preserve the idea of a hub type.
- Reduce the use of the internal type in the code base (validation, admission, ...)
  and eventually remove internal types entirely.


### Non-Goals

- It is not a goal to migrate all APIs in a single release. The migration is
  sequenced over several releases (see [Migration order](#migration-order)), but
  the end state is a single conversion pattern across *all* built-in APIs, not a
  permanent split between migrated and unmigrated groups.
- Changing the conversion-gen, deepcopy-gen, or defaulter-gen tooling contracts
  (they already handle aliases correctly; see Design Details).

## Proposal

### Phasing

- Phase 1: Make the internal types memory-identical to the stable served version.
- Phase 2: Remove the internal types.

#### Commitment

Our Phase 1 goal is to optimize away the conversion costs via memory-identical types.

If Phase 1 is successful, we will expore Phase 2, starting with type aliasing.
We commit to either completing Phase 2 or reverting all code to retain the
internal types within a 3-release (~1 year) window**.

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

### Phase 1: Make the internal types memory-identical to the stable served version

This is our v1.37 goal

This drops allocations for list requests from O(n) to O(1) and early benchmarks
suggest the memory reduction may be up to 60% for large pod lists and is where
we expect the vast majority of the performance and scale benefits will be.

#### Step 1:

Make the internal type field-for-field identical, the main complexity is:

- `core.PodSpec -> v1.PodSpec` (this is also the was we get the biggest scale impact by optimizing)
- `runtime.Object -> RawExtension`
- A few fields that are round-tripped through annotations


#### Step 2:

Introduce guardrails to prevent internal types from becoming needlessly differen
than the stable served version.

We plan to modify conversion-gen to track differences between the types and
require exemptions for differences.

This is important to prevent accidental performance regressions. Today we
already have many memory-identical types and are planning to make pod and other
high traffic APIs memory-identical. If any of those types become different than
the performance will regress.

### Phease 2: Remove internal types

This is our goal for v.1.38+

#### Step 1

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

Object ownership is a bit tricky. We will review all impacted conversion calls, specifically:

- `ConvertToVersion` / `Convert`: Already perform a deep-copy for self-conversion. This is safe. We're going to need to explicitly opt-out of deep-copy even after switching to a versioned hub.
- `UnsafeConvertToVersion`: Already shares data, the existing expectation is that owner of the converted object is the new owner, so this is low risk.
- Unsafe convert in the PATCH handler.
- Unsafe encode does a hack where is GVK stamps and then removes the stamp in a defer and will need careful review.

Also:

1. Get rid of any internal-only methods
2. Disable deepcopy gen on the internal type

#### Step 2

Phase 2 retires the `__internal` registration for the resource and make the
storage version the hub.

- Stop registering the resource under `runtime.APIVersionInternal` and update the
  encoding configuration (`DefaultResourceEncodingConfig`).
- Update decode target and the field-management `HubGroupVersion` to the
  versioned type.
- Adjust the round-trip fuzz harness to work primarily off the new hub version

#### Step 3

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