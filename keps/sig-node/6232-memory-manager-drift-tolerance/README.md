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
configuration changes. The tolerance is off by default and is turned on with a
memory manager policy option, which derives the bound from the size of the
running kernel image or pins it to a value. That shape follows the other kubelet
policy options: the existing behavior stays the default until an administrator
asks for the new one, and the option is also the only way to set the bound on a
platform where it cannot be derived. Whether `auto` becomes the default once the
feature is on by default is a beta decision, taken on the drift data gathered
during alpha.

The mechanism is Linux-specific: the drift comes from Linux kernel behavior and
the bound is read from `/proc/iomem`. It is validated on x86-64, where all the
reports so far come from. arm64 is expected to behave the same (the kernel image
is placed randomly by the EFI stub and `/proc/iomem` carries the same entries)
and will be confirmed during alpha. On platforms other than Linux, including
Windows, the memory manager keeps its exact comparison, and the `None` and
`BestEffort` policies reject the option.

## Motivation

### Background: why per-node MemTotal moves across a reboot

Two independent effects, at very different scales, both change the per-node
total that the policy checks. Kernel references below are to Linux 6.12 on
x86-64. The mechanisms are unchanged since KASLR became the default.

**1. KASLR relocation of the kernel image (major, MB-scale).** The physical half
of `CONFIG_RANDOMIZE_BASE` (KASLR) chooses a random, 2 MiB-aligned base for the
kernel image across all usable RAM on every boot (`find_random_phys_addr()` in
`arch/x86/boot/compressed/kaslr.c` scans the whole e820/EFI map, not just node
0). The image is `memblock_reserve`d at that address (`__pa_symbol(_text)`), and
per-node `MemTotal` is the sum of `zone_managed_pages` = present - reserved, so
the reservation subtracts from whichever NUMA node owns those physical addresses.
Between boots the image can land on a different node: one node gains ~the image
size, another loses it, and the sum across nodes stays constant. The magnitude
equals the resident image size (`text+rodata+data+bss`, since `.init.*` is freed
before the policy reads meminfo), typically tens of MiB, up to ~130 MiB for a
large distro kernel. This needs more than one NUMA node to be visible.

Observed in #131253: a bare-metal node moves exactly 12130 pages off node0 /
12131 onto node1 (~47.4 MiB), and that machine's dmesg reports a ~49 MiB resident
image (essentially the whole image relocating). A VM case shows +41.67 MiB /
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
- The bound is derived from the running kernel rather than set as a fixed
  constant.
- Operators can keep the exact comparison, or set the bound themselves where it
  cannot be derived.

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
   initialized. When it cannot be derived, keep the exact comparison.
3. **A memory manager policy option** (`memoryManagerPolicyOptions`, a new
   `KubeletConfiguration` field shaped like `topologyManagerPolicyOptions`)
   that turns the tolerance on and can pin the bound. The exact comparison stays
   the default even with the feature gate enabled: enabling is a double opt-in,
   the gate makes the option available and the option turns the tolerance on,
   as for the other kubelet policy options.

### User Stories

- As a cluster administrator running the memory manager `Static` policy, when a
  reboot (a kernel update, a power event) happens to move the memory a NUMA node
  reports, which is the case on some reboots and not others, I want to set
  `memory-drift-tolerance: auto` once and have kubelet come back `Ready` on its
  own. Today kubelet fails until someone deletes
  `/var/lib/kubelet/memory_manager_state` on that node.
- As a cluster administrator, when a node really loses memory (a failed DIMM, a
  changed `systemReserved`) I still want kubelet to refuse to start on the stale
  state, so that pods with pinned memory are not silently under-served.
- As a cluster administrator of a strictly controlled fleet, I want nothing to
  change unless I ask for it: the exact comparison stays until I set the option,
  and I can pin exactly how much drift is acceptable.

### Risks and Mitigations

