---
title: Service Account signing key retrieval
authors:
  - "@mikedanese"
  - "@cceckman"
owning-sig: sig-auth
participating-sigs:
  - sig-auth
reviewers:
  - "@liggitt"
  - "@enj"
  - "@micahhausler"
  - "@ericchiang"
approvers:
  - "@liggitt"
  - "@enj"
  - "@micahhausler"
  - "@ericchiang"
editor: TBD
creation-date: 2018-06-26
last-updated: 2019-07-30
status: provisional
replaces:
  - "https://github.com/kubernetes/community/pull/2314/"
---

# Service Account signing key retrieval

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

The Kubernetes API server generates (signs) JSON Web Tokens that are meant to
authenticate Kubernetes service accounts. The API server can indeed authenticate
these tokens, but in general, no other system can: there is no standard way to
get the public portion of the keypair used to sign KSA tokens. Systems ("relying
parties") that want authenticate KSA tokens must either send them back to the
API server (via `TokenReview`) or use some provider-specific method to get the
authentication key.

If a relying party could obtain trusted metadata about the service account token
provider - in particular, the issuer (`iss`) value and the public key(s) used -
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
without Kuberentes-specific or provider-specific logic.

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

If services (other than the API server could authenticate KSA tokens directly:

-   the API server wouldn't have to scale with the data-plane (i.e.
    authentication) load; and
-   workloads could use native credentials to authenticate to services outside
    of the cluster.

### Goals

-   Allow (authorized) systems to discover the information they need to
    authenticate KSA tokens.
-   Attempt compatibility with OIDC: common libraries that authenticate OIDC
    tokens should be able to authenticate KSA tokens.

As a stretch goal / consideration:

-   Support authentication when the API server is not directly reachable by the
    relying party.
    -   e.g.: a cloud-based service authenticating an API server that doesn't
        have a public Internet address.

Note that the API server has a very different flow from OIDC with respect to
generating tokens. As such, our goal is OIDC *compatibility*...

### Non-Goals

...but not OIDC *compliance*.

We aren't trying to: - Make the KSA token process fully compliant with OIDC
specifications. - OIDC includes flows for token acquisition, token exchange,
getting user data out of tokens, etc. - But we're only interested in the parts
relevant to *relying parties*, i.e. token authentication. - We don't need to do
something `REQUIRED` just because the spec says `REQUIRED`; but we may need to
do it if relying parties expect that field. - This may mean that our
implementation doesn't comply with the spec; e.g. we might skip `token_endpoint`
in the
[discovery document](https://openid.net/specs/openid-connect-discovery-1_0.html#ProviderMetadata),
even without supporting the
[Implicit Flow](https://openid.net/specs/openid-connect-core-1_0.html#ImplicitFlowAuth). -
Define new cryptographic or authentication protocols. - Change the format of KSA
tokens, how they're generated, or how they're issued.

## Proposal

A non-resource API will be added to the apiserver that will expose an
[OIDC discovery](https://openid.net/specs/openid-connect-discovery-1_0.html)-
compatible document. This document will reflect the issuer value and provide
some additional information about the tokens issued.

In addition, the API server will serve a JWKS consisting of the public keys from
`--service-account-key-file` and `--service-account-signing-key`, i.e. all the
public keys used for valid KSA tokens. The OIDC discovery document will point to
the JWKS path as the `jwks_uri`.

For example, if the API server is configured with the `--service-account-issuer`
value `https://dev.cluster.internal`, the API server could expose the following
configuration:

```
> GET /.well-known/openid-configuration
{
  "issuer": "https://dev.cluster.internal",
  "jwks_uri": "https://dev.cluster.internal/serviceaccountkeys/v1",
  "authorization_endpoint": "urn:kubernetes:programmatic_authorization",
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
  "claims_supported": [
    "sub",
    "iss"
  ]
}
> GET /serviceaccountkeys/v1
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
appropriately. We will consider expanding `system:public-info-viewer` RBAC
ClusterRole to grant access to the new paths; some other roles may have
permission already via a pattern match on `nonResourceURLs` (e.g.
`cluster-admin`).

## Implementation History

-   2018-06-26: Proposed in https://github.com/kubernetes/community/pull/2314
-   2018, 2019: Various comments on pull request
-   2019-07-30: Moved to a KEP (with no edits from the original proposal)
-   2019-08-05: Updated KEP with more details.
-   2019-10-18: Updated KEP with more RBAC details.
