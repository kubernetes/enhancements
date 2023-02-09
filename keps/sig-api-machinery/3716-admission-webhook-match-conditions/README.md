# KEP-3716: Admission Webhook Match Conditions

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [User Stories](#user-stories)
    - [Exclude resources from a wildcard rule](#exclude-resources-from-a-wildcard-rule)
    - [Exempt system users from security policy](#exempt-system-users-from-security-policy)
    - [Scope an NFS access management webhook to Pods mounting NFS volumes](#scope-an-nfs-access-management-webhook-to-pods-mounting-nfs-volumes)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [API](#api)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Security](#security)
    - [Debuggability](#debuggability)
    - [Performance](#performance)
- [Design Details](#design-details)
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
- [Future Work](#future-work)
  - [Cross-webhook match conditions](#cross-webhook-match-conditions)
- [Alternatives](#alternatives)
  - [Exclusion Expressions](#exclusion-expressions)
  - [Resource Exclusions](#resource-exclusions)
  - [CEL Admission Control](#cel-admission-control)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests for meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [ ] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes adding "match conditions" to admission webhooks, as an extension to the
existing `rules` to define the scope of a webhook. A `matchCondition` is a
[CEL](https://github.com/google/cel-spec) expression that must evaluate to true for the admission
request to be sent to the webhook. If a `matchCondition` evaluates to false, the webhook is skipped for
that request (implicitly allowed).

## Motivation

**Reliability:** Admission webhooks continue to be an operational sore spot for many Kubernetes
users. Webhooks that target cluster critical resources put the admission controller backing the
webhook in the critical path of cluster stability. Even if tools like namespace scoping are used to
avoid circular-dependencies and exclude critical system resources, a webhook outage can still have a
major impact on cluster availability. This proposal aims to mitigate (but not eliminate) these
issues by allowing webhooks to be more narrowly scoped and targeted.

**Performance:** Admission webhooks sit in the critical request path for write-requests. Validating
webhooks can be run in parallel, but Mutating webhooks must be run in serial (up to 2 times!). This
makes webhooks extremely latency sensitive, and even a webhook that doesn't do any work still needs
to pay the network round-trip cost.

**Supportability:** For hosted or managed Kubernetes distributions, webhooks can be a problem when
they interfere with requests by managed components. The existing criteria for filtering out requests
are insufficient for many use cases, and aren't easily appended with provider rules.

_What about [CEL for Admission Control](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/3488-cel-admission-control)?_
`ValidatingAdmissionPolicy` is an exciting new feature which we hope will greatly reduce the need
for admission webhooks, but it is intentionally not attempting to cover every possible use case.
This proposal aims to improve the situation for those webhooks that cannot be migrated.

### User Stories

#### Exclude resources from a wildcard rule

> I want to enforce metadata policy through an admission webhook without adding latency & risk to
> high QPS system requests.

Currently, if a webhook uses wildcard match rules, there is no way to filter out a subset of
resources or requests from matching the wildcard. If the webhook instead enumerates every resource
that should match, it must be kept up-to-date with every CRD that's added.

With CEL match conditions, the webhook could specify wildcard match rules, and add match conditions
to filter out the desired resources:

```yaml
rules:
  # Match CREATE & UPDATE on all resources:
  - operations:
    - CREATE
    - UPDATE
    apiGroups: '*'
    apiVersions: '*'
    resources: '*'
matchConditions:
  - name: 'exclude-leases'
    expression: '!(request.resource.group == "coordination.k8s.io" && resource.resource == "leases")'
```

#### Exempt system users from security policy

> As a managed cluster provider, I want to prevent user webhooks from intercepting critical system
> requests.

System _resources_ can currently be exempted through a namespace or label selector, but requests by
system components against non-system resources cannot be. For example, update pod status requests by
Kubelets cannot be excluded from user webhooks intercepting all pod requests.

With `matchConditions`, a managed cluster could append system-exclusion rules to each webhook. For example:

```yaml
matchConditions:
  - name: 'exclude-kubelet-requests'
    expression: '!("system:nodes" in request.userInfo.groups)'
```

Since the expression will be evaluated using a common Kubernetes CEL library, these expressions
should also get automatic access to the secondary authorization check mechanism described in
[KEP-3488: CEL for Admission Control](/keps/sig-api-machinery/3488-cel-admission-control#secondary-authz).
In practice, this means that RBAC bindings can be used to opt-out privileged users from security policy:

```yaml
matchConditions:
  # Requests by users without breakglass should be included.
  - name: 'breakglass'
    expression: 'authorizer.resource('admissionregistration.k8s.io', 'validatingwebhookconfigurations', '*').name('security-policy').check('breakglass').denied()'
```

#### Scope an NFS access management webhook to Pods mounting NFS volumes

> I want to narrowly scope my webhook to only the relevant requests, in order to reduce load on the
> webhook and reduce latency in irrelevant requests.

Concrete example:

> A NFS deployment uses an third-party access management system. I have an admission webhook that
> performs an access check for against the external system for pods that mount NFS volumes. Only
> pods with NFS volumes need to be checked.

Currently, there is no way to achieve this. Many webhook implementations today start by checking
that the request is within scope, and return early if it's not. This adds latency and an additional
failure point to irrelevant requests. This example requires an external integration, and thus is not
a candidate for migration to CEL `ValidatingAdmissionPolicy`.

With match conditions, the expressions can check whether the request object is in-scope for the
webhook:

```yaml
rules:
  - operations: ['CREATE']
    apiGroups: '' # core
    apiVersions: '*'
    resources: 'pods'
matchConditions:
  # Only include pods with an NFS volume.
  - name: 'nfs-volume-present'
    expression: 'object.spec.volumes.exists(v, v.has(nfs))'
```

### Goals

1. Provide a filtering mechanism for excluding requests from an admission webhook
2. Maintain consistency with `ValidatingAdmissionPolicy`

### Non-Goals

* Provide a mechanism to exclude requests from all webhooks.

## Proposal

### API

Both `ValidatingWebhook` and `MutatingWebhook` (in `admissionregistration.k8s.io`) will be updated
with a new `MatchConditions` field:

```go

type ValidatingWebhook struct {
  // ...

  // MatchConditions is a list of conditions on the AdmissionRequest ('request') that must be met
  // for a request to be sent to this webhook. All conditions in the list must evaluate to TRUE for
  // the request to be matched.
  // +optional
  // +patchMergeKey=name
  // +patchStrategy=merge
  MatchConditions []MatchCondition `json:"matchConditions,omitempty"`
}

type MutatingWebhook struct {
  // ...
  MatchConditions []MatchCondition `json:"matchConditions,omitempty"`
}

// MatchCondition represents a condition which must by fulfilled for a request to be sent to a webhook.
type MatchCondition struct {
  // Name is an identifier for this match condition, used for strategic merging of MatchConditions,
  // as well as providing an identifier for logging purposes. A good name should be descriptive of
  // the associated expression.
  // Name must be a valid RFC 1123 DNS subdomain, and unique in a set of MatchConditions.
  // Required.
  Name string `json:"name"`
  // NOTE: Placeholder documentation, to be replaced by https://github.com/kubernetes/website/issues/39089.
  //
	// Expression represents the expression which will be evaluated by CEL.
	// ref: https://github.com/google/cel-spec
	// CEL expressions have access to the contents of the AdmissionRequest, organized into CEL variables:
	//
	// 'object' - The object from the incoming request. The value is null for DELETE requests.
	// 'oldObject' - The existing object. The value is null for CREATE requests.
	// 'request' - Attributes of the admission request([ref](/pkg/apis/admission/types.go#AdmissionRequest)).
	//
	// Required.
	Expression string `json:"expression"`
}
```

The match condition expression is evaluated by the same libraries as those used for CEL
ValidatingAdmissionPolicy. The only difference in expressions is the availability of the `params`
variable. Expressions requiring access to additional information outside the AdmissionRequest must
be performed in the webhook, and are out of scope for this proposal.

### Risks and Mitigations

#### Security

**Risk: Attacker adds or changes a match condition to weaken an admission policy.**

This is does not represent a new threat, as doing so would require update access to the admission
registration object, and with that permission an attacker could already disable the policy through
manipulating match rules, namespace selector, or object selector (or reroute the webhook entirely).

**Risk: Logic error in match condition expression.**

Currently the match conditions must be encoded in the webhook backend itself. Moving the logic into
a CEL expression adds a potential failure point. This can be mitigated by testing, but the CEL
ecosystem currently lacks some of the tools that would make this easier.

Of particular significance are match conditions tied to non-functional properties of an object, such
as using labels to decide whether to opt an object out of a policy. Without additional admission
controls on who can set those non-functional aspects, exempting the policy based on that could be a
security vulnerability. In contrast, the
[NFS example usecase](#scope-an-nfs-access-management-webhook-to-pods-mounting-nfs-volumes) exempts
the policy on a _functional_ aspect - whether an NFS volume is mounted, and thus whether the policy
is relevant.

These risks are inherent to the feature being proposed and cannot be mitigated through technical
means, but should be highlighted in the documentation.

#### Debuggability

We do not normally log, audit, or emit an event when a webhook is out-of-scope for a request, and
the same will _mostly_ be true for match conditions.

At [log level V(5)](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-instrumentation/logging.md#what-method-to-use),
we will emit a log when a request that would otherwise be in-scope for a webhook is excluded for a
non-matching match condition.

Short of increasing log verbosity, the recommended debug strategy is to capture or reproduce a
relevant AdmissionRequest (for example, in a non-prod cluster disable all match conditions and log
the requests from a webhook). Then, manually test the match conditions against the request, and
iterate as necessary.

#### Performance

The CEL expression evaluation will leverage the same [Resource Constraints](https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/2876-crd-validation-expression-language#resource-constraints)
used by CEL CRD Validation & CEL Admission Control. All the match conditions for a given webhook will
share the same resource budget.

<<[UNRESOLVED resource constraints ]>>
_NON-BLOCKING for Alpha_
Details TBD.
<<[/UNRESOLVED]>>

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### Test Plan

<!--
**Note:** *Not required until targeted at a release.*
The goal is to ensure that we don't accept enhancements with inadequate testing.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

TBD - unit tests will be added as this feature is implemented.

##### Integration tests

Test cases to add:

- [ ] Feature gate enablement / disablement is a no-op when no `matchConditions` are set
- [ ] Feature gate enablement / disablement works as expected when `matchConditions` are set
- [ ] Single match condition:
    - [ ] Request out of scope without `matchConditions`
    - [ ] Request in scope without `matchConditions`, but not matching
    - [ ] Request in scope without `matchConditions`, and also matching
- [ ] Multiple match conditions, covering the same cases as the single-condition case

##### e2e tests

We will test the edge cases mostly in integration tests and unit tests.

Once the feature is default enabled in beta, a single E2E test covering hte single-match-condition
cases outlined above will be added.

### Graduation Criteria

#### Alpha

- Feature implemented behind `AdmissionWebhookMatchConditions` feature flag
- [Integration tests](#integration-tests) implemented

#### Beta

- Add E2E test coverage
- Resolve resource constraints validation

<<[UNRESOLVED resource constraints ]>>
Additional beta requirements TBD
<<[/UNRESOLVED]>>

#### GA

<<[UNRESOLVED resource constraints ]>>
GA requirements TBD
<<[/UNRESOLVED]>>

### Upgrade / Downgrade Strategy

Downgrading in a way that disables match conditions after it is already in use can increase the
scope of requests evaluated by a webhook. See
[Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?](#can-the-feature-be-disabled-once-it-has-been-enabled-ie-can-we-roll-back-the-enablement)
for more details

### Version Skew Strategy

The new field is only evaluated by the apiserver, so only HA apiserver version skew is relevant. In
this case, if the feature is enabled in one apiserver and not another, a request could
non-deterministically be sent to a webhook. Enabling match conditions without setting
`matchConditions` on an webhooks is a no-op, so the version skew non-determinism is best avoided by
waiting until it has been enabled in all apiservers before starting to use the new field.

## Production Readiness Review Questionnaire

<!--

Production readiness reviews are intended to ensure that features merging into
Kubernetes are observable, scalable and supportable; can be safely operated in
production environments, and can be disabled or rolled back in the event they
cause increased failures in production. See more in the PRR KEP at
https://git.k8s.io/enhancements/keps/sig-architecture/1194-prod-readiness.

The production readiness review questionnaire must be completed and approved
for the KEP to move to `implementable` status and be included in the release.

In some cases, the questions below should also have answers in `kep.yaml`. This
is to enable automation to verify the presence of the review, and to reduce review
burden and latency.

The KEP must have a approver from the
[`prod-readiness-approvers`](http://git.k8s.io/enhancements/OWNERS_ALIASES)
team. Please reach out on the
[#prod-readiness](https://kubernetes.slack.com/archives/CPNHUMN74) channel if
you need any help or guidance.
-->

### Feature Enablement and Rollback

<!--
This section must be completed when targeting alpha to a release.
-->

###### How can this feature be enabled / disabled in a live cluster?

- [X] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `AdmissionWebhookMatchConditions`
  - Components depending on the feature gate: `kube-apiserver`

###### Does enabling the feature change any default behavior?

No. If the feature is enabled, but the `matchConditions` field is unset, the default behavior
remains unchanged.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Disabling the feature gate will ignore any `matchConditions` set, and return to the default
behavior. Disabling `AdmissionWebhookMatchConditions` could increase the traffic to the webhook, and
potentially increase the error rate if the webhook fails to process the additional requests
correctly.

###### What happens if we reenable the feature if it was previously rolled back?

Any `matchConditions` that were already stored on existing webhooks will be enforced.

Note: enabling `matchConditions` can only reduce the number of requests being sent to a webhook (or
remain unchanged). Enabling it will never increase the number of requests.

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.

Additionally, for features that are introducing a new API field, unit tests that
are exercising the `switch` of feature gate itself (what happens if I disable a
feature gate after having objects written with the new field) are also critical.
You can take a look at one potential example of such test in:
https://github.com/kubernetes/kubernetes/pull/97058/files#diff-7826f7adbc1996a05ab52e3f5f02429e94b68ce6bce0dc534d1be636154fded3R246-R282
-->

[Registry tests](https://github.com/kubernetes/kubernetes/blob/c4ebbeeb747cd3e2b1d83733a14d367a65723a45/pkg/registry/core/pod/strategy_test.go)
will verify the drop disabled fields logic is correctly implemented.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout or rollback fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?

Be sure to consider highly-available clusters, where, for example,
feature flags will be enabled on some API servers and not others during the
rollout. Similarly, consider large clusters and how enablement/disablement
will rollout across nodes.
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

A new per-webhook metric will measure the number of requests excluded by match conditions:

Metric name: `webhook_admission_match_condition_exclusions_total`
Labels:
- `name`: webhook name
- `type`: `validate` or `admit`
- `operation`: the admission operation

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

- [ ] Events
  - Event Reason: 
- [ ] API .status
  - Condition name: 
  - Other field: 
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

<!--
This is your opportunity to define what "normal" quality of service looks like
for a feature.

It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99.9% of /health requests per day finish with 200 code

These goals will help you determine what you need to measure (SLIs) in the next
question.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

See [Debuggability](#debuggability).

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Future Work

### Cross-webhook match conditions

In the future, we should explore ways to apply common match conditions across multiple webhooks.

Example use cases:
- Apply a [break-glass exemption](#exempt-system-users-from-security-policy) across many (or all) webhooks.
- Managed cluster provider wants to exempt provider-managed resources from user-managed webhooks.

Considerations:
- Access by managed cluster provider vs. cluster admin
- Side effects & mutations

## Alternatives

### Exclusion Expressions

The `matchCondition` expression could be inverted, so that requests that match are excluded rather than
included. In this case, we would probably also want to change from requiring all expressions to
match, to excluding the request if any match.

Although this approach would simplify some usecases, such as
[excluding resources from a wildcard rule](#exclude-resources-from-a-wildcard-rule) or
[exempting system users from a security policy](#exempt-system-users-from-security-policy), it
means other expressions would become double-negatives, which generally goes against API design
best-practices.

### Resource Exclusions

[KEP-3693](https://github.com/kubernetes/enhancements/issues/3693) Proposes an alternative approach
using a more structured format for expressing resource exclusions. This approach may be more
approachable to users who are not comfortable writing CEL expressions, but it is significantly less
powerful. This would address
[Exclude resources from a wildcard rule](#exclude-resources-from-a-wildcard-rule),
and could be extended with subject exclusions to
address [Exempt system users from security policy](#exempt-system-users-from-security-policy), but
would not be sufficient to address
[Scope an NFS access management webhook to Pods mounting NFS volumes](#scope-an-nfs-access-management-webhook-to-pods-mounting-nfs-volumes).

These two approaches are not mutually exclusive.

### CEL Admission Control

[KEP-3488: CEL for Admission Control](/keps/sig-api-machinery/3488-cel-admission-control) adds the
ability for admission webhooks to be replaced entirely by CEL expressions, but this is not intended
to cover 100% of webhook use cases. For example, the user story described in
[Scope an NFS access management webhook to Pods mounting NFS volumes](#scope-an-nfs-access-management-webhook-to-pods-mounting-nfs-volumes)
requires integrating with a third-party system, and is not implementable through a CEL
ValidatingAdmissionPolicy.

With a mutating CEL admission policy (not yet implemented), a combination of mutating & validating
policies could ensure that objects have a designated scoping label applied, which could be filtered
using the `ObjectSelector` on the webhook. However, such an approach adds a lot of overhead and
complexity beyond this proposal.
