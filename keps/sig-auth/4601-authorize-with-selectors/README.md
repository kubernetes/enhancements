# KEP-4601: Authorize with Selectors

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Authorization Attributes changes](#authorization-attributes-changes)
    - [Future-proofing your authorization webhook for future verbs](#future-proofing-your-authorization-webhook-for-future-verbs)
  - [SubjectAccessReview Changes](#subjectaccessreview-changes)
  - [Node Authorizer Changes](#node-authorizer-changes)
  - [CEL Authorizer Changes](#cel-authorizer-changes)
  - [User Stories (Optional)](#user-stories-optional)
    - [As a SAR client, I want to check a request with a field or label selector](#as-a-sar-client-i-want-to-check-a-request-with-a-field-or-label-selector)
    - [As an authorization webhook author, I want to easily consume the field and label selectors](#as-an-authorization-webhook-author-i-want-to-easily-consume-the-field-and-label-selectors)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [client provides field or label selector to kube-apiserver that does not parse](#client-provides-field-or-label-selector-to-kube-apiserver-that-does-not-parse)
    - [client provides field or label selector to kube-apiserver with improper verb](#client-provides-field-or-label-selector-to-kube-apiserver-with-improper-verb)
    - [client provides SAR where field rawSelector does not match field requirements.](#client-provides-sar-where-field-rawselector-does-not-match-field-requirements)
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
    - [New kube-apiserver, old webhook authorizer](#new-kube-apiserver-old-webhook-authorizer)
    - [Old kube-apiserver, new in-cluster authorizer (or any SAR client)](#old-kube-apiserver-new-in-cluster-authorizer-or-any-sar-client)
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

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [ ] e2e Tests for all Beta API Operations (endpoints)
  - [ ] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
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

The authorization attributes will be extended to include field selectors and label selectors from
List, Watch, and DeleteCollection.
This will allow authorizers to use these selectors when making an authorization decision.

## Motivation

Security for per-node workloads could be improved by exposing field and label selectors to authorizers.
Adding them as authorization attributes allows the development of new kinds of authorizers that
leverage this information to provide security.
In particular, it enables out-of-tree authorizers to experiment with ways to express restrictions based on field and label selectors.

### Goals

* Add field and label selectors to authorization attributes for List, Watch, and DeleteCollection verbs.
* Add field and label selectors to webhook authorization types.
* Add field and label selectors to SelfSubjectAccessReview (SSAR), SubjectAccessReview (SAR), and LocalSubjectAccessReview.
* Update node authorizer to restrict on nodeName field selector.
* Add field and label selectors to CEL authorizer implementation. 

### Non-Goals

* Create a generic in-tree authorizer that manages field or label selectors.
* Expand the audit surface area, since requestURI is already included
* Expand the admission surface area (admission.Attributes, AdmissionReview, available to admission) 
  since admission verbs don't support field/label selectors


## Proposal

List, Watch, and DeleteCollection requests directly have field and label selector options.
A single-item List or Watch request is still a list as normal (including selectors), but also includes a name.

### Authorization Attributes changes

The authorization attributes have easy access to the query parameter field and label selectors.
To avoid confusion, field and label selectors will not be included in authorization attributes for kube-apiserver requests
with verbs where the field selector has no semantic meaning.
In practice this means that (for now), only List, Watch, and DeleteCollection have field and label selectors.

SubjectAccessReviews submitted to the kube-apiserver with verbs that do not honor the selectors will NOT modify the field and label selector attributes.
The client is trusted to be sending only combinations that will be honored.

Any authorizer that gets an error from `GetFieldSelector` or `GetLabelSelector` may attempt to authorize without
field or label selectors since that will authorize using a wider permission (field and label selectors can only reduce access).

```go
type Attributes interface {
  // GetFieldSelector is lazy, thread-safe, and stores the parsed result and error.
  // It can return an error if the field selector cannot be parsed.
  // Remember that field selector formats vary based on the version of the API being used!
  GetFieldSelector() (fields.Requirements, error)
  
  // GetLabelSelector is lazy, thread-safe, and stores the parsed result and error.
  // It can return an error if the field selector cannot be parsed.
  GetLabelSelector() (labels.Requirements, error)
```

Webhook authors: remember that the list of verbs accepting field and label selectors may change over time.
If the kube-apiserver sends the FieldSelector or LabelSelector to a webhook, the kube-apiserver intends to honor the selector attributes.

#### Future-proofing your authorization webhook for future verbs

As of 1.31, the only verbs with field and label selectors are List, Watch, and DeleteCollection.
In the future, the kube-apiserver may add field and label selectors to Get, Create, Update, Patch, and Delete.
* For Get, this means the field and label selector of the retrieved object must match.
* For Create, this means that the resource after all mutation is complete (finalObject) must match the field and label selector. 
* For Update/Patch, this means that the finalNewObject and oldObject must match the field and label selector.
* For Delete, this means that the oldObject must match the field and label selector.
* For subresources, if the storage layer cannot verify the parent object matches the selector (both old and new), the request must be rejected.
 
We do not allow field and label selectors for Get, because if a client is specifying a selector, they can add a `.metadata.name`
field selector and use a List to get equivalent functionality.

### SubjectAccessReview Changes

SubjectAccessReview is used for two purposes:
1. Authorization webhook calls from the kube-apiserver to a webhook.
   This usage likely benefits from a serialization with `[]Requirement`.
2. Authorization checks from a client (often a server process using in-cluster authorization like kube-rbac-proxy)
   This usage likely benefits from a serialization that matches the query parameter.

Their needs are best met with two different serialization (see user stories)

```go

type SubjectAccessReviewSpec struct {
	ResourceAttributes *ResourceAttributes
}

type ResourceAttributes struct {
	FieldSelector *FieldSelectorAttributes

	LabelSelector *LabelSelectorAttributes
}

// FieldSelectorAttributes indicates a field limited access.
// For webhooks:
// The kube-apiserver will never send a request with rawSelector set, but we cannot control what other clients directly send.
// * If rawSelector is empty and requirements are empty, the request is not limited.
// * If rawSelector is present and requirements are empty, the request is not limited.
// * If rawSelector is empty and requirements are present, the requirements should be honored
// * If rawSelector is present and requirements are present, the request is invalid.
// Webhook authors are encouraged to
// * ensure rawSelector and requirements are not both set
// * consider the requirements field if set
// * not try to parse or consider the rawSelector field if set.
//   This is to avoid another CVE-2022-2880 (i.e. getting different systems to agree on how exactly to parse
//   a query is not something we want), see https://www.oxeye.io/resources/golang-parameter-smuggling-attack for more details.
// For the kube-apiserver:
// * If rawSelector is empty and requirements are empty, the request is not limited.
// * If rawSelector is present and requirements are empty, the rawSelector will be parsed and limited if the parsing succeeds.
// * If rawSelector is empty and requirements are present, the requirements should be honored
// * If rawSelector is present and requirements are present, the request is invalid.
type FieldSelectorAttributes struct {
	// rawSelector is the serialization of a field selector that would be included in a query parameter.
	// Webhook implementations are encouraged to ignore rawSelector.
    // The kube-apiserver's SubjectAccessReview will parse the rawSelector. 
	RawSelector string

	// requirements is the parsed interpretation of a field selector.
	// All requirements must be met for a resource instance to match the selector.
	// Webhook implementations should handle requirements, but how to handle them is up to the webhook.
	// Since requirements can only limit the request, it is safe to authorize as unlimited request if the requirements
	// are not understood.
	Requirements []FieldSelectorRequirement
}

// LabelSelectorAttributes indicates a label limited access.
// For webhooks:
// The kube-apiserver will never send a request with rawSelector set, but we cannot control what other clients directly send.
// * If rawSelector is empty and requirements are empty, the request is not limited.
// * If rawSelector is present and requirements are empty, the request is not limited.
// * If rawSelector is empty and requirements are present, the requirements should be honored
// * If rawSelector is present and requirements are present, the request is invalid.
// Webhook authors are encouraged to
// * ensure rawSelector and requirements are not both set
// * consider the requirements field if set
// * not try to parse or consider the rawSelector field if set.
//   This is to avoid another CVE-2022-2880 (i.e. getting different systems to agree on how exactly to parse
//   a query is not something we want), see https://www.oxeye.io/resources/golang-parameter-smuggling-attack for more details.
// For the kube-apiserver:
// * If rawSelector is empty and requirements are empty, the request is not limited.
// * If rawSelector is present and requirements are empty, the rawSelector will be parsed and limited if the parsing succeeds.
// * If rawSelector is empty and requirements are present, the requirements should be honored
// * If rawSelector is present and requirements are present, the request is invalid.
type LabelSelectorAttributes struct {
	// rawSelector is the serialization of a field selector that would be included in a query parameter.
    // Webhook implementations are encouraged to ignore rawSelector.
	// The kube-apiserver's SubjectAccessReview will parse the rawSelector. 
	RawSelector string

    // requirements is the parsed interpretation of a label selector.
    // All requirements must be met for a resource instance to match the selector.
    // Webhook implementations should handle requirements, but how to handle them is up to the webhook.
    // Since requirements can only limit the request, it is safe to authorize as unlimited request if the requirements
    // are not understood.
	Requirements []metav1.LabelSelectorRequirement
}

type FieldSelectorRequirement struct {
	// key is the field selector key that the requirement applies to.
	Key string `json:"key" protobuf:"bytes,1,opt,name=key"`
	// operator represents a key's relationship to a set of values.
	// Valid operators are In, NotIn, Exists, DoesNotExist
	// The list of operators may grow in the future.
	// Webhook authors are encouraged to ignore unrecognized operators and assume they don't limit the request.
	// The semantics of "all requirements are AND'd will not change, so other requirements can continue to be enforced.
	Operator LabelSelectorOperator `json:"operator" protobuf:"bytes,2,opt,name=operator,casttype=LabelSelectorOperator"`
	// values is an array of string values. If the operator is In or NotIn,
	// the values array must be non-empty. If the operator is Exists or DoesNotExist,
	// the values array must be empty.
	// +optional
	// +listType=atomic
	Values []string `json:"values,omitempty" protobuf:"bytes,3,rep,name=values"`
}


```

Importantly, if old webhook authorizers do not honor these new fields, they will assume the broadest possible access and fail closed.
If old in-cluster authorization does not include field and label selectors, the kube-apiserver will assume the broadest possible access and fail closed.

### Node Authorizer Changes

The node authorizer will be modified to only authorize node clients to `list` and `watch` pods with fieldSelectors
containing `spec.nodeName=$nodeName`.
The node authorizer will be modified to authorize pod `get` requests based on the graph.

### CEL Authorizer Changes

While admission isn't supported on List, Watch, or DeleteCollection, it is reasonable to expect that secondary authorization
checks may desire to use those verbs and leverage the field and label selector capabilities.
To support this we will two congruent options similar to
```go
	"fieldSelector": {
		cel.MemberOverload("resourcecheck_fieldselector", []*cel.Type{ResourceCheckType, cel.StringType}, ResourceCheckType,
			cel.BinaryBinding(resourceCheckName))},
    }
```
This will allow usage like `authorizer.group('').resource('pods').fieldSelector('spec.nodeName=foo').check('list').allowed()`.
The parsing will happen during the call to `allowed` where we track errors and have means of handling them already.
Field and label selectors that fail to parse will be ignored.
No checking of valid verb,selector pairs is made.

### User Stories (Optional)

#### As a SAR client, I want to check a request with a field or label selector

This type of usage probably finds the stringified serialization format used in the query parameters the
most convenient format to build their request with.
Providing the query parameter serialization format avoids the need for a client to grow a decently complex lexer/parser.

#### As an authorization webhook author, I want to easily consume the field and label selectors

This type of usage probably finds a serialized `[]Requirement` to be the most convenient way to consume the field and label selector.
Providing the parsed value avoids the need for every consumer to grow a decently complex lexer/parser.

### Notes/Constraints/Caveats (Optional)

Remember to update these places in existing code:
1. authorization webhook matchConditions, which evaluates the v1 SubjectAccessReview that would be sent to the webhook: [ref](https://github.com/kubernetes/kubernetes/blob/bb838fde5bb9df4becb9fd267c84759be9f5400f/staging/src/k8s.io/apiserver/pkg/authorization/cel/compile.go#L197-L205).
2. v1 / v1beta1 SAR translation function [ref](https://github.com/kubernetes/kubernetes/blob/bb838fde5bb9df4becb9fd267c84759be9f5400f/staging/src/k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook.go#L472-L485)
3. v1 SubjectAccessReview construction function [ref](https://github.com/kubernetes/kubernetes/blob/bb838fde5bb9df4becb9fd267c84759be9f5400f/staging/src/k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook.go#L198)
4. cache size decision [ref](https://github.com/kubernetes/kubernetes/blob/bb838fde5bb9df4becb9fd267c84759be9f5400f/staging/src/k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook.go#L440)


### Risks and Mitigations

#### client provides field or label selector to kube-apiserver that does not parse

The kube-apiserver may still authorize the request without considering the selectors (system:masters for instance).
It will be up to the REST handler to accept or reject requests for bad selectors.
This approach also allows an aggregated API server to have extended field and label selector syntax, though we strongly discourage doing so.
The kube-apiserver will attempt to authorize without the selector information.
* If the client is authorized without the selector, then Allow since they have broader permission.
* If the client is not authorized without the selector then either NoOpinion or Fail depending on intent.

#### client provides field or label selector to kube-apiserver with improper verb

Consider a client that sends an Update request with a field selector on it.
The metav1.UpdateOption doesn't allow this, but imagine devious-user with an alternative library.
The `ResolveRequestInfo` method will not add field and label selectors to the `requestInfo`, so they will not appear
in the `authorization.Attributes`, so the spurious selectors are not passed to the authorizer.
This keeps authorization behavior exactly as it was previously.

SubjectAccessReviews are not modified prior to calling the kube-apiserver authorizer.
This allows skew in support between the kube-apiserver and other apiservers.

#### client provides SAR where field rawSelector does not match field requirements.

The request is rejected.
Only one of `rawSelector` and `requirements` can be specified.

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

[x] I/we understand the owners of the involved components may require updates to
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

```
k8s.io/kubernetes/pkg/registry/authorization/subjectaccessreview: 61.9% of statements
k8s.io/kubernetes/pkg/registry/authorization/util: 82.6% of statements
k8s.io/kubernetes/plugin/pkg/auth/authorizer/node: 77.0% of statements
k8s.io/kubernetes/pkg/apis/admissionregistration/validation: 87.6% of statements
k8s.io/kubernetes/pkg/apis/authorization/validation: 97.0% of statements
k8s.io/apiserver/pkg/admission/plugin/cel: 83.6% of statements
k8s.io/apiserver/pkg/authorization/cel: 53.9% of statements
k8s.io/apiserver/pkg/endpoints/filters: 77.2% of statements
k8s.io/apiserver/pkg/endpoints/request: 65.4% of statements
k8s.io/apiserver/plugin/pkg/authorizer/webhook: 86.6% of statements
```

Unit tests exercise node authorization, CEL compilation for authorization webhook and admission `matchConditions`,
and CEL compilation for authorizer use with and without the feature enabled:

https://github.com/kubernetes/kubernetes/blob/0b1d123fd040359da11dc772947a7908ee907910/plugin/pkg/auth/authorizer/node/node_authorizer_test.go#L75-L81

https://github.com/kubernetes/kubernetes/blob/0b1d123fd040359da11dc772947a7908ee907910/staging/src/k8s.io/apiserver/pkg/authorization/cel/compile_test.go#L34

https://github.com/kubernetes/kubernetes/blob/0b1d123fd040359da11dc772947a7908ee907910/staging/src/k8s.io/apiserver/plugin/pkg/authorizer/webhook/webhook_v1_test.go#L806

https://github.com/kubernetes/kubernetes/blob/0b1d123fd040359da11dc772947a7908ee907910/staging/src/k8s.io/apiserver/pkg/admission/plugin/cel/filter_test.go#L503-L620

##### Integration tests

<!--
Integration tests are contained in k8s.io/kubernetes/test/integration.
Integration tests allow control of the configuration parameters used to start the binaries under test.
This is different from e2e tests which do not allow configuration of parameters.
Doing this allows testing non-default options and multiple different and potentially conflicting command line options.
-->

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

- [`test/integration/apiserver/cel/authorizerselector/...`](https://github.com/kubernetes/kubernetes/tree/c5f2fc05ad5ef3d68f35263f9f965101b371b8cc/test/integration/apiserver/cel/authorizerselector) - [triage history](https://storage.googleapis.com/k8s-triage/index.html?test=test%2Fintegration%2Fapiserver%2Fcel%2Fauthorizerselector)
  - Fully exercise the new CEL authorizer functions with the feature enabled and disabled

- [`test/integration/auth TestMultiWebhookAuthzConfig`](https://github.com/kubernetes/kubernetes/blob/c5f2fc05ad5ef3d68f35263f9f965101b371b8cc/test/integration/auth/authz_config_test.go#L472-L485) - [triage history](https://storage.googleapis.com/k8s-triage/index.html?text=TestMultiWebhookAuthzConfig&test=test%2Fintegration%2Fauth)
- positive and negative match tests for a webhook matchCondition using selector matching, on actual API requests using selectors and on SubjectAccessReview requests

[Test history](https://testgrid.k8s.io/sig-release-master-blocking#integration-master&include-filter-by-regex=test/integration/apiserver/cel/authorizerselector|test/integration/auth&width=5)

##### e2e tests

This feature is fully tested with unit and integration tests

### Graduation Criteria

<!--
**Note:** *Not required until targeted at a release.*

Define graduation milestones.

These may be defined in terms of API maturity, [feature gate] graduations, or as
something else. The KEP should keep this high-level with a focus on what
signals will be looked at to determine graduation.

Consider the following in developing the graduation criteria for this enhancement:
- [Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels]
- [Feature gate][feature gate] lifecycle
- [Deprecation policy][deprecation-policy]

Clearly define what graduation means by either linking to the [API doc
definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
or by redefining what graduation means.

In general we try to use the same stages (alpha, beta, GA), regardless of how the
functionality is accessed.

[feature gate]: https://git.k8s.io/community/contributors/devel/sig-architecture/feature-gates.md
[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

Below are some examples to consider, in addition to the aforementioned [maturity levels][maturity-levels].
-->

#### Alpha

- Feature implemented behind a feature flag
- Unit tests demonstrating wiring and fallback
- Integration test demonstrating field selector wiring
  - must include fallback on parsing error as well

#### Beta

- Determine if additional tests are necessary
- Ensure reliability of existing tests

#### GA

- All bugs resolved and no new bugs requiring code change since the previous shipped release

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

On upgrade to a version that enables the feature, no configuration changes are required
to maintain previous behavior of CEL expressions and authorization webhooks.
All existing CEL expressions and authorization webhook responses behave identically.

On upgrade to a version that enables the feature, to make use of the new feature:
* authorization webhooks can inspect incoming SubjectAccessReview requests for field and label selector information
* authorization webhook configuration files can include `matchConditions` that inspect field and label selector information
* admission webhook API `matchConditions` can use authorizer fieldSelector / labelSelector functions
* SubjectAccessReview API requests can specify fieldSelector / labelSelector fields

On downgrade to a version that does not enable the feature by default, or if the feature is disabled:
* field and label selector information will no longer be sent to authorization webhooks
* authorization webhook configuration files can no longer include `matchConditions` that inspect field and label selector information
* admission webhook API `matchConditions` use authorizer fieldSelector / labelSelector functions will not error, but will no-op
* SubjectAccessReview API requests that specify fieldSelector / labelSelector fields will drop those fields

### Version Skew Strategy

#### New kube-apiserver, old webhook authorizer

The new kube-apiserver will include the field and label selectors, but the old webhook authorizer will ignore them.
The old authorizer will assume the broadest possible action and authorize accordingly.
Because the old authorizer will only allow the action if the user has permission to act on th entire collection, this fails safely.
There may be more rejections than expected, but this behavior matches previous behavior.

#### Old kube-apiserver, new in-cluster authorizer (or any SAR client)

The new client will include the field and label selectors, but the kube-apiserver will ignore them.
The kube-apiserver will assume the broadest possible action and authorize accordingly.
Because the kube-apiserver will only allow the action if the user has permission to act on th entire collection, this fails safely.
There may be more rejections than expected, but this behavior matches previous behavior.

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
  - Feature gate name: AuthorizeWithSelectors
  - Components depending on the feature gate:
    - kube-apiserver
  - Feature gate name: AuthorizeNodeWithSelectors
  - Components depending on the feature gate:
    - kube-apiserver

###### Does enabling the feature change any default behavior?

Yes.  The kube-apiserver will send field and label selector information to authorization webhooks.
The node authorizer will start preventing kubelets from listing pods that are not on their node.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.  Set the FeatureGate to false and restart the kube-apiserver.
The kube-apiserver will stop sending field and label selector information to authorization webhooks.
Persisted CEL expressions using `fieldSelector` and `labelSelector` authorization functions will still function.

###### What happens if we reenable the feature if it was previously rolled back?

The kube-apiserver will send field and label selector information to authorization webhooks.

###### Are there any tests for feature enablement/disablement?

Yes. Integration tests exercise behavior of CEL expressions with the feature enabled and disabled.

https://github.com/kubernetes/kubernetes/tree/0b1d123fd040359da11dc772947a7908ee907910/test/integration/apiserver/cel/authorizerselector

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

Non-kubelet clients using kubelet credentials to make API requests could be forbidden
if they are listing/watching pods without filtering to pods scheduled to the node,
or if they are listing/watching nodes other than their own node.

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->
Use of kubelet credentials to make API requests the kubelet is not authorized to make
is unexpected, but could be detected in the `authorization_attempts_total{result=denied}`
metric increasing and audit events showing requests from a user in the `system:nodes` group
with an `authorization.k8s.io/decision=forbid` audit annotation.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->
Handling of persisted CEL expressions using selector features was tested
with the feature disabled, and with a compatibility version of 1.30,
to ensure that a previous version API server would not have to handle
CEL expressions it did not understand.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->
No

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.

For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field.
-->
None

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->
Workloads do not use this feature directly.

Audit events of SubjectAccessReview API requests would show if
selector information was being provided.

Authorization webhooks would be able to observe selector information
provided in requests.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

Most of the uses are internal to cluster administrators:
- authorization webhooks configured with matchConditions using fieldSelector/labelSelector
  pass validation and only route requests passing those conditions to the webhook
  (`apiserver_authorization_match_condition_exclusions_total` metric will increment if match conditions skip)
- authorization webhooks can inspect the SubjectAccessReview requests sent to them to observe selector information
- admission webhooks and validating admission policies can use `fieldSelector` and `labelSelector` authorizer methods
  and pass API validation.

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

Use of this feature should not change existing API SLOs.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

Use of this feature should not change existing API SLIs.

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->
There are already metrics for the layers this feature is adding to:
- authorization latency
- authorization success
- webhook authorizer match condition latency
- webhook authorizer match condition success
- webhook admission match condition latency
- webhook admission match condition success
- validating admission policy match condition latency
- validating admission policy match condition success

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
No

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
No.

###### Will enabling / using this feature result in introducing new API types?

<!--
Describe them, providing:
  - API type
  - Supported number of objects per cluster
  - Supported number of objects per namespace (for namespace-scoped objects)
-->
No.

###### Will enabling / using this feature result in any new calls to the cloud provider?

<!--
Describe them, providing:
  - Which API(s):
  - Estimated increase:
-->
No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->
Existing API fields containing CEL expressions support additional CEL functions.

SubjectAccessReview types (which are not persisted) add new fields for fieldSelector and labelSelector data.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->
Enabling the feature adds negligible size to authorization webhook payloads.

Using the authorization selector functions in CEL expressions in authorization webhook matchConditions,
admission webhook matchConditions, and validating admission policies can take additional time,
though this is no different from increasing the complexity or number of CEL expressions generally.
CEL expressions that can be set via REST APIs are subject to cost estimation to limit the complexity
and size of the input data used for selectors.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->
No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->
No, this feature does not touch nodes.

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

This feature is fully contained within the API server.

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
- Non-kubelet clients using kubelet credentials are forbidden
  - Detection: logs of non-kubelet client, `authorization_attempts_total{result=denied}`
    metric increasing, audit events showing requests from a user in the `system:nodes` group
    with an `authorization.k8s.io/decision=forbid` audit annotation
  - Mitigations:
    - change the non-kubelet client to use its own credential (preferred)
    - adjust the non-kubelet client to use field selectors on pods and nodes
    - temporarily disable the `AuthorizeNodeWithSelectors` feature gate in kube-apiserver
  - Diagnostics: the node authorizer logs the following messages at verbosity level 2
    when a client attempts to use kubelet credentials to read nodes or pods without
    using the expected field selector:
    - `node '...' cannot read all nodes, only its own Node object`
    - `node '...' cannot read '...', only its own Node object`
    - `can only list/watch pods with spec.nodeName field selector`
  - Testing: There are tests ensuring the node authorizer forbids these overly broad
    read requests. Use of kubelet credentials by non-kubelet clients to make API
    requests the kubelet is not authorized to make is unexpected and unwanted.

###### What steps should be taken if SLOs are not being met to determine the problem?

Determine if webhook latency or matchCondition latency of matchConditions using these selector
functions is the primary contributor, and if that change correlates with enablement of this feature.
Test if eliminating use of the CEL selector functions in the offending CEL expression resolves the issue.

## Implementation History

- v1.31: Alpha release
- v1.32: Beta release

## Drawbacks

None considered

## Alternatives

None considered

## Infrastructure Needed (Optional)

None