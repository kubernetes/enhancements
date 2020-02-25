---
title: Support external signing of service account keys
authors:
  - "@micahhausler"
owning-sig: sig-auth
participating-sigs: []
reviewers:
  - "@mikedanese"
  - "@liggitt"
  - "@tallclair"
  - "@enj"
approvers:
  - "@mikedanese"
  - "@liggitt"
  - "@tallclair"
editor: '@micahhausler'
creation-date: 2019-01-16
last-updated: 2020-02-25
status: implementable
see-also: []
replaces: []
superseded-by: []
---

# Support external signing of service account keys

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Preserve existing behavior](#preserve-existing-behavior)
  - [Updates to API server token generation](#updates-to-api-server-token-generation)
  - [New API](#new-api)
  - [Implementation Details/Notes/Constraints](#implementation-detailsnotesconstraints)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Drawbacks [optional]](#drawbacks-optional)
- [Alternatives [optional]](#alternatives-optional)
- [Infrastructure Needed [optional]](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

**ACTION REQUIRED:** In order to merge code into a release, there must be an issue in [kubernetes/enhancements] referencing this KEP and targeting a release milestone **before [Enhancement Freeze](https://github.com/kubernetes/sig-release/tree/master/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core Kubernetes i.e., [kubernetes/kubernetes], we require the following Release Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These checklist items _must_ be updated for the enhancement to be released.

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR) ([#740](https://github.com/kubernetes/enhancements/issues/740))
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

**Note:** Any PRs to move a KEP to `implementable` or significant changes once it is marked `implementable` should be approved by each of the KEP approvers. If any of those approvers is no longer appropriate than changes to that list should be approved by the remaining approvers and/or the owning SIG (or SIG-arch for cross cutting KEPs).

**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://github.com/kubernetes/enhancements/issues
[kubernetes/kubernetes]: https://github.com/kubernetes/kubernetes
[kubernetes/website]: https://github.com/kubernetes/website

## Summary

The Kubernetes API server has always read service account keys from disk as the
process starts, and kept them in memory for the duration of the server's
lifetime. As the API server can now verify and issue projected volume tokens, it
would be advantageous to support external signing and verifying of token data
over an API, as well as reading public keys from an API.

## Motivation

For operators who want to regularly rotate the signing and verifying keys for
projected volume tokens, the Kubernetes API server must be restarted in order to
use a new key. To facilitate easy key rotation, this KEP includes an proposal
for a grpc API to support out of process signing and listing of signing keys.
Additionally,

### Goals

- Support for out-of-process JWT signing
- Support for listing public verifying keys
- Preserve existing behavior and performance for keys not read over a socket

### Non-Goals

- Reading TLS serving certificates and key from a socket or reloading of the API
    server with new cert and key
- Reading any other certificates from a file

## Proposal

### Preserve existing behavior

The API server flags `--service-account-key-file` and
`--service-account-signing-key-file` will continue be used for reading from
files.

### Updates to API server token generation

As of Kubernetes v1.13.2, the API server uses the functions `JWTTokenGenerator`
and `JWTTokenAuthenticator`. New types that implement the `TokenGenerator`
interface and support token validation will be added to
`k8s.io/kubernetes/pkg/serviceaccount/`.

### New API

I'm proposing creating a new versioned grpc API under
`k8s.io/kubernetes/pkg/serviceaccount`. This will be similar to how the KMS
envelope encryption has an API at
`k8s.io/kubernetes/staging/src/k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1/service.proto`

```proto
syntax = "proto3";

package v1alpha1;

service KeyService {
  // Sign an incoming payload
  rpc SignPayload(SignPayloadRequest) returns (SignPayloadResponse) {}
  // List all active public keys
  rpc ListPublicKeys(ListPublicKeysRequest) returns (ListPublicKeysResponse) {}
}

message SignPayloadRequest {
  // payload is the content to be signed. JWT headers must be included by the caller
  bytes payload = 1;
  // algorithm specifies which algorithm to sign with
  string algorithm = 2;
}
message SignPayloadResponse {
  // content returns the signed payload
  bytes content = 1;
}


message PublicKey {
  // public_key is a PEM encoded public key
  bytes public_key = 1;
  // certificate is a concatenated list of PEM encoded x509 certificates
  bytes certificates = 2;
  // key_id is the key's ID
  string key_id = 3;
  // algorithm states the algorithm the key uses
  string algorithm = 4;
}

message ListPublicKeysRequest {}
message ListPublicKeysResponse {
  // key_id is the key's ID
  string active_key_id = 1;
  // public_keys is a list of public verifying keys
  repeated PublicKey public_keys = 2;
}
```

### Implementation Details/Notes/Constraints

The API server flag `--service-account-key-file` can be specified multiple times
for legacy SA tokens and projected tokens. Validation keys from this flag will
be merged with the response of `ListPublicKeys()`. A new flag
`--key-service-url` will be added to the API server specifying a unix socket
where the key service will be accessible.

### Risks and Mitigations

New token generation and validation could suffer a performance difference when
reading over a socket, as an external process will be signing data or delegating
off host.

Signing and verifying tokens over a grpc API carries the risk of a server side
request forgery, where a malicious client could generate tokens. To mitigate
this risk, the API will only be accessible over a unix socket.

Additionally, if the signing server becomes unavailable, creation of new tokens
and verificaiton of existing tokens would be impacted.

## Design Details

### Test Plan

Unit tests covering:
* ProjectedToken signing works with external signer client
* OIDC keys URL shows public keys from external signer

Integration tests covering:
* ProjectedToken signing works with simple signer server
* OIDC keys URL shows public keys from external signer
* Validate apiserver flag logic (require issuer to be set)
* Validate that legacy Service Account tokens can be validated while external
    signer is down
* Validate that projected token signing fails while external signer is down

E2E tests covering:
* ProjectedToken signing works with simple signer server
* Pods with projected tokens still create if signing server response times are
    over 5s
* Pods without projected tokens are not impacted if external signer is down

### Graduation Criteria

* Alpha v1.19
    * Start API at v1alpha1
    * Proto API generator scripts
    * Unit test coverage of new work
* Beta
    * Migrate API to v1beta1
    * Performance metrics are added
    * Integration tests added
    * Multiple implementations are available or in use
    * Collect user feedback and usage reports
    * Documentation added to website
* Stable/GA
    * Migrate API to v1
    * E2E tests added with 2 different implementations

### Upgrade / Downgrade Strategy

TODO

### Version Skew Strategy

TODO

## Implementation History

* Initial PR: kubernetes/kubernetes#73110

## Drawbacks [optional]

TODO

## Alternatives [optional]

To achieve signing key rotation without a server restart, the API server could
be modified to re-read signing keys from disk periodically.

To achieve external signing, PKCS#11 is an industry standard that supports
external signing of data. At the time of writing, there is not a go library that
implements the PKCS#11 interface without resorting to using CGO.

## Infrastructure Needed [optional]

In order to graduate Stable/GA, at least two different public implementations
will need to be avaialable for E2E tests.
