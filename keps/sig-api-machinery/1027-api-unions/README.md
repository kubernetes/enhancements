# KEP-1027: Union types

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Go tags](#go-tags)
  - [OpenAPI](#openapi)
  - [Discriminator](#discriminator)
  - [Normalizing on updates](#normalizing-on-updates)
    - [&quot;At most one&quot; versus &quot;exactly one&quot;](#at-most-one-versus-exactly-one)
    - [Clearing all the fields](#clearing-all-the-fields)
  - [Backward compatibilities properties](#backward-compatibilities-properties)
  - [Validation](#validation)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Future Work](#future-work)
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

Modern data model definitions like OpenAPI v3 and protobuf (versions 2 and 3)
have a keyword to implement “oneof” or "union". They allow APIs to have a better
semantics, typically as a way to say “only one of the given fields can be
set”. We currently have multiple occurrences of this semantics in kubernetes core
types, at least:
- VolumeSource is a structure that holds the definition of all the possible
  volume types, only one of them must be set, it doesn't have a discriminator.
- DeploymentStrategy is a structure that has a discrminator
  "DeploymentStrategyType" which decides if "RollingUpate" should be set

The problem with the lack of solution is that:
- The API is implicit, and people don't know how to use it
- Clients can't know how to deal with that, especially if they can't parse the
  OpenAPI
- Server can't understand the user intent and normalize the object properly

## Motivation

Currently, changing a value in an oneof type is difficult because the semantics
is implicit, which means that nothing can be built to automatically fix unions,
leading to many bugs and issues:
- https://github.com/kubernetes/kubernetes/issues/35345
- https://github.com/kubernetes/kubernetes/issues/24238
- https://github.com/kubernetes/kubernetes/issues/34292
- https://github.com/kubernetes/kubernetes/issues/6979
- https://github.com/kubernetes/kubernetes/issues/33766
- https://github.com/kubernetes/kubernetes/issues/24198
- https://github.com/kubernetes/kubernetes/issues/60340

And then, for other people:
- https://github.com/rancher/rancher/issues/13584
- https://github.com/helm/charts/pull/12319
- https://github.com/EnMasseProject/enmasse/pull/1974
- https://github.com/helm/charts/pull/11546
- https://github.com/kubernetes/kubernetes/pull/35343

This is replacing a lot of previous work and long-standing effort:
- Initially: https://github.com/kubernetes/community/issues/229, then
- https://github.com/kubernetes/community/pull/278
- https://github.com/kubernetes/community/pull/620
- https://github.com/kubernetes/kubernetes/pull/44597
- https://github.com/kubernetes/kubernetes/pull/50296
- https://github.com/kubernetes/kubernetes/pull/70436

Server-side [apply](http://features.k8s.io/555) is what enables this proposal to
become possible.

### Goals

The goal is to enable a union or "oneof" semantics in Kubernetes types, both for
in-tree types and for CRDs.

### Non-Goals

We're not planning to use this KEP to release the feature, but mostly as a way
to document what we're doing.

## Proposal

In order to support unions in a backward compatible way in kubernetes, we're
proposing the following changes.

Note that this proposes unions to be "at most one of". Whether exactly one is
supported or not should be implemented by the validation logic.

### User Stories (Optional)

<!--
Detail the things that people will be able to do if this KEP is implemented.
Include as much detail as possible so that people can understand the "how" of
the system. The goal here is to make this feel real for users without getting
bogged down.
-->

#### Story 1

#### Story 2

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

## Design Details

### Go tags

We're proposing a new type of tags for go types (in-tree types, and also
kubebuilder types):

- `// +union` before a structure means that the structure is a union. All the
  fields must be optional (beside the discriminator) and will be included as
  members of the union. That structure CAN be embedded in another structure.
- `// +unionDeprecated` before a field means that it is part of the
  union. Multiple fields can have this prefix. These fields MUST BE optional,
  omitempty and pointers. The field is named deprecated because we don't want
  people to embed their unions directly in structures, and only exist because of
  some existing core types (e.g. `Value` and `ValueFrom` in
  [EnvVar](https://github.com/kubernetes/kubernetes/blob/3ebb8ddd8a21b/staging/src/k8s.io/api/core/v1/types.go#L1817-L1836)).
- `// +unionDiscriminator` before a field means that this field is the
  discriminator for the union. Only one field per structure can have this
  prefix. This field HAS TO be a string, and CAN be optional.

Multiple unions can exist per structure, but unions can't span across multiple
go structures (all the fields that are part of a union has to be together in the
same structure), examples of what is allowed:

```
// This will have one embedded union.
type TopLevelUnion struct {
	Name string `json:"name"`

	Union `json:",inline"`
}

// This will generate one union, with two fields and a discriminator.
// +union
type Union struct {
	// +unionDiscriminator
	// +optional
	UnionType string `json:"unionType"`

	// +optional
	FieldA int `json:"fieldA"`
    // +optional
	FieldB int `json:"fieldB"`
}

// This also generates one union, with two fields and on discriminator.
type Union2 struct {
	// +unionDiscriminator
	Type string `json:"type"`
	// +unionDeprecated
    // +optional
	Alpha int `json:"alpha"`
	// +unionDeprecated
    // +optional
	Beta int `json:"beta"`
}

// This has 3 embedded unions:
// One for the fields that are directly embedded, one for Union, and one for Union2.
type InlinedUnion struct {
	Name string `json:"name"`

	// +unionDeprecated
	// +optional
	Field1 *int `json:"field1,omitempty"`
	// +unionDeprecated
	// +optional
	Field2 *int `json:"field2,omitempty"`

	Union  `json:",inline"`
	Union2 `json:",inline"`
}
```

### OpenAPI

OpenAPI v3 already allows a "oneOf" form, which is accepted by CRD validation
(and will continue to be accepted in the future). That oneOf form will be used
for validation, but is "on-top" of this proposal.

A new extension is created in the openapi to describe the behavior:
`x-kubernetes-unions`.

This is a list of unions that are part of this structure/object. Here is what
each list item is made of:
- `discriminator: <discriminator>` is set to the name of the discriminator
  field, if present,
- `fields-to-discriminateBy: {"<fieldName>": "<discriminateName>"}` is a map of
  fields that belong to the union to their discriminated names. The
  discriminatedValue will typically be set to the name of the Go variable.

Conversion between OpenAPI v2 and OpenAPI v3 will preserve these fields.

### Discriminator

For backward compatibility reasons, discriminators should be added to existing
union structures as an optional string. This has a nice property that it's going
to allow conflict detection when the selected union field is changed.

We also do strongly recommend new APIs to be written with a discriminator, and
tools (kube-builder) should probably enforce the presence of a discriminator in
CRDs.

The value of the discriminator is going to be set automatically by the apiserver
when a new field is changed in the union. It will be set to the value of the
`fields-to-discriminateBy` for that specific field.

When the value of the discriminator is explicitly changed by the client, it
will be interpreted as an intention to clear all the other fields. See section
below.

### Normalizing on updates

A "normalization process" will run automatically for all creations and
modifications (either with update or patch). It will happen automatically in order
to clear fields and update discriminators. This process will run for both
core-types and CRDs. It will take place before validation. The sent object
doesn't have to be valid, but fields will be cleared in order to make it valid.
This process will also happen before fields tracking (server-side apply), so
changes in discriminator, even if implicit, will be owned by the client making
the update (and may result in conflicts).

This process works as follows:
- If there is a discriminator, and its value has changed, clear all fields but
  the one specified by the discriminator,
- If there is no discriminator, or if its value hasn't changed,
  - if there is exactly one field, set the discriminator when there is one
    to that value. Otherwise,
  - compare the fields set before and after. If there is exactly one field
    added, set the discriminator (if present) to that value, and remove all
    other fields. if more than one field has been added, leave the process so
    that validation will fail.

#### "At most one" versus "exactly one"

The goal of this proposal is not to change the validation, but to help clients
to clear other fields in the union. Validation should be implemented for in-tree
types as it is today, or through "oneOf" properties in CRDs.

In other word, this is proposing to implement "at most one", and the exactly one
should be provided through another layer of validation (separated).

#### Clearing all the fields

Since the system is trying to do the right thing, it can be hard to "clear
everything". In that case, each API could decide to have their own "Nothing"
value in the discriminator, which will automatically trigger a clearing of all
fields beside "Nothing".

### Backward compatibilities properties

This normalization process has a few nice properties, especially for dumb
clients, when it comes to backward compatibility:
- A dumb client that doesn't know which fields belong to the union can just set
  a new field and get all the others cleared automatically
- A dumb client that doesn't know about the discriminator is going to change a
  field, leave the discriminator as it is, and should still expect the fields to
  be cleared accordingly
- A dumb client that knows about the discriminator can change the discriminator
  without knowing which fields to clear, they will get cleared automatically


### Validation

Objects have to be validated AFTER the normalization process, which is going to
leave multiple fields of the union if it can't normalize. As discussed in
drawbacks below, it can also be useful to validate apply requests before
applying them.

### Test Plan

There are mostly 3 aspects to this plan:
- [x] Core functionality, isolated from all other components: https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/union_test.go
- [x] Functionality as part of server-side apply: How human and robot interactions work: https://github.com/kubernetes-sigs/structured-merge-diff/blob/master/typed/union_test.go
- [x] Integration in kubernetes: https://github.com/kubernetes/kubernetes/pull/77370/files#diff-4ac5831d494b1b52c7c7be81e552a458

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

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

Since this project is a sub-project of Server-side apply, it will be introduced
directly as Beta, and will graduate to GA in a later release, according to the
criteria below.

#### Beta -> GA Graduation

- CRD support has been proven successful
- Core-types all implement the semantics properly
- Stable and bug-free for two releases

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

Documentation is available on [feature gate lifecycle] and expectations, as
well as the [existing list] of feature gates.

[feature gate lifecycle]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[existing list]: https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/
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

Feature gates are typically disabled by setting the flag to `false` and
restarting the component. No other changes should be necessary to disable the
feature.

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

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

## Future Work

Since the proposal only normalizes the object after the patch has been applied,
it is hard to fail if the patch is invalid. There are scenarios where the patch
is invalid but it results in an unpredictable object. For example, if a patch
sends both a discriminator and a field that is not the discriminated field, it
will either clear the value sent if the discriminator changes, or it will change
the value of the sent discriminator.

Validating patches is not a problem that we want to tackle now, but we can
validate "Apply" objects to make sure that they do not define such broken
semantics.

## Implementation History

Here are the major milestones for this KEP:
- Initial discussion happened a year before the creation of this kep:
  https://docs.google.com/document/d/1lrV-P25ZTWukixE9ZWyvchfFR0NE2eCHlObiCUgNQGQ/edit#heading=h.w5eqnf1f76x5
- Points made in the initial document have been improved and put into this kep,
  which has approved by sig-api-machinery tech-leads
- KEP has been implemented:
  - logic mostly lives in sigs.k8s.io/structured-merge-diff
  - conversion between schema and openapi definition are in k8s.io/kube-openapi
  - core types have been modified in k8s.io/kubernetes
- Feature is ready to be released in Beta in kubernetes 1.15

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->
* Stutter with discriminator
* Inconsistent for existing types here

## Alternatives
* Non-Discrminated

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->
