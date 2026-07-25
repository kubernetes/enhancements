# KEP-6252: Hugepages Reservation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: OVS-DPDK](#story-1-ovs-dpdk)
    - [Story 2: Other system daemons](#story-2-other-system-daemons)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Flag Parsing](#flag-parsing)
  - [Allocatable Computation](#allocatable-computation)
  - [Memory Manager Integration](#memory-manager-integration)
  - [Feature Gate](#feature-gate)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
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

This KEP proposes extending kubelet's `--system-reserved` and `--kube-reserved`
flags to accept hugepages resources (e.g., `hugepages-2Mi=512Mi`). When set,
the specified hugepages are subtracted from the node's `Allocatable` capacity,
preventing kubelet from offering hugepages that are already consumed by system
daemons to pods.

## Motivation

System daemons such as OVS-DPDK allocate hugepages at startup for their own use
(see [OVS-DPDK hugepages setup](https://docs.openvswitch.org/en/latest/intro/install/dpdk/#setup-hugepages)).
When this happens, kubelet is unaware that those hugepages are taken and
continues to report them as `Allocatable`. This leads to pods being scheduled
with hugepages requests that cannot actually be fulfilled, causing allocation
failures at runtime.

Today, `--system-reserved` and `--kube-reserved` only accept `cpu`, `memory`,
`pid`, and `ephemeral-storage`. There is no supported way to tell kubelet to
exclude system-consumed hugepages from the allocatable pool.

See also [kubernetes/kubernetes#140544](https://github.com/kubernetes/kubernetes/issues/140544)
and [kubernetes/kubernetes#124357](https://github.com/kubernetes/kubernetes/pull/124357),
which implemented this feature but went stale due to lack of a concrete use case
at the time.

### Goals

- Allow `hugepages-<size>` resource types in `--system-reserved` and
  `--kube-reserved` kubelet flags.
- Subtract reserved hugepages from node `Allocatable` so the scheduler does not
  over-commit hugepages that are consumed by system processes.
- Enable `--reserved-memory` to work with hugepages end-to-end. The memory
  manager's validation requires that `--reserved-memory` totals match
  `system-reserved + kube-reserved + eviction-threshold` for each resource type.
  Once hugepages are accepted in `--system-reserved` / `--kube-reserved`, this
  validation will naturally pass.

### Non-Goals

- Changes to the Kubernetes scheduler.
- Changes to the hugepages resource model (sizes, naming conventions).
- Hugepages support in eviction thresholds.

## Proposal

Extend the kubelet flag validation to accept `hugepages-<size>` keys in
`--system-reserved` and `--kube-reserved`, gated behind a new
`SystemReservedHugepages` feature gate.

The existing `Allocatable` computation in kubelet already iterates over all
resource types in node capacity and subtracts matching entries from
`system-reserved` and `kube-reserved`. The only change needed is to allow
hugepages keys in the flag validation.

### User Stories

#### Story 1: OVS-DPDK

As a cluster administrator running OVS-DPDK for high-performance networking,
I configure OVS to use DPDK which requires allocating hugepages for packet
buffers. I want to reserve those hugepages via `--system-reserved` so that
kubelet does not offer them to pods, avoiding runtime allocation failures.

Example:
```
--system-reserved=cpu=500m,memory=1Gi,hugepages-1Gi=2Gi
```

#### Story 2: Other system daemons

As a cluster administrator running system daemons that require hugepages,
I need to reserve those hugepages at the kubelet level to prevent resource
contention with pods.

### Notes/Constraints/Caveats (Optional)

The `--reserved-memory` flag already shows hugepages in its help text example
(`--reserved-memory 0:memory=1Gi,hugepages-1M=2Gi`), but in practice this
cannot be used for hugepages today because the memory manager's validation
requires matching values in `system-reserved + kube-reserved +
eviction-threshold`, which don't accept hugepages. This KEP unblocks that
path as a side effect.

### Risks and Mitigations

**Risk:** Misconfiguration — an administrator reserves more hugepages than
available on the node.
**Mitigation:** Kubelet already validates that reservations do not exceed node
capacity via `validateNodeAllocatable()`. This validation applies to all
resource types including hugepages once they are accepted.

**Risk:** Feature interaction with Memory Manager — reserved hugepages must be
consistent with `--reserved-memory` when the Memory Manager is enabled.
**Mitigation:** The existing `validateReservedMemory()` check enforces this
consistency. No additional validation is needed.

## Design Details

### Flag Parsing

`--system-reserved` and `--kube-reserved` use `MapStringString` and accept
arbitrary key-value pairs. The values are parsed by `parseResourceList()` in
`cmd/kubelet/app/server.go`, which has a switch case that only accepts `cpu`,
`memory`, `ephemeral-storage`, and `pid`. Any other resource type is rejected
with `"cannot reserve %q resource"`. This switch case needs to be expanded to
also accept `hugepages-<size>` keys when the `SystemReservedHugepages` feature
gate is enabled.

### Allocatable Computation

`GetNodeAllocatableReservation()` in `pkg/kubelet/cm/node_container_manager_linux.go`
iterates over all resource types in node capacity and sums `SystemReserved[k] +
KubeReserved[k] + evictionReservation[k]`. The result is subtracted from
`Capacity` to compute `Allocatable` on the node status. Once hugepages are
accepted in the flags, they will be included in this subtraction automatically
with no code changes needed in this path.

### Memory Manager Integration

`validateReservedMemory()` in `pkg/kubelet/cm/memorymanager/memory_manager.go`
checks that `--reserved-memory` totals equal `nodeAllocatableReservation` for
each memory-type resource. Once hugepages flow into `system-reserved` /
`kube-reserved`, the hugepages portion of `nodeAllocatableReservation` will be
non-zero, and the validation will pass for matching `--reserved-memory` values.

### Feature Gate

A new feature gate `SystemReservedHugepages` controls this behavior:
- **Disabled (default in alpha):** Hugepages keys in `--system-reserved` and
  `--kube-reserved` are rejected, preserving current behavior.
- **Enabled:** Hugepages keys are accepted and processed.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None.

##### Unit tests

- Extend `TestNodeAllocatableReservationForScheduling` in
  `pkg/kubelet/cm/node_container_manager_linux_test.go` to include hugepages
  resources in `SystemReserved` and `KubeReserved` and verify the allocatable
  computation.
- Add validation tests for flag parsing with hugepages keys, both with the
  feature gate enabled and disabled.

- `k8s.io/kubernetes/pkg/kubelet/cm`: `TBD`

##### Integration tests

- TBD

##### e2e tests

- TBD

### Graduation Criteria

#### Alpha

- Feature implemented behind the `SystemReservedHugepages` feature gate.
- Unit tests covering flag validation and allocatable computation.
- Initial e2e tests completed and enabled.

#### Beta

- Gather feedback from developers and users.
- Feature gate enabled by default.
- Extend e2e test coverage.

#### GA

- Feature gate locked to enabled.
- At least two releases since beta.
- Real-world usage confirmed.

### Upgrade / Downgrade Strategy

No special upgrade steps required. The feature is opt-in via the
`--system-reserved` / `--kube-reserved` flags. Existing clusters that do not
set hugepages in these flags are unaffected.

On downgrade, if hugepages were configured in `--system-reserved` or
`--kube-reserved`, the kubelet will reject the configuration and fail to start.
Administrators must remove hugepages entries from these flags before downgrading.

### Version Skew Strategy

This feature is kubelet-only and does not involve coordination with the control
plane. The kubelet independently computes `Allocatable` and reports it on the
node status. No version skew concerns exist.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `SystemReservedHugepages`
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

No. The feature only takes effect when an administrator explicitly adds
hugepages entries to `--system-reserved` or `--kube-reserved`. Without those
entries, behavior is identical to today.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disable the feature gate and remove hugepages entries from
`--system-reserved` / `--kube-reserved`, then restart kubelet. The node's
`Allocatable` will return to its previous values.

###### What happens if we reenable the feature if it was previously rolled back?

Hugepages entries in `--system-reserved` / `--kube-reserved` will be accepted
again and subtracted from `Allocatable`. No state is persisted beyond the flag
values.

###### Are there any tests for feature enablement/disablement?

Unit tests will verify that hugepages keys are rejected when the feature gate
is disabled and accepted when enabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout cannot impact running workloads because the feature only changes
`Allocatable` reporting. Already-running pods that have hugepages allocated will
continue to function. The change only affects future scheduling decisions.

A rollback (disabling the feature gate) while hugepages entries are still in
`--system-reserved` will cause kubelet to reject the configuration on restart.
Administrators must remove the entries first.

###### What specific metrics should inform a rollback?

If pods fail to schedule due to insufficient hugepages despite the node having
enough total hugepages to accommodate both system daemons and pod requests,
the reservation values may be misconfigured. Verify the `--system-reserved` /
`--kube-reserved` hugepages values match the actual system daemon consumption.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Will be tested during beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Check the node's `status.allocatable` for hugepages resources and compare with
`status.capacity`. If they differ, hugepages reservation is active.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: `node.status.allocatable[hugepages-<size>]` reflects
    `capacity - system-reserved - kube-reserved`.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No new SLOs. This feature affects node status reporting, which is covered by
existing kubelet SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Other (treat as last resort)
  - Details: Compare `node.status.capacity[hugepages-<size>]` with
    `node.status.allocatable[hugepages-<size>]`. The difference should equal
    the configured reservation.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The existing node capacity and allocatable fields provide sufficient
observability.

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

No. The node status already includes hugepages in capacity and allocatable.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Kubelet computes `Allocatable` locally, but it needs the API server to update
the node status. If the API server is unavailable, the node status will not
reflect the reserved hugepages until connectivity is restored. This is the same
behavior as existing resource reservations (cpu, memory).

###### What are other known failure modes?

- Misconfigured hugepages reservation exceeding node capacity
  - Detection: Kubelet will fail to start with a validation error.
  - Mitigations: Fix the `--system-reserved` / `--kube-reserved` values.
  - Diagnostics: Kubelet logs will contain the validation error message.
  - Testing: Unit tests cover this case.

###### What steps should be taken if SLOs are not being met to determine the problem?

This feature does not introduce new SLOs. It modifies a static value on the
node status at kubelet startup. If the node's `Allocatable` hugepages value
is incorrect, check the `--system-reserved` and `--kube-reserved` flag values.

## Implementation History

- 2024-04-18: Prior implementation PR [kubernetes/kubernetes#124357](https://github.com/kubernetes/kubernetes/pull/124357) opened.
- 2024-10-15: PR closed as stale.
- 2026-07-20: KEP created.

## Drawbacks

This feature extends the accepted resource types for existing flags, which
increases the surface area for misconfiguration. However, the existing
validation mechanisms (`validateNodeAllocatable`) already handle this.

## Alternatives

N/A
