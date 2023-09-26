# KEP-4193: bound service account token improvements (embedding requester information into JWTs)

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Embedding Pod's bound Node information in tokens](#embedding-pods-bound-node-information-in-tokens)
  - [Extending TokenReview to allow cross-checking the embedded Node information with existing Node objects](#extending-tokenreview-to-allow-cross-checking-the-embedded-node-information-with-existing-node-objects)
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

## Motivation

### Goals

* Embedding information about the Node that a pod is running on into signed JWTs.
* Make it easier to track the actions a single token has taken, and cross-reference that back to the origin of the token
  (via audit log inspection).
* Provide a means of checking whether a Pod's token is associated with the same Node as it was associated with when the
  initial TokenRequest was made (via TokenReview).

### Non-Goals

* Embedding requester information beyond just the requesting node. This is discussed further in the alternatives
  considered section, and a future KEP may revisit this.
* Embedding information beyond the Node name and UID into the token. We aim to mimic what is done with the ref fields
  for secret, pod and serviceaccount (not introduce any additional properties).
* Changing default behaviour of the TokenReview API to enforce the referenced Node object still exists.

## Proposal

### Embedding Pod's bound Node information in tokens

The kube-apiserver will be extended to automatically embed the `name` and `uid` of the *Node* a Pod is associated
with (via `spec.nodeName`) in generated tokens when a TokenRequest `create` call is serviced.

As the 'pod' is already available in this area of code, which contains the `nodeName`, we will just need to plumb
through a Getter for Node objects into the TokenRequest storage layer so the node's UID can be fetched, similar to
what is done for pod & secret objects.

### Extending TokenReview to allow cross-checking the embedded Node information with existing Node objects

The TokenReview API will also be extended to check whether a JWT's embedded node information is still valid/current,
and if not, will reject the review/indicate to the client that the token mismatches with the current state of Nodes.

This will involve first checking whether the Node object with the name given in the JWT still exists, and if it does,
validating whether the UID of that Node is equal to the UID embedded in the token.

This means administrators can **delete node objects to invalidate all JWTs issued that embed that Node's info**.

To avoid unexpected breaking changes in behaviour, this behaviour will not only be protected by the feature flag whilst
the feature is graduating to GA, but will also be gated behind an additional kube-apiserver flag which defaults to 'off'.

### Including a UUID ([JTI](https://datatracker.ietf.org/doc/html/rfc7519#section-4.1.7)) on each issued JWT

When a TokenRequest is being issued/fulfilled, we will modify the issuing code to also generate and embed a UUID which
can be later used to trace the requests that a specific issued token has made to the apiserver via the audit log.

This will require changing the JWT issuing code to actually generate this UUID, as well as extending the code around the
audit log to have it record this information into audit entries.

### User Stories (Optional)

#### Story 1

Alice is building an identity issuance system that relies on verifying projected service account tokens to ensure that
a pod exists, and is associated with a particular service account.

Alice wants to prevent a malicious actor being able to replay this token from another host, so that she can be certain
that the user requesting the identity document is also the same user that initially requested the token (as the identity
used to request the token requires validations rooted in the cryptographic root of trust on the host, e.g. a TPM).

This not only allows us to prevent replay attacks, but allows us to issue identity documents that we can 'root' in the
node's own root of trust, the TPM (as the external service can now clearly validate that the user requesting the identity
not only has a copy of that token, but also that it was the original entity to request that token).

#### Story 2

Bob is an administrator of a cluster and has noticed some strange request patterns from an unknown service account.

Bob would like to understand who initially issued/authorised this token to be issued. To do so, Bob looks up the JTI
of the token making the suspicious requests by looking inside the audit log entries for these suspect requests.

This JTI is then used for a further audit log lookup - namely, looking for the TokenRequest `create` call which contains
the audit annotation with key `authentication.kubernetes.io/token-identifier` and the value set to that of the suspect token.

