# KEP-NNNN: DRA Shared Consumable Capacities Across Related Devices

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: SR-IOV bandwidth as a shared parent resource](#story-1-sr-iov-bandwidth-as-a-shared-parent-resource)
    - [Story 2: Generic parent-scoped shared resource](#story-2-generic-parent-scoped-shared-resource)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API additions (draft)](#api-additions-draft)
  - [Allocation behavior (draft)](#allocation-behavior-draft)
  - [Feature gate](#feature-gate)
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
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e tests for all Beta API operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests]
  - [ ] (R) Minimum two week window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) all GA endpoints must be hit by [Conformance Tests] within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] Implementation history section is up to date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation, such as design docs, SIG meeting notes, relevant PRs/issues, release notes

## Summary

Dynamic Resource Allocation (DRA) currently supports:

- parent-scoped shared integer counters via partitionable devices, and
- per-device request-driven consumable capacity via `allowMultipleAllocations`.

However, there is no upstream mechanism that combines both capabilities so that:

- multiple related devices can consume from one shared parent-scoped capacity pool, and
- the consumed amount is determined by the request at allocation time.

This KEP proposes **shared consumable capacities** for DRA. A driver can define shared capacity sets on a resource pool (for example, a parent device budget), associate devices with one or more sets, and have scheduler accounting consume request-driven amounts from those shared sets during allocation.

The proposal is generic and applies to multiple hardware models. SR-IOV VF bandwidth accounting is one motivating use case, but not the only target.

## Motivation

Some resources are naturally parent-scoped and shared across multiple allocatable child devices. Examples include link bandwidth, shared memory channels, or scheduler-governed throughput budgets.

In these cases, publishing only child devices is desirable for user experience and hardware mapping, but scheduler-side accounting must still guard the aggregate shared parent budget.

Existing DRA capabilities leave a gap:

- Partitionable counters are shared across devices, but consumption is static in `ResourceSlice` data.
- Consumable capacity is request-driven, but accounting is device-local.

This causes either over-advertisement, rigid static partitioning, or custom out-of-band admission behavior, all of which reduce correctness and portability.

### Goals

- Introduce a generic DRA model for parent-scoped shared capacities consumed by related child devices.
- Keep request-driven consumption semantics aligned with consumable capacity (`capacity.requests` + request policy).
- Ensure scheduler allocator prevents aggregate over-allocation for each shared capacity set.
- Preserve compatibility with existing DRA features and patterns where possible.

### Non-Goals

- Defining device- or vendor-specific runtime enforcement mechanisms (for example, tc, firmware limits, NIC policy engines).
- Defining standard capacity names across vendors.
- Solving cross-node or cross-pool global capacity aggregation.
- Replacing existing partitionable counters or consumable capacity; this is an additive capability.

## Proposal

Add a new pool-scoped API concept in `ResourceSlice` for shared consumable capacities and allow devices to reference those pools. Reuse existing `ResourceClaim` request syntax for capacity requests, with scheduler accounting performed against referenced shared pools.

At a high level:

1. A driver publishes one or more shared capacity sets, each with named capacities and optional request policy.
2. Each allocatable device references the shared set(s) it draws from.
3. A claim requests capacity using existing `capacity.requests`.
4. During allocation, the scheduler checks and consumes the requested amount from all relevant pools (device-specific and/or shared), rejecting candidates that would exceed any pool limit.

### User Stories

#### Story 1: SR-IOV bandwidth as a shared parent resource

As a cluster user, I request one or more VFs and a bandwidth amount per request.
As an operator, I want scheduler admission to ensure that total bandwidth promised across VFs on the same PF does not exceed PF capacity.

This allows:

- keeping VF as the allocatable unit, and
- preventing over-allocation of PF bandwidth without static pre-partitioning.

#### Story 2: Generic parent-scoped shared resource

As a device driver author, I expose child devices independently but need all allocations to respect a shared parent budget (for example throughput, transaction credits, queue budget).

I can publish one shared capacity set per parent and map children to that set, with scheduler accounting handling aggregate request-driven consumption.

### Notes/Constraints/Caveats

- This proposal adds API and allocator behavior and therefore requires cross-SIG review (at minimum SIG Scheduling and SIG API Machinery).
- Request policy semantics should stay as close as possible to existing consumable-capacity behavior to reduce user surprise.
- Drivers must publish consistent relationships between devices and shared capacity sets; invalid references must be rejected or treated as unsatisfiable.

### Risks and Mitigations

- **Risk: scheduler overhead increases with extra accounting.**  
  Mitigation: follow existing allocator cache/aggregation patterns used by DRA allocation logic and add perf coverage.

- **Risk: API complexity for users and driver authors.**  
  Mitigation: keep user request syntax unchanged (`capacity.requests`) and keep shared-set definitions driver-facing.

- **Risk: invalid or inconsistent pool references.**  
  Mitigation: API validation plus allocator safeguards that reject incomplete or invalid candidate mappings.

## Design Details

### API additions (draft)

This KEP proposes a new field conceptually shaped as:

```yaml
spec:
  sharedCapacities:
  - name: parent-0
    capacities:
    - name: bandwidth
      value: "100G"
      requestPolicy:
        default: "1G"
        validRange:
          min: "1M"
          max: "100G"
          step: "1M"
  devices:
  - name: child-a
    sharedCapacity:
    - capacitySet: parent-0
```

Key points:

- `sharedCapacities` is pool-scoped and supports one or more named capacities per set.
- Devices can reference one or more shared sets through `sharedCapacity[*].capacitySet`.
- Claims continue to use existing `capacity.requests` fields.

### Allocation behavior (draft)

For each capacity request in a candidate device allocation:

1. Resolve all matching capacity pools for that capacity name:
   - device-local pool (if present), and
   - all referenced shared pools that define that name.
2. Apply request policy for each pool to compute consumed amount (defaulting/rounding/range rules).
3. Reject candidate if any pool would exceed available capacity.
4. Tentatively account consumption during in-progress claim allocation to avoid internal over-commit.
5. Persist resulting consumption for reconciliation/restart-safe accounting.

This behavior allows one request to satisfy both:

- child-specific limits (for example per-device cap), and
- parent aggregate limits (shared pool).

### Feature gate

Proposed feature gate name:

- `DRASharedConsumableCapacities`

Components:

- kube-apiserver (field enablement/validation)
- kube-scheduler (allocator accounting logic)

### Test Plan

[ ] I understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

#### Prerequisite testing updates

- Extend or add allocator-focused test fixtures for DRA capacity accounting with mixed device-local and shared pools.

#### Unit tests

- API validation:
  - valid/invalid `sharedCapacities` definitions
  - valid/invalid device references to capacity sets
  - feature-gate transition behavior for new fields
- Scheduler allocator:
  - aggregate accounting across multiple claims
  - candidate rejection when shared pool would be exceeded
  - request policy handling (default/range/step/values)
  - mixed pool accounting when same capacity name exists in both local and shared pools

- `<k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources>`: `<TBD date>` - `<TBD coverage>`

#### Integration tests

- Add integration tests for scheduler allocation with shared capacity sets under feature gate on/off.
- Validate deterministic rejection/success outcomes with multiple competing claims.

#### e2e tests

- Add DRA e2e coverage with a test driver that publishes:
  - multiple child devices mapped to one shared set
  - at least one scenario where aggregate requests exceed shared capacity
- Confirm:
  - scheduling succeeds while capacity remains,
  - claims/pods remain pending after exhaustion,
  - release of allocations restores schedulability.

### Graduation Criteria

#### Alpha

- Feature implemented behind `DRASharedConsumableCapacities`
- Core unit and integration tests in place
- Initial e2e signal for shared-pool accounting correctness

#### Beta

- API shape and policy semantics settled
- Stable periodic tests and no unresolved critical correctness bugs
- Monitoring and troubleshooting guidance documented
- Rollout and rollback behavior validated

#### GA

- Sustained test stability across at least two releases after beta
- No unresolved major correctness regressions
- User docs complete and maintained

### Upgrade / Downgrade Strategy

- Upgrade with feature gate disabled preserves existing behavior (new fields ignored/dropped according to API conventions).
- Enabling feature gate activates shared-pool validation and accounting for compatible objects.
- Downgrade/disable support requires explicit compatibility behavior for objects containing gated fields; exact behavior to be documented with final API review.

### Version Skew Strategy

- Primary logic resides in apiserver and scheduler.
- During control-plane skew, schedulers without feature support must treat gated fields as unsupported and avoid partial accounting behavior.
- Skew handling and failure mode expectations to be validated with integration coverage.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `DRASharedConsumableCapacities`
  - Components depending on the feature gate: kube-apiserver, kube-scheduler
- [ ] Other

###### Does enabling the feature change any default behavior?

Only for workloads and resources that use the new fields and request patterns. Existing DRA objects without the new fields behave as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, by disabling the gate and restarting affected components. Backward handling of already persisted gated fields follows API conventions and final implementation details.

###### What happens if we reenable the feature if it was previously rolled back?

Existing compatible objects become active for shared-capacity accounting again once supporting components are restarted with the gate enabled.

###### Are there any tests for feature enablement/disablement?

Planned:

- API field and validation behavior with gate on/off
- allocator behavior with gate on/off
- compatibility tests for objects written while gate was enabled

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

- Incorrect accounting logic could cause false negatives (pending claims) or false positives (over-allocation).  
- Already running workloads should remain running; primary impact is on new allocations.

###### What specific metrics should inform a rollback?

- Increased allocation failure rate attributable to shared capacity checks
- Unexpected pending claims for requests that should fit
- Scheduler allocation latency regression

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Planned for beta milestone via integration and e2e coverage.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- API inspection of `ResourceSlice` objects containing `sharedCapacities` and devices with `sharedCapacity` references.
- API inspection of `ResourceClaim` allocations consuming named capacities.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Condition name: `<TBD if added>`
  - Other field: allocation results / consumed capacity fields
- [x] Events
  - Event Reason: `<TBD allocation rejected due to shared capacity exhaustion>`
- [ ] Other

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- No material regression to scheduler allocation success latency at supported scale.
- No correctness regressions leading to persistent over-allocation or systematic false rejection under normal operation.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name: `<TBD scheduler dynamic resources allocation metrics>`
  - Components exposing the metric: kube-scheduler
- [x] Other
  - Details: pending claim counts and allocation failure events for targeted classes

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Likely yes. Candidate metrics:

- shared-pool allocation attempts/success/failure by reason
- consumed vs available shared capacity per pool (cardinality constraints permitting)

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No new mandatory external services. Behavior depends on DRA-capable control-plane components and compliant DRA drivers publishing valid resource data.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No fundamentally new call types are expected. It may increase processing per allocation decision due to additional accounting checks.

###### Will enabling / using this feature result in introducing new API types?

No new top-level API types are expected; this proposal adds fields to existing DRA API objects.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, `ResourceSlice` size may increase due to `sharedCapacities` definitions and device references.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Possibly scheduler allocation path latency for DRA-using workloads; this must be measured with integration/perf tests.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Potentially increased scheduler CPU and memory in clusters with many shared-pool tracked allocations. Bounds and acceptable overhead to be validated before beta.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No direct node resource exhaustion risk is expected from this control-plane feature.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Like other scheduler features, new allocations cannot be completed while required API state is unavailable.

###### What are other known failure modes?

- **Invalid driver-published shared capacity references**
  - Detection: allocation failures with clear reason, validation errors where possible
  - Mitigations: fix driver publication; reject invalid objects early
  - Diagnostics: scheduler and apiserver logs/events
  - Testing: unit/integration coverage planned

- **Policy mismatch causing unexpected rounding/default behavior**
  - Detection: discrepancy between requested and consumed values in allocation outputs
  - Mitigations: adjust request policy definitions and documentation
  - Diagnostics: allocation result inspection, scheduler logs/events
  - Testing: unit tests for policy resolution and arithmetic

###### What steps should be taken if SLOs are not being met to determine the problem?

- Compare allocation latency/error rates before and after enablement.
- Inspect pending claims and failure reasons tied to shared capacity checks.
- Disable feature gate as rollback if correctness or latency regression is severe.

## Implementation History

- 2026-03-03: Initial draft created in local design docs, generalized from SR-IOV-specific proposal.

## Drawbacks

- Adds API and scheduler complexity.
- Introduces another DRA concept to learn for driver authors.
- Requires careful policy semantics to avoid ambiguity across capacity sources.

## Alternatives

- Static parent capacity partitioning into child-specific fixed slices (simple but inflexible).
- Model parent capacity as a separate allocatable device and require multi-request claims tied by constraints (works today, but higher user complexity).
- Continue with device-local consumable capacity only (cannot model cross-device parent-scoped budgets correctly).

## Infrastructure Needed (Optional)

None currently.

[Conformance Tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website
