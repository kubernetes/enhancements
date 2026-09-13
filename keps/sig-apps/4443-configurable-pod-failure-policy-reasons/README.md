# KEP-4443: More granular Job failure reasons for PodFailurePolicyRule

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [API changes](#api-changes)
  - [Defaulting](#defaulting)
  - [Validation](#validation)
  - [Business logic](#business-logic)
  - [Metrics](#metrics)
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

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in
  [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [X] (R) Test plan is in place, giving consideration to SIG Architecture and
  SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for
    [Conformance Tests]
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [X] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints] must be hit by
    [Conformance Tests] within one minor version of promotion to GA
- [X] (R) Production readiness review completed
- [X] (R) Production readiness review approved
- [X] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for
  publication to [kubernetes.io]
- [ ] Supporting documentation -- e.g. additional design documents, links to
  mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[all GA Endpoints]: https://github.com/kubernetes/community/pull/1806
[Conformance Tests]: https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding an optional `name` field to `PodFailurePolicyRule` in
the Job API. When a rule with `action: FailJob` matches a failed Pod, the Job
controller will set the Job `Failed` condition reason to
`PodFailurePolicy_<ruleName>`. If the field is unset, the controller derives the
suffix from the rule index, preserving a deterministic machine-readable reason
without requiring users to name every rule.

The purpose of naming rules is to expose more detailed failure information in
the Job status condition. This lets users configure multiple pod failure policy
rules and lets higher-level controllers distinguish which rule, if any, caused a
Job failure.

## Motivation

Higher-level Kubernetes APIs are often built by composing lower-level APIs. APIs
such as JobSet use Jobs as building blocks and need to distinguish different
types of Job failures so they can make informed decisions about how to react.

Today, the Job API does not propagate granular failure reason information, such
as container exit codes, in a way higher-level controllers can consume
programmatically. A `PodFailurePolicy` can cause a failed Job to receive a
generic `PodFailurePolicy` condition reason, but different rules targeting
different failure modes all use the same reason. This prevents controllers such
as JobSet from applying different actions for different failure causes.

For a concrete use case, see the JobSet
[Configurable Failure Policy KEP], which highlighted the need for more granular
pod failure policy reasons.

[Configurable Failure Policy KEP]: https://github.com/kubernetes-sigs/jobset/blob/main/keps/262-ConfigurableFailurePolicy/README.md

### Goals

- Allow pod failure policy rules to communicate distinct failure types to
  higher-level APIs and controllers.
- Provide a stable, machine-readable Job failure condition reason for the
  matching `FailJob` rule.
- Preserve the existing pod failure policy matching and action semantics.
- Keep observability bounded by avoiding user-provided rule names in metric
  label values.

### Non-Goals

- Changing `PodFailurePolicy` matching, ordering, or actions.
- Allowing users to set arbitrary Job condition reasons directly.
- Having the Job controller use the new field for any purpose other than the
  Job failure condition reason described in this KEP.
- Adding new Job APIs or new Job controller metrics.

## Proposal

Add an optional `name` field to `PodFailurePolicyRule`. When a
`PodFailurePolicyRule` matches a Pod failure and the rule action is `FailJob`,
the Job controller will append the rule name to the Job `Failed` condition
reason using the format `PodFailurePolicy_<ruleName>`.

If `name` is unset, the controller will use the zero-based index of the matching
rule in `podFailurePolicy.rules` as the suffix. For example, if the first rule
is unnamed and causes the Job to fail, the condition reason will be
`PodFailurePolicy_0`.

### User Stories

#### Story 1

As a user, I am using a JobSet to manage a group of Jobs, and I want to decide
whether to fail the JobSet based on the exact container exit code that caused a
child Job failure.

Example JobSet:

```yaml
apiVersion: jobset.x-k8s.io/v1alpha2
kind: JobSet
metadata:
  name: fail-jobset-example
spec:
  failurePolicy:
    rules:
    # If the Job fails due to a Pod failing with exit code 2, fail the JobSet
    # immediately without attempting restarts.
    - action: FailJobSet
      targetReplicatedJobs:
      - workers
      onJobFailureReasons:
      - PodFailurePolicy_ExitCode2
    maxRestarts: 10
  replicatedJobs:
  - name: workers
    replicas: 10
    template:
      spec:
        parallelism: 1
        completions: 1
        backoffLimit: 0
        podFailurePolicy:
          rules:
          - name: ExitCode2
            action: FailJob
            onExitCodes:
              containerName: main
              operator: In
              values: [2]
        template:
          spec:
            restartPolicy: Never
            containers:
            - name: main
              image: python:3.10
              command: ["..."]
```

#### Story 2

As a user, I am using a JobSet to manage a group of Jobs, each running an HPC
simulation. Each Job runs a simulation with different random initial parameters.
When a simulation ends, the application exits with one of two exit codes:

- Exit code 2 indicates the simulation produced an invalid result due to bad
  starting parameters and should not be retried.
- Exit code 3 indicates the simulation produced an invalid result, but the
  initial parameters were reasonable, so the simulation should be restarted.

When a Job fails due to exit code 2, I want my job management software to leave
the Job in a failed state. When a Job fails due to exit code 3, I want my job
management software to restart the Job.

Example JobSet:

```yaml
apiVersion: jobset.x-k8s.io/v1alpha2
kind: JobSet
metadata:
  name: restart-job-example
  annotations:
    alpha.jobset.sigs.k8s.io/exclusive-topology: {{topologyDomain}}
spec:
  failurePolicy:
    rules:
    # If the Job fails due to a Pod failing with exit code 2, leave it in a
    # failed state.
    - action: FailJob
      targetReplicatedJobs:
      - simulations
      onJobFailureReasons:
      - PodFailurePolicy_ExitCode2
    # If the Job fails due to a Pod failing with exit code 3, restart that Job.
    - action: RestartJob
      targetReplicatedJobs:
      - simulations
      onJobFailureReasons:
      - PodFailurePolicy_ExitCode3
    maxRestarts: 10
  replicatedJobs:
  - name: simulations
    replicas: 10
    template:
      spec:
        parallelism: 1
        completions: 1
        backoffLimit: 0
        podFailurePolicy:
          rules:
          - name: ExitCode2
            action: FailJob
            onExitCodes:
              containerName: main
              operator: In
              values: [2]
          - name: ExitCode3
            action: FailJob
            onExitCodes:
              containerName: main
              operator: In
              values: [3]
        template:
          spec:
            restartPolicy: Never
            containers:
            - name: main
              image: python:3.10
              command: ["..."]
```

### Notes/Constraints/Caveats

Pod failure policy rules are evaluated in order when a Pod fails. Only the first
matching rule is executed, even if multiple rules match the same Pod failure.
The emitted reason therefore reflects the first matching `FailJob` rule.

The new reason format is feature-gated. While the feature gate is disabled, Jobs
continue to use the existing `PodFailurePolicy` condition reason.

### Risks and Mitigations

The main compatibility risk is that enabling the feature changes the condition
reason observed by clients when a pod failure policy rule fails a Job. Clients
that read Job failure reasons will need to handle both the legacy
`PodFailurePolicy` reason and the new `PodFailurePolicy_<ruleName>` format
during rollout, rollback, and version skew. The feature gate limits this change
to clusters that opt in during alpha.

The main API risk is that a user-provided value becomes part of a
machine-readable status reason. Validation mitigates malformed or colliding
inputs by checking uniqueness, generated reason syntax, generated reason length,
and conflicts with Job controller reasons.

The main observability risk is metric cardinality. The Job controller's
`jobs_finished_total` metric has a `reason` label. Implementations of this KEP
must not use user-provided rule names as metric label values. Any granular
`PodFailurePolicy_<ruleName>` condition reason must be normalized back to the
bounded `PodFailurePolicy` label value before recording `jobs_finished_total`.

## Design Details

### API changes

Add an optional `name` field to `PodFailurePolicyRule`.

```go
type PodFailurePolicyRule struct {
    // Name is an optional name for this rule. When the rule matches a Pod
    // failure and the action is FailJob, the Job controller appends this name to
    // the Job Failed condition reason as "PodFailurePolicy_<name>".
    Name string

    // Existing fields omitted.
}
```

### Defaulting

There is no API defaulting for `PodFailurePolicyRule.name`. If `name` is unset,
the stored Job spec remains unchanged. At reconciliation time, the Job
controller derives the reason suffix from the zero-based index of the matching
rule in `podFailurePolicy.rules`.

### Validation

- Validate that all non-empty pod failure policy rule names are unique within a
  Job.
- Validate that a rule name does not collide with an index-generated reason for
  another rule. For example, `name: "0"` is only allowed on rule index 0.
- Validate that the generated Job condition reason
  `PodFailurePolicy_<ruleName>` is a valid Kubernetes condition reason, including
  syntax and the 128-character maximum length.
- Validate that the generated Job condition reason does not conflict with the
  Job controller's internal failure reasons.

### Business logic

When a `PodFailurePolicyRule` matches a Pod failure and `action` is `FailJob`,
the Job controller will set the Job `Failed` condition reason deterministically:

- `PodFailurePolicy_<name>` when the rule has `name` set.
- `PodFailurePolicy_<index>` when the rule has `name` unset.

Rules with `Count` or `Ignore` actions do not set the Job `Failed` condition and
therefore do not use the new reason format.

If the `PodFailurePolicyName` feature gate is disabled, the API server will not
preserve the new field and the Job controller will continue to use the legacy
`PodFailurePolicy` reason. If `PodFailurePolicyName` is enabled but the
underlying `PodFailurePolicy` behavior is disabled, the new field has no effect
because the Job controller only evaluates it as part of pod failure policy
handling.

### Metrics

This KEP does not add new metrics.

The Job controller already records finished Jobs in `jobs_finished_total` with a
bounded `reason` label. To preserve that bounded cardinality, the implementation
must record granular pod failure policy reasons as `PodFailurePolicy` in this
metric instead of recording the full `PodFailurePolicy_<ruleName>` condition
reason.

The existing `pod_failures_handled_by_failure_policy_total` metric remains
unchanged and continues to report the matched action (`FailJob`, `Ignore`, or
`Count`).

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates are required.

##### Unit tests

Unit tests will cover:

- API validation for duplicate rule names, generated reason syntax, generated
  reason length, index collisions, and conflicts with internal Job failure
  reasons.
- API strategy behavior with the feature gate enabled and disabled.
- Controller handling for named and unnamed `FailJob` rules matching
  `onExitCodes`.
- Controller handling for named and unnamed `FailJob` rules matching
  `onPodConditions`.
- Controller behavior when the feature gate is disabled after a Job already has
  a granular failure reason.
- Metric recording that keeps `jobs_finished_total` reason labels bounded by
  using `PodFailurePolicy` rather than user-provided rule names.

Current coverage for packages expected to be touched:

- `k8s.io/kubernetes/pkg/controller/job`: `2024-05-02` - `91.5%`
- `k8s.io/kubernetes/pkg/apis/batch/v1`: `2024-05-06` - `87.3%`
- `k8s.io/kubernetes/pkg/apis/batch/v1beta1`: `2024-05-06` - `78.3%`
- `k8s.io/kubernetes/pkg/apis/batch/validation`: `2024-05-06` - `87.7%`

##### Integration tests

Integration tests will cover:

- When the feature gate is enabled and a Job's pod failure policy triggers a Job
  failure through a named rule, the Job `Failed` condition reason is
  `PodFailurePolicy_<name>`.
- When the feature gate is enabled and a Job's pod failure policy triggers a Job
  failure through an unnamed rule, the Job `Failed` condition reason is
  `PodFailurePolicy_<index>`.
- When the feature gate is disabled, the API server does not preserve the new
  field and Job failures continue to use the legacy `PodFailurePolicy` reason.
- When the feature gate is disabled after a Job already has a granular failure
  condition reason, the Job controller does not rewrite that existing condition
  reason.

##### e2e tests

An e2e test will create a Job with `podFailurePolicy.rules[*].name`, trigger a
Pod failure that matches the named `FailJob` rule, and verify that the Job
`Failed` condition has the expected `PodFailurePolicy_<name>` reason.

An additional e2e test will cover an unnamed rule and verify the
`PodFailurePolicy_<index>` fallback.

### Graduation Criteria

#### Alpha

- Feature implemented behind the `PodFailurePolicyName` feature gate, disabled
  by default.
- API validation, API strategy, Job controller unit tests, and Job controller
  integration tests implemented.
- `jobs_finished_total` continues to use bounded reason label values.
- User-facing documentation updated.

#### Beta

- Address feedback and bug reports from Alpha users.
- Feature is stable in Alpha for one release cycle.
- Feature gate enabled by default.
- e2e tests are implemented, running regularly, and linked in this KEP.
- Upgrade, downgrade, and rollback behavior is tested.
- All functional, security, monitoring, and testing gaps identified during Alpha
  are resolved.

#### GA

- Address feedback and bug reports from Beta users.
- Feature is stable in Beta for two full release cycles.
- Feature gate graduated according to the feature gate lifecycle.

### Upgrade / Downgrade Strategy

On upgrade to a Kubernetes version that supports this feature, no changes are
required for existing Jobs. Users can opt in to granular reasons by enabling the
`PodFailurePolicyName` feature gate and setting
`.spec.podFailurePolicy.rules[*].name` on new Jobs.

On downgrade, or when the feature gate is disabled:

- New Jobs cannot rely on `PodFailurePolicyRule.name`; older API servers or API
  servers with the gate disabled will not preserve the field.
- Existing Jobs that already have the field may be reconciled by a Job
  controller that ignores it, causing new pod-failure-policy-triggered Job
  failures to use the legacy `PodFailurePolicy` reason.
- Existing terminal Job conditions are not rewritten solely because the feature
  is disabled or the control plane is downgraded.

Controllers that consume Job failure reasons should tolerate both
`PodFailurePolicy` and `PodFailurePolicy_<ruleName>` during upgrade, downgrade,
and rollback.

### Version Skew Strategy

This feature is limited to the control plane. It does not require kubelet,
kube-proxy, CRI, CNI, or CSI changes.

In an HA control plane with skewed kube-apiserver versions or feature-gate
configuration, requests that include `PodFailurePolicyRule.name` may be accepted
by some API servers and pruned or rejected by others until rollout is complete.

If the kube-controller-manager leader is an older version or has
`PodFailurePolicyName` disabled, the built-in Job controller ignores the new
field and continues to set the legacy `PodFailurePolicy` condition reason. When
a new controller-manager with the gate enabled becomes leader, subsequent
pod-failure-policy-triggered Job failures use the granular reason format.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate
  - Feature gate name: `PodFailurePolicyName`
  - Components depending on the feature gate:
    - kube-apiserver
    - kube-controller-manager

The feature can be enabled by setting
`--feature-gates=PodFailurePolicyName=true` on kube-apiserver and
kube-controller-manager. It can be disabled by setting the feature gate to
`false` and restarting those components.

###### Does enabling the feature change any default behavior?

Yes. For Jobs whose pod failure policy has a matching `FailJob` rule, enabling
the feature changes the Job `Failed` condition reason from `PodFailurePolicy` to
`PodFailurePolicy_<name>` or `PodFailurePolicy_<index>`. Pod failure policy
matching, ordering, and actions are unchanged.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disable the `PodFailurePolicyName` feature gate on kube-apiserver and
kube-controller-manager. New writes will not preserve the new field, and the Job
controller will resume using the legacy `PodFailurePolicy` reason for subsequent
Job failures.

###### What happens if we reenable the feature if it was previously rolled back?

For new Jobs, the API server will preserve `PodFailurePolicyRule.name` again.
For existing Jobs that still contain the field, the Job controller will resume
using it for subsequent pod-failure-policy-triggered Job failures.

###### Are there any tests for feature enablement/disablement?

The implementation will add unit and integration tests for:

- Feature gate enabled with `PodFailurePolicyRule.name` set.
- Feature gate enabled with `PodFailurePolicyRule.name` unset.
- Feature gate disabled with the field set on incoming objects.
- Feature gate disabled after Jobs already have granular Job failure condition
  reasons.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Partial rollout can result in mixed behavior while different API servers or
controller-manager instances run with different versions or feature-gate values.
Some requests may preserve `PodFailurePolicyRule.name` while others prune or
reject it, and the active Job controller may emit either the legacy or granular
condition reason.

This does not stop already running Pods or Jobs. The impact is limited to Job
spec persistence for the new field and the failure reason observed when a pod
failure policy fails a Job.

###### What specific metrics should inform a rollback?

A substantial increase in `job_sync_duration_seconds` may indicate the Job
controller is spending more time reconciling Jobs after the feature is enabled.

Operators should also monitor `jobs_finished_total{reason="PodFailurePolicy"}`
and `pod_failures_handled_by_failure_policy_total{action="FailJob"}` to check
whether Jobs are failing through pod failure policy at the expected rate. The
`jobs_finished_total` metric must keep the bounded `PodFailurePolicy` reason
label even when Job conditions use granular reasons.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet. The feature implementation is pending. Before Beta, the
upgrade->downgrade->upgrade path will be tested to verify field preservation,
field pruning, legacy reason fallback, granular reason restoration, and
non-rewriting of existing terminal Job conditions.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No APIs, fields, feature gates, or flags are removed. The legacy unsuffixed
`PodFailurePolicy` Job condition reason remains the rollback and version-skew
behavior during Alpha and Beta.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

An operator can inspect Job specs for `.spec.podFailurePolicy.rules[*].name`.
For aggregate observation, a non-zero
`pod_failures_handled_by_failure_policy_total{action="FailJob"}` value indicates
that pod failure policy is failing Jobs, and `jobs_finished_total` indicates how
many Jobs finished with the bounded `PodFailurePolicy` reason label.

###### How can someone using this feature know that it is working for their instance?

- [X] Job API `.status`
  - Condition name: `Failed`
  - Other field: `reason` is set to `PodFailurePolicy_<name>` or
    `PodFailurePolicy_<index>`

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

This feature does not introduce a new SLO. Existing Job controller SLOs should
not be negatively affected.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [X] Metrics
  - Metric name:
    - `job_sync_duration_seconds` (existing): indicates Job controller sync
      latency.
    - `jobs_finished_total` (existing): indicates completed and failed Jobs with
      bounded reason labels.
    - `pod_failures_handled_by_failure_policy_total` (existing): indicates
      failed Pods handled by pod failure policy by action.
  - Components exposing the metric: kube-controller-manager

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No. The feature intentionally does not add a metric containing rule names because
rule names are user-provided and could create unbounded metric cardinality.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No. The feature only depends on kube-apiserver and kube-controller-manager.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No. The Job controller already updates Job status when a Job fails. This feature
only changes the condition reason used in that existing update.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, only for Jobs that set `PodFailurePolicyRule.name` or Jobs that later
receive a granular failure condition reason.

- API type: Job
- Estimated spec size increase: up to the validated length of each configured
  rule name.
- Estimated status size increase: one Job condition `reason` value grows from
  `PodFailurePolicy` to `PodFailurePolicy_<name>` or
  `PodFailurePolicy_<index>` when a pod failure policy fails the Job.
- Estimated object count increase: none.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No. Validation adds bounded checks over the existing
`podFailurePolicy.rules` slice, and reconciliation adds only bounded string
construction for a Job that is already failing.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No. The additional validation and string construction are negligible. Metrics
must not include user-provided rule names as labels, so the feature does not add
metric-cardinality-driven resource usage.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

If kube-apiserver or etcd is unavailable, users cannot create or update Jobs
with the new field and the Job controller cannot persist updated Job status.
Running Pods are not directly affected. The Job controller will retry status
updates when the API server and etcd become available.

###### What are other known failure modes?

- Invalid rule name
  - Detection: Job creation or update is rejected by kube-apiserver validation.
  - Mitigation: Change the rule name so the generated
    `PodFailurePolicy_<ruleName>` reason is valid, unique, within length limits,
    and non-conflicting.
  - Diagnostics: The API validation error identifies the invalid field.
  - Testing: Covered by API validation unit and integration tests.
- Feature gate disabled or old controller-manager leader
  - Detection: Jobs fail with the legacy `PodFailurePolicy` reason even though
    rule names are configured.
  - Mitigation: Confirm `PodFailurePolicyName=true` on kube-apiserver and
    kube-controller-manager, and complete the control-plane rollout.
  - Diagnostics: Inspect component feature-gate configuration and Job
    conditions.
  - Testing: Covered by feature-gate unit and integration tests.
- Skewed kube-apiserver rollout
  - Detection: Writes that include `PodFailurePolicyRule.name` behave
    differently depending on which API server handles the request.
  - Mitigation: Complete the kube-apiserver rollout or temporarily disable use of
    the new field until all API servers are consistent.
  - Diagnostics: Compare API server versions and feature-gate configuration.
  - Testing: Covered by upgrade and rollback testing before Beta.

###### What steps should be taken if SLOs are not being met to determine the problem?

Check `job_sync_duration_seconds` for increased Job controller sync latency,
then inspect whether the affected Jobs use pod failure policy and whether they
are failing through `FailJob` rules. Confirm that `jobs_finished_total` retains
bounded reason labels and does not contain user-provided rule names. If the
problem appears related to this feature, disable `PodFailurePolicyName` and
verify that Job sync latency returns to its previous level.

## Implementation History

- 2024-05-24: KEP published.
- 2026-06-09: Updated the target milestone to v1.38.
- 2026-09-06: Modernized KEP text, PRR answers, and metrics-cardinality
  requirements for the v1.38 Alpha target.

## Drawbacks

The prefixed `PodFailurePolicy_<ruleName>` format is less direct than allowing
users to set the entire Job condition reason themselves. The prefix is
intentional: it preserves Job controller ownership of the reason namespace while
still exposing the specific matching rule.

Clients that consume Job failure reasons need to handle both the legacy
`PodFailurePolicy` reason and granular reasons during rollout, rollback, and
version skew.

## Alternatives

One alternative was to add an optional `reason` or `setConditionReason` field to
`PodFailurePolicyRule` and use that value directly as the Job failure condition
reason. This was rejected because it would let users set arbitrary Job condition
reasons, making validation and reason ownership less clear.

Another alternative was to derive more specific reasons automatically, such as
`PodFailurePolicy_ExitCode143` for `onExitCodes` rules. This does not generalize
well to `onPodConditions`, and it gives users less control over the stable
machine-readable value consumed by higher-level controllers.

## Infrastructure Needed

No new infrastructure is needed.
