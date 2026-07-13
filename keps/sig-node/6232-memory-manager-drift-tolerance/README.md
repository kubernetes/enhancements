# KEP-6232: Tolerate benign per-NUMA memory drift in the Memory Manager

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Background: why per-node MemTotal moves across a reboot](#background-why-per-node-memtotal-moves-across-a-reboot)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Part 1: bounded tolerance, conservation and re-baseline](#part-1-bounded-tolerance-conservation-and-re-baseline)
  - [Part 2: auto-detecting the bound](#part-2-auto-detecting-the-bound)
  - [Part 3: operator option](#part-3-operator-option)
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
- [Infrastructure Needed](#infrastructure-needed)
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
- [ ] User-facing documentation has been created in [kubernetes/website]
- [ ] Supporting documentation

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

The Memory Manager `Static` policy records the total memory of each NUMA node in
its checkpoint and, on kubelet start, requires the recorded per-node totals to
match the current machine exactly. That total (cAdvisor's
`/sys/.../nodeN/meminfo` `MemTotal`) is not stable across reboots, so kubelet
crash-loops on a benign change with `the expected machine state is different
from the real one`, and the node stays `NotReady` until an operator deletes the
checkpoint by hand.

This KEP makes the policy tolerate a bounded, benign per-node memory drift and
re-baseline onto the current machine, while still failing on genuine hardware or
configuration changes. The bound is auto-detected from the running kernel image
size, and an operator option can disable or override it.

## Motivation

### Background: why per-node MemTotal moves across a reboot

Two independent effects, at very different scales, both change the per-node
total that the policy checks.

**1. KASLR relocation of the kernel image (major, MB-scale).** The physical half
of `CONFIG_RANDOMIZE_BASE` (KASLR) chooses a random, 2 MiB-aligned base for the
kernel image across all usable RAM on every boot (`find_random_phys_addr()` in
`arch/x86/boot/compressed/kaslr.c` scans the whole e820/EFI map, not just node
0). The image is `memblock_reserve`d at that address (`__pa_symbol(_text)`), and
per-node `MemTotal` is the sum of `zone_managed_pages` = present - reserved, so
the reservation subtracts from whichever NUMA node owns those physical addresses.
Between boots the image lands on a different node: one node gains ~the image
size, another loses it, and the sum across nodes stays constant. The magnitude
equals the resident image size (`text+rodata+data+bss`; `.init.*` is freed before
the policy reads meminfo), typically tens of MiB, up to ~130 MiB for a large
distro kernel. This needs more than one NUMA node to be visible.

Observed in #131253: a bare-metal node moves exactly 12130 pages off node0 /
12131 onto node1 (~47.4 MiB), and that machine's dmesg reports a ~49 MiB resident
image (essentially the whole image relocating); a VM case shows +41.67 MiB /
-41.67 MiB with a bit-for-bit constant sum.

**2. Variable boot-time reserved-memory freeing (minor, KiB-scale).**
Independently of KASLR, the amount of memory the kernel frees during boot varies
slightly between boots, so the *total* drifts by tens of KiB, which hits even
single-NUMA nodes. Observed on a production single-NUMA node: kubelet
crash-looped with `TotalMemSize1=65839165440` vs `TotalMemSize2=65839153152`, a
12288-byte (3-page) difference with `systemReserved` unchanged.

Both effects have existed since KASLR became default-on across distros
(~2017-2018), which predates the Memory Manager (alpha 1.21, 2021). The strict
per-node equality check never accounted for them, which is why the failure
surfaces on ordinary reboots.

### Goals

- The `Static`/`BestEffort` policies start after a reboot when the only change is
  a benign per-node memory drift, without operator intervention.
- Genuine changes still fail the start: a change to `systemReserved`/reserved
  memory, a hugepage change, an added/removed memory bank (GiB-scale), or an
  assignment that no longer fits.
- The tolerated bound is principled (derived from the running kernel), not a
  hand-picked constant.
- Operators can opt out of, or override, the behavior.

### Non-Goals

- Changing how memory is allocated to pods, or the checkpoint format.
- Eliminating the fluctuation itself (a kernel/firmware concern).
- Covering the `None` policy (it does not validate machine state).

## Proposal

Three pieces, matching the shape outlined on the tracking issue:

1. **Tolerate a bounded drift and re-baseline** (implemented in
   kubernetes/kubernetes#140473).
2. **Auto-detect the bound** from the kernel image size.
3. **An operator option** to disable the tolerance or set an explicit bound.

### User Stories

- *Routine reboot:* a node reboots (or finishes an OS update) and kubelet comes
  back `Ready` without anyone deleting `/var/lib/kubelet/memory_manager_state`.
- *Strict environment:* an operator who wants the previous strict behavior, or a
  specific bound, sets a policy option.

### Risks and Mitigations

- *Tolerating a real change.* Mitigated by keeping `systemReserved`, hugepages,
  the assignment structure and the group-reserved sum exact, by bounding the
  drift, and by the conservation check (an assignment that no longer fits still
  fails the start).
- *Wrong auto-detected bound.* Mitigated by a generous grace and a conservative
  fallback when `/proc/iomem` is unreadable; the operator option is the final
  override.

## Design Details

### Part 1: bounded tolerance, conservation and re-baseline

In `validateState`, when `areMachineStatesEqual` fails, accept the difference iff
the states differ only within the tolerated drift: identical topology (nodes,
NUMA grouping, assignment count), identical `SystemReserved`, hugepage totals
exact, and the regular-memory `TotalMemSize`/`Allocatable` within the bound per
node. Per-node `Reserved` is not compared - a drift can legitimately reshuffle a
cross-NUMA assignment's split while the group total is unchanged, and the
assignments are re-derived from the persisted blocks. `updateExpectedMachineState`
returns an error when a recorded assignment no longer fits, so a reduction that
would under-serve a pod still fails regardless of the bound. On success the policy
re-baselines with `SetMachineState(expected)`. Implemented and tested in #140473.

### Part 2: auto-detecting the bound

At policy construction, derive the bound from the running kernel image: read the
`Kernel code`/`Kernel data`/`Kernel bss` lines of `/proc/iomem`, take the physical
span from the lowest start to the highest end (the image is contiguous; this
captures rodata/alignment gaps), and add a grace for the KiB-scale secondary drift
and rounding. The span gives both the movable size and (via the code start and
`/sys/devices/system/memory/block_size_bytes`) the current node.

Fallbacks: `/proc/iomem` hides its addresses without `CAP_SYS_ADMIN` (a
KASLR-leak mitigation); kubelet has the capability, but if the addresses read as
zero, or the file is absent (non-Linux), fall back to a conservative default. The
read is in kubelet, not cAdvisor: the value is boot-static so cAdvisor's
collection loop adds nothing, and cAdvisor exposes only `nodeN/meminfo MemTotal`
today, so it is net-new either way.

On the reported single-NUMA node this yields ~119 MiB (55 MiB image + 64 MiB
grace), which tolerates the observed 12 KiB drift by ~4 orders of magnitude and
is tighter than a fixed 256 MiB, so it detects real changes sooner.

### Part 3: operator option

Add a `memoryManagerPolicyOptions` map (mirroring `cpuManagerPolicyOptions` /
`topologyManagerPolicyOptions`), gated behind the `MemoryManagerDriftTolerance`
feature gate, with an option to disable the tolerance (restore strict behavior)
or set an explicit byte bound. The default keeps the auto-detected behavior.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None. The existing Memory Manager policy tests already cover the `validateState`
start-up path this enhancement extends.

##### Unit tests

Part 1 is implemented with unit tests in kubernetes/kubernetes#140473; parts 2-3
add tests in the same package.

- `k8s.io/kubernetes/pkg/kubelet/cm/memorymanager`: `2026-07-13` - covered by the
  existing policy suite plus the cases below.

Cases (added / planned):
- policy: a small drift with no assignments; a small drift where assignments still
  fit; a reduction that no longer fits (start fails); a drift above the bound
  (start fails); a cross-NUMA assignment whose split shifts under drift; a manager
  restart with a drifted `machineInfo`.
- autodetect: parse the kernel span from a captured `/proc/iomem` (including the
  rodata gap); reject zeroed / absent addresses; fall back on a missing file.
- option: strict mode rejects any drift; an explicit bound is honored.

##### Integration tests

None planned. The behavior is kubelet node-local at start-up with no control-plane
configuration to exercise; unit tests and node e2e cover it.

##### e2e tests

- A node-e2e that writes a Memory Manager checkpoint, restarts kubelet with a
  slightly different per-NUMA `MemTotal`, and asserts the node returns `Ready`
  instead of crash-looping.
- Optionally document the `nokaslr` boot option as the confirming experiment for
  the KASLR factor (diagnostic only, not a fix).

### Graduation Criteria

- *Alpha:* parts 1-2 behind the Memory Manager machinery; the option (part 3)
  behind the `MemoryManagerDriftTolerance` feature gate, default off.
- *Beta:* option on by default; e2e_node coverage; no open correctness issues.
- *GA:* soak across releases with no regressions.

### Upgrade / Downgrade Strategy

No configuration change is required to keep working: a kubelet with the feature
tolerates benign drift automatically, and an operator who wants the previous
strict behavior sets the opt-out option. The checkpoint format is unchanged, so
there is no state migration. On downgrade the kubelet reverts to strict equality
(and the pre-existing reboot failure can reappear).

### Version Skew Strategy

This is a node-local kubelet behavior with no control-plane coordination and no
change to the checkpoint format, CRI, CNI or CSI. Nodes without the feature keep
the old strict behavior. There are no skew concerns.

## Production Readiness Review Questionnaire

<!-- Completed for `implementable`; provisional answers below. -->

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `MemoryManagerDriftTolerance`
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

Yes. The `Static` policy tolerates a bounded per-node memory drift on start and
re-baselines, instead of failing. Genuine hardware/configuration changes and
non-fitting assignments still fail as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate restores the strict per-node equality check. It only
affects start-time state validation, so there is no impact on running workloads.
`disable-supported: true`.

###### What happens if we reenable the feature if it was previously rolled back?

The tolerant validation applies again on the next kubelet start; no persisted
state depends on the gate.

###### Are there any tests for feature enablement/disablement?

Unit tests exercise the tolerant and strict paths; option tests will cover the
enable/disable switch.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature only changes the Memory Manager's start-time state validation on a
node. It cannot impact already running workloads; the worst case on rollback is
the pre-existing behavior this KEP fixes (kubelet failing to start on a benign
drift).

###### What specific metrics should inform a rollback?

An increase in kubelet start failures with
`the expected machine state is different from the real one`, or nodes going
`NotReady` after reboot under the `Static` memory manager policy.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Covered by unit tests exercising the gate on/off. The checkpoint format is
unchanged, so the upgrade/downgrade/upgrade path does not migrate state.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

It is node-level, not workload-level. Its effect is visible in kubelet logs (a
re-baseline message when a benign drift is tolerated) and in the node staying
`Ready` after a reboot.

###### How can someone using this feature know that it is working for their instance?

- [x] Other (treatment): a benign drift is logged as tolerated and re-baselined
  at start; a drift beyond the bound still fails with the existing error. A
  counter (`memory_manager_drift_tolerated_total`, proposed) will make this
  observable via metrics.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Node startup succeeds after a benign reboot drift, with no change to allocation
behavior.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `memory_manager_drift_tolerated_total` (proposed)
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

The proposed counter above.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No cluster services. On Linux it reads `/proc/iomem` to size the bound and falls
back to a conservative default when that is unavailable, so there is no hard
dependency.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No (aside from the opt-in policy option, which is node configuration, not a
cluster API object).

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. It adds a single boot-time read of `/proc/iomem` at Memory Manager start.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Unaffected; it is entirely node-local at kubelet start.

###### What are other known failure modes?

If `/proc/iomem` is unreadable (addresses hidden without `CAP_SYS_ADMIN`, or a
non-Linux node), the bound falls back to a conservative default; behavior stays
correct, only the bound is less tight, and this is logged at start.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect the kubelet log at start for the memory manager validation messages, and
compare the persisted `/var/lib/kubelet/memory_manager_state` totals against the
current `/sys/.../nodeN/meminfo`.

## Implementation History

- 2026-07-12: bug reported / fix opened (kubernetes/kubernetes#140473, part 1).
- 2026-07-13: root cause (KASLR + boot-reserved drift) analyzed on the tracking
  issue (#131253); KEP drafted.

## Drawbacks

Adds a bounded heuristic to a path that was a strict equality, and a small
platform-specific `/proc/iomem` read.

## Alternatives

- *Fixed absolute bound (no autodetect).* Simpler, but the right value is
  per-kernel, not one global constant.
- *Fraction of node RAM.* Rejected: the kernel image does not scale with RAM, so
  a fraction would tolerate multi-GiB changes on large machines and hide a real
  memory loss.
- *Extend cAdvisor to report the bound.* Cleaner layering but net-new API plus a
  cross-repo revendor cycle for a single, boot-static consumer.
- *Sum-only comparison.* Insufficient: on bare metal the sum itself drifts, and a
  whole-image relocation is not a sum change.
- *`nokaslr`.* Not acceptable as a fix (weakens security); useful only to confirm
  the root cause.

## Infrastructure Needed

None.