- *Tolerating a real change.* Mitigated by keeping `systemReserved`, hugepages,
  the assignment structure and the group-reserved sum exact, by bounding the
  drift, and by the conservation check (an assignment that no longer fits still
  fails the start).
- *Wrong derived bound.* Mitigated by a grace on top of the image size and by
  not guessing: when the size cannot be read the tolerance stays off and the
  start keeps the exact comparison. The policy option is the only switch and the
final override.

## Design Details

### Part 1: bounded tolerance, conservation and re-baseline

In `validateState` (kubelet v1.37,
`pkg/kubelet/cm/memorymanager/policy_static.go`), when `areMachineStatesEqual`
fails, accept the difference iff the states differ only within the tolerated
drift: identical topology (nodes, NUMA grouping, assignment count), identical
`SystemReserved`, hugepage totals exact, every recorded memory table internally
consistent (`Free + Reserved == Allocatable`), the reserved memory of each NUMA
group equal to what the assignments need, and the regular-memory `TotalMemSize`
within the bound per node, with `Allocatable` moving by the same amount.
Per-node `Reserved` is not compared - a drift can legitimately reshuffle a
cross-NUMA assignment's split while the group total is unchanged, and the
assignments are re-derived from the persisted blocks. If a recorded assignment
no longer fits the machine, the start fails. On success the policy logs the
drift it measured for each node and re-baselines with
`SetMachineState(expected)`. When no tolerance is in effect (the option is not
set, or it is `off`, or the bound could not be derived) the existing start path
runs unchanged, with the existing error.

### Part 2: deriving the bound

When the memory manager is initialized at kubelet start (the static policy
object is created) and the option asks for `auto`, derive the bound from the
running kernel image: read the `Kernel code`/`Kernel data`/`Kernel bss` lines of
`/proc/iomem`, take the physical span from the lowest start to the highest end
(the image is contiguous, so the span includes the rodata and alignment gaps),
and add a 64 MiB grace for the KiB-scale secondary drift and rounding. The bound
is recomputed on every start from the kernel that actually booted, so it follows
kernel upgrades.

The evidence behind the grace is small but consistent: 12 KiB on the production
node that motivated this KEP, tens of KiB in the reports on
kubernetes/kubernetes#131253, and a mechanism (boot-time allocations whose size
depends on the randomized layout) that is KiB to low-MiB in size, so
64 MiB leaves about three orders of magnitude of headroom. The observed-drift
metric exists to check this in the field during alpha. If the numbers come in
higher, the grace changes before beta, and the explicit option covers any single
node in the meantime.

Whether KASLR is enabled does not need to be detected: the image size bounds the
possible shift either way, and with KASLR off the image never moves, so the
tolerance is simply never exercised.

`/proc/iomem` is world-readable, but since Linux 4.6 it shows zeroed addresses
to readers without `CAP_SYS_ADMIN`, and kubelet runs with it. If the entries are
absent (another platform or architecture) or zeroed, the size is unknown and the
policy does not guess: the tolerance stays off, the start keeps the exact
comparison, and kubelet logs that the bound can be set explicitly with
`memory-drift-tolerance`. The read is in kubelet, not cAdvisor: the value is
boot-static so cAdvisor's collection loop adds nothing, and kubelet and cAdvisor
already read procfs (`/proc/meminfo`, `/proc/cpuinfo`), so this is not a new
kind of dependency.

On the reported single-NUMA node this yields ~119 MiB (55 MiB image + 64 MiB
grace), which tolerates the observed 12 KiB drift by ~4 orders of magnitude and
stays far below a memory bank, so a real change is still caught.

### Part 3: operator option

Add a `memoryManagerPolicyOptions` map to the kubelet configuration, shaped like
`topologyManagerPolicyOptions`: a map on `KubeletConfiguration`, parsed into a
policy options struct when the memory manager is created, with an availability
check per option. One option for now, `memory-drift-tolerance`:

- unset (the default): the exact comparison, regardless of the feature gate, so
  an upgrade or enabling the gate alone changes nothing.
