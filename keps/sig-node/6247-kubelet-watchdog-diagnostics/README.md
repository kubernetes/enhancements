# KEP-6247: Kubelet Systemd Watchdog Diagnostic Guardrails

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Triage watchdog-triggered kubelet restarts](#story-1-triage-watchdog-triggered-kubelet-restarts)
    - [Story 2: Preserve evidence at default log levels](#story-2-preserve-evidence-at-default-log-levels)
    - [Story 3: Separate node environment issues from kubelet health issues](#story-3-separate-node-environment-issues-from-kubelet-health-issues)
    - [Story 4: Investigate repeated node restarts at fleet scale](#story-4-investigate-repeated-node-restarts-at-fleet-scale)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Current kubelet watchdog flow](#current-kubelet-watchdog-flow)
  - [Health checker failure logging](#health-checker-failure-logging)
  - [Notification failure logging](#notification-failure-logging)
  - [Structured diagnostic logs](#structured-diagnostic-logs)
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
  - [Timeout and cancellation](#timeout-and-cancellation)
  - [Metrics, events, or NPD](#metrics-events-or-npd)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required prior to targeting a milestone or release.

- [ ] (R) Enhancement issue is in the release milestone and links to this KEP.
- [ ] (R) KEP approvers have approved status `implementable`.
- [ ] (R) Design details and test plan are documented.
- [ ] (R) Graduation criteria are documented.
- [ ] (R) Production readiness review is completed and approved.
- [ ] Implementation History and supporting documentation are up to date.

## Summary

On Linux nodes, kubelet can use the systemd watchdog. It periodically runs
health checkers and sends a heartbeat through `SdNotify()`. If heartbeats stop,
systemd can restart kubelet.

This KEP makes existing watchdog failures easier to diagnose. It adds
default-visible structured logs for checker errors and `SdNotify()` errors, plus
one summary when existing notification retries are exhausted. The new
diagnostics are controlled by the Alpha `KubeletWatchdogDiagnostics` feature
gate, which is disabled by default. The feature does not change watchdog
execution, timing, retry count, socket behavior, or cancellation.

## Motivation

A watchdog-triggered kubelet restart can currently leave insufficient evidence
to distinguish a checker error from a notification error. The existing retry
path also does not provide a concise default-visible summary after all attempts
fail.

### Goals

- Identify failed watchdog health checkers by name and error.
- Make returned `SdNotify()` errors visible at the default log level.
- Summarize exhausted notification retries.
- Preserve existing successful heartbeat and failure behavior.
- Keep the change small and kubelet-local.

### Non-Goals

- Checker or notification timeouts.
- Cancellation, goroutine wrappers, or socket adapters.
- Modification or upgrade of `go-systemd`.
- Metrics, events, NodeConditions, or Node Problem Detector integration.
- Dynamic checker disablement, self-healing, or automatic thread dumps.
- Changes to non-Linux watchdog behavior.
- Diagnostics for operations that block without returning an error.

## Proposal

Keep the existing watchdog control flow and add diagnostics at existing failure
points:

1. A checker error logs its name and error at the default level, then preserves
   the existing fail-fast behavior.
2. An `SdNotify()` error logs at the default level, then preserves the existing
   retry and backoff behavior.
3. Exhausted retries emit one structured summary with the attempt count and
   final error/result.

The Alpha `KubeletWatchdogDiagnostics` feature gate controls all three new
diagnostic outputs and is disabled by default. When the gate is disabled, the
existing watchdog control flow and logging remain unchanged. Enabling the gate
only increases failure-log visibility; it does not change watchdog behavior.

### User Stories

#### Story 1: Triage watchdog-triggered kubelet restarts

After enabling `KubeletWatchdogDiagnostics`, an operator detects a kubelet
restart from existing process or systemd monitoring, checks kubelet logs and the
systemd journal, and finds a structured entry identifying a failed checker,
notification error, or exhausted retries.

#### Story 2: Preserve evidence at default log levels

An SRE investigating a recovered node where `KubeletWatchdogDiagnostics` is
enabled uses default kubelet logs and finds the failure entry without enabling
higher verbosity. The SRE preserves the error and retry evidence for the
incident record.

#### Story 3: Separate node environment issues from kubelet health issues

An operator compares kubelet and systemd journal entries. A named checker error
leads to kubelet investigation; an `SdNotify()` error leads to node-local
systemd notification investigation.

#### Story 4: Investigate repeated node restarts at fleet scale

A platform engineer queries externally collected kubelet logs by operation,
checker, error, and retry summary to determine whether repeated restarts are
isolated or correlated with a node image or systemd configuration.

### Notes/Constraints/Caveats

This KEP does not bound or cancel a checker or notification call that blocks
without returning. Such failures may produce no diagnostic entry and remain
subject to existing watchdog behavior.

Diagnostic logs are best effort. High-latency or stalled journal/log storage may
delay or prevent persistence before systemd terminates kubelet. Operators should
correlate kubelet logs with the systemd journal and available core or
process-stack evidence.

### Risks and Mitigations

- Additional default-visible logs may be noisy during repeated failures. The
  feature gate provides an immediate rollback mechanism; stable fields and rate
  limiting reduce repeated failure records while it is enabled.
- The change could accidentally alter retry behavior. Tests must verify existing
  retry count, backoff, and successful retry behavior.
- Timeout and cancellation are intentionally deferred because they require a
  separate lifecycle and watchdog-budget design.

## Design Details

### Current kubelet watchdog flow

On Linux, kubelet obtains the systemd watchdog timeout and uses half of it as
the notification interval. Each iteration runs health checkers serially. An
error skips notification for that iteration. If all checkers pass, kubelet
calls `SdNotify(false)` and uses the existing exponential backoff retry path.

### Health checker failure logging

When a checker returns an error, log a default-visible structured event with:

- `operation=watchdog_health_check`;
- `checker=<checker name>`;
- `result=error`;
- the returned error.

After logging, return through the existing fail-fast path. No timeout, new
goroutine, parallel execution, or single-flight state is added.

### Notification failure logging

When `SdNotify()` returns an error, log a default-visible structured event with:

- `operation=watchdog_notify`;
- `attempt=<retry attempt>`;
- `result=error`;
- `watchdog_interval=<interval>`;
- the returned error.

`ack=false, err=nil` retains the existing terminal unsupported behavior. A
normal returned error retains the existing retry policy. When retries end,
emit one summary containing the final result/error and total attempts.

### Structured diagnostic logs

Failure records use stable fields: `operation`, `checker` when applicable,
`attempt` when applicable, `watchdog_interval`, `result`, and `error` when
available. Success logs remain high verbosity. Repeated failures may be
rate-limited, but the first failure in a window should be retained. These new
records are emitted only when `KubeletWatchdogDiagnostics` is enabled.

### Test Plan

#### Prerequisite testing updates

No prerequisite infrastructure changes are required.

#### Unit tests

Unit tests should verify:

- `KubeletWatchdogDiagnostics` is disabled by default;
- with the feature gate disabled, no new diagnostic records or retry summary
  are emitted and existing control flow is unchanged;
- with the feature gate enabled, all three new diagnostic paths are active;
- disabling the feature after enabling it stops the new diagnostics, and
  reenabling it restores them without state cleanup or migration;
- watchdog-disabled behavior is unchanged;
- successful checker and notification behavior is unchanged;
- checker error logs the checker name and error at the default level;
- checker error still skips notification and remains fail-fast;
- notification returned errors are default-visible;
- `ack=false, err=nil` retains terminal unsupported behavior;
- existing retry count and backoff behavior are unchanged;
- a later successful retry still sends the heartbeat;
- exhausted retries emit exactly one summary with attempts and final error;
- repeated failure records are rate-limited;
- non-Linux watchdog behavior remains a no-op where applicable.

#### Integration tests

No integration tests are required for the initial implementation. A fake
watchdog client and test log sink cover the unchanged transport and control
flow.

#### e2e tests

No e2e tests are required because this KEP adds no Kubernetes API behavior.

### Graduation Criteria

#### Alpha

- KEP is merged as implementable.
- `KubeletWatchdogDiagnostics` is disabled by default.
- Unit tests cover gate enablement and disablement, success, checker failure,
  notify failure, retry, and logging.
- Implementation does not change watchdog execution or timing.

#### Beta

- SIG Node confirms the diagnostics are useful after at least one alpha release.
- No unresolved reports show unacceptable default-visible log noise.
- Existing retry and heartbeat behavior has no regression.
- `KubeletWatchdogDiagnostics` is enabled by default.

#### GA

- Diagnostics have been enabled by default for at least one release without
  significant regressions.
- No unresolved production-readiness concerns remain for usefulness or log
  volume.
- The `KubeletWatchdogDiagnostics` feature gate is locked on and subsequently
  removed according to the Kubernetes feature-gate lifecycle.

### Upgrade / Downgrade Strategy

Upgrading to an Alpha kubelet adds the disabled-by-default
`KubeletWatchdogDiagnostics` feature gate. Enabling it requires setting the gate
for kubelet and restarting kubelet; no node reprovisioning is required.
Disabling the gate and restarting kubelet removes the additional diagnostics.
Downgrading to a version without the gate also removes them. Successful
heartbeat behavior is unchanged, and no state or configuration migration is
involved.

### Version Skew Strategy

The change is kubelet-local and requires no coordination with other components.
Nodes may run versions with and without the feature gate, and the gate may be
enabled independently on each node.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `KubeletWatchdogDiagnostics`
  - Components depending on the feature gate: `kubelet`

The gate is disabled by default during Alpha. Enabling or disabling it requires
updating the kubelet feature-gate configuration and restarting kubelet. This
causes kubelet downtime on that node but does not require control-plane downtime
or node reprovisioning.

###### Does enabling the feature change any default behavior?

The gate is disabled by default, so upgrading does not change default behavior.
When an operator enables it, only failure-log visibility and retry summary
content change. Watchdog timing, execution, retry count, and successful
heartbeat behavior do not change.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Set `KubeletWatchdogDiagnostics=false` and restart kubelet. The new
diagnostic records stop, while watchdog behavior remains unchanged. No persisted
state, workload change, or configuration migration is involved.

###### What happens if we reenable the feature if it was previously rolled back?

Set `KubeletWatchdogDiagnostics=true` and restart kubelet. The diagnostic logs
and retry summary resume with the same behavior as the initial enablement. There
is no persisted feature state to recover or reconcile.

###### Are there any tests for feature enablement/disablement?

Unit tests will verify that the gate is disabled by default, that disabling it
emits none of the new diagnostic records, and that enabling it activates the
checker error, notification error, and retry-exhaustion summary paths. Tests
will also cover disablement after enablement and reenabling after rollback; no
persisted data or API conversion tests are required.

### Rollout, Upgrade and Rollback Planning

The main rollout risk is noisy logs in repeatedly failing environments. The
feature can be rolled back by disabling `KubeletWatchdogDiagnostics` and
restarting kubelet. No new metrics are proposed; existing kubelet restart
signals, kubelet logs, and systemd journal entries should inform rollback. Unit
tests verify compatibility because no persisted state or API migration is added.
No deprecations or removals are introduced.

### Monitoring Requirements

Operators can determine whether the feature is enabled from the kubelet
feature-gate configuration and can confirm the underlying watchdog setting from
the kubelet systemd unit. Failure logs and retry summaries are the diagnostic
signal when a failure occurs; their absence alone does not show that the feature
is disabled. Fleet-wide analysis depends on external log collection; metrics,
events, and NPD remain future options.

### Dependencies

No cluster service dependency is introduced. The change observes the existing
systemd watchdog integration and adds no new node service dependency.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The kubelet watchdog path remains node-local and does not call the
Kubernetes API.

###### Will enabling / using this feature result in introducing new API types?

No. This KEP adds no Kubernetes API types or fields.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No. The feature does not use cloud-provider APIs.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No. The feature neither creates nor updates Kubernetes API objects.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. The existing watchdog execution, notification interval, retry count, and
backoff remain unchanged. The additional work is limited to emitting logs after
a returned checker or notification error.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No material steady-state increase is expected. The gate is disabled by default,
and, when enabled, additional CPU and I/O occur only on watchdog failure paths
to emit structured records and one retry-exhaustion summary. Repeated records
are rate-limited to bound log volume.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No. The feature creates no goroutines, sockets, files, API objects, or persistent
state. Rate limiting bounds the added failure-path log records; unit tests cover
the retry-exhaustion summary and repeated-failure rate limiting.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The feature is node-local and does not call the API server or etcd. Their
unavailability does not directly affect diagnostic logging. A health checker
that reports an error as a consequence of another component's unavailability is
logged in the same way as any other returned checker error.

###### What are other known failure modes?

- If a checker or `SdNotify()` blocks without returning, this feature cannot emit
  a returned-error record. Detect the missed heartbeat or kubelet restart through
  existing process or systemd monitoring. Inspect the systemd journal and any
  available core dump or process-stack evidence. Timeout and cancellation remain
  outside this KEP.
- High-latency or stalled journal/log storage may delay or lose diagnostic
  records before systemd terminates kubelet. Correlate all available node-local
  evidence; this feature does not guarantee durable log persistence.
- Repeated failures may produce noisy logs. Records may be rate-limited while
  retaining the first failure in a window. If the volume remains unacceptable,
  disable `KubeletWatchdogDiagnostics` and restart kubelet.

Unit tests cover returned checker errors, returned notification errors, retry
exhaustion, and rate limiting. Blocking calls and failures of the underlying log
storage are documented limitations rather than behaviors introduced by this
feature.

###### What steps should be taken if SLOs are not being met to determine the problem?

Confirm the kubelet feature-gate configuration and the systemd watchdog setting,
then inspect kubelet logs and the systemd journal around affected restarts.
Correlate the structured `operation`, `checker`, `attempt`, `result`, and `error`
fields across affected nodes. If no structured failure record exists, inspect
available core dumps or process-stack evidence and determine whether a checker,
notification call, or logging path blocked before returning.

## Implementation History

- 2026-07-19: Enhancement issue
  [kubernetes/enhancements#6247](https://github.com/kubernetes/enhancements/issues/6247)
  created.

## Drawbacks

When enabled, the change increases default-visible log volume during failures
and provides no additional evidence when an operation blocks without returning.
Timeout and cancellation are intentionally left to a future proposal.

## Alternatives

### Timeout and cancellation

This would diagnose stalled operations, but the current checker and
`go-systemd` interfaces do not support it. It requires a separate lifecycle,
compatibility, and watchdog-budget design.

### Metrics, events, or NPD

These could improve fleet-level visibility, but add stability, reliability,
cardinality, and API review concerns. Structured logs are the smallest first
step for local post-restart investigation.

## Infrastructure Needed

No new infrastructure is required.
