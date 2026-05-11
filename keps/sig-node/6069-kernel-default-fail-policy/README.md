# KEP-6069: Configurable failure mode for kubelet `protectKernelDefaults`

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Feature gate](#feature-gate)
  - [Kubelet configuration API](#kubelet-configuration-api)
  - [Behaviour matrix](#behaviour-matrix)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Current behaviour of `protectKernelDefaults` in code](#current-behaviour-of-protectkerneldefaults-in-code)
  - [Where `failProtectKernelDefaults` plugs in](#where-failprotectkerneldefaults-plugs-in)
  - [Observability for the soft-fail path](#observability-for-the-soft-fail-path)
  - [Why setting sysctls is intentionally out of scope](#why-setting-sysctls-is-intentionally-out-of-scope)
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
<!-- /toc -->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

As of today, kubelet provides a field [protectKernelDefaults](https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/#kubelet-config-k8s-io-v1beta1-KubeletConfiguration), which checks for expected sysctl values as defined [here](https://github.com/kubernetes/kubernetes/blob/5ce17ed71b124a8c89414f929a58d536e008cce7/pkg/kubelet/cm/container_manager_linux.go#L483). Depending on the value set for `protectKernelDefaults`, kubelet has the following behaviour:

- `true` → the kubelet only **checks** the values; on any mismatch it aggregates an error and **exits**.
- `false` (default) → the kubelet attempts to **write** the expected values to `/proc/sys/...`; on any write failure it aggregates an error and **exits**.

In both cases, if the host's kernel-tunable state does not match what the kubelet expects, the kubelet refuses to start. There is no way to configure the kubelet to *log a warning and continue* when this check fails.

This KEP proposes a single, additive `KubeletConfiguration` boolean: **`failProtectKernelDefaults`**. It controls whether a `protectKernelDefaults` failure is fatal for kubelet startup:

- `failProtectKernelDefaults: true` (default) → the kubelet exits on a `protectKernelDefaults` failure (today's behaviour).
- `failProtectKernelDefaults: false` → the kubelet logs a warning and continues.

This KEP **does not** add any kubelet-driven mechanism for setting sysctls. Configuring node-level sysctls remains the responsibility of the Kubernetes administrator via OS-level mechanisms (`/etc/sysctl.d/*.conf`, cloud-init, ignition, distro tooling, etc.). The motivation for that scoping is documented in [Why setting sysctls is intentionally out of scope](#why-setting-sysctls-is-intentionally-out-of-scope).

The intention of this KEP is to give the Kubernetes administrator the option to retain the kernel settings (specifically sysctls, as of today) configured on the node.

## Motivation

Kubernetes administrators cannot set some sysctls on Kubernetes nodes because they are [controlled by the kubelet](https://github.com/kubernetes/kubernetes/blob/5ce17ed71b124a8c89414f929a58d536e008cce7/pkg/kubelet/cm/container_manager_linux.go#L483). There are cases, such as running PostgreSQL on a node, where a different setting than the one set by kubelet is preferable; for [example](https://www.postgresql.org/docs/current/kernel-resources.html#LINUX-MEMORY-OVERCOMMIT), setting `vm.overcommit_memory=2` can significantly lower the chances of OOM and lead to more robust system behaviour.

Even though it is good to enforce some standards, in some use cases it is better to allow more configurable options. As of now, it is impossible to set different values for sysctls that are already managed by the kubelet.

### Goals

- Add a single boolean field `failProtectKernelDefaults` to `KubeletConfiguration`, defaulting to `true`, that controls whether a `protectKernelDefaults` failure is fatal for kubelet startup.
- Preserve current behaviour exactly when the field is unset or set to its default value (`true`). No breaking change for any cluster.
- Clearly indicate the actual behavior and any mismatch in kernel flags in the kubelet logs.

### Non-Goals

- **Adding any kubelet-driven mechanism to set node-level sysctls.** This KEP does not introduce a `kernelConfig` / `sysctls` map or anything similar for setting sysctls on the node. Reasoning is captured in [Why setting sysctls is intentionally out of scope](#why-setting-sysctls-is-intentionally-out-of-scope).
- Changing the semantics of the existing `protectKernelDefaults` field. It continues to mean what it means today.
- Changing the set of sysctls checked by `setupKernelTunables` or their expected values.
- Adding any labels or annotations to nodes to identify nodes with changed kernel defaults. It is the responsibility of the user to add relevant labels when needed at the node group / node setup level.

## Proposal

Add a field **`failProtectKernelDefaults`** to `KubeletConfiguration` which decides the kubelet startup behaviour when the kernel defaults do not match the kubelet's expectations.

- `failProtectKernelDefaults: true` (default) → the kubelet exits on a `protectKernelDefaults` failure (today's behaviour).
- `failProtectKernelDefaults: false` → the kubelet logs a warning, emits an event, and continues.

### User Stories

#### Story 1 — Kubernetes-as-a-Service provider

As a provider of Kubernetes-as-a-Service, I want to give users the option to choose their own set of sysctls, without the kubelet refusing to start when they have used a different sysctl value than the one expected by the kubelet. Users are often more aware of their use case than the kubelet's generic defaults.

An opt-in mechanism that prevents the kubelet from failing on a kernel-default mismatch will provide a better user experience and can cover diverse use cases without any intervention from the kubelet.

#### Story 2 — Running PostgreSQL on Kubernetes

As a user running PostgreSQL on Kubernetes, I want to set `vm.overcommit_memory` to `2`. This significantly lowers the chances of an OOM kill and leads to more robust system behaviour.

For someone running production workloads this is very useful, and I want better control of node settings than what the kubelet picks as a generic default.

#### Story 3 — Troubleshooting

When investigating system issues on a node, it might help to set `kernel.panic` to a higher value than the kubelet's expected `10` seconds in order to capture dumps before the node reboots.

### Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Incorrect sysctl configuration can make the system unpredictable / unstable. | By default, the recommended settings are enforced with `failProtectKernelDefaults: true` (today's behaviour). Any change away from this is strictly opt-in by the administrator. |
| Not preventing incorrect configurations. | The number of kernel settings managed by the kubelet (mainly sysctls today) is minimal, and we assume that an administrator changing these parameters is aware of what they are doing. However, it is impossible to prevent every bad configuration a user could apply. |

## Design Details

### Kubelet configuration API

A single new boolean field is added to `KubeletConfiguration` :

```yaml
apiVersion: kubelet.config.k8s.io/v1beta1
kind: KubeletConfiguration

# Existing field, unchanged.
protectKernelDefaults: true

# NEW. Default: true. Controls whether a protectKernelDefaults failure
# (mismatch in `true` mode, write failure in `false` mode) causes the
# kubelet to exit at startup.
#   true  -> kubelet exits with an error on failure (current behaviour).
#   false -> kubelet logs a warning, emits an event, and continues.
failProtectKernelDefaults: true
```

Go type sketch

```go
// KubeletConfiguration adds:
type KubeletConfiguration struct {
    // ... existing fields ...

    // failProtectKernelDefaults tells the Kubelet to fail to start if the
    // protectKernelDefaults check fails (a sysctl mismatch when
    // protectKernelDefaults is true, or a sysctl write failure when
    // protectKernelDefaults is false). When set to false, the Kubelet logs
    // a warning, emits an event, and continues running.
    // Default: true
    // +featureGate=KernelDefaultsFailPolicy
    // +optional
    FailProtectKernelDefaults *bool `json:"failProtectKernelDefaults,omitempty"`
}
```

`KernelTunableBehavior` defined in `pkg/kubelet/cm/container_manager_linux.go` (Linux only) already has `warn`, `error`, and `modify` options. `KernelTunableWarn` can be used when `failProtectKernelDefaults` is set to `false` in the [setupNode](https://github.com/kubernetes/kubernetes/blob/5ce17ed71b124a8c89414f929a58d536e008cce7/pkg/kubelet/cm/container_manager_linux.go#L539) function.

```go
// pkg/kubelet/cm/container_manager_linux.go
type KernelTunableBehavior string

const (
    KernelTunableWarn   KernelTunableBehavior = "warn"
    KernelTunableError  KernelTunableBehavior = "error"
    KernelTunableModify KernelTunableBehavior = "modify"
)

```

### Behaviour matrix

`failProtectKernelDefaults` is meaningful regardless of the value of `protectKernelDefaults`. Both branches in `setupKernelTunables` can produce an aggregated error today and abort kubelet startup; `failProtectKernelDefaults` controls the post-failure outcome of either.

| `protectKernelDefaults` | `failProtectKernelDefaults` | What the kubelet does on failure |
|---|---|---|
| `true` | `true` (default) | Reads expected sysctls; on any mismatch, **kubelet exits** (today's behaviour, preserved). |
| `true` | `false` | Reads expected sysctls; on any mismatch, kubelet **logs a warning, emits an event, and continues**. |
| `false` (default) | `true` (default) | Attempts to write expected sysctls; on any write failure, **kubelet exits** (today's behaviour, preserved). |
| `false` (default) | `false` | Attempts to write expected sysctls; on any write failure, kubelet **logs a warning, emits an event, and continues**. |

Both of the "today's behaviour, preserved" rows are identical to the current code path; this is what guarantees no breaking change for any existing cluster.

Note: with `protectKernelDefaults: false`, the kubelet still **overwrites** the relevant sysctls to its expected values on every successful startup. Administrators who want to keep custom OS-level values for those sysctls must use `protectKernelDefaults: true` combined with `failProtectKernelDefaults: false` (introduced by this KEP) so the kubelet warns instead of exiting.


### Why setting sysctls is intentionally out of scope

A natural extension of this KEP would be to let administrators declare a map of sysctls in `KubeletConfiguration` and have the kubelet apply them. That extension was considered and **deliberately left out**. The reasoning is as follows:

1. **Allowlist criteria are a feature on their own.** Letting the kubelet apply sysctls safely requires a documented allowlist of acceptable keys and a per-key validation policy. The list of sysctls available on a Linux system is extensive, and there is no practical way to define a comprehensive allowlist. Aligning them to a single standard is neither feasible nor sufficiently justified, as appropriate values are highly dependent on workload, environment, and operational requirements.
2. **There is no functional gap that only the kubelet can fill.** Anything the kubelet could write to `/proc/sys/...` can already be addressed by the Kubernetes administrator at the node level via well-established mechanisms, such as adding an entry to `/etc/sysctl.d/*.conf`, modifying `/proc/sys/*` files, using the `sysctl` command, etc.
3. **Smaller scope, easier rollback.** A boolean toggle is a one-field, one-default, one-feature-gate change. It can ship to alpha and graduate quickly, on a much shorter timeline than an allowlist-driven configuration API would justify.
4. **Future-proof.** Nothing in this KEP precludes a follow-up KEP that could let the kubelet set sysctls. Such a KEP can layer cleanly on top, with `failProtectKernelDefaults` becoming the natural failure-mode toggle for both the existing check and any new apply path.
5. **Security considerations.** Declaring sysctl key/value pairs in `KubeletConfiguration` and having the kubelet apply them widens the trust boundary, and any potential bug could have a serious impact.

The KEP therefore does **only** the failure-mode change. Anything sysctl-management-shaped is explicitly listed as a [Non-Goal](#non-goals) and is documented in user-facing docs as "use your existing OS-level mechanism".

### Test Plan

[ ] I/we understand the owners of the involved components may require updates to existing tests prior to implementation.


#### Unit tests

Cover all four combinations of `(protectKernelDefaults, failProtectKernelDefaults)` values:

- `true` / `true` → aggregated error returned on sysctl mismatch (today's behaviour).
- `true` / `false` → no error returned, warning logged.
- `false` / `true` → aggregated error returned on sysctl write failure after attempting to overwrite (today's behaviour).
- `false` / `false` → no error returned, warning logged.

#### Integration tests


#### e2e tests

There are three factors:

- sysctls: match / mismatch
- `protectKernelDefaults`: true / false
- `failProtectKernelDefaults`: true / false

For each combination, assert whether the kubelet **starts successfully** or **fails at startup** (and that logs match warnings vs hard errors where applicable).


### Graduation Criteria

#### Alpha

- Feature gate `KernelDefaultsFailPolicy` (default off).
- `failProtectKernelDefaults *bool` added to `KubeletConfiguration` (v1beta1) with default `true`.

#### Beta

- Evaluate user feedback to determine if this feature is causing any issues. If deemed necessary, revisit the impact and behaviour of the fields.

#### GA

- Sufficient feedback from users using the new behaviour in production environments.
- The feature has been stable in Beta for at least 2 Kubernetes releases.

### Upgrade / Downgrade Strategy

- **Upgrade:** After upgrading to a version that supports this KEP, the `KernelDefaultsFailPolicy` feature gate can be enabled at any time.
- **Downgrade:** On downgrading to a version without this KEP, an older kubelet will ignore the new `KubeletConfiguration` field. The default behaviour is the same as in previous versions.

### Version Skew Strategy

Behaviour is confined to the kubelet. The API server and workload APIs are unchanged. There is no version-skew impact.


## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `KernelDefaultsFailPolicy`
  - Components depending on the feature gate: `kubelet`
- [x] Other
  - Describe the mechanism: Set `failProtectKernelDefaults` in `KubeletConfiguration` to take effect on the next kubelet restart.
  - Will enabling / disabling the feature require downtime of the control plane? No.
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? A kubelet restart is required to pick up the change. No reprovisioning is needed.

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate, or setting `failProtectKernelDefaults` to `true` in `KubeletConfiguration`, has the same effect as disabling this feature. The kubelet must be restarted for any change in behaviour to take effect.

###### What happens if we reenable the feature if it was previously rolled back?

On re-enabling the feature, the kubelet re-reads `KubeletConfiguration` and applies the field's value on the next startup pass. The same functionality is restored.

###### Are there any tests for feature enablement/disablement?

During the alpha stage, unit tests for the toggle behaviour will be added to the validation code. Tests covering all combinations of `failProtectKernelDefaults` and `protectKernelDefaults` will be implemented and documented.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

For both rollout and rollback, the following behaviour will be observed:

- `failProtectKernelDefaults: true`: Users who run with the current defaults see no impact on rollout (today's behaviour).
- `failProtectKernelDefaults: false`: If the sysctls already match or match after the kubelet writes the expected values ,there is no impact. If there is a mismatch , workloads that would not have not run earlier will run with this feature.

A node will go to `NotReady` state in case of a sysctl mismatch under today's behaviour; with this feature enabled and `failProtectKernelDefaults: false`, the same node will become `Ready`. A clearer picture can be obtained by checking the kubelet logs.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be completed during alpha.
Plan: stand up a node, enable the gate with `failProtectKernelDefaults: false` and a deliberately mismatched sysctl, downgrade kubelet (gate gone), confirm the older kubelet exits as expected, re-upgrade, confirm soft-fail resumes.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?


###### How can someone using this feature know that it is working for their instance?

- [x] Events
  - Event Reason: 
- [x] API .status
  - Condition name: 
  - Other field: 
- [x] Other (treat as last resort)
  - Check the kubelet logs.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

There should be no measurable change in kubelet startup time `kubelet_node_startup_duration_seconds` with or without this KEP and the Kubelet SLO should not be impacted.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `kube_node_status_condition`.
  - Components exposing the metric: kubelet.
- [x] Other (treat as last resort)
  - Details: 

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Nodes could carry a label such as `kubernetes.io/mismatch-kernel-defaults: true/false`. (Ideally Kubernetes administrators have their own way of tracking this; adding such a label can be considered if there is sufficient user feedback.)

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

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

No Changes.

###### What are other known failure modes?

None.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect kubelet logs; ideally, there should be no impact on SLOs from this feature.

## Implementation History

- 2026-05-11: Initial draft.

## Drawbacks

- Incorrect configurations applied by the user can result in an unstable system.

## Alternatives

- Run a privileged DaemonSet to set the required sysctl after the kubelet starts: exposes a wider security boundary.
- Set sysctls after the kubelet has started using custom scripts, etc.: this method is not very convenient and is hard to operate at scale.

