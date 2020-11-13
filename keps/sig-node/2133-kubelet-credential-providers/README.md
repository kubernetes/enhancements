# KEP-2133: Kubelet Credential Providers

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Credential Provider Configuration](#credential-provider-configuration)
  - [Credential Provider Request API](#credential-provider-request-api)
  - [Credential Provider Response API](#credential-provider-response-api)
    - [Caching Credentials](#caching-credentials)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Alpha](#alpha)
    - [Alpha -&gt; Beta Graduation](#alpha---beta-graduation)
    - [Beta -&gt; GA Graduation](#beta---ga-graduation)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Production Readiness Review Questionnaire](#production-readiness-review-questionnaire)
  - [Feature Enablement and Rollback](#feature-enablement-and-rollback)
  - [Rollout, Upgrade and Rollback Planning](#rollout-upgrade-and-rollback-planning)
  - [Monitoring Requirements](#monitoring-requirements)
  - [Dependencies](#dependencies)
  - [Scalability](#scalability)
  - [Troubleshooting](#troubleshooting)
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [X] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [X] (R) KEP approvers have approved the KEP status as `implementable`
- [X] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [X] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] Production readiness review approved
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation—e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

<!--
**Note:** This checklist is iterative and should be reviewed and updated every time this enhancement is being considered for a milestone.
-->

[kubernetes.io]: https://kubernetes.io/
[kubernetes/enhancements]: https://git.k8s.io/enhancements
[kubernetes/kubernetes]: https://git.k8s.io/kubernetes
[kubernetes/website]: https://git.k8s.io/website

## Summary

Kubelet has a credential provider mechanism, which gives kubelet the ability to dynamically
fetch credentials for image registries. Today there are three built-in implementations of the
kubelet credential provider for ACR (Azure Container Registry), ECR (Elastic Container Registry),
and GCR (Google Container Registry). This KEP proposes an extensible plugin mechanism so that
kubelet can dynamically fetch image registry credentials for any cloud provider.

## Motivation

This is part of a larger effort in the project to remove built-in functionality that favors any
specific cloud provider. The ACR, ECR and GCR implementations should be removed in favor of an
extensible plugin mechanism that can be used by any cloud provider without adding built-in logic
into the kubelet.

### Goals

* add a plugin mechanism so that kubelet can dynamically fetch credentials for a given registry.
* the plugin should have feature parity with existing functionality in-tree.

### Non-Goals

* removing the built-in ACR, ECR and GCR implementations in kubelet. This is an end-goal but not a goal of this KEP.
* improving image pull performance of kubelet.
* improving kubelet security around using image pull credentials.

## Proposal

The extension mechanism introduced in the kubelet will be done by exec-ing a plugin binary.
The kubelet and the plugin communicate through stdio (stdin, stdout, and stderr) by sending
and receiving json-serialized api-versioned types. The kubelet and the plugin must always
talk the same api version to ensure compatibility as the API evolves.

### Risks and Mitigations

* in contrast to existing built-in implementations, credentials for a image registry is now passed
through stdio of a process invoked by the kubelet, as opposed to those credentials only remaining in-memory.
* exec-ing plugins for image credentials can be expensive for the kubelet.

## Design Details

The credential provider plugin is enabled by passing two flags to the kubelet `--image-credential-provider-config` and
`image-credential-provider-bin-dir`. The former is the path to a file containing the `CredentialProviderConfig` API (more
on this below) and the latter is a directory the kubelet will check for plugin binaries.

### Credential Provider Configuration

The v1alpha1 configuration API read by the kubelet to enable exec plugins is as follows:

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CredentialProviderConfig is the configuration containing information about
// each exec credential provider. Kubelet reads this configuration from disk and enables
// each provider as specified by the CredentialProvider type.
type CredentialProviderConfig struct {
	metav1.TypeMeta `json:",inline"`

	// providers is a list of credential provider plugins that will be enabled by the kubelet.
	// Multiple providers may match against a single image, in which case credentials
	// from all providers will be returned to the kubelet. If multiple providers are called
	// for a single image, the results are combined. If providers return overlapping
	// auth keys, the value from the provider earlier in this list is used.
	Providers []CredentialProvider `json:"providers"`
}

// CredentialProvider represents an exec plugin to be invoked by the kubelet. The plugin is only
// invoked when an image being pulled matches the images handled by the plugin (see matchImages).
type CredentialProvider struct {
	// name is the required name of the credential provider. It must match the name of the
	// provider executable as seen by the kubelet. The executable must be in the kubelet's
	// bin directory (set by the --image-credential-provider-bin-dir flag).
	Name string `json:"name"`

	// matchImages is a required list of strings used to match against images in order to
	// determine if this provider should be invoked. If one of the strings matches the
	// requested image from the kubelet, the plugin will be invoked and given a chance
	// to provide credentials. Images are expected to contain the registry domain
	// and URL path.
	//
	// Each entry in matchImages is a pattern which can optionally contain a port and a path.
	// Globs can be used in the domain, but not in the port or the path. Globs are supported
	// as subdomains like '*.k8s.io' or 'k8s.*.io', and top-level-domains such as 'k8s.*'.
	// Matching partial subdomains like 'app*.k8s.io' is also supported. Each glob can only match
	// a single subdomain segment, so *.io does not match *.k8s.io.
	//
	// A match exists between an image and a matchImage when all of the below are true:
	// - Both contain the same number of domain parts and each part matches.
	// - The URL path of an imageMatch must be a prefix of the target image URL path.
	// - If the imageMatch contains a port, then the port must match in the image as well.
	//
	// Example values of matchImages:
	//   - 123456789.dkr.ecr.us-east-1.amazonaws.com
	//   - *.azurecr.io
	//   - gcr.io
	//   - *.*.registry.io
	//   - registry.io:8080/path
	MatchImages []string `json:"matchImages"`

	// defaultCacheDuration is the default duration the plugin will cache credentials in-memory
	// if a cache duration is not provided in the plugin response. This field is required.
	DefaultCacheDuration *metav1.Duration `json:"defaultCacheDuration"`

	// Required input version of the exec CredentialProviderRequest. The returned CredentialProviderResponse
	// MUST use the same encoding version as the input. Current supported values are:
	// - credentialprovider.kubelet.k8s.io/v1alpha1
	APIVersion string `json:"apiVersion"`

	// Arguments to pass to the command when executing it.
	// +optional
	Args []string `json:"args,omitempty"`

	// Env defines additional environment variables to expose to the process. These
	// are unioned with the host's environment, as well as variables client-go uses
	// to pass argument to the plugin.
	// +optional
	Env []ExecEnvVar `json:"env,omitempty"`
}

// ExecEnvVar is used for setting environment variables when executing an exec-based
// credential plugin.
type ExecEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
```

### Credential Provider Request API

If an exec plugin is enabled AND the kubelet requires authentication information for an image that matches
against a plugin, the kubelet will exec the plugin, passing the `CredentialProviderRequest` API via stdin.
The kubelet will encode the request based on the apiVersion provided in CredentialProviderConfig. It wil also
exec the plugin based on the `args` and `env` fields in `CredentialProviderConfig`.

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CredentialProviderRequest includes the image that the kubelet requires authentication for.
// Kubelet will pass this request object to the plugin via stdin. In general, plugins should
// prefer responding with the same apiVersion they were sent.
type CredentialProviderRequest struct {
	metav1.TypeMeta

	// image is the container image that is being pulled as part of the
	// credential provider plugin request. Plugins may optionally parse the image
	// to extract any information required to fetch credentials.
	Image string
}
```

### Credential Provider Response API

An exec plugin is expected to return an encoded response of the `CredentialProviderResponse` API to the kubelet
via stdout. It is required that the response is encoded with the same apiVersion as the request from stdin.
More details about caching and authentication information in the API docs below:

```go
type PluginCacheKeyType string

const (
	// ImagePluginCacheKeyType means the kubelet will cache credentials on a per-image basis,
	// using the image passed from the kubelet directly as the cache key. This includes
	// the registry domain, port (if specified), and path but does not include tags or SHAs.
	ImagePluginCacheKeyType PluginCacheKeyType = "Image"
	// RegistryPluginCacheKeyType means the kubelet will cache credentials on a per-registry basis.
	// The cache key will be based on the registry domain and port (if present) parsed from the requested image.
	RegistryPluginCacheKeyType PluginCacheKeyType = "Registry"
	// GlobalPluginCacheKeyType means the kubelet will cache credentials for all images that
	// match for a given plugin. This cache key should only be returned by plugins that do not use
	// the image input at all.
	GlobalPluginCacheKeyType PluginCacheKeyType = "Global"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CredentialProviderResponse holds credentials that the kubelet should use for the specified
// image provided in the original request. Kubelet will read the response from the plugin via stdout.
// This response should be set to the same apiVersion as CredentialProviderRequest.
type CredentialProviderResponse struct {
	metav1.TypeMeta

	// cacheKeyType indiciates the type of caching key to use based on the image provided
	// in the request. There are three valid values for the cache key type: Image, Registry, and
	// Global. If an invalid value is specified, the response will NOT be used by the kubelet.
	CacheKeyType PluginCacheKeyType

	// cacheDuration indicates the duration the provided credentials should be cached for.
	// The kubelet will use this field to set the in-memory cache duration for credentials
	// in the AuthConfig. If null, the kubelet will use defaultCacheDuration provided in
	// CredentialProviderConfig. If set to 0, the kubelet will not cache the provided AuthConfig.
	// +optional
	CacheDuration *metav1.Duration

	// auth is a map containing authentication information passed into the kubelet.
	// Each key is a match image string (more on this below). The corresponding authConfig value
	// should be valid for all images that match against this key. A plugin should set
	// this field to null if no valid credentials can be returned for the requested image.
	//
	// Each key in the map is a pattern which can optionally contain a port and a path.
	// Globs can be used in the domain, but not in the port or the path. Globs are supported
	// as subdomains like '*.k8s.io' or 'k8s.*.io', and top-level-domains such as 'k8s.*'.
	// Matching partial subdomains like 'app*.k8s.io' is also supported. Each glob can only match
	// a single subdomain segment, so *.io does not match *.k8s.io.
	//
	// The kubelet will match images against the key when all of the below are true:
	// - Both contain the same number of domain parts and each part matches.
	// - The URL path of an imageMatch must be a prefix of the target image URL path.
	// - If the imageMatch contains a port, then the port must match in the image as well.
	//
	// When multiple keys are returned, the kubelet will traverse all keys in reverse order so that:
	// - longer keys come before shorter keys with the same prefix
	// - non-wildcard keys come before wildcard keys with the same prefix.
	//
	// For any given match, the kubelet will attempt an image pull with the provided credentials,
	// stopping after the first successfully authenticated pull.
	//
	// Example keys:
	//   - 123456789.dkr.ecr.us-east-1.amazonaws.com
	//   - *.azurecr.io
	//   - gcr.io
	//   - *.*.registry.io
	//   - registry.io:8080/path
	// +optional
	Auth map[string]AuthConfig
}

// AuthConfig contains authentication information for a container registry.
// Only username/password based authentication is supported today, but more authentication
// mechanisms may be added in the future.
type AuthConfig struct {
	// username is the username used for authenticating to the container registry
	// An empty username is valid.
	Username string

	// password is the password used for authenticating to the container registry
	// An empty password is valid.
	Password string
}
```

#### Caching Credentials

The kubelet has two configuration options for determining how long credentials should be cached in-memory:
1. the `defaultCacheDuration` in `CredentialProviderConfig`
2. the `cacheDuration` in `CredentialProviderResponse`.

If a plugin specifies `cacheDuration` in the response, the kubelet will use it. If 0 the kubelet will not cache this response.
If the response did not indicate a `cacheDuration`, it will check `defaultCacheDuration` and use that. If `defaultCacheDuration`
is 0, the kubelet will not cache the response.

The plugin can signal to the kubelet how it should cache a given response. There are three codified caching key types the response can return:
1. Global: the kubelet should cache and use this response for images handled by this plugin.
2. Registry: the kubelet should cache and use this response only for future images with the same registry hostname (and port if included).
3. Image: the kubelet should cache and use this response only for future images that match the image exactly.

### Test Plan

Alpha:
* unit tests for the exec plugin provider
* unit tests for API validation

### Graduation Criteria

### Alpha

* adequate unit testing for the plugin provider
* a working reference implementation, proving that the existing functionality of the built-in providers
can be achieved using the exec plugin.

#### Alpha -> Beta Graduation

* integration or e2e tests.
* at least one working plugin implementation.
* improvements to concurrency and caching:
   - use `singleflight.Group` to ensure only a single call per image. Today the kubelet holds a single lock for every call to `Provide`.
     See [this](https://github.com/kubernetes/kubernetes/pull/94196#discussion_r517805701) and [this](https://github.com/kubernetes/kubernetes/pull/94196#discussion_r518487386) discussion.
   - clean up stale entries in kubelet's cache since cache entries only expire if fetched from cache. See [this](https://github.com/kubernetes/kubernetes/pull/94196#discussion_r520635359) discussion.

#### Beta -> GA Graduation

TBD

### Upgrade / Downgrade Strategy

This feature is feature gated so explicit opt-in is required on upgrade and explicit opt-out is required on downgrade.

### Version Skew Strategy

Not applicable because this feature is contained to only the kubelet and does not require communication
to other components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [X] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: KubeletCredentialProvider
    - Components depending on the feature gate: kubelet

* **Does enabling the feature change any default behavior?**
  No, use of this feature still requires extra flags enabled on the kubelet.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, as long as kubelet does not specify the flags `--image-credential-provider-config` and `--image-credential-provider-bin-dir`.

* **What happens if we reenable the feature if it was previously rolled back?**
  Kubelet will continue to invoke exec plugins. No state is stored for this feature to function.

* **Are there any tests for feature enablement/disablement?**
  Yes.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

TBD for beta.

* **What specific metrics should inform a rollback?**

TBD for beta.

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

TBD for beta.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

TBD for beta.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

TBD for beta.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [ ] Metrics
    - Metric name:
    - [Optional] Aggregation method:
    - Components exposing the metric:
  - [ ] Other (treat as last resort)
    - Details:

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

TBD for beta.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

TBD for beta.


### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

TBD for beta.

### Scalability

_For alpha, this section is encouraged: reviewers should consider these questions
and attempt to answer them._

_For beta, this section is required: reviewers must answer these questions._

_For GA, this section is required: approvers should be able to confirm the
previous answers based on experience in the field._

* **Will enabling / using this feature result in any new API calls?**

No

* **Will enabling / using this feature result in introducing new API types?**

It will add a new kubelet-level API. This API only contains a TypeMeta though and is
not an object.

* **Will enabling / using this feature result in any new calls to the cloud
provider?**

No, but a plugin implementation may choose to make API calls to a cloud provider.

* **Will enabling / using this feature result in increasing size or count of
the existing API objects?**

No.

* **Will enabling / using this feature result in increasing time taken by any
operations covered by [existing SLIs/SLOs]?**

Use of the exec plugin may increase image pull times if the exec plugin invoked
by kubelet takes a long time.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

Possibly, it depends on how often kubelet calls the exec plugin and what operations
the exec plugin needs to make.

### Troubleshooting

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.

_This section must be completed when targeting beta graduation to a release._

* **How does this feature react if the API server and/or etcd is unavailable?**

No.

* **What are other known failure modes?**
  For each of them, fill in the following information by copying the below template:
  - [Failure mode brief description]
    - Detection: How can it be detected via metrics? Stated another way:
      how can an operator troubleshoot without logging into a master or worker node?
    - Mitigations: What can be done to stop the bleeding, especially for already
      running user workloads?
    - Diagnostics: What are the useful log messages and their required logging
      levels that could help debug the issue?
      Not required until feature graduated to beta.
    - Testing: Are there any tests for failure mode? If not, describe why.

* **What steps should be taken if SLOs are not being met to determine the problem?**

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos

## Implementation History

- 2019-10-04: initial KEP introducing this functionality was merged.
- 2020-11-11: PR introducing this feature at alpha stage was merged https://github.com/kubernetes/kubernetes/pull/94196

## Drawbacks

* exec plugins may be expensive to invoke by kubelet.
* a poorly implemented exec plugin may halt image pulling for the kubelet.
* a poorly implemented exec plugin may leak credentials or act maliciously.

## Alternatives

1. add more built-in credential providers in-tree.
2. store credentials in a Secret that kubelet can read.

## Infrastructure Needed (Optional)

N/A
