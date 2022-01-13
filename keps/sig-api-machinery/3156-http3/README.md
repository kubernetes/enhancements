# KEP-3156: HTTP3

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
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [kube-apiserver](#kube-apiserver)
  - [client-go](#client-go)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Alpha](#alpha)
    - [Beta](#beta)
    - [GA](#ga)
    - [Deprecation](#deprecation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
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

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

IMPORTANT: THIS KEP HAS BEEN DEFERRED UNTIL HTTP/3 IS SUPPORTED BY THE GOLANG STANDARD LIBRARY

One of the main advantages of HTTP/3 is increased performance, specifically around fetching multiple
objects simultaneously. With HTTP/2, any interruption (packet loss) in the TCP connection blocks all
streams (Head of line blocking).

In addition, HTTP/3 offers 0-RTT support, which means that subsequent connections can start up much
faster by eliminating the TLS acknowledgement from the server when setting up the connection. This
means the client can start requesting data much faster than with a full TLS negotiation.

## Motivation

Kubernetes clusters defaulted to HTTP/2 for the control plane communications to solve all the
performance and reliability problems caused by the previous protocol HTTP/1.
However, there are some notable exceptions:

- [communication between apiserver and webhooks](https://github.com/kubernetes/kubernetes/pull/82090).
- [communication that use SPDY or websockets](https://github.com/kubernetes/kubernetes/issues/7452)

HTTP/3 solves some of the HTTP/2 problems, mainly the Head of Line Blocking problem because it uses
QUIC RFC9000 for the transport layer, improving the performance and the resilience of the control
plane.

Despite the HTTP/3 protocol, as January 2022, is still an [Internet Draft](https://datatracker.ietf.org/doc/draft-ietf-quic-http/), it is already supported by 73% of running web browsers, and there is also a [golang implementation](https://github.com/lucas-clemente/quic-go) that is being used by known projects and companies.

Having support for HTTP/3 will allow the project to have early feedback, and the possibility to influence it.

### Goals

- Support HTTP/3 on the apiserver wherever is possible
- Offer an opt-in option in client-go to use HTTP/3

### Non-Goals

- Use or default any of the Kubernetes components to HTTP/3
- Automatically upgrade connections to HTTP/3
- Replace websockets or SPDY communications

## Proposal

The proposal is two fold:

- support HTTP/3 in the apiserver, that will require to listen on the UDP port, in addition to current TCP port that is being used.

- support HTTP/3 in client-go, so developers can consume it, that will require to add a new roundtripper and a new configuration option to opt-in.


### User Stories (Optional)

#### Story 1

- As a Kubernetes admin I'd like to be able to use HTTP/3 in front of my control-plane, so external users can benefit of its properties.

#### Story 2

- As a Kubernetes developer I'd like to be able to use HTTP/3 as my communication protocol in my operator, to be more resilient on environments with hostile network conditions.

### Risks and Mitigations

This is a new protocol and an experimental feature, it will be behind a feature flag and will not graduate to beta until
the protocol and its implementation are mature.

The protocol is still a Draft, however, it is not likely it change too much at this stage before is published as RFC.

The golang implementation tries to be as much compatible as possible with the golang standard library, but it is not fully compatible. It also brings a considerable amount of dependencies.

## Design Details

HTTP/3 main difference is that it uses UDP as transport instead of TCP, beside that, it is mostly compatible with HTTP/2.

The current golang http/3 implementation tries to be as much compatible as possible with the standard library, something that
simplifies the implementation, a working prototype can be seen here https://github.com/kubernetes/kubernetes/pull/106707/.

### kube-apiserver


A new UDP listener should be added to the kube-apiserver in the same port and address that the current TCP listener.

```go
// https://github.com/kubernetes/kubernetes/blob/1367cca8fd67b09606b01c0a9e46cef59aef3424/staging/src/k8s.io/apiserver/pkg/server/secure_serving.go#L276
func RunServer(
	server *http.Server,
	ln net.Listener,
	shutDownTimeout time.Duration,
	stopCh <-chan struct{},
) (<-chan struct{}, <-chan struct{}, error) {
```

In addition to the listener, there is some logic inside the apiserver that discriminates by protocol like `WrapForHTTP1Or2` that should be adapted.

```go
// https://github.com/kubernetes/kubernetes/blob/1367cca8fd67b09606b01c0a9e46cef59aef3424/staging/src/k8s.io/apiserver/pkg/endpoints/responsewriter/wrapper.go#L58
// WrapForHTTP1Or2 accepts a user-provided decorator of an "inner" http.responseWriter
// object and potentially wraps the user-provided decorator with a new http.ResponseWriter
// object that implements http.CloseNotifier, http.Flusher, and/or http.Hijacker by
// delegating to the user-provided decorator (if it implements the relevant method) or
// the inner http.ResponseWriter (otherwise), so that the returned http.ResponseWriter
// object implements the same subset of those interfaces as the inner http.ResponseWriter.
...
func WrapForHTTP1Or2(decorator UserProvidedDecorator) http.ResponseWriter {
```

### client-go

The golang http/3 implementation exposes a Transport roundtripper, there are two options to use it in client-go:

1. Use the WrapTransport option and let users configure the roundtripper manually.

```go
  tlsConfig, err := TLSConfigFor(config)
  if err != nil {
    return nil, err
  }
  rt = &http3.RoundTripper{
    TLSClientConfig: tlsConfig,
    QuicConfig: &quic.Config{
      KeepAlive: true,
    },
  }

config.WrapTransport = rt
```


2. Add a new option to the RESTConfig of client-go to enable http3 and automate the HTTP3 roundtripper configuration:

```go
// staging/src/k8s.io/client-go/transport/config.go
// Config holds various options for establishing a transport.
type Config struct {
	// UserAgent is an optional field that specifies the caller of this
	// request.
	UserAgent string
	// The base TLS configuration for this transport.
	TLS TLSConfig

// Use HTTP3
	EnableHTTP3 bool
```

### Test Plan

All current tests should pass using the new protocol, this can be done by enabling by default http3 in client go so all the
components of the cluster switch to it.

### Graduation Criteria

#### Alpha

- Feature implemented in both apiserver and client-go behind a feature flag
- Initial e2e tests completed and enabled

#### Beta

- HTTP/3 becomes an Internet RFC
- Golang community decides on HTTP/3 support in the standard library

#### GA

#### Deprecation

### Upgrade / Downgrade Strategy

HTTP Alternative Services has become the primary mechanism for HTTP/3 upgrade, but is explicitly
listed as a non-goal to implement Alt-Svc and automatic connection upgrades.

### Version Skew Strategy

## Implementation History

- (2021/1/18) Proposal
## Drawbacks

The highest risk is that HTTP/3 does not become an RFC, that is very unlikely since the browsers and some of the main companies are already using it.
## Alternatives
