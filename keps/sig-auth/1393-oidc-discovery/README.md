# Service Account signing key retrieval

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [x] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

The Kubernetes API server generates (signs) JSON Web Tokens that are meant to
authenticate Kubernetes service accounts (KSA). The API server can indeed authenticate
these tokens, but in general, no other system can: there is no standard way to
get the public portion of the keypair used to sign KSA tokens. Systems ("relying
parties") that want to authenticate KSA tokens must either send them back to the
API server (via `TokenReview`) or use some provider-specific method to get the
authentication key.

If a relying party could obtain trusted metadata about the service account token
provider, in particular the issuer (`iss`) value and the public key(s) used,
then the relying party could authenticate tokens without putting proportionate
load on the API server. This would allow KSA tokens to be used as a general
authentication mechanism, including to services outside the cluster or in other
clusters.

[OpenID Connect](https://openid.net/connect/) defines a
[discovery](https://openid.net/specs/openid-connect-discovery-1_0.html)
mechanism that, given an issuer URL, allows a client to discover the rest of the
issuer metadata, including the key set. Providing an OIDC-compatible discovery
document would allow flexibility in how relying parties authenticate KSA tokens;
they can use existing OIDC authenticators in their language/framework of choice,
without Kubernetes-specific or provider-specific logic.

## Motivation

Kubernetes workloads can consume a variety of services from a variety of
producers. They have a native identity (KSA), presented in a widely-compatible
format (JWT); but only an API server can authenticate the KSA token, since only
the API server has access to the public key verifying the signature. If services
want to authenticate workloads using KSAs, today, the API server must serve
every authentication request (i.e. `TokenReview`).

When authenticating across clusters, i.e. from within a cluster to somewhere
else, credential management must use a separate system. Someone has to provision
an identity; provision credentials; grant the workload access to the
credentials; and consider:

-   Did I delete ephemeral traces of the credential (e.g. files on my local
    disk)?
-   How securely is the credential stored? Is the storage system hardened? Are
    the ACLs restricted?
-   How much damage can be caused by an exploit of the workload? Could that
    compromise a credential with an extended validity period?
-   How often do the keys expire? How often do I rotate them?

If services (other than the API server) could authenticate KSA tokens directly:

-   The API server wouldn't have to scale with the data-plane (i.e.
    authentication) load.
-   Workloads could use native credentials to authenticate to services outside
    of the cluster.

### Goals

-   Allow (authorized) systems to discover the information they need to
    authenticate KSA tokens.
-   Attempt compatibility with OIDC: common libraries that authenticate OIDC
    tokens should be able to authenticate KSA tokens.
-   Support authentication when the API server is not directly reachable by the
    relying party.
    -   e.g.: a cloud-based service authenticating an API server that doesn't
        have a public Internet address.

Note that the API server has a very different flow from OIDC with respect to
generating tokens. As such, our goal is OIDC *compatibility*.

### Non-Goals

Our goal is *not* OIDC *compliance*.

We aren't trying to make the KSA token process fully compliant with OIDC
specifications. OIDC includes flows for token acquisition, token exchange,
getting user data out of tokens, etc. But we are primarily interested in the parts
relevant to *relying parties*, i.e. token authentication. We don't need to do
something `REQUIRED` just because the spec says `REQUIRED`; but we may need to
do it if relying parties expect that field. This may mean that our
implementation doesn't fully comply with the spec; e.g. we might skip
`token_endpoint` in the
[discovery document](https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata),
even without supporting the
[Implicit Flow](https://openid.net/specs/openid-connect-core-1_0.html#ImplicitFlowAuth).
Ensuring compatibility with existing implementations, for example that libraries
can validate tokens without erroring out during the discovery fetch/parse,
is critical, and is part of the Alpha to Beta graduation criteria. If we
determine that it is impossible to broadly meet our compatibility goals with the
current design, then we will revisit serving a complete discovery doc with
placeholder values.

We will not define new cryptographic or authentication protocols, or change the
format of KSA tokens, how they're generated, or how they're issued.

## Proposal

A non-resource API will be added to the apiserver that will expose an
[OIDC discovery](https://openid.net/specs/openid-connect-discovery-1_0.html)-
compatible document. This document will reflect the issuer value and provide
some additional information about the tokens issued.

In addition, the API server will serve a JWKS consisting of the public keys from
`--service-account-key-file` and `--service-account-signing-key`, i.e. all the
public keys used for valid KSA tokens. The OIDC discovery document will point to
the JWKS path as the `jwks_uri`.

Finally, it will be possible for users to override the `jwks_uri` via a new
`--service-account-jwks-uri` flag. We do attempt to construct a default JWKS URI
from the API server's external address, but without this additional option, it
isn't possible to correctly configure the server in some cases. For example,
even given an issuer URL, we don't necessarily know the JWKS path, because it is
not part of the specification and the issuer URL may not point directly to an
API server (the user may choose an alternative issuer URL if their cluster does
not serve on the public internet, as long as the cluster is configured to use
that URL for issuer claims).

For example, if the API server is configured with the `--service-account-issuer`
value `https://cluster.example.com`, which is also the API server root, and
`--service-account-jwks-uri` value `https://cluster.example.com/openid/v1/jwks`
(the cluster serves the JWKS at the `/openid/v1/jwks` path by default), the
API server could expose the following configuration. Note that if the JWKS
URI overide is not provided, the API server will report it as relative to its
external address. For example: `https://192.168.10.4:443/openid/v1/jwks`. Note
also this example intentionally omits the `authorization_endpoint`
field, as it is not necessary for token verification flows.

```
> GET https://cluster.example.com/.well-known/openid-configuration
{
  "issuer": "https://cluster.example.com",
  "jwks_uri": "https://cluster.example.com/openid/v1/jwks",
  "response_types_supported": [
    "id_token"
  ],
  "subject_types_supported": [
    "public"
  ],
  "id_token_signing_alg_values_supported": [
    "RS256",
    "ES256"
  ],
}
> GET /openid/v1/jwks
{
  "keys": [
    {
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "kid": "ccab4acb107920dc284c96c6205b313270672039",
      "n": "wWGfvdCEjJJy7CQpGcTq6GghmqWLi9H4SNHNTtFMfIDPsv-aWj1e_iSO22505BlC9UcL9LvlSyVH8HmQUy5916YNqxCbhPFPabBAv0a-CpVuzbbyhpDNP3RkRIJgxlzPDh_dB11cbPTQ3yz0A0JARX3QNZfIQ8LFiZ1vh0iZAIm-I3eZeI4QZigImNDviZstSoHB2Ny1tsRmpZn-neYZCxYq717buFctnCVvot4iCwcQpeaGdniqYNDxzN4KlQwwDeCVJm-K0rG9nkiqZ_rq8SgCxi_l7NyF2ZURNTTzZyDwYfBR7jZUhbmjxIDoDZalsa1Tzzy1vzqBfxkFD5Z03w",
      "e": "AQAB"
    },
    {
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "kid": "8770f6158b125040b98e50a1e0e6790ff2f9ea09",
      "n": "vpgsIIPqDO3A3dEuRCIZvQinyfME0BjH_RbeyLAAvrQx-Sv08ryFPjplqxm5t9mC0yULrhOmaIZCVfIuYn5n_dOblZNhpIpoy89bP0qNwV7gxsNv-0Tdu9nj4ymxeoaby6SFiv_c8P2JZ0CSqif_qXgj-o0TqU20FEv1hkizzQWDzFsKZ__IABAkdKfpGqQTOBTylFG9HFLV1tdh9AAdhVRf40982rksaOSDWvN_sfxiz6midGPgG0OOnMnwKAW-3BBNNd_uUrD9baSXZPFA8zo9dlkhQhfrFgg_U6ke4M5DPyFiPKOVitBzpL1Kth_patVZvnBGXtq2frbReF-6pw",
      "e": "AQAB"
    },
    {
      "kty": "RSA",
      "alg": "RS256",
      "use": "sig",
      "kid": "68241231bbf0df8f9123d018cf9e601e2aa3673a",
      "n": "rHozcxeim9flTWQxqC1ObpGP0EjpkUHVHpHNX8WGHHnMcVi63_9PaHn2cJeFuPF9qkI1dMPXeoX0m33N0tgM9-KOSmTg1oGbyJGoUYMFI-A7tdxoobb91LGjeNWJC0la0gLOGPcQ6zQLEU5RGftCZT0wElxMuwEH7FZoVBn5i8Ddvc2ADd4bFW0f_FckwFYN1rIU1uLf6coku_1xBfae3b_JiBq38QOGXPdPgxfPzmJEvIz_LB2WOIcwhl97DY32BQU7l_lNLYz6wMg9HeCKolypPIFEGNxLj1TcuOhwP5-BnSja-PvrB-1FN1JyzlL3nh__uJv8SoKPn0CoBPueWw",
      "e": "AQAB"
    }
  ]
}
```

The API server would treat these as `nonResourceURLs`, and restrict access
appropriately. We will provide a default RBAC `ClusterRole` called
`service-account-issuer-discovery` that provides `GET` access to these
`nonResourceURLs`. To make it easy for in-cluster workloads (via their
service accounts) to consume this info, we will also provide a default
`ClusterRoleBinding` that binds this role to all service accounts (via
the `system:serviceaccounts` group).

Users with certain forms of write access (create pods, create secrets,
create service accounts, etc) can gain access to a service account identity
which would allow them to access this information. This includes the issuer
URL, which is already present in the SA token JWT.  Similarly, SAs can already
gain this same info via introspection of their own token.  Since this discovery
endpoint points to what issued all service account tokens, it seems fitting for
SAs to have this access.

Even though this information is not sensitive, we will *not* provide a
default binding to all *authenticated* and/or *unauthenticated* users.
Such a binding requires further discussion, including ongoing efforts to
harden the unauthenticated API surface area. This leaves the decision of
completely exposing these endpoints up to cluster admins.

### Risks and Mitigations

- Security is being reviewed by @mikedanese and @liggitt.
- This feature exposes public key information that is derived from sensitive
  private keys. It is important that code reviewers pay careful attention
  to the construction of the keysets so that sensitive private keys are not
  exposed.
- This proposal allows Kubernetes identities to be federated into other systems,
  and creates a dependency on the Kubernetes API server to verify these identities.
  This means that if the API server is down, and verification keys are not
  cached by relying parties or an intermediary, workloads may fail to
  authenticate with their dependencies. This can be mitigated by running a high
  availability configuration, or by caching discovery docs and keysets in a
  reliable intermediate location.

## Design Details

### Test Plan

This feature includes unit, integration, and E2E tests:
- Unit tests in `pkg/serviceaccount` to test that the server code constructs
  OIDC responses in the correct format.
- Integration tests in `test/integration/auth/svcacct_test.go` to attempt a
  token verification based on the OIDC flow against an API server, and also
  to verify the expected headers on the responses.
- An E2E test that exercises the full stack by mounting a new-style Kubernetes
  service-account token on a Pod which then attempts to verify its token by
  calling into a third-party library implementation of the OIDC flow
  (github.com/coreos/go-oidc). This helps confirm compatibility with third-party
  implementations.


### Graduation Criteria

This feature does not expose a new K8s style API resource object.
Instead, it provides endpoints for the subset of the OIDC 1.0 spec specified
above. As such, it doesn't quite match up with the
[API doc definition](https://kubernetes.io/docs/concepts/overview/kubernetes-api/#api-versioning)
of versioning. However, we can still treat graduation in terms of
[Maturity levels (`alpha`, `beta`, `stable`)][maturity-levels].

- Zero State to Alpha:
  - KEP is implemented behind a feature gate.
  - Unit, integration, and E2E tests exist, though some tests (e.g. E2E) won't
    automatically run in release-blocking suites.
  - There must be a test that ensures no private keys are emitted via the JWKS endpoint.
- Alpha to Beta:
  - Any fixes or API changes from the Alpha experience are implemented.
  - The feature has been confirmed compatible with several independent
    relying parties by federating K8s identities with multiple top cloud
    providers and ensuring that the most popular OIDC libraries used by relying
    parties are compatible.
  - Test failures are release blocking.
- Beta to Stable:
  - Cluster conformance tests for this feature exist.

[maturity-levels]: https://git.k8s.io/community/contributors/devel/sig-architecture/api_changes.md#alpha-beta-and-stable-versions
[deprecation-policy]: https://kubernetes.io/docs/reference/using-api/deprecation-policy/

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

_This section must be completed when targeting alpha to a release._

* **How can this feature be enabled / disabled in a live cluster?**

  No. This feature is always enabled (post-GA).
  Pre-GA it was possible to disable with the feature gate.

  - [x] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: ServiceAccountIssuerDiscovery
    - Components depending on the feature gate: kube-apiserver
    - Note: This feature is targeted to GA in 1.21, at which point feature gates
      lock to enabled. This means it will not be possible to disable after the
      current dev cycle.

* **Does enabling the feature change any default behavior?**
  No. It adds an entirely new non-resource-url that can be used to discover
  metadata related to the cluster's service account issuer.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**

  No. This feature is always enabled post-GA.
  The only way to roll back is to return to an older K8s version.

  **Describe the consequences on existing workloads (e.g., if this is a runtime
  feature, can it break the existing applications?).**

  Existing applications would have to take a dependency on this feature to
  be broken by it. Thus, enabling the feature for the first time is not a risk
  to existing applications, but disabling it later could be.

* **What happens if we reenable the feature if it was previously rolled back?**

  The feature should continue to work just fine.

* **Are there any tests for feature enablement/disablement?**

  No.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**
  Enablement shouldn't affect any existing workloads. If we broke the feature in
  the future, we would _possibly_ see failures of workloads to authenticate to
  Relying Parties _outside_ the cluster, but in-cluster workload to
  kube-apiserver authentication would still work, since it doesn't rely
  on this path.

* **What specific metrics should inform a rollback?**
  N/A

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**
  The standard upgrade tests would have covered this between alpha and beta,
  when the feature was enabled by default.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

  No.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  Ideally, there would just be usage metrics for all API server endpoints.
  Since we don't currently have that, the next best option would be to examine
  API server logs.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [x] Other (treat as last resort)
    - Details: API server logs, or ability of workloads to authenticate to
      Relying Parties.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  We expect the endpoints to maintain high reliability, with reliability
  matching that of kube-apiserver.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**
  It would be nice to have usage metrics for this endpoint. We haven't added
  them so far because non-resource URLs don't have them by default. This could
  be worth solving in general but a general solution is out of scope for this
  KEP.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**
  It only depends on kube-apiserver being up. If, for example, the issuer is
  configured as https://kubernetes.default.svc, then the corresponding Service
  needs to exist in the cluster as well.


### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**
  Yes.
  - GET `${API_SERVER}/.well-known/openid-configuration`
  - GET `${API_SERVER}/openid/v1/jwks`
  - Note each endpoint serves a response that is pre-rendered when
    kube-apiserver starts up.
  - Originating components: Could be arbitrary. For example:
    - A cluster installer reads these once when configuring identity federation
      with a cloud provider (Low throughput).
    - In-cluster components use this to perform an OIDC discovery flow to
      validate tokens (Medium to High throughput). Note TokenReview is the
      preferred approach in this case.
    - A cluster admin adds additional RBAC to make these endpoints public, and
      points Relying Parties directly at these endpoints (High throughput,
      though RPs _should_ do some caching instead of making calls on every
      token validation).

* **Will enabling / using this feature result in introducing new API types?**
  No new types, just two new non-resource URLs that implement this KEP, as
  described above. There is no new state stored in etcd.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**
  No.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**
  No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**
  No.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**
  This isn't expected, given it's just copying a pre-rendered string into
  the response.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**
  If kube-apiserver is unavailable, this feature is also unavailable. This
  feature is not affected by etcd availability.

* **What are other known failure modes?**
  N/A

* **What steps should be taken if SLOs are not being met to determine the problem?**
- Examine the responses from the above endpoints.
- Examine kube-apiserver logs.
- Examine kube-apiserver configuration related to this KEP.

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

-   2018-06-26: Proposed in https://github.com/kubernetes/community/pull/2314
-   2018, 2019: Various comments on pull request
-   2019-07-30: Moved to a KEP (with no edits from the original proposal)
-   2019-08-05: Updated KEP with more details.
-   2019-10-18: Updated KEP with more RBAC details.
-   2020-01-25: Updated KEP and marked as implementable.
-   2021-01-28: Added PRR questionaire.
