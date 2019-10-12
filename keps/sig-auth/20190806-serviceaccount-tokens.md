---
title: Bound Service Account Tokens
authors:
  - "@mikedanese"
owning-sig: sig-auth
approvers:
  - "@liggitt"
  - TBD
creation-date: 2019-08-06
last-updated: 2019-08-06
status: implemented
---

# Bound Service Account Tokens

## Table Of Contents

<!-- toc -->
- [Summary](#summary)
- [Background](#background)
- [Motivation](#motivation)
- [Design Details](#design-details)
  - [Token attenuations](#token-attenuations)
    - [Audience binding](#audience-binding)
    - [Time binding](#time-binding)
    - [Object binding](#object-binding)
  - [API Changes](#api-changes)
    - [Add <code>tokenrequests.authentication.k8s.io</code>](#add-)
    - [Modify <code>tokenreviews.authentication.k8s.io</code>](#modify-)
    - [Example Flow](#example-flow)
  - [Service Account Authenticator Modification](#service-account-authenticator-modification)
  - [ACLs for TokenRequest](#acls-for-tokenrequest)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
<!-- /toc -->

## Summary

This KEP describes an API that would allow workloads running on Kubernetes
to request JSON Web Tokens that are audience, time and eventually key bound.

## Background

Kubernetes already provisions JWTs to workloads. This functionality is on by
default and thus widely deployed. The current workload JWT system has serious
issues:

1.  Security: JWTs are not audience bound. Any recipient of a JWT can masquerade
    as the presenter to anyone else.
1.  Security: The current model of storing the service account token in a Secret
    and delivering it to nodes results in a broad attack surface for the
    Kubernetes control plane when powerful components are run - giving a service
    account a permission means that any component that can see that service
    account's secrets is at least as powerful as the component.
1.  Security: JWTs are not time bound. A JWT compromised via 1 or 2, is valid
    for as long as the service account exists. This may be mitigated with
    service account signing key rotation but is not supported by client-go and
    not automated by the control plane and thus is not widely deployed.
1.  Scalability: JWTs require a Kubernetes secret per service account.

## Motivation

We would like to introduce a new mechanism for provisioning Kubernetes service
account tokens that is compatible with our current security and scalability
requirements.

## Design Details

Infrastructure to support on demand token requests will be implemented in the
core apiserver. Once this API exists, a client of the apiserver will request an
attenuated token for its own use. The API will enforce required attenuations,
e.g. audience and time binding.

### Token attenuations

#### Audience binding

Tokens issued from this API will be audience bound. Audience of requested tokens
will be bound by the `aud` claim. The `aud` claim is an array of strings
(usually URLs) that correspond to the intended audience of the token. A
recipient of a token is responsible for verifying that it identifies as one of
the values in the audience claim, and should otherwise reject the token. The
TokenReview API will support this validation.

#### Time binding

Tokens issued from this API will be time bound. Time validity of these tokens
will be claimed in the following fields:

* `exp`: expiration time
* `nbf`: not before
* `iat`: issued at

A recipient of a token should verify that the token is valid at the time that
the token is presented, and should otherwise reject the token. The TokenReview
API will support this validation.

Cluster administrators will be able to configure the maximum validity duration
for expiring tokens. During the migration off of the old service account tokens,
clients of this API may request tokens that are valid for many years. These
tokens will be drop in replacements for the current service account tokens.

#### Object binding

Tokens issued from this API may be bound to a Kubernetes object in the same
namespace as the service account. The name, group, version, kind and uid of the
object will be embedded as claims in the issued token. A token bound to an
object will only be valid for as long as that object exists.

Only a subset of object kinds will support object binding. Initially the only
kinds that will be supported are:

* v1/Pod
* v1/Secret

The TokenRequest API will validate this binding.

### API Changes

#### Add `tokenrequests.authentication.k8s.io`

We will add an imperative API (a la TokenReview) to the
`authentication.k8s.io` API group:

```golang
type TokenRequest struct {
  Spec   TokenRequestSpec
  Status TokenRequestStatus
}

type TokenRequestSpec struct {
  // Audiences are the intendend audiences of the token. A token issued
  // for multiple audiences may be used to authenticate against any of
  // the audiences listed. This implies a high degree of trust between
  // the target audiences.
  Audiences []string

  // ValidityDuration is the requested duration of validity of the request. The
  // token issuer may return a token with a different validity duration so a
  // client needs to check the 'expiration' field in a response.
  ValidityDuration metav1.Duration

  // BoundObjectRef is a reference to an object that the token will be bound to.
  // The token will only be valid for as long as the bound object exists.
  BoundObjectRef *BoundObjectReference
}

type BoundObjectReference struct {
  // Kind of the referent. Valid kinds are 'Pod' and 'Secret'.
  Kind string
  // API version of the referent.
  APIVersion string

  // Name of the referent.
  Name string
  // UID of the referent.
  UID types.UID
}

type TokenRequestStatus struct {
  // Token is the token data
  Token string

  // Expiration is the time of expiration of the returned token. Empty means the
  // token does not expire.
  Expiration metav1.Time
}

```

This API will be exposed as a subresource under a serviceaccount object. A
requestor for a token for a specific service account will `POST` a
`TokenRequest` to the `/token` subresource of that serviceaccount object.

#### Modify `tokenreviews.authentication.k8s.io`

The TokenReview API will be extended to support passing an additional audience
field which the service account authenticator will validate.

```golang
type TokenReviewSpec struct {
  // Token is the opaque bearer token.
  Token string
  // Audiences is the identifier that the client identifies as.
  Audiences []string
}
```

#### Example Flow

```
> POST /apis/v1/namespaces/default/serviceaccounts/default/token
> {
>   "kind": "TokenRequest",
>   "apiVersion": "authentication.k8s.io/v1",
>   "spec": {
>     "audience": [
>       "https://kubernetes.default.svc"
>     ],
>     "validityDuration": "99999h",
>     "boundObjectRef": {
>       "kind": "Pod",
>       "apiVersion": "v1",
>       "name": "pod-foo-346acf"
>     }
>   }
> }
{
  "kind": "TokenRequest",
  "apiVersion": "authentication.k8s.io/v1",
  "spec": {
    "audience": [
      "https://kubernetes.default.svc"
    ],
    "validityDuration": "99999h",
    "boundObjectRef": {
      "kind": "Pod",
      "apiVersion": "v1",
      "name": "pod-foo-346acf"
    }
  },
  "status": {
    "token":
    "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJz[payload omitted].EkN-[signature omitted]",
    "expiration": "Jan 24 16:36:00 PST 3018"
  }
}
```

The token payload will be:

```
{
  "iss": "https://example.com/some/path",
  "sub": "system:serviceaccount:default:default,
  "aud": [
    "https://kubernetes.default.svc"
  ],
  "exp": 24412841114,
  "iat": 1516841043,
  "nbf": 1516841043,
  "kubernetes.io": {
    "serviceAccountUID": "c0c98eab-0168-11e8-92e5-42010af00002",
    "boundObjectRef": {
      "kind": "Pod",
      "apiVersion": "v1",
      "uid": "a4bb8aa4-0168-11e8-92e5-42010af00002",
      "name": "pod-foo-346acf"
    }
  }
}
```

### Service Account Authenticator Modification

The service account token authenticator will be extended to support validation
of time and audience binding claims.

### ACLs for TokenRequest

The NodeAuthorizer will allow the kubelet to use its credentials to request a
service account token on behalf of pods running on that node. The
NodeRestriction admission controller will require that these tokens are pod
bound.

### Graduation Criteria

#### Beta -> GA Graduation

- TBD
