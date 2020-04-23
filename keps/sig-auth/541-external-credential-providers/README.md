# KEP-541: External credential providers

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
- [Design Details](#design-details)
  - [Provider configuration](#provider-configuration)
  - [Provider input format](#provider-input-format)
  - [Provider output format](#provider-output-format)
  - [Risks and Mitigations](#risks-and-mitigations)
    - [Client authentication to the binary](#client-authentication-to-the-binary)
    - [Invalid credentials before cache expiry](#invalid-credentials-before-cache-expiry)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
    - [Beta](#beta)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [RPC vs exec](#rpc-vs-exec)
- [Implementation History](#implementation-history)
<!-- /toc -->

## Release Signoff Checklist

- [ ] Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [x] KEP approvers have approved the KEP status as `implementable`
- [ ] Design details are appropriately documented
- [ ] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

External credential providers allow out-of-tree implementation of obtaining
client authentication credentials. These providers handle environment-specific
provisioning of credentials (such as bearer tokens or TLS client certificates)
and expose them to the client.

## Motivation

Client authentication credentials for Kubernetes clients are usually specified
as fields in `kubeconfig` files. These credentials are static and must be
provisioned in advance.

This creates three problems:

1. Credential rotation requires client process restart and extra tooling.
2. Credentials must exist in a plaintext file on disk.
3. Credentials must be long-lived.

Many users already use key management/protection systems, such as Key
Management Systems (KMS), Trusted Platform Modules (TPM) or Hardware Security
Modules (HSM). Others might use authentication providers based on short-lived
tokens.

Standard Kubernetes client authentication libraries should support these
systems to help with key rotation and protect against key exfiltration.

### Goals

1. Credential rotation without client restart.
2. Support standard key management solutions.
3. Support standard token-based protocols.
4. Provisioning logic lives outside of Kubernetes codebase.
5. Kubernetes interface is vendor-neutral.
6. Deprecation of `gcp` and `azure` authentication options (`keystone` has
   already been deprecated and removed).

### Non-Goals

1. Exfiltration protection built into Kubernetes.
2. Kubernetes triggering rotation.
3. Deprecation of the `oidc` authentication option.

## Proposal

A new authentication flow in libraries around `kubeconfig` based on
executables. Before performing a request, client executes a binary and uses its
output for authentication.

There are two modes of authentication:

1. Bearer tokens
2. mTLS

Provider response is cached in-memory by the client and reused in future
requests (no caching is done across different executions of the client).

Client is configured with a binary path, optional arguments and environment
variables to pass to it.

## Design Details

### Provider configuration

Configuration is provided via users section of `kubeconfig` file:

```yaml
apiVersion: v1
kind: Config
users:
- name: my-user
  user:
    exec:
      # API version to use when decoding the ExecCredentials resource. Required.
      apiVersion: "client.authentication.k8s.io/v1beta1"

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
variables set in the client process are also passed to the provider.

### Provider input format

```json
{
  "apiVersion": "client.authentication.k8s.io/v1beta1",
  "kind": "ExecCredential",
  "spec": {
    "cluster": {
      "server": "https://1.2.3.4:8080",
      "certificate-authority": "/etc/kubernetes/ca.pem"
    }
  }
}
```

The `spec.cluster` field is the current cluster that the client is communicating
with.  This struct is defined in `k8s.io/client-go/tools/clientcmd/api/v1` (it
is used in the `kubeconfig` file format).  This allows the executable to perform
different actions based on the current cluster (i.e. get a token for a particular
cluster).

This data is passed to the executable via the `KUBERNETES_EXEC_INFO` environment
variable in a JSON serialized object.

### Provider output format

```json
{
  "apiVersion": "client.authentication.k8s.io/v1beta1",
  "kind": "ExecCredential",
  "status": {
    "expirationTimestamp": "$EXPIRATION",
    "token": "$BEARER_TOKEN",
    "clientKeyData": "$CLIENT_PRIVATE_KEY",
    "clientCertificateData": "$CLIENT_CERTIFICATE",
  }
}
```

`expirationTimestamp` specifies the RFC3339 timestamp with credential expiry.
Client can cache provided credentials in-memory until this time.  If no
`expirationTimestamp` is provided, credentials will be cached in-memory
throughout the runtime of the client (no attempt is made to infer an expiration
time based on the credentials themselves).

After `expirationTimestamp`, client must execute the provider again for any new
connections. For mTLS connections, this applies even if returned certificate
is still valid (i.e. the `NotAfter` date is ignored).

`token` contains a token for use in `Authorization` header of HTTP requests.

`clientKeyData` and `clientCertificateData` contain client TLS credentials in
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
return an error if it cannot provide valid credentials.

In case client gets `401 Unauthorized` response status from remote endpoint when
using credentials from a provider, client should re-execute the provider,
ignoring `expirationTimestamp`.

### Test Plan

Unit tests to confirm:

- Version mismatch is detected
- Credentials are cached in-memory correctly
  + Executable is only called as needed
  + Expired credentials are rotated automatically
  + Credentials are used across many requests (as long as they are still valid)
- Single flight all calls to a given executable (when the config is the same)
- Reasonable timeout to executable calls so clients do not hang indefinitely

Integration (or e2e CLI) tests to confirm:

- Shared informers backed by exec credential work as expected
  + Credential rotation does not cause issues
  + Transient failures are correctly retried
  + Executables requiring interactive prompts fail gracefully
  + Executables are not called in a hot loop during transient failure
- Static forms of auth should interact correctly with exec credential plugin
  + Basic auth
  + Token based auth
  + Cert based auth
- Interactive login flows work

### Graduation Criteria

#### Beta

Feature is already in Beta.

#### Beta -> GA Graduation

- Three examples of real world usage
  + Confirm interactive and non-interactive UX is acceptable
  + Confirm no hacks are being performed to workaround limitations
  + Confirm that configuration of plugin
    * Is correctly handled
    * Is well-supported by the `kubeconfig` file format
- Create the `client.authentication.k8s.io/v1` `ExecCredential` struct
- Address known bugs and add tests to prevent regressions
- Docs are up-to-date with latest version of APIs
- Docs describe set of best practices (i.e. do not mutate `kubeconfig`)

Question: does this need conformance tests?  What would such a test look like?

### Upgrade / Downgrade Strategy

The distribution of executables to end users for use with clients is out of the
scope of this KEP.  Thus end users are responsible for confirming that the
executable they are attempting to use is compatible with `exec.apiVersion`.

### Version Skew Strategy

The client is aware of its configured `exec.apiVersion`.  It must validate that
the status response from the executable has the matching API version to prevent
it from misinterpreting the response.

## Drawbacks

End users must take care to only use trusted executables as they could easily
read sensitive data from the user's file system.

A maliciously crafted `kubeconfig` file could be used to execute arbitrary
commands on the user's file system which could lead to information disclosure,
host compromise if combined with a privilege escalation exploit, etc.

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
- 2018-02-28: Alpha (v1.10) implemented https://github.com/kubernetes/kubernetes/pull/59495
- 2018-06-04: Promoted to Beta (v1.11) https://github.com/kubernetes/kubernetes/pull/64482
