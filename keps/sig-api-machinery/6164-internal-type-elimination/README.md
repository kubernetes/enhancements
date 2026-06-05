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
storage type are alls the same type. 

As a first step, a type alias (`type Pod = corev1.Pod`) can be used to tell
go that the internal type is the *same* Go type, so conversion collapses to a no-op.

Longer term we can remove the internal types from the code entirely.

## Motivation

For list operations served by the kube-apiserver, conversion is the last
remaining non-streaming operation, as a result, peak memory utilization
of the kube-apiserver is heavily influenced by the allocations performed
by conversion.

https://github.com/kubernetes/kubernetes/issues/139026 showed that if conversion
operations are streamed we reduce peak memory by up to 46%. By
eliminating conversion entirely we expect to do even better.

The internal type exists for three historical reasons (see [kubernetes/kubernetes#138097][kk-138097]):

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
   internal type is maintained to mirror the newest/preferred versioned type.

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

### Goals

- Preserve the idea of a hub type.
- Eliminate the internal-to-versioned conversion cost for migrated resources.
- Define a simple, incremental, per-resource migration.
- Focus on the migration of high-traffic APIs to harvest the performance and scale benefits.


### Non-Goals

- It is not a goal to migrate all APIs. Low-traffic groups may keep their
  internal type indefinitely. This does not make such types any worse than they are today.
- Changing the conversion-gen, deepcopy-gen, or defaulter-gen tooling contracts
  (they already handle aliases correctly; see Design Details).

## Proposal

### Migration steps

1. **Alias:** Replace the internal struct definitions with aliases to
   the storage version, reconcile any internal-only methods, and regenerate
   deepcopy/conversion.
1. **Switch to a Versioned hub:** Drop the `__internal` registration for the
   resource and set its in-memory/hub version to the storage version.
1. **Cleanup (can long after the main migration).** Migrate all call sites from the internal
   types to the versioned types and delete the internal package.

### Migration order

We will **prioritize the highest-traffic APIs**:

1. **Demonstrate mechanism** Prove the mechanism on a small,
   single-version, structurally-identical group. `rbac.authorization.k8s.io` is
   the reference PoC; `coordination` (`Lease`) and `storage` are similar.
1. **High-traffic easy APIs** These carry most
   apiserver traffic and alias cleanly today: `coordination/Lease` (kubelet
   heartbeats), `discovery/EndpointSlice`, `events.k8s.io/Event`,
   `core/ConfigMap`, `core/Endpoints`, `core/Node`, `core/ServiceAccount`.
1. **High-traffic complex APIs** `core/Pod`,
   `core/Secret` have hand-written conversions and/or internal-only
   fields and cannot be aliased as-is. In such cases, internal type will first need
   to modified to match the hub type.
   and are sequenced after the clean cases.
1. **The long tail.** Remaining groups migrate opportunistically and so may never migrate.

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


- **Regressions:** *The existing round-trip
  fuzz harness will be a huge help in preventing this.
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

### Migration Step 1

For a structurally-identical resource, the internal type definitions are
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

- **Round-trip fuzz:** This migration depends heavily on the existing roundtrip fuzz tests
  to ensure the changes are safe.
- **Conversion benchmarks:** A per-resource conversion benchmark will be included
  in each migration PR.

### Graduation Criteria

Because there is no single runtime feature gate (see PRR), graduation is defined
by mechanism maturity and migration coverage rather than alpha/beta/GA of a
toggle.

#### Alpha

Alpha is not meaningful for this as all changes are production impacting.

#### Beta

This feature will stay in beta for the migration of all high-traffic APIs.

#### Stable

- Migration of all high-traffic APIs is complete.
- All new APIs use a versioned hub type.

### Upgrade / Downgrade Strategy

N/A

### Version Skew Strategy

N/A

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Other
  - **Describe the mechanism:** This is non-user-facing source code change that cannot be gated.
  - **Will enabling / disabling require downtime of the control plane?** No.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

No. But it is an internal-only code change with rigiorious correctness and performance testing planned.

###### What happens if we reenable the feature if it was previously rolled back?

N/A

###### Are there any tests for feature enablement/disablement?

N/A

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

N/A

###### What specific metrics should inform a rollback?

N/A

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

N/A

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

N/A

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

N/A

###### How can someone using this feature know that it is working for their instance?

N/A

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