# KEP-2799: Reduction of Secret-based Service Account Tokens

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [LegacyServiceAccountTokenNoAutoGeneration:](#legacyserviceaccounttokennoautogeneration)
  - [LegacyServiceAccountTokenTracking](#legacyserviceaccounttokentracking)
  - [LegacyServiceAccountTokenCleanUp](#legacyserviceaccounttokencleanup)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [LegacyServiceAccountTokenNoAutoGeneration](#legacyserviceaccounttokennoautogeneration-1)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [LegacyServiceAccountTokenTracking](#legacyserviceaccounttokentracking-1)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation-1)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation-1)
    - [LegacyServiceAccountTokenCleanUp](#legacyserviceaccounttokencleanup-1)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation-2)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation-2)
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

This KEP proposes actions to reduce the surface area of secret-based service
account tokens.

## Motivation

As BoundServiceAccountTokenVolume is GA in 1.22, pods’ service account tokens
would be obtained via TokenRequest API and stored as projected volume. This
change obviates the need for auto-generation of secret-based service account
tokens which are [less secure than the bound token](https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/1205-bound-service-account-tokens#background).

### Goals

- No auto-generation of secret-based service account token.
- Removal of unused auto-generated secret-based service account tokens

### Non-Goals

- Removal of [explicitly requested secret-based service account tokens](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#manually-create-a-service-account-api-token).

## Proposal

- Change the service account control loop in Token Controller to not auto-create
  secret for service accounts. At the same time, warn usage of auto-created
  secret-based service account tokens and encourage users to use TokenRequest
  API or manually-created secret-based service account tokens.
- Purge unused auto-generated secret-based service account tokens.

### User Stories (Optional)

### Notes/Constraints/Caveats

- A warning mechanism should be implemented to help users
  migrate.
- Auto generated secret-based service account tokens are those requested by
  Token Controller.
- Only clean up auto-generated tokens which:
  - are not referenced by pods
  - have not been used to authenticate for some duration (time duration or number of releases)
- To consult active usage of secret-based tokens, metric
  `serviceaccount_legacy_tokens_total` or audit annotation
  `authentication.k8s.io/legacy-token` could be used.

### Risks and Mitigations

- When feature LegacyServiceAccountTokenNoAutoGeneration is Beta, consumers
  depending directly on waiting for and reading tokens out of auto-generated
  secrets might stop working. To mitigate,
  1. Emit warnings when using auto-generated token secrets.
  2. Publish pointers to TokenRequest or the manual secret request flow.
- When LegacyServiceAccountTokenCleanUp is Beta, usage of auto-generated
  secret-based token might stop working. To mitigate,
  1. When Alpha, annouce the cleanup starts at Beta
  2. Emit warnings when using auto-generated token secrets.
  3. Add pointers of TokenRequest API and manually created tokens in the validation
     result.

## Design Details

### LegacyServiceAccountTokenNoAutoGeneration:

Token Controller stops auto-creating secret for service accounts. This feature would
be enabled when it is implemented since no new code is added and this can make
sure new clusters are in good state.

### LegacyServiceAccountTokenTracking

To facilitate LegacyServiceAccountTokenCleanUp, we implement a simple controller
in kube-apiserver that maintains a bool value configmap in `kube-system` to
indicates if tracking is enabled in the cluster. It is similar to the existing
`ClusterAuthenticationTrustController` that maintains `configmap/extension-apiserver-authentication`.

- When LegacyServiceAccountTokenTracking is enabled in all apiservers,

  - the controller creates/updates a configmap in `kube-system` namespace that
    stores the current date as `tracked-since`.
  - when a legacy token is used, issue a warning, annotate/update the
    `last-used` on the secret at date granularity, and record in a metric.
    optionally, add a label `in-use` for fast query.

- When LegacyServiceAccountTokenTracking is disabled in any apiserver,
  - the controller ensures the configmap in `kube-system` namespace is deleted
    in a periodic way.

### LegacyServiceAccountTokenCleanUp

Token Controller starts to remove unused auto-generated secrets (secrets
bi-directionally referenced by the service account) and not mounted by pods.

When this feature is Beta and enabled by default, delete secrets iff it is over
a sufficient period of time (one year by default) since last used. The period
can be configured by cluster admins.

Determine the date that a given secret was last used:

1. `last-used` if exists and after `tracked-since`.
2. defaults to `tracked-since`

If `tracked-since` is unavailable, no secret would be removed.

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

None

##### Unit tests

- `k8s.io/kubernetes/pkg/controller/serviceaccount`: `2022-06-13` - `67.5%`

##### Integration tests

- Previously auto-generated secret-based token that's used within the
  configurable cleanup duration will continue to work.
- Previously auto-generated secret-based token that's used after the
  configurable cleanup duration will be deleted.

##### e2e tests

- Secret-based tokens would not be auto-generated.
- Still able to explicitly request a secret-based token.
- The explicitly requested token would not be deleted.

### Graduation Criteria

#### LegacyServiceAccountTokenNoAutoGeneration

| Alpha | Beta | GA   |
| ----- | ---- | ---- |
| -     | 1.24 | 1.25 |

Since in 1.24, all pods should be admitted in 1.22+ and they should be using
bound tokens. One release ahead to enable this features would help to reduce
legacy tokens for security practices.

#### Beta -> GA Graduation

- [ ] Approved by PRR and scalability
- [ ] Any known bugs fixed
- [ ] Tests passing

#### Alpha -> Beta Graduation

- [ ] Approved by PRR and scalability
- [ ] Any known bugs fixed
- [ ] Tests passing
- [ ] Document and communicate the available actions that consumers of
      auto-generated secret-based tokens should take. (migrate to either use
      tokenrequest or explicitly request secret-based tokens)

#### LegacyServiceAccountTokenTracking

| Alpha | Beta | GA   |
| ----- | ---- | ---- |
| 1.24  | 1.25 | 1.26 |

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

#### LegacyServiceAccountTokenCleanUp

| Alpha | Beta | GA   |
| ----- | ---- | ---- |
| 1.24  | 1.25 | 1.26 |

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

### Upgrade / Downgrade Strategy

The features can be enabled/disabled via the feature gates in upgrade / downgrade.
What would be changed is described in "Feature Enablement and Rollback" section.

### Version Skew Strategy

The only touches control plane, so version skew strategy is not applicable.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: LegacyServiceAccountTokenNoAutoGeneration
  - Components depending on the feature gate: kube-controller-manager
  - Feature gate name: LegacyServiceAccountTokenTracking
  - Components depending on the feature gate: kube-apiserver
  - Feature gate name: LegacyServiceAccountTokenCleanUp:
  - Components depending on the feature gate: kube-controller-manager

###### Does enabling the feature change any default behavior?

- LegacyServiceAccountTokenNoAutoGeneration: no legacy tokens are auto-generated.
- LegacyServiceAccountTokenTracking: legacy tokens would have new annotation and a configmap would be created in kube-system.
- LegacyServiceAccountTokenCleanUp: unused auto-generated legacy tokens will be removed.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

yes for all feature gates.

###### What happens if we reenable the feature if it was previously rolled back?

- LegacyServiceAccountTokenNoAutoGeneration: the same as enable the feature.
  before the reenablement, Token Controller would create tokens for
  serviceaccounts while the feature was off.
- LegacyServiceAccountTokenTracking: during this sequence of operations,
  only the annotation `last-used` is persisted, but there is no impact on the
  functionality of this feature.
- LegacyServiceAccountTokenCleanUp: the same as enable the feature.

###### Are there any tests for feature enablement/disablement?

yes for all feature gates, covered by integration tests.

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

- LegacyServiceAccountTokenNoAutoGeneration: workloads that expect new
  auto-created secrets and extract tokens from them would fail.
- LegacyServiceAccountTokenTracking: no impact.
- LegacyServiceAccountTokenCleanUp: workloads that reads auto-generated
  secrets after those secrets being considered unused by this feature and
  removed.

###### What specific metrics should inform a rollback?

`serviceaccount_legacy_tokens_total`: cumulative stale service account tokens
used.

this metric is only informational and cannot deterministically tell a rollback
is needed. there is no good way for us to detect scrapers of auto-generated
secrets.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

no since there is not much change between a upgrade and upgrade->downgrade->upgrade.
see section `What happens if we reenable the feature if it was previously rolled back`.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

no

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

check if there is a configmap `tracked-since` in namespace `kube-system`.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name: `serviceaccount_legacy_tokens_total`
  - [Optional] Aggregation method:
  - Components exposing the metric: kube-apiserver

LegacyServiceAccountTokenNoAutoGeneration and LegacyServiceAccountTokenCleanUp
might cause few workloads to fail but there is no way for us to inject metric
in workloads to detect this.

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

none. we expect the number recorded in the above metric going down in the long
term.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

none.

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

no.

### Scalability

<!--
For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them.

For beta, this section is required: reviewers must answer these questions.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

###### Will enabling / using this feature result in any new API calls?

up to one additional write request per day could be made to auto-generated secrets still in use.

###### Will enabling / using this feature result in introducing new API types?

no.

###### Will enabling / using this feature result in any new calls to the cloud provider?

no.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

no. instead, use of the feature reduces the number of API objects.

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

- `tracked-since` configmap cannout be created.
- unable to remove unused auto-generated secrets.

###### What are other known failure modes?

- failure to create `tracked-since` config map
  - Detection: check if `tracked-since` exists in `kube-system`
  - Mitigations: there is no impact on existing systems.
  - Diagnostics: check kube-apiserver log.
  - Testing: TBD.

###### What steps should be taken if SLOs are not being met to determine the problem?

n/a.

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
