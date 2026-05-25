# KEP-6107: Improve HPA explainability for operational tooling

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Existing Signals and Gaps](#existing-signals-and-gaps)
  - [User Stories](#user-stories)
    - [Story 1: Summarize current HPA state in a dashboard](#story-1-summarize-current-hpa-state-in-a-dashboard)
    - [Story 2: Explain why scaling did not occur](#story-2-explain-why-scaling-did-not-occur)
    - [Story 3: Understand multi-metric HPA behavior](#story-3-understand-multi-metric-hpa-behavior)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Open Questions](#open-questions)
    - [Open Questions for Alpha](#open-questions-for-alpha)
    - [Open Questions for Beta / Later](#open-questions-for-beta--later)
- [Design Details](#design-details)
  - [Proposed Field](#proposed-field)
  - [Status Update Semantics](#status-update-semantics)
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
    [Conformance Tests]
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints]
    must be hit by [Conformance Tests]
    within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation, such as additional design documents, links to
  mailing list discussions or SIG meetings, relevant PRs/issues, and release
  notes

[Conformance Tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
[all GA Endpoints]: https://github.com/kubernetes/community/pull/1806
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes.io]: https://kubernetes.io/
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP improves the explainability of HorizontalPodAutoscaler (HPA) for
operational tooling by adding a new field `lastScaleDecision` to
`HorizontalPodAutoscalerStatus`.

Currently, it is difficult for dashboards, alert systems, and operators to
understand why HPA reported a particular `desiredReplicas` value, especially in
multi-metric configurations or when scaling is suppressed by tolerance or
stabilization. Existing conditions and Events provide partial information, but
they do not offer a single, structured, machine-readable summary of the latest
effective scaling outcome.

This KEP proposes a narrowly scoped `lastScaleDecision` field that summarizes
the effective outcome in a stable, user-facing way, without exposing the
controller's internal calculation details.

## Motivation

Platform teams and SRE teams often need to explain HPA behavior from
dashboards, alert enrichment tools, or `kubectl`-style workflows. These tools
frequently need a concise, machine-readable interpretation of why HPA scaled,
did not scale, or reported a particular `desiredReplicas` value.

Today, this information is distributed across HPA status fields, conditions,
Events, controller logs, and user knowledge of HPA behavior. Existing signals
are useful, and this KEP does not aim to replace them. However, several
important workflows remain difficult to support consistently from stable object
state alone.

This KEP improves HPA explainability by adding a bounded current-state summary
of the latest effective scaling outcome. The summary is intended for users and
operational tools, not as a serialized copy of the controller's internal
calculation process.

### Goals

- Add a `lastScaleDecision` field to `HorizontalPodAutoscalerStatus` that
  captures the most recent effective scaling outcome in a machine-readable
  format.
- Enable operational tooling to explain the current observed `desiredReplicas`
  value without parsing controller logs or free-form condition messages.
- Represent stable, user-facing concepts such as outcome, reason, selected
  metric, current replicas, and desired replicas.
- Keep the initial Alpha scope narrow, with Resource metrics as the first
  supported metric source.
- Avoid exposing internal controller decision logic or intermediate calculation
  state as Kubernetes API.
- Preserve backward compatibility and existing HPA scaling behavior.

### Non-Goals

- Exposing the full internal decision trace or every intermediate calculation
  step.
- Exposing raw tolerance thresholds, adjusted metric values, missing-metric pod
  counts, or not-ready pod accounting as stable API in the initial Alpha.
- Persisting historical scaling decisions.
- Changing how HPA computes desired replicas.
- Replacing existing conditions, Events, `status.currentMetrics`, or controller
  logs.
- Defining a new autoscaling recommendation API outside HPA.

## Proposal

This KEP proposes adding a new `lastScaleDecision` field to
`HorizontalPodAutoscalerStatus`. The HPA controller will populate this field
with a user-facing summary of the latest effective scaling outcome.

The field is designed to answer common operational questions such as:

- Did the HPA scale up, scale down, or keep the replica count unchanged?
- What `desiredReplicas` value does the latest effective outcome explain?
- Which stable, user-facing reason best explains the outcome?
- When the answer is unambiguous, which metric selected the effective
  recommendation?
- Was the outcome suppressed, limited, stabilized, or blocked by a metric error?

The field is not intended to expose the ordered sequence of controller helper
calls, exact intermediate per-pod accounting, or all internal inputs used during
replica calculation.

### Existing Signals and Gaps

Existing HPA status already exposes `currentReplicas`, `desiredReplicas`,
`currentMetrics`, `observedGeneration`, and `conditions`. These signals are
useful, and this proposal does not aim to replace them.

However, they do not provide a single structured explanation of the effective
scaling outcome. In particular, tooling cannot reliably determine:

- which metric effectively selected the final recommendation in a multi-metric
  HPA;
- whether the final outcome was suppressed by tolerance or stabilization;
- whether missing metrics, not-ready pods, or metric errors materially affected
  the user-facing outcome;
- whether the current `desiredReplicas` reflects a fresh calculation or a
  conservative fallback.

Events and controller logs can contain additional details, but they are not a
stable current-state API. Events are recent diagnostic history, and logs are
implementation-specific and often unavailable to namespace-scoped users.

### User Stories

#### Story 1: Summarize current HPA state in a dashboard

A platform team builds a dashboard that shows why an HPA currently reports a
particular `desiredReplicas` value. The dashboard reads `lastScaleDecision` to
display a short structured summary such as "scaled up because CPU utilization
selected a higher replica count" without parsing controller logs.

#### Story 2: Explain why scaling did not occur

An operator notices that `desiredReplicas` did not change despite increased
load. By inspecting `lastScaleDecision.outcome` and
`lastScaleDecision.reason`, the operator can tell whether the latest effective
outcome was a no-scale decision, a stabilized decision, a limited decision, or
a conservative fallback caused by metric errors.

#### Story 3: Understand multi-metric HPA behavior

An operator configures an HPA with multiple metrics. `lastScaleDecision`
identifies the selected metric when a single metric clearly drove the result.
In cases where a single metric cannot be clearly identified as the driver due
to errors, ties, or subsequent constraints, `selectedMetric` is left unset, and
`outcome` together with `reason` provide the explanation.

### Risks and Mitigations

- Risk: The new status field may expose too much of the controller's internal
  decision process.
  Mitigation: Limit the field to user-facing outcome summaries and avoid
  exposing intermediate calculations.
- Risk: Updating the field on every reconciliation may increase API server and
  etcd write load.
  Mitigation: Update the field only when the effective outcome or explanation
  materially changes.
- Risk: Tooling may start depending on details that are not intended to be
  stable.
  Mitigation: Clearly document which fields are stable API semantics and avoid
  adding descriptive fields that mirror internal local variables.
- Risk: Multi-metric outcomes may be more complex than a single selected metric.
  Mitigation: Treat `selectedMetric` as optional and require outcome/reason
  values to represent error-blocked, suppressed, limited, or conservative
  fallback cases.

### Open Questions

#### Open Questions for Alpha

- Should this field be protected by a feature gate?
- Should the first Alpha support only Resource metrics?
- What is the minimal set of `Outcome` and `Reason` values required for Alpha?
- Should this be a new status field, new condition reasons, or an improvement
  to existing Events?

#### Open Questions for Beta / Later

- Should missing metric and not-ready pod information be exposed directly,
  summarized, or omitted?
- What is the exact compatibility behavior for `autoscaling/v1` clients?
- Which outcome and reason values should be part of the stable API contract?
- Should multi-metric support include a bounded list of evaluated metric
  summaries, or only the selected metric where unambiguous?

## Design Details

### Proposed Field

The initial API shape is intentionally small. Field and enum names are
illustrative and require SIG Autoscaling review before this KEP is marked
implementable. Alpha should use the smallest useful set of outcome and reason
values. Before Beta, the supported `Outcome` and `Reason` values must be
documented as the stable API contract for this field.

```go
type HorizontalPodAutoscalerStatus struct {
    // Existing fields omitted.

    // lastScaleDecision summarizes the most recent effective scaling outcome
    // observed by the HPA controller.
    // +optional
    LastScaleDecision *LastScaleDecision `json:"lastScaleDecision,omitempty"`
}

type LastScaleDecision struct {
    // lastTransitionTime is updated when the effective outcome or the
    // information needed to explain the current desiredReplicas materially
    // changes. It is not a last-reconciled timestamp.
    LastTransitionTime metav1.Time `json:"lastTransitionTime"`

    CurrentReplicas int32 `json:"currentReplicas"`
    DesiredReplicas int32 `json:"desiredReplicas"`

    // outcome is a stable, user-facing category for the latest effective
    // scaling outcome.
    Outcome ScaleDecisionOutcome `json:"outcome"`

    // reason is a documented, stable, user-facing reason that explains the
    // outcome. The set of valid reasons will be defined as part of the stable
    // API contract before Beta.
    Reason string `json:"reason,omitempty"`

    // selectedMetric identifies the metric that selected the effective
    // recommendation when that is stable and unambiguous.
    // +optional
    SelectedMetric *ScaleDecisionMetric `json:"selectedMetric,omitempty"`
}

type ScaleDecisionOutcome string

// These Outcome values are provisional.
// For Alpha, implementations should start with a minimal useful set
// (recommended: ScaledUp, ScaledDown, NoScale, Limited, ErrorBlocked).
// Stabilized and Suppressed may be added later if needed.
// The final stable set of Outcome and Reason values must be documented
// before Beta graduation.
const (
    ScaleDecisionScaledUp      ScaleDecisionOutcome = "ScaledUp"
    ScaleDecisionScaledDown    ScaleDecisionOutcome = "ScaledDown"
    ScaleDecisionNoScale       ScaleDecisionOutcome = "NoScale"
    ScaleDecisionLimited       ScaleDecisionOutcome = "Limited"
    ScaleDecisionStabilized    ScaleDecisionOutcome = "Stabilized"
    ScaleDecisionSuppressed    ScaleDecisionOutcome = "Suppressed"
    ScaleDecisionErrorBlocked  ScaleDecisionOutcome = "ErrorBlocked"
)

type ScaleDecisionMetric struct {
    Type string `json:"type"`
    Name string `json:"name,omitempty"`
}
```

This KEP does not propose exposing raw per-metric recommendations,
ready-pod counts, missing-metric pod counts, tolerance thresholds, or adjusted
metric values in the initial Alpha. Those details are useful for debugging, but
they risk freezing implementation details into the Kubernetes API.

### Status Update Semantics

The controller SHOULD update `lastScaleDecision` only when the effective
scaling outcome changes, or when the information required to explain the
current `desiredReplicas` materially changes. Implementations should avoid
updating this field solely because reconciliation time has advanced.

`lastTransitionTime` has `lastTransitionTime` semantics: it records when the
reported outcome summary last changed. It is not a `lastEvaluatedTime` and must
not force a status update on every reconciliation.

### Test Plan

#### Prerequisite testing updates

No prerequisite test framework changes are expected. Existing HPA controller
unit tests should be extended where the controller populates status.

#### Unit tests

Unit tests should cover at least:

- scale up for Resource metrics;
- scale down for Resource metrics;
- no-scale due to tolerance;
- min replica limiting;
- max replica limiting;
- metric error cases that block or conservatively affect scaling;
- no status write when only reconciliation time advances and the effective
  outcome summary is unchanged.

#### Integration tests

Integration tests should verify that the HPA status field is persisted and
served correctly through the Kubernetes API when the feature is enabled.

#### e2e tests

No new e2e test is required for Alpha. Beta should consider e2e coverage for a
representative dashboard or CLI consumption workflow if SIG Autoscaling and SIG
Testing consider it valuable.

### Graduation Criteria

#### Alpha

- `lastScaleDecision` is added behind a feature gate.
- The HPA controller populates the field for Resource metrics.
- Unit tests cover scale up, scale down, no-scale due to tolerance, metric
  error handling, and min/max replica limiting.
- Unit tests verify that status is not updated solely because reconciliation
  time advanced.
- Documentation must explain how `lastScaleDecision` relates to existing HPA
  conditions such as `ScalingLimited` and `ScalingActive`,
  `status.currentMetrics`, and Events. It should also clarify when
  `selectedMetric` is expected to be set versus left unset.

#### Beta

- Multi-metric behavior is specified and tested.
- Upgrade and downgrade behavior is documented.
- Status update frequency is measured and shown not to introduce significant API
  server or etcd load.
- Feedback from at least one dashboard, CLI, or operational tooling use case is
  incorporated.
- The stable outcome and reason values are documented.

#### GA

- The field has demonstrated usefulness for operational tooling across common
  HPA configurations.
- No unresolved scalability, compatibility, or API semantics issues remain.
- Documentation covers common troubleshooting scenarios and version skew
  expectations.

### Upgrade / Downgrade Strategy

On upgrade, new kube-controller-manager versions may begin populating
`lastScaleDecision` when the feature gate is enabled and the served HPA API
version includes the field.

On downgrade or feature-gate disablement, older controllers will stop updating
the field. Clients must treat the field as optional. Tooling should fall back to
existing HPA status, conditions, Events, and logs when the field is absent or
stale.

The exact behavior for stored objects and clients using `autoscaling/v1` remains
an open question until the API shape is finalized.

### Version Skew Strategy

During version skew, kube-apiserver and kube-controller-manager versions may not
both understand or populate the new field. Clients must not assume that the
field is present. Tooling should also compare `metadata.generation` and
`status.observedGeneration` before presenting the summary as current.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

The field should be guarded by a feature gate for Alpha unless SIG Autoscaling
and SIG Architecture agree that a gate is unnecessary for this API addition.
Rollback consists of disabling the feature gate or downgrading the controller,
after which clients should rely on existing HPA signals.

### Rollout, Upgrade and Rollback Planning

No workload restart, node reprovisioning, or control-plane downtime is expected.
The change affects HPA status updates produced by kube-controller-manager.

### Monitoring Requirements

Existing API server and etcd metrics should be used to monitor status write
load. The implementation should also be evaluated with HPA controller metrics
to ensure that status updates do not increase unexpectedly.

### Dependencies

This KEP depends on the HPA controller and the `autoscaling/v2` HPA API. There
are no new external service dependencies.

### Scalability

The new field increases HPA object size slightly. The Alpha field shape is kept
small and bounded to avoid unbounded per-metric or per-pod data.

The implementation must avoid updating `lastScaleDecision` on every
reconciliation. Status writes should occur only when the effective outcome or
the explanation of the current `desiredReplicas` materially changes. Beta
graduation requires measurement showing that the additional status updates do
not introduce significant API server or etcd load. That measurement should
compare HPA status update frequency with and without this field enabled for
representative HPA workloads.

### Troubleshooting

Users should continue to inspect HPA status and conditions with
`kubectl get hpa -o yaml`. `lastScaleDecision` provides a concise summary, while
Events and logs remain useful for recent history and deeper debugging.

## Implementation History

- 2026-05-23: Initial provisional KEP draft.
- 2026-05-25: Pivoted to propose `lastScaleDecision` for HPA explainability.
- 2026-05-25: Narrowed `lastScaleDecision` to a user-facing outcome summary
  and added status update, scalability, and compatibility constraints.

## Drawbacks

Adding a new HPA status field increases API surface and object size. Even a
small field creates long-term compatibility expectations once clients depend on
it. After the field is added to the Kubernetes API, removing it or changing the
meaning of documented outcome and reason values would be difficult.

The field may still be less detailed than some debugging workflows want,
because it intentionally avoids exposing internal calculation details. Users
who need detailed historical diagnostics may still need Events, metrics, and
controller logs.

## Alternatives

- Documentation and condition wording improvements only. This is lower risk,
  but may be insufficient for multi-metric cases and stable tooling workflows.
- New or refined condition reasons. This may cover some user-facing outcomes,
  but conditions do not naturally carry selected metric identity or a compact
  outcome summary for tooling.
- Events only. Events are useful for human troubleshooting and recent history,
  but they are not suitable as a stable current-state API for tooling.
- Exposing a full decision trace. This is rejected as too broad and too likely
  to leak internal implementation details.
- Exposing historical decision records. This would help post-incident analysis,
  but it increases storage and API complexity and is outside this KEP's scope.

## Infrastructure Needed

No new infrastructure is required.
