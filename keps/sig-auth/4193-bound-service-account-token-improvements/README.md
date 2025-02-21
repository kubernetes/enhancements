# KEP-4193: bound service account token improvements

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Embedding Pod's bound Node information in tokens](#embedding-pods-bound-node-information-in-tokens)
  - [Allowing ServiceAccount tokens to be bound to a Node object](#allowing-serviceaccount-tokens-to-be-bound-to-a-node-object)
  - [Extending TokenReview to verify tokens bound to Node objects](#extending-tokenreview-to-verify-tokens-bound-to-node-objects)
  - [Including a UUID (<a href="https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.7">JTI</a>) on each issued JWT](#including-a-uuid-jti-on-each-issued-jwt)
  - [User Stories (Optional)](#user-stories-optional)
    - [Story 1](#story-1)
    - [Story 2](#story-2)
  - [Notes/Constraints/Caveats (Optional)](#notesconstraintscaveats-optional)
  - [Risks and Mitigations](#risks-and-mitigations)
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
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [x] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] (R) KEP approvers have approved the KEP status as `implementable`
- [x] (R) Design details are appropriately documented
- [x] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input (including test refactors)
  - [x] e2e Tests for all Beta API Operations (endpoints)
  - [x] (R) Ensure GA e2e tests meet requirements for [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
  - [ ] (R) Minimum Two Week Window for GA e2e tests to prove flake free
- [x] (R) Graduation criteria is in place
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) 
- [x] (R) Production readiness review completed
- [x] (R) Production readiness review approved
- [x] "Implementation History" section is up-to-date for milestone
- [x] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Token projection and alternative audiences on JWTs issued by the apiserver enable an external entity to validate the
identity and certain properties (e.g. associated ServiceAccount or Pod) of the caller.

When attempting to verify a token associated with a Pod, it is not possible to verify that the Pod is associated with
a specific Node without `get`ing the relevant Pod object (embedded as a private claim in the JWT) and cross-referencing
the named `spec.nodeName`.

To allow for a robust chain of identity verification from the requester all the way through to the projected token, it
would be beneficial if the Node object reference associated with the requesting Pod were embedded into the signed JWT.

This is especially useful in cases where the external software wants to avoid replay attacks with projected service account
tokens. The external software can cross-reference the identity of the caller to that service Node reference embedded in
the JWT, which allows this verification to be rooted upon the same root of trust that the kubelet/requesting entity uses.

By embedding the identity of the Node the Pod is running on, we can cross-reference this information with an identity
passed along to the external service, thus removing the ability for a malicious actor to 'replay' a projected token
from another Node.

This will be implemented as an additional `node` entry in the private claims embedded into each JWT returned by the
TokenRequest API, in a similar manner to how the ServiceAccount, Pod or Secret is referenced.

Additionally, to provide a robust means of tracking token usage within the audit log we can embed a unique identifier for
each token which is can then also be recorded in future audit entries made by this token.

As we are adding support for `node` metadata associated with Pods, we will also add the ability to bind a token/JWT
to a Node object directly, similar to how a token can be bound to a Pod or Secret resource today.

## Motivation

### Goals

* Embedding information about the Node that a pod is running on into signed JWTs.
* Make it easier to track the actions a single token has taken, and cross-reference that back to the origin of the token
  (via audit log inspection).
* Provide a means of checking whether a Pod's token is associated with the same Node as it was associated with when the
  initial TokenRequest was made (via an extra field that can be observed from the TokenReview API).

### Non-Goals

* Embedding requester information. This is discussed further in the alternatives
  considered section, and a future KEP may revisit this.
* Embedding information beyond the immutable Node name and UID into the token. We aim to mimic what is done with the ref fields
  for secret, pod and serviceaccount (not introduce any additional properties).
* Changing default behaviour of the SA authenticator to enforce the referenced Node object still exists.

## Proposal

### Embedding Pod's bound Node information in tokens

The kube-apiserver will be extended to automatically embed the `name` and `uid` of the *Node* a Pod is associated
with (via `spec.nodeName`) in generated tokens when a TokenRequest `create` call is serviced.

As the 'pod' is already available in this area of code, which contains the `nodeName`, we will just need to plumb
through a Getter for Node objects into the TokenRequest storage layer so the node's UID can be fetched, similar to
what is done for pod & secret objects.

### Allowing ServiceAccount tokens to be bound to a Node object

Similar to how a token can be bound to a Pod or Secret object, we will also extend the TokenRequest API to allow
binding directly to Node objects (without needing to bind to a Pod as well).

This allows users to obtain a token that is tied specifically to the *Node* objects lifecycle, i.e. when the Node
object is deleted, the token will be invalidated.

### Extending TokenReview to verify tokens bound to Node objects

The SA authenticator will be extended to check whether a token that is bound to a Node object is still valid, by
first checking whether the Node object with the name given in the JWT still exists, and if it does, validating whether
the UID of that Node is equal to the UID embedded in the token.

Tokens bound to Pod objects will continue to only validate the referenced pod.
This avoids changing the previous behaviour for validation of tokens issued for pods.
Deletion of a node triggers deletion of the pods associated with that node after a [period of time](https://github.com/kubernetes/kubernetes/blob/fc786dcd1d2efcc241e0e2392086934f2806555d/pkg/controller/podgc/gc_controller.go#L50-L52),
which ultimately invalidates those tokens.

Tokens that are directly bound to Node objects will always validate the name and UID, as binding tokens to Node objects
is a new option and therefore enforcing this validation check from day 1 is non-breaking.

### Including a UUID ([JTI](https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.7)) on each issued JWT

When a TokenRequest is being issued/fulfilled, we will modify the issuing code to also generate and embed a UUID which
can be later used to trace the requests that a specific issued token has made to the apiserver via the audit log.

This will require changing the JWT issuing code to actually generate this UUID, as well as extending the code around the
audit log to have it record this information into audit entries when a token is issued (via the `authentication.k8s.io/issued-credential-id` audit annotation).

As this UUID will be embedded as part of a user's ExtraInfo, it'll automatically be persisted into audit events for all
requests made using a token that embeds a credential identifier (as `authentication.k8s.io/credential-id`).

### User Stories (Optional)

#### Story 1

Alice hosts a service that verifies host identity using an out-of-band mechanism and also submits a bound token that
contains a node assertion.

The node assertion can be checked to ensure the host identity matches the node assertion of the token.

#### Story 2

Bob is an administrator of a cluster and has noticed some strange request patterns from an unknown service account.

Bob would like to understand who initially issued/authorised this token to be issued. To do so, Bob looks up the JTI
of the token making the suspicious requests by looking inside the audit log entries at user's ExtraInfo for these suspect requests.

This JTI is then used for a further audit log lookup - namely, looking for the TokenRequest `create` call which contains
the audit annotation with key `authentication.kubernetes.io/issued-credential-id` and the value set to that of the suspect token.

This allows Bob to determine precisely who made the original request for this token, and (depending on the 'chain'
above this token), allows Bob to recursively perform this lookup to find all involved parties that led to this token
being issued.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

* Adding additional cross-referencing validation checks into the TokenReview API may break some user workflows that
  involve deleting Node objects and restarting kubelet's to allow them to be recreated. As a result, the TokenReview
  API will **NOT** be modified to permit tightening this validation behaviour. Instead, the existing protections &
  mechanisms for invalidating a Node<>Pod binding (i.e. auto-deletion after a fixed time period after the Node object
  is deleted).

## Design Details

The `pkg/serviceaccount/claims.go` file's `Claims` [function](https://github.com/kubernetes/kubernetes/blob/99190634ab252604a4496882912ac328542d649d/pkg/serviceaccount/claims.go#L61-L97)
will be modified to accept a `core.Node`. This will be made available in the call-site for this function
(`pkg/registry/core/serviceaccount/storage/token.go`) by passing through a Getter for Node objects, similar to how
secret objects are fetched.

The associated `Validator` used to validate and parse service account tokens will also be extended to extract this
new information from tokens if it is available.

In `pkg/registry/core/serviceaccount/storage/token.go`, the `Create` function will also be extended to add an audit
annotation including the generated service account token's JTI, to make it possible to map a future request which
used this token back to the initial point at which the token was generated (i.e. to allow deeper inspection of who
the requester is).

In the file `staging/src/k8s.io/apiserver/pkg/authentication/serviceaccount/util.go`, the `ServiceAccountInfo.UserInfo`
method will be modified to also return this information in the returned `user.Info` struct.

These proposed changes can also be reviewed in [the draft pull request](https://github.com/kubernetes/kubernetes/pull/119739).

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

`pkg/registry/core/serviceaccount/storage`:
* Coverage before (`release-1.28`): `k8s.io/kubernetes/pkg/registry/core/serviceaccount/storage      8.354s  coverage: 10.7% of statements`
* Coverage after: `k8s.io/kubernetes/pkg/registry/core/serviceaccount/storage      8.394s  coverage: 8.7% of statements`
* Test ensuring audit annotations are added to audit events for the `serviceaccounts/<name>/token` subresource.
* Tests verifying it's possible to bind a token to a Node object.
* Tests ensuring tokens bound to pod objects also embed associated node metadata.
* NOTE: the majority of this file is untested with *unit tests* (instead, using integration tests). [#121515](https://github.com/kubernetes/kubernetes/issues/121515).

`staging/src/k8s.io/apiserver/pkg/authentication/serviceaccount`:
* Coverage before (`release-1.28`): `k8s.io/apiserver/pkg/authentication/serviceaccount      0.567s  coverage: 60.8% of statements`
* Coverage after: `k8s.io/apiserver/pkg/authentication/serviceaccount      0.569s  coverage: 70.1% of statements`
* Test ensuring that service account info (JTI, node name and UID) is correctly extracted from a presented JWT.
* Tests to ensure the information is NOT extracted when the feature gate is disabled.

`pkg/serviceaccount`:
* Coverage before (`release-1.28`): `k8s.io/kubernetes/pkg/serviceaccount    0.755s  coverage: 72.4% of statements`
* Coverage after: `k8s.io/kubernetes/pkg/serviceaccount    0.786s  coverage: 72.7% of statements`
* Extending tests to ensure Node info is embedded into extended claims (name and uid)
* Tests to ensure `ID`/`JTI` field is always set to a random UUID.
* Tests to ensure the info embedded on a JWT is extracted from the token and into the ServiceAccountInfo when
  a token is validated.
* Tests to ensure the information is NOT embedded or extracted when the feature gate is disabled.

`staging/src/k8s.io/kubectl/pkg/cmd/create`:
* Coverage before (`release-1.28`): `k8s.io/kubectl/pkg/cmd/create   0.995s  coverage: 55.1% of statements`
* Coverage after: `k8s.io/kubectl/pkg/cmd/create   0.949s  coverage: 55.2% of statements`
* Add tests ensuring it's possible to request a token that is bound to a Node object (gated by environment variable during alpha)

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

##### Integration tests

* Test that calls the TokenRequest API to obtain a token that is bound to a Pod. It should assert that the token embeds
  a reference to the Pod object, as well as to the Node object that the Pod is assigned to.
* Test that calls the TokenRequest API to obtain a token that is bound to a Node. It should assert that the token embeds
  a reference to the Node object.
* Test that calls the TokenReview API with a token that is bound to a Node object that no longer exists. It should
  assert that the token does not validate once the Node has been deleted.

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

`k8s.io/test/integration/sig-auth/svcacct_test.go`
- [TestServiceAccountTokenCreate_bound to a service account and pod](https://github.com/kubernetes/kubernetes/blob/release-1.29/test/integration/auth/svcaccttoken_test.go#L247)
- [TestServiceAccountTokenCreate_bound to service account and a pod with an assigned nodeName that does not exist](https://github.com/kubernetes/kubernetes/blob/release-1.29/test/integration/auth/svcaccttoken_test.go#L415)
- [TestServiceAccountTokenCreate_bound to service account and a pod with an assigned nodeName](https://github.com/kubernetes/kubernetes/blob/release-1.29/test/integration/auth/svcaccttoken_test.go#L416)
- [TestServiceAccountTokenCreate_fails to bind to a Node if the feature gate is disabled](https://github.com/kubernetes/kubernetes/blob/release-1.29/test/integration/auth/svcaccttoken_test.go#L418)
- [TestServiceAccountTokenCreate_bound to service account and node](https://github.com/kubernetes/kubernetes/blob/release-1.29/test/integration/auth/svcaccttoken_test.go#L448)

##### e2e tests

* Extend existing TokenRequest e2e tests to check for embedded scheduled node name & UID + generated JTI is present.

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- JTI feature implemented behind a feature flag `ServiceAccountTokenJTI`.
- Embedding Pod's assigned Node name/uid feature implemented behind a feature flag `ServiceAccountTokenPodNodeInfo`.
- Support verifying JWTs bound to Node objects with feature flag `ServiceAccountTokenNodeBindingValidation`.
- Allowing tokens bound to Node objects to be issued with feature flag flag `ServiceAccountTokenNodeBinding`.
- Initial e2e tests completed and enabled

#### Beta

- Decide what the default of the new flag should be
  - Decision: this flag was not added during alpha, and MAY be added post-beta, but will definitely default to **off**.
  - This does not need to block promotion of ServiceAccountTokenPodNodeInfo feature as a result.
- Decide if using an audit annotation is the correct approach
  - Decision: audit annotation is the correct approach as this is only for `serviceaccounts/<name>/token` requests, not all
  - Renaming audit annotation to `authentication.kubernetes.io/issued-credential-id` to disambiguate from `authentication.kubernetes.io/credential-id` in user's ExtraInfo
- Docs around the SA JWT schema (this does not exist today)

#### GA

- Allowing time for feedback and any other user-experience reports.
- Conformance tests
- Consolidate the existing service account docs to be more coherent and avoid duplication,
  especially in regards to consuming service account tokens outside of Kubernetes:
    - https://kubernetes.io/docs/concepts/security/service-accounts
    - https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin
    - https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account

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

Embedding a Pod's assigned Node name into a JWT does not require any coordination between clients and the apiserver,
as no components require this information to be embedded. This is purely additive, and the only rollback concerns
would be around third party software that consumes this information. This software should always verify whether a
`node` claim is embedded into tokens if they require using it, and provide a fall-back behaviour (i.e. a GET to the
apiserver to fetch the Pod & Node object) if they need to maintain compatibility with older apiservers.

Binding a token to a Node introduces a new validation mechanism, and therefore we must allow one release cycle after
introducing the ability to **validate** tokens, before we can begin permitting **issuance** of these tokens.
This is a critical step from a security standpoint, as otherwise an administrator could:

1) upgrade their apiserver/control plane.
2) a user could request a token bound to a Node, expecting it to be invalidated when the Node is deleted.
3) rollback the apiserver to an older version.
4) the Node object is deleted.
5) the token issued in (2) would now continue to be accepted/validated, despite the Node object no longer existing.

By graduating validation a release **earlier** than issuance, we can ensure any tokens that are bound to a Node
object will be correctly validated even after a rollback.

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

* `ServiceAccountTokenJTI` feature flag will toggle including JTI information in tokens, as well as recording JTIs in the audit log / the SA user info.
* `ServiceAccountTokenPodNodeInfo` feature flag will toggle including node info associated with pods in tokens.
* `ServiceAccountTokenNodeBindingValidation` feature flag will toggle the apiserver validating Node claims in node bound service account tokens.
* `ServiceAccountTokenNodeBinding` feature flag will toggle allowing service account tokens to be bound to Node objects.

The `ServiceAccountTokenNodeBindingValidation` feature will graduate to beta in version v1.30, a release earlier than `ServiceAccountTokenNodeBinding`
to ensure a safe rollback from version v1.31 to v1.30 (more info below in rollback considerations section).

The `ServiceAccountTokenNodeBinding` feature gate must only be enabled once the `ServiceAccountTokenNodeBindingValidation` feature has been enabled.
Disabling the `ServiceAccountTokenNodeBindingValidation` feature whilst keeping `ServiceAccountTokenNodeBinding` would allow tokens that are expected to
be bound to the lifetime of a particular Node to validate even if that Node no longer exists.
The [rollout & rollback section](#rollout-upgrade-and-rollback-planning) below goes into further detail.

All other feature flags can be disabled without any unexpected adverse affects or coordination required.

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `ServiceAccountTokenJTI`
  - Components depending on the feature gate: kube-apiserver

- [x] Feature gate
  - Feature gate name: `ServiceAccountTokenPodNodeInfo`
  - Components depending on the feature gate: kube-apiserver

- [x] Feature gate
  - Feature gate name: `ServiceAccountTokenNodeBinding`
  - Components depending on the feature gate: kube-apiserver

- [x] Feature gate
  - Feature gate name: `ServiceAccountTokenNodeBindingValidation`
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

Enabling the `ServiceAccountTokenPodNodeInfo` and/or `ServiceAccountTokenJTI`  feature gate will cause additional information
to be stored/persisted into service account JWTs, as well as new audit annotations being recorded in the audit log.
This is all purely additive, so no changes to existing features, schemas or fields are expected.

Enabling the `ServiceAccountTokenNodeBinding` will permit binding tokens to Node objects, which is a change in
behaviour (albeit not to an existing feature, so is not problematic).

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Future tokens will then not embed this information. Any existing issued tokens **will** still have this
information embedded, however.

If these fields are deemed to be problematic for other systems interpreting these tokens, users will need to re-issue
these tokens before presenting them elsewhere.

Once the feature(s) have graduated to GA, it will not be possible to disable this behaviour.

###### What happens if we reenable the feature if it was previously rolled back?

Future tokens will once again include this information/no adverse effects.

###### Are there any tests for feature enablement/disablement?

Yes (as noted above in the test plan)

### Rollout, Upgrade and Rollback Planning

Rolling this out will be done by enabling the feature flag on all control plane hosts.

The `ServiceAccountTokenNodeBindingValidation` feature gate should be enabled and complete rollout before the
`ServiceAccountTokenNodeBinding` gate is enabled, so all active servers will correctly validate tokens issued by
any server.

The `ServiceAccountTokenNodeBindingValidation` will be defaulted to on one release **before** `ServiceAccountTokenNodeBinding`
to account for this. Concretely, `ServiceAccountTokenNodeBindingValidation` will be enabled by default in v1.30 and
`ServiceAccountTokenNodeBinding` will be enabled by default in v1.31.

This should not have any issues/affect during upgrades.
Rollback is done by removing/disabling the feature gate(s).

###### How can a rollout or rollback fail? Can it impact already running workloads?

During a rollback, there is a concern that tokens that were issued prior to the rollback that are bound directly to a
Node object (i.e. not bound to a Pod that also embeds node info, which is informational) could be accepted by an older
apiserver even if the bound Node object no longer exists (as it would not know to verify the new `node` claim).

To help avoid this, the feature will be graduated in two phases:

* First, graduating the acceptance/validation of explicitly node-scoped tokens in one release
* Secondly, graduating the issuance of explicitly Node bound tokens

This allows for a safe rollback in which the same security expectations are enforced once a token has been issued.

If a user explicitly *disables* `ServiceAccountTokenNodeBindingValidation` but keeps `ServiceAccountTokenNodeBinding` enabled,
the node claims in the issued tokens will not be properly validated. This configuration will be explicitly denied by the
kube-apiserver and will cause it to exit on startup.

###### What specific metrics should inform a rollback?

* `authentication_attempts`
* `authorization_attempts_total`
* `serviceaccount_valid_tokens_total`

New metrics that can be used to identify if the feature is in use:

* `serviceaccount_authentication_pod_node_ref_verified_total`
* `serviceaccount_authentication_bound_object_verified_total{bound_object_kind="Node"}`
* `serviceaccount_bound_tokens_issued_pod_with_node_tokens_total`
* `serviceaccount_bound_tokens_issued_total{bound_object_kind="Node"}`
* `serviceaccount_bound_tokens_issued_with_identifier_total`

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

**For `ServiceAccountTokenJTI` feature (alpha v1.29, beta v1.30, GA v1.32):**

*Without* the feature gate enabled, issued service account tokens *will not* have their `jti` field set to a random UUID,
and the audit log will not persist the issued credential identifier when issuing a token.

*With* the feature gate enabled, issued service accounts will set the `jti` field to a random UUID.
Additionally, the audit event recorded when issuing a new token will have a new annotation added (`authentication.k8s.io/issued-credential-id`).
As a service account's JTI field is used to infer the credential identifier, which forms part of a users `ExtraInfo`,
audit events generated using this newly issued token will also include this JTI (persisted as `authentication.k8s.io/credential-id`).

If the feature is *disabled* and a token is presented that includes a credential identifier, **it will still be persisted into the audit log**
as part of the UserInfo in the audit event.

As none of these fields are actually used for validating/verifying a token is valid, enabling & disabling the feature
does not cause any adverse side effects.

**For `ServiceAccountTokenNodeBinding` (alpha v1.29, beta v1.31, GA v1.33) and `ServiceAccountTokenNodeBindingValidation` (alpha v1.29, beta v1.30, GA v1.32) feature:**

*Without* the feature gate enabled, service account tokens that have been bound to Node objects will not have their
node reference claims validated (to ensure the referenced node exists).

*With* the feature gate enabled, if a token has a `node` claim contained within it, it'll be validated to ensure the
corresponding Node object actually exists.

Disabling this feature will therefore *relax* the security posture of the cluster in an unexpected way, as tokens that
may have been previously invalid (because their corresponding Node does not exist) may become valid again.

Node bound tokens may only be issued if the `ServiceAccountTokenNodeBinding` feature is enabled, and it is not possible
to enable `ServiceAccountTokenNodeBinding` without `ServiceAccountTokenNodeBindingValidation` being enabled too.

This is further mitigated by graduating the `ServiceAccountTokenNodeBindingValidation` feature one release **earlier**
than `ServiceAccountTokenNodeBinding`.

Tokens that are bound to objects other than Nodes are unaffected.

**For `ServiceAccountTokenPodNodeInfo` feature (alpha v1.29, beta v1.30, GA v1.32):**

*Without* the feature gate enabled, tokens that are bound to Pod objects will not include information about the Node
that the pod is scheduled/assigned to.

*With* the feature enabled, newly minted tokens that are bound to Pod objects will include metadata about the Node, namely
the Node's name and UID.

These fields are **not validated** and therefore disabling the feature after enabling it will not cause any adverse side-effects.

``
<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

New metrics:

* `serviceaccount_authentication_pod_node_ref_verified_total` - new metric that is incremeneted when a token bound to a Pod has its Node reference verified
* `serviceaccount_authentication_bound_object_verified_total{bound_object_kind="Node"}` - new metric that is incremeneted when a token bound to a Node has its reference verified
* `serviceaccount_bound_tokens_issued_pod_with_node_tokens_total` - new metric that is incremented when a node ref is embedded into a bound Pod token (aka implicitly added)
* `serviceaccount_bound_tokens_issued_total{bound_object_kind="Node"}` - new metric that is incremented whenever a bound token is issued that references a Node (explicitly added)
* `serviceaccount_bound_tokens_issued_with_identifier_total` - new metric that is incremented whenever a token that contains an identifier/JTI is issued

###### How can an operator determine if the feature is in use by workloads?

The metrics detailed above provide a clear signal as to whether these features are being used.

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### How can someone using this feature know that it is working for their instance?

For the node info part, using the TokenRequest API and inspecting the contents of the issued JWTs for a token bound to a Pod.
For JTIs, using the TokenRequest API and then inspecting the contents of the issued JWT for any ServiceAccount token.

For the validation/verification, the user can use the SelfSubjectAccessReview API to check whether the token is still valid.
To do so, they'd need to obtain a token that is bound to a Pod, delete the corresponding Node object that the Pod is scheduled
on, and observe that the token is no longer valid via the SelfSubjectAccessReview API.

A similar process could be used for tokens bound to Node objects directly.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

N/A

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

N/A

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

No

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

None

###### Does this feature depend on any specific services running in the cluster?

No

### Scalability

N/A

###### Will enabling / using this feature result in any new API calls?

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Additional audit log annotation keys, as well as extending the JWT claims we embed into service account tokens.

The maximum size of a UUID is 36 bytes.
The maximum size of a Node object's name is 253 bytes.
The maximum size of a Node object's UID is 36 bytes.

This additional data will be recorded into issued JWTs as well as audit log events.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Fractionally increase the time spent issuing service account JWTs (UUID generation mainly). This is expected to be negligible.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No

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

Not applicable. This change is solely within the apiserver, and does not touch etcd.

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

After observing an issue (e.g. uptick in denied authentication requests or a significant shift in any metrics added for this KEP),
kube-apiserver logs from the authenticator may be used to debug.

Additionally, manually attempting to exercise the affected codepaths would surface information that'd aid debugging.
For example, attempting to issue a node bound token, or attempting to authenticate to the apiserver using a node bound token.

## Implementation History

* KEP marked implementable and merged for the v1.29 release
* KEP implemented in an alpha state for v1.29
* Renamed audit annotation used for the `serviceaccounts/<name>/token` endpoint to be clearer: https://github.com/kubernetes/kubernetes/pull/123098
* Added restrictions to disallow enabling `ServiceAccountTokenNodeBinding` without `ServiceAccountTokenNodeBindingValidation`: https://github.com/kubernetes/kubernetes/pull/123135
* `ServiceAccountTokenJTI`, `ServiceAccountTokenNodeBindingValidation` and `ServiceAccountTokenPodNodeInfo` promoted to beta for v1.30 release
* Promoted `ServiceAccountTokenNodeBinding` promoted to beta for v1.31 release
* Promoted `ServiceAccountTokenJTI`, `ServiceAccountTokenPodNodeInfo`, `ServiceAccountTokenNodeBindingValidation` to stable for v1.32 release
* Promoted `ServiceAccountTokenNodeBinding` to stable for v1.33 release

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

* TBC

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

N/A
