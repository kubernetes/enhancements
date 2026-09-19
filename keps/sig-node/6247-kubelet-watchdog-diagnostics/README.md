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
one summary when existing notification retries are exhausted. It does not
change watchdog execution, timing, retry count, socket behavior, or cancellation.

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

No feature gate is proposed because this change only increases failure-log
visibility and adds a summary. It does not change watchdog behavior.

### User Stories

#### Story 1: Triage watchdog-triggered kubelet restarts

An operator detects a kubelet restart from existing process or systemd
monitoring, checks kubelet logs and the systemd journal, and finds a structured
entry identifying a failed checker, notification error, or exhausted retries.

#### Story 2: Preserve evidence at default log levels

An SRE investigates a recovered node using default kubelet logs and finds the
failure entry without enabling higher verbosity. The SRE preserves the error and
retry evidence for the incident record.

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

- Additional default-visible logs may be noisy during repeated failures. Use
  stable fields and rate-limit repeated failure records.
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
rate-limited, but the first failure in a window should be retained.

### Test Plan

#### Prerequisite testing updates

No prerequisite infrastructure changes are required.

#### Unit tests

Unit tests should verify:

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
- Unit tests cover success, checker failure, notify failure, retry, and logging.
- Implementation does not change watchdog execution or timing.

#### Beta

- SIG Node confirms the diagnostics are useful after at least one alpha release.
- No unresolved reports show unacceptable default-visible log noise.
- Existing retry and heartbeat behavior has no regression.

#### GA

- Diagnostics have been enabled by default for at least one release without
  significant regressions.
- No unresolved production-readiness concerns remain for usefulness or log
  volume.

### Upgrade / Downgrade Strategy

Upgrading kubelet adds failure diagnostics and retry summaries. Successful
heartbeat behavior is unchanged. Downgrading kubelet removes the additional
diagnostics. No state or configuration migration is involved.

### Version Skew Strategy

The change is kubelet-local and requires no coordination with other components.
Nodes may run versions with and without the additional diagnostics.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

There is no independent feature gate. It is enabled by upgrading kubelet and
removed by downgrading or reverting kubelet.

###### Does enabling the feature change any default behavior?

Only failure-log visibility and retry summary content change. Watchdog timing,
execution, retry count, and successful heartbeat behavior do not change.

###### Can the feature be disabled once it has been enabled?

Binary rollback removes the behavior; no persisted state or configuration
migration is needed.

###### What metrics should inform a rollback?

No new metrics are proposed. Use existing kubelet restart signals, kubelet logs,
and systemd journal entries.

### Rollout, Upgrade and Rollback Planning

The main rollout risk is noisy logs in repeatedly failing environments. Unit
tests verify compatibility because no persisted state or API migration is added.
No deprecations or removals are introduced.

### Monitoring Requirements

The feature is present when running a kubelet version containing the change.
Operators can confirm the underlying watchdog setting from the kubelet systemd
unit. Failure logs and retry summaries are the diagnostic signal. Fleet-wide
analysis depends on external log collection; metrics, events, and NPD remain
future options.

### Dependencies

No cluster service dependency is introduced. The change observes the existing
systemd watchdog integration and adds no new node service dependency.

### Scalability

No API calls, API types, cloud-provider calls, or API object data are added. No
watchdog timing or Kubernetes operation latency changes. Extra work is limited
to failure-path log records and one retry summary.

### Troubleshooting

Inspect kubelet logs and the systemd journal around the restart. If no structured
failure record exists, inspect available core dumps or process-stack evidence;
the failure may have occurred in a blocking operation outside this KEP. High
latency logging storage may delay or lose records, and repeated failures may
still produce noisy logs despite rate limiting.

## Implementation History

- 2026-07-19: Enhancement issue
  [kubernetes/enhancements#6247](https://github.com/kubernetes/enhancements/issues/6247)
  created.

## Drawbacks

The change increases default-visible log volume during failures and provides no
additional evidence when an operation blocks without returning. Timeout and
cancellation are intentionally left to a future proposal.

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
