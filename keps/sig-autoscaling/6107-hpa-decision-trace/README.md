# KEP-6107: Improve HPA status explainability for operational tooling

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Summarize current HPA state in a dashboard](#story-1-summarize-current-hpa-state-in-a-dashboard)
    - [Story 2: Explain why scaling is limited](#story-2-explain-why-scaling-is-limited)
    - [Story 3: Understand multi-metric HPA outcomes](#story-3-understand-multi-metric-hpa-outcomes)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
  - [Open Questions](#open-questions)
  - [Related Work](#related-work)
- [Design Details](#design-details)
  - [Existing HPA Signals](#existing-hpa-signals)
    - [Current HPA Condition Reference](#current-hpa-condition-reference)
  - [Gap Analysis](#gap-analysis)
  - [Possible Improvements](#possible-improvements)
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
    [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806)
    must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md)
    within one minor version of promotion to GA
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation, such as additional design documents, links to
  mailing list discussions or SIG meetings, relevant PRs/issues, and release
  notes

[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes.io]: https://kubernetes.io/

## Summary

HorizontalPodAutoscaler exposes current and desired replica counts, current
metric values, and conditions such as `AbleToScale`, `ScalingActive`, and
`ScalingLimited`. Events and controller logs provide additional context for
specific reconciliations.

This KEP evaluates whether those existing signals provide enough stable and
actionable information for operational tooling to explain the current HPA state.
The goal is not to expose the HPA controller's internal decision process. The
goal is to identify which troubleshooting workflows are already covered by
existing HPA status, conditions, Events, and logs, and whether any remaining
gaps require documentation improvements, wording improvements, or a narrowly
scoped API change.

The initial direction is intentionally conservative. Several cases that appear
to require a new decision trace are already represented by existing conditions,
for example `ScalingLimited` with `TooFewReplicas` or `TooManyReplicas`,
`ScalingActive=False` with metric-fetch failure reasons, and stabilization
related condition reasons. This KEP therefore starts with a gap analysis before
proposing any new API surface.

## Motivation

Platform teams and SRE teams often need to explain HPA behavior from dashboards,
alert enrichment tools, or `kubectl`-style workflows. These tools frequently
need a concise user-facing interpretation of the current HPA state, such as why
scaling is limited or whether metric collection is failing.

Today, this information is distributed across HPA status fields, HPA conditions,
Events, controller logs, and user knowledge of HPA behavior. Some cases are
already well represented in stable object state. Other cases may require
wording or documentation improvements. A smaller set of cases may reveal actual
API gaps, especially for multi-metric HPAs where users want to understand which
metric, if any, effectively determined the observed `desiredReplicas`.

This KEP is motivated by improving HPA explainability for operational tooling
while avoiding unnecessary API surface and avoiding commitments to controller
implementation details.

### Goals

- Document how existing HPA status fields, conditions, Events, and controller
  logs map to common troubleshooting workflows.
- Identify which workflows are already supported by stable HPA object state.
- Identify gaps that can be addressed by documentation or clearer condition
  wording.
- Identify any remaining narrow API gaps, with particular attention to
  multi-metric HPAs and user-facing interpretation of `desiredReplicas`.
- Preserve the existing HPA scaling algorithm and user-facing scaling behavior.
- Avoid exposing internal controller decision logic as API.

### Non-Goals

- Changing how HPA computes desired replicas.
- Adding a broad machine-readable trace of every HPA decision step.
- Exposing per-metric internal replica recommendations unless SIG Autoscaling
  determines that a narrowly scoped user-facing API is necessary.
- Persisting historical decision traces.
- Providing a general audit log for autoscaling decisions.
- Replacing HPA events, conditions, controller logs, or metrics.
- Defining a new autoscaling recommendation API outside HPA.

## Proposal

This KEP proposes to first treat HPA explainability as a gap-analysis problem,
not as a new status-field design. The KEP will compare the identified
troubleshooting workflows against existing HPA surfaces:

- `status.currentReplicas`, `status.desiredReplicas`, `status.currentMetrics`,
  and `status.observedGeneration`;
- conditions such as `AbleToScale`, `ScalingActive`, and `ScalingLimited`;
- Events emitted by the HPA controller; and
- kube-controller-manager logs.

For each workflow, the KEP classifies the result as one of:

- already covered by existing stable object state;
- covered, but needing better documentation or clearer wording;
- covered only by Events or logs and therefore not ideal for current-state
  tooling; or
- not covered by a stable user-facing surface and potentially requiring a
  narrowly scoped API improvement.

Any API addition proposed after this analysis should be limited to
user-observable state and should avoid exposing the ordered sequence of
controller helper calls, intermediate calculations, or implementation-specific
local variables.

The Alpha outcome of this KEP is expected to be one of:

1. documentation-only guidance for interpreting existing HPA status and
   conditions;
2. condition reason or message clarification without adding new API fields; or
3. a follow-up narrowly scoped API proposal if SIG Autoscaling agrees that a
   real gap remains.

Before this KEP can move to `implementable`, the selected outcome must be
identified explicitly.

### User Stories

#### Story 1: Summarize current HPA state in a dashboard

A platform team builds a dashboard for application teams in a multi-tenant
cluster. Application teams can read their own HPA objects, but they do not have
access to kube-controller-manager logs. The dashboard should summarize whether
the HPA is active, limited, unable to fetch metrics, or affected by other
well-known current-state conditions.

This KEP evaluates whether the dashboard can derive that summary from existing
HPA status and conditions without parsing logs or relying on ephemeral Events.
Before presenting a definitive summary, the dashboard should compare
`status.observedGeneration` with `metadata.generation` and treat status as stale
when the generations differ.

#### Story 2: Explain why scaling is limited

An operator sees that an HPA's `desiredReplicas` is different from what they
expected. They want to know whether the value is limited by `minReplicas`,
`maxReplicas`, metric availability, or stabilization.

Existing conditions already expose several of these cases. For example,
`ScalingLimited=True` with `TooFewReplicas` or `TooManyReplicas` describes
replica bounds, and `ScalingActive=False` with metric-fetch related reasons
describes metric collection failures. This KEP documents those mappings and
identifies whether wording or documentation improvements are enough.

#### Story 3: Understand multi-metric HPA outcomes

An operator configures an HPA with several metrics. The HPA reports a
`desiredReplicas` value, but the operator wants to know which metric, if any,
effectively determined the observed outcome.

When multiple metrics are configured, the HPA selects the maximum recommendation
computed from the valid metrics before applying later constraints. Existing
`status.currentMetrics` entries show observed metric values, but they do not
directly identify which metric produced the effective recommendation that led to
the reported `desiredReplicas`.

The controller internally determines the metric associated with the largest
replica recommendation and may include that information in a condition message
or Event. However, that information is not exposed as a structured field.
Relying on free-form condition messages is not ideal for operational tooling.
Tooling also cannot reliably recompute the exact per-metric recommendations
from `status.currentMetrics` alone, because the current metric values reported
in status do not necessarily expose the adjusted values used internally after
missing-metric and pod-readiness handling.

This may be the narrowest remaining gap after accounting for existing
conditions, Events, and status fields. This KEP treats it as an open question
for SIG Autoscaling rather than assuming that a new decision-trace field is
required.

### Notes/Constraints/Caveats

Conditions are part of the HPA object state and are better suited than logs for
current-state tooling. Events are useful for human troubleshooting and recent
history, but they are not a durable current-state API. Controller logs are
implementation-specific and often not available to namespace-scoped users.

The KEP should not require a new API field for cases that are already described
by existing conditions or status fields. Any proposed improvement should be
limited to information that is stable, user-facing, and meaningful outside the
controller implementation.

### Risks and Mitigations

- Risk: The KEP proposes unnecessary API surface for cases already covered by
  conditions.
  - Mitigation: Start with an explicit mapping from workflows to existing HPA
    signals and prefer documentation or wording improvements where sufficient.
- Risk: A new API accidentally commits Kubernetes to HPA implementation details.
  - Mitigation: Treat any future API addition as a last resort and limit it to
    coarse, user-observable state.
- Risk: Tooling relies on Events or log messages as if they were stable APIs.
  - Mitigation: Clearly distinguish stable object state from diagnostic history
    and implementation-specific logs.
- Risk: Multi-metric explainability remains ambiguous.
  - Mitigation: Keep the multi-metric case as a focused open question and gather
    SIG Autoscaling feedback before proposing a field shape.

### Open Questions

- Can dashboards and alert enrichment tools reliably derive a concise
  user-facing interpretation of the current HPA state from existing conditions?
- Are current condition reasons and messages documented clearly enough for
  users and tool authors?
- Do multi-metric HPAs expose enough stable information for users and tooling to
  understand which metric, if any, effectively determined `desiredReplicas`?
- If a gap remains, is it an API gap or primarily a documentation / wording gap?
- Would narrowly scoped new condition reasons or message improvements be
  preferable to adding a new status field?

### Related Work

- [kubernetes/kubernetes#138992](https://github.com/kubernetes/kubernetes/issues/138992)
  discussed a structured HPA scaling decision status. This KEP supersedes the
  broader direction from that issue by first evaluating whether existing HPA
  status fields, conditions, Events, and logs already cover the identified
  troubleshooting workflows.

## Design Details

### Existing HPA Signals

The following existing surfaces are relevant to HPA explainability:

- `status.currentReplicas` and `status.desiredReplicas` show the observed and
  selected replica counts.
- `status.observedGeneration` identifies the HPA spec generation observed by
  the controller when computing the current status. Tooling should compare it
  with `metadata.generation` before presenting status-derived explanations as
  current.
- `status.currentMetrics` shows current metric values for configured metrics
  when they are available.
- `status.conditions` provides current-state condition types, statuses, reasons,
  and messages.
- Events provide recent diagnostic history such as failed metric fetches and
  successful rescale messages.
- Controller logs provide implementation-specific details for cluster operators
  with access to kube-controller-manager logs.

#### Current HPA Condition Reference

The main HPA condition types used by this KEP are:

- `AbleToScale`, which describes whether the controller can access and update
  the scale target. Example reasons include `SucceededGetScale`,
  `FailedGetScale`, `SucceededRescale`, `FailedUpdateScale`,
  `ReadyForNewScale`, `ScaleUpStabilized`, and `ScaleDownStabilized`.
- `ScalingActive`, which describes whether the HPA can compute replica counts
  from its metric inputs. Example reasons include `ValidMetricFound`,
  `ScalingDisabled`, `InvalidSelector`, `FailedGetResourceMetric`,
  `FailedGetContainerResourceMetric`, `FailedGetExternalMetric`,
  `FailedGetObjectMetric`, and `FailedGetPodsMetric`.
- `ScalingLimited`, which describes whether the computed desired replica count
  was limited by HPA constraints. Example reasons include `TooFewReplicas`,
  `TooManyReplicas`, `ScaleUpLimit`, `ScaleDownLimit`, and
  `DesiredWithinRange`.

The exact condition messages are still important to review for user clarity,
but tools should prefer condition types and documented reasons over parsing
free-form messages. If this KEP recommends relying on particular reasons, those
reasons should be documented as part of the user-facing interpretation
contract.

### Gap Analysis

| Workflow | Existing signal | Initial assessment |
| --- | --- | --- |
| Determine whether status reflects the latest HPA spec | Compare `metadata.generation` and `status.observedGeneration` | Covered by existing status. Tooling should avoid presenting a definitive explanation when status is stale. |
| Determine whether the desired replica count is raised to `minReplicas` | `ScalingLimited=True` with reason `TooFewReplicas` | Covered by existing conditions. Documentation may need to make this easier to discover. |
| Determine whether scaling is capped by `maxReplicas` | `ScalingLimited=True` with reason `TooManyReplicas` | Covered by existing conditions. Documentation may need to make this easier to discover. |
| Determine whether scaling was limited by behavior rate policies | `ScalingLimited=True` with `ScaleUpLimit` or `ScaleDownLimit` | Covered by existing conditions, but documentation should make this easier to discover. |
| Determine that scaling is not currently limited | `ScalingLimited=False` with `DesiredWithinRange` | Covered by existing conditions. Useful as part of dashboard summarization. |
| Detect resource metric fetch failures | `ScalingActive=False` with reasons such as `FailedGetResourceMetric`; Events | Covered by existing conditions and Events. |
| Detect external metric fetch failures | `ScalingActive=False` with reasons such as `FailedGetExternalMetric`; Events | Covered by existing conditions and Events. |
| Identify stabilization effects | Stabilization-related condition reasons such as `ScaleUpStabilized` or `ScaleDownStabilized` | Likely covered, but wording and documentation should be reviewed. |
| Determine whether a condition message contains useful but non-contractual detail | Condition `message` | Available, but tools should avoid parsing messages unless the relevant reason or message contract is documented. |
| Summarize current HPA state for dashboards | Combination of `desiredReplicas`, `currentReplicas`, `currentMetrics`, and conditions | Likely possible, but the expected interpretation should be documented. |
| Recompute exact per-metric replica recommendations from `status.currentMetrics` | `status.currentMetrics` | Not reliably covered. Current metric values do not necessarily expose the adjusted values used internally after missing-metric and pod-readiness handling. |
| Identify the metric that effectively determined `desiredReplicas` for a multi-metric HPA | `status.currentMetrics`, Events such as `SuccessfulRescale`, and controller behavior | Potential remaining gap. `currentMetrics` exposes observed values, but does not directly identify the metric that produced the effective recommendation. Needs SIG Autoscaling feedback before proposing an API change. |

### Possible Improvements

The preferred improvement depends on the result of the gap analysis:

- If existing conditions are sufficient, improve documentation for condition
  types, reasons, and recommended tooling interpretation.
- If current condition messages are confusing, clarify wording while preserving
  compatibility expectations for condition types and reasons.
- If a narrow current-state gap remains, consider adding or refining
  user-facing condition reasons before adding a new status field.
  "Narrowly scoped" in this context means adding or refining condition reasons
  on existing condition types such as `AbleToScale`, `ScalingActive`, and
  `ScalingLimited` without changing the semantics of the condition types
  themselves.
- If the multi-metric case cannot be represented by conditions without
  ambiguity, discuss a minimal API shape with SIG Autoscaling. Such an API
  should expose only stable user-facing information, not the controller's full
  decision trace.

### Test Plan

<!--
**Note:** This section is provisional and must be completed before targeting a
release milestone.
-->

#### Prerequisite testing updates

No prerequisite testing updates are expected while the KEP remains in the
gap-analysis phase.

#### Unit tests

If the outcome is documentation-only, no Kubernetes code unit tests are
required. If condition wording, condition reasons, or HPA status behavior is
changed, existing HPA controller tests should be updated to cover the affected
condition transitions and messages.

#### Integration tests

If no API or controller behavior changes are proposed, no new integration tests
are required. If a narrowly scoped API or condition behavior change is proposed,
integration coverage should verify that the HPA status exposes the intended
current-state signal for representative metric and replica-limit cases.

#### e2e tests

No new e2e tests are expected during the gap-analysis phase. If a future Beta
change adds or changes API-visible behavior, e2e coverage should be considered
for representative HPA troubleshooting scenarios.

### Graduation Criteria

#### Alpha

- Existing HPA signals are mapped to the identified troubleshooting workflows.
- SIG Autoscaling has reviewed the mapping against existing conditions, Events,
  logs, and status fields.
- SIG Autoscaling has reviewed the gap classification and agreed on which
  workflows are considered covered versus potential gaps.
- Any remaining gap is classified as documentation, wording, condition semantics,
  or API surface.
- If an API change is still needed, the Alpha scope is limited to a narrow
  user-facing improvement.

#### Beta

- Feedback from Alpha users and SIG Autoscaling is incorporated.
- Documentation clearly describes how users and tools should interpret relevant
  HPA conditions and status fields.
- Any introduced API-visible behavior has test coverage and compatibility
  expectations documented.

#### GA

- The selected improvement has proven useful for troubleshooting and tooling.
- No unresolved scalability, compatibility, or status interpretation issues
  remain.
- Documentation covers common troubleshooting scenarios.

### Upgrade / Downgrade Strategy

If the final outcome is documentation-only, there is no upgrade or downgrade
impact.

If a future revision proposes an API-visible change, the upgrade and downgrade
strategy must be updated to describe the specific field, condition, or behavior
being added or changed.

### Version Skew Strategy

If the final outcome is documentation-only, version skew is not applicable.

If a future revision proposes an API-visible change, the version skew strategy
must be updated to describe behavior across kube-apiserver,
kube-controller-manager, and clients.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

This KEP is currently in a gap-analysis phase and does not propose enabling a
new feature gate. If the outcome is documentation-only, there is no feature
enablement or rollback behavior.

If a future revision proposes an API-visible change, this section must be
updated with the feature gate, component ownership, and rollback behavior.

### Rollout, Upgrade and Rollback Planning

No control-plane downtime, node reprovisioning, or workload restart is expected
for the gap-analysis or documentation-only outcomes.

The PRR approver is TBD while this KEP is provisional.

### Monitoring Requirements

Existing HPA controller metrics should continue to be used to monitor HPA
behavior. This KEP does not currently require a new SLI.

If a future API-visible change is proposed, reviewers should verify that it does
not increase HPA status update frequency for unchanged HPA state.

### Dependencies

This KEP depends on the HPA controller and the `autoscaling/v2` HPA API. There
are no new external service dependencies.

### Scalability

The gap-analysis and documentation-only outcomes do not increase HPA object
size, status update frequency, or controller work.

If a future revision proposes a new status field or condition behavior, this
section must include analysis of object size, status update frequency, and any
additional controller work.

### Troubleshooting

Users should continue to inspect HPA status and conditions with
`kubectl get hpa -o yaml` or API clients. This KEP aims to document how to
interpret those existing signals and to identify whether any narrow gap remains.

## Implementation History

- 2026-05-23: Initial provisional KEP draft.
- 2026-05-25: Pivoted from a broad decision-trace API proposal to an HPA
  explainability gap analysis focused on existing signals.

## Drawbacks

Starting with gap analysis may delay a concrete API proposal. However, it
reduces the risk of adding unnecessary API surface for information already
available through existing HPA conditions, status fields, Events, or logs.

If the remaining multi-metric explainability gap is real, a documentation-only
outcome may not be enough for tooling. The KEP keeps that case open for focused
SIG Autoscaling discussion.

## Alternatives

- **Add `status.lastScaleDecision`.** The initial draft proposed a bounded
  machine-readable decision trace with per-metric entries, final reasons, and
  constraints. Feedback indicated that much of the desired information is
  already surfaced through HPA conditions and Events, and that exposing decision
  logic risks expanding the API surface unnecessarily. This alternative is
  deferred unless the gap analysis identifies a narrow need that cannot be
  addressed through existing status or conditions.
- **Use events only.** Events are useful for humans and recent history, but they
  are not a durable current-state API for automation.
- **Use conditions only.** Conditions already cover several important workflows.
  This may be sufficient if documentation and wording are improved. The
  multi-metric case needs further evaluation.
- **Use controller logs only.** Logs are implementation-specific and difficult
  for namespace-scoped users and dashboards to consume consistently.
- **Expose a separate subresource.** A subresource could provide more detail,
  but it is a larger API addition and is not justified unless a concrete gap is
  identified.
- **Expose historical traces.** History would help post-incident analysis, but
  it increases storage and API complexity and is outside the scope of this KEP.

## Infrastructure Needed

No new infrastructure is required.
