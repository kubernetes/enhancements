# KEP-2799: Token Controller Deprecation

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Token Controller](#token-controller)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [TokenControllerOptOut](#tokencontrolleroptout)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [TokenControllerPurge](#tokencontrollerpurge)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation-1)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation-1)
    - [TokenControllerDeprecation](#tokencontrollerdeprecation)
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
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required _prior to targeting to a milestone /
release_.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
- [x] (R) Graduation criteria is in place
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

This KEP proposes a namespace-scoped and service-account-scoped binary label to
allow users to opt in/out the provision of secret-based service account tokens
in Token Controller. In addition, it sketches out the phases to deprecate Token
Controller.

## Motivation

As BoundServiceAccountTokenVolume is GA in 1.22, pods’ service account tokens
would be obtained via TokenRequest API and stored as projected volume. This
change obviates the need for Token Controller’s reconciling of secret-based
service account tokens. The secret-based tokens are [not secure by design](https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/1205-bound-service-account-tokens#background)
and the token controller is [fragile by design in some cases](https://github.com/kubernetes/kubernetes/issues/98474)
where it is unable to handle the churns between secrets and service account
controller loops.

### Goals

- Provides a knob to opt in/out the generation of secret-based token for a
  service account
- Provides a knob to opt in/out the generation of secret-based token for all
  service accounts in a namespace
- Deprecate Token Controller
- No Forever Token in Kubernetes

### Non-Goals

## Proposal

With the goal to deprecate Token Controller,

- Phase I: At release N, `tokencontroller.kubernetes.io/reconcile` is `true` by default
  to align with existing behavior of Token Controller. To disable the
  generation of secret-based tokens, users are required to specify
  `tokencontroller.kubernetes.io/reconcile=false` for the service accounts or
  namespaces that they would like to prevent secret-based tokens. To spread
  the “fire”, some service accounts in `kube-system` will be annotaed. At the
  same time, warn usage of secret-based tokens. This phase creates leeway for
  users to migrate off their use of secret-based service account tokens.
- Phase II: At release N+X (X>0), `tokencontroller.kubernetes.io/reconcile` is `false`
  by default. Token Controller will no longer generate secret-based token
  unless `tokencontroller.kubernetes.io/reconcile=true`. In addition, keep
  warn usage of legacy tokens that in future releases, secret-based tokens
  would be purged.
- Phase III: At release N+Y (Y>X), begin to purge all secret-based tokens in existing
  clusters unless the referenced service accounts or namespaces have
  `tokencontroller.kubernetes.io/reconcile=true`.
- Phase IV: At release N+Z (Z>Y), remove the annotation and all
  secret-based tokens are purged.
- Phase V: At release N+A (A>Z), Token Controller and secret-based token authenticator
  are removed from the tree.

### Notes/Constraints/Caveats

- For clusters in upgrade path, users should not upgrade to release >=N+X unless
  they are certain of no active usage of secret-based tokens. To consult that
  information, metric `serviceaccount_stale_tokens_total` or audit annotation
  `authentication.k8s.io/stale-token` could be used.
- A warning mechanism should be implemented to push users to migrate and it
  will exist for at least one year before release N+Y.

### Risks and Mitigations

- Phase I, there is only risk in implementation which would be mitigated by
  tests.
- Phase II, workloads that still use secret-based token might start to fail if
  the tokens are somehow deleted. It could be mitigated by annotate the service
  account or namespace with `tokencontroller.kubernetes.io/reconcile=true` to
  enable the reconciling of Token Controller or switch to bound service account
  token.
- Phase III, workloads that still use secret-based token will start to fail. It
  could be mitigated by annotate the service account or namespace with
  `tokencontroller.kubernetes.io/reconcile=true` to enable the reconciling of
  Token Controller or switch to bound service account token.
- Phase IV, workloads that still use secret-based token will start to fail.
  Workloads have to switch to bound service account token to recover.
- Phase V, None.

## Design Details

### Token Controller

1.  Token Controller firstly examines the existence of the label
    `tokencontroller.kubernetes.io/reconcile` on the service account or the
    namespace of the service account. The value of the label on the service
    account overwrites the one on the namespace if both exists.
2.  If `tokencontroller.kubernetes.io/reconcile=false` either explicitly or
    implicitly (default value), Token Controller would neither generate the
    secret nor update the references on service account.

### Test Plan

- Unit tests
- E2E tests
- Upgrade tests

### Graduation Criteria

#### TokenControllerOptOut

| Alpha | Beta | GA   |
| ----- | ---- | ---- |
| 1.23  | 1.24 | 1.26 |

This feature gate includes phase I and phase II:

- Phase I maps to Alpha and Beta. This should assure the implementation of
  opt-out functionality is sound and can be disabled in case of bugs.
- Phase II maps to GA. This is basically switching the default behavior and
  could break users' workloads. According to the projected schedule above,
  we would alert users about the change for at least one year.

#### Beta -> GA Graduation

- [ ] In use by multiple distributions
- [ ] Approved by PRR and scalability
- [ ] Any known bugs fixed
- [ ] Tests passing
- [ ] At least two releases since Alpha to allow enough time warning users

#### Alpha -> Beta Graduation

- [ ] In use by multiple distributions
- [ ] Approved by PRR and scalability
- [ ] Any known bugs fixed
- [ ] Tests passing

#### TokenControllerPurge

| Alpha | Beta | GA   |
| ----- | ---- | ---- |
| 1.27  | 1.28 | 1.29 |

This feature gate includes phase III and phase IV:

- Phase III maps to Alpha and Beta. This should assure the implementation of
  purge is sound and can be disabled in case of bugs.
- Phase IV maps to GA where we remove the opt out functionality to purge all
  legacy tokens.

#### Beta -> GA Graduation

- [ ] In use by multiple distributions
- [ ] Approved by PRR and scalability
- [ ] Any known bugs fixed
- [ ] Tests passing

#### Alpha -> Beta Graduation

- [ ] In use by multiple distributions
- [ ] Approved by PRR and scalability
- [ ] Any known bugs fixed
- [ ] Tests passing

#### TokenControllerDeprecation

This feature gate includes phase V.

| Alpha | Beta | GA   |
| ----- | ---- | ---- |
| 1.31  | 1.32 | 1.33 |

### Upgrade / Downgrade Strategy

When a cluster upgrades to a version in:

- Phase I, users' workload behavior remains the same; to make use of the
  enhancement, they need to annotate the service accounts or namespaces. however,
  in-tree components' secret-based tokens would not be reconciled once deleted.
- Phase II, users' workload are required to be annotated or switch to use bound
  service account tokens to remain the same behavior.
- Phase III, users' workload are required to be annotated or switch to use bound
  service account tokens to remain the same behavior.
- Phase IV, users' workload are required to be annotated or switch to use bound
  service account tokens to remain the same behavior.
- Phase IV, users' workload has to switch to use bound service account tokens to
  remain the same behavior.
- Phase IV, users' workload has to switch to use bound service account tokens to
  remain the same behavior.

### Version Skew Strategy

The only touches control plane, so version skew strategy is not applicable.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: TokenControllerOptOut
  - Components depending on the feature gate: kube-controller-manager
  - Feature gate name: TokenControllerPurge:
  - Components depending on the feature gate: kube-controller-manager
  - Feature gate name: TokenControllerDeprecation:
  - Components depending on the feature gate: kube-controller-manager, kube-apiserver

###### Does enabling the feature change any default behavior?

- TokenControllerOptOut: no default behavior changed in beta. in ga, workloads
  need to add annotation or migrate to bound token to maintain the same behavior
  if legacy tokens are deleted after enabling the feature.
- TokenControllerPurge: workloads need to add annotation or migrate to bound
  token to maintain the same behavior in beta. in ga, workloads have to migrate
  to use bound tokens.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

the same as enable the feature.

###### Are there any tests for feature enablement/disablement?

no as there is no API changes which could be covered by unit tests.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

workloads using legacy tokens would start to fail.

###### What specific metrics should inform a rollback?

`serviceaccount_stale_tokens_total`: cumulative stale projected service
account tokens used.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TODO in beta

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

yea, token controller would be deprecated.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
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

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

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

no.

###### Will enabling / using this feature result in introducing new API types?

no.

###### Will enabling / using this feature result in any new calls to the cloud provider?

no.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

no.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

no.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

no.

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

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

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
