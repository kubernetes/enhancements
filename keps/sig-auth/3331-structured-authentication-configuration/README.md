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
# KEP-3331: Structured Authentication Config

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
    - [CEL](#cel)
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
  - [Infrastructure Needed](#infrastructure-needed)
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

This enhancement proposal covers adding structured authentication configuration to the Kubernetes API server.
Initially, only a `jwt` configuration will be supported, which will serve as the next iteration of the existing
OIDC authenticator.  OIDC authentication is an important part of Kubernetes, yet it has limitations in its current state.
Below we will discuss that limitation and propose solutions.

# Motivation

Structured config for OIDC authentication: noted in various contexts over the past few years. We want to migrate
away from a flag-based config that is growing without bounds to a proper versioned config format. This would allow us to
better support various features that have been requested.

- [Support multiple ClientIDs](https://github.com/kubernetes/kubernetes/issues/71162)
- [Identify user by more JWT claims than just a single one](https://github.com/kubernetes/kubernetes/issues/71715)
- [Allow any claim from an ID token to be mapped to user info extra attributes](https://github.com/kubernetes/kubernetes/issues/82236)
- [Fail authentication unless user is a member of a specific set of groups](https://github.com/kubernetes/kubernetes/issues/84730)
- [Required claims does not support arrays](https://github.com/kubernetes/kubernetes/issues/101291)
- [Forward audiences to authorization via user info extra attributes](https://github.com/kubernetes/kubernetes/pull/117474)

### Goals

There are features users want to tune. We need to provide customization of the following:

- *Claims validation rules*: current OIDC provider supports only audience claim validation and only by exact values.
- *Claim mappings*: it is only possible to pick a single value from a single claim and prefix groups.
  - This also serves to support the use of nested claims (i.e. not at the top level)
- *Use more than one OIDC provider*: the only option, for now, is to use an external OIDC provider that handles multiplexing support for multiple providers.
  - This also serves to support the use of more than one client ID
- Change authenticator settings without restarting kube-apiserver.
- Support validation rules on the final user info object to allow infra providers to safely expose this functionality to customers
- Easy migration from existing OIDC flags
  - Note: we intend to drop the `--oidc-signing-algs` flag because this configuration provides no benefit (we will always allow all asymmetric algorithms)

### Non-Goals

- Supporting configuration of authentication mechanisms other than `jwt` (this is deferred to future KEPs)
- Supporting methods for keys discovery other than standard OIDC discovery mechanism via the well-known endpoint
- Support for certificate based signing via the `x5c` header field (this is deferred until we have more user evidence)
- Supporting access to the JWT header in rules (this is deferred until we have more user evidence)
- Supporting access to the provider config itself in rules (this is deferred until we have more user evidence)
- Supporting JWTs with multiple signatures (to avoid any security issues caused by signature confusion)

## Proposal

1. Add new `apiserver.config.k8s.io` API object to parse a structured config file `AuthenticationConfiguration`.
2. Add a single flag `--authentication-config` for kube-apiserver to point to the structured config file, automatically reload it on changes.
3. Use an expression language to let users write their own logic for mappings and validation rules
  (expressions should be simple for common cases, yet powerful to cover most user stories).

### Risks and Mitigations

Since this is a new optional feature, no migration is required unless the user wants to replace their
existing OIDC flags usage with the config file.  The use of `--authentication-config` is mutually exclusive
with the existing OIDC flags, so we will provide documentation for migrating from a flag-based config to the new config.

## Design Details

### Configuration file

TODO:

- should we have any revocation mechanism?
  => use revocation endpoint if it is in the discovery document? (lets decide what we want here before beta)
  => related issue https://github.com/kubernetes/kubernetes/issues/71151
- distributed claims with fancier resolution requirements (such as access tokens as input)
- implementation detail: we should probably parse the `iss` claim out once
- should audit annotations be set on validation failure?
- decide what error should be returned if CEL eval fails at runtime
  `500 Internal Sever Error` seem appropriate but authentication can only do `401`

The main part of this proposal is a configuration file. It contains an array of providers:

```yaml
apiVersion: apiserver.config.k8s.io/v1alpha1
kind: AuthenticationConfiguration
jwt:
- issuer:
    url: https://example.com
    audiences:
    - my-app
  claimValidationRules:
  - claim: hd
    requiredValue: example.com
  - expression: 'claims.hd == "example.com"'
    message: the hd claim must be set to example.com
  - expression: 'claims.exp - claims.nbf <= 86400'
    message: total token lifetime must not exceed 24 hours
  claimMappings:
    username:
      expression: 'claims.username + ":external-user"'
    groups:
      expression: 'claims.roles.split(",")'
    uid:
      claim: 'sub'
    extra:
    - key: 'client_name'
      valueExpression: 'claims.some_claim'
  userInfoValidationRules:
  - rule: "!userInfo.username.startsWith('system:')"
    message: username cannot used reserved system: prefix
  - rule: "userInfo.groups.all(group, !group.startsWith('system:'))"
    message: groups cannot used reserved system: prefix
```

The minimum valid payload from a JWT is (`aud` may be a `string`):

TODO:
are `iat` and `nbf` required?
is `sub` required or is the requirement to just have some username field?

```json
{
  "iss": "https://example.com",
  "sub": "001",
  "aud": [
    "cluster-a"
  ],
  "exp": 1684274031,
  "iat": 1684270431,
  "nbf": 1684270431
}
```

Payloads with nested data are supported as well (it will be possible
to use the `foo` value as a claim mapping):

```json
{
  "custom": {
    "data": {
      "name": "foo"
    }
  },
  ...
}
```

The order in which validations and claim mapping occurs is as follows:

TODO: mermaid diagram

1. OIDC validations
    - `iss`
    - TODO enumerate these
2. Claim validation based on `claimValidationRules`
3. Claim mapping based on `claimMappings`
4. User info validation based on `userInfoValidationRules`

```go
type AuthenticationConfiguration struct {
  metav1.TypeMeta `json:",inline"`

    // jwt is a list of OIDC providers to authenticate Kubernetes users.
    // For an incoming token, each JWT authenticator will be attempted in
    // the order in which it is specified in this list.  Note however that
    // other authenticators may run before or after the JWT authenticators.
    // The specific position of JWT authenticators in relation to other
    // authenticators is neither defined nor stable across releases.  Since
    // each JWT authenticator must have a unique issuer URL, at most one
    // JWT authenticator will attempt to cryptographically validate the token.
    JWT []JWTAuthenticator `json:"jwt"`
}
```

Each authenticator has several properties that will be described in detail below.

```go
type JWTAuthenticator struct {
    // issuer is a basic OIDC provider connection options.
    Issuer Issuer `json:"issuer"`

    // claimValidationRules are rules that are applied to validate token claims to authenticate users.
    // +optional
    ClaimValidationRules []ClaimValidationRule `json:"claimValidationRules,omitempty"`

    // claimMappings points claims of a token to be treated as user attributes.
    ClaimMappings ClaimMappings `json:"claimMappings"`

    // ClaimsFilter allows unmarshalling only required claims which positively affects performance.
    // TODO: this is only dist claims -> drop this and figure out to get from CEL
    //
    // 3. `ClaimsFilter` - list of claim names that should be passed to CEL expressions. The assumption is that administrators
    // know the structure of the token and the exact claims they will use in CEL expressions.
    // This option helps to reduce system load and operate only with required claims.
    //
    // +optional
    // ClaimsFilter []string `json:"claimFilters,omitempty"`

    // userInfoValidationRules are rules that are applied to final userInfo before completing authentication.
    // These allow invariants to be applied to incoming identities such as preventing the
    // use of the system: prefix that is commonly used by Kubernetes components.
    // +optional
    UserInfoValidationRules []UserInfoValidationRule `json:"userInfoValidationRules,omitempty"`
}
```

1. `Issuer` - is a section for external provider specific settings, e.g., OIDC discovery URL.

```go
    type Issuer struct {
        // url points to the issuer URL in a format https://url/path.
        // This must match the "iss" claim in the presented JWT, and the issuer returned from discovery.
        // Same value as the --oidc-issuer-url flag.
        // Used to fetch discovery information unless overridden by discoveryURL.
        // Required to be unique.
        // Note that egress selection configuration is not used for this network connection.
        // TODO: decide if we want to support egress selection configuration and how to do so.
        URL string `json:"url"`

        // discoveryURL if specified, overrides the URL used to fetch discovery information.
        // This is for scenarios where the well-known and jwks endpoints are hosted at a different
        // location than the issuer (such as locally in the cluster).
        // Format must be https://url/path.
        //
        // Example:
        // A discovery url that is exposed using kubernetes service 'oidc' in namespace 'oidc-namespace'.
        // certificateAuthority is used to verify the TLS connection and the hostname on the leaf certifcation
        // must be set to 'oidc.oidc-namespace'.
        //
        // curl https://oidc.oidc-namespace (.discoveryURL field)
        // {
        //     issuer: "https://oidc.example.com" (.url field)
        // }
        //
        // Required to be unique.
        // Note that egress selection configuration is not used for this network connection.
        // TODO: decide if we want to support egress selection configuration and how to do so.
        // +optional
        DiscoveryURL *string `json:"discoveryURL,omitempty"`

        // certificateAuthority contains PEM-encoded certificate authority certificates
        // used to validate the connection when fetching discovery information.
        // If unset, the system verifier is used.
        // Same value as the content of the file referenced by the --oidc-ca-file flag.
        // +optional
        CertificateAuthority string `json:"certificateAuthority,omitempty"`

        // audiences is the set of acceptable audiences the JWT must be issued to.
        // At least one of the entries must match the "aud" claim in presented JWTs.
        // Same value as the --oidc-client-id flag (though this field supports an array).
        // Required to be non-empty.
        Audiences []string `json:"audiences,omitempty"`
   }
   ```

2. `ClaimValidationRules` - additional authentication policies. These policies are applied after generic OIDC validations, e.g., checking the token signature, issuer URL, etc. Rules are applicable to distributed claims.

    ```go
    type ClaimValidationRule struct {
        // claim is the name of a required claim.
        // Same as --oidc-required-claim flag.
        // Only string claims are supported.
        // Mutually exclusive with expression and message.
        // +optional
        Claim string `json:"claim"`
        // requiredValue is the value of a required claim.
        // Same as --oidc-required-claim flag.
        // Mutually exclusive with expression and message.
        // +optional
        RequiredValue string `json:"requiredValue"`

        // expression is a logical expression that is written in CEL https://github.com/google/cel-go.
        // Must return true for the validation to pass.
        // Mutually exclusive with claim and requiredValue.
        // +optional
        Expression string `json:"expression"`
        // message customizes the returned error message when expression returns false.
        // Mutually exclusive with claim and requiredValue.
        // Note that messageExpression is explicitly not supported to avoid
        // misconfigured expressions from leaking JWT payload contents.
        // +optional
        Message string `json:"message,omitempty"`
    }
    ```

    For validation expressions, the CEL is used. They are similar to validations functions for [Custom Resources](https://kubernetes.io/docs/tasks/extend-kubernetes/custom-resources/custom-resource-definitions/#resource-use-by-validation-functions).
    Validation expressions must always evaluate a boolean. Token `claims` (payload) are passed to CEL expressions as a dynamic map `decls.NewMapType(decls.String, decls.Dyn)`.

    > NOTE: If an expression returns `false` after evaluation, a `401 Unauthorized` error will be returned
      and the associated `message` will be logged in the API server logs.

    Example validation rules:

    ```yaml
    claimValidationRules:

    - expression: 'claims.aud == "charmander" || claims.aud == "bulbasaur"'
      message: clients other than charmander or bulbasaur are not allowed

    - expression: 'claims.exp - claims.nbf <= 86400'
      message: total token lifetime must not exceed 24 hours
    ```

3. `ClaimMappings` - rules to map claims from a token to Kubernetes user info attributes.

    ```go
    type ClaimMappings struct {
        // username represents an option for the username attribute.
        // Claim must be a singular string claim.
        // TODO: decide whether to support a distributed claim for username (what are we required to correlate between the data retrieved for distributed claims? sub? something else?). Limit distributed claim support to OIDC things with clientID validation?
        // Expression must produce a string value that must be non-empty.
        // Possible prefixes based on the config:
        //     (1) if userName.prefix = "-", no prefix will be added to the username
        //     (2) if userName.prefix = "" and userName.claim != "email", prefix will be "<issuer.url>#"
        //     (3) if userName.expression is set instead, result of expression is used as-is without any implicit prefix
        // (1) and (2) ensure backward compatibility with the --oidc-username-claim and --oidc-username-prefix flags
        // +required
        Username PrefixedClaimOrExpression `json:"username"`
        // groups represents an option for the groups attribute.
        // Claim must be a string or string array claim.
        // Expression must produce a string or string array value.
        // "", [], missing, and null values are treated as having no groups.
        // TODO: investigate if you could make a single expression to construct groups from multiple claims. If not, maybe []PrefixedClaimOrExpression?
        // For input claim:
        // {
        //     "claims": {
        //         "roles":"foo,bar",
        //         "other_roles":"baz,qux"
        //         "is_admin": true
        //     }
        // }
        // To concatenate lists:
        //     claims.roles.split(",") + claims.other_roles.split(",")
        // Constructing single item list and concatenating lists:
        //     claims.roles.split(",") + ["hardcoded_group"]
        //     claims.roles.split(",") + (claims.is_admin ? ["admin"]:[])
        // Type check and wrap in a list if needed:
        //     (type(claims.string_or_list_claim) == string ? [claims.string_or_list_claim] : claims.string_or_list_claim) + ["hardcoded_group"]
        // +optional
        Groups PrefixedClaimOrExpression `json:"groups,omitempty"`
        // uid represents an option for the uid attribute.
        // Claim must be a singular string claim.
        // Expression must produce a string value.
        // TODO: this is net new, should it just be expression?
        // +optional
        UID ClaimOrExpression `json:"uid,omitempty"`
        // extra represents an option for the extra attribute.
        //
        // # hard-coded extra key/value
        // - key: "foo"
        //   valueExpression: "bar"
        //
        // hard-coded key, value copying claim value
        // - key: "foo"
        //   valueExpression: "claims.some_claim"
        //
        // hard-coded key, value derived from claim value
        // - key: "admin"
        //   valueExpression: '(has(claims.is_admin) && claims.is_admin) ? "true":""'
        //
        // If multiple mappings have the same key, the result will be a concatenation of all values
        // with the order preserved.
        // If the value is empty, the extra mapping will not be present.
        //
        // possible future way to pull multiple extra values out via expression.
        // TODO: confirm cel comprehensions/mapping is powerful enough to transform
        // the input claims into a filtered / transformed map[string][]string output):
        // # mutually exclusive with key/valueExpression
        //     keyAndValueExpression: '{"key":"string-value", "key2": ["value1","value2"]}'
        //
        // +optional
        Extra []ExtraMapping `json:"extra,omitempty"`
    }

    type ExtraMapping struct {
        // key is a string to use as the extra attribute key.
        Key string `json:"key"`
        // valueExpression is a CEL expression to extract extra attribute value.
        // valueExpression must produce a string or string array value.
        // "", [], and null values are treated as the extra mapping not being present.
        // Empty string values contained within a string array are filtered out.
        ValueExpression string `json:"valueExpression"`
    }

    type ClaimOrExpression struct {
        // claim is the JWT claim to use.
        // Either claim or expression must be set.
        // +optional
        Claim string `json:"claim"`

        // TODO: think about what happens if the claim is absent or the wrong type
        Expression string `json:"expression"`
    }


    type PrefixedClaimOrExpression struct {
        // claim is the JWT claim to use.
        // Either claim or expression must be set.
        // +optional
        Claim string `json:"claim"`
        // prefix is prepended to claim to prevent clashes with existing names.
        // Mutually exclusive with expression.
        // +optional
        Prefix string `json:"prefix"`

        // expression represents the expression which will be evaluated by CEL.
        // Must produce a string. CEL expressions have access to the contents of the token claims for claimValidationRules and claimMappings, userInfo for userInfoValidationRules. Documentation on CEL: https://kubernetes.io/docs/reference/using-api/cel/
        // Either claim or expression must be set.
        // +optional
        Expression string `json:"expression"`
    }
    ```

    The example of mapping user info attributes:

    ```yaml
    claimMappings:
      username:
        expression: 'claims.username + ":external-user"'
      groups:
        expression: 'claims.roles.split(",")'
      uid:
        claim: 'sub'
      extra:
      - key: '"client_name"'
        value: 'claims.aud'
    ```

    For the token with the following claims:

    ```json
    {
      "sub": "119abc",
      "aud": "kubernetes",
      "username": "jane_doe",
      "roles": "admin,user",
      ...
    }
    ```

    The following user info attributes will be extracted:

    ```yaml
    username: jane_doe:external-user
    uid: "119abc"
    groups: ["admin", "user"]
    extra:
      client_name: kubernetes
    ```

    For distributed claims:

    ```json
        claims = {
          "foo":"bar",
          "foo.bar": "...",
          "true": "...",
          "_claim_names": {
            "groups": "group_source"
           },
           "_claim_sources": {
            "group_source": {"endpoint": "https://example.com/claim_source"}
           }
        }
    ```

    - For claim names containing `.`, we can reference using `claims["foo.bar"]`
    - TODO: can we implement a CEL type resolver so that a cel expression `claims.foo` gets resolved via a distributed claim the first time it is used?
       - this seems likely and preferable so we only resolve the things we need (in case an early validation rule fails and short-circuits).

### CEL

* CEL runtime should be compiled only once if structured authentication config option is enabled.
* There will be a maximum allowed CEL expression cost per authenticator (no limit on total authenticators is required due to the issuer uniqueness requirement).
* One variable will be available to use in `claimValidationRules` and `claimMappings`:
  * `claims` for JWT claims (payload)
* One variable will be available to use in `userInfoValidationRules`:
  * `userInfo` with the same schema as [authentication.k8s.io/v1, Kind=UserInfo](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#userinfo-v1-authentication-k8s-io)
* The standard Kubernetes CEL environment, including extension libraries, will be used.
  * Current environment:
    * [Extension libraries](https://github.com/kubernetes/kubernetes/blob/5fe3563ad7e04d5470368aa821f42f131d3bd8fc/staging/src/k8s.io/apiserver/pkg/cel/library/libraries.go#L26)
    * [Base environment](https://github.com/kubernetes/kubernetes/blob/5fe3563ad7e04d5470368aa821f42f131d3bd8fc/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel/compilation.go#L83)
  * The encoding library needs to be added to the environment since it's currently not used. Doing so will help keep CEL consistent across the API.
* Benchmarks are required to see how different CEL expressions affects authentication time.
  * There will be a upper bound of 5s for the CEL expression evaluation.
* Caching will be used to prevent having to execute the CEL expressions on every request.
    - TODO decide what the behavior of the token cache will be on config reload
    - TODO should the token expiration cache know about the `exp` field instead of hard coding `10` seconds?
      this requires awareness of key rotation to implement safely
* TODO: decide how to safe guard access to fields that might not exist or stop existing at any moment.
  * Using `has()` to guard access to fields.
  * Could we do some kind of defaulting for fields that don't exist?

> Notes from PR review (jpbetz):
>
> You can pass a context to CEL and cancel runtime evaluation if the context is canceled. This causes the CEL expression to halt execution promptly and evaluate to an error.
> You can also put a runtime limit (measured in abstract cost units that are hardware and wall clock independent) on CEL expressions to bound running time.
> (There is also a way to set a limit for the estimated cost, which is computed statically on compiled CEL programs if you know the worst case size of the input data, but this might be overkill for this feature)

### Flags

To use this feature, the `--authentication-config` flag must be set to the configuration file.  This flag
is mutually exclusive with all existing `--oidc-*` flags.  The API server will attempt to re-read this file
every minute.  If the hash of the file contents is unchanged, no action will be taken.  Otherwise, the API
server will validate the config.  If it is invalid, no action will be taken and the previous valid config
will remain active.  Otherwise, the new config will become active (via an atomic pointer swap).

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes necessary
to implement this enhancement.

##### Prerequisite testing updates

[kubernetes#110782](https://github.com/kubernetes/kubernetes/issues/110782) tracks that lack of
test coverage for OIDC, and [kubernetes#115122](https://github.com/kubernetes/kubernetes/pull/115122)
attempts to rectify that gap.

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

- `k8s.io/kubernetes/pkg/kubeapiserver/options/authentication.go`: `2023-06-06` - `74.7`
- `k8s.io/kubernetes/pkg/kubeapiserver/authenticator/config.go`: `2023-06-06` - `0.0`
- `k8s.io/apiserver/plugin/pkg/authenticator/token/oidc`: `2023-06-06` - `84`

Note that as of 2023-06-06, the existing OIDC authenticator has no integration or e2e tests.

Unit tests will be expanded to cover the new feature set of this KEP:

- Structured config (including validation)
- CEL based expressions
- Multiple client ID support
- Discovery URL overrides
- Automatic config reload
- Multiple authenticators

##### Integration tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html
-->

Integration tests will cover parts of the new feature set as well:

- CEL based expressions
- Automatic config reload
- Multiple authenticators


##### e2e tests

<!--
This question should be filled when targeting a release.
For Alpha, describe what tests will be added to ensure proper quality of the enhancement.

For Beta and GA, add links to added tests together with links to k8s-triage for those tests:
https://storage.googleapis.com/k8s-triage/index.html

We expect no non-infra related flakes in the last month as a GA graduation criteria.
-->

e2e tests will focus on testing a broad set of features together with "real" OIDC
providers such as Okta, Azure AD, etc:

- CEL based expressions
- Multiple client ID support
- Discovery URL overrides
- Multiple authenticators

### Graduation Criteria

#### Alpha

- Feature implemented behind a feature flag
- Unit tests to validate CEL semantics
- Unit tests for config validation
- Initial integration tests completed and enabled

#### Beta

- Gather feedback
- Complete benchmarks
- Add metrics
- Initial e2e test with an external provider completed and enabled

#### GA

- Add a full documentation with examples for the most popular providers, e.g., Okta, Dex, Auth0
- Migration guide
- Deprecation warnings for non-structured OIDC provider configuration

#### Deprecation

kube-apiserver `--oidc-*` flags require deprecation warnings on the stable release of the feature.

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

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Feature gate
  - Feature gate name: `StructuredAuthenticationConfiguration`
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

Yes.  Note that if the `--oidc-*` flags were previously in use, they must be restored for OIDC authentication to function correctly.

###### What happens if we reenable the feature if it was previously rolled back?

No impact (generally speaking, authentication does not cause persisted state in the cluster).

###### Are there any tests for feature enablement/disablement?

Feature enablement/disablement unit/integration tests will be added

> An example test could be: unit test that demonstrates that when the featuregate is false, the validation function on the Options type reports a failure when the flag is set.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

It cannot fail until a bug in kube-apiserver connected to parsing structured config file occurs.

Possible consequences are:
* A cluster administrator rolls out the feature with the addition of some validation rules that may allow access to previously restricted users.
* Other cluster components can depend on claim validations. Rolling back would mean losing validation functionality.
* If the cluster admin fails to restore any previously in-use `--oidc-*` flags on a rollback, OIDC authentication will not function.

###### What specific metrics should inform a rollback?

TODO

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

TODO

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

TODO

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
* By checking the metrics emitted by the kube-apiserver.

###### How can someone using this feature know that it is working for their instance?

<!--
For instance, if this is a pod-related feature, it should be possible to determine if the feature is functioning properly
for each individual pod.
Pick one more of these and delete the rest.
Please describe all items visible to end users below with sufficient detail so that they can verify correct enablement
and operation of this feature.
Recall that end users cannot usually observe component logs or access metrics.
-->

Metrics

- Last successful load of the file
- Last time keys were fetched (would be per issuer)
- JWT authenticator latency metrics
- Authentication metrics should include which JWT authenticator was used

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

SLOs for actual requests should not change in any way compared to the flag-based OIDC configuration.

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

No new API calls will be made.

However, the authenticator does make network requests per OIDC provider to fetch public keys.

###### Will enabling / using this feature result in introducing new API types?

Yes. Group `apiserver.config.k8s.io`, object `AuthenticationConfiguration`.

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

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

<!--
Focus not just on happy cases, but primarily on more pathological cases
(e.g. probes taking a minute instead of milliseconds, failed pods consuming resources, etc.).
If any of the resources can be exhausted, how this is mitigated with the existing limits
(e.g. pods per node) or new limits added by this KEP?

Are there any tests that were run/should be run to understand performance characteristics better
and validate the declared limits?
-->

TBA.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature is a part of authentication flow. It does not rely on etcd, but strictly connected to the kube-apiserver.

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

- This feature will be the first adoption of using CEL for a config file.

## Alternatives

- Invest more into external software like Dex and officially make it the OIDC provider socket.
- Do not add any more OIDC provider customization to Kubernetes.
  Instead, add more guides and docs about customizing Kubernetes authentication with external software.

TODO: describe why we removed the skip validations around audience and issuer, as well as why we never
wanted to support skipping exp/iat/nbf.

## Infrastructure Needed

Tests against real infra like Azure AD, Okta, etc.

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
