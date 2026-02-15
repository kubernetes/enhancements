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
    - [Pre-GA follow-up](#pre-ga-follow-up)
    - [GA](#ga)
    - [Possible future work](#possible-future-work)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Questions](#questions)
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

## Motivation

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

The main part of this proposal is a configuration file. It contains an array of providers:

```yaml
apiVersion: apiserver.config.k8s.io/v1beta1
kind: AuthenticationConfiguration
jwt:
- issuer:
    url: https://example.com
    audiences:
    - my-app
    - other-app
    audienceMatchPolicy: MatchAny
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
  userValidationRules:
  - expression: "!user.username.startsWith('system:')"
    message: username cannot used reserved system: prefix
  - expression: "user.groups.all(group, !group.startsWith('system:'))"
    message: groups cannot used reserved system: prefix
```

The minimum valid JWT payload must contain the following claims:

```yaml
{
  "iss": "https://example.com",   // must match the issuer.url
  "aud": ["my-app"],              // at least one of the entries in issuer.audiences must match the "aud" claim in presented JWTs.
  "exp": 1234567890,              // token expiration as Unix time (the number of seconds elapsed since January 1, 1970 UTC)
  "<username-claim>": "user"      // this is the username claim configured in the claimMappings.username.claim or claimMappings.username.expression
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
4. User validation based on `userValidationRules`

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

    // userValidationRules are rules that are applied to final userInfo before completing authentication.
    // These allow invariants to be applied to incoming identities such as preventing the
    // use of the system: prefix that is commonly used by Kubernetes components.
    // +optional
    UserValidationRules []UserValidationRule `json:"userValidationRules,omitempty"`
}
```

1. `Issuer` - is a section for external provider specific settings, e.g., OIDC discovery URL.

```go
    type Issuer struct {
        // url points to the issuer URL in a format https://url or https://url/path.
        // This must match the "iss" claim in the presented JWT, and the issuer returned from discovery.
        // Same value as the --oidc-issuer-url flag.
        // Discovery information is fetched from "{url}/.well-known/openid-configuration" unless overridden by discoveryURL.
        // Required to be unique across all JWT authenticators.
        // Note that egress selection configuration is not used for this network connection.
        // TODO: decide if we want to support egress selection configuration and how to do so.
        URL string `json:"url"`

       	// discoveryURL, if specified, overrides the URL used to fetch discovery
        // information instead of using "{url}/.well-known/openid-configuration".
        // The exact value specified is used, so "/.well-known/openid-configuration"
        // must be included in discoveryURL if needed.
        //
        // The "issuer" field in the fetched discovery information must match the "issuer.url" field
        // in the AuthenticationConfiguration and will be used to validate the "iss" claim in the presented JWT.
        // This is for scenarios where the well-known and jwks endpoints are hosted at a different
        // location than the issuer (such as locally in the cluster).
        //
        // Example:
        // A discovery url that is exposed using kubernetes service 'oidc' in namespace 'oidc-namespace'
        // and discovery information is available at '/.well-known/openid-configuration'.
        // discoveryURL: "https://oidc.oidc-namespace/.well-known/openid-configuration"
        // certificateAuthority is used to verify the TLS connection and the hostname on the leaf certificate
        // must be set to 'oidc.oidc-namespace'.
        //
        // curl https://oidc.oidc-namespace/.well-known/openid-configuration (.discoveryURL field)
        // {
        //     issuer: "https://oidc.example.com" (.url field)
        // }
        //
        // discoveryURL must be different from url.
        // Required to be unique across all JWT authenticators.
        // Note that egress selection configuration is not used for this network connection.
        // TODO: decide if we want to support egress selection configuration and how to do so.
        // +optional
        DiscoveryURL string `json:"discoveryURL,omitempty"`

        // certificateAuthority contains PEM-encoded certificate authority certificates
        // used to validate the connection when fetching discovery information.
        // If unset, the system verifier is used.
        // Same value as the content of the file referenced by the --oidc-ca-file flag.
        // +optional
        CertificateAuthority string `json:"certificateAuthority,omitempty"`

        // audiences is the set of acceptable audiences the JWT must be issued to.
        // Same value as the --oidc-client-id flag (though this field supports an array).
        // Required to be non-empty.
        Audiences []string `json:"audiences,omitempty"`

        // audienceMatchPolicy defines how the "audiences" field is used to match the "aud" claim in the presented JWT.
        // Allowed values are:
        // 1. "MatchAny" when multiple audiences are specified and
        // 2. empty (or unset) or "MatchAny" when a single audience is specified.
        //
        // - MatchAny: the "aud" claim in the presented JWT must match at least one of the entries in the "audiences" field.
        // For example, if "audiences" is ["foo", "bar"], the "aud" claim in the presented JWT must contain either "foo" or "bar" (and may contain both).
        //
        // - "": The match policy can be empty (or unset) when a single audience is specified in the "audiences" field. The "aud" claim in the presented JWT must contain the single audience (and may contain others).
        //
        // For more nuanced audience validation, use claimValidationRules.
        //   example: claimValidationRule[].expression: 'sets.equivalent(claims.aud, ["bar", "foo", "baz"])' to require an exact match.
        AudienceMatchPolicy AudienceMatchPolicy `json:"audienceMatchPolicy"`
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
        // If username.expression uses 'claims.email', then 'claims.email_verified' must be used in
        // username.expression or extra[*].valueExpression or claimValidationRules[*].expression.
        // An example claim validation rule expression that matches the validation automatically
        // applied when username.claim is set to 'email' is 'claims.?email_verified.orValue(true)'.
        // +required
        Username PrefixedClaimOrExpression `json:"username"`
        // groups represents an option for the groups attribute.
        // Claim must be a string or string array claim.
        // Expression must produce a string or string array value.
        // "", [], missing, and null values are treated as having no groups.
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
        // If the value is empty, the extra mapping will not be present.
        //
        // possible future way to pull multiple extra values out via expression.
        // # mutually exclusive with key/valueExpression
        //     keyAndValueExpression: '{"key":"string-value", "key2": ["value1","value2"]}'
        //
        // +optional
        Extra []ExtraMapping `json:"extra,omitempty"`
    }

    type ExtraMapping struct {
        // key is a string to use as the extra attribute key.
        // key must be a domain-prefix path (e.g. example.org/foo). All characters before the first "/" must be a valid
        // subdomain as defined by RFC 1123. All characters trailing the first "/" must
        // be valid HTTP Path characters as defined by RFC 3986.
        // key must be lowercase.
        // key must be unique across all extra mappings.
        // +required
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

        // expression represents the expression which will be evaluated by CEL.
        // Either claim or expression must be set.
        // +optional
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
        Prefix *string `json:"prefix"`

        // expression represents the expression which will be evaluated by CEL.
        // Must produce a string. CEL expressions have access to the contents of the token claims for claimValidationRules and claimMappings, user for userValidationRules. Documentation on CEL: https://kubernetes.io/docs/reference/using-api/cel/
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
      client_name: ["kubernetes"]
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

### CEL

* CEL runtime should be compiled only once if structured authentication config option is enabled.
* The API server trusts the CEL expressions provided in the authentication configuration file to be safe and cost-effective. As a result, the API server will not set a maximum CEL expression cost per authenticator.
* One variable will be available to use in `claimValidationRules` and `claimMappings`:
  * `claims` for JWT claims (payload)
* One variable will be available to use in `userValidationRules`:
  * `user` with the same schema as [authentication.k8s.io/v1, Kind=UserInfo](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.27/#userinfo-v1-authentication-k8s-io)
* The standard Kubernetes CEL environment, including extension libraries, will be used.
  * Current environment:
    * [Extension libraries](https://github.com/kubernetes/kubernetes/blob/5fe3563ad7e04d5470368aa821f42f131d3bd8fc/staging/src/k8s.io/apiserver/pkg/cel/library/libraries.go#L26)
    * [Base environment](https://github.com/kubernetes/kubernetes/blob/5fe3563ad7e04d5470368aa821f42f131d3bd8fc/staging/src/k8s.io/apiextensions-apiserver/pkg/apiserver/schema/cel/compilation.go#L83)
  * The encoding library needs to be added to the environment since it's currently not used. Doing so will help keep CEL consistent across the API.

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

- Complete benchmarks
  -  Benchmarks are required to see how different CEL expressions affects authentication time.
     -  There will be a upper bound of 5s for the CEL expression evaluation. We will use the `apiserver_authentication_latency_seconds` metric to monitor this.
- Add metrics
- Support > 1 JWT authenticator and > 1 audiences
- Enable automatic reload of the configuration
  - If there is a failure in the new configuration, the old configuration will remain active.
  - Typo in issuer URL will not be detected since an issuer is explicitly allowed to be offline when an API server is starting up to allow for self-hosted IDPs.
- Add tests
  - Tests for automatic reload of the configuration

#### Pre-GA follow-up

- With automatic reload of configuration, typo in issuer URL will not be detected since an issuer is explicitly allowed to be offline when an API server is starting up to allow for self-hosted IDPs. We need
  to come-up with an approach to make this more robust.

#### GA

- Gather feedback
  - Confirm with [Kubespray](https://github.com/kubernetes-sigs/kubespray) that the new functionality addresses their use case.
    - The Kubespray team is integrating Structured Authentication in this [PR](https://github.com/kubernetes-sigs/kubespray/pull/11841).
  - Confirm with [Gardener](https://gardener.cloud/) that the new functionality addresses their use case.
    - The Gardener team has integrated Structured Authentication in this [PR](https://github.com/gardener/gardener/pull/10244).
  - Confirm with [SPIFFE](https://spiffe.io/) that the new functionality addresses their use case.
    - The SPIFFE team recommends Structured Authentication in their [README](https://github.com/spiffe/k8s-spiffe-workload-jwt-exec-auth?tab=readme-ov-file#setup-the-kubernetes-cluster-auth).
- Migration guide
  - This is already done in the [beta blog post](https://kubernetes.io/blog/2024/04/25/structured-authentication-moves-to-beta/#migration-from-command-line-arguments-to-configuration-file).

#### Possible future work

- These tasks are not required for GA, but they are important to consider for the future of the feature. They will be added behind feature gates and will be fully backward compatible.
  - Get distributed claims working with CEL
    - can we implement a CEL type resolver so that a cel expression `claims.foo` gets resolved via a distributed claim the first time it is used?
    - this seems likely and preferable so we only resolve the things we need (in case an early validation rule fails and short-circuits).
  - Egress selection configuration
  - Better caching to support distributed claims in a more performant way
    - Should the token expiration cache know about the `exp` field instead of hard coding `10` seconds?
- Remove 64 jwt authenticators limit
  - implementation detail: we should probably parse the `iss` claim out once.
  - Currently the structure of authenticators is a list of authenticators, but we could change it to a map of authenticators with issuer as the key.
- Audit annotations set on claim/user validation failure

### Upgrade / Downgrade Strategy

While the feature is in Alpha, there is no change if cluster administrators want to
keep on using command line flags.

When the feature goes to Beta/GA or the cluster administrators want to configure
jwt authenticators using the configuration file, they need to make sure:

1. The configuration file is available on the API server and the `--authentication-config` flag is set.
2. No `--oidc-*` flags are set.

When downgrading from the structured configuration to the flag-based configuration, they need to
unset the `--authentication-config` flag and restore the `--oidc-*` flags to configure the JWT authenticator.

### Version Skew Strategy

This is an API server only change and does not affect other components. If the API server is
not the minimum required version (v1.29), the feature will not be available.

## Questions

- should we have any revocation mechanism?
  => use revocation endpoint if it is in the discovery document?
  => related issue https://github.com/kubernetes/kubernetes/issues/71151

> We don't have any plans to add revocation at this time. Because of this the docs will be updated to make sure the tokens are short-lived as they are not revocable.

> While full token revocation is not supported, it is possible to approximate revocation by writing user info validation rules (e.g., via CEL) based on a unique identifier in the token, such as the jti claim (if present). Even without a jti, any claim that uniquely identifies the token can be used to simulate revocation by checking it against a denylist or revocation list. However, we still recommend using short-lived tokens as managing revocation this way can become complex and hard to scale.

Example of a revocation rule using the jti claim:

```yaml
userValidationRules:
- expression: "!(user.extra[?'authentication.kubernetes.io/credential-id'][0].orValue('') in ["JTI=e28ed49-2e11-4280-9ec5-bc3d1d84661a"])",
  message:    "credential id is revoked",
```

- decide what error should be returned if CEL eval fails at runtime
  `500 Internal Sever Error` seem appropriate but authentication can only do `401`

> We always return `401 Unathorized` and log the error message. This is consistent with the existing OIDC authenticator behavior.

- distributed claims with fancier resolution requirements (such as access tokens as input)
  - This will be considered for getting distributed claims working with CEL

> This is suggesting a scenario where your distributed claims require an access token, but that token isn’t embedded in the ID token. Instead, the cluster admin configuring the system would somehow provide the access token. For example, there could be a client credentials plugin at the API server that fetches or refreshes an access token and makes it available for distributed claim fetching. We are not going to support this approach.

- are `iat` and `nbf` required?

> We have already documented the minimum required JWT payload. The `iat` and `nbf` claims are not required.

- is `sub` required or is the requirement to just have some username field?

> We have already documented the minimum required JWT payload. The `sub` claim is not required, but the username claim must be present.

- confirm cel comprehensions/mapping is powerful enough to transform the input claims into a filtered / transformed `map[string][]string` output for extra

> We don't need to this because we changed the structure of our configuration of `ExtraMapping` to be a list of key and expression. So we don't need one CEL comprehension to do this anymore.

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

- For CEL expressions, do we want to safe guard access to fields that might not exist or stop existing at any moment?
  - Using `has()` to guard access to fields.
  - Could we do some kind of defaulting for fields that don't exist?

> We are not planning to do this. Users can use `OptionalTypes` in CEL to check if a field exists or not and to provide default values if they don't.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- Feature gate
  - Feature gate name: `StructuredAuthenticationConfiguration`
  - Components depending on the feature gate:
    - kube-apiserver

**Alpha**

```go
FeatureSpec{
  Default: false,
  LockToDefault: false,
  PreRelease: featuregate.Alpha,
}
```

**Beta**

```go
FeatureSpec{
  Default: true,
  LockToDefault: false,
  PreRelease: featuregate.Beta,
}
```

**Stable**

```go
FeatureSpec{
  Default: true,
  LockToDefault: true,
  PreRelease: featuregate.GA,
}
```

###### Does enabling the feature change any default behavior?

No. `AuthenticationConfiguration`is new in the v1.29 release. Furthermore, even with the feature enabled by default, the user needs to
explicitly set the `--authentication-config` flag to use the structured configuration.

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

- `apiserver_authentication_config_controller_automatic_reload_failures_total` - This metric will be incremented when the API server fails to reload the configuration file.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

This will be covered by integration tests.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

New metrics:

- `apiserver_authentication_config_controller_automatic_reload_last_timestamp_seconds` - This metric will be updated every time the API server reloads the configuration file.
- `apiserver_authentication_config_controller_automatic_reloads_total` - This metric will be incremented every time the API server reloads the configuration file partitioned by status (success/failure).
- `apiserver_authentication_config_controller_automatic_reload_last_config_hash` - This metric will be set to the hash of the loaded configuration file after a successful reload.
- `apiserver_authentication_jwt_authenticator_latency_seconds` - This metric will be used to monitor the time it takes to Authenticate token. This will only be set for token authentication requests for matching issuer.
- `apiserver_authentication_jwks_fetch_last_timestamp_seconds` - This metric will be updated every time the API server makes a request to the JWKS endpoint.
- `apiserver_authentication_jwks_fetch_last_keyset_hash` - This metric will be set to the hash of the keyset fetched from the JWKS endpoint after successfully fetching the keyset.
  - We will use https://pkg.go.dev/hash/fnv#New64 to hash the keyset.
- `apiserver_authentication_jwt_authenticator_provider_status_timestamp_seconds` - This metric will indicate the status of the JWT authenticator provider.

###### How can an operator determine if the feature is in use by workloads?

* There will be a corresponding message in kube-apiserver logs.
* By checking the kube-apiserver flags.
* By checking the metrics emitted by the kube-apiserver.

###### How can someone using this feature know that it is working for their instance?

Metrics

- Last successful load of the file
- Last time keys were fetched (would be per issuer)
- JWT authenticator latency metrics
- Authentication metrics should include which JWT authenticator was used

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

SLOs for actual requests should not change in any way compared to the flag-based OIDC configuration.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

None other than what we are planning to add as part of the feature.

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

No. There would be very minimal addition to the memory used by the API Server and
number of log entries written to the disk.

We do plan on watching config changes and dynamically updating the authenticators. This involves re-parsing the CEL expressions
and re-fetching public keys. This is expected to be a low frequency operation. We will perform benchmarks for this.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

No.

We do plan on watching config changes and dynamically updating the authenticators. This is expected to be a low frequency operation
and will be done carefully to avoid any resource exhaustion.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

This feature is a part of authentication flow. It does not rely on etcd, but strictly connected to the kube-apiserver.

###### What are other known failure modes?

The same failure modes and diagnostics as for the non-structured OIDC provider are applicable here.

###### What steps should be taken if SLOs are not being met to determine the problem?

The same steps as for the flag-based OIDC provider are applicable here.

## Implementation History

- [x] 2022-06-22 - Provisional KEP introduced
- [x] 2023-06-13 - KEP Accepted as implementable
- [x] 2023-09-05 - Alpha implementation merged https://github.com/kubernetes/kubernetes/pull/119142
- [x] 2023-10-31 - CEL support for authentication configuration merged https://github.com/kubernetes/kubernetes/pull/121078 
- [x] 2023-12-13 - First release (1.29) when feature available
- [x] 2024-01-31 - Targeting beta in 1.30

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
