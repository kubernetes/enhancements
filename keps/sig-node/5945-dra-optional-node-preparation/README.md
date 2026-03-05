# KEP-5945: DRA Optional Node Preparation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: No-op lifecycle class](#story-1-no-op-lifecycle-class)
    - [Story 2: Mixed required and no-op allocations](#story-2-mixed-required-and-no-op-allocations)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API Changes](#api-changes)
  - [Structured Allocator Behavior](#structured-allocator-behavior)
  - [Kubelet Behavior](#kubelet-behavior)
  - [Compatibility, Upgrade, and Skew](#compatibility-upgrade-and-skew)
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
    - [How can this feature be enabled / disabled in a live cluster?](#how-can-this-feature-be-enabled--disabled-in-a-live-cluster)
    - [Does enabling the feature change any default behavior?](#does-enabling-the-feature-change-any-default-behavior)
    - [Can the feature be disabled once it has been enabled?](#can-the-feature-be-disabled-once-it-has-been-enabled)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
    - [How can a rollout failure be detected?](#how-can-a-rollout-failure-be-detected)
    - [How can a rollback failure be detected?](#how-can-a-rollback-failure-be-detected)
  - [Monitoring Requirements](#monitoring-requirements)
    - [How can an operator determine that the feature is in use?](#how-can-an-operator-determine-that-the-feature-is-in-use)
    - [What are the SLO implications of the feature?](#what-are-the-slo-implications-of-the-feature)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
    - [How does this feature react to misconfiguration?](#how-does-this-feature-react-to-misconfiguration)
    - [What are mitigation steps?](#what-are-mitigation-steps)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
      [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
      SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests]
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints] must be hit by [Conformance Tests] within one
        minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
      publication to [kubernetes.io]
- [ ] Supporting documentation, for example additional design documents, links
      to SIG discussions, relevant PRs/issues, and release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website
[Conformance Tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md
[all GA Endpoints]: https://github.com/kubernetes/community/pull/1806

## Summary

Add an explicit DRA allocation-level signal that node-local lifecycle operations
are not needed for certain allocated devices. When that signal is false,
kubelet skips `NodePrepareResources` and `NodeUnprepareResources` for those
results and does not require a kubelet plugin for them.

Default behavior remains unchanged: if the signal is unset, kubelet continues to
require prepare/unprepare semantics exactly as today.

## Motivation

Issue [kubernetes/kubernetes#137122] requests an option for DRA workloads where
node-local prepare/unprepare operations are no-ops.

Current behavior is strict by design:
- kubelet requires successful unprepare before final pod deletion;
- this protects DRA cleanup guarantees and claim lifecycle.

This strict model is correct for resources that need teardown, but it creates
avoidable operational coupling for resources that do not need any node-local
work. Operators may be forced to deploy and maintain a no-op kubelet plugin
only to satisfy interface requirements.

This KEP introduces explicit opt-out semantics that preserve safety-by-default
while allowing no-op cases to avoid unnecessary plugin dependency.

[kubernetes/kubernetes#137122]: https://github.com/kubernetes/kubernetes/issues/137122

### Goals

1. Add explicit API surface to represent whether node-local preparation is
   required.
2. Keep existing behavior for all workloads that do not opt in.
3. Allow kubelet to skip both prepare and unprepare calls for allocations that
   explicitly declare `requiresNodePreparation=false`.
4. Allow classes that never require node-local work to run without a kubelet
   plugin.

### Non-Goals

1. Auto-detect no-op behavior from plugin failures or plugin absence.
2. Relax cleanup guarantees for allocations that do require node-local teardown.
3. Add separate toggles for prepare-only or unprepare-only in this iteration.
4. Introduce asynchronous deferred cleanup in kubelet for this feature.

## Proposal

Introduce a new feature gate `DRANodePreparation` and two optional API fields:

1. `DeviceRequestAllocationResult.requiresNodePreparation` (authoritative runtime
   signal used by kubelet).
2. `DeviceClassSpec.requiresNodePreparation` (input convenience for in-tree
   structured allocator).

Semantics:
- `nil` or `true`: current behavior (plugin required; prepare/unprepare called).
- `false`: kubelet skips prepare/unprepare for that allocation result.

When multiple results exist for one driver in one claim, kubelet computes
per-driver requirement as logical OR over results for that driver:
- if any result is `nil` or `true`, that driver is still required.

### User Stories

#### Story 1: No-op lifecycle class

As a DRA driver author for network-attached or control-plane-managed devices, I
can mark a class as not requiring node preparation. The structured allocator
copies this into allocation results, and kubelet starts and deletes pods without
a local kubelet plugin for those allocations.

#### Story 2: Mixed required and no-op allocations

As a cluster operator, I can run pods with multiple DRA allocations where one
driver requires node preparation and another does not. kubelet still enforces
strict behavior for the required driver while skipping plugin interactions for
the no-op one.

### Notes/Constraints/Caveats

- `DeviceClassSpec` is mutable by cluster admins. Kubelet must rely on
  `AllocationResult` values, not current class state, to avoid retroactive
  behavioral changes.
- This feature is explicit opt-in. Wrongly setting `false` for resources that do
  need teardown may leak external resource state.
- No-op behavior applies only to node-local lifecycle operations.

### Risks and Mitigations

Risk: incorrect opt-out can skip required teardown.
- Mitigation: default remains strict; field is explicit and optional.
- Mitigation: documentation must state when `false` is safe.
- Mitigation: testing covers mixed claims and required-driver retention.

Risk: behavior confusion during skew/downgrade.
- Mitigation: unset means strict behavior, so older kubelets remain safe.

## Design Details

### API Changes

Add fields behind `DRANodePreparation`:

```go
// resource.k8s.io
// DeviceRequestAllocationResult
RequiresNodePreparation *bool `json:"requiresNodePreparation,omitempty"`

// DeviceClassSpec
RequiresNodePreparation *bool `json:"requiresNodePreparation,omitempty"`
```

Field semantics:
- if unset, treated as required (`true`) for backward compatibility.
- this guarantees old objects preserve current strict behavior.

### Structured Allocator Behavior

In-tree structured allocator copies class-level setting to each generated
`DeviceRequestAllocationResult`.

External allocators/controllers may set result-level field directly.

### Kubelet Behavior

During `ClaimInfo` construction, kubelet only includes drivers that require
node preparation. Existing prepare/unprepare flow remains unchanged and naturally
iterates only required drivers.

Pseudo-logic:

```go
for _, result := range claim.Status.Allocation.Devices.Results {
    if gateEnabled && result.RequiresNodePreparation != nil && !*result.RequiresNodePreparation {
        continue
    }
    claimInfoState.DriverState[result.Driver] = state.DriverState{}
}
```

### Compatibility, Upgrade, and Skew

- Backward compatibility: `nil` means strict behavior.
- New apiserver + old kubelet: old kubelet ignores field and remains strict.
- New kubelet + old allocations: field absent, strict behavior.
- Downgrade: behavior reverts to strict; safe by default.

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

No prerequisite framework changes required.

##### Unit tests

Core touched packages and planned coverage:
- `pkg/kubelet/cm/dra`
  - `TestNewClaimInfoFromClaim`: skip drivers when result is no-op.
  - `TestPrepareResources`: no plugin required for no-op-only claim.
  - `TestUnprepareResources`: no unprepare call when no required drivers exist.
- `staging/src/k8s.io/dynamic-resource-allocation/structured/internal/...`
  - allocator tests for class->result propagation.
- `pkg/apis/resource/...`
  - conversions/deepcopy/validation roundtrip for new fields.

##### Integration tests

No dedicated integration test is strictly required for alpha because behavior is
contained in kubelet unit tests and existing API machinery tests.

If requested during review, add kubelet integration coverage for mixed claims.

##### e2e tests

Add a DRA e2e scenario:
- class/allocations with `requiresNodePreparation=false`
- no kubelet plugin for that driver
- pod starts and terminates cleanly

Retain existing e2e semantics for required drivers.

### Graduation Criteria

#### Alpha

- Feature gate `DRANodePreparation` implemented and disabled by default.
- API fields present and generated artifacts updated.
- Unit tests for kubelet and structured allocator behavior merged.
- Initial e2e coverage for no-op path.

#### Beta

- Feature gate enabled by default after feedback from early adopters.
- No unresolved correctness issues for mixed required/no-op claims.
- Stable e2e signal for at least one release cycle.

#### GA

- Feature gate removed after at least two releases at Beta.
- Documentation finalized with operator guidance and failure-mode handling.

### Upgrade / Downgrade Strategy

Upgrade:
- New components read field when gate enabled.
- Existing objects without field remain strict.

Downgrade:
- Older kubelet ignores field and behaves strictly.
- Safety properties are preserved (possible extra strictness only).

### Version Skew Strategy

No new skew requirement beyond existing DRA guarantees.

Skew outcome is intentionally conservative:
- unknown field -> strict behavior.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

#### How can this feature be enabled / disabled in a live cluster?

- Feature gate: `DRANodePreparation`
- Components: kube-apiserver, kube-scheduler, kubelet

#### Does enabling the feature change any default behavior?

No. Default behavior for allocations without explicit field remains unchanged.

#### Can the feature be disabled once it has been enabled?

Yes. Disabling reverts to strict behavior.

### Rollout, Upgrade and Rollback Planning

#### How can a rollout failure be detected?

- Pods unexpectedly stuck in Pending/Terminating
- kubelet DRA operation metrics and logs

#### How can a rollback failure be detected?

Rollback to strict behavior is expected; failures should follow existing DRA
operational patterns.

### Monitoring Requirements

#### How can an operator determine that the feature is in use?

- Observe `requiresNodePreparation` in allocation results.
- Compare DRA gRPC call patterns in kubelet metrics/logs.

#### What are the SLO implications of the feature?

Potentially lower startup/deletion latency for no-op allocations.
No negative effect expected for required allocations.

### Dependencies

Depends on Dynamic Resource Allocation being enabled.

### Scalability

No meaningful additional cost. Driver filtering reduces gRPC calls in no-op
cases.

### Troubleshooting

#### How does this feature react to misconfiguration?

If a resource is incorrectly marked no-op but actually needs teardown, external
state can leak. Pods may complete while external cleanup is incomplete.

#### What are mitigation steps?

- Correct class/allocation configuration.
- For uncertain drivers, keep default strict behavior.
- Follow vendor operational guidance for teardown requirements.

## Implementation History

- 2026-03-04: KEP draft created in provisional state.

## Drawbacks

- Adds API surface and feature-gate complexity.
- Places correctness burden on driver/operator configuration in no-op cases.

## Alternatives

1. Keep strict behavior only and rely on documentation.
   - Rejected: does not solve operational dependency for true no-op cases.
2. Infer no-op from plugin absence/timeouts.
   - Rejected: unsafe and may violate cleanup guarantees.
3. Add asynchronous deferred unprepare in kubelet.
   - Deferred: broader behavior change with larger risk surface.

## Infrastructure Needed

None.
