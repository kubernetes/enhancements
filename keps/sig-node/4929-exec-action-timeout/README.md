# KEP-4929: ExecAction timeout for lifecycle hooks

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
  - [Execution semantics](#execution-semantics)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Validation](#validation)
  - [Kubelet behavior](#kubelet-behavior)
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
    - [Upgrade](#upgrade)
    - [Downgrade](#downgrade)
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
  - [Set a default lifecycle exec timeout](#set-a-default-lifecycle-exec-timeout)
  - [Add timeout support to the kubelet <code>/run</code> API](#add-timeout-support-to-the-kubelet-run-api)
  - [Require users to wrap commands with image-specific timeout tooling](#require-users-to-wrap-commands-with-image-specific-timeout-tooling)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation--e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubernetes lifecycle hooks support an `exec` action for both `postStart` and
`preStop`, but lifecycle `ExecAction` does not let users specify how long the
command may run. The kubelet currently invokes lifecycle exec hooks with a
timeout of `0`, which means no timeout. If a `postStart` exec hook hangs, the
container start sequence can block indefinitely. In particular, this can block
the pod worker from processing later updates such as pod deletion.

This KEP proposes adding an optional `timeoutSeconds` field to `ExecAction`.
When the field is unset or set to `0`, Kubernetes preserves the existing
behavior and does not enforce a timeout. When the field is set to a positive
value, kubelet passes that timeout to the runtime exec path for lifecycle hooks.
This gives users an API-level way to bound lifecycle exec hooks without changing
the default behavior for existing workloads.

## Motivation

Lifecycle hooks are often used to coordinate application startup and graceful
shutdown. The `exec` action is the most flexible lifecycle hook action because
it can run arbitrary commands inside the container. That flexibility also makes
it possible for hook commands to hang forever because of a blocked shell script,
a stuck helper process, or an unavailable dependency.

For `postStart`, an unbounded exec hook is especially problematic. A stuck
`postStart` hook can keep the kubelet's pod worker waiting for the hook to
return. While that worker is blocked, the kubelet might be unable to process
later pod updates, including termination of that pod. For `preStop`, an
unbounded exec hook is partially bounded by pod termination grace period, but
users still cannot express a shorter hook-specific limit without wrapping the
command in image-specific tooling such as `timeout`.

### Goals

- Allow users to set an optional timeout value for exec actions in lifecycle
  hooks to prevent arbitrary command execution from hanging indefinitely.
- Preserve existing behavior for workloads that do not opt in to a lifecycle
  exec timeout.

### Non-Goals

- This KEP does not define default timeouts for lifecycle exec hooks.
- This KEP does not change probe timeout behavior. Exec probes already use
  `Probe.timeoutSeconds`, and kubelet exec probe timeout behavior is covered by
  [KEP-1972](../1972-kubelet-exec-probe-timeouts/README.md).
- This KEP does not add timeout fields to HTTP, TCP, gRPC, or sleep lifecycle
  actions.
- This KEP does not change `terminationGracePeriodSeconds` semantics.
- This KEP does not introduce a new kubelet API parameter for `/run`.

## Proposal

Add a `timeoutSeconds` field to `ExecAction`:

```go
// ExecAction describes a "run in container" action.
type ExecAction struct {
    // Command is the command line to execute inside the container, the working
    // directory for the command is root ('/') in the container's filesystem.
    // The command is simply exec'd, it is not run inside a shell, so traditional
    // shell instructions ('|', etc) won't work. To use a shell, you need to
    // explicitly call out to that shell.
    // Exit status of 0 is treated as live/healthy and non-zero is unhealthy.
    // +optional
    // +listType=atomic
    Command []string `json:"command,omitempty" protobuf:"bytes,1,rep,name=command"`

    // TimeoutSeconds is the number of seconds after which the exec action
    // times out. A value of 0 or an unset field means no timeout, preserving
    // the existing behavior. The value must be non-negative.
    // +optional
    TimeoutSeconds int32 `json:"timeoutSeconds,omitempty" protobuf:"varint,2,opt,name=timeoutSeconds"`
}
```

The field applies when `ExecAction` is used by lifecycle hooks:

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: lifecycle-exec-timeout
spec:
  containers:
  - name: app
    image: busybox:1.36
    command: ["sh", "-c", "sleep 1d"]
    lifecycle:
      postStart:
        exec:
          command: ["sh", "-c", "setup-work || exit 1"]
          timeoutSeconds: 30
      preStop:
        exec:
          command: ["sh", "-c", "cleanup-work"]
          timeoutSeconds: 10
```

### API

`timeoutSeconds` is added to the core/v1 `ExecAction` type and the internal
core `ExecAction` type. The field is optional and defaults to `0`. A value of
`0` means no timeout.

Although `ExecAction` is shared by lifecycle hooks and probes, this KEP only
changes lifecycle hook execution. For probes, timeout behavior is already
defined by `Probe.timeoutSeconds` and KEP-1972. The kube-apiserver will reject
non-zero `probe.exec.timeoutSeconds` values and direct users to set
`probe.timeoutSeconds` instead.

The API server validates that `timeoutSeconds` is non-negative when the
`PodLifecycleExecActionTimeout` feature gate is enabled. When the feature gate
is disabled on a kube-apiserver version that supports this field, new non-zero
lifecycle hook values are not accepted. Existing pods that already have a
non-zero value remain readable and can continue to run.

### Execution semantics

When kubelet executes a lifecycle hook with `exec.timeoutSeconds`:

- `timeoutSeconds` unset or `0`: pass `0` to `RunInContainer`, preserving the
  current no-timeout behavior.
- `timeoutSeconds` greater than `0`: pass that duration to `RunInContainer`.
- If the exec action times out, the lifecycle hook fails and kubelet handles
  that failure the same way it handles other lifecycle exec failures. The
  failure message and kubelet log should make it clear that the exec action hit
  its configured timeout.

For `preStop`, the hook remains subject to the existing pod termination flow and
grace-period cancellation. The exec timeout provides an additional
hook-specific bound when it is shorter than the remaining termination grace
period. It does not extend the pod termination grace period; a `preStop`
timeout value longer than the remaining grace period can still be interrupted by
pod termination.

### User Stories (Optional)

#### Story 1

As a cluster user, I want to set a timeout on a `postStart` exec hook so that a
stuck startup script cannot block the kubelet pod worker indefinitely.

#### Story 2

As a workload author, I want to set a timeout on a best-effort `preStop` cleanup
or drain command so that it has a smaller, explicit budget inside the pod
termination grace period, leaving time for the main process to finish graceful
shutdown.

### Notes/Constraints/Caveats (Optional)

The field is intentionally opt-in. This KEP does not set a default lifecycle
exec timeout because existing workloads might intentionally run long lifecycle
hooks. Changing the default could cause previously working pods to fail during
startup or shutdown.

This KEP also avoids changing the kubelet `/run` HTTP API. A previous attempt
to add timeout support there had kubelet API impact and did not provide a
workload-level API for lifecycle hook timeout configuration.

For `preStop`, users need to choose a timeout that fits within the pod's
termination grace period. Setting `preStop.exec.timeoutSeconds` larger than
`terminationGracePeriodSeconds` does not extend termination; it only defines the
maximum time the hook may run if that much grace period remains.

### Risks and Mitigations

The main risk is user misconfiguration. A timeout that is too small can make a
valid lifecycle hook fail. This risk is mitigated by making the field opt-in and
by preserving the existing no-timeout behavior when the field is unset or `0`.

Another risk is version skew between kube-apiserver and kubelet. This is
mitigated by feature-gating the field in both components and by documenting
that an older or disabled kubelet will ignore the field and preserve the
existing no-timeout execution behavior.

## Design Details

### Validation

Validation for lifecycle `ExecAction` will be updated to reject negative
`timeoutSeconds` values.

Validation for probe `ExecAction` will reject non-zero `timeoutSeconds` values.
This avoids introducing two timeout fields for probes:

```yaml
livenessProbe:
  timeoutSeconds: 5
  exec:
    command: [...]
    timeoutSeconds: 10 # rejected; use probe.timeoutSeconds instead
```

When the `PodLifecycleExecActionTimeout` feature gate is disabled:

- new pods cannot set a non-zero lifecycle hook `exec.timeoutSeconds`;
- updates cannot newly set a non-zero lifecycle hook `exec.timeoutSeconds`;
- existing pods with a non-zero lifecycle hook value keep the field to avoid
  clearing stored data during feature gate rollback.

### Kubelet behavior

The kubelet lifecycle handler currently calls:

```go
hr.commandRunner.RunInContainer(ctx, containerID, handler.Exec.Command, 0)
```

The implementation will derive a timeout from `handler.Exec.TimeoutSeconds`.
If the feature gate is disabled, kubelet will behave as if the value is `0`.
If the feature gate is enabled and the value is greater than `0`, kubelet will
pass `time.Duration(timeoutSeconds) * time.Second` to `RunInContainer`.

No CRI API changes are expected. Kubelet already passes a timeout value through
the existing exec sync path.

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

N/A

##### Unit tests

Alpha:

- Validate that pods with lifecycle exec `timeoutSeconds` unset are accepted.
- Validate that pods with lifecycle exec `timeoutSeconds: 0` are accepted.
- Validate that pods with positive lifecycle exec `timeoutSeconds` are accepted
  when `PodLifecycleExecActionTimeout` is enabled.
- Validate that negative lifecycle exec `timeoutSeconds` values are rejected.
- Validate that non-zero probe exec `timeoutSeconds` values are rejected and
  that probe-level `timeoutSeconds` remains valid.
- Validate feature gate disablement behavior for create and update paths.
- Test kubelet lifecycle handler passes `0` to `RunInContainer` when the field
  is unset or `0`.
- Test kubelet lifecycle handler passes the configured timeout duration to
  `RunInContainer` when the field is positive and the feature gate is enabled.
- Test kubelet lifecycle handler behaves as no-timeout when the feature gate is
  disabled.

##### Integration tests

N/A for alpha.

##### e2e tests

Alpha:

- Create a pod with a `postStart` exec hook that blocks longer than
  `timeoutSeconds`; verify the hook times out and the container is handled as a
  failed lifecycle hook.
- Create a pod with a `preStop` exec hook whose command sleeps longer than
  `timeoutSeconds`; delete the pod and verify the hook does not run longer than
  the configured timeout.
- Create a pod with an unset lifecycle exec timeout and verify behavior matches
  the pre-existing no-timeout behavior.

### Graduation Criteria

#### Alpha

- API field implemented behind the `PodLifecycleExecActionTimeout` feature gate.
- Kubelet lifecycle exec path honors positive `timeoutSeconds` values.
- Unit tests for API validation and kubelet timeout propagation are completed.
- Initial e2e coverage for `postStart` and `preStop` lifecycle exec timeout is
  completed.

#### Beta

- Feedback from alpha users is addressed.
- Upgrade, downgrade, and feature gate disablement behavior is tested.
- e2e tests are stable in CI.
- Documentation is added to describe lifecycle exec timeout usage, including the
  interaction between `preStop.exec.timeoutSeconds` and pod termination grace
  period.

#### GA

- No significant bug reports from beta.
- The feature has been enabled by default for at least one release.
- e2e tests are promoted as appropriate.

### Upgrade / Downgrade Strategy

#### Upgrade

Existing workloads are not changed during upgrade. Users who enable the feature
gate can start setting `exec.timeoutSeconds` on lifecycle hooks. Workloads that
do not set the field continue to run with the current no-timeout behavior.

#### Downgrade

If the feature gate is disabled on a kube-apiserver version that supports this
field, new pods cannot set non-zero lifecycle exec timeouts. Existing pods that
already contain the field remain stored and readable, and updates are allowed to
preserve the old value without newly enabling the feature on objects that did
not already use it.

If kube-apiserver is downgraded to a version that does not know the field,
existing pods continue to run on kubelets that have already observed them.
Updates through the older kube-apiserver may reject or drop the new field
according to that version's API decoding and validation behavior. Users should
remove non-zero lifecycle exec timeouts before downgrading the control plane.

If kubelet is downgraded or the feature gate is disabled, kubelet ignores the
field and lifecycle exec hooks run with the existing no-timeout behavior.

### Version Skew Strategy

Both kube-apiserver and kubelet need the feature gate enabled for the full
feature to be available.

If kube-apiserver enables the feature but kubelet does not, the API server can
accept pods with lifecycle exec timeouts, but the kubelet ignores the timeout
and preserves existing no-timeout behavior.

If kubelet enables the feature but kube-apiserver does not, ordinary pods cannot
set non-zero lifecycle exec timeouts through the API server. Static pods on that
kubelet can use the field if the kubelet understands it and the feature gate is
enabled.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: PodLifecycleExecActionTimeout
  - Components depending on the feature gate: kube-apiserver, kubelet
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node?

###### Does enabling the feature change any default behavior?

No. The default value is `0`, which preserves the current no-timeout behavior.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate in kube-apiserver prevents new non-zero values
from being created. Disabling the feature gate in kubelet makes kubelet ignore
the value and run lifecycle exec hooks without a timeout.

###### What happens if we reenable the feature if it was previously rolled back?

New pods can again set lifecycle exec `timeoutSeconds`. Existing pods that kept
the field during rollback will have the timeout honored again by kubelets with
the feature gate enabled.

###### Are there any tests for feature enablement/disablement?

Unit tests are planned for API validation and kubelet behavior with the feature
gate enabled and disabled.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

The feature is opt-in and does not impact workloads that do not set
`exec.timeoutSeconds`. A workload that sets a timeout that is too small can fail
its lifecycle hook. Rolling back kubelet behavior makes the hook run without a
timeout again.

###### What specific metrics should inform a rollback?

No new metric is proposed for alpha. Operators can use pod event and kubelet log
signals for lifecycle hook failures. The implementation should ensure timeout
failures are distinguishable from other lifecycle exec failures in those
signals. If users see unexpected lifecycle hook timeout failures after enabling
this feature, they can disable the feature gate or remove the field from
affected workloads.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be tested before beta. For alpha, unit tests will cover feature gate
enablement and disablement behavior.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

Inspect pod specs for non-zero lifecycle `exec.timeoutSeconds` values in
`postStart` or `preStop` hooks.

###### How can someone using this feature know that it is working for their instance?

- [ ] Events
  - Event Reason:
- [ ] API .status
  - Condition name:
  - Other field:
- [x] Other (treat as last resort)
  - Details: Observe lifecycle hook completion, pod events, and kubelet logs.
    A hook command that runs longer than its configured timeout should fail, and
    the resulting failure should identify the configured timeout as the cause.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [x] Other (treat as last resort)
  - Details: Watch pod lifecycle hook failures through pod events and kubelet
    logs. Timeout failures should be identifiable separately from other exec
    failures.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A kubelet metric counting lifecycle exec hook timeouts could be useful if this
feature is widely used. This KEP does not propose a new metric for alpha.

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

Yes. `ExecAction` gains one optional scalar field. This can slightly increase
the size of pods that set lifecycle exec timeouts.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. Bounded exec actions can reduce the time a stuck lifecycle hook keeps
resources alive.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The same way any pod spec field behaves. The feature does not add new
dependencies on kube-apiserver or etcd after the pod has been observed by
kubelet.

###### What are other known failure modes?

The configured timeout can be too short for the hook command. In that case the
exec action fails. Users should increase `timeoutSeconds`, remove the field, or
fix the hook command. For `preStop`, users should also check whether the pod
termination grace period is shorter than the configured hook timeout.

If a kubelet does not support the feature or has the feature gate disabled, the
hook runs with the existing no-timeout behavior.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect affected pod specs for lifecycle exec `timeoutSeconds`, check pod
events and kubelet logs for lifecycle hook failures, verify the configured
`preStop` timeout fits within the pod termination grace period, and disable the
`PodLifecycleExecActionTimeout` feature gate or remove the field if necessary.

## Implementation History

- 2024-10-22: Initial discussion opened in kubernetes/kubernetes#128250.
- 2024-10-23: Enhancement issue kubernetes/enhancements#4929 opened.
- 2026-07-14: Initial KEP draft prepared for v1.38.

## Drawbacks

The API surface of `ExecAction` becomes larger. Users can also misconfigure a
timeout that is too small and cause lifecycle hook failures.

## Alternatives

### Set a default lifecycle exec timeout

Kubernetes could set a default timeout for lifecycle exec hooks. This would
protect users from stuck hooks without requiring any new pod spec field, but it
would be a behavior change for existing workloads and could break hooks that
legitimately run for a long time. This KEP avoids that compatibility risk.

### Add timeout support to the kubelet `/run` API

Kubernetes could add a timeout parameter to the kubelet `/run` API and thread it
through `RunInContainer`. This was explored previously, but it changes the
kubelet API and does not give workload authors a pod spec field to configure
lifecycle hook behavior. This KEP avoids changing the kubelet `/run` API.

### Require users to wrap commands with image-specific timeout tooling

Users can run shell commands such as `timeout 30s ./hook.sh`, but this requires
the container image to include the relevant shell and timeout binary. It also
pushes common lifecycle behavior into per-image scripts. A first-class API field
is easier to validate, document, and reason about.

## Infrastructure Needed (Optional)

N/A
