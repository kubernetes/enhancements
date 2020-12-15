---
title: Support external signing of service account keys
authors:
  - "@micahhausler"
owning-sig: sig-auth
participating-sigs: []
reviewers:
  - "@mikedanese"
  - "@liggit"
  - "@tallclair"
approvers:
  - "@mikedanese"
  - "@liggit"
  - "@tallclair"
editor: '@micahhausler'
creation-date: 2019-01-16
last-updated: 2019-05-17
status: implementable
see-also: []
replaces: []
superseded-by: []
---

# Support external signing of service account keys

## Table of Contents

<!-- toc -->
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
- [Graduation Criteria](#graduation-criteria)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

The Kubernetes API server has always read service account keys from disk as the process starts, and kept them in memory for the duration of the server's lifetime. As the API server can now verify and issue projected volume tokens, it would be advantageous to support external signing and verifying of token data over an API, as well as reading public keys from an API.

## Motivation

For operators who want to regularly rotate the signing and verifying keys for projected volume tokens, the Kubernetes API server must be restarted in order to use a new key. To facilitate easy key rotation, this KEP includes an proposal for a grpc API to support out of process signing and listing of signing keys.

### Goals

- Support for out-of-process JWT signing
- Support for listing public verifying keys
- Preserve existing behavior and performance for keys not read over a socket

### Non-Goals

- Reading TLS serving certificates and key from a socket or reloading of the API server with new cert and key
- Reading any other certificates from a file

## Proposal

### Preserve existing behavior

The API server flags `--service-account-key-file` and `--service-account-signing-key-file` will continue be used for reading from files.

### Updates to API server token generation

As of Kubernetes v1.13.2, the API server uses the functions `JWTTokenGenerator` and `JWTTokenAuthenticator`. New types that implement the `TokenGenerator` interface and support token validation will be added to `k8s.io/kubernetes/pkg/serviceaccount/`.

### New API

I'm proposing creating a new versioned grpc API under `k8s.io/kubernetes/pkg/serviceaccount`. This will be similar to how the KMS envelope encryption has an API at `k8s.io/kubernetes/staging/src/k8s.io/apiserver/pkg/storage/value/encrypt/envelope/v1beta1/service.proto`

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

The API server flag `--service-account-key-file` can be specified multiple times for legacy SA tokens and projected tokens. Validation keys from this flag will be merged with the response of `ListPublicKeys()`. A new flag `--key-service-url` will be added to the API server specifying a unix socket where the key service will be accessible.

### Risks and Mitigations

New token generation and validation could suffer a performance difference when reading over a socket, as an external process will be signing data.

Signing and verifying tokens over a grpc API carries the risk of a server side request forgery, where a malicious client could generate tokens. To mitigate this risk, the API will only be accessible over a unix socket.

## Graduation Criteria

<!-- TODO -->

## Implementation History

* Initial PR: kubernetes/kubernetes#73110
