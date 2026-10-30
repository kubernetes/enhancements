# KEP-5917: Authenticator Provided Authorization Constraints
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
  - [Test Plan](#test-plan)
      - [Prerequisite testing updates](#prerequisite-testing-updates)
      - [Unit tests](#unit-tests)
      - [Integration tests](#integration-tests)
      - [e2e tests](#e2e-tests)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

An authorizer, built-in to the kube-apiserver, that evaluates constraints provided
by authenticators enabling the introduction of token formats (either in-tree or
out-of-tree) that can issue constraints for what actions can be done on the
cluster with that token.

## Motivation

Client authentication credentials for Kubernetes today map to an identity,
giving full access to everything that identity is allowed to do.

While this has been acceptable for most human-to-cluster interactions,
as delegation-based interaction models evolve having all-or-nothing
access to what the identity associated with a token can do becomes troublesome.

As a more specific example, with the rise of agentic workflows, delegating an
action to an agent today requires that a user trusts the agent (or the MCP server)
to have the same level of access to the cluster as they do.
This is known to be a bad practice, but there is no trivial way for a user to
delegate only a subset of their permissions to something/someone else.

The only way to achieve this kind of behavior today would be to:

- Create a `ServiceAccount`
- Assign it the subset of your permissions you'd like it to have
- Fetch a token for the `ServiceAccount`
- Provide the `ServiceAccount` token to the agent

This approach falls over for users that don't have permissions to create
`ServiceAccount`s or RBAC resources.

### Goals

- Allow authenticators, both in-tree and out-of-tree, to specify constraints to be evaluated as part of  authorization decisions.

### Non-Goals

- Introducing a new in-tree token format that makes use of authentication constraints.

## Proposal

- Add a new `user.Info` field, `constraints` that contains a list of constraints, provided by the authenticator to be enforced during authorization. 
- Add a new built-in authorizer, added first in the authorizer chain, that evaluates authenticator constraints.
If no authenticator constraints allow the request, the authorizer short circuits authorization by issuing a `Deny` decision,
otherwise it issues a `NoOpinion` decision.

### User Stories (Optional)

#### Story 1

As a cluster administrator, I want to configure a webhook authenticator that
allows my users to attenuate their permissions without creating additional
resources on the cluster, so that they can delegate actions to someone else or
some other system without requiring elevated permissions themselves.

#### Story 2

As a cluster administrator, I want tokens with specific claim values to determine
the level of access the token in the request is allowed, so that I can ensure tokens
issued for a user with a specific scope cannot be used beyond that scope.

### Notes/Constraints/Caveats (Optional)

While it is possible for someone to achieve the proposed functionality using a webhook authorizer,
it means that end-users need to explicitly configure the webhook authorizer to take advantage of this
behavior. Not all vendors allow webhook authorizers to be configured.

Adding this functionality in-tree means:

- It is available in all Kubernetes distributions by default.
- Enables future in-tree authenticator implementations that use it.

### Risks and Mitigations

**Risk:** Privilege escalation through authenticator constraints.
**Mitigation:** Authenticator provided constraints can only be restrictive.
This will be achieved by the authorizer responsible for evaluating these
constraints only ever returning either `NoOpinion` or `Deny` decisions,
never an `Allow` decision.

**Risk:** DoS via authenticator providing excessive constraints.
**Mitigation:** Authenticator implementations should be attentive to
the amount of constraints they are pushing to the authorization layer for
evaluation. The authorizer could set a hard limit on the number of constraints
it will attempt to enforce, but that will require additional work to understand
determine a reasonable threshold.

## Design Details

### New Go types

To facilitate communicating constraints from authenticators to the authorization
layer, the following new Go types are proposed.

```go
type AuthenticationConstraint struct {
    // for versioning
    metav1.TypeMeta

    // type is a required representation of the type of
    // authentication constraint to be enforced.
    //
    // Allowed values are "Rule".
    //
    // When set to Rule, the authentication rule specified
    // in 'rule' will be evaluated by the authentication constraint  
    // authorizer during the authorization stage of request handling.
    Type AuthenticationConstraintType `json:"type,omitempty"`

    // rule is the rule to be enforced by this authentication constraint.
    // required when type is 'Rule', and forbidden otherwise.
    Rule AuthenticationConstraintRule `json:"rule,omitempty"`
}

type AuthenticationConstraintType string

const (
    AuthenticationConstraintTypeRule AuthenticationConstraintType = "Rule"
)

// Note: This type is essentially a wrapper around
// the existing RBAC PolicyRule type
type AuthenticationConstraintRule struct {
    // apiGroups is a required set of named API groups that contains the resources.
    // If multiple API groups are specified, any action requested against one of the
    // enumerated resources in any API group will be allowed. 
    // "" represents the core API group and "*" represents all API groups.
    APIGroups []string `json:"apiGroups,omitempty"`

    // verbs is a required list of Verbs that apply to ALL the ResourceKinds
    // contained in this rule. '*' represents all verbs.
    Verbs []string `json:"verbs,omitempty"`

    // Resources is a list of resources this rule applies to.
    // '*' represents all resources in the specified apiGroups.
    // '*/foo' represents the subresource 'foo' for all resources
    // in the specified apiGroups.
    Resources []string `json:"resources,omitempty"`

    // resourceNames is an optional white list of names that the rule applies to.
    // An empty set means that everything is allowed.
    // The white list is ignored for actions in which a resource name is not
    // available during the authorization stage for evaluation.
    // '*' is not allowed.
    ResourceNames []string `json:"resourceNames,omitempty"`

    // resourceNamespaces is an optional white list of namespaces
    // that the rule applies to.
    // An empty set means that everything is allowed.
    // The white list is ignored for cluster-scoped resources
    // because they are not namespaced.
    // '*' is not allowed.
    // NOTE: this field isn't present in RBAC PolicyRule
    ResourceNamespaces []string `json:"resourceNamespaces,omitempty"`

    // nonResourceURLs is a set of partial urls that a user should have access to.
    // *s are allowed, but only as the full, final step in the path.
    // If an action is not a resource API request, then the URL is split on '/'
    // and is checked against the NonResourceURLs to look for a match.
    // Since non-resource URLs are not namespaced, this field is only applicable
    // for ClusterRoles referenced from a ClusterRoleBinding.
    // Rules can either apply to API resources (such as "pods" or "secrets")
    // or non-resource URL paths (such as "/api"),  but not both.
    NonResourceURLs []string `json:"nonResourceURLs,omitempty"`
}
```

These new Go types will be added to `/pkg/apis/authentication/types.go`.
The new APIs will be added as `v1alpha1`.

A note about `resourceNamespaces` - this is added because without it,
authenticator constraints don't have a way to represent that only requests
for namespace-scoped resources within a set of namespaces is allowed.
This deviates from the existing RBAC API because there isn't a trivial way
to represent the same cluster/namespace-scoped bindings that exist for the
RBAC APIs today, without requiring authenticators to do some evaluation of
whether or not a resource is cluster- or namespace-scoped.
Forcing authenticators to do that adds unnecessary latency to authenticator
decision making and identity mapping processes.

### Using `user.Info.Extra` to Communicate Constraints

Authenticators can use a Kubernetes-reserved key, `authentication.kubernetes.io/constraints`,
to add JSON blobs of `AuthenticatorConstraint` to `user.Info.Extra`.

Nothing but the authorizer proposed in this KEP is intended to perform any actions
with this information.

In practice, this might look something like:
```yaml
apiVersion: authentication.k8s.io/v1
kind: TokenReview
metadata:
  name: test
spec:
  token: ${TOKEN}
status:
  audiences:
  - https://kubernetes.default.svc.cluster.local
  authenticated: true
  user:
    extra:
      authentication.kubernetes.io/constraints:
      - '{"apiVersion": "authentication.k8s.io/v1alpha1", "kind": "AuthenticationConstraint", "type": "Rule", "rule": {"apiGroups":[""],"resources":["pods"],"verbs":["get", "list"],"resourceNamespaces":["one"],"resourceNames":["*"]}}'
    groups:
    - admin
    - system:authenticated
    username: username@email.com
```

### Authentication Constraint Authorizer

A new authorizer for processing authentication constraint rules
is added to the authorizer chain as the first authorizer.
This allows for short-circuiting the rest of the authorizer chain
if attenuation rules explicitly deny a request.

#### Constraint Evaluation

The authorizer will attempt to deserialize all blobs in the
`user.Info.Extra["authentication.kubernetes.io/constraints"]` value to
`AuthenticatorConstraint` objects.
Failure to deserialize a value will result in the authorizer ignoring
the constraint.
In the event the authorizer is only able to deserialize a subset of the
authenticator constraints, that subset of constraints will be enforced.
This may mean that an action that was intended to be allowed by
constraints is not actually allowed, but this approach ensures we are
never unintentionally letting actions bypass enforceable constraints.
If constraints cannot be parsed, it will be logged and/or an event
created.

Once the authorizer has the set of constraints to evaluate, constraints
will be evaluated as OR operations.
By default, the authorizer will issue a `NoOpinion` decision.
Using the `NoOpinion` decision ensures that "allows" issued by this authorizer
are non-authoritative and that future authorizers can still explicitly
deny or allow the request.
This prevents privilege escalation through authenticator provided constraints
(i.e a user mapping with authenticator constraints that are more permissive
than the RBAC rules that apply to them will never allow something that RBAC
does not).

In the event the set of constraints do not allow an action, the authorizer
will issue an explicit `Deny` decision, short-circuiting the rest of the
authorization chain.

If there are no authentication constraints to enforce,
the authorizer immediately returns a `NoOpinion`.

#### Impacts to Impersonation

Impersonation semantics would not change. Because authenticator provided constraints
are token-bound and not user-bound, impersonating a user without constraints means
that you get full access to the things that user can do.

Impersonation requests that specify constraints in the `user.Info.Extra` would
be subject to constraints just like any other request where constraints are specified.

Impersonation attempts itself are subject to authenticator constraint evaluation.
As an example, when using a token that results in constraints that do not allow
`impersonate`, any attempt to impersonate a user would be denied.
This prevents a user using a token that results in constraints from performing
privilege escalation by impersonating themselves without constraints.

#### Example Scenario

Assuming a user is cluster-admin and given constraints of:

```yaml
constraints:
- type: Rule
  rule:
    apiGroups: [""]
    resources: ["pods"]
    verbs: ["get"]
    resourceNamespaces: ["default"]
- type: Rule
  rule:
    apiGroups: [""]
    resources: ["configmaps"]
    verbs: ["list"]
    resourceNamespaces: ["default"]
```

Running `kubectl get pods/mypod -n default` would be expected
to succeed because the first constraint is satisfied.

Running `kubectl get pods/mypod -n other` would fail because no
constraints would have been satisfied.
An error message would be returned like:

```sh
Error from server (Forbidden): pods is forbidden: User "alice" cannot get resource "pods" in API group "" in the namespace "production": No authenticator constraints allowed this action.
```

### How this proposal relates to [KEP-5681: Conditional Authorization](https://github.com/kubernetes/enhancements/tree/master/keps/sig-auth/5681-conditional-authorization)

This proposal is distinct from the conditional authorization KEP.

It differs because it adds a new authorizer early in the authorization chain that
can short-circuit authorization if no authenticator provided constraint rules
allow the request, instead of waiting all the way until admission time to evaluate them.
Additionally, without some kind of authorizer to translate authenticator provided constraints
to something that can be handled in authorization/admission stages of the request, nothing
would happen if you set authenticator constraints.

Currently, this proposal only includes RBAC-like constraint rules, but in a future
where conditional authorization is GA, this authorizer could also add support
for deferring evaluation of conditions to admission, just like any other authorizer
implementation.

### Future Work

Future work for this enhancement could include:
- Expanding the allowed `AuthenticatorConstraint` API with a type for referencing RBAC resources.
For example, tying a token to be constrained by the permissions present in a `(Cluster)Role`.
- Once KEP-5681 is implemented, expanding the allowed `AuthenticatorConstraint` API with a
type for specifying more complex constraints like "only allow creating pods if they set
the label user=bob"

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

TBD

##### Unit tests

- `k8s.io/apiserver/pkg/authorization/authorizerfactory`: `02-11-2026` - `40.7` 

##### Integration tests

An integration test will be added for this feature in
`k8s.io/kubernetes/test/integration/auth/`, both when
the feature is enabled and disabled.

##### e2e tests

- TBD

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- Gather feedback from developers and surveys
- Additional tests are in Testgrid and linked in KEP
- More rigorous forms of testing—e.g., downgrade tests and scalability tests
- All functionality completed
- All security enforcement completed
- All monitoring requirements completed
- All testing requirements completed
- All known pre-release issues and gaps resolved

#### GA

- N (to be determined later) examples of real-world usage
- N (to be determined later) installs
- Allowing time for feedback
- All issues and gaps identified as feedback during beta are resolved

#### Deprecation

N/A

### Upgrade / Downgrade Strategy

No changes are required to maintain previous behavior after an upgrade.

After an upgrade, an existing cluster will need to be configured with an
authenticator that makes use of this new authenticator constraints feature
to use this feature.

On a downgrade, if the feature is in use a user would need to undo configuration
that leverages the feature or be OK with authenticator constraints being ignored.

### Version Skew Strategy

When there is version skew from the kube-apiserver with other components/clients,
older components/clients will work exactly as they always have.

When there is version skew between component/clients that use the new functionality
and the kube-apiserver (i.e a kube-apiserver that does not understand
authenticator constraints), authenticator constraints will not be enforced, which may
result in unexpected behavior.

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

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name: `AuthenticatorConstraints`
  - Components depending on the feature gate: `kube-apiserver`

###### Does enabling the feature change any default behavior?

No. By default, no authenticators would specify authenticator constraints.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. If rolled back, authenticator constraints would no longer be respected
and could result in unexpected behavior for authenticators that are leveraging
this functionality.

###### What happens if we reenable the feature if it was previously rolled back?

If re-enabled after previously rolling back the feature, authenticator constraints
would be respected again if there are any authenticators configured
that are leveraging the feature.

###### Are there any tests for feature enablement/disablement?

Tests will be added for this.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

This feature should not impact rollout or rollback.

###### What specific metrics should inform a rollback?

- High latency in authorizer decisions

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

Not yet.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

- A new metric counting the number of `Deny` decisions issued by the new authorizer
- Creating a `TokenReview` for a token that is known to result in authenticator constraints.

###### How can someone using this feature know that it is working for their instance?

Creating a `TokenReview` for a token that is known to result in authenticator constraints
and checking the status for presence of authenticator constraints.

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

###### Does this feature depend on any specific services running in the cluster?

Until there is an in-tree authenticator that uses the feature, if ever, users
will need to provide their own webhook authenticator that uses the feature.

### Scalability

###### Will enabling / using this feature result in any new API calls?

No.

###### Will enabling / using this feature result in introducing new API types?

No. It would modify existing ephemeral APIs.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

<!--
Describe them, providing:
  - API type(s):
  - Estimated increase in size: (e.g., new annotation of size 32B)
  - Estimated amount of new objects: (e.g., new Object X for every existing Pod)
-->

Yes.

`(Self)SubjectAccessReview` and `TokenReview` statuses will now include a list of authenticator provided
constraints in the `user.Info.Extra`. These would be roughly the same size as RBAC policy rules.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes. If an authenticator provides constraints to be enforced by the new authorizer,
the authorizer stage could take slightly longer. This should be a negligible increase in time taken
for the authorization stage to complete.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Maybe. In the event of an authenticator returning a large amount of constraints, it may increase CPU and RAM
usage in the kube-apiserver.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No (?).

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

It doesn't - it is a feature within the API server.

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

- TBD

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

This can be implemented out-of-tree using webhook authorizers.

Without some kind of authorizer that uses the functionality,
it is effectively useless.

This also adds some additional complexity to the authorization path.

## Alternatives

### Changes to existing Go types

The existing [`pkg/apis/authentication.UserInfo`](https://github.com/kubernetes/kubernetes/blob/7b21ce7c9adc2b491c57b5c3439ce98693e2a7e1/pkg/apis/authentication/types.go#L93-L104)
type will be updated to have a new field:

```go
type UserInfo struct {
    // ...
    // A set of constraints provided by the authenticator
    // to be evaluated during authorization.
    Constraints []AuthenticationConstraint
}
```

The existing [`k8s.io/apiserver/pkg/authentication/user.Info`](https://github.com/kubernetes/kubernetes/blob/7b21ce7c9adc2b491c57b5c3439ce98693e2a7e1/staging/src/k8s.io/apiserver/pkg/authentication/user/user.go#L20)
interface will be updated to contain a new method:

```go
type Info interface {
    // ...
    // GetConstraints returns the constraints provided by the authenticator
    // that should be enforced during the authorization stage.
    GetConstraints() []authenticationv1.AuthenticationConstraint
}

type DefaultInfo struct {
    //...
    Constraints []authenticationv1.AuthenticationConstraint
}

func (i *DefaultInfo) GetConstraints() []authenticationv1.AuthenticationConstraint {
    return i.Constraints
}
```

All authenticator implementations would be updated to implement the changes to the existing interface.

An example of how this might manifest in a `TokenReview`:
```yaml
apiVersion: authentication.k8s.io/v1
kind: TokenReview
status:
  authenticated: true
  user:
    username: user@example.com
    constraints:
    - type: Rule
      rule:
        apiGroups: [""]
        resources: ["pods"]
        verbs: ["get", "list"]
        resourceNamespaces: ["default"]
```

**Reasons for rejection**:
- `user.Info.Extra` is intentionally designed as a mechanism for authenticators to provide "interesting" information
to later request stages. Leveraging this is a natural fit.
- Using `user.Info.Extra` with a Kubernetes-reserved key is a clear signal that _only_ Kubernetes components
should react to the presence of this key as opposed to a new field in existing types having to carry
additional information through to something like webhook authorizers in a less clear way that they should
not use that information.
