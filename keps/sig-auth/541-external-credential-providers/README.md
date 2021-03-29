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
  - [Metrics](#metrics)
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
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature enablement and rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
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
requests (no caching is done across different executions of the client).  Note
that the executable is free to perform any actions internally (i.e. it may
cache credentials on disk / external hardware, communicate with arbitrary
external APIs, perform arbitrary computations, etc).

Client is configured with a binary path, optional arguments and environment
variables to pass to it.  An optional install hint can be included to help a
user determine how to install an executable if it is missing.  A simple mechanism
is available to handle per cluster configuration.

## Design Details

### Provider configuration

Configuration is provided via the users and clusters section of `kubeconfig` file:

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

      # Text shown to the user when the executable doesn't seem to be present. Optional.
      installHint: |
        example-client-go-exec-plugin is required to authenticate
        to the current cluster.  It can be installed:

        On macOS: brew install example-client-go-exec-plugin

        On Ubuntu: apt-get install example-client-go-exec-plugin

        On Fedora: dnf install example-client-go-exec-plugin

        ...

      # Whether or not to provide cluster information, which could potentially contain
      # very large CA data, to this exec plugin as a part of the KUBERNETES_EXEC_INFO
      # environment variable. Optional. Defaults to false.
      provideClusterInfo: true
clusters:
- name: my-cluster
  cluster:
    server: "https://1.2.3.4:8080"
    certificate-authority: "/etc/kubernetes/ca.pem"
    extensions:
    - name: client.authentication.k8s.io/exec  # reserved extension name for per cluster exec config
      extension:
        some-config-per-cluster: config-data  # arbitrary config
contexts:
- name: my-cluster
  context:
    cluster: my-cluster
    user: my-user
current-context: my-cluster
```

The Go struct for the `users[...].user.exec` field:

```golang
// ExecConfig specifies a command to provide client credentials. The command is exec'd
// and outputs structured stdout holding credentials.
//
// See the client.authentiction.k8s.io API group for specifications of the exact input
// and output format
type ExecConfig struct {
  // Command to execute.
  Command string `json:"command"`
  // Arguments to pass to the command when executing it.
  // +optional
  Args []string `json:"args"`
  // Env defines additional environment variables to expose to the process. These
  // are unioned with the host's environment, as well as variables client-go uses
  // to pass argument to the plugin.
  // +optional
  Env []ExecEnvVar `json:"env"`

  // Preferred input version of the ExecInfo. The returned ExecCredentials MUST use
  // the same encoding version as the input.
  APIVersion string `json:"apiVersion,omitempty"`

  // This text is shown to the user when the executable doesn't seem to be
  // present. For example, `brew install foo-cli` might be a good InstallHint for
  // foo-cli on Mac OS systems.
  InstallHint string `json:"installHint,omitempty"`

  // ProvideClusterInfo determines whether or not to provide cluster information,
  // which could potentially contain very large CA data, to this exec plugin as a
  // part of the KUBERNETES_EXEC_INFO environment variable. By default, it is set
  // to false. Package k8s.io/client-go/tools/auth/exec provides helper methods for
  // reading this environment variable.
  ProvideClusterInfo bool `json:"provideClusterInfo"`
}
```

`apiVersion` specifies the expected version of this API that the plugin
implements. If the version differs, client must return an error.

`command` specifies the path to the provider binary. The file at this path must
be readable and executable by the client process.

`args` specifies extra arguments passed to the executable.

`env` specifies environment variables to pass to the provider. The environment
variables set in the client process are also passed to the provider.

`installHint` specifies help text to print to the user when the required binary
is missing.

`provideClusterInfo` specifies whether to provide cluster information, which could
potentially contain very large CA data, to this exec plugin as a part
of the `KUBERNETES_EXEC_INFO` environment variable.

### Provider input format

In JSON:

```json
{
  "apiVersion": "client.authentication.k8s.io/v1beta1",
  "kind": "ExecCredential",
  "spec": {
    "cluster": {
      "server": "https://1.2.3.4:8080",
      "tls-server-name": "bar",
      "insecure-skip-tls-verify": true,
      "certificate-authority-data": " ... ",
      "proxy-url": "https://4.5.6.7:9090/proxy",
      "config": { ... }
    }
  }
}
```

The Go struct:

```golang
// ExecCredential is used by exec-based plugins to communicate credentials to
// HTTP transports.
type ExecCredential struct {
  metav1.TypeMeta `json:",inline"`

  // Spec holds information passed to the plugin by the transport.
  Spec ExecCredentialSpec `json:"spec,omitempty"`

  // Status is filled in by the plugin and holds the credentials that the
  // transport should use to contact the API.
  // +optional
  Status *ExecCredentialStatus `json:"status,omitempty"`
}

// ExecCredentialSpec holds request and runtime specific information provided by
// the transport.
type ExecCredentialSpec struct {
  // Cluster contains information to allow an exec plugin to communicate with the
  // kubernetes cluster being authenticated to. Note that Cluster is non-nil only
  // when provideClusterInfo is set to true in the exec provider config (i.e.,
  // ExecConfig.ProvideClusterInfo).
  // +optional
  Cluster *Cluster `json:"cluster,omitempty"`
}

// Cluster contains information to allow an exec plugin to communicate with the
// kubernetes cluster being authenticated to.
//
// To ensure that this struct contains everything someone would need to communicate
// with a kubernetes cluster (just like they would via a kubeconfig), the fields
// should shadow "k8s.io/client-go/tools/clientcmd/api/v1".Cluster, with the exception
// of CertificateAuthority, since CA data will always be passed to the plugin as bytes.
type Cluster struct {
  // Server is the address of the kubernetes cluster (https://hostname:port).
  Server string `json:"server"`
  // TLSServerName is passed to the server for SNI and is used in the client to
  // check server certificates against. If ServerName is empty, the hostname
  // used to contact the server is used.
  // +optional
  TLSServerName string `json:"tls-server-name,omitempty"`
  // InsecureSkipTLSVerify skips the validity check for the server's certificate.
  // This will make your HTTPS connections insecure.
  // +optional
  InsecureSkipTLSVerify bool `json:"insecure-skip-tls-verify,omitempty"`
  // CAData contains PEM-encoded certificate authority certificates.
  // If empty, system roots should be used.
  // +listType=atomic
  // +optional
  CertificateAuthorityData []byte `json:"certificate-authority-data,omitempty"`
  // ProxyURL is the URL to the proxy to be used for all requests to this
  // cluster.
  // +optional
  ProxyURL string `json:"proxy-url,omitempty"`
  // Config holds additional config data that is specific to the exec
  // plugin with regards to the cluster being authenticated to.
  //
  // This data is sourced from the clientcmd Cluster object's
  // extensions[client.authentication.k8s.io/exec] field:
  //
  // clusters:
  // - name: my-cluster
  //   cluster:
  //     ...
  //     extensions:
  //     - name: client.authentication.k8s.io/exec  # reserved extension name for per cluster exec config
  //       extension:
  //         audience: 06e3fbd18de8  # arbitrary config
  //
  // In some environments, the user config may be exactly the same across many clusters
  // (i.e. call this exec plugin) minus some details that are specific to each cluster
  // such as the audience.  This field allows the per cluster config to be directly
  // specified with the cluster info.  Using this field to store secret data is not
  // recommended as one of the prime benefits of exec plugins is that no secrets need
  // to be stored directly in the kubeconfig.
  // +optional
  Config runtime.RawExtension `json:"config,omitempty"`
}
```

The Go struct for the `clusters.[...].cluster.extensions[...].extension` field:

```go
// Cluster contains information about how to communicate with a kubernetes cluster
type Cluster struct {
  // Server is the address of the kubernetes cluster (https://hostname:port).
  Server string `json:"server"`

  ... omitted for brevity ...

  // Extensions holds additional information. This is useful for extenders so
  // that reads and writes don't clobber unknown fields
  // +optional
  Extensions []NamedExtension `json:"extensions,omitempty"`
}

// NamedExtension relates nicknames to extension information
type NamedExtension struct {
  // Name is the nickname for this Extension
  Name string `json:"name"`
  // Extension holds the extension information
  Extension runtime.RawExtension `json:"extension"`
}
```

The `spec.cluster` field is the current cluster that the client is communicating
with (i.e it is the cluster the client knows it must communicate with after it
has completed parsing its `kubeconfig`, flags and environment variables).
This allows the executable to perform different actions based on the current
cluster (i.e. get a token for a particular cluster).  The `Cluster` struct is
flexible in that it not only provides all details required to communicate with
the cluster as one would via a `kubeconfig` (i.e. everything from
`"k8s.io/client-go/tools/clientcmd/api/v1".Cluster`), but also allows arbitrary
per-cluster configuration to be passed to the executable via the `config` field.
This field can contain arbitrary data that is passed to the executable without
modification.  This allows extra user-defined data (i.e. an OAuth client ID for
audience scoping) to be passed through the `spec.cluster` field.  The user
configures this via the `kubeconfig`'s `clusters.[...].cluster.extensions[client.authentication.k8s.io/exec].extension`
field.  The `client.authentication.k8s.io/exec` named extension is reserved for this
purpose.

The `spec.cluster` field is a pointer so that the plugin can easily determine whether
the cluster information is valid. The cluster information will only be provided when 1)
the `ExecCredential` version supports the `spec.cluster` field (note: `v1alpha1`
does not support this field) and 2) the `provideClusterInfo` field is set to `true`
for this `kubeconfig` `AuthInfo` entry. The `provideClusterInfo` option is opt-in for
the following reasons.
1. To prevent potentially large CA bundles from being set in the environment via the
   `Cluster.CertificateAuthorityData` field and causing system issues.
1. To design for the majority of plugins that will not use this cluster information
   (this is an assumption).
1. To give the `kubeconfig` creator, which most likely understands the runtime and
   security properties of the exec plugin, the power to enable or disable this
   cluster information being set in the environment for a plugin to consume.

To ensure that the `Cluster` struct maintains the same cluster connection
capabilities as one gets from a `kubeconfig`, the `Cluster` fields (Go and JSON) must
be named the same as `"k8s.io/client-go/tools/clientcmd/api/v1".Cluster`, with the
exception of `CertificateAuthority`, which has been left out since CA data will always
be passed to the plugin as bytes.

This data is passed to the executable via the `KUBERNETES_EXEC_INFO` environment
variable in a JSON serialized object.  Note that an environment variable is used
over passing this information via standard input because standard input is
reserved for interactive flows between the user and executable (i.e. to prompt
for username and password).

To make it easier for a plugin to use this `KUBERNETES_EXEC_INFO` environment
variable to connect to the referent cluster, a set of helper functions will be added
to create a `"k8s.io/client-go/rest".Config` from the `KUBERNETES_EXEC_INFO`
environment variable. These helper functions will live in
`k8s.io/client-go/tools/auth/exec/exec.go` so that non-plugin developers won't pull
in this new package and the new package can safely depend on
`k8s.io/client-go/rest`. The helper functions are as follows.

```golang
// LoadExecCredentialFromEnv is a helper-wrapper around LoadExecCredential that loads from the
// well-known KUBERNETES_EXEC_INFO environment variable.
//
// When the KUBERNETES_EXEC_INFO environment variable is not set or is empty, then this function
// will immediately return an error.
func LoadExecCredentialFromEnv() (runtime.Object, *rest.Config, error)

// LoadExecCredential loads the configuration needed for an exec plugin to communicate with a
// cluster.
//
// LoadExecCredential expects the provided data to be a serialized client.authentication.k8s.io
// ExecCredential object (of any version). If the provided data is invalid (i.e., it cannot be
// unmarshalled into any known client.authentication.k8s.io ExecCredential version), an error will
// be returned. A successfully unmarshalled ExecCredential will be returned as the first return
// value.
//
// If the provided data is successfully unmarshalled, but it does not contain cluster information
// (i.e., ExecCredential.Spec.Cluster == nil), then the returned rest.Config and error will be nil.
//
// Note that the returned rest.Config will use anonymous authentication, since the exec plugin has
// not returned credentials for this cluster yet.
func LoadExecCredential(data []byte) (runtime.Object, *rest.Config, error)
```

### Provider output format

In JSON:

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

The Go struct:

```golang
// ExecCredentialStatus holds credentials for the transport to use.
//
// Token and ClientKeyData are sensitive fields. This data should only be
// transmitted in-memory between client and exec plugin process. Exec plugin
// itself should at least be protected via file permissions.
type ExecCredentialStatus struct {
  // ExpirationTimestamp indicates a time when the provided credentials expire.
  // +optional
  ExpirationTimestamp *metav1.Time `json:"expirationTimestamp,omitempty"`
  // Token is a bearer token used by the client for request authentication.
  Token string `json:"token,omitempty"`
  // PEM-encoded client TLS certificates (including intermediates, if any).
  ClientCertificateData string `json:"clientCertificateData,omitempty"`
  // PEM-encoded private key for the above certificate.
  ClientKeyData string `json:"clientKeyData,omitempty"`
}
```

`expirationTimestamp` specifies the RFC3339 timestamp with credential expiry.
Client can cache provided credentials in-memory until this time.  If no
`expirationTimestamp` is provided, credentials will be cached in-memory
throughout the runtime of the client (no attempt is made to infer an expiration
time based on the credentials themselves).

After `expirationTimestamp`, client must execute the provider again for any new
connections. For mTLS connections, this applies even if the returned certificate
is still valid (i.e. the `NotAfter` date is ignored).  Existing connections can
be kept open as long as possible even if the associated credential is expired
(it is the responsibility of the server to close connections for expired
credentials).

`token` contains a token for use in `Authorization` header of HTTP requests.

`clientKeyData` and `clientCertificateData` contain client TLS credentials in
PEM format. The certificate must be valid at the time of execution. These
credentials are used for mTLS handshakes.

### Metrics

As discussed [below](#rollout-upgrade-and-rollback-planning), there are 3
primary metrics used by this feature set.

```golang
var (
  execPluginCertTTL = k8smetrics.NewGaugeFunc(
    k8smetrics.GaugeOpts{
      Name: "rest_client_exec_plugin_ttl_seconds",
      Help: "Gauge of the shortest TTL (time-to-live) of the client " +
        "certificate(s) managed by the auth exec plugin. The value " +
        "is in seconds until certificate expiry (negative if " +
        "already expired). If auth exec plugins are unused or manage no " +
        "TLS certificates, the value will be +INF.",
    },
    func() float64 {
      if execPluginCertTTLAdapter.e == nil {
        return math.Inf(1)
      }
      return execPluginCertTTLAdapter.e.Sub(time.Now()).Seconds()
    },
  )

  execPluginCertRotation = k8smetrics.NewHistogram(
    &k8smetrics.HistogramOpts{
      Name: "rest_client_exec_plugin_certificate_rotation_age",
      Help: "Histogram of the number of seconds the last auth exec " +
        "plugin client certificate lived before being rotated. " +
        "If auth exec plugin client certificates are unused, " +
        "histogram will contain no data.",
      // There are three sets of ranges these buckets intend to capture:
      //   - 10-60 minutes: captures a rotation cadence which is
      //     happening too quickly.
      //   - 4 hours - 1 month: captures an ideal rotation cadence.
      //   - 3 months - 4 years: captures a rotation cadence which is
      //     is probably too slow or much too slow.
      Buckets: []float64{
        600,     // 10 minutes
        1800,    // 30 minutes
        3600,    // 1  hour
        14400,   // 4  hours
        86400,   // 1  day
        604800,  // 1  week
        2592000,   // 1  month
        7776000,   // 3  months
        15552000,  // 6  months
        31104000,  // 1  year
        124416000, // 4  years
      },
    },
  )

  execPluginCalls = k8smetrics.NewCounterVec(
    &k8smetrics.CounterOpts{
      Name: "rest_client_exec_plugin_call_total",
      Help: "Number of calls to an exec plugin, partitioned by the type of " +
        "event encountered (no_error, plugin_execution_error, plugin_not_found_error, " +
        "client_internal_error) and an optional exit code. The exit code will " +
        "be set to 0 if and only if the plugin call was successful.",
    },
    []string{"code", "call_status"},
  )
)
```

As is common practice, these labels will be hidden behind abstract global
variables that will be called by the exec plugin code.
```golang
// DurationMetric is a measurement of some amount of time.
type DurationMetric interface {
  Observe(duration time.Duration)
}

// ExpiryMetric sets some time of expiry. If nil, assume not relevant.
type ExpiryMetric interface {
  Set(expiry *time.Time)
}

// CallsMetric counts calls that take place for a specific exec plugin.
type CallsMetric interface {
  // Increment increments a counter per exitCode and callStatus.
  Increment(exitCode int, callStatus string)
}

var (
  // ClientCertExpiry is the expiry time of a client certificate
  ClientCertExpiry ExpiryMetric = noopExpiry{}
  // ClientCertRotationAge is the age of a certificate that has just been rotated.
  ClientCertRotationAge DurationMetric = noopDuration{}
  // ExecPluginCalls is the number of calls made to an exec plugin, partitioned by
  // exit code and call status.
  ExecPluginCalls CallsMetric = noopCalls{}
)
```

The `"code"` and `"call_status"` labels of these metrics are an attempt to
elucidate the exec plugin failure mode to the user.

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

In case the client gets a `401 Unauthorized` response status from the remote
endpoint when using credentials from a provider, the client should re-execute
the provider and disregard the `expirationTimestamp`.

### Test Plan

Unit tests to confirm:

- Version mismatch is detected
- Credentials are cached in-memory correctly
  + Executable is only called as needed
  + Expired credentials are rotated automatically
  + Credentials are used across many requests (as long as they are still valid)
- Single flight all calls to a given executable (when the config is the same)
- Reasonable timeout to executable calls so clients do not hang indefinitely
- `"k8s.io/client-go/pkg/apis/clientauthentication".Cluster` (and external types)
  fields (Go and JSON) properly shadow
  `"k8s.io/client-go/tools/clientcmd/api/v1".Cluster` fields (with the exception of
  `CertificateAuthority` for reasons stated in design) so
  that structs are kept up to date
- Helper methods properly create `"k8s.io/client-go/rest".Config` from
  `"k8s.io/client-go/pkg/apis/clientauthentication".Cluster` and vice versa
- Metrics are reported as they should

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
  + TTY forwarding between client and executable works
- Metrics are reported as they should

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
- Sufficient metrics

Note: this feature set does not need conformance tests because it is inherently
opt-in on the client-side and it relies on an extra binary to be present.

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

Credential provider could be exposed as a network endpoint. Instead of executing
a binary and passing request/response over `KUBERNETES_EXEC_INFO` environment
variable/standard output, client could open a network connection and send
request/response over that.

The downsides of this approach compared to exec model are:

- if credential provider is remote, design for client authentication is
  required (aka "chicken-and-egg problem")
- credential provider must constantly run, consuming resources; clients refresh
  their credentials infrequently

## Production Readiness Review Questionnaire

### Feature enablement and rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name:
    - Components depending on the feature gate:
  - [x] Other
    - Describe the mechanism:
      - This feature is explicitly opt-in since it requires the presence of
        kubeconfig settings.
    - Will enabling / disabling the feature require downtime of the control
      plane?
        - No. Disabling the feature would result in the client needing to choose
          a different authentication method.
    - Will enabling / disabling the feature require downtime or reprovisioning
      of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).
        - No. Disabling the feature would result in the client needing to choose
          a different authentication method.

* **Does enabling the feature change any default behavior?**
  - No. The feature is explicitly opt-in, so default behavior will be preserved
    unless the client's `kubeconfig` has been updated.

* **Can the feature be disabled once it has been enabled (i.e. can we rollback
  the enablement)?**
  - Yes. Since the feature is explicitly opt-in, disabling the feature can be
    done simply by changing `kubeconfig` settings.

* **What happens if we reenable the feature if it was previously rolled back?**
  - Nothing. The feature will start respecting the explicit opt-in `kubeconfig`
    settings again, just as it would if it was enabled for the first time.

* **Are there any tests for feature enablement/disablement?**
  - There are unit tests in `k8s.io/client-go/plugin/pkg/auth/exec` that
    verify what happens when various parts of this feature set are enabled (e.g.,
    `provideClusterInfo`)
  - There are unit tests in `k8s.io/client-go/tools/clientcmd/...` that validate
    `kubeconfig`'s are handled correctly when they do not contain exec plugin
    configuration.
  - There are unit tests in `k8s.io/client-go/rest` that validate what happens when
    a REST client does not have an exec plugin configuration.

### Rollout, Upgrade and Rollback Planning

* **How can a rollout fail? Can it impact already running workloads?**
  - It is very unlikely that a rollout would fail. If you upgrade your client to
    a version that contains this exec plugin feature set, then your client would still
    continue to function as it did before, since the new behavior that this KEP provides
    is opt-in via a `kubeconfig`.
  - If a client did indeed enable the corresponding settings in its `kubeconfig` after
    rolling out this feature, then it may cause a client-side authentication failure if
    the client's exec plugin fails to return a credential properly. However, this would be
    an issue on the client side with a third-party exec plugin.

* **What specific metrics should inform a rollback?**
  - Note that `kubectl` isn't the only consumer of client-go that can make use of these
    exec plugins. Some client-go consumers are long-running and publish metrics that could
    give visibility to the health of the exec plugin and surrounding machinery.
  - When a certificate credential is refreshed (i.e., upon the first invocation of an exec
    plugin within a client's runtime, when the credential has expired, or when we get a
    401 HTTP status from the API), the certificate's expiration time will be emitted as a
    metric. The certificate expiration should remain constant until the expiration time
    when it should get increased. If this is not the case, then the exec plugin
    authenticator could be behaving incorrectly. For example, if the certificate
    expiration time is constantly increasing upon every authentication to the API, then
    perhaps the exec plugin authenticator is refreshing the certificate credential too
    often. Furthermore, the certificate's age (i.e., the time since the certificate's
    `NotBefore` field) will be emitted as a metric. If this value is frequently much smaller
    than the certificate's expected lifetime, then the exec plugin authenticator may be
    rotating credentials too quickly which may point to a bug.
  - The total number of calls to the exec plugin would also be helpful to obtain.  This
    metric should increase each time a credential is refreshed (see previous bullet point
    for when this happens). If this number increases rapidly, then the exec plugin
    authenticator could be behaving incorrectly. For example, the exec plugin could be
    receiving 401 HTTP statuses from the API, or the calculation of the expiration time
    could be incorrect, or the credential could have been incorrectly evicted from the
    exec plugin authenticator's cache.
  - The number of errors encountered when calling the exec plugin would also be helpful to
    obtain. This metric should ideally remain very low. If this number increases very
    quickly, then then one may want to inspect why the client is not able to run the exec
    plugin by viewing the client's logs or running the exec plugin manually in the target
    environment.

* **Were upgrade and rollback tested? Was upgrade->downgrade->upgrade path tested?**
  - N/A.

* **Is the rollout accompanied by any deprecations and/or removals of features,
  APIs, fields of API types, flags, etc.?**
  - Deprecation of `gcp` and `azure` authentication options. These authentication options
    can be used going forward via this exec plugin feature set.
  - Otherwise, this feature set contains the usual alpha, beta, and GA
    stages, and will follow the same canonical deprecation pattern for
    its API versions.

### Monitoring requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**
  - Clients provide metrics for usage today.
  - One could also look in the `kubeconfig` in use by the client to see if an exec
    credential provider is being used.

* **What are the SLIs (Service Level Indicators) an operator can use to
  determine the health of the service?**
  - [X] Metrics
    - Metric name: `rest_client_exec_plugin_ttl_seconds`, `rest_client_exec_plugin_certificate_rotation_age`,
      `rest_client_exec_plugin_call_total`
    - Components exposing the metric: client-go
  - [ ] Other (treat as last resort)
    - Details:
      - This feature set operates on the client-side.

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**
  - We target certificate rotations to happen within 1% of a certificate's
    lifetime. This is measured by
    `rest_client_exec_plugin_certificate_rotation_age` and
    `rest_client_exec_plugin_ttl_seconds`.
  - We target 0.01% unsuccessful calls to the exec plugin in a moving 24h
    window. This is measured by
    `rest_client_exec_plugin_call_total`.

* **Are there any missing metrics that would be useful to have to improve
  observability if this feature?**
  - As discussed [above](#rollout-upgrade-and-rollback-planning), the total number of
    calls and number of errors encountered when calling the exec plugin would make the
    behavior of this feature set more observable.

### Dependencies

* **Does this feature depend on any specific services running in the cluster?**
  - No.

## Implementation History

- 2018-01-29: Proposal submitted https://github.com/kubernetes/community/pull/1503
- 2018-02-28: Alpha (v1.10) implemented https://github.com/kubernetes/kubernetes/pull/59495
- 2018-06-04: Promoted to Beta (v1.11) https://github.com/kubernetes/kubernetes/pull/64482
