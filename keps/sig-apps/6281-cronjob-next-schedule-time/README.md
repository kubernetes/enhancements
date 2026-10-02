# KEP-6281: Surface next scheduled time in CronJob status

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API changes](#api-changes)
  - [Controller changes](#controller-changes)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Scalability](#scalability)
  - [Dependencies / Failure Modes](#dependencies--failure-modes)
- [Implementation History](#implementation-history)
- [Alternatives](#alternatives)
<!-- /toc -->

## Release Signoff Checklist

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements]
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed and approved

## Summary

CronJob objects describe a recurring schedule, but their status does not expose
**when the next Job will be created**. Today users must parse the cron
expression (and time zone) themselves to answer "when does this run next?".

This KEP adds a `nextScheduleTime` field to `CronJobStatus`, populated by the
CronJob controller with the next time the schedule will fire. It is gated behind
a new alpha feature gate `CronJobsNextScheduleTime`.

## Motivation

`kubectl get cronjob` shows `LAST SCHEDULE` but not `NEXT SCHEDULE`. Operators,
dashboards, and tooling routinely want the next run time — for alerting on missed
runs, for UIs, and for debugging suspended or misconfigured schedules. Every
consumer currently reimplements cron parsing (including the time-zone handling
the controller already does), which is error-prone and duplicative. The
controller already computes this value internally to decide when to requeue; this
KEP simply surfaces it.

This is a long-standing community request (kubernetes/kubernetes#78564, open since
2019, `help wanted`).

### Goals

- Expose the next time a CronJob is scheduled to create a Job, in its status.
- Reuse the controller's existing schedule computation (including time zone).
- Ship behind an alpha feature gate, off by default.

### Non-Goals

- Changing scheduling behavior in any way. This is a read-only status field.
- Backfilling the field for CronJobs while the feature is disabled.
- Exposing a full list of upcoming run times (only the immediate next one).

## Proposal

Add an optional `nextScheduleTime *metav1.Time` field to `CronJobStatus`. The
CronJob controller sets it, during its normal reconcile, to the next time the
schedule fires after "now" in the CronJob's time zone. It is cleared (nil) when
the CronJob is suspended or has no valid/parseable schedule.

### User Stories

- **Operator:** "I want `kubectl get cronjob -o wide` / a dashboard to show the
  next run so I can confirm a schedule change took effect."
- **Alerting:** "I want to alert if `now` passes `nextScheduleTime` without a new
  Job appearing, without reimplementing cron parsing."

### Risks and Mitigations

- **Extra status writes:** the controller writes status when `nextScheduleTime`
  changes. Mitigation: it is only updated when the computed value actually
  differs from the stored one, and the controller already writes status for
  `lastScheduleTime`/`lastSuccessfulTime` during the same reconcile.
- **Field present when feature later disabled:** handled by the standard
  drop-disabled-fields pattern — the field is dropped on write unless it was
  already set, so a downgrade does not thrash the value.

## Design Details

### API changes

```go
// CronJobStatus represents the current state of a cron job.
type CronJobStatus struct {
    // ... existing fields ...

    // nextScheduleTime is the next time the CronJob is scheduled to create a
    // Job, computed from the schedule and, if set, the time zone. It is not set
    // when the CronJob is suspended or does not have a valid future schedule.
    // +optional
    // +featureGate=CronJobsNextScheduleTime
    NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty" protobuf:"bytes,6,opt,name=nextScheduleTime"`
}
```

The field is added to `batch/v1` (and the internal type). Feature-gate handling
lives in the CronJob status registry strategy: when
`CronJobsNextScheduleTime` is disabled and the old object did not already have the
field set, it is dropped.

### Controller changes

In `syncCronJob`, after the schedule is successfully parsed, when the feature is
enabled the controller sets `status.nextScheduleTime = sched.Next(now)` and marks
status for update if the value changed. When the CronJob is suspended, the field
is cleared. This reuses the exact schedule/time-zone parsing the controller
already performs.

### Test Plan

- API round-trip / fuzz (automatic via the API testing harness).
- Registry strategy unit test: field is dropped when the gate is off and retained
  when it was already set / the gate is on.
- Controller unit tests: field is populated for an active CronJob, updated when it
  changes, and cleared on suspend; not populated when the gate is off.

### Graduation Criteria

**Alpha (this KEP):** feature implemented behind `CronJobsNextScheduleTime`,
off by default; unit tests as above.

**Beta:** enabled by default; e2e coverage; no scaling/perf regressions on
clusters with many CronJobs; `kubectl` wide output optionally shows the column.

**GA:** feedback addressed; field always populated; feature gate locked to true
then removed.

### Upgrade / Downgrade Strategy

Enabling the gate begins populating the field on the next reconcile. Disabling it
stops updates; existing values are retained (drop-unless-already-set), so no
thrashing. No data migration.

### Version Skew Strategy

The field is populated only by a kube-controller-manager with the gate enabled.
An apiserver without the field defined would drop it, but since the field is
added to the same release as the gate, a skewed apiserver simply would not serve
it. No node-component involvement.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

- **How can this feature be enabled / disabled?** Feature gate
  `CronJobsNextScheduleTime` on kube-apiserver and kube-controller-manager;
  alpha, default off.
- **Does enabling change default behavior?** Only adds a populated status field;
  no scheduling behavior changes.
- **Can it be disabled once enabled?** Yes; the field stops updating and is
  retained, not removed mid-flight.
- **Rollback:** disable the gate. No stored-object migration needed.

### Monitoring Requirements

- Operators can observe the field directly on CronJob objects. No new metrics
  required for alpha; a possible `cronjob_controller` status-update counter could
  be considered for beta.

### Scalability

- One additional `*metav1.Time` per CronJob status; status is written only when
  the value changes. Negligible impact; no new API calls beyond the existing
  status update path.

### Dependencies / Failure Modes

- No new dependencies. If the controller is down, the field simply goes stale,
  identically to `lastScheduleTime` today.

## Implementation History

- 2026-08-17: KEP drafted; alpha implementation (feature gate, API field,
  controller, strategy, tests) prepared.

## Alternatives

- **Annotation instead of a status field:** rejected — annotations are untyped,
  not part of the schema, and awkward for tooling/printers.
- **Client-side computation:** the status quo; duplicates cron/time-zone logic in
  every consumer and cannot account for controller-side schedule handling.
- **List of upcoming times:** larger surface with unclear demand; the immediate
  next time covers the primary use cases.
