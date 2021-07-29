# KEP-2843: Forward Additional Request Metadata in {Token,SubjectAccess,Admission}Review

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Alternatives Considered](#alternatives-considered)
    - [Front Proxy](#front-proxy)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
<!-- /toc -->

## Summary

This KEP proposes enhancing the TokenReview, SubjectAccessReview and
AdmissionReview APIs to convey additional request context to the backing
implementation.

## Motivation

Today, certain request attributes that are known to kube-apiserver may be
relevant to the review webhook backend.

For example a TokenReview backend may want answers to:

* Was the request made over https?
* What are the IP addresses of the peer and origin of the request?

These attributes can be used to enhance audit logs in an IDP backend, determine
access levels (such as whether the request originates from a trusted network),
or enforce security policy centrally (such as disallowing bearer token
authentication over insecure protocols).

Beyond authentication, these attributes have similarly plausible applications in
SubjectAccessReview and AdmissionReview such as deny-only authorization checks.

### Goals

* Forward additional request metadata to TokenReview, SubjectAccessReview and
  AdmissionReview

### Non-Goals

* Forward resource metadata to \*Review: The object of the request (e.g.
  which verb, which namespace, which pod) should remain unknown to the IDP.

## Proposal

We will extend TokenReviewSpec, SubjectAccessReviewSpec and AdmissionRequest to
add an additional field:

```golang
type TokenReviewSpec struct {
  ...

  // RequestAttributes is a collection of request attributes that may be relevant
  // to a TokenReview backend. These attributes should generally be stable over
  // multiple requests.
  //
  // Note: TokenReview webhook backends must handle absences of this field or
  // subfields of the RequestAttribute object. This data may be absent when the
  // API Server initiating the TokenReview is old or the data was unavailable.
  RequestAttributes RequestAttributes
}
...

type SubjectAccessReviewSpec struct {
  ...
  RequestAttributes RequestAttributes
}
...

type AdmissionRequest struct {
  ...
  RequestAttributes RequestAttributes
}
...

type RequestAttributes struct {
  // The HTTP request `Host` header value.
  Host string

  // The HTTP URL scheme, such as `http` and `https`.
  Scheme string

  // The IP address of the remote peer.
  //
  // Note: This is only the IP address of the originator of the request, when
  // the network path of the request does not traverse any hops that perform
  // SNAT. For example, if the request path traverses a simple proxy, this
  // will be the address of the proxy.
  PeerIP string

  // Additional standard headers used by frontproxies to forward request
  // attributes to backends. Currently, the only forwarded headers are:
  // * X-Forwarded-For
  // * X-Forwarded-Proto
  ProxyHeaders map[string][]string
}
```

These attributes will be resolved in RequestInfo filter and populated in the
[RequestInfo] struct.

Various caches (such as the token cache, and webhook authorization cache) will
be updated to include request attributes in the hash key. Notably, this means
that volatile attributes (e.g. a millisecond timestamp or trace ID) are not
suitable to include in RequestAttributes.

[RequestInfo]: https://github.com/kubernetes/kubernetes/blob/fffaadc01331cca57cafa0fc066a2a3eec23acb8/staging/src/k8s.io/apiserver/pkg/endpoints/request/requestinfo.go#L42

### Alternatives Considered

#### Front Proxy

Today, front proxies are supported for authentication offload but not for
authorization or admission. It is conceivable that front proxies could be used
to extend authorization and admission, but the architecture poses some practical
challenges such as:

* Integration with built in functionality of kube-apiserver such as audit
  logging and priority and fairness: It is not clear how to integrate a front
  proxy performing authorization and admission with these builtin systems. This
  functionality may need to be reimplemented in the front proxy.
* Resource deserialization and versioning: kube-apiserver supports a large and
  growing set of media types and API versions. A front proxy would need to keep
  pace with this support to apply policy that bases decisions on the request
  content.

Because of the additional implementation complexity well beyond what we've asked
from authentication proxies so far, this KEP proposes a minimal expansion of
existing APIs.

### Test Plan

This functionality will be tested in unit and integration tests.

### Graduation Criteria

TBD

### Upgrade / Downgrade Strategy

This change is backwards compatible.

### Version Skew Strategy

This change is backwards compatible. 
