# KEP-6107: Machine-readable HPA scaling decision trace

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1: Explain a scaling decision with multiple metrics](#story-1-explain-a-scaling-decision-with-multiple-metrics)
    - [Story 2: Detect why a workload did not scale down](#story-2-detect-why-a-workload-did-not-scale-down)
    - [Story 3: Build an HPA troubleshooting view](#story-3-build-an-hpa-troubleshooting-view)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API](#api)
  - [Decision Model](#decision-model)
  - [Feature Gate](#feature-gate)
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

HorizontalPodAutoscaler exposes the current and desired replica counts, the
current metric values, and high-level conditions. These fields are useful, but
they do not provide a machine-readable explanation of how the controller chose
the final `desiredReplicas` value in the most recent reconciliation.

This KEP proposes adding a bounded, machine-readable trace of the latest HPA
scaling decision. The trace is intended to answer questions such as which metric
proposed the highest replica count, which metrics were invalid or ignored, and
whether the final recommendation was changed by min/max replica limits,
stabilization, tolerance, or scaling behavior policies.

The initial scope is intentionally limited to the most recent reconciliation
decision. The proposal does not change the HPA scaling algorithm.

## Motivation

HPA users often configure multiple metrics. During an incident, they need to
understand why the HPA selected a particular desired replica count, or why it did
not scale when they expected it to. Today, operators often need to combine HPA
status, events, controller logs, and knowledge of the controller implementation
to reconstruct the decision.

That reconstruction is hard to automate. Events are not a durable API for the
current decision, logs are implementation-specific, and existing status fields
do not show per-metric recommendations or later constraints applied by the HPA
controller. This makes user interfaces, automation, and support tooling rely on
heuristics.

Providing the latest decision in status gives users and tools a consistent
source of truth for explaining HPA behavior while keeping the controller's
existing scaling semantics intact.

### Goals

- Expose a bounded, machine-readable description of the latest HPA scaling
  decision.
- Show the per-metric replica recommendation, where one was computed.
- Show invalid metric reasons in a way that can be consumed by tools.
- Show whether the final desired replica count was affected by min/max replica
  limits, stabilization, tolerance, invalid metrics, or scaling behavior
  policies.
- Preserve the existing HPA scaling algorithm and user-facing scaling behavior.
- Keep the Alpha API small enough to evolve based on SIG Autoscaling feedback.

### Non-Goals

- Changing how HPA computes desired replicas.
- Persisting historical decision traces.
- Providing a general audit log for autoscaling decisions.
- Replacing HPA events, conditions, controller logs, or metrics.
- Exposing every internal intermediate value used by the controller.
- Defining a new autoscaling recommendation API outside HPA.

## Proposal

Add a new optional `status.lastScaleDecision` field to
`autoscaling/v2.HorizontalPodAutoscaler`. The field records the latest
meaningful scaling decision observed by the HPA controller.

The field is written only by the HPA controller and is guarded by the
`HPAStatusDecisionTrace` feature gate. When the feature gate is disabled, the
controller does not populate the field. The apiserver clears the field from
incoming create, update, and status update requests before storage, subject to
the implementation pattern used for gated alpha fields on GA APIs.

The field is a snapshot of one decision. It is replaced on successful status
updates by the controller when the semantic decision changes. The controller
must not refresh the field solely because another reconciliation occurred with
the same decision. It is not intended to be a complete history.

### User Stories

#### Story 1: Explain a scaling decision with multiple metrics

An operator configures an HPA with CPU, requests per second, and queue depth
metrics. The HPA scales to 12 replicas. The operator wants to know which metric
drove the decision without reading controller logs.

With this enhancement, the operator can inspect `status.lastScaleDecision` and
see the proposed replica count from each metric, the selected recommendation,
and the final desired replica count.

#### Story 2: Detect why a workload did not scale down

An HPA appears ready to scale from 20 replicas to 8 replicas, but it remains at
20. The operator wants to know whether scale down was blocked by stabilization,
tolerance, an invalid metric, or a behavior policy.

With this enhancement, the decision trace reports the initial recommendation and
the coarse constraint that kept the final desired replica count at 20.

#### Story 3: Build an HPA troubleshooting view

A platform team builds an internal dashboard for application teams in a
multi-tenant cluster. Application teams can read their HPA objects, but they do
not have access to kube-controller-manager logs. The dashboard needs a stable,
machine-readable API that can explain current HPA behavior across namespaces.

With this enhancement, the dashboard can read HPA status and present the current
decision without scraping events, parsing controller logs, or requiring
cluster-wide log access.

### Notes/Constraints/Caveats

The API must avoid exposing controller internals that would be difficult to
evolve. The trace should focus on user-observable reasons and stable decision
categories, not on every local variable in the implementation.

The decision trace can be stale if the HPA controller is not reconciling, cannot
update status, or has not observed the latest HPA spec generation. Consumers
must compare the decision timestamp and `observedGeneration` with the HPA
object.

### Risks and Mitigations

- Risk: The API becomes a commitment to internal implementation details.
  - Mitigation: Use coarse, user-facing reason enums and avoid recording every
    intermediate calculation.
- Risk: Status size grows too much for HPAs with many metrics.
  - Mitigation: Store only the latest decision, bound the per-metric entries to
    configured metrics, and use coarse enum values instead of verbose
    per-decision messages.
- Risk: Users treat the trace as a historical audit log.
  - Mitigation: Document that only the latest meaningful decision is retained and
    recommend events, logs, or external observability systems for history.
- Risk: Additional status writes increase apiserver load.
  - Mitigation: Update the trace as part of the existing HPA status update path
    and avoid extra writes solely for the trace.
- Risk: Consumers misinterpret the metric index after a spec change.
  - Mitigation: Record `observedGeneration` with the decision and document that
    `metricIndex` is valid only for the HPA spec generation observed by the
    controller.

## Design Details

### API

Add a new optional field to `HorizontalPodAutoscalerStatus`:

```go
type HorizontalPodAutoscalerStatus struct {
    // existing fields omitted

    // lastScaleDecision describes the most recent meaningful scaling decision
    // computed by the HPA controller.
    // +optional
    LastScaleDecision *HorizontalPodAutoscalerScaleDecision `json:"lastScaleDecision,omitempty" protobuf:"bytes,8,opt,name=lastScaleDecision"`
}
```

Add the following new types:

```go
type HorizontalPodAutoscalerScaleDecision struct {
    // Time is the time when the controller computed this decision.
    Time metav1.Time `json:"time" protobuf:"bytes,1,opt,name=time"`

    // ObservedGeneration is the HPA generation observed for this decision.
    ObservedGeneration int64 `json:"observedGeneration" protobuf:"varint,2,opt,name=observedGeneration"`

    // CurrentReplicas is the observed replica count used for the decision.
    CurrentReplicas int32 `json:"currentReplicas" protobuf:"varint,3,opt,name=currentReplicas"`

    // RecommendedReplicas is the replica count recommended before final HPA
    // constraints such as min/max replicas, stabilization, tolerance, and
    // behavior policies are applied, when such a recommendation could be
    // computed. It is absent when all metrics are invalid or unavailable and
    // the controller cannot compute an initial recommendation.
    // +optional
    RecommendedReplicas *int32 `json:"recommendedReplicas,omitempty" protobuf:"varint,4,opt,name=recommendedReplicas"`

    // DesiredReplicas is the final desired replica count selected by the HPA.
    DesiredReplicas int32 `json:"desiredReplicas" protobuf:"varint,5,opt,name=desiredReplicas"`

    // Direction describes whether the final decision scales up, scales down, or
    // keeps the current replica count.
    Direction HPAScaleDecisionDirection `json:"direction" protobuf:"bytes,6,opt,name=direction,casttype=HPAScaleDecisionDirection"`

    // Reason is a stable, coarse reason for the final decision.
    Reason HPAScaleDecisionReason `json:"reason" protobuf:"bytes,7,opt,name=reason,casttype=HPAScaleDecisionReason"`

    // Metrics contains one entry per configured metric that was evaluated.
    // +listType=atomic
    // +optional
    Metrics []HPAMetricScaleDecision `json:"metrics,omitempty" protobuf:"bytes,8,rep,name=metrics"`

    // Constraints describes coarse HPA constraints that affected the final
    // desired replica count after the initial recommendation was computed.
    // +listType=atomic
    // +optional
    Constraints []HPAScaleDecisionConstraint `json:"constraints,omitempty" protobuf:"bytes,9,rep,name=constraints"`
}

type HPAScaleDecisionDirection string

const (
    HPAScaleDecisionDirectionUp   HPAScaleDecisionDirection = "Up"
    HPAScaleDecisionDirectionDown HPAScaleDecisionDirection = "Down"
    HPAScaleDecisionDirectionNone HPAScaleDecisionDirection = "None"
)

type HPAScaleDecisionReason string

const (
    HPAScaleDecisionReasonMetricRecommendation HPAScaleDecisionReason = "MetricRecommendation"
    HPAScaleDecisionReasonWithinTolerance      HPAScaleDecisionReason = "WithinTolerance"
    HPAScaleDecisionReasonStabilized           HPAScaleDecisionReason = "Stabilized"
    HPAScaleDecisionReasonPolicyLimited        HPAScaleDecisionReason = "PolicyLimited"
    HPAScaleDecisionReasonReplicaLimit         HPAScaleDecisionReason = "ReplicaLimit"
    HPAScaleDecisionReasonInvalidMetrics       HPAScaleDecisionReason = "InvalidMetrics"
    HPAScaleDecisionReasonRecommendationFailed HPAScaleDecisionReason = "RecommendationFailed"
)

type HPAMetricScaleDecision struct {
    // MetricIndex is the index of the corresponding entry in spec.metrics for
    // the HPA generation recorded in observedGeneration. This avoids redefining
    // MetricSpec in the trace and works for all HPA metric source types,
    // including resource metrics such as CPU that are not represented by
    // MetricIdentifier.
    MetricIndex int32 `json:"metricIndex" protobuf:"varint,1,opt,name=metricIndex"`

    // ProposedReplicas is the desired replica count computed from this metric,
    // when a recommendation could be computed.
    // +optional
    ProposedReplicas *int32 `json:"proposedReplicas,omitempty" protobuf:"varint,2,opt,name=proposedReplicas"`

    // Status describes whether the metric produced a recommendation.
    Status HPAMetricScaleDecisionStatus `json:"status" protobuf:"bytes,3,opt,name=status,casttype=HPAMetricScaleDecisionStatus"`

    // Reason is a stable, coarse reason for this metric's status.
    // +optional
    Reason HPAMetricScaleDecisionReason `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason,casttype=HPAMetricScaleDecisionReason"`
}

type HPAMetricScaleDecisionStatus string

const (
    HPAMetricScaleDecisionStatusUsed    HPAMetricScaleDecisionStatus = "Used"
    HPAMetricScaleDecisionStatusValid   HPAMetricScaleDecisionStatus = "Valid"
    HPAMetricScaleDecisionStatusInvalid HPAMetricScaleDecisionStatus = "Invalid"
    HPAMetricScaleDecisionStatusIgnored HPAMetricScaleDecisionStatus = "Ignored"
)

type HPAMetricScaleDecisionReason string

const (
    HPAMetricScaleDecisionReasonComputed           HPAMetricScaleDecisionReason = "Computed"
    HPAMetricScaleDecisionReasonFetchFailed        HPAMetricScaleDecisionReason = "FetchFailed"
    HPAMetricScaleDecisionReasonInvalidMetricValue HPAMetricScaleDecisionReason = "InvalidMetricValue"
    HPAMetricScaleDecisionReasonNoMatchingPods     HPAMetricScaleDecisionReason = "NoMatchingPods"
    HPAMetricScaleDecisionReasonSkippedScaleDown   HPAMetricScaleDecisionReason = "SkippedScaleDown"
)

type HPAScaleDecisionConstraint string

const (
    HPAScaleDecisionConstraintMinReplicas        HPAScaleDecisionConstraint = "MinReplicas"
    HPAScaleDecisionConstraintMaxReplicas        HPAScaleDecisionConstraint = "MaxReplicas"
    HPAScaleDecisionConstraintStabilization      HPAScaleDecisionConstraint = "Stabilization"
    HPAScaleDecisionConstraintTolerance          HPAScaleDecisionConstraint = "Tolerance"
    HPAScaleDecisionConstraintScalingPolicy      HPAScaleDecisionConstraint = "ScalingPolicy"
    HPAScaleDecisionConstraintInvalidMetricGuard HPAScaleDecisionConstraint = "InvalidMetricGuard"
)
```

The exact field names and enum values are provisional and should be refined with
SIG Autoscaling before this KEP moves to `implementable`.

Reason fields use enum types. As with Kubernetes condition reasons, new enum
values may be added in later releases and clients must tolerate unknown values.
Existing enum values will not be repurposed.

### Decision Model

The controller records:

1. the current replica count observed for the scale target;
2. each configured metric evaluation, identified by its `spec.metrics` index,
   and its proposed replica count, if one was computed;
3. the initial recommendation selected from valid metrics;
4. any coarse HPA constraint that affected the final decision; and
5. the final desired replica count written to HPA status.

For a multi-metric HPA, the metric entry that determines the initial
recommendation has `status: Used`. Other valid metrics have `status: Valid`.
Metrics that cannot be converted into a recommendation have `status: Invalid`.
If no metric can be converted into a recommendation, `recommendedReplicas` is
absent, `desiredReplicas` remains the controller's final desired replica count
for the reconciliation, and `reason` is `RecommendationFailed` or a more
specific failure reason.

`metricIndex` is intentionally the join key for metric decisions within the HPA
generation recorded in `observedGeneration`. Clients can use it to look up the
full metric identity in `spec.metrics[metricIndex]` when the HPA object's
current `metadata.generation` matches `lastScaleDecision.observedGeneration`.
Clients must treat the metric index as stale if the HPA spec changed after the
recorded decision. This keeps the trace from embedding a second copy of the
MetricSpec union and avoids gaps for resource, pods, object, container
resource, and external metrics.

When invalid metrics prevent a scale down, the trace records an
`InvalidMetricGuard` constraint. When the HPA stays within tolerance, the trace
records a `Tolerance` constraint and the final direction is `None`.

`constraints` is an unordered set of coarse, user-observable constraint
categories that affected the final decision. It intentionally does not expose
the sequence of controller helper calls or intermediate replica counts.

The controller must not write status solely because `time` changed. Trace
updates piggyback on the existing HPA status update path. If the semantic
decision is unchanged, including current replicas, recommendation, desired
replicas, metric statuses, proposed replicas, reasons, and constraints, the
controller leaves `lastScaleDecision` unchanged rather than refreshing only the
timestamp.

### Feature Gate

The feature gate is named `HPAStatusDecisionTrace`.

The gate is configured for:

- `kube-apiserver`, to accept and persist the new status field; and
- `kube-controller-manager`, to populate the field.

When the apiserver feature gate is disabled:

- create and update requests clear `status.lastScaleDecision` before storage;
- status update requests clear `status.lastScaleDecision` before validation and
  storage;
- OpenAPI and generated clients follow the standard implementation pattern for
  gated alpha fields on existing APIs; and
- storage version and conversion remain unchanged because this is an optional
  status field on the existing `autoscaling/v2` API.

Disabling the gate therefore behaves like other gated alpha fields on GA APIs:
new writes cannot introduce or preserve the field and downgrade does not
require an explicit data migration.

### Test Plan

<!--
**Note:** This section is provisional and must be completed before targeting a
release milestone.
-->

#### Prerequisite testing updates

No prerequisite testing updates are expected.

#### Unit tests

- HPA controller tests for per-metric decision entries.
- HPA controller tests for min/max replica constraints.
- HPA controller tests for tolerance and stabilization constraints.
- HPA controller tests for invalid metrics preventing scale down.
- HPA controller tests that unchanged semantic decisions do not trigger status
  writes only to refresh the decision timestamp.
- API serialization, defaulting, and feature-gate tests for the new status
  field.

#### Integration tests

- Verify that the apiserver serves and persists `status.lastScaleDecision` only
  when the feature gate is enabled, and clears it on create, update, and status
  update when disabled.
- Verify status updates from the HPA controller populate the field for common
  resource and external metric cases.

#### e2e tests

Alpha does not require conformance e2e tests. Beta should add e2e coverage that
creates an HPA with multiple metrics and verifies that the latest decision trace
identifies the selected metric, final desired replica count, and any coarse
constraint that affected the decision.

### Graduation Criteria

#### Alpha

- API types and feature gate are added.
- HPA controller populates the latest decision trace.
- Unit tests cover major decision outcomes.
- Unit tests verify that trace population does not add status writes when only
  the decision timestamp would change.
- User documentation describes the Alpha field and its limitations.

#### Beta

- Feedback from Alpha users and SIG Autoscaling is incorporated.
- Field names and enum values are considered stable enough for Beta.
- e2e coverage exists for at least one multi-metric decision.
- Scalability testing or analysis demonstrates that the field size remains
  bounded by the configured metric count and that status update frequency does
  not increase for unchanged decisions.
- Metrics or logs exist to detect controller errors while populating the trace,
  if needed.

#### GA

- The API has proven useful for troubleshooting and tooling.
- No unresolved scalability, status write frequency, or compatibility issues
  remain.
- Documentation covers common troubleshooting scenarios.

### Upgrade / Downgrade Strategy

On upgrade, the field remains absent until the feature gate is enabled and the
HPA controller reconciles an HPA.

On downgrade or when disabling the feature gate, the apiserver clears the field
from subsequent create, update, and status update requests before storage. The
controller stops writing it. The HPA scaling algorithm and existing status
fields continue to behave as before.

### Version Skew Strategy

If the apiserver enables the feature gate but an older HPA controller is
running, the field is not populated. If the HPA controller enables the feature
gate but the apiserver does not, the apiserver clears the field before storage
and the status update otherwise follows normal validation and update behavior.
Cluster operators should keep the feature gate settings consistent across
`kube-apiserver` and `kube-controller-manager`.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

This feature is enabled by the `HPAStatusDecisionTrace` feature gate on
`kube-apiserver` and `kube-controller-manager`.

Rollback is supported by disabling the feature gate. Disabling the feature gate
removes the new observability surface but does not change HPA scaling behavior.

### Rollout, Upgrade and Rollback Planning

The feature can be rolled out by enabling the gate first on the apiserver and
then on the controller manager. Rollback can be performed in the reverse order.

### Monitoring Requirements

Existing HPA controller metrics should continue to be used to monitor HPA
behavior. This KEP does not require a new SLI. Additional controller metrics may
be considered if Alpha feedback shows that trace population can fail
independently from normal HPA reconciliation.

During Alpha, reviewers should verify that enabling the feature does not
increase HPA status update frequency when the computed decision is unchanged.

### Dependencies

This feature depends on the HPA controller and the `autoscaling/v2` HPA API.
There are no new external service dependencies.

### Scalability

The field stores only the latest decision and one entry per configured HPA
metric. It should scale with the existing HPA metric count. Implementations must
avoid adding extra status writes only for the trace. Before Beta, the KEP should
include either scalability test results or analysis showing the maximum added
status size for supported HPA metric counts.

### Troubleshooting

Users can inspect `status.lastScaleDecision` with `kubectl get hpa -o yaml` or
through API clients. If the field is missing, users should check whether the
feature gate is enabled, whether the HPA controller has reconciled the object,
and whether the decision timestamp and `observedGeneration` are current.

## Implementation History

- 2026-05-23: Initial provisional KEP draft.

## Drawbacks

Adding a new status field increases the public API surface and may make future
HPA implementation changes more constrained if the field exposes too much
detail. The KEP therefore uses coarse decision categories and intentionally does
not expose the ordered sequence of controller helper calls or intermediate
replica counts.

## Alternatives

- **Use events only.** Events are useful for humans and history, but they are
  not a durable, current-state API for automation.
- **Use conditions only.** Conditions can show high-level state, but they are
  not well suited for per-metric recommendations and coarse constraint
  categories.
- **Use controller logs only.** Logs are implementation-specific and difficult
  for cluster users to consume consistently.
- **Expose a separate subresource.** A subresource could provide more detail,
  but it is a larger API addition than needed for the latest meaningful
  decision.
- **Expose historical traces.** History would help post-incident analysis, but
  it increases storage and API complexity. This KEP starts with the latest
  decision only.

## Infrastructure Needed

No new infrastructure is required.
