# KEP-6523: Kubelet handling of HugeTLB memory accounting on cgroup v2

Status: provisional. The node-side contract below is proposed for review. No target release is assigned.

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Questions for design review](#questions-for-design-review)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Why kubelet keeps the separate-resource contract](#why-kubelet-keeps-the-separate-resource-contract)
  - [Mode detection](#mode-detection)
  - [Mode changes on a running node](#mode-changes-on-a-running-node)
  - [Affected Pods](#affected-pods)
  - [Admission](#admission)
  - [Node reporting](#node-reporting)
  - [Running Pods after a mode change](#running-pods-after-a-mode-change)
  - [Upstream opt-outs](#upstream-opt-outs)
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
    - [Deprecation](#deprecation)
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
  - [Combined memory limit (M+H)](#combined-memory-limit-mh)
  - [Keep memory.max unchanged and do nothing](#keep-memorymax-unchanged-and-do-nothing)
  - [Adjust memory.max whenever HugeTLB usage changes](#adjust-memorymax-whenever-hugetlb-usage-changes)
  - [Kubelet remounts cgroup2 without the option](#kubelet-remounts-cgroup2-without-the-option)
  - [Rely only on upstream opt-outs](#rely-only-on-upstream-opt-outs)
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

With the cgroup v2 mount option `memory_hugetlb_accounting`, the kernel charges HugeTLB pages to the memory controller as well as the hugetlb controller. Kubernetes gives `memory` and `hugepages-<size>` separate budgets, and kubelet sets `memory.max` from `memory` alone. On these nodes a Pod that stays within both of its allowances can still be OOM-killed, or hang on a page fault.

systemd v259 started mounting cgroup2 with this option whenever the kernel supports it. Nothing changed in Kubernetes.

An earlier draft of this KEP proposed raising each workload `memory.max` to memory plus hugepage limits. We dropped that: it makes `memory` stop capping ordinary memory, and a late hugepage fault against a full combined budget either retries forever (on Linux 6.14–7.3) or fails with `SIGBUS` (on Linux 7.4+ and stable backports) even though the Pod is within its hugepage limit. See [Combined memory limit (M+H)](#combined-memory-limit-mh).

This KEP keeps the existing resource contract. Kubelet detects the accounting mode from the live cgroup2 mount, keeps checking it while running, rejects Pods that request hugepages when the mode is on, and reports the condition on the Node. The real fix is to turn the option off for Kubernetes nodes: today through an fstab remount, later perhaps through a per-cgroup kernel opt-out. Kubelet's job is to make the broken combination visible and stop placing workloads into it.

## Motivation

The failure is silent in the worst case. In our Fedora 6.19.10 test, a Pod filled its memory limit with ordinary memory and then touched its hugepages. The HugeTLB memcg charge returned `-ENOMEM`, the fault returned `VM_FAULT_OOM`, and the task kept faulting. Kernel logs printed `Retrying PF`. There was no OOM kill and no event, and the Pod stayed Running. The reverse order produced a normal memcg OOM kill.

The kernel source explains why. `alloc_hugetlb_folio()` charges memcg with `__GFP_RETRY_MAYFAIL`, so `try_charge_memcg()` gives up before invoking the memcg OOM killer. `vmf_error()` turns the `-ENOMEM` into `VM_FAULT_OOM`, and on cgroup v2 `pagefault_out_of_memory()` returns without killing anything, so the fault is retried. This infinite retry loop is a Linux 6.14–7.3 regression introduced by commit `991135774c0e` and fixed in Linux 7.4-rc1 by commit `956aeff185e7` (`Cc: stable`), which returns `-ENOSPC` (`VM_FAULT_SIGBUS`) instead. Current distributions shipping systemd v259 (such as Ubuntu 26.04 on 7.0 and Fedora 44 on 7.2) fall in the affected window, and even with `956aeff185e7` the workload still fails with `SIGBUS` while within both limits. Line references are in [Evidence](#evidence-that-affects-the-design).

The mode is not fixed for the life of a node. The kernel applies cgroup2 root flags on every new cgroup2 mount made from the initial cgroup namespace, and clears flags the new mount does not list. A privileged container that shares the host cgroup namespace and mounts cgroup2 can therefore flip the mode, and so can an administrator remounting `/sys/fs/cgroup`. A check at kubelet startup is not enough.

systemd v259 and later enable the option whenever the kernel supports it, and systemd maintainers want to keep that default. Distributions shipping these versions (for example Ubuntu 26.04 and Fedora 44) will be in use for years. Kubelet needs defined behavior on these nodes regardless of what lands upstream.

### Goals

- Detect whether HugeTLB memory accounting is in effect on the node, from the live mount, not from version numbers.
- Notice when the mode changes while kubelet is running.
- Stop admitting Pods whose hugepage allowance cannot be honored, with a reason the user can act on.
- Report the condition on the Node so operators and controllers can find affected nodes.
- Leave behavior unchanged on nodes where the option is off.

### Non-Goals

- Change what `memory` or `hugepages-<size>` mean, or add hugepages to memory limits.
- Remount cgroup2 or change host mount options from kubelet.
- Evict or kill running Pods automatically when the mode changes.
- Support Pods with hugepages on accounting-enabled nodes. That needs a kernel-level opt-out.
- Change scheduling, Node Allocatable, eviction or MemoryQoS calculations.

## Proposal

Kubelet reads the cgroup2 mount options that are in effect and treats `memory_hugetlb_accounting` as a node-level mode. It re-reads them periodically and before admitting a Pod that needs hugepages.

When the mode is on:

- Pods that request hugepages are rejected at kubelet admission with a stable reason.
- The Node reports a condition saying hugepage workloads are not supported in the current mode.
- Pods already running are left alone. Kubelet emits a warning event for each running Pod that uses hugepages.

When the mode is off, nothing changes.

### User Stories

A node operator upgrades to an OS image with systemd v259. Without this KEP, hugepage Pods are scheduled there and some of them hang. With it, kubelet rejects those Pods with a clear reason, the Node shows the condition, and the operator knows to apply the OS opt-out before uncordoning.

A platform team runs a privileged agent that mounts cgroup2 inside a container sharing the host cgroup namespace. Kubelet notices that the mode changed, updates the Node condition, and warns about running hugepage Pods instead of failing silently.

### Questions for design review

1. **Reject at admission, or block the node?** We propose admission rejection for hugepage Pods plus a Node condition. Making the node NotReady would also take down unrelated Pods. Doing nothing leaves the silent hang.
2. **Should there be a scheduler-visible signal?** A Node condition does not affect scheduling by itself. Rejected Pods go to `Failed`, and the scheduler may place replacements on the same node again. Options are documenting a taint operators can add, having kubelet add one, zeroing/withholding hugepage capacity in `Node.Status.Allocatable` while the mode is `On` (at the cost of altering capacity views for already-running Pods), or accepting the retry churn in alpha.
3. **What should happen to running hugepage Pods when the mode flips on?** We propose warning only. Evicting them would turn a mount-option change into an outage. Saying nothing hides the risk.

### Risks and Mitigations

Rejecting Pods that work today. A Pod that never uses its memory and hugepages at the same time may run fine on an accounting-enabled node. Rejecting it is a behavior change. Mitigation: the check is behind a feature gate in alpha, and the rejection message says how to fix the node.

Detection reads the wrong mount. Kubelet may run in a container or in a different mount namespace. Mitigation: read the cgroup2 mount kubelet itself uses for workload cgroups, and report `Unknown` rather than `Off` when it cannot be read.

Polling misses a short-lived flip. A mode that turns on and off between two checks is invisible. Mitigation: re-read before admitting each hugepage Pod. Pods already running during the window are not protected either way.

## Design Details

### Why kubelet keeps the separate-resource contract

`mem_cgroup_charge_hugetlb()` charges a HugeTLB folio through the same `charge_memcg()` used for ordinary memory. One cgroup has one memory counter. With accounting on, no combination of `memory.max` and `hugetlb.<size>.max` in the same cgroup expresses "ordinary memory at most M and HugeTLB at most H". Any single `memory.max` value breaks either the hugepage allowance (M) or the ordinary-memory cap (M+H). [Combined memory limit (M+H)](#combined-memory-limit-mh) covers why M+H is not acceptable.

Since kubelet cannot express the contract on these nodes, it should say so rather than switch to a different contract.

### Mode detection

Kubelet reads the mount entry for the cgroup2 hierarchy it manages (normally `/sys/fs/cgroup`) from `/proc/self/mountinfo`. If the super options contain `memory_hugetlb_accounting`, the mode is on.

This reflects the kernel's root flag: the cgroup2 mount options show `memory_hugetlb_accounting` from `cgrp_dfl_root.flags`, the same flag `memcg_accounts_hugetlb()` checks when charging.

Kubelet does not infer the mode from the systemd or kernel version. The same versions can run in either mode.

The detection result is `On`, `Off`, or `Unknown`. `Unknown` covers unreadable mountinfo and a missing cgroup2 entry. cgroup v1 hosts are always `Off`, since the option does not exist there. `Unknown` is reported but does not reject Pods in alpha.

Kubelet also records whether the hugetlb controller is enabled. The admission check only applies when the node advertises hugepage resources.

### Mode changes on a running node

The kernel applies root flags on each new cgroup2 mount made from the initial cgroup namespace, and clears flags the mount does not list. Two paths can flip the mode without a reboot:

- An administrator remounts `/sys/fs/cgroup`.
- A container that shares the host cgroup namespace mounts a new cgroup2 instance. The flip can go either way: containerd copies only `nsdelegate` and `memory_recursiveprot` from the host into that mount (and runc <= 1.4.0 had the same issue before `runc#5267`), which drops `memory_hugetlb_accounting` (`On` to `Off`), while `systemd-nspawn` without a cgroup namespace mounts cgroup2 with `memory_hugetlb_accounting`, which turns it back on (`Off` to `On`).

Kubelet re-reads the mode:

- on a fixed interval, using the existing node status update loop;
- synchronously, before admitting a Pod that requests hugepages.

On a change, kubelet logs the old and new mode, updates the Node condition, and emits a Node event.

Pages already charged keep their charge until they are freed. Pages allocated while the mode was off are not charged retroactively. Kubelet does not try to reason about this; it reports the current mode.

### Affected Pods

A Pod is affected if any of its effective resource lists include a non-zero `hugepages-<size>` limit. That covers containers, init containers, pod-level resources, and hugepages supplied through KEP 5517 DRA mappings. The check uses the same aggregation kubelet already uses when setting hugetlb limits.

Pods without hugepages are not affected. Kubelet already sets their hugetlb limits to 0, so they cannot fault in HugeTLB pages under kubelet-managed cgroups.

### Admission

When the feature gate is enabled and the mode is `On`, a new `PodAdmitHandler` rejects affected Pods with:

- reason: `HugePageAccountingUnsupported` (name to be reviewed);
- message: states the node mode and that turning `memory_hugetlb_accounting` off on the node restores support, with a documentation link.

The handler runs after the existing resource admission handlers, so it does not change the rejection reason for Pods that already fail for other reasons.

Static Pods go through the same admission path and are rejected the same way.

### Node reporting

Kubelet adds a Node condition (name to be reviewed, for example `HugePageAccountingUnsupported`):

- `True` when the node advertises hugepage capacity and the mode is `On`.
- `False` when the mode is `Off`, or the node has no hugepage capacity.
- `Unknown` when detection fails.

Kubelet also exposes a gauge, `kubelet_hugetlb_memory_accounting_mode` (name to be reviewed), with bounded label values `on`, `off`, `unknown`.

### Running Pods after a mode change

When the mode changes from `Off` to `On`, kubelet emits a `Warning` event on each running affected Pod. It does not evict, restart, or change limits. When the mode goes back to `Off`, no action is needed.

### Upstream opt-outs

These are the ways to turn the option off. Kubelet does not depend on any of them; it reports the mode in effect.

| Mechanism | Status as of this draft | Notes |
| --- | --- | --- |
| `/etc/fstab` entry for `/sys/fs/cgroup` without the option | Works today; the approach systemd maintainers recommend ([1](https://github.com/systemd/systemd/pull/45018#issuecomment-7126558906), [2](https://github.com/systemd/systemd/pull/45018#issuecomment-7131319992)) | Applied by `systemd-remount-fs` before `local-fs.target`, so before the container runtime. A later cgroup2 mount from the initial cgroup namespace can change the mode again |
| systemd `systemd.memory_hugetlb_accounting=` switch, default off | [systemd#45018](https://github.com/systemd/systemd/pull/45018), closed (rejected) on 2026-11-24 | Maintainers keep accounting on by default and recommend `/etc/fstab` for opt-out |
| Per-cgroup memory controller opt-out | [Kernel RFC](https://lore.kernel.org/all/20261124003145.80705-1-harpatil@redhat.com/) posted 2026-11-24 | Would let kubelet opt out for `kubepods` only; not in any released kernel |

systemd's position is that HugeTLB memory accounting should stay on by default and that setups like Kubernetes should opt out. Accounting-enabled nodes are therefore the long-term default on new distributions, not a transition limited to systemd v259-v262. The admission message points operators at the fstab opt-out.

If a per-cgroup control lands, a follow-up can let kubelet set it on the cgroups it owns and lift the admission rejection on kernels that support it. That is out of scope here.

### Evidence that affects the design

Tests ran on Fedora `6.19.10-300.fc44.x86_64` with accounting and the hugetlb controller enabled, using kubelet v1.38.0-beta.0, containerd v2.4.1, and runc v1.5.2 with the systemd cgroup driver.

- With `memory.max` raised to M+H, filling the budget with ordinary memory and then faulting in hugepages did not complete within a 15s timeout. Return probes recorded repeated `-ENOMEM` from the HugeTLB charge and `VM_FAULT_OOM` from `hugetlb_fault`, and the kernel logged `Retrying PF`. The reverse order produced a memcg OOM kill.
- With M+H, ordinary memory used the unused hugepage headroom. [Harshal's tests](https://github.com/kubernetes/kubernetes/issues/144131#issuecomment-7104174643) saw a Pod with a 96Mi memory limit run at 152Mi.
- With `memory.max` unchanged, the [original reproduction][reproduction] was OOM-killed after touching 32Mi of hugepages and 80Mi of ordinary memory under a 96Mi limit.
- fstab opt-out (systemd 259.5): an fstab line for `/sys/fs/cgroup` without the option is skipped by `systemd-fstab-generator`, since it is an API mount, and applied by `systemd-remount-fs`. Running it removed the option. Faulting 32Mi of hugepages then raised `memory.current` by 3Mi with `memory.stat:hugetlb` at 0, against 35Mi and 32Mi with the option on. A real reboot and an initrd that mounts cgroup2 first were not tested.
- Runtime flip: on the same host, `mount -t cgroup2 -o nsdelegate,memory_recursiveprot none /tmp/x` removed `memory_hugetlb_accounting` from `/sys/fs/cgroup` for the whole host.

Source references (Linux v7.0):

- `mm/hugetlb.c:2918`: `alloc_hugetlb_folio()` uses `htlb_alloc_mask(h) | __GFP_RETRY_MAYFAIL`.
- `mm/hugetlb.c:3042-3052`: a memcg charge failure frees the folio and returns `-ENOMEM`.
- `mm/hugetlb.c:5792-5807`: `hugetlb_no_page()` passes that error to `vmf_error()`.
- `mm/memcontrol.c:4779-4800`: `mem_cgroup_charge_hugetlb()` calls `charge_memcg()` when `memcg_accounts_hugetlb()` is true.
- `mm/memcontrol.c:2444-2445`: `__GFP_RETRY_MAYFAIL` skips `mem_cgroup_oom()`.
- `include/linux/mm.h:4182-4183`: `vmf_error(-ENOMEM)` returns `VM_FAULT_OOM`.
- `mm/oom_kill.c:1196-1209`: `pagefault_out_of_memory()` returns when no memcg OOM is pending, and the fault is retried.
- Linux commit `956aeff185e7` (7.4-rc1, `Fixes: 991135774c0e`, `Cc: stable`): changes `hugetlb_alloc_folio()` to return `-ENOSPC` on `mem_cgroup_charge_hugetlb()` failure, converting the infinite retry loop back to `VM_FAULT_SIGBUS`.
- `kernel/cgroup/cgroup.c:2045-2069,2328`: a new cgroup2 mount from the initial cgroup namespace sets or clears `CGRP_ROOT_MEMORY_HUGETLB_ACCOUNTING`.

These results cover one environment. The detection and admission design has not been implemented.

### Test Plan

##### Prerequisite testing updates

Node e2e needs Linux hosts with hugepages and cgroup v2, where tests can mount cgroup2 with and without `memory_hugetlb_accounting`. Tests that remount must run serially on a dedicated node.

##### Unit tests

- Mountinfo parsing: option present, absent, cgroup v1, missing entry, unreadable file.
- Mode change detection and event emission.
- Affected-Pod classification: container, init container, sidecar, pod-level resources, DRA-supplied hugepages, zero hugepage limits.
- Admission: gate off, mode off, mode on, mode unknown, Pods with and without hugepages.
- Node condition transitions.

##### Integration tests

None planned. The behavior depends on the host mount and is covered by node e2e.

##### e2e tests

| Scenario | Expected |
| --- | --- |
| Accounting off, hugepage Pod | Admitted, runs as today |
| Accounting on, hugepage Pod | Rejected with the new reason; Node condition `True` |
| Accounting on, Pod without hugepages | Admitted |
| Remount with the option while a hugepage Pod runs | Mode change logged, Node condition flips, warning event on the Pod, Pod keeps running |
| Remount without the option | Condition clears; new hugepage Pods admitted |
| Privileged container with host cgroupns mounts cgroup2 | Mode change is detected |
| Gate off, accounting on | Current behavior, no rejection |

### Graduation Criteria

#### Alpha

- Feature gate, default off.
- Mode detection, re-check, admission handler, Node condition and metric implemented.
- Unit tests and the node e2e scenarios above.
- Questions for design review resolved.

#### Beta

- Gate default on.
- Feedback from at least one distribution shipping systemd v259 or later.
- Rejection reason and condition names settled.
- Decision recorded on a scheduler-visible signal (question 2).

#### GA

- Stable for two releases in beta.
- Re-evaluate whether the check is still needed if a per-cgroup kernel opt-out is widely available.

#### Deprecation

Nothing is deprecated. If a per-cgroup kernel opt-out becomes widely available and kubelet uses it, the admission rejection can be narrowed to kernels without it, following the normal feature lifecycle.

### Upgrade / Downgrade Strategy

Upgrading kubelet on a node with accounting off changes nothing. On a node with accounting on and the gate enabled, kubelet starts rejecting new hugepage Pods. Running Pods are not touched.

Downgrading kubelet removes the check and restores today's behavior, including the silent failure.

Turning the option off on a node needs a reboot, or a remount before workloads start. Drain the node first. Pages charged while the option was on keep their charge until freed.

### Version Skew Strategy

The change is kubelet-only, with no API, scheduler, or CRI changes. Nodes with and without the check can coexist. A Node condition type the control plane does not know is stored and displayed as-is.

## Production Readiness Review Questionnaire

These are draft answers. They do not record PRR approval.

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

Kubelet feature gate `HugePageAccountingCheck` (name to be reviewed). Changing it requires a kubelet restart. No control-plane change.

###### Does enabling the feature change any default behavior?

Only on nodes with `memory_hugetlb_accounting` on: new Pods that request hugepages are rejected, and a Node condition and Pod warning events are added.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disable the gate and restart kubelet. Rejected Pods are not recreated automatically; their controllers create replacements.

###### What happens if we reenable the feature if it was previously rolled back?

Kubelet detects the mode again and resumes rejecting affected Pods. Running Pods are unaffected.

###### Are there any tests for feature enablement/disablement?

Planned in node e2e: gate off and on with accounting enabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A wrong `On` result would reject hugepage Pods that would have worked. Running workloads are never evicted or changed by this feature.

###### What specific metrics should inform a rollback?

Admission rejections with the new reason on nodes believed to be in `Off` mode, and the mode gauge disagreeing with the host's mount options.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

The Node condition and the mode gauge. Rejected Pods show the new reason in their status.

###### How can someone using this feature know that it is working for their instance?

On a node with the option on, a Pod requesting hugepages is rejected with the new reason, and `kubectl describe node` shows the condition.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Mode changes are reflected in the Node condition within one node status update interval.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The mode gauge, the number of admission rejections with the new reason, and mode change events.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None identified beyond the mode gauge.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. It reads node-local mount information.

### Scalability

###### Will enabling / using this feature result in any new API calls?

A Node condition update when the mode changes, through the existing node status path. Warning events on mode changes, at most one per affected running Pod per change.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

One additional Node condition.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

One mountinfo read before admitting each Pod that requests hugepages. Not expected to be measurable; to be confirmed in alpha.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Detection and admission are node-local. Condition updates are delayed until the API server is reachable.

###### What are other known failure modes?

- Kubelet reads a different mount namespace than the one workloads use, so the mode can be wrong. Each detection log line records the mount ID and options read.
- The mode flips between two polls. Pods admitted in that window are not protected.

###### What steps should be taken if SLOs are not being met to determine the problem?

Compare `findmnt -o OPTIONS /sys/fs/cgroup` on the host with the kubelet log line and the mode gauge. Check for privileged containers that share the host cgroup namespace and mount cgroup2.

## Implementation History

- 2026-11-19: Initial provisional draft proposing a combined M+H memory limit.
- 2026-11-20: Added fault-return tracing, eviction, shared-page and churn evidence.
- 2026-11-24: Dropped the M+H proposal after tracing the late-fault retry loop. Reworked around mode detection, admission rejection, and upstream opt-outs.
- 2026-11-27: Clarified the Linux 6.14–7.3 hang regression window (`991135774c0e` / `956aeff185e7`), updated `systemd#45018` to closed, documented bidirectional runtime flips, and expanded scheduler-signal options.

## Drawbacks

Hugepage workloads cannot run on accounting-enabled nodes until the node is fixed. Some of them would have run, as long as they never needed both allowances at once.

Rejected Pods can churn if the scheduler keeps placing them on the same node.

Kubelet does not fix the problem itself. Until systemd or the kernel ships an opt-out, operators need a boot-time workaround.

## Alternatives

### Combined memory limit (M+H)

Raise each finite workload `memory.max` to the memory limit plus the effective hugepage limits, and keep the per-size hugetlb limits. This was the earlier proposal of this KEP. We dropped it for two reasons:

- `memory` stops capping ordinary memory. A Pod with a 96Mi memory limit and 64Mi of hugepages can use up to 160Mi of ordinary memory while its hugepages are unused. That changes isolation for every existing hugepage Pod.
- Late hugepage faults fail unexpectedly. Once ordinary memory fills the combined budget, a HugeTLB fault retries forever on Linux 6.14–7.3 (leaving the Pod Running with no event, see [Motivation](#motivation)), or fails with `SIGBUS` on Linux 7.4+ (`956aeff185e7`), even though the Pod is within its hugepage limit.

It would also need new rules for eviction, statistics and MemoryQoS on a combined counter. The earlier draft listed those as open questions.

### Keep memory.max unchanged and do nothing

This is today's behavior. Pods are OOM-killed or hang depending on allocation order, and nothing points at the node configuration.

### Adjust memory.max whenever HugeTLB usage changes

Sampling and limit writes race with allocations. Raising the limit can come too late, and lowering it can trigger reclaim or OOM. The kernel commit that added the option describes this approach as "cumbersome and buggy".

### Kubelet remounts cgroup2 without the option

Kubelet would remount a hierarchy it does not own, after other components may have started using it, and later mounts can set or clear the option again. Mount configuration belongs to the init system.

### Rely only on upstream opt-outs

systemd maintainers prefer to keep accounting on by default, with Kubernetes nodes opting out through fstab. The per-cgroup kernel opt-out is an RFC. A node that forgets the fstab entry, or where the mode flips at runtime, still needs kubelet to notice.

## Infrastructure Needed

Node e2e hosts with cgroup v2, hugepages configured, and permission to remount `/sys/fs/cgroup`.

## References

- [Issue #144131](https://github.com/kubernetes/kubernetes/issues/144131).
- [Original reproduction][reproduction].
- [Kernel commit adding the option](https://github.com/torvalds/linux/commit/8cba9576df601c384abd334a503c3f6e1e29eefb).
- [Kernel commit introducing the 6.14–7.3 fault-retry regression](https://github.com/torvalds/linux/commit/991135774c0e05a4734e6d32aa03b00355e4cac9) and [7.4-rc1 fix returning `-ENOSPC` / `SIGBUS`](https://github.com/torvalds/linux/commit/956aeff185e7e16b989202bde0bb29859adba99d).
- [Linux v7.0 cgroup-v2 documentation](https://github.com/torvalds/linux/blob/v7.0/Documentation/admin-guide/cgroup-v2.rst?plain=1#L220-L225).
- [Kernel RFC for per-cgroup `memory.hugetlb_accounting`](https://lore.kernel.org/all/20261124003145.80705-1-harpatil@redhat.com/).
- [systemd#39486: enable the option by default](https://github.com/systemd/systemd/pull/39486).
- [systemd#45018: proposal to make the option opt-in](https://github.com/systemd/systemd/pull/45018); closed (rejected) — maintainers prefer opt-out via fstab.
- [runc#5258](https://github.com/opencontainers/runc/issues/5258) and [runc#5267](https://github.com/opencontainers/runc/pull/5267): cgroupfs mounts with a shared cgroup namespace.
- [Kubelet hugetlb limits default to zero](https://github.com/kubernetes/kubernetes/blob/5e2cad18bceac0c1a8031ed5e11c2bb3ee3c4959/pkg/kubelet/kuberuntime/kuberuntime_container_linux.go#L384-L392).
- [Node allocatable subtracts the hugepage pool](https://github.com/kubernetes/kubernetes/blob/5e2cad18bceac0c1a8031ed5e11c2bb3ee3c4959/pkg/kubelet/nodestatus/setters.go#L329-L341).
- [KEP 5517](https://github.com/kubernetes/enhancements/blob/ebf37575297a991155890e7f1fae7c62bbee7ca0/keps/sig-scheduling/5517-dra-node-allocatable-resources/README.md).

[reproduction]: https://github.com/kubernetes/kubernetes/issues/144131#issuecomment-6990948125
