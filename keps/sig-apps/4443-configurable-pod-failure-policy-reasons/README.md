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
controller will include the rule name in the Job failure condition message.
The `FailureTarget` and `Failed` condition reasons remain `PodFailurePolicy`.
The message continues to identify the matching rule by its zero-based index,
including when the name is unset.

The purpose of naming rules is to expose more detailed failure information in
the Job status condition. This lets users configure multiple pod failure policy
rules and identify which rule caused a Job failure when inspecting status.
The message is diagnostic text, not a structured programmatic interface.

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

The Alpha implementation addresses diagnostics first. Structured identification
for those controllers is deferred for reconsideration during graduation.

[Configurable Failure Policy KEP]: https://github.com/kubernetes-sigs/jobset/blob/main/keps/262-ConfigurableFailurePolicy/README.md

### Goals

- Allow users to identify the matching `FailJob` rule by name in Job failure
  condition messages.
- Preserve the stable, machine-readable `PodFailurePolicy` condition reason.
- Preserve the existing pod failure policy matching and action semantics.
- Keep observability bounded by avoiding user-provided rule names in metric
  label values.

### Non-Goals

- Changing `PodFailurePolicy` matching, ordering, or actions.
- Allowing users to set arbitrary Job condition reasons directly.
- Having the Job controller use the new field for any purpose other than the
  Job failure condition message described in this KEP.
- Providing structured, machine-readable identification of individual rules
  during Alpha. This can be revisited during graduation.
- Adding new Job APIs or new Job controller metrics.

## Proposal

Add an optional `name` field to `PodFailurePolicyRule`. When a
`PodFailurePolicyRule` matches a Pod failure and the rule action is `FailJob`,
the Job controller will append the rule name in parentheses to the Job
`FailureTarget` condition message. The final `Failed` condition carries the
same reason and message. Both condition reasons remain `PodFailurePolicy`.

The message retains the zero-based index of the matching rule in
`podFailurePolicy.rules`. If `name` is unset, the message is unchanged.

