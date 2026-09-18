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
  - [Part 2: deriving the bound](#part-2-deriving-the-bound)
  - [Part 3: operator option](#part-3-operator-option)
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
configuration changes. The bound is derived from the size of the running kernel
image, and a memory manager policy option can disable the tolerance or pin the
bound.

The mechanism is Linux-specific: the drift comes from Linux kernel behavior and
the bound is read from `/proc/iomem`. It is validated on x86-64, where all the
reports so far come from. arm64 is expected to behave the same (the kernel image
is placed randomly by the EFI stub and `/proc/iomem` carries the same entries)
and will be confirmed during alpha. On every other platform, including Windows,
the memory manager keeps its exact comparison and this feature is a no-op.

## Motivation

### Background: why per-node MemTotal moves across a reboot

Two independent effects, at very different scales, both change the per-node
total that the policy checks. Kernel references below are to Linux 6.12 on
x86-64; the mechanisms are unchanged since KASLR became the default.

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

- The memory manager `Static` policy starts after a reboot when the only change
  is a benign per-node memory drift, without operator intervention.
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
- Platforms other than Linux: the exact comparison stays as it is. This also
  leaves out the `BestEffort` policy, which exists only on Windows.

## Proposal

Three pieces, matching the shape outlined on the tracking issue:

1. **Tolerate a bounded drift and re-baseline.**
2. **Derive the bound** from the kernel image size when the memory manager is
   initialized; when it cannot be derived, keep the exact comparison.
3. **A memory manager policy option** (`memoryManagerPolicyOptions`, a new
   `KubeletConfiguration` field that mirrors `cpuManagerPolicyOptions` and
   `topologyManagerPolicyOptions`) to disable the tolerance or pin the bound.

kubernetes/kubernetes#140473 (parts 1-2) and kubernetes/kubernetes#142121
(part 3) are reference implementations that ground the discussion; the design
in this document is what counts.

### User Stories

- As a cluster administrator running the memory manager `Static` policy, when a
  reboot (a kernel update, a power event) happens to move the memory a NUMA node
  reports, which is the case on some reboots and not others, I want kubelet to
  come back `Ready` on its own. Today it fails until someone deletes
  `/var/lib/kubelet/memory_manager_state` on that node, and that workaround is
  the problem.
- As a cluster administrator, when a node really loses memory (a failed DIMM, a
  changed `systemReserved`) I still want kubelet to refuse to start on the stale
  state, so that pods with pinned memory are not silently under-served.
- As a cluster administrator of a strictly controlled fleet, I want to keep the
  exact comparison, or decide myself how much drift is acceptable, through the
  memory manager policy options.

### Risks and Mitigations

- *Tolerating a real change.* Mitigated by keeping `systemReserved`, hugepages,
  the assignment structure and the group-reserved sum exact, by bounding the
  drift, and by the conservation check (an assignment that no longer fits still
  fails the start).
- *Wrong derived bound.* Mitigated by a grace on top of the image size and by
  not guessing: when the size cannot be read the tolerance stays off and the
  start keeps the exact comparison. The policy option is the final override.

## Design Details

### Part 1: bounded tolerance, conservation and re-baseline

In `validateState` (kubelet v1.37, `pkg/kubelet/cm/memorymanager/policy_static.go`),
when `areMachineStatesEqual` fails, accept the difference iff the states differ
only within the tolerated drift: identical topology (nodes,
NUMA grouping, assignment count), identical `SystemReserved`, hugepage totals
exact, and the regular-memory `TotalMemSize`/`Allocatable` within the bound per
node. Per-node `Reserved` is not compared - a drift can legitimately reshuffle a
cross-NUMA assignment's split while the group total is unchanged, and the
assignments are re-derived from the persisted blocks. `updateExpectedMachineState`
returns an error when a recorded assignment no longer fits, so a reduction that
would under-serve a pod still fails regardless of the bound. On success the policy
re-baselines with `SetMachineState(expected)`. When no tolerance is in effect
(the bound could not be derived, or the option turned it off) the existing error
is returned unchanged.

### Part 2: deriving the bound

When the memory manager is initialized at kubelet start (the static policy
object is created), derive the bound from the running kernel image: read the
`Kernel code`/`Kernel data`/`Kernel bss` lines of `/proc/iomem`, take the physical
span from the lowest start to the highest end (the image is contiguous; this
captures rodata/alignment gaps), and add a 64 MiB grace for the KiB-scale
secondary drift and rounding. The bound is recomputed on every start from the
kernel that actually booted, so it follows kernel upgrades by construction.

The evidence behind the grace is small but consistent: 12 KiB on the production
node that motivated this KEP, tens of KiB in the reports on
kubernetes/kubernetes#131253, and a mechanism (boot-time allocations whose size
depends on the randomized layout) that is KiB to low-MiB by construction, so
64 MiB leaves about three orders of magnitude of headroom. The observed-drift
metric exists to check this in the field during alpha; if the numbers come in
higher, the grace changes before beta, and the explicit option covers any single
node in the meantime.

Whether KASLR is enabled does not need to be detected: the image size bounds the
possible shift either way, and with KASLR off the image never moves, so the
tolerance is simply never exercised.

`/proc/iomem` is world-readable, but since Linux 4.6 it shows zeroed addresses
to readers without `CAP_SYS_ADMIN`; kubelet runs with it. If the entries are
absent (another platform or architecture) or zeroed, the size is unknown and the
policy does not guess: the tolerance stays off, the start keeps the exact
comparison, and a log line names the `memory-drift-tolerance` option as the way
to set the bound explicitly. The read is in kubelet, not cAdvisor: the value is
boot-static so cAdvisor's collection loop adds nothing, and kubelet and cAdvisor
already read procfs (`/proc/meminfo`, `/proc/cpuinfo`), so this is not a new
kind of dependency.

On the reported single-NUMA node this yields ~119 MiB (55 MiB image + 64 MiB
grace), which tolerates the observed 12 KiB drift by ~4 orders of magnitude and
stays far below a memory bank, so a real change is still caught.

### Part 3: operator option

Add a `memoryManagerPolicyOptions` map to the kubelet configuration (mirroring
`cpuManagerPolicyOptions` / `topologyManagerPolicyOptions`) with one option,
`memory-drift-tolerance`: `auto` (default: the bound derived from the kernel
image, or the exact comparison when it cannot be derived), `off` (the exact
comparison) or an explicit quantity such as `128Mi`. Unknown options and values
are rejected at kubelet start, as for the other managers. This is how an
operator keeps the exact comparison, or sets the bound on a platform where it
cannot be derived. `auto` is the default only once the feature itself is
enabled; an administrator who has not enabled it sees no change.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None. The existing Memory Manager policy tests already cover the `validateState`
start-up path this enhancement extends.

##### Unit tests

Part 1 is implemented with unit tests in kubernetes/kubernetes#140473; the gate
and parts 2-3 add tests in the same package.

- `k8s.io/kubernetes/pkg/kubelet/cm/memorymanager`: `2026-09-15` - `86.9%`
  (`go test -cover` on master); the start-up validation path this enhancement
  extends is exercised by `TestStaticPolicyStart` and
  `TestMemoryManagerRestoreState`.

Cases (added / planned):
- policy: a small drift with no assignments; a small drift where assignments still
  fit; a reduction that no longer fits (start fails); a drift above the bound
  (start fails); a cross-NUMA assignment whose split shifts under drift; a manager
  restart with a drifted `machineInfo`.
- feature gate: the same drifted state fails with the existing error when the
  gate is disabled, and is tolerated and re-baselined when it is enabled.
- derived bound: parse the kernel span from a captured `/proc/iomem` (including
  the rodata gap); reject zeroed / absent addresses; with no readable size the
  start keeps the exact comparison.
- option: `off` rejects any drift; an explicit bound is honored; an option set
  without the gate is rejected by configuration validation.
- metrics: the tolerance in effect and the observed per-node drift are exported
  after a start.

##### Integration tests

None planned. The behavior is kubelet node-local at start-up with no control-plane
configuration to exercise; unit tests and node e2e cover it.

##### e2e tests

- A node e2e (`test/e2e_node`, serial, `Feature:MemoryManagerDriftTolerance`),
  targeted at beta: with the `Static` policy running, stop kubelet, rewrite the
  persisted per-node `TotalMemSize` in the `memory_manager_state` checkpoint by a
  few MiB through the state package (the e2e environment cannot change the
  machine's real `MemTotal`), restart kubelet and assert it comes up `Ready` with
  the assignments restored; the same edit with the gate disabled reproduces the
  existing start failure.
- Optionally document the `nokaslr` boot option as the confirming experiment for
  the KASLR factor (diagnostic only, not a fix).

### Graduation Criteria

#### Alpha

- Parts 1-3 (bounded tolerance with conservation and re-baseline; derived
  bound; `memoryManagerPolicyOptions` with `memory-drift-tolerance`) implemented
  behind the `MemoryManagerDriftTolerance` feature gate, disabled by default.
- The `kubelet_memory_manager_drift_tolerance_bytes` and
  `kubelet_memory_manager_memory_drift_bytes` metrics.
- Unit tests for the tolerated, rejected, gate-disabled and option paths.

#### Beta

- Feature gate enabled by default.
- arm64 confirmed.
- The node e2e test in place and passing in the sig-node periodic jobs.
- Feedback from users affected by kubernetes/kubernetes#131253; no open
  correctness issues.

#### GA

- At least two releases in beta with no regressions and no reported false
  tolerations; the gate is locked to enabled (removal follows the feature gate
  lifecycle).

### Upgrade / Downgrade Strategy

No configuration change is required to keep working: a kubelet with the feature
tolerates benign drift automatically, and an operator who wants the exact
comparison sets the policy option. The checkpoint format is unchanged, so there
is no state migration. On downgrade the kubelet reverts to the exact comparison
(and the pre-existing reboot failure can reappear).

### Version Skew Strategy

This is a node-local kubelet behavior with no control-plane coordination and no
change to the checkpoint format, CRI, CNI or CSI. Nodes without the feature keep
the old strict behavior. There are no skew concerns.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `MemoryManagerDriftTolerance`
  - Components depending on the feature gate: kubelet

###### Does enabling the feature change any default behavior?

Yes. The memory manager `Static` policy tolerates a bounded per-node memory
drift on start and re-baselines, instead of failing. Genuine hardware/configuration changes and non-fitting
assignments still fail as before.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the gate restores the strict per-node equality check. It only
affects start-time state validation, so there is no impact on running workloads.
`disable-supported: true`.

###### What happens if we reenable the feature if it was previously rolled back?

The tolerant validation applies again on the next kubelet start; no persisted
state depends on the gate.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests in `pkg/kubelet/cm/memorymanager` start the policy on a drifted
checkpoint with the gate disabled (the existing error is returned) and enabled
(the drift is tolerated and re-baselined), using
`featuregatetesting.SetFeatureGateDuringTest`. Beta adds the node e2e described
in the test plan.

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

Not applicable: the feature changes no API and no on-disk format. Enabling the
gate only changes how an unchanged state file is validated at start, and
disabling it restores the exact comparison on the same file, so there is nothing
to migrate in either direction.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

It is node-level, not workload-level. `kubelet_memory_manager_drift_tolerance_bytes`
above zero means the tolerance is in effect on that node, and
`kubelet_memory_manager_memory_drift_bytes{numa_node}` shows the drift observed
at the last start. Kubelet logs carry a re-baseline message when a benign drift
is tolerated.

###### How can someone using this feature know that it is working for their instance?

- [x] Metrics
  - Metric name: `kubelet_memory_manager_drift_tolerance_bytes` (the bound in
    effect, 0 when the exact comparison applies) and
    `kubelet_memory_manager_memory_drift_bytes{numa_node}` (the drift observed
    at the last start)
  - Components exposing the metric: kubelet
- [x] Other (treatment): a benign drift is logged as tolerated and re-baselined
  at start; a drift beyond the bound still fails with the existing error.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No SLO change is expected: the feature affects only the kubelet start path,
with negligible overhead (one `/proc/iomem` read and a per-node comparison).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_memory_manager_drift_tolerance_bytes`,
    `kubelet_memory_manager_memory_drift_bytes`
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None beyond the two above.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No cluster services. On Linux it reads `/proc/iomem` to size the bound; when
that is unavailable the tolerance stays off, so there is no hard dependency.

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

If the kernel image size cannot be read from `/proc/iomem` (addresses hidden
without `CAP_SYS_ADMIN`, or a platform without the entries), the tolerance stays
off and the start keeps the exact comparison; this is logged at start and
`kubelet_memory_manager_drift_tolerance_bytes` reads 0. Setting
`memory-drift-tolerance` explicitly enables it.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect the kubelet log at start for the memory manager validation messages, and
compare the persisted `/var/lib/kubelet/memory_manager_state` totals against the
current `/sys/.../nodeN/meminfo`.

## Implementation History

- 2026-07-12: bug reported / fix opened (kubernetes/kubernetes#140473, part 1).
- 2026-07-13: root cause (KASLR + boot-reserved drift) analyzed on the tracking
  issue (#131253); KEP drafted (kubernetes/enhancements#6233).
- 2026-09-08: opted into v1.38 by sig-node (alpha).
- 2026-09-15: KEP marked `implementable` for v1.38.
- 2026-09-16: first sig-node review pass; the fallback now keeps the exact
  comparison instead of a default bound, metrics moved to alpha, platform scope
  stated.

## Drawbacks

Adds a bounded heuristic to a path that was a strict equality, and a small
platform-specific `/proc/iomem` read.

## Alternatives

- *Fixed absolute bound (no derivation).* Simpler, but the right value is
  per-kernel, not one global constant.
- *A default bound when the size cannot be read.* Rejected in review: a guessed
  bound can hide a real change; keeping the exact comparison and letting the
  administrator set the bound is safer.
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
