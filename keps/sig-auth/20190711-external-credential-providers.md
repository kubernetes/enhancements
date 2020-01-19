---
title: External credential providers
authors:
  - "@awly"
owning-sig: sig-auth
participating-sigs:
  - sig-cli
  - sig-api-machinery
reviewers:
  - "@liggitt"
  - "@mikedanese"
approvers:
  - "@liggitt"
  - "@mikedanese"
creation-date: 2019-07-11
last-updated: 2019-07-11
status: implementable
---

# External credential providers

## Table of Contents

<!-- toc -->
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Provider configuration](#provider-configuration)
  - [Provider input format](#provider-input-format)
  - [Provider output format](#provider-output-format)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Client authentication to the binary](#client-authentication-to-the-binary)
    - [Invalid credentials before cache expiry](#invalid-credentials-before-cache-expiry)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta](#beta)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
- [Alternatives](#alternatives)
  - [RPC vs exec](#rpc-vs-exec)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Summary

External credential providers allow out-of-tree implementation of obtaining
client authentication credentials. These providers handle environment-specific
provisioning of credentials (such as bearer tokens or TLS client certificates)
and expose them to the client.

## Motivation

Client authentication credentials for Kubernetes clients are usually specified
as fields in `kubeconfig` files. These credentials are static and must be
provisioned in advance.

This creates 3 problems:

1. Credential rotation requires client process restart and extra tooling.
1. Credentials must exist in a plaintext, file on disk.
1. Credentials must be long-lived.

Many users already use key management/protection systems, such as Key
Management Systems (KMS), Trusted Platform Modules (TPM) or Hardware Security
Modules (HSM). Others might use authentication providers based on short-lived
tokens.

Standard Kubernetes client authentication libraries should support these
systems to help with key rotation and protect against key exfiltration.

### Goals

1. Credential rotation without client restart.
1. Support standard key management solutions.
1. Support standard token-based protocols.
1. Provisioning logic lives outside of Kubernetes codebase.
1. Kubernetes interface is vendor-neutral.

### Non-Goals

1. Exfiltration protection built into Kubernetes.
1. Kubernetes triggering rotation.
1. Deprecation of existing authentication options.

## Proposal

A new authentication flow in libraries around `kubeconfig` based on
executables. Before performing a request, client executes a binary and uses its
output for authentication.

There are 2 modes of authentication:

1. bearer tokens
1. mTLS

Provider response is cached and reused in future requests.

Client is configured with a binary path, optional arguments and environment
variables to pass to it.

### Provider configuration

Configuration is provided via users section of `kubeconfig` file:

```
apiVersion: v1
kind: Config
users:
- name: my-user
  user:
    exec:
      # API version to use when decoding the ExecCredentials resource. Required.
      apiVersion: "client.authentication.k8s.io/<version>"

      # Command to execute. Required.
      command: "example-client-go-exec-plugin"

      # Arguments to pass when executing the plugin. Optional.
      args:
      - "arg1"
      - "arg2"

      # Environment variables to set when executing the plugin. Optional.
      env:
      - name: "FOO"
        value: "bar"
clusters:
- name: my-cluster
  cluster:
    server: "https://1.2.3.4:8080"
    certificate-authority: "/etc/kubernetes/ca.pem"
contexts:
- name: my-cluster
  context:
    cluster: my-cluster
    user: my-user
current-context: my-cluster
```

`apiVersion` specifies the expected version of this API that the plugin
implements. If the version differs, client must return an error.

`command` specifies the path to the provider binary. The file at this path must
be readable and executable by the client process.

`args` specifies extra arguments passed to the executable.

`env` specifies environment variables to pass to the provider. The environment
variables set in the client process are not passed.

### Provider input format

```
{
  "apiVersion": "client.authentication.k8s.io/<version>",
  "kind": "ExecCredential"
}
```

Provider can safely ignore `stdin` since input object doesn't carry any data.

### Provider output format

```
{
  "apiVersion": "client.authentication.k8s.io/<version>",
  "kind": "ExecCredential",
  "status": {
    "expirationTimestamp": "$EXPIRATION",
    "token": "$BEARER_TOKEN",
    "clientKeyData": "$CLIENT_PRIVATE_KEY",
    "clientCertificateData": "$CLIENT_CERTIFICATE",
  }
}
```

`EXPIRATION` contains the RFC3339 timestamp with credential expiry. Client can
cache provided credentials until this time.

After `EXPIRATION`, client must execute the provider again for any new
connections. For `client_key` mode, this applies even if returned certificate
is still valid.

`BEARER_TOKEN` contains a token for use in `Authorization` header of HTTP
requests.

`CLIENT_PRIVATE_KEY` and `CLIENT_CERTIFICATE` contain client TLS credentials in
PEM format. The certificate must be valid at the time of execution. These
credentials are used for mTLS handshakes.

### Risks and Mitigations

#### Client authentication to the binary

Credential provider can authenticate the caller via env vars or arguments
specified in its `kubeconfig`. This is optional.

It is recommended to restrict access to the binary using exec Unix permissions.

#### Invalid credentials before cache expiry

Credentials may become invalid (e.g. expire) after being returned by the
provider but before `expirationTimestamp` in the returned `ExecCredential`.

Credential provider should ensure validity of the credentials it returns and
return an error if it can't provide valid credentials.

In case client gets `401 Unauthorized` or `403 Forbidden` response status from
remote endpoint when using credentials from a provider, client should
re-execute the provider, ignoring `expirationTimestamp`.

### Graduation Criteria

#### Beta

Feature is already in Beta.

#### Beta -> GA Graduation

- 3 examples of real world usage
- support for remote TLS handshakes (e.g. TPM/KMS-hosted keys)

## Alternatives

### RPC vs exec

Credential provider could be exposed as a network endpoint. Instead of
executing a binary and passing request/response over `stdin`/`stdout`, client
could open a network connection and send request/response over that.

The downsides of this approach compared to exec model are:

- if credential provider is remote, design for client authentication is
  required (aka "chicken-and-egg problem")
- credential provider must constantly run, consuming resources; clients refresh
  their credentials infrequently

## Implementation History

- 2018-01-29: Proposal submitted https://github.com/kubernetes/community/pull/1503
- 2018-02-28: Alpha implemented https://github.com/kubernetes/kubernetes/pull/59495
- 2018-06-04: Promoted to Beta https://github.com/kubernetes/kubernetes/pull/64482