- `auto`: the bound derived from the kernel image, or the exact comparison when
  it cannot be derived.
- a positive quantity such as `128Mi`: that bound, on any Linux architecture.
- `off`: the exact comparison.

A zero quantity is rejected. Use `off` to keep the exact comparison.

The option is alpha and requires the `MemoryManagerDriftTolerance` gate. It
graduates with the gate rather than through separate alpha/beta option gates.
Unknown options and values are rejected at kubelet start, as for the other
managers, and the option is rejected for the `None` and `BestEffort` policies.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None. The existing Memory Manager policy tests already cover the `validateState`
start-up path this enhancement extends.

##### Unit tests

- `k8s.io/kubernetes/pkg/kubelet/cm/memorymanager`: `2026-09-15` - `86.9%`
  (`go test -cover` on master). The start-up validation path this enhancement
  extends is exercised by `TestStaticPolicyStart` and
  `TestMemoryManagerRestoreState`.

Cases:
- policy: a small drift with no assignments, a small drift where assignments
  still fit, a reduction that no longer fits (start fails), a drift above the
  bound (start fails), a cross-NUMA assignment whose split shifts under drift,
  and a manager restart with a drifted `machineInfo`.
- feature gate: the same drifted state fails with the existing error when the
  gate is disabled, and is tolerated and re-baselined when the gate is enabled
  and a bound is set.
- BestEffort, which shares the start-up validation: a restart with an assignment
  larger than its NUMA node still starts as before.
- derived bound: the kernel span is parsed from a sample `/proc/iomem`,
  including the rodata gap, zeroed or absent addresses are rejected, and with no
  readable size the start keeps the exact comparison.
- option: unset and `off` keep the exact comparison, `auto` derives the bound,
  an explicit bound is honored, and a zero bound, an option set without the
  gate, or an option for a policy other than `Static` is rejected.
- metrics: the tolerance in effect and the observed per-node drift are exported
  after a start.

##### Integration tests

None planned. The behavior is kubelet node-local at start-up with no control-plane
configuration to exercise. Unit tests and node e2e cover it.

##### e2e tests

- Serial node e2e tests start the `Static` policy with the gate and the option
  enabled through the kubelet configuration.
  - After a fresh start, the tolerance and a zero drift for every NUMA node are
    reported.
  - kubelet restarted with `memory-drift-tolerance: auto` comes back with the
    memory assignment of a running pod kept.
  - A small drift written into the recorded state while kubelet is stopped is
    tolerated: kubelet starts, re-baselines the state, keeps the assignment and
    reports the drift. The test changes the recorded state because an e2e node
    cannot change its real memory.

### Graduation Criteria

#### Alpha

- Parts 1-3 (bounded tolerance with conservation and re-baseline, derived
  bound, `memoryManagerPolicyOptions` with `memory-drift-tolerance`) implemented
  behind the `MemoryManagerDriftTolerance` feature gate, disabled by default.
  The tolerance itself stays off until the option is set.
- The `kubelet_memory_manager_drift_tolerance_bytes` and
  `kubelet_memory_manager_drift_bytes` metrics.
- Unit tests for the tolerated, rejected, gate-disabled and option paths.
- Node e2e tests for a fresh start, a kubelet restart and a drift recorded in
  the checkpoint.

#### Beta

- Feature gate enabled by default.
- arm64 confirmed.
- The 64 MiB grace confirmed or refined from the observed-drift metric gathered
  during alpha.
- Decided, on the same data, whether `auto` becomes the default once the feature
  is on by default. `off` stays as the opt-out.
- A health SLI for the feature identified and documented in the PRR (kubelet
  start success after a reboot, `kubelet_pod_start_sli_duration_seconds`), on
  top of the informational gauges.
- The node e2e tests passing in the sig-node periodic jobs.
- Feedback from users affected by kubernetes/kubernetes#131253, with no open
  correctness issues.

#### GA

- At least two releases in beta with no regressions and no reported false
  tolerations. The gate is locked to enabled (removal follows the feature gate
  lifecycle).

