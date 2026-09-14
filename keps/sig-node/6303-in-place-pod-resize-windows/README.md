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
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: VPA in-place mode on a Windows StatefulSet](#story-1-vpa-in-place-mode-on-a-windows-statefulset)
    - [Story 2: A disabled-by-default Windows gate](#story-2-a-disabled-by-default-windows-gate)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Kubelet Gating Changes](#kubelet-gating-changes)
  - [Windows Resize Reconciliation Path](#windows-resize-reconciliation-path)
  - [CRI Resource Update for Windows Containers](#cri-resource-update-for-windows-containers)
  - [CPU Resource Update](#cpu-resource-update)
  - [Memory Limit Enforcement on Windows](#memory-limit-enforcement-on-windows)
  - [Test Plan](#test-plan)
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
- [ ] (R) KEP approvers have approved the KEP status as implementable
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for Conformance Tests
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] Implementation History section is up-to-date for milestone
- [ ] User-facing documentation has been created in kubernetes/website

[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website

## Summary

In-place pod vertical scaling (InPlacePodVerticalScaling, KEP-1287) is GA and default-on on
Linux, but the Windows kubelet hard-rejects every resource resize with the message In-place
pod resize is not supported on Windows, from the per-OS gate in
pkg/kubelet/allocation/features_windows.go (IsInPlacePodVerticalScalingAllowed and
IsInPlacePodLevelResourcesVerticalScalingAllowed).

This KEP removes that gap. It adds a new, disable-aware Alpha feature gate (WindowsInPlacePodResize)
that controls accepting the existing Linux-GA InPlacePodVerticalScaling feature on Windows, and
implements the CRI UpdateContainerResources live-update path for Windows containers using the exact
CPU (CpuMaximum) and memory (commit cap) semantics the Windows kubelet already applies at container
creation. The change is confined to the kubelet Windows build; behavior is unchanged when the new gate
is off.

## Motivation

Windows workers are excluded from in-place pod vertical scaling only because the kubelet hard-rejects
every resize through a hand-written per-OS gate. The runtime (containerd on Windows, backed by HCS
job objects) already supports updating live resource limits for containers, and KEP-1287 explicitly
intended UpdateContainerResources, the CRI update call, to work for Windows. That goal was never wired
into the Windows kubelet, and several shared resize-pipeline pieces today abort on Windows (see Windows
Resize Reconciliation Path). Closing the gap enables:

- On-line CPU vertical scaling: raise or lower a running Windows container CPU maximum (CpuMaximum)
  without restart.
- Commit-cap memory resize without restart: a Windows memory limit is a job-object commit cap
  (JOB_OBJECT_LIMIT_JOB_MEMORY) surfaced as an allocation failure, moved in place by updating
  resources.memory.limit.
- Resource and OS-consistency so scheduler, eviction, and accounting match enforced values.
- Operational convenience for VPA in-place and StatefulSet resizes on Windows node pools.

### Goals

- Container-level CPU and memory in-place resize on Windows without Pod recreation or container restart.
- Reuse existing Windows creation-time semantics (CpuMaximum for CPU limits, job commit cap for the
  memory limit) so a resize never changes what creation enforces.
- Introduce a Windows Alpha gate (WindowsInPlacePodResize) toggled independently of the locked, GA
  InPlacePodVerticalScaling gate.
- Specify and implement the Windows path through the shared resize reconciliation pipeline that
  aborts on Windows today.

### Non-Goals

- Changing the PodSpec Resources API or QoS-class semantics.
- In-place vertical scaling for Hyper-V isolated pods in Alpha (initial scope is process-isolated
  Windows containers).
- Any change to the Linux path.
- Reproducing Linux OOM-kill semantics on Windows: the Windows commit cap surfaces allocation failures,
  documented rather than hidden.
- Pod-level-resource resize on Windows in Alpha; it moves to the Beta milestone.

## Proposal

### User Stories

#### Story 1: VPA in-place mode on a Windows StatefulSet

A Windows-hosted .NET workload scales CPU and memory live via Vertical Pod Autoscaler in-place mode.
Today every update forces Pod recreation, dropping connections. After this KEP the VPA can resize
requests and limits live without restarting containers.

#### Story 2: A disabled-by-default Windows gate

A cluster whose Linux nodes run InPlacePodVerticalScaling GA-default-on should not enable that behavior
on Windows node pools until an admin opts in. The WindowsInPlacePodResize gate (alpha, off by default)
gives that opt-in and a clean rollback.

### Notes/Constraints/Caveats

- Windows has no cgroups. Enforcement uses the HCS job object surfaced through the runtime (containerd);
  the kubelet abstracts it behind the CRI update path.
- The CRI spec already carries a Windows container-resources message (WindowsContainerResources) with
  CPU count/maximum and memory-limit fields; the kubelet maps these at creation
  (kuberuntime_container_windows.go, calculateWindowsResources), and resize reuses that mapping.
- The Linux gates InPlacePodVerticalScaling and InPlacePodLevelResourcesVerticalScaling are GA default-on
  and cannot be toggled; they do not gate the Windows alpha feature.

### Risks and Mitigations

- Risk: runtimes expose inconsistent live-update semantics across Windows Server versions.
  Mitigation: probe runtime capability at kubelet start and at each resize; fail closed with a clear
  reason when the runtime reports no live-update support.
- Risk: users assume Windows memory behaves like Linux memcg commit (OOM kill).
  Mitigation: document commit-cap semantics and log the divergence so operators can plan capacity.
- Risk: scope creep into adjacent Windows parity items.
  Mitigation: keep strict non-goals; pod-level resources and OOM observability are fenced to their own
  follow-ups.

## Design Details

### Kubelet Gating Changes

Today pkg/kubelet/allocation/features_windows.go bypasses feature-gate checks and hard-rejects every
resize. The change makes the Windows kubelet honor a new, disable-aware Alpha gate gated on the existing
Linux gate, and only accepts a resize on Windows when the Windows gate is enabled.

Sketch of the post-change Windows gate:

    // features_windows.go (after)
    func IsInPlacePodVerticalScalingAllowed(...) (bool, string, string) {
        if !utilfeature.DefaultFeatureGate.Enabled(features.InPlacePodVerticalScaling) {
            return false, "InPlacePodVerticalScaling is disabled", "feature_gate_off"
        }
        if !utilfeature.DefaultFeatureGate.Enabled(features.WindowsInPlacePodResize) {
            return false, "WindowsInPlacePodResize is disabled", "windows_gate_off"
        }
        return true, "", ""
    }

The existing InPlacePodVerticalScaling gate cannot be a toggle on Windows: it is GA, default-on, and
LockToDefault since 1.35 and scheduled for removal in 1.38 (the current milestone target). A separate
disable-supported Windows gate is required so the Alpha can be turned off independently on Windows node
pools, keeping the documented rollback (gate-flip) implementable and the gate-off default matching current
behavior. Adding the new gate means the Alpha milestone keeps disable-supported: true.

### Windows Resize Reconciliation Path

The gate change alone is not enough: the shared resize pipeline aborts on Windows before reaching the
runtime because Linux-only dependencies return nil on non-Linux builds:

1. cm.ResourceConfigForPod (pkg/kubelet/cm/helpers_unsupported.go) returns nil on Windows; doPodResizeAction
   (pkg/kubelet/kuberuntime/kuberuntime_manager.go) fails the resize with unable-to-get-resource-configuration
   when it is nil.
2. generateUpdatePodSandboxResourcesRequest (kuberuntime_container_windows.go) returns nil for the pod-level
   sandbox update.

Making ResourceConfigForPod non-nil alone is not sufficient: even when it returns a value, the shared
doPodResizeAction reads and dereferences cgroup-only fields unconditionally. Specifically (kuberuntime_manager.go):

- it reads pod cgroup configuration via PodContainerManager.GetPodCgroupConfig for memory and CPU,
- in the CPU leg it rejects a missing podResources.CPUShares (fails the resize when nil) and dereferences
  currentPodCPUConfig.CPUQuota / podResources.CPUQuota / CPUShares into resizeContainers,
- the cm.ResourceConfig type (pkg/kubelet/cm/types.go) has no CpuMaximum field at all,
none of which exist or are meaningful on Windows (no cgroups), so populating only HCS-enforceable fields and making
the pod sandbox update best-effort still leaves CPU container updates reaching the cgroup dereference.

The Alpha change therefore adds a Windows-specific branch (or platform abstraction) in the resize path that,
for a Windows node, bypasses the cgroup-config read, the CPUShares nil-check and the CPUQuota/CPUShares dereference,
and instead builds the desired WindowsContainerResources directly from the container resources via
calculateWindowsResources here and performs the container update. The Windows branch must preserve the guarantees
that the shared path provides on Linux:
- container-update ordering and the under-gate failure -> event reason (not a hard gate rejection),
- the memory-decrease safety check (see Memory Limit Enforcement on Windows),
- actuated-resource tracking through setActuatedContainerResources.

Partial updates and actuated state: after a successful UpdateContainerResources on the Windows branch, the kubelet
calls setActuatedContainerResources and records the actuated CPU maximum and memory commit limit actually applied
(not the pod-level aggregate) so accounting does not drift; controller retries converge idempotently against the
runtime path.

### CRI Resource Update for Windows Containers

Keep using the existing CRI UpdateContainerResources and the Windows container-resources message. The kubelet must:

1. build the desired WindowsContainerResources from the CPU limit and memory limit using the existing calculation
   (calculateWindowsResources at kuberuntime_container_windows.go) reused for the update, not a new conversion,
2. invoke UpdateContainerResources and update the actuated resource record,
3. surface an under-gate failure as an event reason rather than a hard rejection.

No CRI API addition is required for the container-level scope.

### CPU Resource Update

Windows has no CFS quota/periods. The kubelet deliberately enforces a CPU limit with CpuMaximum (percent,
1-10000) as a hard cap, and a comment in kuberuntime_container_windows.go (calculateWindowsResources) explains
why weights (shares) are not used: they are relative and setting a weight alongside a maximum can let the maximum
be ignored. The proposal therefore does NOT convert limits to shares. On resize:

- Reuse calculateCPUMaximum(cpuLimit, processorCount) to set CpuMaximum from resources.limits.cpu, with the same
  meaning as creation.
- A CPU request is not enforced as a cap on Windows and is tracked only for accounting; the resize path records
  the desired requested value but does not translate it to a weight.
- CPU count precedence is preserved: when CpuCount is set, it takes precedence over CpuMaximum (Windows
  mutual-exclusion).

**Running-container limitation:** containerd preserves existing CPU fields when applying new nonzero values. If a
container was created with CpuMaximum set, sending additional CPU fields (nonzero CpuCount/CpuWeight) can leave both
CpuMaximum and CpuWeight set, which hcsshim rejects as mutually exclusive. Resize therefore validates the target of a running container that already has a finite CPU limit so a
limit-only resize never injects a weight. The e2e includes
a finite-CPU-limit running container.

### Memory Limit Enforcement on Windows

Windows process-isolated containers enforce the memory limit as a commit cap via the job object
(JOB_OBJECT_LIMIT_JOB_MEMORY), limiting the job's committed memory; an allocation that would exceed the cap fails.
This is distinct from Linux cgroup memory.max (an OOM kill) and from working-set trimming. The design:

- sets WindowsContainerResources.MemoryLimitInBytes directly to resources.limits.memory — the commit cap — the same
  field used at creation; no new enforcement mechanism,
- preserves the existing memory-decrease safety check: the shared validator rejects a new limit that is at or
  below the current committed usage before the runtime is called, so a requested decrease that would be unsafe is
  NOT applied. The lower cap stays at its previous value and is retried on subsequent reconciles until committed
  usage falls below the requested limit,
- documents that a successfully applied decrease reclaims nothing already committed: if committed usage later grows
  into the (smaller) cap, the next allocation fails and the container may become unhealthy — the operator is told
  this at apply time rather than the resize being force-applied,
- removes the proposed WindowsWorkingSetLimitApplied event; instead it emits WindowsMemoryLimitApplied reporting the
  commit cap actually enforced, which aids allocation-failure troubleshooting.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to existing tests.

#### Unit tests
- pkg/kubelet/allocation: Windows build asserts the new WindowsInPlacePodResize gate gates the Windows feature
  (off=refuse, on=accept) while InPlacePodVerticalScaling stays GA.
- kubelet/kuberuntime: calculateWindowsResources is reused for the update (CPU CpuMaximum mapping, memory commit
  cap); limit-only resize of a running finite-CPU-limit container; CPU request does not become a weight.
- Update generates correct WindowsContainerResources for CPU/memory decrease and increase and preserves the
  CpuCount/CpuMaximum mutual exclusion; validator keeps a below-usage memory decrease unapplied until usage permits.

#### Integration tests
- test/integration/kubelet: a Windows node honors the resize flow without restart and records actuated resources.
- test/integration/controlplane: API surface unchanged.

#### e2e tests (Windows)
- InPlacePodVerticalScaling Windows: increase and decrease CPU (CpuMaximum) and memory limit on a running Windows
  container without restart.
- Resize a running container created with a finite CPU limit (no CpuMaximum+CpuWeight mutual-exclusion error).
- Memory-limit decrease when committed usage is already at/above the requested value is safely rejected by
  the validator (limit stays at the previous value, resize retried) - verify no under-gate force-apply occurs.
- Memory-limit decrease to a value strictly above current committed usage succeeds; afterwards, allocating beyond
  the (now smaller) applied cap surfaces the documented allocation failure / container-unhealthy behavior.
- Process-isolated Windows containers are the initial scope; Hyper-V is a follow-up.
- Run in the periodic SIG-Windows conformance jobs with a two-week stability window before alpha.

### Graduation Criteria

#### Alpha (v1.38)
- New WindowsInPlacePodResize gate (off by default), disable-supported.
- Container-level CPU (CpuMaximum) and memory commit resize works for a process-isolated container via the
  Windows-specific reconciliation branch that bypasses the cgroup-only CPUShares/CPUQuota requirements.
- Gate-off / gate-on behavior and event reason covered; e2e stable two weeks.
#### Beta (v1.39)
- Gate default flips on (with SIG sign-off); pod-level (Always) resize on Windows.
- CPU and memory parity documented and tested (CpuMaximum and commit cap).
#### GA (v1.41)
- Conformance e2e for Windows in-place resize present and passing; maintained Windows CI.

### Upgrade / Downgrade Strategy

The new Windows gate is off by default and disable-supported: on upgrade the feature stays off until enabled;
downgrade restores the rejection. The InPlacePodVerticalScaling gates are unchanged (GA on Linux). No API change.

### Version Skew Strategy

The kubelet gate is per-node; apiserver/scheduler unchanged. Skew between the kubelet and the runtime (containerd)
is handled by a runtime capability probe: if it cannot live-update, the kubelet fails with an event, so old and new
pairings degrade gracefully.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- Enabled/disabled with the new WindowsInPlacePodResize feature gate on the kubelet; off by default so Windows
  matches today's behavior until given.
- No new API object; runtime-capability facing.
- Does enabling change default behavior? Yes, only when the Windows gate is on and the runtime supports a live update.
- What happens if we re-enable after rollback? A fresh kubelet start re-arms the gate; no replay or migration needed.
- Enable/disable tests: unit tests flip the Windows gate and assert accept/refuse; e2e runs with the gate on.

### Rollout, Upgrade and Rollback Planning

- Ships in the kubelet binary (Windows). Rollback is a gate flip or downgrade; no migration.
- The e2e must run against every release in the Windows CI.

### Monitoring Requirements

- Collect kubelet_inplace_pod_resize_total with a label for the OS and outcome so operators can observe accepted vs
  refused resize on Windows nodes.
- Emit a kubelet event reason (WindowsMemoryLimitApplied) when a memory limit (commit) is applied, for the
  allocation-failure troubleshooting path.

**SLI:** kubelet_inplace_pod_resize_total{os=windows,outcome=success|refused} and per-resize event latency are the
primary signals.

**SLO (alpha/beta):** successful-apply rate >= 99.9% on the Windows readiness e2e over the pre-release two-week
window, with no open flake-only failures in SIG-Windows testgrid.

**In-use signal for operators:** rate(kubelet_inplace_pod_resize_total{os=windows}[5m]) > 0 on a node-pool indicates
resize policies are actively issuing there.

### Dependencies

- k8s.io/cri-api (unchanged for container scope); containerd with the existing Windows UpdateContainerResources
  implementation. No new third-party dependency.

### Scalability

- No new API objects or control-plane channels. Per-node resize calls are the same as Linux; the existing kubelet
  rate limit bounds call volume.

### Troubleshooting

- Resize refused when the gate is on: check kubelet events for runtime capability/unsupported reason; confirm
  WindowsInPlacePodResize is enabled.
- If committed usage is at or above the requested memory limit, the existing limit remains applied and the
  resize is retried. After a successful decrease, subsequent allocations that exceed the new cap can fail.

## Implementation History

- 2026-08-21: Initial provisional draft. Authored with AI assistance; human author responsible.
- 2026-09-09: Review-feedback pass addressing four items: (1) new disable-aware Windows Alpha gate; (2) CPU resize
  uses CpuMaximum, not shares; (3) Windows resize reconciliation path for the nil ResourceConfigForPod and sandbox
  update; (4) memory described as a job-object commit cap, not working-set, with the event renamed and allocation-failure
  behavior defined.
- 2026-09-09 (b): second review pass - add a Windows-specific branch to bypass cgroup CPUShares/CPUQuota requirements in
  the reconciliation path; preserve the resize validator so a below-usage memory decrease stays unapplied and is retried;
  regenerate the table of contents (alpha-v138 / beta-v139 / ga-v141 anchors).
- 2026-09-09 (c): final editorial pass - align the troubleshooting guidance with the corrected memory-decrease
  behavior (below-usage decrease is retried, not applied; allocation failure only after a successful decrease).
- Tracking issue: kubernetes/enhancements#6303.

## Drawbacks

- Adds a Windows-gate knob operators must learn.
- The commit-cap vs Linux divergence requires documentation.
- The Windows reconciliation path is new platform plumbing that must stay in parity.

## Alternatives

- **Just remove the rejection in features_windows.go:** unsafe and ineffective because the pipeline aborts; and a
  disableable gate is required for alpha. Rejected.
- **Extend KEP-1287 in place:** already implemented/stable; the remaining Windows work is a new scoped KEP. Rejected.
- **Gate Windows behind the legacy InPlacePodVerticalScaling gate:** impossible because it is LockToDefault. This is
  why a new Windows gate is introduced.

## Infrastructure Needed (Optional)

A maintained Windows CI job that runs the new Windows resize e2e in the sig-windows periodic suite.
