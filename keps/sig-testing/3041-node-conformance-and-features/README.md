# KEP-3041: NodeConformance, NodeFeature, and Feature Gate labels cleanup

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Clarify the semantic of a <code>[Feature:]</code> label](#clarify-the-semantic-of-a--label)
  - [Feature label clean up](#feature-label-clean-up)
  - [Clean up labels that means whether special environment is needed](#clean-up-labels-that-means-whether-special-environment-is-needed)
    - [NodeSpecialFeature](#nodespecialfeature)
    - [NodeFeature](#nodefeature)
  - [NodeConformance](#nodeconformance)
  - [See also](#see-also)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes](#notes)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Existing test definitions](#existing-test-definitions)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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

The document started as an analysis of whether `NodeFeature` label needs to be
renamed to simply `Feature` as it carries an identical semantics [kubernetes/kubernetes#94289](https://github.com/kubernetes/kubernetes/issues/94289),
but looking into it, scope needed to be extended into other labels currently
being used, see also [this discussion](https://groups.google.com/g/kubernetes-sig-testing/c/8NVCt9UoV_Q/m/Vqz3_TidAwAJ?utm_medium=email&utm_source=footer).

Document proposes changes into `Feature` label description, and additional labels
that will make it clearer for contributors how to apply these labels.

## Motivation

Tests have various clarifiers - they may be testing the feature that is currently
in development and depends on feature gate, test may require a special environment,
or test may only work with specific hardware. Today a few labels are used to represent
the combination of these "dimensions", in most cases a single label `[Feature:]` is used.

The universal nature of a single `[Feature:]` label makes it hard to apply it
consistently and query tests.

There are a few specific problems we saw recently:

- There is no way to disable tests for a specific feature gate
  (example where it’s needed: https://github.com/kubernetes/kubernetes/issues/99854).
  It is a "feature" guarded by feature gate, but not marked as "feature".
- There are no tests validating that k8s works with ALL beta feature gates disabled.
  Creating such test would be hard with today's test labeling.
- Some tests are degraded in how `[Feature]` and `[NodeFeature]` labels are applied
  (for example: https://github.com/kubernetes/kubernetes/pull/105921).
- labels that we apply heavily depend on the environment tests are running on.
  For example, https://github.com/kubernetes/kubernetes/pull/104803 is not marked
  as `Feature`, even though it depends on the environment - if `test-handler`
  is not installed, test will not succeed. Similar thing to AppArmor tests.

### Goals

Split the special environment and feature development labels:

- Clarify the semantic of a `[Feature:]` label.
- Introduce the `[FeatureGate:]` label with the stage (`[Alpha]`, `[Beta]`, etc.)
  addition, replace tags like `[NodeAlphaFeature]`.

Clean up labels that means whether special environment is needed:

- Eliminate the `[NodeFeature:]` label by renaming to `[Feature:]` when applicable.
- Document the meaning of `[NodeConformance]` label.
- Remove the `[NodeSpecialFeature:]` label.
- Introduce the `[Environment:Foo]` label to indicate the special environment
  that is not a standard OSS test infrastructure machines needed to run the feature.

### Non-Goals

- As `[NodeConformance]` may be a confusing term and may be confused with the
  k8s Conformance tests, renaming of it is not in a scope of this KEP.
- Convert textual tags into the ginkgo v2 tags. This is optional, but not
  required and can be handled outside of this KEP.

## Proposal

### Clarify the semantic of a `[Feature:]` label

The current [feature definition is](https://github.com/kubernetes/community/blob/32a1c14d04ff78684d78b827ac7c49f70352d509/contributors/devel/sig-testing/e2e-tests.md#kinds-of-tests):

> * `[Feature:.+]`: If a test has non-default requirements to run or targets
>   some non-core functionality, and thus should not be run as part of the
>   standard suite, it receives a `[Feature:.+]` label, e.g.
>   `[Feature:Performance]` or `[Feature:Ingress]`. `[Feature:.+]` tests are not
>   run in our core suites, instead running in custom suites. If a feature is
>   experimental or alpha and is not enabled by default due to being incomplete
>   or potentially subject to breaking changes, it does not block PR merges, and
>   thus should run in some separate test suites owned by the feature owner(s)
>   (see Continuous Integration below).

Feature will indicate that things may NOT work on all k8s distros and/or node
capabilities. This flag will not indicate the Feature Gate state.

Proposed change:

> * `[Feature:.+]`: If a test validates functionality that may only work
>   outside the minimal conformant installation of Kubernetes, e.g. when
>   specific node capabilities are enabled, having a certain addons available,
>   like loadbalancer integration, or when test depends on the
>   functionality of underlying components like container runtime, and thus may
>   need to be skipped on certain environments, it receives a `[Feature:.+]`
>   label, e.g. `[Feature:AppArmor]`. This label can also be applied to signify that
>   test validates non-core functionality, like `[Feature:Ingress]`.
>   `[Feature:.+]` tests are not run in a core suites, but typically can run together
>   on the standard environment as many capabilities and functionality of underlying
>   components are pre-enabled on the standard environment. Use `[Environment:Foo]` to
>   signify the specific environment configuration needed to run the test.
>   For example if test requires GPU to be configured, but tests the non-optional
>   feature like a device plugin.
>   `[Feature:+]` label should not be confused with `[FeatureGate:]` label.

And another two:

> * `[FeatureGate:.+]`: If a test only works when the certain feature gate is
>   enabled it receives a `[FeatureGate:.+]` label. `[FeatureGate:.+]` tests
>   must also be marked with the status of this feature gate: `[Alpha]`,
>   `[Beta]`, `[Stable]`, `[Deprecated]`. This label helps to skip tests that
>   should not work on specific k8s distributive that has a certain feature gate
>   disabled. This label has to be removed when feature gate value is "locked" or
>   removed.

> * `[Environment:.+]`: If test requires non-standard environment
>   (different from standard OSS test machines) to run it
>   receives the `[Environment:.+]` tag. Typically only tests with the matching
>   `[Environment:.+]` tags can run together. Examples may be GPU needs to be
>   provisioned, Memory Swap enabled, high-memory or high-CPU machines are needed for
>   Performance environment, etc.

### Feature label clean up

It is already true today that only a handful of tests marked as `Feature` don't
have `Serial` or `Disruptive` labels as well. Out of all tests that has `Feature`
label without `Serial` or `Disruptive`, most of them just degraded,
and don’t actually need the `Feature` label any longer. Going forward, features
that require a special environment outside of predefined `LinuxOnly`, `Serial`, etc.
will need to define their "custom" labels and be commented with the description
of the special environment needed.

### Clean up labels that means whether special environment is needed

#### NodeSpecialFeature

The label `NodeSpecialFeature:` was introduced in [this document](https://docs.google.com/document/d/1BdNVUGtYO6NDx10x_fueRh_DLT-SVdlPC_SsXjYCHOE/edit?usp=sharing),
but was never consistently used. Today we only have a couple uses of it that
might be cleaned up in favor of using `Feature:` tag when applicable.

#### NodeFeature

`NodeFeatures` label was [introduced](https://docs.google.com/document/d/1BdNVUGtYO6NDx10x_fueRh_DLT-SVdlPC_SsXjYCHOE/edit?usp=sharing)
to indicate that the feature may not work the same way on different container
runtimes or environments. It is NOT used today as a direct analog of `Features`
label, as `Feature` label indicates that the special environment configuration
is needed to run tests in the “standard” CI.
See the previous section and definition [here](https://github.com/kubernetes/community/blob/32a1c14d04ff78684d78b827ac7c49f70352d509/contributors/devel/sig-testing/e2e-tests.md#kinds-of-tests).

The use of these labels in tests today is indicative of the labels' different
meaning. Looking at tests configuration, wildcard `[NodeFeature:*`
label is always used as focus and individual `NodeFeatures` are used in skip.
It is opposite for the `Feature` label. The wildcard `[Feature:*` is always
used to skip tests, while individual tests are present in focus.

The reason for this difference is that runtimes that are being tested support
all the NodeFeatures today and there is no need to list all NodeFeatures
individually in test definitions. If we will have more fragmented support of
`NodeFeatures` in runtimes being tested in CI, labels will have exactly the
same semantics.

Also both labels, `NodeFeature` and `Feature` degraded over time - they weren’t
applied using this semantics.

This KEP proposes to adjust the definition of the `Feature:` label which makes
definitions compatible. The proposal is to unify `NodeFeatures` and `Features`
and start relying on alternative labels for filtering in environments.

- Rename NodeFeature to Feature. https://testgrid.k8s.io/sig-node-containerd#node-e2e-features will execute tests for all Features, excluding everything that needs a special environment, see labels above.
- Document the meaning of NodeConformance. NodeConformance will indicate ALL NODE tests that are testing enabled out of the box functionality that is not runtime or k8s distro specific. These tests may still require an additional environment set up. NodeConformance tests may have all labels specifying its environment requirements, like Slow, Disruptive, Special, etc.
- Introduce a new periodic job that runs all NodeConformance tests with all FeatureGates turned off.

### NodeConformance

NodeConformance label represents tests that have to be working on all environments
and k8s distros. The name is similar to the `[Conformance]` cluster tests that
are part of the Kubernetes Conformance Tests, which serves as the base for the
Kubernetes certification program. The similarity in the name may be confusing,
but this confusion will not be addressed in this KEP.

One difference proposed in this document is to allow NodeConformance to be
applied to `[Beta]` Features. Since beta features are enabled out of the box and
in most deployments all Beta features stay enabled, applying Feature label to
these tests may be misleading. This difference is intended, as we want to make
sure Beta features as enabled out of the box are tested on PR validation and
across different environments. This will give a better signal for GA-ing the
feature.

Proposed definition of `NodeConformance`:

* `[NodeConformance]`: Node-level tests that validating behavior that doesn't
  depend on specific Node capabilities being present, hardware, or feature set
  of a dependency (like a container runtime), must be labeled as `[NodeConformance]`.
  For the ease of test querying, each node-level test that is not testing alpha feature
  (marked as `[FeatureGate:Foo][Alpha]`) is supposed to be either `NodeConformance`
  or `Feature`.

### See also

- https://github.com/kubernetes/kubernetes/issues/59001
- https://docs.google.com/document/d/1BdNVUGtYO6NDx10x_fueRh_DLT-SVdlPC_SsXjYCHOE/edit?usp=sharing
- https://github.com/kubernetes/community/blob/32a1c14d04ff78684d78b827ac7c49f70352d509/contributors/devel/sig-testing/e2e-tests.md#kinds-of-tests
- https://groups.google.com/g/kubernetes-sig-testing/c/8NVCt9UoV_Q/m/Vqz3_TidAwAJ?utm_medium=email&utm_source=footer

### User Stories (Optional)

#### Story 1

1. New feature gate is introduced as Alpha.
1. Tests for this functionality are marked as `[FeatureGate:Foo][Alpha]`.
1. New test grid tab is added to run these tests while enabling the feature gate explicitly.
1. Feature gate is promoted to Beta and enabled by default.
1. Tests for this functionality are marked as `[FeatureGate:Foo][Beta]`.
1. Depending on whether test targets all environments or specific ones,
   `NodeConformance` or `Feature:` labels are added. Note, the `Feature:` label
   may (but not necessarily) match the `FeatureGate:` name.
1. Test infra runs all these tests as part of Features or NodeConformance runs
   to ensure that default installation of k8s has all the features working.
1. Test infra runs all tests with every feature gate disabled, catching potential
   GA features dependencies on the new functionality.
1. Feature gate is promoted to GA. Feature gate is locked to the value and
   cannot be disabled.
1. Tests for this functionality are dropping the labels `[FeatureGate:Foo][Beta]`,
   while keeping either `NodeConformance` or `Feature:` label.

#### Story 2

Decision making tree for the test labels:

- Is test only works on Linux? Apply `[LinuxOnly]`
- Is test validate functionality that is controlled by a Feature Gate? Apply `[FeatureGate:Foo][Alpha|Beta|Deprecated]`
- Is test validate Core API that is enabled by default, GA, and works on any environment? Apply `[Conformance]`. See more at https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md
- Is test validate node-specific functionality that is enabled by default and works on "any" node? Apply `[NodeConformance]`
- Is test only works when underlying container runtime has a specific feature enabled or specific node configuration is set? Apply `[Feature:Foo]` to describe this feature or configuration.
- Can test only run on the "default" test infra node? If not, apply `[Environment:Foo]` to describe the specific environment that needs to be pre-configured.

### Notes

PRR and test plan sections are not applicable to this KEP.

See https://kubernetes.slack.com/archives/CPNHUMN74/p1665177817264029

<!--

STANDARD TEMPLATE AFTER THIS LINE, NO MEANINGFUL CONTENT

-->


### Notes/Constraints/Caveats (Optional)

<!--
What are the caveats to the proposal?
What are some important details that didn't come across above?
Go in to as much detail as necessary here.
This might be a good place to talk about core concepts and how they relate.
-->

### Risks and Mitigations

<!--
What are the risks of this proposal, and how do we mitigate? Think broadly.
For example, consider both security and how this will impact the larger
Kubernetes ecosystem.

How will security be reviewed, and by whom?

How will UX be reviewed, and by whom?

Consider including folks who also work outside the SIG or subproject.
-->

#### Existing test definitions

Existing test definitions may be affected as they may start running different set of tests.
Skew tests may be affected especially as the labels are being modified.

Mitigation:

We will take the iterative approach.

- Add new labels. `FeatureGate` and `Environment` bring additive value and should not affect existing tests
- Features which should be `NodeConformance` or `Conformance`. This is typically an easy transition as
  `NodeConformance` and `Conformance` tests are run more often than `Feature` and it will unlikely lead to
  less tests are being run in general.
- We don't expect many `NodeConformance` and `Conformance` tests to be reverted to `Feature`. Thus it should not be an issue.

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

Consider the following in developing a test plan for this enhancement:
- Will there be e2e and integration tests, in addition to unit tests?
- How will it be tested in isolation vs with other components?

No need to outline all of the test cases, just the general strategy. Anything
that would count as tricky in the implementation, and anything particularly
challenging to test, should be called out.

All code is expected to have adequate tests (eventually with coverage
expectations). Please adhere to the [Kubernetes testing guidelines][testing-guidelines]
when drafting this test plan.

[testing-guidelines]: https://git.k8s.io/community/contributors/devel/sig-testing/testing.md
-->

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, or as something else. The KEP
should keep this high-level with a focus on what signals will be looked at to
determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Complete features A, B, C
- Additional tests are in Testgrid and linked in KEP

#### GA

- N examples of real-world usage
- N installs
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- Allowing time for feedback

**Note:** Generally we also wait at least two releases between beta and
GA/stable, because there's no opportunity for user feedback, or even bug reports,
in back-to-back releases.

**For non-optional features moving to GA, the graduation criteria must include
[conformance tests].**

[conformance tests]: https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md

#### Deprecation

- Announce deprecation and support policy of the existing flag
- Two versions passed since introducing the functionality that deprecates the flag (to address version skew)
- Address feedback on usage/changed behavior, provided on GitHub issues
- Deprecate the flag
-->

### Upgrade / Downgrade Strategy

<!--
If applicable, how will the component be upgraded and downgraded? Make sure
this is in the test plan.

Consider the following in developing an upgrade/downgrade strategy for this
enhancement:
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to maintain previous behavior?
- What changes (in invocations, configurations, API use, etc.) is an existing
  cluster required to make on upgrade, in order to make use of the enhancement?
-->

### Version Skew Strategy

<!--
If applicable, how will the component handle version skew with other
components? What are the guarantees? Make sure this is in the test plan.

Consider the following in developing a version skew strategy for this
enhancement:
- Does this enhancement involve coordinating behavior in the control plane and
  in the kubelet? How does an n-2 kubelet without this feature available behave
  when this feature is used?
- Will any other components on the node change? For example, changes to CSI,
  CRI or CNI may require updating that component before the kubelet.
-->

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

<!--
Pick one of these and delete the rest.
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

<!--
Any change of default behavior may be surprising to users or break existing
automations, so be extremely careful here.
-->

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

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

<!--
This section must be completed when targeting beta to a release.
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
