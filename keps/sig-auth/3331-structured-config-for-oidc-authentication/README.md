<!--

**Note:** When your KEP is complete, all of these comment blocks should be removed.

To get started with this template:

- [ ] **Pick a hosting SIG.**
  Make sure that the problem space is something the SIG is interested in taking
  up. KEPs should not be checked in without a sponsoring SIG.
- [ ] **Create an issue in kubernetes/enhancements**
  When filing an enhancement tracking issue, please make sure to complete all
  fields in that template. One of the fields asks for a link to the KEP. You
  can leave that blank until this KEP is filed, and then go back to the
  enhancement and add the link.
- [ ] **Make a copy of this template directory.**
  Copy this template into the owning SIG's directory and name it
  `NNNN-short-descriptive-title`, where `NNNN` is the issue number (with no
  leading-zero padding) assigned to your enhancement above.
- [ ] **Fill out as much of the kep.yaml file as you can.**
  At minimum, you should fill in the "Title", "Authors", "Owning-sig",
  "Status", and date-related fields.
- [ ] **Fill out this file as best you can.**
  At minimum, you should fill in the "Summary" and "Motivation" sections.
  These should be easy if you've preflighted the idea of the KEP with the
  appropriate SIG(s).
- [ ] **Create a PR for this KEP.**
  Assign it to people in the SIG who are sponsoring this process.
- [ ] **Merge early and iterate.**
  Avoid getting hung up on specific details and instead aim to get the goals of
  the KEP clarified and merged quickly. The best way to do this is to just
  start with the high-level sections and fill out details incrementally in
  subsequent PRs.

Just because a KEP is merged does not mean it is complete or approved. Any KEP
marked as `provisional` is a working document and subject to change. You can
denote sections that are under active debate as follows:

```
<<[UNRESOLVED optional short context or usernames ]>>
Stuff that is being argued.
<<[/UNRESOLVED]>>
```

When editing KEPS, aim for tightly-scoped, single-topic PRs to keep discussions
focused. If you disagree with what is already in a document, open a new PR
with suggested changes.

One KEP corresponds to one "feature" or "enhancement" for its whole lifecycle.
You do not need a new KEP to move from beta to GA, for example. If
new details emerge that belong in the KEP, edit the KEP. Once a feature has become
"implemented", major changes should get new KEPs.

The canonical place for the latest set of instructions (and the likely source
of this file) is [here](/keps/NNNN-kep-template/README.md).

**Note:** Any PRs to move a KEP to `implementable`, or significant changes once
it is marked `implementable`, must be approved by each of the KEP approvers.
If none of those approvers are still appropriate, then changes to that list
should be approved by the remaining approvers and/or the owning SIG (or
SIG Architecture for cross-cutting KEPs).
-->
# KEP-3331: Structured config for OIDC authentication

