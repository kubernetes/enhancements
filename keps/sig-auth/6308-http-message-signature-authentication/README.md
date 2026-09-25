# KEP-NNNN: HTTP Message Signature Authentication

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Client signing](#client-signing)
  - [Server verification](#server-verification)
  - [User Stories](#user-stories)
    - [Story 1: An existing signing credential, with no token exchange](#story-1-an-existing-signing-credential-with-no-token-exchange)
    - [Story 2: A captured request log](#story-2-a-captured-request-log)
    - [Story 3: A workload behind a TLS-terminating ingress](#story-3-a-workload-behind-a-tls-terminating-ingress)
  - [Notes/Constraints/Caveats](#notesconstraintscaveats)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Wire format](#wire-format)
    - [<code>Signature-Input</code> and <code>Signature</code>](#signature-input-and-signature)
    - [<code>keyid</code>, and how a request selects an authenticator](#keyid-and-how-a-request-selects-an-authenticator)
    - [<code>Content-Digest</code>](#content-digest)
    - [<code>Signature-Certificate</code>](#signature-certificate)
    - [Recognizing a signed request](#recognizing-a-signed-request)
  - [What a signature covers](#what-a-signature-covers)
    - [Why <code>@authority</code> is in the floor](#why-authority-is-in-the-floor)
    - [Required signed headers](#required-signed-headers)
  - [Client configuration](#client-configuration)
    - [A cloud provider example](#a-cloud-provider-example)
    - [A certificate and its key](#a-certificate-and-its-key)
  - [Server configuration](#server-configuration)
  - [Identity from a certificate](#identity-from-a-certificate)
  - [Identity from a resolver](#identity-from-a-resolver)
  - [Verification order](#verification-order)
  - [Replay](#replay)
  - [Metrics](#metrics)
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
- [Open Questions](#open-questions)
  - [Where should <code>httpSignature</code> live in a kubeconfig?](#where-should-httpsignature-live-in-a-kubeconfig)
  - [Who owns the RFC 9421 and structured field values implementations](#who-owns-the-rfc-9421-and-structured-field-values-implementations)
  - [What bounds resolver calls from an unauthenticated caller](#what-bounds-resolver-calls-from-an-unauthenticated-caller)
  - [Cross-cluster replay of an identical request](#cross-cluster-replay-of-an-identical-request)
  - [Whether replay is narrowed below the acceptance window for certificates](#whether-replay-is-narrowed-below-the-acceptance-window-for-certificates)
  - [Aggregated API servers, admission, and conversion webhooks](#aggregated-api-servers-admission-and-conversion-webhooks)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [DPoP](#dpop)
  - [Send the request to a webhook and verify there](#send-the-request-to-a-webhook-and-verify-there)
  - [A signing proxy on the client side](#a-signing-proxy-on-the-client-side)
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
  - [ ] (R) [all GA Endpoints](https://github.com/kubernetes/community/pull/1806) must be hit by [Conformance Tests](https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/conformance-tests.md) within one minor version of promotion to GA
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

This KEP adds a new authentication format to the Kubernetes API: an [RFC 9421][rfc9421] HTTP message
signature, carried in the `Signature-Input` and `Signature` fields rather than in an
`Authorization` header. The client holds a key and signs each request. kube-apiserver can validate a
signature and looks up the user's key and `UserInfo`.

[rfc9421]: https://www.rfc-editor.org/rfc/rfc9421.html

The credential on each request is an operation rather than a value. Nothing the client sends can be
captured and used against a different request, and because the proof rides in the message rather
than in the transport. The request can still be verified after an intermediary has terminated TLS.

The HTTP signature authenticator is additive, off by default, and configured in the API server's
`AuthenticationConfiguration` alongside JWT and anonymous authentication behind the
`HTTPSignatureAuthentication` feature gate. Bearer tokens, client certificates, and every other
authenticator keep working unchanged.

Two in-tree methods of resolving a signature to an identity are proposed. In the first, the client
carries an X.509 certificate and the server holds only the authority that issued it. In the second,
the server asks a resolver process over a Unix socket which key verifies a given key ID and whose
identity it is.

A signature is verified against the bytes of the request as received, before anything has been
decoded, defaulted, or serialized. Shipping those bytes to another process to be verified there is
not proposed: byte-exact verification does not survive a round trip through another system's request
format, and a process asked to verify a signature must be given the whole request, where a process
asked for a key needs only a key ID.

## Motivation

Every bearer credential Kubernetes accepts travels on the wire. For bearer tokens, service account
tokens, and OIDC ID tokens, the client sends the credential itself on every request, so whoever
captures or observes one can replay it against any request until it expires. A shorter credential
lifetime shrinks that window without changing the kind of credential.

Client certificates are not in this category, but have different tradeoffs. The client signs the
transport rather than sending its key, so mTLS is already proof of possession and a complete answer
for direct API server access. That proof belongs to the connection, so it ends at the first load
balancer or ingress that terminates TLS, and the API server behind one must trust the proxy's claim
about who called.

A message signature differs from both existing methods by keeping the key with the client, and
verified at the API server. The credential included in a request is the result of using it: a value
computed over this request and no other. This yields three benefits:

1. **A captured request is a receipt, not a key.** A proxy access log, a compromised sidecar, or a
   node exfiltration yields signatures over requests that already happened. There is nothing in them
   to reuse against a different request.
2. **A signature only fits its own request.** It covers the method, the authority, the path, the
   query, a digest of the body when there is one, and every header the client can set. Nobody can
   move a captured signature onto a different verb, path, or body.
3. **A proxy that terminates TLS does not break it.** The proof is in the message, so it reaches
   kube-apiserver intact and nothing in the path has to be trusted to report who called. This holds
   where the intermediary preserves what the signature covers.

The client never needs the key as a value, only the ability to use it. That leaves room for the key
to live somewhere it cannot leave: a TPM, a hardware token, a non-exportable platform key. This KEP
does not deliver a hardware-backed signer, but client-go customizations for hardware could be
integrated with request signing. The client-go plumbing passes a signer rather than a key, which
keeps that door open.

Some existing credentials can already function as signing keys. A cloud provider's symmetric
credential, an X509 certificate, and a hardware token all work by signing something rather than by
being handed over. A caller holding one today may reach the Kubernetes API today by exchanging it
for a bearer token. That exchange manufactures a replayable credential out of one that was not
replayable. This KEP lets such a caller authenticate with the credential it already holds.

### Goals

- Accept an RFC 9421 signature as an authentication format on the Kubernetes API, additive to every
  existing authenticator.
- Support signing requests with a signing key supplied by an exec credential plugin, so a rotating
  credential is not stored in kubeconfig.
- Specify the covered request component set as server policy, so a signature that covers less than
  the server requires is rejected rather than accepted for what it happens to name.
- Reject any signed request carrying a security-relevant header the signature does not cover, so an
  intermediary cannot add impersonation to a signed request.
- Resolve a verified key to a `UserInfo` by at least one mechanism that needs no per-client state on
  the server.
- Allow operators to configure the residual replay window in configuration
- [TODO: Define if kubelet validating signatures is in-scope]

### Non-Goals

- Replacing or deprecating bearer tokens, service account tokens, or client certificates. This is an
  additive change.
- Requiring signatures. An API server with this configured still accepts every other credential it
  is configured for.
- Authorization. This terminates at the authenticator and produces a `UserInfo`. Expressing what a
  signature permits is [KEP-5681](/keps/sig-auth/5681-conditional-authorization)'s problem.
- End-to-end coverage past kube-apiserver. The aggregation layer re-originates a request to an
  extension API server with kube-apiserver's own client certificate and `X-Remote-User` headers. The
  properties above stop at kube-apiserver. The same applies for admission and conversion webhooks.
  [TODO: Decide if we scope in kube-apiserver should support signatures for requests it sends
  to conversion/validating/mutating webhooks and aggregate APIs]
- Eliminating replay. A captured request can be replayed as itself until it ages out of the
  acceptance window. Narrowing that below the window requires state shared by every API server; see
  [Replay](#replay). [TODO: replay could be mitigated with a pluggable shared nonce cache]
- Key distribution. Which party holds which key material, and how it gets there, is outside
  kube-apiserver.

## Proposal

The following proposal is a summary of the user and operator configurations and mechanics. Critical
design details are farther down.

### Client signing

A client signs each request and sends two fields, `Signature-Input` and `Signature`.
kube-apiserver reconstructs the signature base from the request it received, verifies the signature
against a key, and maps the key to a user.

Clients using standard client-go can specify a signing key in one of two ways: either pointing to a
key and certificate, or executing an authenticating process.

```yaml
users:
- name: key-pair-user
  user:
    httpSignature:
      apiVersion: client.authentication.k8s.io/v1alpha1
      ttl: 30s
      certFile: /home/user/.kube/keys/cert.crt
      keyFile: /home/user/.kube/keys/cert.key
```
_[Example 1]: A user-specified key pair in a kubeconfig_

In the example of the asymmetric key pair, client-go automatically identifies the type of key used
in the certificate (ECDSA or RSA), includes the public certificate in the `Signature-Certificate`
HTTP header, a signs each request (and certificate header) using the key.

```yaml
users:
- name: hmac
  user:
    httpSignature:
      apiVersion: client.authentication.k8s.io/v1alpha1
      ttl: 30s
      # The API must configure the same derivation function. The request is HMAC
      # signed with a key derived using the original secret with this function,
      # using the date in the signature's own created parameter, so the two
      # agree even across a date boundary. The server holds only the derived key
      # for one day, and never the user's secret key.
      keyDerivation:
        hash: sha-256
        kind: hmac-ladder
        secretPrefix: K8S1
        steps:
        - date: YYYYMMDD
          name: date
        - name: cluster
          scope: true
        - literal: k8s1_request
          name: terminator
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: /home/user/bin/k8s-httpsig-creds
      args: [hmac, --cluster, httpsig-verifier]
```
_[Example 2]: A user-specified asymmetric key and derivation specification with an executed
retrieval program in a kubeconfig_

In the example of a symmetric key, when client-go executes the credential program it specifies via
STDIN the `client.authentication.k8s.io/v1` JSON structure including the HTTP signature
requirements and key derivation. The authentication program can then compute the required
derived signing key and return that with the expiration time to client-go for request signing.

### Server verification

For verification, the API server must be configured to accept signed requests. Kubernetes does not
maintain any database of credentials or signing keys, so this proposal delegates key lookup to two
sources.

For requests signed with an X509 certificate, operators can configure the API server's
authentication configuration file with an X509 certificate authority. Operators may also specify
validation rules and claim mappings on certificate-based identities similar to existing JWT
authentication.

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AuthenticationConfiguration
httpSignature:
  # Every signature must cover these headers and carry one of the listed values.
  # X-K8s-Audience is conventional for preventing replay across clusters
  requiredSignedHeaders:
    X-K8s-Audience:
    - https://my-apiserver.example.com
  # A verifier cannot measure a client's clock, so this is
  # a risk budget justified by this server's time synchronization.
  maxClockSkew: 5s
  authenticators:
  - name: workload-certificates
    # Checked against the created timestamp in the signature. May be shorter than the
    # client's specified TTL value
    maxAge: 1m
    x509:
      certificateAuthority: |
        -----BEGIN CERTIFICATE-----
        ...
        -----END CERTIFICATE-----
      # Rules run before the mappings, so a mapping never reads a certificate no
      # rule has vetted.
      certificateValidationRules:
      - expression: cert.notAfter - cert.notBefore <= duration('24h')
        message: certificate lifetime must not exceed 24 hours
      claimMappings:
        username:
          expression: '"cert:" + cert.uriSANs[0]'
        groups:
          expression: cert.subject.organization
    # The mapping above derives groups from the certificate's subject, which hands
    # the choice of group to whoever can request one.
    userValidationRules:
    - expression: '!user.username.startsWith("system:") && !user.groups.exists(g, g.startsWith("system:"))'
      message: 'this authenticator may not assert an identity under the system: prefix'
```

For symmetric or asymmetric keys, operators can configure a resolver API to lookup key IDs.
Implementations of this API allows operators to use their own key databases.

```yaml
apiVersion: apiserver.config.k8s.io/v1alpha1
kind: AuthenticationConfiguration
httpSignature:
  requiredSignedHeaders:
    X-K8s-Audience:
    - https://my-apiserver.example.com
  authenticators:
  - name: corp-resolver
    maxAge: 1m
    resolver:
      endpoint: unix:///var/run/httpsig/resolver.sock
      # Only key IDs whose first segment matches reach this resolver. Omitting
      # this means it is asked about every key ID.
      keyIDPrefixes: [corp]
      # A value the client covers with its signature and this server passes on,
      # for a resolver that decides identity from a session token rather than
      # from a key ID.
      relayedHeaders: [X-Session-Token]
```

### User Stories

#### Story 1: An existing signing credential, with no token exchange

ACME Corp has a corporate CA that signs user certificates for each employee's laptop. Users daily
run an `acme-cred-init` program that uses a physical passkey to request a new x509 certificate valid
for 20-hours.

Previously, ACME had to maintain an internal token exchange API where users could authenticate using
their x509 certificate to get an OIDC token to authenticate to a Kubernetes cluster. The security
mandated that replayable OIDC tokens must be audience-scoped to a specific Kubernetes cluster to
prevent cross-cluster token replay in the event of a cluster compromise. For developers connecting
to any of several dozen Kubernetes clusters, this required fetching a new token each new cluster
each day.

```yaml
users:
- name: corp-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: acme-oidc-helper
      args:
      - --cluster
      - prod-014
```

With HTTP request signatures, ACME Corp has updated the authentication configuration file for their
Kubernetes API servers to trust their corporate user CA. They are able to specify authentication
rules on each cluster, denying authentication to non-developers. RBAC is used for authorization to
ensure the correct teams are authorized on each cluster.

```yaml
  - name: workload-certificates
    x509:
      certificateAuthority: |
        -----BEGIN CERTIFICATE-----
        ...
        -----END CERTIFICATE-----
      certificateValidationRules:
      - expression: cert.notAfter - cert.notBefore <= duration('20h')
        message: certificate lifetime must not exceed 20 hours
      - expression: !cert.organization.exists(ou, ou == "developers")
        message: user must be in the developers OU
      claimMappings:
        username:
          expression: '"cert:" + cert.uriSANs[0]'
        groups:
          expression: cert.subject.organization
    userValidationRules:
    - expression: '!user.username.startsWith("system:") && !user.groups.exists(g, g.startsWith("system:"))'
      message: 'this authenticator may not assert an identity under the system: prefix'
```

All a developer has to do now update their kubeconfig files to specify their ACME user certificate.
Each request is signed, and the signature covers an `X-K8s-Audience` header naming the cluster,
so ACME security is satisfied that a request captured against one cluster cannot be replayed against
another. All user-originated requests are now signed, and the existing OIDC exchange is slated for
tear-down.

```yaml
users:
- name: key-pair-user
  user:
    httpSignature:
      apiVersion: client.authentication.k8s.io/v1alpha1
      ttl: 30s
      certFile: /home/user/.acme/keys/cert.crt
      keyFile: /home/user/.acme/keys/cert.key
```

#### Story 2: A captured request log

While investigating a memory leak issue, an operator increased the log verbosity of ACME Corp's
Kubelets to try and gain insight. The log verbosity was left elevated for a month and due to an
undiscovered issue in Kubernetes, OTEL clients `Authorization` headers for `/metrics/cadvisor`
requests were sent to the Kubelet's log. These logs were shipped off host to a central logging
server that recently suffered disk fullness issues, so access logs of who reviewed kubelet logs were
lost. ACME was unable to verify who accessed kubelet logs and could have replayed OTEL credentials.

(Kubernetes has had CVEs for token logging in the past: [CVE-2019-11250][CVE-2019-11250],
[CVE-2020-8565][CVE-2020-8565])

[CVE-2019-11250]: https://github.com/kubernetes/kubernetes/issues/81114
[CVE-2020-8565]: https://github.com/kubernetes/kubernetes/issues/95623

By switching the OTEL collection system to use HTTP request signing, each request's unique signature
is only replayable to the exact same kubelet for 1 minute after the request was signed. Kubelet logs
are batched before being sent off host, so any operator who could review the logs if they contained
a signature would only be able to replay the same request (eg: `/metrics/cadvisor` collection) for
one minute.

#### Story 3: A workload behind a TLS-terminating ingress

ACME Corp used an [authenticating proxy][auth-proxy] in front of their Kubernetes API servers
because of corporate mandates that public TLS certificate keys exist within an HSM. While
investigating a disk fullness issue, an operator SSH'd into ACME Corp's TLS-terminating proxy.
Because of log volume fullness, kernel audit logs were lost during the operator's actions, and ACME
now has to report an audit event to their regulator and include the lack of non-repudiation during
the event. Because of the authenticating proxy's ability to mint identities to the Kubernetes API
via `X-Remote-User` and `X-Remote-Group` headers, the Kubernetes Audit log is unable to
differentiate a correctly authenticated request versus one that an operator could have forged
an `X-Remote-User` header for.

[auth-proxy]: https://kubernetes.io/docs/reference/access-authn-authz/authentication/#authenticating-proxy

After switching their Kubernetes clusters to use HTTP request signing, each request is signed with
the user's key and is now tamper-evident. The TLS terminating proxy no longer functions as an
authenticating proxy so another disk fullness and operator SSHing into a host with log loss is no
longer a reportable event.


### Notes/Constraints/Caveats

**A signature is not a name.** Verifying a signature establishes that the holder of a key made this
request. It says nothing about who that is. Every deployment therefore needs a mapping from key to
identity, which is why this KEP proposes two and why the identity side carries most of the
validation rules.

**Coverage is server policy, not client choice.** RFC 9421 signatures declare what they cover. A
verifier that checks only "this signature is valid for the components it names" can accept a
signature covering nothing at all. The covered set is therefore fixed by the server, and the client
rules exist so a client produces something the server accepts. Coordination between server and
client for server-required components is out of scope of this KEP.

**Coverage prevents removal, not addition.** A covered header cannot be stripped, because the
signature base could not then be reconstructed. Nothing stops an intermediary appending unsigned
headers like `Impersonate-User` to a request that carried none. The verifier implementation closes
that by requiring every protected header *present* on a request to be covered.

**The client's clock becomes an operational dependency.** A signature carries a `created` timestamp
and is rejected outside a server-specified window. A client whose clock is wrong by more than
`maxClockSkew` cannot authenticate.

### Risks and Mitigations

**A resolver can claim any identity.** Whoever holds a resolver's socket chooses the username and
groups outright, which is a strictly larger grant than vending a key. The same is true of a
configured certificate authority.

The bound on what an x509 certificate authority HTTP signature may claim is a CEL expression in
`AuthenticationConfiguration`, not a hardcoded rule. `userValidationRules` on the authenticator is
where a cluster states it, using the same field and the same wording for both backends and matching
the JWT authenticator in the same configuration object. A cluster that does not operate its own
resolver wants the rule refusing the `system:` prefix, and it is spelled out in the reference
configuration so it is a rule to copy rather than a rule to remember:

```yaml
    userValidationRules:
    - expression: '!user.username.startsWith("system:") && !user.groups.exists(g, g.startsWith("system:"))'
      message: 'this authenticator may not assert an identity under the system: prefix'
```

One class of name is refused for a different reason: `system:authenticated`,
`system:unauthenticated`, and `system:anonymous` are assigned by the authentication machinery. The
first two are set depending on whether authentication succeeded, and the last by the anonymous
authenticator about a request that carried no credential. A backend claiming one is a layer
confusion rather than a privilege grant, so no expression is needed to refuse it and none can permit
it.

**A key lookup happens before any signature has verified,** because verifying needs the key. So an
unauthenticated caller can drive a resolver call. Four things reduce the cost: the key ID is
length-capped before it becomes a cache key, signature age is checked before the lookup, concurrent
lookups for one key collapse to one call, and absence is remembered briefly. None of them caps the
rate for a caller cycling through distinct key IDs. Metrics are proposed so the question can be
answered with measurement rather than a guessed constant, and the certificate backend does not have
this exposure at all. See [What bounds resolver calls from an unauthenticated
caller](#what-bounds-resolver-calls-from-an-unauthenticated-caller).

**Signature verification costs CPU on the request path.** An unauthenticated caller can make the server
perform an asymmetric verification. Key type and size are bounded before a verifier is constructed,
and the certificate backend memoizes validated certificates. Priority and fairness bounds
concurrency but is scoped to resources rather than to authenticators.

**An intermediary that rewrites a covered component breaks authentication.** Host rewriting, path
prefix stripping, query manipulation, and body recompression all change the signature base. An L7
proxy in front of the API server must propagate the authority and leave the rest alone, which is
stated as a deployment requirement in [Why `@authority` is in the
floor](#why-authority-is-in-the-floor). It fails closed and visibly: every client behind that
intermediary stops authenticating at once.

## Design Details

### Wire format

RFC 9421 is a toolkit rather than a finished protocol. Its Section 1.4 requires an application of it
to state the covered components, the required signature parameters, how key material is retrieved,
which algorithms are allowed, and how the verifier derives component values. This section and the
two that follow are that profile for the Kubernetes API.

A signed request:

```
POST /api/v1/namespaces/default/pods?dryRun=All HTTP/1.1
Host: kubernetes.example.com
Content-Type: application/json
Content-Digest: sha-256=:X48E9qOokqqrvdts8nOJRJN3OWDUoyWxBf7kbu9DBPE=:
Signature-Input: sig1=("@method" "@authority" "@path" "@query" "content-digest" \
                       "content-type" "accept" "user-agent" "x-k8s-audience");\
                 created=1774483200;nonce="Ur9Xb3sK1pQ";alg="ecdsa-p384-sha384";\
                 tag="kubernetes";keyid="x509-sha256:9f86d081884c7d65"
Signature: sig1=:MEUCIQDdX7k...:
X-K8s-Audience: https://my-apiserver.my-domain.com
Signature-Certificate: :MIIB8jCCAZigAwIBAgIU...:
```

Five fields participate, from three sources:

| Field | Defined by | Carries |
| --- | --- | --- |
| `Signature-Input` | RFC 9421 | which components this signature covers, and the signature's own parameters |
| `Signature` | RFC 9421 | the signature bytes |
| `Content-Digest` | RFC 9530 | a digest of the body, so a signature can cover the body by reference |
| `Signature-Certificate` | this KEP | the certificate a signature was made under, where a certificate is the identity |
| `X-K8s-Audience` | Conventional in this KEP | Identifies which cluster the signature was made for |

#### `Signature-Input` and `Signature`

These two are a matched pair keyed by a label, `sig1` above. A request may carry more than one, and
kube-apiserver considers each in turn. The label carries no meaning. RFC 9421 Section 7.2.5 says so,
and an intermediary may relabel a signature without breaking it, because the label is not part of
the signature base. `client-go` emits `sig1`. Nothing on either side depends on that.

`Signature-Input` states the covered components in order. That order is what lets the verifier
rebuild the same signature base the signer built. [What a signature
covers](#what-a-signature-covers) is where the required set is fixed. After the component list come
the signature's parameters:

| Parameter | Value | Used for |
| --- | --- | --- |
| `created` | UNIX timestamp | Checked against `maxAge` and `maxClockSkew` before any key lookup |
| `nonce` | unique per request | Recorded where a resolver records nonces |
| `alg` | IANA HTTP Signature Algorithms registry name | Selecting the verifier, and refusing a key under an algorithm its holder did not intend |
| `tag` | `kubernetes` | Marking the signature as made for the Kubernetes API |
| `keyid` | see below | Finding the key |

All five are required. The parameters are not in the covered component list and do not need to be:
RFC 9421 appends them to the signature base as its last line, so every signature covers its own
parameters. Nobody can alter a `created`, an `alg`, a `tag`, or a `keyid` without breaking
verification.

kube-apiserver refuses a signature whose `tag` is absent or is anything other than `kubernetes`.
Without that check the tag would buy nothing. RFC 9421 Section 1.4 offers `tag` for exactly this
purpose, and Section 7.2.7 describes what goes wrong without it: a key held for signing requests to
one application produces signatures a second application will also accept, so a signature made for
some other RFC 9421 service becomes a Kubernetes API credential. The tag is what confines a
signature to the application it was made for, and only the verifier can enforce that.

#### `keyid`, and how a request selects an authenticator

`keyid` is the only field the server reads before it has done any cryptography, so its form decides
which authenticator handles the request. RFC 9421 does not impose any constraints on key
identifiers, so this KEP devises one prefix scheme, and allows for prefix selection.

`x509-sha256:` followed by the leaf certificate's SHA-256 digest means the identity is the
certificate in `Signature-Certificate`. The server recomputes that digest from the bytes it received
rather than trusting the claim.

Anything else is a resolver key ID, an opaque name whose first segment routes it to a resolver by
`keyIDPrefixes`. Its structure is the resolver operator's to choose.

#### `Content-Digest`

RFC 9421 does not sign the body. It signs component values, so covering a body means covering a
field that digests it, which is what RFC 9530 provides. A request that arrives with a body and no
covered `content-digest` is refused.

#### `Signature-Certificate`

The certificate travels as its DER encoding in a structured field byte sequence ([RFC
9651](https://www.rfc-editor.org/rfc/rfc9651.html)).

RFC 9440 registers `Client-Cert` with the same syntax, and this deliberately does not reuse it.
`Client-Cert` means "a proxy asserts this certificate was used on the TLS connection", which is a
different claim from "the client asserts this is the identity it signed with". A deployment running
such a proxy in front of the API server would have the two collide.

#### Recognizing a signed request

kube-apiserver treats a request as intending to authenticate by signature when `Signature-Input` or
`Signature` is present. There is no new `Authorization` scheme. An `Authorization` header, if one
is also sent, is left to the authenticators that own it, and configuration cannot add
`Authorization` to the signed set, because a header injection mechanism that could write it would be
a way around the paths that own it.

Once an authenticator accepts a signature, the server clears `Signature`, `Signature-Input`, and
`Signature-Certificate`, so nothing downstream can read them as credentials. The bearer token and
front proxy authenticators clear theirs the same way.

<<[UNRESOLVED sig-auth ]>>
The 2026-08-26 discussion described this as `Authorization: Signature` in place of `Authorization:
Bearer`. The current prototype does not do that, and detects a signed request by field presence
instead. An `Authorization` scheme name would have to be [registered with IANA][authz-scheme], and `Signature` is
not registered today. Whether the extra field is worth having is open: it would make intent explicit
rather than inferred, at the cost of a registration and a value that carries no information.
<<[/UNRESOLVED]>>

[authz-scheme]: https://www.iana.org/assignments/http-authschemes

### What a signature covers

Four classes:

| Class | Covered | Rule |
| --- | --- | --- |
| Floor | always | `@method`, `@authority`, `@path`, `@query` |
| Body | when a body is present | `content-digest` |
| Protected headers | when present on the request | `impersonate-user`, `impersonate-uid`, `impersonate-group`, `impersonate-extra-*`, `audit-id`, `accept`, `content-type`, `user-agent`, `signature-certificate` |
| Required signed headers | always, and the value is checked | whatever `requiredSignedHeaders` names |

The query is in the floor because Kubernetes API semantics live there: `dryRun`, `watch`,
`fieldSelector`, `resourceVersion`. A signature that leaves the query uncovered lets a bystander
turn a dry run into a real write.

`Content-Type` is protected because `Content-Digest` binds the bytes of a body but not their
interpretation: the API server parses the same bytes as JSON, YAML, or protobuf depending on this
header.

`Authorization` is deliberately not covered, and configuration cannot add it. Covering it would
mean signing over a credential that should not have been sent in the first place, and a header
injection mechanism able to write it would be a way around the authenticators that own it.

Configuration may add covered headers. Nothing removes from the floor.

#### Why `@authority` is in the floor

`@authority` records which server the client believed it was addressing. A signature that leaves it
uncovered says nothing about the intended destination, and an intermediary could hand the request to
a differently named backend with the signature still intact. Covering it makes the client's intent
part of what was signed.

**An intermediary in front of the API server MUST propagate the authority.** The verifier
reconstructs `@authority` from the authority on the request it received. An intermediary that
rewrites the host produces a different signature base, so verification fails for every client behind
it at once. There is no configured override for the verifier to substitute instead: a single
configured value cannot serve a cluster that clients legitimately reach under several names, and
reading `Forwarded` or `X-Forwarded-*` would rebuild the signature base from unsigned input the
caller controls.

For HTTP/2 that means the `:authority` pseudo-header, and for HTTP/1.1 the `Host` header.
Translation between the two preserves the value, as RFC 9113 Section 8.3.1 requires. What does not
survive is a deliberate rewrite, which some proxies do by default and which has to be turned off.

`@authority` coverage does not serve as replay defense against other cluster in all cases, and does
not uniquely identify a cluster.

The verifier reconstructs the authority from what the sender put on the wire, and nothing validates
that against the connection: Go's HTTP/2 server deliberately does not compare `:authority` with the
TLS server name, and no API server filter inspects it. A party replaying a captured request supplies
the authority the signature covered, and the base reconstructs. Coverage binds a legitimate client's
intent; it does not constrain someone attacking.

An in-cluster client builds its authority from the API server's ClusterIP and port, and that value
is the same in every cluster built on the same service network. See [Cross-cluster replay of an
identical request](#cross-cluster-replay-of-an-identical-request).

#### Required signed headers

The `httpSignature` section names headers that every signature must cover, together
with the values this server accepts for each:

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AuthenticationConfiguration
httpSignature:
  requiredSignedHeaders:
    # The header for cluster scoping. `X-K8s-Audience` is conventional for uniquely
    # identifying a cluster
    X-K8s-Audience:
    - https://my-apiserver.example.com
    # An empty list requires the header to be present and covered without
    # constraining its value, for a value only a resolver can interpret.
    X-Session-Token: []
```

The type is a map from header name to accepted values. Three rules apply to each entry, and the
first two reuse machinery the floor and the protected headers already need:

- The header must be present on the request. A missing one is refused, so an intermediary cannot
  strip the check by removing what it checks.
- The signature must cover it. An uncovered one is refused, so an intermediary cannot supply the
  value.
- Where the list is non-empty, the value must equal one of its entries. Comparison is exact and
  case-sensitive.

In this configuration, the `X-K8s-Audience` header names a cluster rather than a route to one, so it
is stable across the several names one cluster legitimately answers to and it does not change when
the cluster gains another. What it adds over `@authority` is a server-held list: the verifier
refuses a value this cluster does not claim, where the authority is whatever the sender wrote.
Listing several accepts a rename or an alias. Listing another cluster's audience reintroduces
exactly the replay this narrows, so an entry belongs here only if it names this cluster.

Where two clusters must accept the same value, which the in-cluster service names force, the
audience stops narrowing anything. See [Cross-cluster replay of an identical
request](#cross-cluster-replay-of-an-identical-request).

The other reserved names may not appear here at all. `Authorization` and the ersonation headers have
their own configuration paths, and the signature fields cannot be written without corrupting
signing. A mechanism able to name any of them would be a way around the paths that own them.

`Forwarded` and `X-Forwarded-*` are never consulted for anything. They are unsigned, so a verifier
reading them would rebuild the signature base from a value the caller controls.

### Client configuration

A kubeconfig user gains an `httpSignature` block holding what does not change. The key comes from an
ordinary exec credential plugin, which supplies what rotates.

```yaml
users:
- name: ecdsa
  user:
    httpSignature:
      apiVersion: client.authentication.k8s.io/v1alpha1
      ttl: 30s
      signedHeaders:
        - name: X-K8s-Audience
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: my-credential-helper
      args: ['credential', '--cluster', 'https://my-apiserver.example.com']
      interactiveMode: Never
```

The plugin is the mechanism that already returns bearer tokens and client certificates
([KEP-541](/keps/sig-auth/541-external-credential-providers)). Client-go sets the `ExecCredential`'s
`spec` with requirements for the credential .

```json
{
  "apiVersion": "client.authentication.k8s.io/v1",
  "kind": "ExecCredential",
  "spec": {
    "interactive": false,
    "signedHeaders": [
      {"name": "X-K8s-Audience"}
    ]
  }
}
```

It returns signing material instead of a carried credential in a new `status.httpSignature` field
along with an algorithm and required signed headers:

```json
{
  "apiVersion": "client.authentication.k8s.io/v1",
  "kind": "ExecCredential",
  "status": {
    "httpSignature": {
      "keyID": "demo-ecdsa-p384",
      "algorithm": "ecdsa-p384-sha384",
      "privateKey": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n",
      "signedHeaders": {
        "X-K8s-Audience": "https://my-apiserver.example.com"
      }
    }
  }
}
```

`client-go` states what headers require a value in `spec.httpSignature`, so the plugin is not
guessing a contract it will be held to. No key and no key ID appears in the kubeconfig, which is
what lets a credential rotate with no kubeconfig edit.

For a symmetric credential the plugin returns `secret` instead of `privateKey`, and the kubeconfig
states a key derivation ladder. The signing key is then derived from the credential rather than
being the credential, so what signs requests is scoped to a purpose and, with a date step, to a day:

```yaml
users:
- name: hmac
  user:
    httpSignature:
      apiVersion: client.authentication.k8s.io/v1alpha1
      ttl: 30s
      keyDerivation:
        kind: hmac-ladder
        hash: sha-256
        secretPrefix: demo1
        steps:
        - {name: day, date: YYYYMMDD}
        - {name: cell, scope: true}
        - {name: terminator, literal: demo1_request}
```

A ladder is not a secret and is not specific to one party. Every party that derives states the same
one, so the client's copy and the resolver's have to agree. A step's input is a literal, a
deployment-scoped value each party supplies, or a date.

The date format comes from a closed set, `YYYYMMDD` or `YYYY-MM-DD`, rather than being a Go layout
or a `strftime` string. A ladder is read by implementations in any language, and a format token has
to mean the same thing to all of them. The date rendered is the signature's own `created` timestamp
in UTC, so the signer and the verifier produce the same value without consulting their own clocks,
including across a date boundary.

The client may hold the root secret and fold the whole chain per request, or the plugin may return
an intermediate rung and say which, in which case the client folds only the remainder. A rung's
bytes cannot say what was folded into them, so the position travels with the material rather than
being inferred.

A shared secret has a property no asymmetric key has: the server holds a copy, so the server can
produce signatures indistinguishable from this client's. Derivation narrows what that copy is good
for. It does not remove the property.

The plugin is called at process launch and prior to expiry, not per request. `client-go` holds a
signer and refreshes it as it nears expiry, so signing costs no process execution. This is the
reason signing lives in `client-go` rather than in the plugin: handing the request bytes to a plugin
and taking headers back would mean one exec per request.

Key derivation could conceivably be performed in a credential process rather than client-go, but
this would have an impact on date-bound derived keys. A day-bound derived signing key expires at UTC
midnight, and requires a new derived key across the day boundary. If derivation were performed in
the credential process, it would have to expire a date-bound derived key at 23:59:59 UTC. If
client-go were to request a new credential prior to expiration, that new credential would need to be
specified as not valid until 00:00:00 UTC. Client-go has no such "not before" credential semantics
today which would be required to support an out-of-process key derivation.

#### A cloud provider example

The `hmac-ladder` shape covers AWS Signature Version 4's key derivation exactly: the secret access
key prefixed with `AWS4`, then a date, a region, a service, and the `aws4_request` terminator. An AWS
credential is therefore usable as signing material with no exchange and no new key to distribute.

```yaml
users:
- name: aws
  user:
    httpSignature:
      apiVersion: client.authentication.k8s.io/v1alpha1
      # Short, because a signature is per request. It bounds how long a captured
      # one is worth replaying.
      ttl: 30s
      keyDerivation:
        kind: hmac-ladder
        hash: sha-256
        secretPrefix: AWS4
        steps:
        - {name: date, date: YYYYMMDD}
        - {name: region, scope: true}
        - {name: service, scope: true}
        - {name: terminator, literal: aws4_request}
      # Names only. Both values come from the credential, which is what keeps a
      # rotating session token out of this file.
      signedHeaders:
      - name: X-Amz-Security-Token
      - name: X-K8s-Audience
    exec:
      apiVersion: client.authentication.k8s.io/v1
      command: awscredential
      args: ['--region', 'us-west-2',
             '--service', 'eks',
             '--cluster', 'arn:aws:eks:us-west-2:111122223333:cluster/my-cluster']
      interactiveMode: Never
```

The plugin returns the secret and a value for each scope step the ladder declares. `keyID` is the
access key ID, which is a name rather than a whole key ID: the client renders the key ID by appending
each step's input to it, which produces SigV4's own credential scope,
`ASIAIOSFODNN7EXAMPLE/20260315/us-west-2/eks/aws4_request`.

```json
{
  "apiVersion": "client.authentication.k8s.io/v1",
  "kind": "ExecCredential",
  "status": {
    "expirationTimestamp": "2026-03-15T18:32:11Z",
    "httpSignature": {
      "keyID": "ASIAIOSFODNN7EXAMPLE",
      "secret": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
      "stage": {
        "scope": {"region": "us-west-2", "service": "eks"}
      },
      "signedHeaders": {
        "X-Amz-Security-Token": "IQoJb3JpZ2luX2VjE...",
        "X-K8s-Audience": "arn:aws:eks:us-west-2:111122223333:cluster/my-cluster"
      }
    }
  }
}
```

`stage` says where in the ladder each returned derivation material sits. It names no step here, so
the material is the root secret and the client folds the whole chain per request. A plugin that
would rather not hand over the root secret returns one rung and names the step it is the output of,
and the client folds only what remains.

Folding the region and the service in means a key that signs Kubernetes requests is not a key that
signs requests to any other AWS service, and the date step makes it a key for one day. The session
token travels as a covered header rather than as hardcoded kubeconfig content.

An access key ID is per-credential, so it cannot be listed in a resolver's `keyIDPrefixes`. A
deployment whose resolver answers for every AWS credential omits that field, which is what makes the
resolver asked about every key ID.

#### A certificate and its key

Where the identity is a certificate, the client states the certificate and its key and nothing else.
The certificate's key type determines the algorithm and the key ID is the certificate's digest:

```yaml
users:
- name: workload
  user:
    httpSignature:
      apiVersion: client.authentication.k8s.io/v1alpha1
      certFile: /var/run/secrets/workload/tls.crt
      keyFile:  /var/run/secrets/workload/tls.key
```

A one-file form, `credentialBundleFile`, is what a `PodCertificateProjection` writes: one read
returns a consistent key and certificate, where two files can be read between the two writes of a
rotation.

### Server configuration

kube-apiserver is configured by an `httpSignature` list in `AuthenticationConfiguration`, the same
file that configures JWT and anonymous authentication. Editing the file takes effect without a
restart.

```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AuthenticationConfiguration
httpSignature:
  # Headers every signature must cover, and the values accepted for each. See
  # "Required signed headers".
  requiredSignedHeaders:
    X-K8s-Audience:
    - https://my-apiserver.example.com
  # A verifier cannot measure a client's clock, so this is
  # a risk budget justified by this server's time synchronization.
  maxClockSkew: 5s
  authenticators:
  - name: workload-certificates
    # Checked against the created timestamp in the signature.
    maxAge: 1m
    x509: {...}
    userValidationRules: [...]
  - name: corp-resolver
    maxAge: 1m
    resolver: {...}
    userValidationRules: [...]
```

Exactly one of `x509` or `resolver` is set per authenticator. `requiredSignedHeaders` and
`maxClockSkew` sit on the section: coverage is decided before an authenticator has been selected,
and clock allowance has one true value per process.

### Identity from a certificate

The client carries its leaf certificate; the server holds the authority that issued
it and nothing per client.

```yaml
  - name: workload-certificates
    maxAge: 1m
    x509:
      # Issue an authority for this purpose. Pointing this at the cluster's client
      # CA would let every certificate already issued for connection
      # authentication sign detached messages, which its issuer never agreed to.
      certificateAuthority: |
        -----BEGIN CERTIFICATE-----
        ...
        -----END CERTIFICATE-----
      # Rules run before the mappings, so a mapping never reads a certificate no
      # rule has vetted.
      certificateValidationRules:
      - expression: cert.notAfter - cert.notBefore <= duration('24h')
        message: certificate lifetime must not exceed 24 hours
      claimMappings:
        username:
          expression: '"cert:" + cert.uriSANs[0]'
        groups:
          expression: cert.subject.organization
    # The mapping above derives groups from the certificate's subject, which hands
    # the choice of group to whoever can request one.
    userValidationRules:
    - expression: '!user.username.startsWith("system:") && !user.groups.exists(g, g.startsWith("system:"))'
      message: 'this authenticator may not assert an identity under the system: prefix'
```

What binds the certificate to the signature is the `keyid`, which must be `x509-sha256:` followed by
the leaf's digest. The verifier recomputes that digest from the bytes it received rather than
trusting the claim.

A certificate selects exactly one authenticator, by the trust anchor its `authorityKeyIdentifier`
names, indexed at configuration load. Both that extension and a `subjectKeyIdentifier` on every
configured anchor are required, which RFC 5280 already requires of conforming issuers. Without this,
every certificate authenticator claims every certificate and a request pays N parses, N signature
verifications, and N chain builds, all but one failing. Trust anchors must be disjoint across
authenticators by identifier and public key.

Extended key usage is not checked. The trust anchor bundle is the opt-in.

The server holds nothing per client, which is the point, so there is nothing to revoke. A
certificate's lifetime is the withdrawal window, narrowed by the validation cache's TTL and by
whatever lifetime rule the configuration states.

### Identity from a resolver

kube-apiserver asks a process over a Unix socket which key verifies a given key ID and whose
identity it is.

```yaml
  - name: corp-resolver
    maxAge: 1m
    resolver:
      endpoint: unix:///var/run/httpsig/resolver.sock
      # Only key IDs whose first segment matches reach this resolver. Omitting
      # this means it is asked about every key ID.
      keyIDPrefixes: [corp]
      # A value the client covers with its signature and this server passes on,
      # for a resolver that decides identity from a session token rather than
      # from a key ID.
      relayedHeaders: [X-Session-Token]
```

[TODO: this could be an extended TokenReview authentication API?]

The protocol is a new staging module, `k8s.io/externalhttpsig`, with no `k8s.io` dependencies so a
resolver author does not import `k8s.io/apiserver`. It follows the shape of the KMS provider and the
external JWT signer rather than the token authentication webhook, so authentication does not depend
on the API server's own storage being available. Three unary RPCs:

| RPC | Purpose |
| --- | --- |
| `Metadata` | Key derivation ladder; doubles as a health probe |
| `ResolveKey` | For a key ID: key material, a `UserInfo`, and a cache duration |
| `ConsumeNonce` | Records the nonce of an accepted signature |

A resolver may return a whole secret or one rung of a key derivation ladder, in which case
kube-apiserver folds the remaining steps per request. That is what lets a resolver hand out a key
scoped to one cluster and one day rather than the root secret.

A resolver states a cache duration per answer and the configuration caps it, so the revocation
window is a number an operator can read. Zero means do not cache, which is correct when the answer
depended on a relayed value that rotates. Answers are cached with bounded size, a bounded negative
entry lifetime, and collapsed concurrent duplicates, because the cache key is peer-chosen. A
resolver that fails takes down only the keys it serves; a failed lookup is not cached, because
caching it would extend an outage past its end.

`relayedHeaders` is named in the API server's own configuration, not requested by the resolver. That
is what keeps this from becoming a session token pattern: a resolver cannot ask for more of the
request than an administrator wrote down.

Key distribution is out of scope. Which party holds which key material and how it gets there is the
resolver operator's problem.

### Verification order

Cost and trust both order this. The first step that fails ends the request.

1. Parse `Signature-Input` and `Signature`. Length caps apply before any allocation proportional to
   the input.
2. Reject a signature whose `created` is outside `maxAge` plus `maxClockSkew`.  This precedes any
   lookup, so an ancient or future timestamp costs nothing.
3. Check that the covered set satisfies the floor, the body rule, and the protected header rule for
   this request.
4. Obtain the key: recompute the certificate digest and select an authenticator by issuer, or
   resolve the key ID.
5. Verify the signature. For the certificate backend this precedes chain building and CEL
   evaluation: proof of possession comes before trust.
6. Build the identity, run `userValidationRules`, and record the nonce where a resolver does that.

### Replay

A captured request can be replayed as itself, unchanged, until it ages out. That
matters most for reads, where replaying a `GET` returns the response again.

`maxAge` plus `maxClockSkew` is the whole of that window unless something records
nonces. Recording requires a store every API server in the set shares: a
per-process cache is not replay protection when a client can reach any API server,
and it is worse than none, because a full store that evicts permits the replay it
was preventing.

The resolver is such a store, so `ConsumeNonce` records the nonce of an accepted
signature there. Records are per key ID, so one client's traffic cannot evict or
reject another's. A full store refuses rather than evicting. A resolver with no
nonce store is a real case, so `resolver.nonceHandling: Ignore` turns the call off
and the API server stops making it. That is stated in configuration rather than
faked with a resolver that always answers yes, because the latter costs a round
trip and leaves nothing an operator can audit.

The certificate backend has no such store, so `maxAge` plus `maxClockSkew` is the
whole window there. The configuration says so structurally: `nonceHandling` lives
inside `resolver`, so there is no field on an `x509` authenticator to set.

The nonce parameter is required on the wire and covered by the signature in both
cases, so recording can begin later with no change at any client.

Everything above is about replay into the same cluster. Replay into a *different*
cluster that trusts the same key is a separate gap, and neither the covered authority
nor the audience closes it. See [Cross-cluster replay of an identical
request](#cross-cluster-replay-of-an-identical-request).

### Metrics

All `ALPHA` stability, under `apiserver_httpsig_`:

| Metric | Labels | Answers |
| --- | --- | --- |
| `signature_outcomes_total` | `authenticator`, `outcome` | Accepted, forged, stale, refused by a user rule. A clock disagreement is counted apart from a bad signature, and a required signed header that was absent, uncovered, or carried an unaccepted value is counted apart from both. |
| `unclaimed_signatures_total` | `reason` | Signatures no authenticator claimed |
| `resolver_request_total` | `resolver`, `method`, `code` | Per-RPC outcome |
| `resolver_request_duration_seconds` | `resolver`, `method`, `code` | Resolver latency on the request path |
| `resolver_metadata_success_timestamp` | `resolver` | Last successful health probe |
| `key_cache_lookup_total` | `resolver`, `result` | Hit, miss, negative hit |
| `resolver_key_derivation_info` | `resolver`, `sha256` | Which derivation ladder this server is using |
| `resolver_nonce_tracking` | `resolver` | Whether nonces are recorded for this resolver |
| `certificate_validation_cache_lookups_total` | `authenticator`, `result` | Certificate validation cache effectiveness |

### Test Plan

[x] I/we understand the owners of the involved components may require updates to
existing tests to make this code solid enough prior to committing the changes
necessary to implement this enhancement.

##### Prerequisite testing updates

None identified.

##### Unit tests

- The coverage rules, shared by client and server, including a signature that covers less than the
  floor, a body with no covered digest, and a protected header present but uncovered.
- Signature base construction against the RFC 9421 test vectors.
- The required parameters, one case each for absent and wrong: `created`, `nonce`, `alg`, `tag`,
  `keyid`. A signature that is cryptographically valid but carries a `tag` other than `kubernetes`
  is refused.
- A signature under a label other than `sig1` verifies, and a request carrying two signatures under
  different labels authenticates if either is acceptable.
- Required signed headers: absent, present but uncovered, present and covered with an unaccepted
  value, present and covered with each accepted value, and an entry with an empty accepted list
  where any value passes.
- `X-K8s-Audience` is refused when absent even though no operator named it, and an entry for it with
  an empty accepted list fails configuration validation.
- Configuration naming a reserved header in `requiredSignedHeaders` fails validation.
- `client-go` refuses to build a signer when no `audience` is configured.
- A signature that does not cover `@authority` is refused, and a request whose `Host` was rewritten
  in flight fails verification rather than authenticating.
- Certificate path: `keyid` digest mismatch, untrusted issuer, expired leaf, missing
  `authorityKeyIdentifier`, overlapping trust anchors across authenticators, key type and size
  bounds.
- Resolver path: cache hit, miss, negative entry expiry, collapsed concurrent lookups, resolver
  failure, `nonceHandling` both ways, key ID prefix routing.
- CEL certificate mappings and `userValidationRules`, including the three names the machinery
  asserts itself.
- Configuration decode and validation, including strict decoding of the `httpSignature` section and
  reload.
- `client-go`: signer construction from each credential form, exec plugin response handling, key
  refresh near expiry.

##### Integration tests

Against a real kube-apiserver:

- Both identity backends authenticate a request and produce the expected `UserInfo` and groups.
- A tampered key is rejected, and rejected in a way that cannot be explained by another
  authenticator admitting the request.
- A certificate from an untrusted authority is rejected.
- A signature replayed against a different path, verb, or body is rejected.
- A signature carrying another cluster's `X-K8s-Audience` is rejected, and the same request with
  this cluster's audience is accepted.
- A request whose `Host` was rewritten between client and server fails, which is what says the
  authority is covered.
- A signature outside `maxAge` is rejected.
- An added, uncovered impersonation header is rejected.
- Editing `AuthenticationConfiguration` takes effect without a restart, and a reload that fails
  validation leaves the previous configuration in place.
- With the feature gate off, a signed request is not authenticated and the configuration section is
  rejected.

##### e2e tests

A cluster brought up with a resolver and a certificate authority configured, and `kubectl`
authenticating by signature through an exec credential plugin.

### Graduation Criteria

#### Alpha

- Feature gate `HTTPSignatureAuthentication`, off by default.
- `httpSignature` in `AuthenticationConfiguration` at `v1alpha1`, with both identity backends:
  `x509` and `resolver`, and `requiredSignedHeaders`.
- `httpSignature` in kubeconfig and `ExecCredential` at `client.authentication.k8s.io/v1alpha1`,
  with asymmetric keys, symmetric keys, and the HMAC derivation ladder.
- `k8s.io/externalhttpsig` as a staging module.
- A reference `AuthenticationConfiguration` covering both backends, validated on every test run by
  the API server's own decode and validation.
- Unit and integration tests above.
- Metrics above.

Both backends are in alpha rather than staged. They are not two implementations of one thing: the
certificate backend depends on nothing at request time and cannot carry a symmetric key, and the
resolver backend is the only one that supports HMAC, key derivation, and nonce recording. Shipping
only the certificate backend would ship the path that does not serve the motivating case of a caller
whose existing credential is a shared secret.

#### Beta

- e2e tests.
- Replay narrower than the acceptance window, or a stated decision not to narrow it for the
  certificate backend.
- A mechanism preventing replay of an identical request into another cluster that trusts the same
  key, or a stated non-goal with the precondition spelled out.
- A bound on resolver calls from an unauthenticated caller, informed by the metrics above, or
  evidence that the existing bounds suffice.
- Scalability numbers for verification cost on the request path.
- Ownership of the RFC 9421 and structured field values implementations settled.
- Documentation on kubernetes.io.

#### GA

- Two releases at beta with no unresolved issues.
- Conformance considerations settled. An authentication format nobody is required
  to enable is not obviously a conformance surface.

### Upgrade / Downgrade Strategy

The feature is per-request and additive. Enabling it means an API server accepts one more credential
format; disabling it means signed requests are no longer authenticated and clients configured to
sign fall back to nothing, so they receive
401. No stored object changes, so downgrade needs no data migration.

An operator turning the gate off should expect requests from signing clients to fail until those
clients are reconfigured. Configuring the server first and the clients second has no failure window.

### Version Skew Strategy

A `client-go` newer than the API server signs requests an older server does not recognize, and
receives 401. There is no negotiation. A `kubectl` older than its kubeconfig ignores the
`httpSignature` block and sends no credential, which is the same 401 by a different route.

kube-apiserver and a resolver skew independently. `Metadata` is the version handshake and doubles as
a health probe: a resolver that cannot answer it is not used, and the health gate runs before a
configuration swap takes effect.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

###### How can this feature be enabled / disabled in a live cluster?

- [x] Feature gate
  - Feature gate name: `HTTPSignatureAuthentication`
  - Components depending on the feature gate: kube-apiserver
- [x] Other
  - Describe the mechanism: an `httpSignature` section in `AuthenticationConfiguration`. With the
    gate on and no section present, nothing changes.
  - Will enabling / disabling the feature require downtime of the control plane?  No. The gate
    requires a kube-apiserver restart; the configuration section is reloadable.
  - Will enabling / disabling the feature require downtime or reprovisioning of a node? No.

###### Does enabling the feature change any default behavior?

No. With no `httpSignature` section configured, no request is authenticated differently.

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

Yes. Clients that authenticate by signature will receive 401 until they are given another
credential.

###### What happens if we reenable the feature if it was previously rolled back?

Signed requests authenticate again. There is no persisted state.

###### Are there any tests for feature enablement/disablement?

Yes: with the gate off, the configuration section is rejected and a signed request
is not authenticated.

### Rollout, Upgrade and Rollback Planning

###### How can a rollout or rollback fail? Can it impact already running workloads?

A rollout cannot affect a workload using any other credential. A misconfigured `httpSignature`
section fails validation and the previous configuration stays in effect. A resolver that is
unreachable rejects requests bearing the keys it serves and nothing else.

###### What specific metrics should inform a rollback?

`apiserver_httpsig_signature_outcomes_total` with a non-accepted outcome,
`apiserver_httpsig_unclaimed_signatures_total`, and
`apiserver_httpsig_resolver_request_total` with a non-OK code.

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

To be completed before beta.

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

No.

### Monitoring Requirements

###### How can an operator determine if the feature is in use by workloads?

`apiserver_httpsig_signature_outcomes_total` is non-zero.

###### How can someone using this feature know that it is working for their instance?

- [x] API .status
  - Other field: a `SelfSubjectReview` returns the identity the signature resolved to.
- [x] Other
  - Details: `apiserver_httpsig_signature_outcomes_total{outcome="accepted"}` increasing for the
    authenticator in question.

###### What are the reasonable SLOs (Service Level Objectives) for the enhancement?

Signature verification adds latency to the authentication phase of a request. For the certificate
backend that is one signature verification plus a cached certificate validation. For the resolver
backend it is one signature verification plus, on a cache miss, a Unix socket round trip.

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

- [x] Metrics
  - Metric name: `apiserver_httpsig_resolver_request_duration_seconds`,
    `apiserver_httpsig_key_cache_lookup_total`,
    `apiserver_httpsig_certificate_validation_cache_lookups_total`
  - Components exposing the metric: kube-apiserver

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

A histogram of verification CPU time per algorithm would inform the scalability answer below. To be
decided before beta.

### Dependencies

###### Does this feature depend on any specific services running in the cluster?

The certificate backend depends on nothing at request time. The resolver backend depends on a
process reachable on a Unix socket on each control plane node. Its availability becomes the
availability of authentication for the keys it serves, and nothing more.

<<[UNRESOLVED sig-auth ]>>
An RFC 9421 implementation and a structured field values implementation are required, and there is
no mature Go implementation of either in wide use. Whether they are donated to `kubernetes-sigs`,
reimplemented in staging, or depended on as external modules is unsettled. See [Who owns the RFC
9421 and structured field values
implementations](#who-owns-the-rfc-9421-and-structured-field-values-implementations).
<<[/UNRESOLVED]>>

### Scalability

###### Will enabling / using this feature result in any new API calls?

No calls to the Kubernetes API. The resolver backend makes one `ResolveKey` call per cache miss and
one `ConsumeNonce` call per accepted signature where nonce recording is on, both over a local Unix
socket.

###### Will enabling / using this feature result in introducing new API types?

`AuthenticationConfiguration` gains an `httpSignature` section. `ExecCredential` and kubeconfig
`AuthInfo` gain an `httpSignature` field. The resolver protocol is a new staging module, not a
Kubernetes API type.

###### Will enabling / using this feature result in any new calls to the cloud provider?

No.

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

Yes, for requests that authenticate by signature: one signature verification, and on a resolver
cache miss one local round trip. Requests using other credentials are unaffected. Numbers to be
gathered before beta.

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

Verification is CPU on the request path and is reachable by an unauthenticated caller. Caches for
resolved keys and validated certificates are bounded in size.

###### Can enabling / using this feature result in resource exhaustion of some node resources (PIDs, sockets, inodes, etc.)?

See the unauthenticated lookup risk above.

### Troubleshooting

###### How does this feature react if the API server and/or etcd is unavailable?

Neither backend reads from etcd, so authentication by signature does not depend on
storage.

###### What are other known failure modes?

- **Signature does not verify, cause unknown.** The client and the server each construct a signature
  base; a mismatch anywhere produces one indistinguishable failure. The signing round tripper logs
  `Signature-Input` and `Signature` whole at verbosity 7, and the verifier logs the base it
  reconstructed at the same verbosity, so the two can be compared. Neither is masked, because a
  masked signature cannot be compared against what the verifier built.
- **Derivation ladder mismatch.** Where a client and a resolver both state a key derivation ladder
  and the two disagree, every signature fails to verify. Both publish a digest of theirs, the
  server's as a metric label, so the comparison is one metric read against one log line.
- **Intermediary rewrote a covered component.** Presents as a signature that does not verify, from
  every client behind that intermediary at once.
- **Clock skew.** Counted separately from a bad signature in
  `apiserver_httpsig_signature_outcomes_total`.
- **Resolver unreachable.** Requests bearing that resolver's keys are rejected.
  `apiserver_httpsig_resolver_metadata_success_timestamp` stops advancing.

###### What steps should be taken if SLOs are not being met to determine the problem?

Compare `key_cache_lookup_total` hit rate and `resolver_request_duration_seconds`.  A low hit rate
with a peer-chosen cache key suggests key ID churn rather than a slow resolver.

## Open Questions

### Where should `httpSignature` live in a kubeconfig?

The prototype today has it exist as a peer to `exec` with its own apiversion. Should it:
* remain as a peer? For certificate-based signing, no `exec` is required.
* Move under `exec`?
* Go somewhere else?

### Who owns the RFC 9421 and structured field values implementations

Signing and verifying require an implementation of [RFC
9421](https://www.rfc-editor.org/rfc/rfc9421.html) and of the [RFC
9651](https://www.rfc-editor.org/rfc/rfc9651.html) structured field values it is built on. There is
no mature Go implementation of either in wide use. Prototypes exist as
`github.com/micahhausler/httpsig` and `github.com/micahhausler/sfv`.

Three options:

- Donate both to a `kubernetes-sigs` repository, and have `client-go` and `k8s.io/apiserver` depend
  on that. Names an owner without putting an RFC implementation in the `k/k` tree.
- Reimplement in staging. No external dependency; SIG Auth owns the RFC implementation permanently.
- Depend on the external modules as they are.

The author is willing to donate and has no preference among the outcomes. This is a KEP question
rather than an implementation question, because it decides who maintains an RFC implementation
sitting on the authentication path of every request.  It must be settled before beta.

### What bounds resolver calls from an unauthenticated caller

A key lookup precedes signature verification, because verification needs the key, so an
unauthenticated caller can drive a resolver call. Length caps, an age check before the lookup,
collapsed concurrent duplicates, and a short memory of unknown key IDs all reduce the cost. None
caps the rate for a caller cycling through distinct key IDs.

Two places could hold a limit and it is not obvious which. The resolver knows its own capacity and
is reachable only over a socket an administrator controls. The API server has priority and fairness
in front of every request, which bounds concurrency but is scoped to resources rather than to
authenticators. A third limiter inside the authenticator would be a knob whose right value nobody
can state, which is the argument for measuring first: `apiserver_httpsig_resolver_request_total` and
`apiserver_httpsig_key_cache_lookup_total` are what say whether this is a real problem or a
predicted one.

The certificate backend does not have this exposure.

### Cross-cluster replay of an identical request

Nothing in this design stops a captured request being replayed, unchanged, into a different cluster
that trusts the same key, inside the acceptance window and the `@authority` component does not
resolve this.

`@authority` is reconstructed from what the sender put on the wire, and nothing validates that
against the connection: Go's HTTP/2 server does not compare `:authority` with the TLS server name,
and no API server filter reads it. Whoever replays the request supplies the authority the signature
covered.  `X-K8s-Audience` closes it only where the two clusters accept disjoint values, and the
in-cluster service names defeat that: every cluster answers to `https://kubernetes.default.svc`, so
both must list it, so a signature made for one satisfies the other.

The precondition is a shared key, which narrows who is exposed. Two clusters trusting one
certificate authority, or one resolver serving both, or one HMAC secret derived without a cluster
step in its ladder. A deployment that issues per-cluster trust anchors, or puts a cluster scope step
in its derivation ladder, already has distinct keys and is not affected. What is missing is a
mechanism that does not depend on the operator having arranged that.

Candidate directions:

- **Bind the audience to something a cluster cannot share.** A value derived at install time rather
  than from a DNS name, stated in configuration and in each client. The cost is a value every client
  has to be told, which the in-cluster case currently gets for free from its environment.
- **Let the resolver decide.** `ResolveKey` already carries relayed header values, so a resolver
  serving several clusters could refuse a key for the wrong one. This works only for the resolver
  backend and moves the check out of kube-apiserver.
- **Record nonces per cluster.** `ConsumeNonce` at a resolver shared by both clusters would reject
  the second use of a nonce, which turns cross-cluster replay into the same problem as replay within
  one cluster. Depends on a shared resolver, so it does not help the certificate backend.
- **Cover a component only the intended server can reconstruct.** A channel binding to the TLS
  exporter of the client's own connection would be exact, and is unavailable to a client behind a
  TLS-terminating proxy, which is the deployment this feature is for.

`X-K8s-Audience` is conventional in this KEP, and there is not a way to define required signed
headers or values outside of a kubeconfig. For in-cluster clients, such as pod certificates,
no mechanism exists to introduce cluster scoping.

This has to be settled before beta. It is stated here rather than left implicit because the KEP
claims a signature is bound to its request, and "its request" turns out to mean the request rather
than the request together with its destination.


### Whether replay is narrowed below the acceptance window for certificates

`ConsumeNonce` gives the resolver backend a store every API server shares. The certificate backend
has none, so `maxAge` plus `maxClockSkew` is its whole replay window.

The nonce is required on the wire and covered by the signature in both cases, so recording can begin
later with no client change. What is missing is a design rather than a constant. A bucket keyed on
the trust anchor would put every client under one authority into one shared, peer-driven cache,
which is the arrangement that turns replay tracking into a replay enabling mechanism. Pointing a
certificate authenticator at a resolver purely to record nonces was considered and rejected: it
would make `endpoint` mean two different things depending on whether `x509` was also set, and cost a
round trip per request for a client whose key the resolver never sees.

### Aggregated API servers, admission, and conversion webhooks

An extension API server behind the aggregation layer never sees the client's signature.
kube-apiserver authenticates the request and re-originates it with its own client certificate and
`X-Remote-User` headers. This composes correctly and it means the properties this KEP claims stop at
kube-apiserver. Resigning to an aggregate, with a client's derived key or with the API server's own
identity, is possible future work and is not proposed here.

## Implementation History

- 2026-08-26: discussed at SIG Auth. Prototype presented; no KEP filed yet.
- 2026-09-08: this KEP drafted from the prototype branch.

## Drawbacks

Kubernetes would own an implementation of RFC 9421 and of RFC 9651 structured field values on the
authentication path of every request, in a language with no mature implementation of either. That is
the cost Mo Khan named in the SIG Auth discussion, and it is real regardless of how the library
ownership question is settled.

The covered header set is a contract between client and server that grows. Adding a
security-relevant header to the Kubernetes API means adding it to the protected set, which means
older clients do not cover it and newer servers must decide whether to reject them.

The signature format is HTTP-shaped, and some Kubernetes traffic is not: WebSocket frames, SPDY
streams. Those are authenticated by their upgrade request and not beyond it.

## Alternatives

### DPoP

[RFC 9449](https://www.rfc-editor.org/rfc/rfc9449.html) binds a public key into a token at issuance
and requires a proof signed by the matching private key on each request. Any identity provider
implementing it gives Kubernetes proof of possession through the OIDC path already configured.

DPoP and RFC 9421 are unrelated RFCs solving adjacent problems. DPoP requires an identity provider
that issues confirmation-bound tokens, and a token still exists.  It is complementary rather than an
alternative: a cluster could want both.

### Send the request to a webhook and verify there

Ship the request to a `TokenReview`-style webhook and have it verify the signature. Rejected:
byte-exact verification after re-serialization through another system's request format is fragile in
a way that fails closed but unreadably, and the webhook would need the whole request.

### A signing proxy on the client side

Run a local proxy that signs requests, so no client library changes. Proposed before in [KEP
2718][kep2718] and never implemented. It moves the credential to a listening socket, adds a process
to every client's deployment, and does not work for a client that cannot be pointed at a proxy.
Signing in `client-go` is preferred, and a server-side proxy in front of an API server that does not
support the format remains available to anyone who wants it.

[kep2718]: https://github.com/kubernetes/enhancements/issues/2718

## Infrastructure Needed (Optional)

If the RFC 9421 and structured field values libraries are donated, a `kubernetes-sigs` repository
for each, or one holding both.
