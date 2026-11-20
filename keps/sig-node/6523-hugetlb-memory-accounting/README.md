# KEP-6523: Kubelet support for HugeTLB memory accounting on cgroup v2

Status: provisional. The shared-budget policy and related eviction and MemoryQoS choices are proposed for review. No target release is assigned.

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Three questions for design review](#three-questions-for-design-review)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Terms and limits](#terms-and-limits)
  - [Resource aggregation and ownership](#resource-aggregation-and-ownership)
  - [Node Allocatable and ancestor cgroups](#node-allocatable-and-ancestor-cgroups)
  - [CRI and OCI runtimes](#cri-and-oci-runtimes)
  - [Statistics and eviction](#statistics-and-eviction)
  - [MemoryQoS](#memoryqos)
  - [Mode detection and feature enablement](#mode-detection-and-feature-enablement)
  - [Evidence that affects the design](#evidence-that-affects-the-design)
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
  - [Keep memory.max unchanged](#keep-memorymax-unchanged)
  - [Keep independent ordinary-memory and HugeTLB hard limits](#keep-independent-ordinary-memory-and-hugetlb-hard-limits)
  - [Adjust memory.max whenever HugeTLB usage changes](#adjust-memorymax-whenever-hugetlb-usage-changes)
  - [Add hugepage requests to the eviction memory request](#add-hugepage-requests-to-the-eviction-memory-request)
  - [Disable memory_hugetlb_accounting on the host](#disable-memory_hugetlb_accounting-on-the-host)
- [Infrastructure Needed](#infrastructure-needed)
- [References](#references)
<!-- /toc -->

## Release Signoff Checklist

- [ ] Enhancement tracking issue links to the KEP directory and targets a release.
- [ ] KEP approvers have approved `implementable` status.
- [ ] Design details, test plan, and graduation criteria are complete and approved.
- [ ] Production readiness review is complete and approved.
- [ ] Implementation history is up to date.
- [ ] User documentation and release notes are ready.
- [ ] Applicable beta API and GA conformance requirements have been assessed.

## Summary

With `memory_hugetlb_accounting` enabled, HugeTLB usage counts toward the cgroup memory limit. Kubelet still sets that limit from `memory` alone, so a Pod can hit it before using the memory and hugepages listed in its spec.

We propose adding the effective hugepage limits to each finite workload memory limit, while keeping the per-size hugepage limits. For `memory: 96Mi` and `hugepages-2Mi: 64Mi`, that means `memory.max=160Mi` and `hugetlb.2MB.max=64Mi`.

This permits ordinary memory to use unused hugepage headroom. It does not reserve that headroom for later HugeTLB faults or preserve the old 96Mi ordinary-memory cap. Whether Kubernetes should accept that change is the main question in this draft. Eviction and MemoryQoS also need rules for the combined counter.

## Motivation

An OS upgrade can enable this accounting mode without any change to kubelet or the Pod spec. In [Harshal's reproduction][reproduction], a Pod with a 96Mi memory limit and a 64Mi hugepage limit touched 32Mi of hugepages, then tried to use 80Mi of ordinary memory. It completed with accounting off and was OOMKilled with accounting on. That kernel lacked `CONFIG_CGROUP_HUGETLB`, so the test covered memory accounting but not the separate hugepage limit.

Failure is not always an OOM kill or SIGBUS. In our Fedora 6.19.10 test, HugeTLB charge failure returned `-ENOMEM`, became `VM_FAULT_OOM`, and caused repeated page faults. This matters for detection: watching only OOMKilled misses a workload that keeps retrying.

The effect also reaches kubelet policy. Working set can include HugeTLB while eviction compares it against the memory request. MemoryQoS uses that same combined kernel counter with thresholds derived from memory resources. Raising `memory.max` alone leaves those differences unresolved.

### Goals

- Define workload behavior with accounting on and off.
- Make Pod and container limits consistent with that behavior.
- Keep scheduling, eviction, statistics, and MemoryQoS resource meanings clear.
- Include hugepages supplied through KEP 5517 resource mappings.
- Provide a node upgrade and rollback procedure that does not silently change the accounting of running workloads.

### Non-Goals

- Add live hugepage resizing or change the scheduler's resource API.
- Require applications to fault in all hugepages at startup.
- Make HugeTLB pages reclaimable or manage the host's hugepage pool.
- Emulate an independent ordinary-memory hard limit with a kubelet polling loop.
- Deprecate accounting-off nodes in this KEP.
- Promise a backport before the compatibility impact is understood.

## Proposal

Kubelet will detect the effective accounting mode and use it consistently when building cgroup limits and interpreting memory statistics. It will not infer the mode from the systemd version or change the host mount options itself.

When accounting is off, resource behavior stays as it is today. When it is on, a finite memory limit becomes the memory limit plus the effective hugepage limits. Each per-size hugepage limit remains enforced. An absent memory limit stays absent; a hugepage limit must not accidentally introduce a finite memory limit.

Requests remain separate Kubernetes resources. In particular, unused hugepage capacity is not an extra ordinary-memory scheduling request. This distinction also drives the proposed eviction policy below.

### User Stories

A node operator upgrades an OS image that enables HugeTLB memory accounting. After draining and validating the node, workloads can consume their configured memory and hugepage allowances without hitting an unadjusted Pod memory limit.

An application faults in hugepages after startup. Its operator can see that the memory limit is shared on this node and understands why earlier ordinary-memory use can cause a later hugepage allocation to fail.

### Three questions for design review

1. **Should ordinary memory be allowed to use unused hugepage headroom?** A fixed `M+H` limit allows this. Later HugeTLB faults can fail if the headroom has been consumed. We prefer this to a kubelet loop that adjusts limits as usage changes, but it changes isolation and needs agreement.
2. **What usage should eviction compare with the memory request?** We prefer an ordinary-memory comparison. Subtracting charged HugeTLB is a possible implementation, but only if the provider supplies compatible samples. The current CRI fields and independently read kernel counters do not settle that problem.
3. **How should MemoryQoS protect ordinary memory?** Adding H to min/low protects more ordinary memory when hugepages are unused; leaving them unchanged can reduce ordinary-memory protection when hugepages are used. Tests show both costs. We have not chosen a protection rule.

Enablement, unsupported configurations, and rollback also remain open. The relevant sections describe those choices; the draft does not assign a gate name or target release.

### Risks and Mitigations

A workload that fills its combined budget with ordinary memory may be unable to fault in hugepages later. Applications cannot rely on reclaim returning borrowed headroom. Document this behavior and test both allocation orders.

Updating only the container limit leaves the Pod or another ancestor limiting usage. Create, resize, and reconciliation must use the same resource calculation at each affected level.

Incorrect or stale HugeTLB statistics could change eviction ordering. The provider must distinguish a missing measurement from zero and identify the accounting scope. Host pool capacity and process mappings are not substitutes for charged usage.

## Design Details

### Terms and limits

For one cgroup, let:

- `M` be its effective finite Kubernetes memory limit.
- `H_i` be its effective hugepage limit for page size `i`; `H = sum(H_i)`.
- `R` be non-HugeTLB memory charged to that cgroup, including charged kernel memory. It is not simply the application's malloc total.
- `U` be HugeTLB memory charged to the memory controller for that cgroup.

Ignoring transient overages and tighter ancestor limits, the old constraints are `R <= M` and `U_i <= H_i`. The proposal uses `R + U <= M + H`, while keeping the per-size constraints. These constraints are not equivalent.

With `M=96Mi` and `H=64Mi`, the following is intentional under the proposal:

| Charged HugeTLB | Remaining combined budget for other charged memory |
| --- | --- |
| 0Mi | Up to 160Mi |
| 32Mi | Up to 128Mi |
| 64Mi | Up to 96Mi |

These are accounting bounds, not guarantees that an application can allocate those exact byte counts. Ancestors, overhead, pool availability, and other limits still apply. Reservations in the hugepage pool do not reserve memcg space.

### Resource aggregation and ownership

Aggregate Kubernetes resources before translating them to cgroup limits. Use the existing rules for init containers, restartable init containers, Pod overhead, and PodLevelResources. A Pod limit must not be computed by summing already translated child limits.

Include KEP 5517 memory and hugepage mappings in the effective resource lists. Shared claims must retain their existing ownership rules; translation must not add a claim twice or give every container the full shared allocation.

Keep track of whether a memory limit was declared. An absent limit stays absent. Check additions for overflow and return an error identifying the resource calculation that failed.

Use the same conversion for creation, updates, and reconciliation. During memory resize, preserve parent/child ordering and compare limits in the same units. Live hugepage resizing is outside this proposal.

### Node Allocatable and ancestor cgroups

Keep the scheduler's memory and hugepage resources separate. `Node.Status.Allocatable.memory` already subtracts hugepage pool capacity; subtracting it again would understate ordinary-memory capacity.

The top-level workload cgroup starts from machine capacity minus reservations. Its budget already includes the physical memory used by the hugepage pool, so the Pod `M+H` formula must not be applied there unchanged.

QoSReserved and system/kube-reserved limits need their own calculations. A higher leaf limit cannot override an ancestor. Any downward update must also account for the reclaim/OOM behavior discussed in #141609.

Returning a HugeTLB page to the pool does not normally return it to the ordinary allocator. Our tests restored HugePages_Free without reducing HugePages_Total. Extra ordinary-memory cgroup headroom therefore does not mean extra ordinary physical memory is available.

### CRI and OCI runtimes

Existing CRI fields can carry the combined memory limit and separate hugepage limits. The inspected containerd/CRI-O create paths and runc/crun setters do not add the hugepage amount again.

In the tested containerd v2.4.1/runc v1.5.2 combination, a memory-only CRI update changed the container limit from 96Mi to 160Mi and retained the 64Mi hugepage limit. The Pod stayed at 96Mi, which confirms that a container-only change is insufficient.

Other runtime versions and both cgroup managers still need testing. The inspected CRI-O and runc update paths omit typed hugepage-limit changes; that matters if live hugepage resizing is added later, but does not expand this KEP's scope.

### Statistics and eviction

Keep total charged memory visible: it explains the kernel's combined limit. Eviction needs a separate decision about what usage to compare with an ordinary-memory request.

We prefer to retain ordinary-memory request comparisons. For example, an 80Mi working set containing 32Mi charged HugeTLB would count as 48Mi for that comparison. With no HugeTLB charged, the same 80Mi remains above a 64Mi request even if the Pod has unused hugepage allowance. Applicable DRA memory requests remain included.

The subtraction needs more than another counter. Total usage and charged HugeTLB must refer to the same cgroup, accounting mode, and compatible samples. A provider that already excludes HugeTLB must not subtract it again. In a churn test, reading the counters 3ms apart produced negative estimates in 167 of 200 samples. Clamping them to zero hides the error. We need to agree how the provider collects the value, reports unavailable data, and handles sample skew before enabling it for eviction.

Shared mappings add another constraint. The backing folio is charged to the first-faulting cgroup, not every process that maps it. Our test retained the charge after process exit and owner cgroup removal until the backing file was released. Summing mappings or substituting the host pool size would give the wrong value. Charged usage is also not a prediction of how much ordinary memory a Pod eviction will free.

The real kubelet test confirms the current mismatch: two equal-priority Pods requested 48Mi memory each. The Pod using 32Mi ordinary memory plus 32Mi HugeTLB was evicted as above its request; the Pod using 40Mi ordinary memory remained running. We triggered the actual eviction path by raising `memory.available`'s threshold on the dedicated node, rather than exhausting host RAM. This tests existing selection, not a replacement policy.

Node-root `memory.available` and the allocatable-cgroup signal need separate treatment. The inspected root reader derives usage from `anon + file`, while the summary provider separately accounts for pool capacity. Applying the Pod subtraction there could subtract twice. Non-root available memory should continue to describe headroom under its reported combined limit.

**Open:** CRI `MemoryUsage` has no dedicated charged-HugeTLB breakdown. Decide whether to extend it or use another supported collection path. Missing must differ from zero. The KEP cannot reach `implementable` until the eviction policy and provider behavior are defined together.

### MemoryQoS

The hard limit does not determine reclaim protection. `memory.high` throttles usage; `memory.min` and `memory.low` protect it from reclaim. All use the combined memory counter.

For an existing finite high threshold, we suggest testing `oldHigh + H`. At full HugeTLB usage, this leaves the old headroom for other memory; otherwise ordinary memory can use the unused part. Unset or unlimited thresholds stay unchanged. This rule still needs validation at container and Pod levels.

For min/low, neither fixed choice preserves the old behavior:

| Setting, with memory request Q=64Mi and hugepage allowance H=64Mi | No HugeTLB charged | 64Mi HugeTLB charged |
| --- | --- | --- |
| Keep min/low at Q | Nominal protection covers 64Mi ordinary usage | HugeTLB occupies that threshold |
| Set min/low to Q+H | Up to 128Mi ordinary usage falls below the threshold | Threshold leaves 64Mi for ordinary usage |

Actual protection also depends on ancestors and sibling usage. In our stronger-pressure test with 64Mi HugeTLB in use, min=128Mi retained more ordinary cache and the sibling was OOM-killed. Min=64Mi allowed more reclaim. That may be the intended cost of protecting requested memory; the OOM alone does not make the policy wrong. When HugeTLB was unused, however, the higher min also protected extra ordinary memory. Low could be breached, so it must not be treated as equivalent to min.

With a 64Mi protection limit on an intervening ancestor and pressure from an external sibling, the leaf's min=128Mi no longer produced the same OOM result. The QoS ancestor calculation must therefore be part of the chosen rule.

**Open:** choose the protection behavior and define its ancestor calculation. Do not automatically add H everywhere, change which controls a QoS class uses, or count shared DRA allocations twice. MemoryQoS support is not ready for release until this is resolved and tested.

### Mode detection and feature enablement

Read the effective cgroup v2 mount used for workload cgroups. Do not infer accounting mode from systemd's version. Check hugetlb controller availability separately and report missing runtime or statistics support.

Initially, change modes only on drained nodes and create fresh workload cgroups. Kubelet will not remount the hierarchy. Pages used while accounting was off are not retroactively charged after a remount.

**Open:** define how users enable this support and what kubelet does when the node cannot support it. A default-off gate that ignores an enabled kernel mode leaves the original failure in place. Automatic detection should follow the real mode; an override cannot change what the kernel counts. We still need to choose whether an unsupported configuration rejects affected Pods or prevents node readiness. No gate or configuration field is assigned yet.

### Evidence that affects the design

Tests ran on Fedora `6.19.10-300.fc44.x86_64` with accounting and the hugetlb controller enabled. The kubelet/runtime checks used kubelet v1.38.0-beta.0, containerd v2.4.1, and runc v1.5.2 with the systemd cgroup driver.

- Return probes recorded repeated `-ENOMEM` from HugeTLB charge/allocation and `VM_FAULT_OOM` from `hugetlb_fault`. Kernel logs printed `Retrying PF`. The matching Fedora source maps the error through `vmf_error()` into the x86 OOM fault path. This explains the repeated fault on this version, not a regression range across kernels.
- Shared-file tests kept the first-fault charge until backing-file release. Freed pages returned to the pool; its total size did not shrink.
- CRI and kubelet container working sets included HugeTLB. Separately timed samples failed the proposed subtraction under churn. The first Pod summary could also lag the container sample.
- The eviction and protection results are described above. Protection tests biased the sibling's OOM score to make victim selection predictable; they were not kubelet eviction tests. The initial tmpfs attempt was excluded and rerun with disk-backed cache.

The working notes retain scripts, raw counters, and failed attempts. These results cover one environment. A patched kubelet, the full runtime matrix, and the proposed statistics and MemoryQoS changes have not been validated.

### Test Plan

This draft is based on source inspection and the reproduction reported in the issue. Direct-kernel experiments and one kubelet/CRI mapping check have been run; see the observations above. The integration and release matrices remain unexecuted.

##### Prerequisite testing updates

Provide Linux nodes that can independently vary accounting and hugetlb controller availability. Record kernel config, mount options, runtime versions, and the full cgroup ancestry. Use fresh allocations after a mode change. Tests must distinguish SIGBUS, memcg OOM, node OOM, and failure of the separate hugepage limit.

##### Unit tests

- Mode selection and capability/mismatch results.
- Finite, absent, zero, and overflowing memory budgets; multiple hugepage sizes.
- Pod/container aggregation, init/sidecars, overhead, PodLevelResources, and DRA.
- Shared claim ownership and exactly-once translation.
- Statistics normalization, missing versus zero data, and eviction comparisons.
- The selected MemoryQoS rules, disabled controls, and ancestor budgets; add protection-policy assertions only after choosing that policy.

##### Integration tests

Exercise both stats providers and actual kubelet resource-generation/update paths. Verify parent and child limits agree with the intended ownership model. Check memory resize and restart reconciliation without introducing hugepage resizing. Verify Pod eviction ordering with equal ordinary usage but different HugeTLB use. Separately verify root and allocatable memory-pressure signals.

##### e2e tests

| Scenario | Required observation |
| --- | --- |
| Accounting off | Existing separate-resource behavior is retained |
| Accounting on, both controllers active | Combined memory limit and per-size hugepage limits are applied |
| Ordinary memory first, then HugeTLB faults | Shared headroom is usable; later charge failure is correctly identified |
| HugeTLB first, then ordinary memory | Ordinary-memory exhaustion follows the expected memcg path |
| HugeTLB reaches its own limit | Separate hugepage enforcement still works with spare combined budget |
| Ancestor below leaf limit | The ancestor remains limiting and failures identify that cgroup |
| MemoryQoS and sibling pressure | Compare both protection candidates; validate the chosen policy and throttling under contention |
| Missing hugetlb controller or stats capability | Agreed unsupported-configuration behavior is observable |
| Drained-node upgrade and rollback | Newly created Pods use the intended mode and old state is not misinterpreted |

Test containerd and CRI-O with supported runc/crun combinations, including systemd and cgroupfs managers. Include 2Mi pages and 1Gi pages where supported; a missing hardware capability is a documented skip, not a pass. Collect cgroup counters and kernel logs with exit status. Do not rely on an application's byte count alone.

### Graduation Criteria

#### Alpha

- The resource contract, eviction/statistics contract, MemoryQoS protection policy, and enablement mechanism are approved.
- Required stats capabilities and runtime dependencies are implemented.
- Both accounting modes and independent hugepage enforcement have e2e coverage.
- Late HugeTLB allocation and ordinary-memory borrowing behavior are documented.
- Upgrade/rollback is tested on drained nodes.

#### Beta

- Supported runtime/manager combinations have repeatable coverage.
- Eviction, MemoryQoS, DRA, and PodLevelResources interactions are covered.
- Operator feedback shows the shared-budget behavior and failure diagnostics are usable; unresolved isolation regressions block promotion.
- Default behavior, skew support, metrics, and rollback have renewed PRR approval.

#### GA

- The behavior has remained stable through beta and supported upgrades.
- No known correctness gaps remain in the supported accounting/statistics paths.
- Long-term support and any conformance eligibility are agreed with the relevant SIGs. Feature lifecycle and removal of rollout controls follow normal policy.

No release numbers or graduation dates are proposed yet.

### Upgrade / Downgrade Strategy

Cordon and drain before changing accounting mode or moving between kubelet versions that interpret limits differently. Check the mode and controllers, restart the required components, recreate Pods, and verify their limits before uncordoning. Live transitions are outside the initial rollout.

Downgrading kubelet while leaving accounting enabled can restore the original failure. Rollback must restore a compatible kubelet/OS configuration on a drained node. Shrinking existing limits can trigger reclaim or OOM during the write, as described in #141609.

### Version Skew Strategy

During rollout, nodes may use different accounting modes. Scheduling resource names stay the same, but ordinary-memory borrowing differs by node. Document that difference.

Runtime compatibility depends on the statistics design. Any new CRI field must be optional on the wire, with a way to distinguish unsupported data from zero. The KEP must list supported combinations once that interface is chosen.

## Production Readiness Review Questionnaire

These are draft design answers. They do not record PRR approval.

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

The enablement mechanism is still open. The initial rollout requires draining the node, checking its accounting/controller/statistics support, and recreating Pods. No control-plane downtime is expected.

###### Does enabling the feature change any default behavior?

Yes. On accounting-enabled nodes, ordinary memory can use unused hugepage headroom. Eviction and MemoryQoS behavior also depend on the choices above. Accounting-off behavior should stay unchanged.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Not by toggling a gate on a busy node. Drain it and restore a compatible kubelet and accounting configuration. Leaving accounting enabled with an old kubelet restores the original problem.

###### What happens if we reenable the feature if it was previously rolled back?

Recheck support and recreate workloads on the drained node. Pages allocated while accounting was off will not become charged merely because it is enabled again.

###### Are there any tests for feature enablement/disablement?

Not yet. The alpha plan includes drained-node enablement, rollback, and reenable tests.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Mismatched accounting, stale limits, missing statistics, or an ancestor cap can break allocations or eviction decisions. Lowering memory.max can kill running workloads. The initial rollout requires draining.

###### What specific metrics should inform a rollback?

Watch allocation failures, repeated faults, memcg OOM events, evictions, memory.high pressure, and statistics errors. Compare with the node's baseline. OOMKilled alone misses failed or repeatedly retried HugeTLB faults.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

No. This remains an alpha release requirement.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No removals or deprecations are proposed. Accounting-off nodes remain supported.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Kubelet should report the detected accounting mode and missing dependencies at startup. A node-level metric is proposed, with its name and labels still to be reviewed.

###### How can someone using this feature know that it is working for their instance?

Check the effective Pod resources against container and ancestor memory.max and per-size hugepage limits. Compare charged HugeTLB with the statistics used for eviction, then run a mixed-allocation test Pod.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

No new availability SLO. Resource setup should meet existing kubelet latency and error expectations. A workload must not fail just because kubelet left its hugepage allowance out of a finite combined limit.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

Resource setup and statistics errors, memory pressure, evictions, memcg OOM events, and HugeTLB allocation failures. Some allocation failures are expected when the shared budget is exhausted.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

We need visibility into detected mode, unsupported configurations, and unavailable HugeTLB measurements. Metric names and any CRI additions depend on the final interface design.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No new cluster service. Required node support includes cgroup v2 accounting, hugetlb enforcement, and compatible runtime/statistics providers. MemoryQoS and DRA interactions apply when those features are enabled.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No new API-server calls. Statistics collection may require an extra read per sample; its cost needs measurement.

###### Will enabling / using this feature result in introducing new API types?

No new Kubernetes resource type. A CRI statistics extension is still under discussion.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No Pod or Node object changes are proposed. A CRI breakdown would add data to each affected statistics response.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Resource conversion adds arithmetic, and collecting the HugeTLB breakdown may add work to sampling. Neither cost has been benchmarked yet.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Kubelet will not track individual pages or poll usage to resize limits. Detection is node-scoped and conversion happens during resource updates. Statistics overhead still needs measurement on dense nodes.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No additional unbounded resource is planned. The larger ordinary-memory allowance can increase memory pressure without increasing ordinary physical capacity, so node enforcement and eviction still matter.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Existing cgroup limits continue to apply. The kernel allocation path does not need an API-server call. Kubelet otherwise follows its existing behavior during control-plane outages.

###### What are other known failure modes?

Late HugeTLB charge failures or repeated faults, absent controllers, stale or incomplete statistics, incompatible runtimes, and tighter ancestor limits. Shared backing files can retain charges after their processes exit.

###### What steps should be taken if SLOs are not being met to determine the problem?

Collect mount options, kernel/runtime versions, Pod resources, ancestor limits, memory.current/stat/events, per-size hugetlb counters, and kernel logs. Identify the failing controller and check sample timestamps before changing limits. Reproduce mode changes on a drained test node.

## Implementation History

- 2026-11-20: Added direct fault-return tracing, kubelet node-pressure eviction, shared-page lifetime, churn, and exact Fedora source evidence to the provisional design. No kubelet implementation, runtime matrix, or approval is claimed.

## Drawbacks

A combined budget relaxes the independent ordinary-memory cap and makes allocation order matter. It does not increase the ordinary memory available outside the reserved hugepage pool. Statistics and protection rules need additional work, and initial rollout requires draining nodes.

## Alternatives

### Keep memory.max unchanged

With accounting enabled, ordinary memory and HugeTLB compete inside M. This reproduces the original failure even when each resource stays within its own allowance.

### Keep independent ordinary-memory and HugeTLB hard limits

This would preserve isolation, but the inspected controls in the same accounting-enabled cgroup do not express a separate ordinary-memory cap. A CRI field cannot supply enforcement the kernel lacks. We would need another concrete mechanism before choosing this option.

### Adjust memory.max whenever HugeTLB usage changes

Kubelet sampling and limit writes race with allocations. Increasing the limit can arrive too late, and lowering it can cause reclaim or OOM. We do not recommend this approach.

### Add hugepage requests to the eviction memory request

Comparing combined usage with combined requests avoids subtracting usage, but gives unused hugepage allowance a role in ordinary-memory eviction priority. That differs from the ordinary-memory comparison proposed here and needs an explicit decision.

### Disable memory_hugetlb_accounting on the host

An OS-supported opt-out can be a workaround. It does not provide kubelet support for accounting-enabled nodes. Kubelet will not remount the hierarchy or assume every distribution provides the same switch.

## Infrastructure Needed

We need Linux nodes where we can test with `memory_hugetlb_accounting` on and off. Run the tests with containerd and CRI-O, using the runc/crun and cgroup driver combinations we plan to support. Enable the hugetlb controller when testing hugepage limits; without it, we can test memory accounting but not hugepage limit enforcement.

## References

- [Issue #144131 and design discussion](https://github.com/kubernetes/kubernetes/issues/144131).
- [Latest budget discussion](https://github.com/kubernetes/kubernetes/issues/144131#issuecomment-7006072982).
- [Linux v6.18 accounting documentation](https://github.com/torvalds/linux/blob/7d0a66e4bb9081d75c82ec4957c50034cb0ea449/Documentation/admin-guide/cgroup-v2.rst#L219-L246).
- [HugeTLB memcg charge](https://github.com/torvalds/linux/blob/7d0a66e4bb9081d75c82ec4957c50034cb0ea449/mm/memcontrol.c#L4723-L4758).
- [Node allocatable reporting](https://github.com/kubernetes/kubernetes/blob/9a2fac740d0be5ae1e831857b622707af07208be/pkg/kubelet/nodestatus/setters.go#L329-L342).
- [Node cgroup budget](https://github.com/kubernetes/kubernetes/blob/9a2fac740d0be5ae1e831857b622707af07208be/pkg/kubelet/cm/node_container_manager_linux.go#L259-L281).
- [Eviction request comparisons](https://github.com/kubernetes/kubernetes/blob/9a2fac740d0be5ae1e831857b622707af07208be/pkg/kubelet/eviction/helpers.go#L693-L729).
- [MemoryQoS resource generation](https://github.com/kubernetes/kubernetes/blob/9a2fac740d0be5ae1e831857b622707af07208be/pkg/kubelet/kuberuntime/kuberuntime_container_linux.go#L192-L250).
- [KEP 5517](https://github.com/kubernetes/enhancements/blob/ebf37575297a991155890e7f1fae7c62bbee7ca0/keps/sig-scheduling/5517-dra-node-allocatable-resources/README.md).
- [Node limit update issue #141609](https://github.com/kubernetes/kubernetes/issues/141609).

[reproduction]: https://github.com/kubernetes/kubernetes/issues/144131#issuecomment-6990948125