This follows the [API review recommendation] adopted in
[kubernetes/kubernetes#142913]. Names provide diagnostics without changing the
reason selected by existing clients.

[API review recommendation]: https://github.com/kubernetes/kubernetes/pull/142913#discussion_r4391207198
[kubernetes/kubernetes#142913]: https://github.com/kubernetes/kubernetes/pull/142913

### User Stories

#### Story 1

As a user managing a group of Jobs, I want to identify the rule that failed a
Job without looking up the rule index in its specification.

Example Job:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: named-rule-example
spec:
  podFailurePolicy:
    rules:
    - name: exit-code-2
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
        image: busybox
        command: ["sh", "-c", "exit 2"]
```

The failure condition reason is `PodFailurePolicy`. Its message identifies
`rule at index 0 (exit-code-2)`. Higher-level controllers can continue selecting
the stable reason; they must not depend on parsing this message.

#### Story 2

As a user running HPC simulations, I want to distinguish failures caused by
invalid starting parameters from failures caused by invalid results when
inspecting Jobs. The application exits with one of two exit codes:

- Exit code 2 indicates the simulation produced an invalid result due to bad
  starting parameters and should not be retried.
- Exit code 3 indicates the simulation produced an invalid result, but the
  initial parameters were reasonable, so the simulation should be restarted.

I can name the corresponding rules `invalid-parameters` and `invalid-result`
to make the failure message easier to interpret:

```yaml
podFailurePolicy:
  rules:
  - name: invalid-parameters
    action: FailJob
    onExitCodes:
      operator: In
      values: [2]
  - name: invalid-result
    action: FailJob
    onExitCodes:
      operator: In
      values: [3]
```

These names do not enable different automatic restart actions based on
`onJobFailureReasons`: both rules produce the same `PodFailurePolicy` reason.
Structured identification for higher-level controllers remains a possible
graduation follow-up.

### Notes/Constraints/Caveats

Pod failure policy rules are evaluated in order when a Pod fails. Only the first
matching rule is executed, even if multiple rules match the same Pod failure.
The failure message therefore identifies the first matching `FailJob` rule.

Including names in messages is feature-gated. While the feature gate is
disabled, the message retains the existing index-only format. The condition
reason is `PodFailurePolicy` regardless of the feature gate.

### Risks and Mitigations

The failure message changes when a named rule fails a Job with the feature
enabled. Clients must treat messages as diagnostic text rather than a stable
machine-readable interface. Keeping the condition reason unchanged preserves
existing reason-based selection during rollout, rollback, and version skew.

Validation bounds names to unique DNS labels of at most 63 characters, avoiding
arbitrary text in messages.

The main observability risk is metric cardinality. The Job controller's
`jobs_finished_total` metric has a `reason` label. Implementations of this KEP
must not use user-provided rule names as metric label values. Because the
condition reason remains `PodFailurePolicy`, no metric normalization is needed.

## Design Details

### API changes

Add an optional `name` field to `PodFailurePolicyRule`.

```go
type PodFailurePolicyRule struct {
    // Name is an optional name for this rule. When the rule matches a Pod
    // failure and the action is FailJob, the Job controller appends this name
    // in parentheses to the failure condition message.
    // +featureGate=PodFailurePolicyName
    // +optional
    // +k8s:optional
    // +k8s:format=k8s-short-name
    Name *string `json:"name,omitempty" protobuf:"bytes,4,opt,name=name"`

    // Existing fields omitted.
}
```

### Defaulting

There is no API defaulting for `PodFailurePolicyRule.name`. If `name` is unset,
the stored Job spec remains unchanged, and the controller uses the existing
index-only failure message.

### Validation

- Declarative validation uses `k8s-short-name`: a name must be a non-empty DNS
  label of at most 63 characters, containing only lowercase alphanumeric
  characters or hyphens, starting and ending with an alphanumeric character.
- Handwritten validation checks that names are unique within the Job.
- Unnamed rules are allowed. An explicitly empty name is invalid.
- Numeric names are allowed, including names equal to another rule's index,
  because names and indexes are separate in the message.

### Business logic

When a `PodFailurePolicyRule` matches a Pod failure and `action` is `FailJob`,
the Job controller will set the Job `FailureTarget` condition reason to
`PodFailurePolicy` and construct the failure message:

- Retain the existing failure details and `rule at index <index>` text.
- Append ` (<name>)` when the feature is enabled and the rule has `name` set.
- Leave the message unchanged for unnamed rules or when the feature is disabled.

The final `Failed` condition carries the same reason and message. Names on
`Count`, `Ignore`, or `FailIndex` rules do not affect failure messages.

If the `PodFailurePolicyName` feature gate is disabled, the API server will not
preserve newly introduced names, unless names are already in use in the stored
Job spec. This pruning behavior is shared by Jobs and CronJob templates. The
Job controller ignores names when the gate is disabled; the reason stays
`PodFailurePolicy` in all cases.

### Metrics

This KEP does not add new metrics.

The Job controller already records finished Jobs in `jobs_finished_total` with a
bounded `reason` label. Since the condition reason remains `PodFailurePolicy`,
metric recording is unchanged and rule names do not appear in labels.

The existing `pod_failures_handled_by_failure_policy_total` metric remains
unchanged and continues to report the matched action.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

No prerequisite testing updates are required.

##### Unit tests

Unit tests will cover:

- API validation for duplicate rule names, DNS label syntax, and the 63-character
  limit, including rejection of empty, uppercase, and dotted names.
- Declarative validation equivalence for Jobs and CronJobs, including CronJob
  template updates.
- API strategy behavior with the feature gate enabled and disabled.
- Controller handling for named and unnamed `FailJob` rules matching
  `onExitCodes`.
- Controller handling for named and unnamed `FailJob` rules matching
  `onPodConditions`.
- Controller behavior with names present while the feature gate is disabled.
- Metric recording that keeps `jobs_finished_total` reason labels bounded by
  retaining the `PodFailurePolicy` reason.

Current coverage for packages expected to be touched:

- `k8s.io/kubernetes/pkg/controller/job`: `2024-05-02` - `91.5%`
- `k8s.io/kubernetes/pkg/apis/batch/v1`: `2024-05-06` - `87.3%`
- `k8s.io/kubernetes/pkg/apis/batch/v1beta1`: `2024-05-06` - `78.3%`
- `k8s.io/kubernetes/pkg/apis/batch/validation`: `2024-05-06` - `87.7%`

##### Integration tests

Integration tests will cover:

- When the feature gate is enabled and a Job's pod failure policy triggers a Job
  failure through a named rule, the `FailureTarget` and `Failed` conditions have
  reason `PodFailurePolicy` and include the name in their messages.
- When the feature gate is enabled and a Job's pod failure policy triggers a Job
  failure through an unnamed rule, the message retains the index-only format.
- When the feature gate is disabled, new names are pruned and the Job failure
  message retains the index-only format.
- The `jobs_finished_total` reason label remains `PodFailurePolicy`.

##### e2e tests

An e2e test will create a Job with `podFailurePolicy.rules[*].name`, trigger a
Pod failure that matches the named `FailJob` rule, and verify that the Job
`Failed` condition has reason `PodFailurePolicy` and includes the name in its
message.

An additional e2e test will cover an unnamed rule and verify the
unchanged index-only message.

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
- Revisit whether higher-level controllers need structured rule identification.
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
required for existing Jobs. Users can opt in to named failure messages by
enabling the `PodFailurePolicyName` feature gate and setting
`.spec.podFailurePolicy.rules[*].name` on new Jobs.

On downgrade, or when the feature gate is disabled:

- New Jobs cannot rely on `PodFailurePolicyRule.name`; older API servers or API
  servers with the gate disabled will prune newly introduced names.
- Existing Jobs that already have the field may be reconciled by a Job
  controller that ignores it, causing new pod-failure-policy-triggered Job
  failures to use index-only messages. API servers with the gate disabled
  preserve names when they are already in use in the stored spec.
- Existing terminal Job conditions are not rewritten solely because the feature
  is disabled or the control plane is downgraded.

Controllers that consume Job failure reasons continue to see `PodFailurePolicy`
during upgrade, downgrade, and rollback. They must not parse messages to
identify individual rules programmatically.

### Version Skew Strategy

This feature is limited to the control plane. It does not require kubelet,
kube-proxy, CRI, CNI, or CSI changes.

In an HA control plane with skewed kube-apiserver versions or feature-gate
configuration, requests that include `PodFailurePolicyRule.name` may be accepted
by some API servers and pruned or rejected by others until rollout is complete.

If the kube-controller-manager leader is an older version or has
`PodFailurePolicyName` disabled, the built-in Job controller ignores the new
field and uses index-only failure messages. When
a new controller-manager with the gate enabled becomes leader, subsequent
pod-failure-policy-triggered Job failures include configured names in messages.
The condition reason remains `PodFailurePolicy` in both cases.

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

For Jobs whose pod failure policy has a named matching `FailJob` rule, enabling
the feature appends the name to the failure condition message. Unnamed rules
retain their existing messages. Condition reasons, pod failure policy matching,
ordering, and actions are unchanged.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disable the `PodFailurePolicyName` feature gate on kube-apiserver and
kube-controller-manager. New writes prune newly introduced names, while names
already in use in the stored spec are preserved. The Job controller resumes
using index-only messages for subsequent Job failures.

###### What happens if we reenable the feature if it was previously rolled back?

For new Jobs, the API server will preserve `PodFailurePolicyRule.name` again.
For existing Jobs that still contain the field, the Job controller will resume
using it for subsequent pod-failure-policy-triggered Job failures.

###### Are there any tests for feature enablement/disablement?

The implementation will add unit and integration tests for:

- Feature gate enabled with `PodFailurePolicyRule.name` set.
- Feature gate enabled with `PodFailurePolicyRule.name` unset.
- Feature gate disabled with the field set on incoming objects.
- Feature gate disabled with names already in use in stored specs.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Partial rollout can result in mixed behavior while different API servers or
controller-manager instances run with different versions or feature-gate values.
Some requests may preserve `PodFailurePolicyRule.name` while others prune or
reject it, and the active Job controller may emit index-only messages or
messages with rule names. The condition reason is unchanged.

This does not stop already running Pods or Jobs. The impact is limited to Job
spec persistence for the new field and the failure message observed when a pod
failure policy fails a Job.

###### What specific metrics should inform a rollback?

A substantial increase in `job_sync_duration_seconds` may indicate the Job
controller is spending more time reconciling Jobs after the feature is enabled.

Operators should also monitor `jobs_finished_total{reason="PodFailurePolicy"}`
and `pod_failures_handled_by_failure_policy_total{action="FailJob"}` to check
whether Jobs are failing through pod failure policy at the expected rate. The
`jobs_finished_total` metric must keep the bounded `PodFailurePolicy` reason
label regardless of whether Job condition messages include rule names.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

The Alpha implementation includes tests for enabled and disabled behavior and
preservation of names already in use. The full upgrade->downgrade->upgrade path
has not yet been tested. Before Beta, it will be tested to verify field
preservation, field pruning, index-only message fallback, restoration of names
in messages, and non-rewriting of existing terminal Job conditions.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No APIs, fields, feature gates, or flags are removed. The `PodFailurePolicy`
Job condition reason is unchanged.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

An operator can inspect Job specs for `.spec.podFailurePolicy.rules[*].name`.
For aggregate observation, a non-zero
`pod_failures_handled_by_failure_policy_total{action="FailJob"}` value indicates
that pod failure policy is failing Jobs, and `jobs_finished_total` indicates how
many Jobs finished with the bounded `PodFailurePolicy` reason label.

###### How can someone using this feature know that it is working for their instance?

- [X] Job API `.status`
  - Condition names: `FailureTarget` and `Failed`
  - Other fields: `reason` is `PodFailurePolicy`, and `message` includes the
    configured rule name in parentheses after the rule index.

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
only changes the condition message used in that existing update.

###### Will enabling / using this feature result in introducing new API types?

No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Yes, for Jobs and CronJob templates that set `PodFailurePolicyRule.name`, and
Jobs that later receive a failure condition message containing the name.

- API types: Job and CronJob
- Estimated spec size increase: up to 63 characters per configured rule name,
  plus serialization overhead.
- Estimated status size increase: each Job failure condition message may grow
  by up to 66 characters for the name and its ` (<name>)` suffix. Condition
  reasons are unchanged.
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
  - Mitigation: Use a unique, non-empty lowercase DNS label of at most 63
    characters, with alphanumeric characters at both ends.
  - Diagnostics: The API validation error identifies the invalid field.
  - Testing: Covered by API validation unit and integration tests.
- Feature gate disabled or old controller-manager leader
  - Detection: Job failure messages omit names even though rule names are
    configured and a named `FailJob` rule matches.
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
- 2026-11-07: Alpha implementation merged in [kubernetes/kubernetes#142913],
  retaining the `PodFailurePolicy` reason and including rule names in messages.

## Drawbacks

Rule names improve diagnostics but do not provide a structured way for
higher-level controllers to select individual failure rules. Such controllers
must continue using the `PodFailurePolicy` reason and must not rely on message
parsing. Structured identification can be revisited during graduation.

## Alternatives

One alternative was to add an optional `reason` or `setConditionReason` field to
`PodFailurePolicyRule` and use that value directly as the Job failure condition
reason. This was rejected because it would let users set arbitrary Job condition
reasons, making validation and reason ownership less clear.

Another alternative was to derive more specific reasons automatically, such as
`PodFailurePolicy_ExitCode143` for `onExitCodes` rules. This does not generalize
well to `onPodConditions`, and it gives users less control over the stable
machine-readable value consumed by higher-level controllers.

The earlier proposal generated `PodFailurePolicy_<name>` or
`PodFailurePolicy_<index>` reasons. API review favored keeping the existing
reason stable and including names in messages, because generated reasons are
harder to select programmatically and require more complex validation.

## Infrastructure Needed

No new infrastructure is needed.