### Upgrade / Downgrade Strategy

No configuration change is required to keep working, and nothing changes on
upgrade: the exact comparison stays until an administrator sets
`memory-drift-tolerance`. Removing the option, or downgrading, restores it (and
the pre-existing reboot failure can reappear). The checkpoint format is
unchanged, so there is no state migration.

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
- [x] Other
  - Describe the mechanism: the `memory-drift-tolerance` memory manager policy
    option. Both are required: the option is rejected without the gate, and the
    gate alone does not change the start validation.
  - Will enabling / disabling the feature require downtime of the control
    plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? A kubelet restart, as for any kubelet configuration change.

###### Does enabling the feature change any default behavior?

No. Enabling the gate makes the option available and registers the two gauges.
The exact comparison stays the default until `memory-drift-tolerance` sets a
bound. Once it does, the memory manager `Static` policy tolerates a bounded
per-node memory drift on start and re-baselines instead of failing. Genuine
hardware or configuration changes and non-fitting assignments still fail the
start.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Removing the option restores the exact comparison on the next kubelet start.
The gate can be disabled in the same configuration change or later. Disabling
the gate while the option is still set fails the kubelet start, as for
`topologyManagerPolicyOptions`. The feature only
affects start-time state validation, so there is no impact on running workloads.
`disable-supported: true`.

###### What happens if we reenable the feature if it was previously rolled back?

The tolerant validation applies again on the next kubelet start once the gate is
enabled and the option is set again. No persisted state depends on the gate or
the option.

###### Are there any tests for feature enablement/disablement?

Yes. Unit tests in `pkg/kubelet/cm/memorymanager` check that the option is
rejected while the gate is disabled. They also start the policy on a drifted
state: with the gate disabled or the option unset the existing error is
returned, and with a bound set the drift is tolerated and the state is
re-baselined. The node e2e tests enable the gate and the option.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature only changes the Memory Manager's start-time state validation on a
node and does not touch running containers. A rollout fails if the option is
rejected: a zero, negative or malformed value, an unknown option, the option
without the gate, or the option with the `None` or `BestEffort` policy stops the
kubelet start, and the node stays
`NotReady` until the configuration is fixed, as with any kubelet configuration
error. A rollback that disables the gate but leaves the option set fails the
same way. Otherwise the worst case on rollback is the pre-existing behavior this
KEP fixes, kubelet failing to start on a benign drift.

###### What specific metrics should inform a rollback?

A node under the `Static` memory manager policy that stays `NotReady` after a
reboot with the tolerance enabled: kubelet still failing to start with
`the expected machine state is different from the real one` (a drift beyond the
bound, or a node where the size could not be derived and the tolerance is off,
which kubelet logs at start) or with
`the memory assignment does not fit the machine state` (a real capacity loss,
reported while the tolerance is in effect). Apart from a rejected option, the
feature can only remove start failures, not add new ones, so the signal is a
node that does not come back rather than a new error.
`kubelet_memory_manager_drift_bytes` approaching
`kubelet_memory_manager_drift_tolerance_bytes` on a node is the early warning.
Admission and running pods are not affected: after a tolerated start the
re-baselined state is what a fresh start on that machine would produce.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not applicable: the feature adds an optional kubelet configuration field and
changes no on-disk format. Setting the option only changes how an unchanged
state file is validated at start, and removing it restores the exact comparison
on the same file, so there is nothing to migrate in either direction.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

It is node-level, not workload-level. `kubelet_memory_manager_drift_tolerance_bytes`
above zero means the tolerance is in effect on that node, and
`kubelet_memory_manager_drift_bytes{numa_node}` shows the drift observed
at the last start. Kubelet logs carry a re-baseline message when a benign drift
is tolerated.

###### How can someone using this feature know that it is working for their instance?