<!-- toc -->
  - [Release Signoff Checklist](#release-signoff-checklist)
  - [Summary](#summary)
- [Motivation](#motivation)
    - [Goals](#goals)
    - [Non-Goals](#non-goals)
  - [Proposal](#proposal)
    - [Risks and Mitigations](#risks-and-mitigations)
  - [Design Details](#design-details)
    - [Configuration file](#configuration-file)
    - [More about CEL](#more-about-cel)
    - [Flags](#flags)
    - [Test Plan](#test-plan)
        - [Prerequisite testing updates](#prerequisite-testing-updates)
        - [Unit tests](#unit-tests)
        - [Integration tests](#integration-tests)
        - [e2e tests](#e2e-tests)
    - [Graduation Criteria](#graduation-criteria)
      - [Alpha](#alpha)
      - [Beta](#beta)
      - [GA](#ga)
      - [Deprecation](#deprecation)
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

This enhancement proposal covers implementing the structured configuration for the OIDC authenticator.
OIDC authentication is essential part of Kubernetes, yet it has limitations in its current state. 
Bellow we will discuss that limitation and propose solutions.

# Motivation

Structured config for OIDC authentication: noted in various contexts over the past few years. We want to migrate
away from a flag-based config that is growing without bounds to a proper versioned config format. This would allow us to
better support various features that have been requested.

### Goals

There are features users want to tune. We need to provide customization of the following:

- Claims validation rules: current OIDC provider supports only audience claim validation and only by exact values
- Claim mappings: it is only possible to pick a single value from a single claim and prefix groups
- Use more than one OIDC provider: the only option, for now, is to use [Dex](https://dexidp.io/) (or similar software)
  as a lightweight OIDC proxy to connect many providers to Kubernetes
- Change authenticator settings without restarting kube-apiserver

### Non-Goals

- Monitoring

## Proposal

1. Add new `authentication` API object to parse a structured config file `OIDCConfiguration`.
2. Add a single flag for kube-apiserver to point to the structured config file, automatically reload it on changes.
3. Use an expression language to let users write their own logic for mappings and validation rules 
  (expressions should be simple for common cases, yet powerful to cover most user stories).

### Risks and Mitigations

Since this is a new optional feature, no migration is required. Before the Stable release of the feature,
we should provide examples of migrating from a flag-based config to a new structured config.

## Design Details

### Configuration file

The main part of this proposal is a configuration file. It contains an array of providers:

```go
type OIDCConfiguration struct {
    metav1.TypeMeta
    // Providers is a list of OIDC providers to authenticate Kubernetes users.
    Providers []Provider
}
```

Each provider has several properties that will be described in detail below.

```go
type Provider struct {
    // Issuer is a basic OIDC provider connection options.
    Issuer Issuer
    // ClaimValidationRules are rules that are applied to validate ID token claims to authorize users.
    // +optional
    ClaimValidationRules []ClaimValidationRule
    // ClaimMappings points claims of an ID token to be treated as user attributes.
    // All mappings are logical expressions that is written in CEL https://github.com/google/cel-go.
    ClaimMappings UserAttributes
    // ClaimsFilter allows unmarshalling only required claims which positively affects performance.
    // +optional
    ClaimsFilter []string
}
```

1. `Issuer` - is a sections for external provider specific settings, e.g., OIDC discovery URL. 
  There is also default validation settings (ClientID, SkipOIDCValidations) that's available out of the box for flag-based 
  OIDC provider, but for structured config they can be disabled.

    ```go
    type Issuer struct {
        // URL points to the issuer URL in a format schema://url/path.
        URL string
        // CertificateAuthorityData contains PEM-encoded certificate authority certificates. Overrides CertificateAuthority
        CertificateAuthorityData []byte
        // ClientID the JWT must be issued for, the "sub" field. This plugin only trusts a single
        // client to ensure the plugin can be used with public providers.
        // Do not affect anything with the SkipOIDCValidations option enabled.
        // +optional
        ClientID string
        // SkipOIDCValidations is a flag to turn off issuer validation, client id validation.
        // OIDC related checks.
        // +optional
        SkipOIDCValidations bool
   }
   ```

2. `ClaimValidationRules` - additional rules for authorization.
    ```go
    type ClaimValidationRule struct {
        // Rule is a logical expression that is written in CEL https://github.com/google/cel-go.
        Rule string
        // Message customize returning message for validation error of the particular rule.
        // +optional
        Message string
    }
    ```

    For validation expressions, the CEL is used. They are similar to validations functions for [Custom Resources](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#resource-use-by-validation-functions).
    `Rule` expression should always evaluate a boolean. Token `claims` are passed to CEL expressions as a dynamic map `decls.NewMapType(decls.String, decls.Dyn)`.

    You can find a snippet of validation rules below:

    ```yaml
    claimValidationRules:
    
    - rule: 'claims.aud == "charmander" || claims.aud == "bulbasaur"'
      message: clients other than charmander or bulbasaur are not allowed
    
    - rule: 'claims.roles.exists(r, r == "kubernetes-user")'
      message: only kubernetes-user group members can access the cluster
    ```

3. `ClaimMappings` - rules to map claims from a token to Kubernetes user attributes.
    ```go
    type UserAttributes struct {
        // Username represents an option for the username attribute.
        Username string
        // Groups represents an option for the groups attribute. 
        // +optional
        Groups string
        // UID represents an option for the uid attribute.
        // +optional
        UID string
        // Extra represents an option for the extra attribute.
        // +optional
        Extra []ExtraMapping
    }
   
    type ExtraMapping struct {
        // Key is a CEL expression to extract extra attribute key.
        Key string
        // Expression is a CEL expression to extract extra attribute value.
        Expression string
    }
    ```

    Every field of structures above is a CEL expression. For username and uid, the expression should evaluate a string.
    For groups, it should evaluate to an array. A map of extra attributes is composed of key/value pairs, each with their
    own CEL expressions, both should evaluate to a string.

    The example of mapping user attributes can be found below:

    ```yaml
    claimMappings:
      username: 'claims.username + ":external-user"'
      groups: 'claims.roles.split(",")'
      uid: 'claims.sub'
      extra:
      - key: '"client_name"'
        expression: 'claims.aud'
    ```
   
    For the token with the following claims
    ```json
    {
      "sub": "119abc",
      "aud": "kubernetes",
      "username": "jane_doe",
      "roles": "admin,user",
      ...
    }
    ```
    our authenticator will extract the following user attributes:
    ```yaml
    username: jane_doe:external-user
    groups: ["admin", "user"]
    uid: "119abc"
    extra:
      client_name: kubernetes
    ```
4. `ClaimsFilter` - list of claim names that should be passed to CEL expressions. The assumption is that administrators
   know the structure of the token and the exact claims they will use in CEL expressions.
   This option helps to reduce system load and operate only with required claims.

### More about CEL

* CEL runtime should be compiled only once if structured OIDC config option is enabled
* To make working with strings more convinient, `strings` and `encoding` [CEL extensions](https://github.com/google/cel-go/tree/v0.9.0/ext) should be enabled, 
e.g, to be able to split a string with comma separated fields and use them as a single array
* Benchmarks are required to see how different CEL expressions affects authentication time
* CEL expressions are called on each request. We should properly investigate the influence of these calls on the system
  latency and, if necessary, prove caching or other mechanisms to improve performance.

### Flags

The only flag requires to enable the feature is the `--oidc-configuration-path` flag. It points to the configuration file.
On startup, kube-apiserver enables the file watcher for the configuration file and reacts to any file change.

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

TBA

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

<!--
In principle every added code should have complete unit test coverage, so providing
the exact set of tests will not bring additional value.
However, if complete unit test coverage is not possible, explain the reason of it
together with explanation why this is acceptable.
-->

<!--
Additionally, for Alpha try to enumerate the core package you will be touching
to implement this enhancement and provide the current unit coverage for those
in the form of:
- <package>: <date> - <current test coverage>
The data can be easily read from:
https://testgrid.k8s.io/sig-testing-canaries#ci-kubernetes-coverage-unit

This can inform certain test coverage improvements that we want to do before
extending the production code to implement this enhancement.
-->

- `<package>`: `<date>` - `<test coverage>`

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- <test>: <link to test coverage>

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete benchmarks

#### GA

- Add metrics
- Add a full documentation with examples for the most popular providers, e.g., Okta, Dex, Auth0
- Migration guide
- Deprecation warnings for non-stuctured OIDC provider configuration

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

kube-apiserver `--oidc-*` flags require deprecation warnings on the stable release of the feature.
It is possible to react only to the `--oidc-issuer-url` flag because other flags cannot be enabled separately from this one.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Feature gate
  - Feature gate name: `StructuredOIDCConfiguration`
  - Components depending on the feature gate:
    - kube-apiserver

```go
FeatureSpec{
	Default: false,
	LockToDefault: false,
	PreRelease: featuregate.Alpha,
}
```

###### Does enabling the feature change any default behavior?

No.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes, but disable means also deleting the flag from the kube-apiserver manifest.

###### What happens if we reenable the feature if it was previously rolled back?

No impact.

###### Are there any tests for feature enablement/disablement?

Not required.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It cannot fail.

###### What specific metrics should inform a rollback?

No specific metrics are required.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Yes. It works.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes, the `--oidc-configuration-path` flag should be removed from the kube-apiserver manifest.

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->

TBA

###### How can an operator determine if the feature is in use by workloads?

* There will be a corresponding message in kube-apiserver logs.
* By checking the kube-apiserver flags.

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

The feature should work 99.9% of the time.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

TBA.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

No.

### Scalability

###### Will enabling / using this feature result in any new API calls?

Yes, the authenticator requests an OIDC provider to get public keys and validate the token.

###### Will enabling / using this feature result in introducing new API types?

Yes. Group `authenticator.apiserver.k8s.io`, object `OIDCConfiguration`.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

It can affect authentication time, but the actual latency depends on a provider configuration.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

TBA.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature is a part of authentication flow. It does not rely on etcd, but stricktly connected to the kube-apiserver.

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

The same failure modes and diagnostics as for the non-structured OIDC provider are applicable here.

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

TBA.

## Drawbacks

Nothing can hold us. Everyone agrees that we have to implement this feature to fulfill user expectations.

## Alternatives

Invest more into external software like Dex and officially make it the OIDC provider socket.
Do not add any more OIDC provider customization to Kubernetes.
Instead, add more guides and docs about customizing Kubernetes authentication with external software.
