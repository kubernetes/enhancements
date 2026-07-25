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
  - [Bounded health checker execution](#bounded-health-checker-execution)
  - [Bounded systemd notification](#bounded-systemd-notification)
  - [Structured diagnostic logs](#structured-diagnostic-logs)
  - [Feature gate](#feature-gate)
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
  - [Only raise <code>SdNotify()</code> error log visibility](#only-raise-sdnotify-error-log-visibility)
  - [Add kubelet configuration for watchdog timeout budgets](#add-kubelet-configuration-for-watchdog-timeout-budgets)
  - [Only wrap <code>SdNotify()</code> outside the source library](#only-wrap-sdnotify-outside-the-source-library)
  - [Add metrics or events](#add-metrics-or-events)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and
  SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for
    [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    within one minor version of promotion to GA
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806)
    must be hit by
    [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation--e.g., additional design documents, links to
  mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubelet can integrate with systemd watchdog on Linux nodes. When enabled,
kubelet periodically runs watchdog health checks and sends a heartbeat to
systemd through `SdNotify()`. If heartbeats stop, systemd can restart kubelet.

Today, operators may see the eventual watchdog-triggered kubelet restart without
enough pre-restart evidence to identify why heartbeats stopped. A watchdog
health checker might have returned an error, a checker might have stalled,
`SdNotify()` might have returned an error, or the systemd notification path
might have blocked. The current diagnostics do not make these cases easy to
distinguish.

This KEP proposes bounded diagnostic guardrails for kubelet's existing systemd
watchdog path. The change adds bounded waits for watchdog health checks and
systemd notification calls, default-visible structured logs for failure paths,
and retry summary logging when notification retries are exhausted.

## Motivation

The systemd watchdog integration is meant to help recover from an unhealthy
kubelet, but a watchdog-triggered restart is difficult to investigate if kubelet
does not leave enough diagnostic information before it stops sending
heartbeats. This increases time to recovery for node operators and makes it hard
to distinguish kubelet health-check problems from systemd notification problems.

Bounded diagnostics provide root-cause hints before systemd restarts kubelet.
Operators should be able to answer whether kubelet skipped a heartbeat because
of a failed checker, a stalled checker, a returned notify error, or a blocked
notify call.

### Goals

- Identify failed watchdog health checkers by name.
- Identify watchdog health checkers that exceed their timeout budget.
- Identify `SdNotify()` calls that return errors at default-visible log levels.
- Identify `SdNotify()` calls that exceed their timeout budget.
- Summarize exhausted notification retry attempts.
- Preserve existing successful watchdog heartbeat behavior.
- Keep the watchdog loop non-blocking and reliable when an individual health
  checker or systemd notification operation stalls.
- Avoid new Kubernetes API, kubelet configuration, metric, event, or
  NodeCondition surface in the initial implementation.

### Non-Goals

- This KEP does not add kubelet configuration fields for watchdog timeout
  budgets.
- This KEP does not add metrics, events, or NodeConditions.
- This KEP does not initially add Node Problem Detector integration.
- This KEP does not redesign kubelet health checking.
- This KEP does not define node self-healing behavior beyond existing systemd
  watchdog behavior.
- This KEP does not dynamically disable watchdog health checks that fail or
  time out.
- This KEP does not change the non-Linux watchdog implementation.
- This KEP does not guarantee forced cancellation of health checker or
  `SdNotify()` internals that block in non-cancellable code.

## Proposal

Enhance kubelet's Linux systemd watchdog loop with internal diagnostic
guardrails:

- Run watchdog health checkers with bounded wait time.
- Treat a health checker timeout as a watchdog health check failure for the
  current iteration.
- Call `SdNotify()` with bounded wait time.
- Treat a notify timeout as a failed notification attempt for the current
  iteration.
- Log `SdNotify()` returned errors at a default-visible level.
- Preserve existing notification retry/backoff semantics.
- Emit a structured retry summary when notification retries are exhausted.

This proposal is intentionally internal to kubelet. It does not add new API
surface and does not change how users enable systemd watchdog.

The alpha implementation is guarded by the `KubeletWatchdogDiagnostics`
kubelet feature gate. The gate gives SIG Node and operators a rollback switch
while timeout budgets and log volume are validated.

During implementation, kubelet should evaluate whether the underlying
`SdNotify()` path can accept cancellation directly. If the current source
library cannot support context-aware notification calls, the implementation may
need a small library contribution or kubelet-local adapter. This KEP does not
require a broad redesign of the notification library, but it should not assume
that a goroutine wrapper is the only viable implementation.

### User Stories

#### Story 1: Triage watchdog-triggered kubelet restarts

A cluster operator receives an alert that kubelet restarted on a node, for
example from existing process restart monitoring or systemd journal entries. The
operator checks the normal kubelet logs around the restart timestamp and finds a
structured watchdog diagnostic entry. The entry indicates whether the last
watchdog iteration failed because a health checker returned an error, a checker
timed out, `SdNotify()` returned an error, or notification retries were
exhausted. The operator uses that evidence to decide whether to investigate
kubelet health, node-local systemd notification delivery, or a failing watchdog
checker.

#### Story 2: Preserve evidence at default log levels

An on-call SRE investigates a watchdog-triggered kubelet restart after the node
has already recovered. The SRE only has the default kubelet logs collected from
the affected node and cannot reproduce the failure with higher verbosity. The
SRE finds default-visible watchdog failure logs with stable fields such as the
operation, elapsed time, timeout budget, retry attempt, and returned error when
available. Those fields let the SRE preserve the incident evidence and avoid
classifying the restart only as an unexplained kubelet process restart.

#### Story 3: Separate node environment issues from kubelet health issues

A node operator compares kubelet logs with systemd journal entries after a
watchdog restart. If kubelet reports a watchdog health-check failure, the
operator investigates kubelet internals and the failing checker. If kubelet
reports `SdNotify()` errors, notification timeouts, or exhausted notification
retries, the operator investigates node-local systemd notification delivery
instead. The diagnostic distinction reduces the chance of misdiagnosing a node
environment problem as a kubelet health problem, or the reverse.

#### Story 4: Investigate repeated node restarts at fleet scale

A platform engineer sees repeated kubelet watchdog restarts across multiple
nodes from existing restart-count monitoring. The engineer queries collected
kubelet logs for the structured watchdog operation fields and groups failures by
checker name, notify error, timeout, and retry exhaustion. This helps determine
whether the incident is isolated to one machine, correlated with a node image or
systemd configuration, or likely caused by broader kubelet behavior.

### Notes/Constraints/Caveats

Timeouts in this KEP are diagnostic boundaries. Go cannot safely terminate an
arbitrary goroutine that is blocked in non-cancellable code. If a checker or
`SdNotify()` call is wrapped with a timeout and does not return, kubelet can stop
waiting for that operation and log the timeout, but the underlying operation may
continue until the blocked call returns.

The implementation must avoid repeatedly creating unbounded blocked goroutines
across watchdog iterations. If the implementation uses goroutines to enforce
bounded waits, it should avoid starting duplicate copies of an operation while a
previous copy is still known to be in flight.

Concretely, kubelet must not start another instance of the same watchdog health
checker while its previous invocation is still in flight. If a previous
invocation is still running at the next watchdog tick, kubelet should log that
the checker is still in flight and treat the checker as failed for that
iteration. The same single-flight rule applies to `SdNotify()` attempts: if a
previous notify call is still in flight, kubelet should not start another
notify call for the current iteration.

### Risks and Mitigations

- A timeout budget that is too small could skip watchdog notifications on slow
  but healthy nodes. The timeout budget should be derived from the existing
  watchdog notification interval instead of using a fixed global value.
- Wrapping blocking calls can leave goroutines running after timeout. The
  implementation must treat timeouts as diagnostic boundaries and avoid
  unbounded goroutine accumulation.
- Dynamically disabling a failing or timed-out watchdog health checker could
  make the watchdog loop appear healthy while hiding a real kubelet health
  problem. The initial implementation should keep failed checks visible and
  continue treating them as failed for the current watchdog iteration.
- Default-visible notify error logs could be noisy in repeatedly failing
  environments. The implementation should log exceptional failure and timeout
  paths at default visibility while keeping success logs at high verbosity.
- SIG Node may decide this does not require a KEP. The proposal is scoped so it
  can also serve as focused implementation rationale if maintainers prefer a
  direct PR.

## Design Details

### Current kubelet watchdog flow

On Linux, kubelet creates a watchdog health checker using
`SdWatchdogEnabled(false)`. When systemd watchdog is not enabled, kubelet does
not start watchdog health checking. When watchdog is enabled, kubelet uses half
of the systemd watchdog timeout as its notification interval.

On each watchdog tick, kubelet currently:

1. Runs all configured watchdog health checkers serially.
2. Skips notifying systemd if any checker returns an error.
3. Calls `SdNotify(false)` through the watchdog client when all checkers pass.
4. Retries notification using the existing exponential backoff path.

The current flow does not bound each checker and does not bound the notify call.

### Bounded health checker execution

Each watchdog health checker should be run with a timeout budget derived from
the existing watchdog notification interval. The initial implementation should
use a single iteration deadline equal to the watchdog notification interval and
derive per-operation budgets from that interval. For example, kubelet can divide
the interval into slots for all registered health checkers plus one notify slot,
then clamp each slot to a small bounded range such as one to ten seconds. The
exact constants should be finalized during implementation review, but the
invariant is that checker execution and notification retries for one watchdog
tick must not consume more than the current watchdog interval.

If a checker returns an error before the timeout, kubelet should keep the
existing behavior of skipping the systemd notification for the current
iteration.

If a checker exceeds its timeout budget, kubelet should log a structured
diagnostic entry containing at least:

- `operation`: `watchdog_health_check`
- `checker`: the health checker name
- `elapsed`: how long kubelet waited
- `timeout`: the timeout budget
- `watchdogInterval`: the watchdog notification interval

A checker timeout should be treated as a failed watchdog health check for the
current iteration.

### Bounded systemd notification

Each `SdNotify()` attempt should be run with a timeout budget derived from the
same iteration deadline used for health checkers. Retry backoff and retry
attempts should fit inside the remaining iteration budget. If the remaining
budget is exhausted, kubelet should stop retrying for the current iteration and
log an exhausted retry summary.

If `SdNotify()` returns an error, kubelet should log the returned error at a
default-visible level and continue through the existing retry/backoff path while
budget remains.

If `SdNotify()` does not return before the timeout budget expires, kubelet
should log a structured diagnostic entry containing at least:

- `operation`: `watchdog_notify`
- `attempt`: the notify retry attempt
- `elapsed`: how long kubelet waited
- `timeout`: the timeout budget
- `watchdogInterval`: the watchdog notification interval

A notify timeout should be treated as a failed notification attempt for the
current iteration.

### Structured diagnostic logs

Failure logs should use stable structured fields so node operators can query
logs consistently:

- `operation`
- `checker`, when applicable
- `attempt`, when applicable
- `elapsed`
- `timeout`, when applicable
- `watchdogInterval`
- returned error, when applicable

Successful watchdog notifications should remain high-verbosity logs to avoid
increasing normal kubelet log volume.

### Feature gate

This KEP introduces the `KubeletWatchdogDiagnostics` kubelet feature gate.

For alpha, the gate is disabled by default. When enabled, kubelet applies the
bounded watchdog health-check execution, bounded `SdNotify()` waits,
default-visible `SdNotify()` error logging, and exhausted retry summaries
described in this KEP.

The gate is proposed even though the enhancement does not add API surface
because bounded waits can change watchdog failure-path behavior. A feature gate
keeps the initial rollout reversible while SIG Node evaluates timeout budgets
and default-visible log volume.

### Test Plan

#### Prerequisite testing updates

No prerequisite test infrastructure changes are expected.

#### Unit tests

Unit tests should cover:

- Watchdog disabled behavior remains unchanged.
- `KubeletWatchdogDiagnostics=false` preserves the previous watchdog behavior.
- Successful health checks and successful notification preserve the existing
  heartbeat path.
- A health checker returned error skips notification and logs the checker name.
- A health checker timeout skips notification and logs checker name, elapsed
  time, timeout budget, and watchdog interval.
- A returned `SdNotify()` error is logged at a default-visible level.
- An `SdNotify()` timeout is logged and treated as a failed notification
  attempt.
- Notification retries that are exhausted produce a retry summary.
- A later successful retry after earlier failures still sends the heartbeat.
- Non-Linux watchdog implementation remains a no-op where applicable.

#### Integration tests

No integration tests are required for the initial implementation.

#### e2e tests

No e2e tests are required for the initial implementation because this KEP does
not add Kubernetes API behavior and targets kubelet's internal systemd watchdog
diagnostics. Unit tests with mocked watchdog clients and health checkers should
cover the behavior.

### Graduation Criteria

#### Alpha

- KEP merged as implementable.
- Unit tests cover success, failure, timeout, retry, and disabled paths.
- Alpha implementation merged behind `KubeletWatchdogDiagnostics`, disabled by
  default.
- Timeout budget constants and single-flight behavior are documented in code and
  covered by unit tests.

#### Beta

- SIG Node confirms the diagnostic behavior is useful after at least one alpha
  release.
- No unresolved reports show excessive default-visible log noise from normal
  kubelet operation.
- Known timeout budget issues are resolved, adjusted, or documented.
- `KubeletWatchdogDiagnostics` is enabled by default, or SIG Node explicitly
  decides that it should remain disabled by default for another release.

#### GA

- The behavior has been enabled by default for at least one release without
  significant regressions.
- No unresolved production-readiness concerns remain for timeout behavior or log
  volume.
- `KubeletWatchdogDiagnostics` is locked or removed according to Kubernetes
  feature gate policy.

### Upgrade / Downgrade Strategy

Upgrading kubelet to a version with this enhancement adds diagnostic logs on
failure and timeout paths for nodes using systemd watchdog. Successful watchdog
heartbeats should remain compatible.

Downgrading kubelet removes the new diagnostic guardrails and returns to the
previous watchdog behavior. No persisted state, API object, or configuration
migration is involved.

### Version Skew Strategy

This enhancement is kubelet-local. It does not require apiserver,
controller-manager, scheduler, CRI, or kubelet-to-kubelet version coordination.
Different nodes may run kubelet versions with or without these diagnostics, and
different nodes may enable or disable `KubeletWatchdogDiagnostics`
independently.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

This feature is enabled by running a kubelet version that includes the
diagnostic guardrails on a Linux node with systemd watchdog enabled and the
`KubeletWatchdogDiagnostics` kubelet feature gate enabled.

###### Does enabling the feature change any default behavior?

It changes failure-path diagnostics for kubelet's Linux systemd watchdog path.
Successful watchdog notification behavior should remain unchanged.

###### Can the feature be disabled once it has been enabled?

During alpha, disabling can be done by turning off the
`KubeletWatchdogDiagnostics` kubelet feature gate and restarting kubelet. A full
rollback to a kubelet version without this enhancement also removes the
diagnostic behavior.

###### What happens if we reenable the feature if it was previously rolled back?

The diagnostic behavior is restored. No persisted state needs migration.

###### Are there any tests for feature enablement/disablement?

Unit tests should verify watchdog disabled behavior,
`KubeletWatchdogDiagnostics=false`, and normal successful watchdog behavior
remain unchanged.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail?

The most likely rollout risk is unexpectedly noisy failure logs or a timeout
budget that is too strict for some nodes.

###### What specific metrics should inform a rollback?

No new metrics are proposed. Operators can use existing kubelet restart signals
and kubelet logs to detect unexpected behavior. Node-local logs are the intended
alpha diagnostic interface because the user story is investigation of a local
kubelet watchdog restart, often before any cluster-level signal can be emitted.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This should be covered by unit tests for compatibility paths before graduating
beyond alpha. No persisted state or API migration is involved.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

This is not workload-specific. It is in use when kubelet's systemd watchdog
integration is enabled on a Linux node running a version with this enhancement.

###### How can someone using this feature know that it is working for their instance?

On failure paths, kubelet logs should identify watchdog health check failures,
health check timeouts, notify errors, notify timeouts, and exhausted notify
retries with structured fields.

###### What are the reasonable SLOs?

No new SLOs are introduced. The enhancement improves diagnostic evidence for
the existing systemd watchdog path.

###### What are the SLIs an operator can use to determine the health of the service?

No new SLIs are introduced. Operators can use existing kubelet process health,
restart counts, systemd watchdog behavior, and kubelet logs.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

Metrics for watchdog checker timeout count or notify failure count could be
useful in the future, but they are intentionally out of scope for the initial
proposal to avoid expanding metric stability and label cardinality concerns.

This KEP does not propose Node Problem Detector integration for alpha. The
initial recommendation is to keep the diagnostic signal in kubelet logs because
the first user story is local post-restart investigation using kubelet and
systemd logs. If SIG Node finds that these diagnostics need a cluster-visible
surface, a follow-up enhancement could evaluate whether NPD conditions, events,
or another node-level reporting mechanism is appropriate.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No cluster service dependency is introduced.

###### Does this feature depend on any specific services running on the node?

It applies only when kubelet's systemd watchdog integration is enabled, which
depends on systemd watchdog support on the node.

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

No Kubernetes API operation should be affected. The watchdog loop may do small
additional bookkeeping on failure and timeout paths.

###### Will enabling / using this feature result in non-negligible increase of resource usage?

The expected overhead is negligible on successful paths. Implementations that
use goroutines for timeout handling must avoid unbounded accumulation if an
operation remains blocked.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature is kubelet-local and does not depend on apiserver or etcd.

###### What are other known failure modes?

- A checker can block in non-cancellable code and outlive the timeout wrapper.
- `SdNotify()` can block in the systemd notify path and outlive the timeout
  wrapper.
- Timeout budgets can be too aggressive for slow environments.
- Repeated notify failures can produce repeated default-visible logs.

###### What steps should be taken if SLOs are not being met to determine the problem?

Inspect kubelet logs for structured watchdog diagnostics with
`operation=watchdog_health_check` or `operation=watchdog_notify`. Compare those
logs with systemd journal entries and kubelet restart timestamps.

## Implementation History

- 2026-07-19: Enhancement issue
  [kubernetes/enhancements#6247](https://github.com/kubernetes/enhancements/issues/6247)
  created.

## Drawbacks

This change adds complexity to a small kubelet watchdog package and may require
careful implementation to avoid goroutine accumulation after timeout. It also
adds default-visible logs for failure paths, which can increase log volume in
broken environments.

## Alternatives

### Only raise `SdNotify()` error log visibility

This would be the smallest change, but it would not diagnose stalled health
checkers or blocked notify calls. Operators would still lack root-cause evidence
for important watchdog restart cases.

### Add kubelet configuration for watchdog timeout budgets

This would provide operator control, but it would add kubelet configuration API
surface and significantly increase review scope. The initial proposal keeps
timeouts internal and derived from the existing watchdog interval.

### Only wrap `SdNotify()` outside the source library

A kubelet-local wrapper around `SdNotify()` would minimize changes to
dependencies, but it may leave blocked notification calls running until the
underlying operation returns. During implementation, kubelet should evaluate
whether the notification source library can support cancellation directly. A
small context-aware library change may be preferable if it avoids lingering
blocked notification operations without expanding this KEP into a broad library
redesign.

### Add metrics or events

Metrics or events could improve fleet-level visibility, but they introduce
metric stability, label cardinality, event reliability, and API review concerns.
Structured logs are a smaller first step and directly support the pre-restart
diagnostic user story.

## Infrastructure Needed

No new infrastructure is required.
