# KEP-6303: In-place Pod Vertical Scaling (Container Resize) on Windows

> **AI assistance disclosure:** This KEP draft was written with the assistance of an AI coding
> agent. The human author (github.com/MartinForReal) is fully responsible for the content and
> for shepherding this proposal through the Kubernetes enhancement process. Per the contributor
> guidelines, AI use is disclosed here and in the PR description, and the author verifies every
> design decision before it is committed.

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [The situation](#the-situation)
  - [The complication](#the-complication)
  - [The question](#the-question)
  - [The answer](#the-answer)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: VPA in-place mode on a Windows StatefulSet](#story-1-vpa-in-place-mode-on-a-windows-statefulset)
    - [Story 2: A disabled-by-default Windows gate](#story-2-a-disabled-by-default-windows-gate)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Design Overview](#design-overview)
  - [Kubelet Gating Changes](#kubelet-gating-changes)
  - [Windows Resize Reconciliation Path](#windows-resize-reconciliation-path)
  - [CRI Resource Update for Windows Containers](#cri-resource-update-for-windows-containers)
  - [CPU Resource Update](#cpu-resource-update)
  - [Memory Limit Enforcement on Windows](#memory-limit-enforcement-on-windows)
  - [Test Plan](#test-plan)
    - [Prerequisite testing updates](#prerequisite-testing-updates)
    - [Unit tests](#unit-tests)
    - [Integration tests](#integration-tests)
    - [e2e tests (Windows)](#e2e-tests-windows)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha (v1.38)](#alpha-v138)
    - [Beta (v1.39)](#beta-v139)
    - [GA (v1.41)](#ga-v141)
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

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in kubernetes/enhancements
  (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as implementable
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for Conformance Tests, and that all GA Endpoints
    are hit by Conformance Tests within one minor version
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] Implementation History section is up-to-date for milestone
- [ ] User-facing documentation has been created in kubernetes/website, for the current release
  (with supporting documentation in relevant PRs/issues/release notes)

[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website

## Summary

**Windows nodes will support in-place pod vertical scaling (container resize) for CPU and memory,
behind a new, disable-aware alpha feature gate. The change is confined to the kubelet, reuses the
existing CRI `UpdateContainerResources` call and the Windows container-resource mapping the kubelet
already applies at container creation, and adds no API, scheduler, or control-plane change. With the
new gate off, Windows keeps rejecting resizes exactly as it does today.**

Three decisions carry the proposal:

1. **Enablement is gated, not removed.** A new alpha gate, `WindowsInPlacePodResize` (off by default),
   governs Windows separately from the GA `InPlacePodVerticalScaling` gate.
2. **Enforcement reuses creation-time semantics.** CPU limits map to `CpuMaximum` (a hard percent cap);
   memory limits map to the job-object commit cap (`MemoryLimitInBytes`). Resize introduces no second
   enforcement mechanism, so a resized container enforces what a recreated one would.
3. **A Windows branch replaces the cgroup-only pipeline.** The shared resize path reads and
   dereferences pod cgroups unconditionally and aborts on Windows. Windows therefore needs a platform
   branch that builds `WindowsContainerResources` directly, while preserving the shared path's
   ordering, memory-decrease validation, and actuated-resource accounting.

## Motivation

### The situation

In-place pod vertical scaling (`InPlacePodVerticalScaling`, KEP-1287) is GA and on by default. On
Linux a resize reaches the runtime through CRI `UpdateContainerResources` and is enforced by cgroups.

### The complication

On Windows the feature is entirely absent, for two distinct reasons:

- **A deliberate hard-reject.** `pkg/kubelet/allocation/features_windows.go` returns
  `(false, "In-place pod resize is not supported on Windows", "windows")` from both
  `IsInPlacePodVerticalScalingAllowed` and `IsInPlacePodLevelResourcesVerticalScalingAllowed`.
- **A pipeline that cannot run.** Even with that rejection removed, the shared resize path reads pod
  cgroup configuration and dereferences `CPUShares`/`CPUQuota`, none of which exist on Windows.
  KEP-1287 intended `UpdateContainerResources` to work on Windows; the wiring was never done.

### The question

What is the smallest change that makes Windows resize work, without touching the API, QoS semantics,
or the Linux path?

### The answer

A gated Windows resize path, and nothing more. It delivers:

- **On-line CPU scaling.** Raise or lower a running Windows container's `CpuMaximum` without restart.
- **Commit-cap memory resize.** Move the job-object commit cap (`JOB_OBJECT_LIMIT_JOB_MEMORY`) in place
  by updating `resources.memory.limit`.
- **Consistency between advertised and enforced resources**, so scheduler, eviction, and accounting
  agree with what the runtime enforces.
- **Working VPA in-place and StatefulSet resize** on Windows node pools.

### Goals

- Container-level CPU and memory in-place resize on Windows, with no pod recreation or container restart.
- Reuse creation-time semantics (`CpuMaximum`, job commit cap) so resize never changes what is enforced.
- Add a Windows alpha gate (`WindowsInPlacePodResize`) that can be toggled independently of the GA
  `InPlacePodVerticalScaling` gate.
- Specify and implement the Windows path through the shared resize reconciliation pipeline.

### Non-Goals

- Changing the PodSpec resources API or QoS-class semantics.
- Hyper-V-isolated pod resize in alpha (initial scope is process-isolated Windows containers).
- Any change to the Linux path.
- Reproducing Linux OOM-kill semantics on Windows. The commit cap surfaces allocation failures; this is
  documented, not hidden.
- Pod-level-resource resize on Windows in alpha; it is deferred to beta.

## Proposal

### User Stories

#### Story 1: VPA in-place mode on a Windows StatefulSet

A Windows-hosted .NET workload is scaled live by VPA in-place mode. Today every update forces pod
recreation and drops connections. After this KEP, VPA resizes requests and limits in place.

#### Story 2: A disabled-by-default Windows gate

A cluster runs `InPlacePodVerticalScaling` GA on its Linux nodes. The operator does not want that
behavior on Windows node pools until they opt in. `WindowsInPlacePodResize` (alpha, off by default)
provides both the opt-in and a clean rollback.

### Notes/Constraints/Caveats

- Windows has no cgroups. Enforcement uses the HCS job object, surfaced through the runtime
  (containerd); the kubelet abstracts it behind the CRI update path.
- The CRI spec already carries `WindowsContainerResources` with CPU count/maximum and memory-limit
  fields. The kubelet maps these at creation (`calculateWindowsResources` in
  `kuberuntime_container_windows.go`); resize reuses that mapping rather than adding a conversion.
- Feature-gate lifecycle (verified against `pkg/features/kube_features.go`):
  - `InPlacePodVerticalScaling` is GA (1.35), `LockToDefault: true`, and carries a `// remove in 1.38`
    note. It therefore cannot be used as a Windows toggle, and the Windows gate must not depend on the
    symbol after it is deleted.
  - `InPlacePodLevelResourcesVerticalScaling` is **Beta** (1.35 alpha, 1.36 beta, default on), not GA,
    and remains toggleable. It gates the *pod-level* resource feature and is out of scope here.

### Risks and Mitigations

- **Risk:** runtime live-update behaviour differs across Windows Server versions and containerd
  releases. **Mitigation:** define the failure contract (treat CRI `Unimplemented` as a refusal with an
  event reason) and validate against the containerd version shipped at implementation time (see
  [Dependencies](#dependencies)).
- **Risk:** users assume Windows memory behaves like Linux `memory.max` (OOM kill). **Mitigation:**
  document commit-cap semantics, emit an event on apply, and log the divergence.
- **Risk:** scope creep into adjacent Windows parity work. **Mitigation:** strict non-goals; pod-level
  resources and OOM observability are fenced to follow-ups.

## Design Details

### Design Overview

The design is one new code path with four parts, one per concern.

| # | Concern | Decision |
|---|---------|----------|
| 1 | Who may resize? | Windows honors the new alpha gate; the GA gate is honoured only while it exists. |
| 2 | Which code path runs? | A Windows branch replaces the cgroup-only reconciliation path. |
| 3 | What is sent to the runtime? | `WindowsContainerResources` built by the existing creation mapping. |
| 4 | What is recorded? | Actuated CPU maximum and memory commit limit, via `setActuatedContainerResources`. |

The flow for one resize:

1. Allocation code asks `IsInPlacePodVerticalScalingAllowed`; Windows answers from the Windows gate.
2. `doPodResizeAction` takes the Windows branch instead of reading pod cgroups.
3. The branch builds `WindowsContainerResources`, runs the memory-decrease validation, and calls CRI
   `UpdateContainerResources`.
4. On success it records actuated resources; on failure it reports an event reason (not a hard gate
   rejection), so controller retries converge.

Nothing outside this flow changes.

### Kubelet Gating Changes

**Decision:** on Windows, a resize is accepted only when `WindowsInPlacePodResize` is enabled (and,
while it still exists, `InPlacePodVerticalScaling` is also enabled).

Today `pkg/kubelet/allocation/features_windows.go` hard-rejects every resize:

    func IsInPlacePodVerticalScalingAllowed(_ *v1.Pod) (bool, string, string) {
        return false, "In-place pod resize is not supported on Windows", "windows"
    }

The post-change Windows gate:

    // features_windows.go (after)
    func IsInPlacePodVerticalScalingAllowed(...) (bool, string, string) {
        // InPlacePodVerticalScaling is GA/LockToDefault and is deleted in 1.38;
        // keep the check behind build-time availability so this compiles before
        // and after its removal.
        if v, ok := featureGateIfPresent(features.InPlacePodVerticalScaling); ok && !v {
            return false, "InPlacePodVerticalScaling is disabled", "feature_gate_off"
        }
        if !utilfeature.DefaultFeatureGate.Enabled(features.WindowsInPlacePodResize) {
            return false, "WindowsInPlacePodResize is disabled", "windows_gate_off"
        }
        return true, "", ""
    }

**Why a new gate rather than removing the rejection.** `InPlacePodVerticalScaling` is GA,
default-on, and `LockToDefault`, so it cannot act as a Windows toggle, and it is scheduled for removal
in 1.38 — the very release this KEP targets. An alpha feature must be off by default and
disable-supportable, which only a separate Windows gate can express. The sketch above therefore treats
the GA gate as *optional*: while it exists it is still honoured, and after its removal the Windows gate
alone decides. Before the GA gate is deleted, a follow-up must confirm that the delete-only removal
leaves the Windows decision intact.

### Windows Resize Reconciliation Path

**Decision:** add a Windows branch to `doPodResizeAction` that bypasses cgroup reads and builds the
runtime request directly, while preserving the shared path's ordering, validation, and accounting.

**Why the gate alone is not enough.** The shared path aborts on Windows before reaching the runtime:

1. `cm.ResourceConfigForPod` (`pkg/kubelet/cm/helpers_unsupported.go`) returns nil on Windows, and
   `doPodResizeAction` fails the resize with `unable to get resource configuration processing resize
   for pod %q` (`kuberuntime_manager.go`).
2. `generateUpdatePodSandboxResourcesRequest` (`kuberuntime_container_windows.go`) returns nil for the
   pod-level sandbox update, and `updatePodSandboxResources` hard-fails on a nil request.

**Why making `ResourceConfigForPod` non-nil still is not enough.** Even with a value, the shared path
reads and dereferences cgroup-only fields unconditionally: `PodContainerManager.GetPodCgroupConfig`
for memory and CPU; a nil-`CPUShares` rejection; and `currentPodCPUConfig.CPUQuota` /
`podResources.CPUQuota` / `CPUShares` dereferences. `cm.ResourceConfig` (`pkg/kubelet/cm/types.go`)
has no `CpuMaximum` field at all. None of these are meaningful on Windows.

**What the Windows branch must do.**

- Build `WindowsContainerResources` directly from container resources via `calculateWindowsResources`.
- **Explicitly skip `UpdatePodSandboxResources` for the container-level alpha.** The pod-level sandbox
  update is a no-op on Windows and must not be attempted; skipping it is a scope decision, not an
  oversight, and pod-level resize is deferred to beta.
- Preserve container-update ordering, and surface under-gate failures as an event reason rather than a
  hard rejection.
- Run the memory-decrease validation described in
  [Memory Limit Enforcement on Windows](#memory-limit-enforcement-on-windows) before calling the runtime.
- Track actuated resources through `setActuatedContainerResources`.

**Partial updates and actuated state.** After a successful `UpdateContainerResources`, the kubelet
records the actuated CPU maximum and memory commit limit actually applied — the container-level values,
not the pod-level aggregate — so accounting does not drift. Controller retries converge idempotently.

### CRI Resource Update for Windows Containers

**Decision:** reuse CRI `UpdateContainerResources` and `WindowsContainerResources`. No CRI API addition
is required for the container-level scope.

The kubelet must:

1. build the desired `WindowsContainerResources` from the CPU limit and memory limit using the existing
   `calculateWindowsResources` mapping, reused rather than reimplemented;
2. invoke `UpdateContainerResources` and update the actuated resource record;
3. surface an under-gate failure as an event reason rather than a hard rejection.

**Failure contract.** Because no CRI capability advertises live-update support, the kubelet does not
probe for one. It calls `UpdateContainerResources` and maps a CRI `Unimplemented` (or an equivalent
runtime error) to a refusal with an event reason, leaving the previous limits in force. This keeps the
feature fail-closed without a new CRI API.

### CPU Resource Update

**Decision:** CPU limits are enforced as `CpuMaximum` on resize, exactly as at creation. CPU requests
are accounting-only and never become a weight.

Windows has no CFS quota or periods. The kubelet intentionally enforces a CPU limit with `CpuMaximum`
(a percentage in \[1, 10000\]) and does not use weights, because a weight is relative and, when set
alongside a maximum, can cause the maximum to be ignored — the existing comment in
`calculateWindowsResources` records this. On resize:

- reuse `calculateCPUMaximum(cpuLimit, processorCount)` to set `CpuMaximum` from `resources.limits.cpu`,
  with the same meaning as creation;
- record the requested CPU value for accounting only; do not translate it to a weight;
- do not send `CpuCount` or `CpuWeight` from the resize path.

**On `CpuCount` precedence.** `calculateWindowsResources` contains a `CpuCount`-over-`CpuMaximum`
precedence branch, but it is currently unreachable: the kubelet never populates `CpuCount` or
`CpuWeight` from the Pod API. This KEP therefore does not rely on that branch. If a future change starts
populating those fields, the mutual-exclusion behaviour must be re-derived against the runtime then.

**Limit removal.** Linux has an explicit TODO for removing a CPU limit; Windows cannot express
`CpuMaximum` removal the same way. Alpha therefore scopes to setting or changing a finite limit, and
removing a CPU limit on Windows is deferred and tracked as a follow-up.

### Memory Limit Enforcement on Windows

**Decision:** a memory limit is the job-object commit cap (`MemoryLimitInBytes`), applied in place,
subject to an explicit pre-apply validation. It is not Linux `memory.max` and not working-set trimming.

Windows process-isolated containers enforce the memory limit as a commit cap via the job object
(`JOB_OBJECT_LIMIT_JOB_MEMORY`); an allocation that would exceed the cap fails. On resize:

- set `WindowsContainerResources.MemoryLimitInBytes` directly from `resources.limits.memory` — the same
  field and mechanism used at creation;
- **validate before applying, and be explicit about what the validation reads.** The shared validator
  compares the *reported memory usage* from `PodCPUAndMemoryStats` (`statsapi ...Memory.UsageBytes`)
  against the requested limit. This is **not** the job-object committed-byte counter. If the CRI
  implementation does not report memory, the stats layer coalesces a missing value to zero, and a
  zero-usage comparison passes vacuously — i.e. an unsafe decrease could be applied.
  - Alpha therefore **requires** the Windows CRI implementation to report container memory usage, and
    the Windows branch must treat a missing/zero reported usage as *unknown* and refuse the decrease
    (fail closed) rather than treating it as safe.
  - Reading the job-object commit counter directly is the more precise alternative and is the preferred
    follow-up; it is out of scope for alpha only because it needs a new runtime-facing call.
- a refused decrease leaves the previous cap in force and the resize is retried on later reconciles;
- document that a successfully applied decrease reclaims nothing already committed: if committed usage
  later grows into the smaller cap, the next allocation fails and the container may become unhealthy.
  The operator is told at apply time instead of the resize being force-applied;
- emit `WindowsMemoryLimitApplied` (a **new** event, added by this KEP) reporting the commit cap actually
  enforced.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests.

#### Prerequisite testing updates

Describe what parts of the feature will be covered by existing tests and what additional tests need to
be added. We base this on reviews and discussions with owners of the involved components.

- `kubelet/kuberuntime` and `pkg/kubelet/allocation` already own resize unit tests; the Windows gate and
  the Windows branch extend those suites rather than creating new harnesses.
- No new test framework is required. The Windows branch needs a small amount of platform-specific test
  plumbing so the resource mapping can be asserted without a live Windows node.

#### Unit tests

- `pkg/kubelet/allocation` (Windows build): `WindowsInPlacePodResize` gates the feature (off = refuse,
  on = accept); the GA gate is still honoured while present.
- `kubelet/kuberuntime`: `calculateWindowsResources` is reused for the update (CPU `CpuMaximum` mapping,
  memory commit cap); a CPU request never becomes a weight; a limit-only resize of a running
  finite-CPU-limit container.
- Update produces correct `WindowsContainerResources` for CPU/memory increase and decrease.
- Memory-decrease validation: a reported usage at or above the requested limit refuses the decrease; a
  missing/zero reported usage is treated as unknown and also refuses (no vacuous pass).

#### Integration tests

- API surface unchanged (control-plane integration suites).
- The kubelet integration suite does not run a Windows kubelet, so the resize flow itself is covered by
  the Windows e2e below rather than by `test/integration/kubelet`.

#### e2e tests (Windows)

- Increase and decrease CPU (`CpuMaximum`) and memory limit on a running Windows container without restart.
- A memory-limit decrease at or below reported usage is refused (limit unchanged, resize retried);
  verify no force-apply occurs.
- A memory-limit decrease strictly above reported usage succeeds; afterwards, allocating beyond the new
  cap surfaces the documented allocation failure / container-unhealthy behavior.
- Process-isolated Windows containers are the initial scope; Hyper-V is a follow-up.
- Run in the periodic SIG-Windows jobs.

### Graduation Criteria

#### Alpha (v1.38)

- New `WindowsInPlacePodResize` gate, off by default, `disable-supported: true`.
- Container-level CPU (`CpuMaximum`) and memory commit resize works for process-isolated containers
  through the Windows branch, bypassing the cgroup-only `CPUShares`/`CPUQuota` requirements.
- Gate on/off behavior, the CRI `Unimplemented` refusal path, and the memory-decrease validation are
  covered by tests.
- A flake-free window in SIG-Windows periodic jobs before beta.

#### Beta (v1.39)

- Gate defaults to on, with SIG sign-off.
- Pod-level resource resize on Windows is implemented (`UpdatePodSandboxResources` no longer skipped).
- CPU and memory parity (`CpuMaximum`, commit cap) documented and tested.

#### GA (v1.41)

- Conformance e2e for Windows in-place resize present and passing, with a maintained Windows CI job.
- The required minimum two-week flake-free window for GA e2e tests is satisfied.

### Upgrade / Downgrade Strategy

The new Windows gate is off by default and disable-supported, so upgrade keeps current behavior until an
operator opts in, and downgrade restores the rejection. The `InPlacePodVerticalScaling` gate is
unchanged by this KEP. There is no API change.

### Version Skew Strategy

The gate is evaluated per node by the kubelet; apiserver and scheduler are unchanged. Kubelet/runtime
skew is handled by the failure contract in
[CRI Resource Update for Windows Containers](#cri-resource-update-for-windows-containers): if the runtime
cannot live-update, the call fails and the kubelet refuses the resize with an event, so older and newer
pairings degrade gracefully.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `WindowsInPlacePodResize`
  - Components depending on the feature gate: kubelet
- [ ] Other

Enabling or disabling requires a kubelet restart on the Windows nodes. No control-plane downtime and no
node reprovisioning are required.

###### Does enabling the feature change any default behavior?

No. The gate is off by default, so behavior is unchanged until an operator enables it. When enabled,
Windows nodes begin accepting resizes that currently fail.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Set `WindowsInPlacePodResize=false` and restart the kubelet. Resizes then fail with the
pre-existing rejection reason. Containers already resized keep their current limits and are not
restarted; no migration or replay is needed.

###### What happens if we reenable the feature if it was previously rolled back?

A fresh kubelet start re-arms the gate. There is no persisted state to reconcile, so no migration.

###### Are there any tests for feature enablement/disablement?

Yes. Windows unit tests flip `WindowsInPlacePodResize` and assert accept (on) and refuse (off). The
Windows e2e runs with the gate enabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout can fail only if the runtime cannot live-update; the kubelet then refuses the resize with an
event and running workloads are unaffected. Because the gate is per node, nodes roll independently and
there is no cross-node consistency requirement.

###### What specific metrics should inform a rollback?

The existing resize metrics, filtered to Windows nodes: a rising
`kubelet_pod_infeasible_resizes_total` or a falling `kubelet_pod_resize_duration_milliseconds{success}`
share, together with refusal event reasons.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet; planned as manual validation on a Windows node pool during alpha.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No. This KEP does not deprecate or remove anything. It does depend on the removal of the
`InPlacePodVerticalScaling` gate in 1.38, which is tracked by that gate's own lifecycle and is handled by
the optional-check sketch above.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The existing `kubelet_container_requested_resizes_total` metric (labeled by resource/requirement/
operation) rises on a Windows node pool when resize policies are actively issuing there. No new metric
is introduced by this KEP.

###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: `WindowsMemoryLimitApplied` (new, emitted when a memory commit cap is applied)
- [x] API .status
  - Condition name: existing pod resize conditions (`PodResizePending` / `PodResizeInProgress`)
- [ ] Other

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Successful-apply rate of at least 99.9% on the Windows readiness e2e over the two-week pre-release
window, with no open flake-only failures in SIG-Windows testgrid.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: existing kubelet resize metrics (`kubelet_pod_resize_duration_milliseconds`,
    `kubelet_pod_infeasible_resizes_total`, `kubelet_container_requested_resizes_total`)
  - Components exposing the metric: kubelet

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Per-resize latency split by node OS is not directly available: the existing resize metrics are not
OS-labeled. Extending them with an OS dimension (or adding a Windows-only label) is a candidate
follow-up rather than a new counter.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

- **Container runtime (Windows)**
  - Usage description: apply live CPU/memory updates via CRI `UpdateContainerResources`, and report
    container memory usage so the decrease validation is not vacuous.
  - Impact of its outage: resizes fail with an event; running workloads are unaffected.
  - Impact of degraded performance or missing memory stats: resizes are refused and retried.

**Containerd/HCS evidence to attach at implementation time.** The kubelet side is verified in tree:
creation maps CPU to `CpuMaximum` and memory to `MemoryLimitInBytes` (`kuberuntime_container_windows.go`),
and the update request carries `WindowsContainerResources` (`api.proto`). The runtime side — that
containerd implements `UpdateContainerResources` for Windows and that memory is enforced as
`JOB_OBJECT_LIMIT_JOB_MEMORY` in HCS — lives outside this repository and **must be pinned to a specific
containerd/hcsshim version (with the relevant PR) before alpha**. The minimum supported containerd
version for Windows resize will be recorded here once confirmed.

`k8s.io/cri-api` is unchanged for the container scope, and no new third-party dependency is added.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API calls; resizes flow through the existing pod update path.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. Resize reuses the existing `resources` fields and resize conditions.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The per-node work is the same class as Linux resize: one CRI update per changed container.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. Work is bounded by the container count per pod and existing kubelet rate limits.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No new resource class is consumed. Resize does not create processes or containers.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Resize is driven by pod updates, so no new resize work starts. Already-applied limits persist.

###### What are other known failure modes?

- **Resize refused with the gate on**
  - Detection: kubelet event with a refusal reason; rising `kubelet_pod_infeasible_resizes_total`.
  - Mitigations: confirm `WindowsInPlacePodResize` is enabled and the runtime supports live updates.
  - Diagnostics: kubelet logs at the resize call site.
  - Testing: Windows unit and e2e tests cover accept/refuse.
- **Memory decrease not applied**
  - Detection: the container memory limit remains the previous value across reconciles.
  - Mitigations: none needed; the kubelet retries once reported usage falls below the requested limit.
  - Testing: validation unit test plus a Windows e2e case.
- **Allocation failures after a successful decrease**
  - Detection: container becomes unhealthy after growth into the smaller cap.
  - Mitigations: raise the limit again; the documented commit-cap trade-off.
  - Testing: Windows e2e case.

###### What steps should be taken if SLOs are not being met to determine the problem?

Break down the resize metrics by node and correlate with kubelet resize logs and the runtime version to
separate capability/refusal causes from validation rejects.

## Implementation History

- 2026-08-21: Initial provisional draft. Authored with AI assistance; human author responsible.
- 2026-09-09: Review-feedback pass: (1) new disable-aware Windows alpha gate; (2) CPU resize uses
  `CpuMaximum`, not shares; (3) Windows reconciliation path for the nil `ResourceConfigForPod` and the
  sandbox update; (4) memory described as a job-object commit cap.
- 2026-09-09 (b): Windows-specific branch bypassing the cgroup `CPUShares`/`CPUQuota` requirements;
  preserve the validator so a below-usage memory decrease stays unapplied and is retried.
- 2026-09-09 (c): align troubleshooting with the corrected memory-decrease behavior.
- 2026-09-23: sig-windows asked to include this KEP in the next release plan (1.38); SIG-Node assigned
  reviewers. Code-grounded review pass, restructured to put the answer first, with the design overview,
  and corrections: feature-gate lifecycle stated accurately (`InPlacePodVerticalScaling` is GA/
  LockToDefault and removed in 1.38; `InPlacePodLevelResourcesVerticalScaling` is Beta, not GA); the
  Windows gate no longer assumes the GA gate survives; `UpdatePodSandboxResources` is explicitly
  skipped; the memory-decrease validation is described by the signal it actually reads and now fails
  closed on missing usage; the unimplementable runtime capability probe is replaced by a CRI
  `Unimplemented` failure contract; new metrics are dropped in favour of existing resize metrics; the
  infeasible Windows kubelet integration test is removed; and the two-week window is scoped to GA.
- Tracking issue: kubernetes/enhancements#6303.

## Drawbacks

- Adds a Windows-only gate operators must learn.
- The commit-cap vs. Linux OOM divergence requires documentation and operator education.
- The Windows reconciliation branch is new platform plumbing that must be kept in parity with the shared
  path as it evolves.
- Alpha depends on the Windows CRI implementation reporting container memory usage; without it, memory
  decreases must be refused rather than validated.

## Alternatives

- **Remove the rejection in `features_windows.go` only.** Unsafe and ineffective: the pipeline still
  aborts, and alpha requires a disableable gate. Rejected.
- **Extend KEP-1287 in place.** KEP-1287 is implemented and stable; the remaining work is a new, scoped
  enhancement. Rejected.
- **Gate Windows behind the legacy `InPlacePodVerticalScaling` gate.** Impossible: it is `LockToDefault`
  and removed in 1.38. This is why a new gate is introduced.
- **Add a CRI capability for live resource updates.** More precise than an `Unimplemented` failure
  contract, but it changes the CRI API for a container-level feature that already has a
  well-defined call. Deferred; reconsider if other runtimes need to advertise partial support.

## Infrastructure Needed (Optional)

A maintained Windows CI job that runs the new Windows resize e2e in the SIG-Windows periodic suite.
