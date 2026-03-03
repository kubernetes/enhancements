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
    - [Story 2: CPU/Memory resource alignment via PCIE root grouping](#story-2-cpumemory-resource-alignment-via-pcie-root-grouping)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API additions](#api-additions)
  - [Allocation behavior](#allocation-behavior)
  - [Feature gate](#feature-gate)
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
    - [Recommended Rollout](#recommended-rollout)
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

This KEP proposes extending the existing DRA `sharedCounters` with `requestPolicy` support and adding a `valueFrom` mapping in `consumesCounters`, so that multiple related devices can consume request-driven amounts from shared counter sets during allocation, with scheduler accounting that prevents aggregate over-allocation.

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
- Replacing existing partitionable counters or consumable capacity; this extends existing API types.

## Proposal

Extend the existing `sharedCounters` in `ResourceSlice` with `requestPolicy` support, and add a `valueFrom` mapping to `consumesCounters` so that devices can map inbound capacity requests to shared counters. Reuse existing `ResourceClaim` request syntax for capacity requests, with scheduler accounting performed against referenced shared counter sets.

At a high level:

1. A driver publishes `sharedCounters` with counters that include `requestPolicy` (default, valid range, step).
2. Each allocatable device references the shared counter set(s) it draws from via `consumesCounters`, using `valueFrom` to map capacity request keys to counters.
3. A claim requests capacity using existing `capacity.requests` fields.
4. During allocation, the scheduler resolves the `valueFrom` mappings, checks and consumes the requested amount from all relevant counter sets (device-specific and/or shared), rejecting candidates that would exceed any counter limit.

### User Stories

#### Story 1: SR-IOV bandwidth as a shared parent resource

As a cluster user, I request one or more VFs and a bandwidth amount per request.
As an operator, I want scheduler admission to ensure that total bandwidth promised across VFs on the same PF does not exceed PF capacity.

This allows:

- keeping VF as the allocatable unit, and
- preventing over-allocation of PF bandwidth without static pre-partitioning.

#### Story 2: CPU/Memory resource alignment via PCIE root grouping

Co-locating compute resources (cores, devices, memory) is essential for low-latency workloads, and there is work to allocate CPU and memory resources using DRA drivers. There is a growing consensus that NUMA or Socket IDs do not correctly represent modern CPU hardware; the community is shifting towards using PCIE roots as the alignment attribute. This is demonstrated by the fact that `pcieRoot` was the first DRA standard attribute. In modern CPUs, more than one PCIE root can be attached to the same NUMA zone, which further reinforces the case for using PCIE roots as the alignment mechanism.

Therefore, a non-directly-accessible (parent) set of resources (a CPUSet and a NUMA node) is shared across accessible (child) devices, represented by PCIE roots.

Example from real hardware (dual XEON Gold 6320R):

```
"root"="pci0000:00" "localCPUs"="0,2,4,...,102" "NUMANode"=0
"root"="pci0000:17" "localCPUs"="0,2,4,...,102" "NUMANode"=0
"root"="pci0000:3a" "localCPUs"="0,2,4,...,102" "NUMANode"=0
"root"="pci0000:85" "localCPUs"="1,3,5,...,103" "NUMANode"=1
"root"="pci0000:ae" "localCPUs"="1,3,5,...,103" "NUMANode"=1
"root"="pci0000:d7" "localCPUs"="1,3,5,...,103" "NUMANode"=1
```

A CPU/memory DRA driver can publish one shared capacity set per NUMA node (the parent budget of available CPUs and memory) and map each PCIE root device to the appropriate set. Scheduler accounting then prevents aggregate over-allocation of CPUs or memory across PCIE roots that share the same NUMA zone.

See also: [kubernetes/enhancements#5491](https://github.com/kubernetes/enhancements/issues/5491) for related work on PCIE-root-based resource alignment.

### Notes/Constraints/Caveats

- This proposal adds API and allocator behavior and therefore requires cross-SIG review (at minimum SIG Scheduling and SIG API Machinery).
- Request policy semantics should stay as close as possible to existing consumable-capacity behavior to reduce user surprise.
- Drivers must publish consistent relationships between devices and shared counter sets; invalid `valueFrom` references must be rejected or treated as unsatisfiable.

### Risks and Mitigations

- **Risk: scheduler overhead increases with extra accounting.**  
  Mitigation: follow existing allocator cache/aggregation patterns used by DRA allocation logic and add perf coverage.

- **Risk: API complexity for users and driver authors.**  
  Mitigation: keep user request syntax unchanged (`capacity.requests`) and keep shared-set definitions driver-facing.

- **Risk: invalid or inconsistent pool references.**  
  Mitigation: API validation plus allocator safeguards that reject incomplete or invalid candidate mappings.

## Design Details

### API additions

This KEP proposes extending existing DRA API types by adding **optional fields** to the existing `Counter` struct. No existing types are removed, renamed, or split. All existing `ResourceSlice` objects remain valid without modification.

Both `sharedCounters` and `consumesCounters` (and the `Counter` type they use) are alpha APIs (`+k8s:alpha(since: "1.36")`), gated behind the `DRAPartitionableDevices` feature gate. Alpha APIs carry no backward compatibility guarantees, but this proposal is designed to be purely additive regardless.

The changes to the `Counter` struct are:

1. **Add an optional `requestPolicy` field** to `Counter`. When a counter is defined in a `CounterSet` (inside `sharedCounters`), this field specifies default values, valid ranges, and step sizes for request-driven consumption. In other contexts this field is ignored.

2. **Add an optional `valueFrom` field** to `Counter`. When a counter is referenced in `DeviceCounterConsumption` (inside `consumesCounters`), this field maps an inbound `capacity.requests` key to the counter, making consumption request-driven rather than static.

3. **Relax the `value` field** from required to conditionally required: in `consumesCounters`, either `value` (static consumption, existing behavior) or `valueFrom` (request-driven consumption, new behavior) must be specified. Existing objects that set `value` continue to work unchanged.

Example `ResourceSlice` with shared counter request policy:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: counter-slice
spec:
  driver: "resource-driver.example.com"
  pool:
    generation: 1
    name: "my-pool"
    resourceSliceCount: 2
  sharedCounters:
  - name: pf-0-counter-set
    counters:
      bandwidth:
        value: "100G"
        requestPolicy:
          default: "1G"
          validRange:
            min: "1M"
            max: "100G"
            step: "1M"
```

Example `ResourceSlice` with device `valueFrom` mapping:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceSlice
metadata:
  name: device-slice
spec:
  driver: "resource-driver.example.com"
  pool:
    generation: 1
    name: "my-pool"
    resourceSliceCount: 2
  nodeName: "my-node"
  devices:
  - name: vf-0
    consumesCounters:
    - counterSet: pf-0-counter-set
      counters:
        bandwidth:
          valueFrom:
            capacityKey: "resource-driver.example.com/bandwidth"
  - name: vf-1
    consumesCounters:
    - counterSet: pf-0-counter-set
      counters:
        bandwidth:
          valueFrom:
            capacityKey: "resource-driver.example.com/bandwidth"
```

Example `ResourceClaim` requesting bandwidth from one of the VFs above:

```yaml
apiVersion: resource.k8s.io/v1
kind: ResourceClaim
metadata:
  name: my-vf-claim
spec:
  devices:
    requests:
    - name: vf-request
      exactly:
        deviceClassName: sriov-vfs
        capacity:
          requests:
            resource-driver.example.com/bandwidth: "10G"
```

The `capacity.requests` key `resource-driver.example.com/bandwidth` matches the `capacityKey` declared in the device's `consumesCounters[].counters[].valueFrom`. When the scheduler allocates `vf-0` or `vf-1` for this claim, it resolves that mapping and subtracts 10G from the shared `pf-0-counter-set` bandwidth counter (100G total), subject to the `requestPolicy` defined on the counter set.

Key points:

- **No breaking changes.** No existing types are removed, renamed, or split. Only optional fields are added to the existing `Counter` struct. All existing `ResourceSlice` objects remain valid.
- `requestPolicy` on a shared counter defines how request values are validated and defaulted.
- `valueFrom` on a device counter consumption maps a `capacity.requests` key to the counter, making consumption request-driven rather than static.
- Making `valueFrom` a struct allows future extensions (e.g. multipliers or transformations) without further API changes.
- Claims continue to use existing `capacity.requests` fields.

### Allocation behavior

For each capacity request in a candidate device allocation:

1. Resolve all `valueFrom` mappings on the candidate device's `consumesCounters` to determine which capacity request keys feed into which shared counters.
2. For each resolved counter, apply the counter's `requestPolicy` to compute the consumed amount (defaulting/rounding/range rules).
3. Reject the candidate if any counter set would exceed available capacity.
4. Tentatively account consumption during in-progress claim allocation to avoid internal over-commit.
5. Persist resulting consumption for reconciliation/restart-safe accounting.

This behavior allows one request to satisfy both:

- device-specific static counter consumption (existing `value` path), and
- parent aggregate limits via shared counter sets (new `valueFrom` path).

### Feature gate

Proposed feature gate name:

- `DRASharedConsumableCapacity`

Components:

- kube-apiserver (field enablement/validation)
- kube-scheduler (allocator accounting logic)

### Test Plan

[ ] I understand the owners of the involved components may require updates to existing tests to make this code solid enough prior to committing the changes necessary to implement this enhancement.

#### Prerequisite testing updates

- Extend or add allocator-focused test fixtures for DRA counter accounting with mixed static and `valueFrom`-driven consumption.

#### Unit tests

- API validation:
  - valid/invalid `requestPolicy` on shared counter definitions
  - valid/invalid `valueFrom` references in device counter consumption
  - feature-gate transition behavior for new fields
- Scheduler allocator:
  - aggregate accounting across multiple claims
  - candidate rejection when shared counter set would be exceeded
  - `requestPolicy` handling (default/range/step/values)
  - mixed accounting when same counter has both static `value` and `valueFrom`-driven consumption

- `k8s.io/dynamic-resource-allocation/structured/internal/experimental`: `<TBD date>` - `<TBD coverage>`
- `k8s.io/dynamic-resource-allocation/structured/internal/incubating`: `<TBD date>` - `<TBD coverage>`
- `k8s.io/kubernetes/pkg/apis/resource/validation`: `<TBD date>` - `<TBD coverage>`
- `k8s.io/kubernetes/pkg/registry/resource/resourceslice`: `<TBD date>` - `<TBD coverage>`
- `k8s.io/kubernetes/pkg/scheduler/framework/plugins/dynamicresources`: `<TBD date>` - `<TBD coverage>`

#### Integration tests

- Add integration tests for scheduler allocation with `requestPolicy` and `valueFrom` under feature gate on/off.
- Validate deterministic rejection/success outcomes with multiple competing claims.

#### e2e tests

- Add DRA e2e coverage with a test driver that publishes:
  - multiple child devices with `valueFrom` mapped to one shared counter set
  - at least one scenario where aggregate requests exceed shared counter capacity
- Confirm:
  - scheduling succeeds while capacity remains,
  - claims/pods remain pending after exhaustion,
  - release of allocations restores schedulability.

### Graduation Criteria

#### Alpha

- Feature implemented behind `DRASharedConsumableCapacity`
- Core unit and integration tests in place
- Initial e2e signal for shared counter accounting correctness

#### Beta

- API shape and policy semantics settled
- Stable periodic tests and no unresolved critical correctness bugs
- Monitoring and troubleshooting guidance documented
- Rollout and rollback behavior validated
- scheduler_perf tests covering overhead of shared counter accounting

#### GA

- Sustained test stability across at least two releases after beta
- No unresolved major correctness regressions
- User docs complete and maintained

### Upgrade / Downgrade Strategy

#### Recommended Rollout

Three components are involved: the kube-apiserver, the kube-scheduler, and the DRA driver that publishes ResourceSlices.

The recommended enablement / upgrade sequence:

1. **Enable the gate (`DRASharedConsumableCapacity`)** on kube-apiserver and kube-scheduler.
   Until any driver publishes `requestPolicy` or `valueFrom` fields, this is a no-op for scheduling.

2. **Update the DRA driver** to publish `requestPolicy` on shared counters and `valueFrom` in `consumesCounters`.
   From this point on, the scheduler enforces shared counter accounting for devices using `valueFrom`.

**Why this order**: when the gate is OFF on the apiserver, `requestPolicy` and `valueFrom` are stripped from incoming ResourceSlice writes (standard alpha-field handling). A driver that publishes these fields before the gate is enabled will see them silently dropped; enabling the gate later does not retroactively restore them, and the driver must republish.

**Recommended downgrade / disablement order**: reverse of upgrade — update the DRA driver first (remove the new fields), then disable the gate on scheduler and apiserver.

### Version Skew Strategy

- **kube-apiserver**: Must be upgraded first to accept the new `requestPolicy` and `valueFrom` fields on `Counter`.
- **kube-scheduler**:
  - A scheduler that understands this feature resolves `valueFrom` mappings and enforces `requestPolicy` during shared counter accounting.
  - An older scheduler ignores `requestPolicy` and `valueFrom`. Devices with `valueFrom` (no static `value`) in `consumesCounters` are treated as consuming zero from the shared counter set, so placement may be overly permissive until the scheduler is upgraded.
- **kubelet**: No changes required; kubelet does not interpret `requestPolicy` or `valueFrom`.
- **DRA driver**:
  - Drivers publish ResourceSlices with `requestPolicy` on shared counters and `valueFrom` in `consumesCounters`.
  - A driver that publishes the new fields before the apiserver gate is enabled will see them silently dropped at write time.

During version skew, the main risk is overly permissive scheduling by an older scheduler that does not enforce shared counter budgets. This is operationally acceptable as a transient state during rolling upgrade, similar to other DRA features.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `DRASharedConsumableCapacity`
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

- Increased allocation failure rate attributable to shared counter checks
- Unexpected pending claims for requests that should fit
- Scheduler allocation latency regression

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Planned for beta milestone via integration and e2e coverage.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- API inspection of `ResourceSlice` objects containing `sharedCounters` with `requestPolicy` and devices with `valueFrom` in `consumesCounters`.
- API inspection of `ResourceClaim` allocations consuming named capacities.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: `ResourceClaim.Status.Allocation` reflects resolved `valueFrom` consumption against shared counter sets
- [ ] Events
  - Event Reason:
- [ ] Other

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

- No material regression to scheduler allocation success latency at supported scale.
- No correctness regressions leading to persistent over-allocation or systematic false rejection under normal operation.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric names:
    - `scheduler_unschedulable_pods` with `plugin="DynamicResources"`
    - `scheduler_plugin_execution_duration_seconds` with `plugin="DynamicResources"`
    - `apiserver_request` with `resource="resourceclaims"`
  - Components exposing the metric: kube-apiserver, kube-scheduler
- [ ] Other
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. Shared counter consumption visibility is not covered by existing metrics. Per-counter utilization metrics would be too granular and high-cardinality for the scheduler. Pool-level resource availability visibility is tracked by [KEP-5677](https://github.com/kubernetes/enhancements/issues/5677), which could be extended in the future to include shared counter consumption data.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No new mandatory external services. Behavior depends on DRA-capable control-plane components and compliant DRA drivers publishing valid resource data.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No fundamentally new call types are expected. It may increase processing per allocation decision due to additional accounting checks.

###### Will enabling / using this feature result in introducing new API types?

No new top-level API types are expected; this proposal extends fields on existing DRA API objects (`Counter`, `DeviceCounterConsumption`).

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, `ResourceSlice` size may increase slightly due to `requestPolicy` on shared counters and `valueFrom` on device counter consumption entries.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Possibly scheduler allocation path latency for DRA-using workloads; this must be measured with integration/perf tests.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Potentially increased scheduler CPU and memory in clusters with many `valueFrom`-driven shared counter allocations. Bounds and acceptable overhead to be validated before beta.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No direct node resource exhaustion risk is expected from this control-plane feature.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Like other scheduler features, new allocations cannot be completed while required API state is unavailable.

###### What are other known failure modes?

- **Invalid driver-published `valueFrom` or counter references**
  - Detection: allocation failures with clear reason, validation errors where possible
  - Mitigations: fix driver publication; reject invalid objects early
  - Diagnostics: scheduler and apiserver logs/events
  - Testing: unit/integration coverage planned

- **`requestPolicy` mismatch causing unexpected rounding/default behavior**
  - Detection: discrepancy between requested and consumed values in allocation outputs
  - Mitigations: adjust `requestPolicy` definitions and documentation
  - Diagnostics: allocation result inspection, scheduler logs/events
  - Testing: unit tests for policy resolution and arithmetic

###### What steps should be taken if SLOs are not being met to determine the problem?

- Compare allocation latency/error rates before and after enablement.
- Inspect pending claims and failure reasons tied to shared capacity checks.
- Disable feature gate as rollback if correctness or latency regression is severe.

## Implementation History

- 2026-03-03: Initial draft created in local design docs, generalized from SR-IOV-specific proposal.

## Drawbacks

- Adds scheduler complexity for `valueFrom` resolution and `requestPolicy` enforcement.
- Adds optional fields to the existing `Counter` struct, which introduces context-dependent semantics (`requestPolicy` only meaningful in `sharedCounters`, `valueFrom` only meaningful in `consumesCounters`).
- Requires careful `requestPolicy` semantics to avoid ambiguity when both static and request-driven consumption apply.

## Alternatives

- **Introduce a new `sharedCapacities` top-level concept.** An earlier version of this KEP proposed a separate `sharedCapacities` field in `ResourceSlice` with its own capacity sets and device references. This was rejected in favor of extending the existing `sharedCounters` and `consumesCounters` API, which avoids introducing a new concept and keeps the API surface smaller.
- **Static parent capacity partitioning into child-specific fixed slices.** Simple but inflexible; cannot adapt to varying per-request consumption patterns.
- **Model parent capacity as a separate allocatable device and require multi-request claims tied by constraints.** Works today, but increases user complexity significantly.
- **Continue with device-local consumable capacity only.** Cannot model cross-device parent-scoped budgets correctly.
- **Promote shared counters to the scheduler framework level for cross-plugin use.** This would allow shared capacity accounting across different scheduler plugins (e.g. DRA and ResourceFit, see [#5517](https://github.com/kubernetes/enhancements/issues/5517)). Feasible but requires solving single-owner population semantics and is a larger scope change. This KEP focuses on the DRA-specific mechanism; framework-level promotion could be a future evolution built on top of this work.

## Infrastructure Needed (Optional)

None currently.

[Conformance Tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website
