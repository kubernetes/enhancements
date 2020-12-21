# Out-of-Tree Credential Providers

## Table of Contents

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [External Credential Provider](#external-credential-provider)
  - [Example](#example)
  - [Moving Credential Providers to staging](#moving-credential-providers-to-staging)
  - [Alternatives Considered](#alternatives-considered)
    - [API Server Proxy](#api-server-proxy)
    - [Sidecar Credential Daemon](#sidecar-credential-daemon)
    - [Bound Service Account Token Flow](#bound-service-account-token-flow)
    - [Pushing Credential Management into the CRI](#pushing-credential-management-into-the-cri)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
  - [Upgrade / Downgrade Strategy](#upgrade--downgrade-strategy)
  - [Version Skew Strategy](#version-skew-strategy)
- [Implementation History](#implementation-history)
- [Infrastructure Needed](#infrastructure-needed)
<!-- /toc -->

## Release Signoff Checklist

- [x] kubernetes/enhancements issue in release milestone, which links to KEP (this should be a link to the KEP location in kubernetes/enhancements, not the initial KEP PR)
- [x] KEP approvers have set the KEP status to `implementable`
- [x] Design details are appropriately documented
- [x] Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [x] Graduation criteria is in place
- [ ] "Implementation History" section is up-to-date for milestone
- [ ] User-facing documentation has been created in [kubernetes/website], for publication to [kubernetes.io]
- [ ] Supporting documentation e.g., additional design documents, links to mailing list discussions/SIG meetings, relevant PRs/issues, release notes

## Summary

This KEP replaces the existing in-tree container image registry credential providers with an external and pluggable credential provider mechanism and removes the in-tree credential providers.

## Motivation

Kubelet uses cloud provider specific SDKs to obtain credentials when pulling container images from cloud provider specific registries. The use of cloud provider specific SDKs from within the main Kubernetes tree is deprecated by [KEP-0002](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20180530-cloud-controller-manager.md) and all existing uses need to be migrated out-of-tree. This KEP supports that migration process by removing this SDK usage.

### Goals

* Develop/test/release an interface for kubelet to obtain registry credentials from a cloud provider specific binary
* Update/test/release the credential acquisition logic within kubelet
* Build user documentation for out-of-tree credential providers
* Support migration from existing in-tree credential providers to the new credential provider interface, along with dynamic roll back.
* Remove in-tree credential provider code from Kubernetes core

### Non-Goals

* Broad removal of cloud SDK usage falls under the [KEP for removing in-tree providers](https://github.com/kubernetes/enhancements/blob/master/keps/sig-cloud-provider/20190125-removing-in-tree-providers.md).
* Continuing to support projects that import the credential provider package.

## Proposal

### External Credential Provider

An executable capable of providing container registry credentials will be pre-installed on each node so that it exists when kubelet starts running. This binary will be executed by the kubelet to obtain container registry credentials in a format compatible with container runtimes. Credential responses may be cached within the kubelet.

This architecture is similar to the approach taken by the exec based credential plugin architecture already present in client-go and CNI, and is a well understood pattern.  The API types are modeled after the ExecConfig and ExecCredential in client-go which define exec based credential retrieval for similar use cases.

A `RegistryCredentialConfig` and `RegistryCredentialProvider` configuration API type (similar to [clientauthentication](https://github.com/kubernetes/kubernetes/tree/0273d43ae9486e9d0be292c01de2dd4143522b86/staging/src/k8s.io/client-go/pkg/apis/clientauthentication/v1beta1)) will be added to Kubernetes:

```go
type RegistryCredentialConfig struct {
    metav1.TypeMeta `json:",inline"`

    Providers []RegistryCredentialProvider `json:"providers"`
}

// RegistryCredentialProvider is used by the kubelet container runtime to match the
// image property string (from the container spec) with exec-based credential provider
// plugins that provide container registry credentials.
type RegistryCredentialProvider struct {
    metav1.TypeMeta `json:",inline"`

    // The name of the plugin.  It must match the name of the binary located in the
    // search path.
    Name string `json:"name"`

    // ImageMatchers is a list of strings used to match against the image property
    // (sometimes called "registry path") to determine which images to provide
    // credentials for.  If one of the strings matches the image property, then the
    // RegistryCredentialProvider will be used by kubelet to provide credentials
    // for the image pull.

    // The image property of a container supports the same syntax as the docker
    // command does, including private registries and tags. A registry path is
    // similar to a URL, but does not contain a protocol specifier (https://).
    //
    // Each ImageMatcher string is a pattern which can optionally contain
    // a port and a path, similar to the image spec.  Globs can be used in the
    // hostname (but not the port or the path).
    //
    // Globs are supported as subdomains (*.k8s.io) or (k8s.*.io), and
    // top-level-domains (k8s.*).  Matching partial subdomains is also supported
    // (app*.k8s.io).  Each glob can only match a single subdomain segment, so
    // *.io does not match *.k8s.io.
    //
    // The image property matches when it has the same number of parts as the
    // ImageMatcher string, and each part matches.  Additionally the path of
    // ImageMatcher must be a prefix of the target URL. If the ImageMatcher
    // contains a port, then the port must match as well.
    ImageMatchers []string `json:"imageMatchers"`

    // ExtraArgs specifies arguments to be passed to the plugin via environment variables.
    // +optional
    ExtraArgs []PluginArg `json:"extraArgs"`
}

// PluginArg is used for passing extra arguments to the exec credential plugin binary.
type PluginArg struct {
    Name  string `json:"name"`
    Value string `json:"value"`
}
```

The RegistryCredentialConfig will be encoded in YAML and located in a file on disk.  The exact path of the credential provider configuration file will be passed to kubelet via a new configuration option `RegistryCredentialConfigPath`.

We will create new types `RegistryCredentialPluginRequest` and `RegistryCredentialPluginResponse` which will define the interface between the plugin and the kubelet runtime.  After the kubelet matches the image property string to a RegistryCredentialProvider, the kubelet will exec the plugin binary, providing the argument get-credentials, and pass the JSON encoded request to the plugin via stdin, which includes the image that is to be pulled.  The plugin will report back the response, which includes the credentials that kubelet needs to pull the image.

In the in-tree implementation, the docker keyring, which has N credential providers, returns an `[]AuthConfig` on a Lookup(image string) call.  This struct will be populated by the plugin rather than the in-tree provider.

```go
// RegistryCredentialPluginRequest is passed to the plugin via stdin, and includes the image that will be pulled by kubelet.
type RegistryCredentialPluginRequest struct {
    // Image is used when passed to registry credential providers as part of an
    // image pull
    Image string `json:"image"`
}

// RegistryCredentialPluginResponse holds credentials for the kubelet runtime
// to use for image pulls.  It is returned from the plugin as stdout.
type RegistryCredentialPluginResponse struct {
    metav1.TypeMeta `json:",inline"`

    // +optional
    ExpirationTimestamp *metav1.Time `json:"expirationTimestamp,omitempty"`

    Username string `json:"username"`
    Password string `json:"password"`
}

```

### Example

A registry credential provider configuration for Amazon ECR could look like the following:

```yaml
kind: credentialprovider
apiVersion: v1alpha1
providers:
-
  name: ecr-creds
  imageMatchers:
  - *.dkr.ecr.*.amazonaws.com
  - *.dkr.ecr.*.amazonaws.com.cn
  extraArgs:
    region: us-west-2
```

Where ecr-creds is a binary that vends ecr credentials.  This would execute the binary `ecr-creds` for the image `012345678910.dkr.ecr.us-east-1.amazonaws.com/my-image`.  The extra args would be passed in as environment variables, so the entire command would be:

```
REGION=us-west-2 ecr-creds get-credentials
```

### Moving Credential Providers to staging

The credential provider code is currently located in `pkg/credentialprovider`, but the API objects mentioned above must be consumed by the plugins.  To avoid forcing the plugins to consume the entire kubernetes repository as a dependency, we can use the familiar staging export method so that we can create a new github repository which will be consumed by the plugins.  This means all the credential provider code would move to `staging/src/k8s.io/credentialprovider`, and would be exported (mirrored) to github.com/kubernetes/credentialprovider.

### Alternatives Considered

#### API Server Proxy

The API server would act as a proxy to an external container registry credential provider that may support multiple cloud providers. The credential provider service will return container runtime compatible responses of the type currently used by the credential provider infrastructure within the kubelet along with credential expiration information to allow the API server to cache credential responses for a period of time.

This limits the cloud-specific privileges required for each node for the purpose of fetching credentials. Centralized caching helps to avoid cloud-specific rate limits for credential acquisition by consolidating that credential acquisition within the API server.

We chose not to follow this approach because although less privileges on each node and centralized caching are good, we have not seen enough evidence that these features are commonly requested by users.  Also, it is outside the stated goals of this KEP.  Lastly, taking the time to design such a system would probably take long enough to push back the date that we could extract the in-tree cloud providers completely from Kubernetes.

#### Sidecar Credential Daemon

Each node would run a sidecar credential daemon that can obtain cloud-specific container registry credentials and may support multiple cloud providers. This service will be available to the kubelet on the local host and will return container runtime responses compatible with those currently used by the credential provider infrastructure within kubelet. Each daemon will perform its own caching of credentials for the node on which it runs.

The added complexity of running a daemon over executing a binary made this option less desirable to us.  If a daemon implementation is necessary for a cloud provider, the binary can talk to one to retrieve credentials upon each execution.

#### Bound Service Account Token Flow

Suggested in https://github.com/kubernetes/kubernetes/issues/68810, an image pull flow built on bound service account tokens would provide kubelet with credentials to pull images for pods running as a specific service account.

This approach might be better suited as a future enhancement to either the credential provider or ImagePullSecrets, but is out of scope for extracting the cloud provider specific code.

#### Pushing Credential Management into the CRI

Another possibility is moving the credential management logic into the CRI, so that Kubelet doesn't provide any credentials for image pulls.  Similarly, this approach is also out of scope for extracting cloud provider code because it would be a more significant redesign but should be considered for a future enhancement.

### Risks and Mitigations

This is a critical feature of kubelet and pods cannot start if it does not work correctly.  This functionality will be labeled alpha and hidden behind a feature gate in v1.18.  It will use DynamicKubeletConfig so that it can be disabled during runtime if any problems occur.

## Design Details

### Test Plan

* Unit tests for image matching logic.
* E2E tests for image pulls from cloud providers.

### Graduation Criteria

Successful Alpha Criteria
* Multiple plugin implentations created.
* One E2E test implemented.

### Upgrade / Downgrade Strategy

Upgrading
* Add any cloud provider plugin binaries for image repositories that you use to your worker nodes.
* Enable this feature in kubelet with a feature flag.

Downgrading
* Disable this feature in kubelet with a feature flag.

### Version Skew Strategy

TODO

## Implementation History

TODO

## Infrastructure Needed

* New GitHub repos for existing credential providers (AWS, Azure, GCP)
