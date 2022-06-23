# KEP-3325: Self subject attributes review API

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Request](#request)
  - [RBAC](#rbac)
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
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

There is no resource which represents a user in Kubernetes. Instead, Kubernetes has authenticators to get user attributes from tokens or x509 certificates or by using the OIDC provider or receiving them from the external webhook. This KEP proposes adding a new API endpoint to see what attributes the current user has after the authentication.

## Motivation

Authentication is complicated, especially made by [proxy](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#authenticating-proxy) or [webhook](https://kubernetes.io/docs/reference/access-authn-authz/authentication/#webhook-token-authentication) authenticators or their combinations.
It may be obscure which user attributes the user eventually gets after all that magic happened before authentication.
The motivation for this KEP is to reduce obscurity and help users with debugging the authentication stack.

### Goals

- Add the API endpoint to get user attributes
- Add a corresponding kubectl command - `kubectl auth who-am-i`

### Non-Goals

## Proposal

Add a new API endpoint to the `authentication.k8s.io` group - `SelfSubjectAttributesReview`.
The user will hit the endpoint after authentication happens, so all attributes will be available to return.

## Design Details

This design is inspired by the `*AccessReview` and `TokenReview` APIs.
The endpoint has no input parameters or a `spec` field because only the authentication result is required.

### Request

The structure for building a request:
```go
type SelfSubjectAttributesReview struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`
	// Status is filled in by the server with the user attributes.
	Status SelfSubjectAttributesReviewStatus `json:"status,omitempty" protobuf:"bytes,2,opt,name=status"`
}
```
```go
type SelfSubjectAttributesReviewStatus struct {
	// User attributes of the current user.
	// +optional
	UserInfo authenticationv1.UserInfo `json:"userInfo,omitempty" protobuf:"bytes,1,opt,name=userInfo"`
}
```

On receiving a request, the Kubernetes API server fills the status with the user attributes and returns it to the user.

Request example (the body would be a `SelfSubjectAttributesReview` object):
```
POST /apis/authentication.k8s.io/v1alpha1/selfsubjectattributesreview
```
```json
{
  "apiVersion": "authentication.k8s.io/v1alpha1",
  "kind": "SelfSubjectAttributesReview"
}
```
Response example:

```json
{
  "apiVersion": "authentication.k8s.io/v1alpha1",
  "kind": "SelfSubjectAttributesReview",
  "status": {
    "name": "jane.doe",
    "uid": "b6c7cfd4-f166-11ec-8ea0-0242ac120002",
    "groups": ["viewers", "editors"],
    "extra": {
      "provider_id": "token.company.dev"
    }
  }
}
```

User attributes are known at the moment of accessing the rest API endpoint and can be extracted from the request context.

NOTE: Unlike the TokenReview, there are no audiences in requests and responses since 
the SelfSubjectAttributesReview API can only be accessed using valid credentials against the API server, 
meaning that the audience must always be that of the API server. Thus learning this value is not practical.

### RBAC

RBAC rules to grant access to this API should be present in the cluster by default.
It is implied that the `system:basic-user` cluster role will be extended to the following:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    rbac.authorization.kubernetes.io/autoupdate: "true"
  labels:
    kubernetes.io/bootstrapping: rbac-defaults
  name: system:basic-user
rules:
- apiGroups:
  - authorization.k8s.io
  resources:
  - selfsubjectaccessreviews
  - selfsubjectrulesreviews
  verbs:
  - create
- apiGroups:
  - authentication.k8s.io
  resources:
  - selfsubjectattributesreviews
  verbs:
  - create
```

After reaching GA, the SelfSubjectAttributesReview API will be enabled by default. 
If necessary, it will be possible to disable this API by using the following kube-apiserver flag:
```
--runtime-config=authentication.k8s.io/v1alpha1/selfsubjectattributesreviews=false
```

### Test Plan

[X] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

<!--
Based on reviewers feedback describe what additional tests need to be added prior
implementing this enhancement to ensure the enhancements have also solid foundations.
-->

N/A

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

The plan to test the SelfSubjectAttributesReview API is:

1. Request returns all user attributes
2. Request returns some user attributes
3. Request with a status returns overridden fields

Command line interface tests covering:
1. How successful responses are rendered in the terminal with various output modes.
2. How errors are rendered.

Given that a new API package is introduced as part of this feature there is
no existing test coverage to link to.

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

1. Successful authentication through a simple authenticator, e.g., token or certificate authenticator
2. Successful authentication through a complicated authenticator, e.g., webhook or authentication proxy authenticator
3. Failed authentication

##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.
For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

There are no e2e tests planned for the alpha milestone.

### Graduation Criteria

`authentication.k8s.io/v1alpha1` and `authentication.k8s.io/v1beta1` apis will be reintroduced to go through the graduation cycle.

#### Alpha

- Feature implemented behind a feature flag
- Initial unit and integration tests completed and enabled

#### Beta

- Gather feedback from users

#### GA

- Corresponding kubectl command implemented

NOTE: Should not be a part of [conformance tests](https://git.k8s.io/community/contributors/devel/sig-architecture/conformance-tests.md).
The fact that a user possesses a token does not necessarily imply the power to know to whom that token belongs.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- Feature gate
  - Feature gate name: `SelfSubjectAttributesReview`
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

It only adds new behavior and does not affect other pars of the Kubernetes.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes.

###### What happens if we reenable the feature if it was previously rolled back?

It is possible to toggle this feature any number of times.

###### Are there any tests for feature enablement/disablement?

Does not require special testing frameworks.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

Enabling the feature does not affect any workloads.

###### What specific metrics should inform a rollback?

Specific metrics are not required for the monitoring of this feature.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

Yes.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

It is possible to see the rate of requests to the REST API endpoint.

###### How can someone using this feature know that it is working for their instance?

It will be possible to make a request to the REST API endpoint.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

The feature utilizes core mechanisms of the Kubernetes API server, so the maximum possible SLO is applicable.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

The apiserver_request_* metrics family is helpful to be aware of how many requests to the endpoint are in your cluster and how many of them failed.
```
{__name__=~"apiserver_request_.*", group="authentication.k8s.io", resource="selfsubjectattributesreview"}
```

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

All useful metrics are already present. This feature only requires metrics linked to authentication process.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

It depends only on authentication and how this process is tuned in the current Kubernetes cluster.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

```
Group: authentication.k8s.io
Kind: SelfSubjectAttributesReview
```

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

No.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

No.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

The authentication error will be returned.

###### What are other known failure modes?

The only possible errors are authentication errors.

###### What steps should be taken if SLOs are not being met to determine the problem?

No steps required.

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

## Alternatives

This feature can be implemented by delegating some requests to the external API server.
A good example of this schema working is [vmware-tanzu/pinniped](https://github.com/vmware-tanzu/pinniped) and their [`whoami`](https://github.com/vmware-tanzu/pinniped/blob/main/apis/concierge/identity/types_whoami.go.tmpl) API.

However, it is complicated to maintain an additional API server for this case, and it integrates poorly with tooling, e.g., client-go, kubectl.