This allows Bob to determine precisely who made the original request for this token, and (depending on the 'chain'
above this token), allows Bob to recursively perform this lookup to find all involved parties that led to this token
being issued.

### Notes/Constraints/Caveats (Optional)

### Risks and Mitigations

* Adding additional cross-referencing validation checks into the TokenReview API may break some user workflows that
  involve deleting Node objects and restarting kubelet's to allow them to be recreated. As a result, the TokenReview
  behaviour changes will be gated behind an additional flag in kube-apiserver, which defaults to 'off'.
  This may be revisited in future once we have a better understanding of user expectations around Node objects and
  associated JWTs.

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

[ ] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

##### Unit tests

`staging/src/k8s.io/apiserver/pkg/authentication/serviceaccount`:
* Test ensuring that service account info (JTI, node name and UID) is correctly extracted from a presented JWT.
* Tests to ensure the information is NOT extracted when the feature gate is disabled.

`pkg/serviceaccount`:
* Extending tests to ensure Node info is embedded into extended claims (name and uid)
* Tests to ensure `ID`/`JTI` field is always set to a random UUID.
* Tests to ensure the info embedded on a JWT is extracted from the token and into the ServiceAccountInfo when
  a token is validated.
* Tests to ensure the information is NOT embedded or extracted when the feature gate is disabled.

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

* Test that calls the TokenRequest API as a node user, to create a new token, and asserts that the current requesting
  node's information is correctly embedded into the resulting JWT.

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

- <test>: <link to test coverage>

##### e2e tests

* Extend existing TokenRequest e2e tests to check for embedded requester node name & UID + generated JTI is present.

- <test>: <link to test coverage>

### Graduation Criteria

#### Alpha

- JTI feature implemented behind a feature flag `ServiceAccountTokenJTI`.
- Node name/uid feature implemented behind a feature flag `ServiceAccountTokenNodeInfo`.
- TokenReview extended validation gated behind `ServiceAccountTokenNodeInfo` feature flag, as well as an extra
  kube-apiserver flag (name TBD, `--service-account-token-validate-node-info`?).
- Initial e2e tests completed and enabled

#### Beta

- TBD

#### GA

- Allowing time for feedback and any other user-experience reports.
- Conformance tests

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

This feature does not require any coordination between clients and the apiserver, as no components require this
information to be embedded. This is purely additive, and the only rollback concerns would be around third party
software that consumes this information.

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

* `ServiceAccountTokenJTI` feature flag will toggle including JTI information in tokens, as well as recording JTIs in the audit log.
* `ServiceAccountTokenNodeInfo` feature flag will toggle including node info in tokens.

Both of these feature flags can be disabled without any unexpected adverse affects or coordination required.

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `ServiceAccountTokenJTI`
  - Components depending on the feature gate: kube-apiserver

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `EmbedServiceAccountRequesterInfo`
  - Components depending on the feature gate: kube-apiserver

###### Does enabling the feature change any default behavior?

Enabling the feature gate will cause additional information to be stored/persisted into service account JWTs, as well
as new audit annotations being recorded in the audit log. This is all purely additive, so no changes to existing
features, schemas or fields are expected.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Future tokens will then not embed this information. Any existing issued tokens **will** still have this
information embedded however.

If these fields are deemed to be problematic for other systems interpreting these tokens, users will need to re-issue
these tokens before presenting them elsewhere.

###### What happens if we reenable the feature if it was previously rolled back?

Future tokens will once again include this information/no adverse effects.

###### Are there any tests for feature enablement/disablement?

Yes (as noted above in the test plan)

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

No

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

Additional audit log annotation keys, as well as extending the JWT claims we embed into service account tokens.

The maximum size of a UUID is X bytes.
The maximum size of a user's UID is Y bytes.
The maximum size of a user's username is Z bytes.

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

* There are some unknowns around whether embedding the `username` of a user is deemed a privacy concern based on
  some providers considering usernames as PII. A survey of large k8s providers is expected prior to this feature
  graduating beyond alpha.

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

N/A