- [x] Metrics
  - Metric name: `kubelet_memory_manager_drift_tolerance_bytes` (the bound in
    effect, 0 when the exact comparison applies) and
    `kubelet_memory_manager_drift_bytes{numa_node}` (the drift observed
    at the last start)
  - Components exposing the metric: kubelet
- [x] Other (treatment): a benign drift is logged as tolerated and re-baselined
  at start. A drift beyond the bound still fails with the existing error.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No SLO change is expected: the feature affects only the kubelet start path,
with negligible overhead (one `/proc/iomem` read and a per-node comparison).

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kubelet_pod_start_sli_duration_seconds` and the node `Ready`
    condition after a reboot: a `Static`-policy node that comes back `Ready` and
    starts pods at its usual latency is the health signal.
    `kubelet_memory_manager_drift_tolerance_bytes` and
    `kubelet_memory_manager_drift_bytes` are informational: whether the
    tolerance is in effect and how far the node is from the bound.
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A feature-specific health SLI (kubelet start success rate after a reboot) is a
beta item, see the graduation criteria.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No cluster services. On Linux it reads `/proc/iomem` to size the bound. When
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

Unaffected. It is entirely node-local at kubelet start.

###### What are other known failure modes?

If the kernel image size cannot be read from `/proc/iomem` (addresses hidden
without `CAP_SYS_ADMIN`, or a platform without the entries), the tolerance stays
off and the start keeps the exact comparison. This is logged at start and
`kubelet_memory_manager_drift_tolerance_bytes` reads 0. Setting
`memory-drift-tolerance` explicitly enables it.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect the kubelet log at start for the memory manager validation messages, and
compare the persisted `/var/lib/kubelet/memory_manager_state` totals against the
current `/sys/.../nodeN/meminfo`.

## Implementation History

- 2026-07-12: bug reported / fix opened (kubernetes/kubernetes#140473, part 1).
- 2026-07-13: root cause (KASLR + boot-reserved drift) analyzed on the tracking
  issue (#131253), KEP drafted (kubernetes/enhancements#6233).
- 2026-09-08: opted into v1.38 by sig-node (alpha).
- 2026-09-15: KEP marked `implementable` for v1.38.
- 2026-09-16: first sig-node review pass: the fallback now keeps the exact
  comparison instead of a default bound, metrics moved to alpha, platform scope
  stated.
- 2026-09-22: third review pass: the tolerance is opt-in through the option
  even with the gate enabled, grace refinement moved to the beta criteria.
- 2026-09-26: PRR review. Rollback signals and SLIs reworded around node
  readiness and pod start latency, `auto` by default and a health SLI added to
  the beta criteria.
- 2026-11-06: code review of kubernetes/kubernetes#140473. The observed-drift
  metric is renamed, node e2e tests move to alpha, a zero tolerance is
  rejected, and the rollback order and failure modes in the PRR are corrected.

## Drawbacks

Adds a bounded heuristic to a path that was a strict equality, and a small
platform-specific `/proc/iomem` read.

## Alternatives

- *Fixed absolute bound (no derivation).* Simpler, but the right value is
  per-kernel, not one global constant.
- *A default bound when the size cannot be read.* Rejected in review: a guessed
  bound can hide a real change, and keeping the exact comparison and letting the
  administrator set the bound is safer.
- *Tolerance on by default once the gate is enabled.* Rejected in review: the
  existing behavior stays the default regardless of the gate, as for the other
  kubelet policy options. Enabling is the gate plus the option.
- *Fraction of node RAM.* Rejected: the kernel image does not scale with RAM, so
  a fraction would tolerate multi-GiB changes on large machines and hide a real
  memory loss.
- *Extend cAdvisor to report the bound.* Cleaner layering but net-new API plus a
  cross-repo revendor cycle for a single, boot-static consumer.
- *Sum-only comparison.* Insufficient: on bare metal the sum itself drifts, and a
  whole-image relocation is not a sum change.
- *`nokaslr`.* Not acceptable as a fix (it weakens security), useful only to confirm
  the root cause.

## Infrastructure Needed

None.
