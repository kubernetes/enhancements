# KEP-2369: Kubelet Sizing Providers

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [Risks and Mitigations](#risks-and-mitigations)
- [Design Details](#design-details)
  - [Sizing Provider Configuration](#sizing-provider-configuration)
  - [Sizing Provider Request API](#sizing-provider-request-api)
  - [Sizing Provider Response API](#sizing-provider-response-api)
  - [Metrics](#metrics)
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
- [Implementation History](#implementation-history)
- [Drawbacks](#drawbacks)
- [Alternatives](#alternatives)
  - [Add a built-in sizing provider in-tree.](#add-a-built-in-sizing-provider-in-tree)
    - [Pros](#pros)
    - [Cons](#cons)
- [Infrastructure Needed (Optional)](#infrastructure-needed-optional)
<!-- /toc -->

## Release Signoff Checklist

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
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

Kubelet should have a sizing provider mechanism, which could give kubelet an ability to dynamically fetch sizing values for memory and cpu reservations.

Today the sizing values are passed manually to kubelet using `--kube-reserved` and `--system-reserved` flags. Many cloud providers provide reference values for their customers to help them select optimal values based on the node sizes. e.g. [GKE](https://cloud.google.com/kubernetes-engine/docs/concepts/cluster-architecture#memory_cpu), [AKS](https://docs.microsoft.com/en-us/azure/aks/concepts-clusters-workloads#resource-reservations)

This KEP proposes an extensible plugin mechanism so that kubelet can dynamically fetch sizing values for any node size from user provided guidance irrespective of the cloud provider.

## Motivation

Kubelet’s `system reserved` and `kube reserved` play a crucial role in the OOMKilling the resource intensive pods. Without an adequate enough `system reserved` and `kube reserved` we risk freezing the node making it completely unavailable for other pods. 

We have observed that varying the value of `system reserved` and `kube reserved` with respect to the installed capacity of the node helps to deduce optimal values.

Currently, the only way to customize the `system reserved` and `kube reserved` limits is to pre-calculate the values prior to Kubelet start. If the Kubelet is deployed to various instance types, then the limits need to be tuned for every instance type manually.

### Goals

* Enable Kubelet to determine the value of the `system reserved` and `kube reserved` automatically during start up. 
* Add a plugin mechanism so that kubelet can dynamically fetch sizing values for a given node.

### Non-Goals

* For now the plugin mechanism is proposed here is only for fetching values of `system reserved` and `kube reserved`. Similar approach can be taken to dynamically fetch the values of other parameters of the kubelet (e.g. `evictionHard`) but they are out of scope of this KEP. 

## Proposal

The extension mechanism introduced in the kubelet will be done by exec-ing a plugin binary. The kubelet and the plugin communicate through stdio (stdin, stdout, and stderr) by exchanging json-serialized api-versioned types. The kubelet and the plugin must always talk the same api version to ensure compatibility as the API evolves.

### Risks and Mitigations

* exec-ing plugins for sizing values can add a slight delay in the kubelet startup.

## Design Details

The sizing provider plugin is enabled by passing two flags to the kubelet `--sizing-provider-config` and `sizing-provider-bin-dir`. The former is the path to a file containing the `SizingProviderConfig` API (more on this below) and the latter is a directory the kubelet will check for plugin binaries.

### Sizing Provider Configuration

The v1alpha1 configuration API read by the kubelet to enable exec plugins is as follows:

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SizingProviderConfig is the configuration containing information about
// each exec sizing provider. Kubelet reads this configuration from disk and enables
// each provider as specified by the SizingProvider type.
type SizingProviderConfig struct {
	metav1.TypeMeta `json:",inline"`
        // Providers is a list of sizing provider plugins that will be enabled by the kubelet.
        // Multiple providers may provide different sizing parameters (e.g. cpu for system-reserved, 
        // memory for system-reserved etc.), in which case sizing values from all providers will 
        // be returned to the kubelet. If multiple providers return overlapping values for a single 
        // kubelet parameter (e.g. cpu for system-reserved), then the value from the provider 
        // earlier in this list is used.
	Providers []SizingProvider `json:"providers"`
}

// SizingProvider represents an exec plugin to be invoked by the kubelet. The plugin is only
// invoked when `--sizing-provider-config` parameter is passed during kubelet startup.
type SizingProvider struct {
	// name is the required name of the sizing provider. It must match the name of the
	// provider executable as seen by the kubelet. The executable must be in the kubelet's
	// bin directory (set by the --image-sizing-provider-bin-dir flag).
	Name string `json:"name"`

	// Required input version of the exec SizingProviderRequest. The returned SizingProviderResponse
	// MUST use the same encoding version as the input. Supported values are:
	// - sizingprovider.kubelet.k8s.io/v1alpha1
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
// sizing plugin.
type ExecEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
```

### Sizing Provider Request API

If an exec plugin is enabled the kubelet will exec the plugin during the startup, passing the `SizingProviderRequest` API via stdin.

The kubelet will encode the request based on the apiVersion provided in SizingProviderConfig. It wil also exec the plugin based on the `args` and `env` fields in `SizingProviderConfig`.

```go
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SizingProviderRequest will be passed to the plugin via stdin. In general,
// plugins should prefer responding with the same apiVersion they were sent.
type SizingProviderRequest struct {
	metav1.TypeMeta
}
```

### Sizing Provider Response API

An exec plugin is expected to return an encoded response of the `SizingProviderResponse` API to the kubelet via stdout. 

```go

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SizingProviderResponse holds sizing values that the kubelet should use for the specified
// image provided in the original request. Kubelet will read the response from the plugin via stdout.
type SizingProviderResponse struct {
	metav1.TypeMeta
 
      // A set of ResourceName=ResourceQuantity (e.g. cpu=200m,memory=150G,pid=100) pairs
	// that describe resources reserved for non-kubernetes components.
	// Currently only cpu and memory are supported.
	// See http://kubernetes.io/docs/user-guide/compute-resources for more detail.
	SystemReserved map[string]string `json:"systemReserve"`
	// A set of ResourceName=ResourceQuantity (e.g. cpu=200m,memory=150G,pid=100) pairs
	// that describe resources reserved for kubernetes system components.
	// Currently cpu, memory and local ephemeral storage for root file system are supported.
	// See http://kubernetes.io/docs/user-guide/compute-resources for more detail.
	KubeReserved map[string]string `json:"kubeReserve"`
}

```

### Metrics

N/A

### Test Plan

Alpha:
* Unit tests for the exec plugin provider
* Unit tests for API validation

### Graduation Criteria

### Alpha

* Adequate unit testing for the plugin provider
* A working reference implementation, proving that the existing functionality of the built-in providers
can be achieved using the exec plugin.

#### Alpha -> Beta Graduation

* Integration or e2e tests.
* At least one working plugin implementation.

#### Beta -> GA Graduation

TBD

### Upgrade / Downgrade Strategy

This feature is feature gated so explicit opt-in is required on upgrade and explicit opt-out is required on downgrade.

### Version Skew Strategy

Not applicable because this feature is contained to only the kubelet and does not require communication to other components.

## Production Readiness Review Questionnaire

### Feature Enablement and Rollback

* **How can this feature be enabled / disabled in a live cluster?**
  - [ ] Feature gate (also fill in values in `kep.yaml`)
    - Feature gate name: KubeletSizingProvider
    - Components depending on the feature gate: kubelet

* **Does enabling the feature change any default behavior?**
  No, use of this feature still requires extra flags enabled on the kubelet.

* **Can the feature be disabled once it has been enabled (i.e. can we roll back
  the enablement)?**
  Yes, as long as kubelet does not specify the flags `--sizing-provider-config` and `--sizing-provider-bin-dir`.

* **What happens if we reenable the feature if it was previously rolled back?**
  Kubelet will continue to invoke exec plugins. No state is stored for this feature to function.

* **Are there any tests for feature enablement/disablement?**
  Yes.

### Rollout, Upgrade and Rollback Planning

_This section must be completed when targeting beta graduation to a release._

* **How can a rollout fail? Can it impact already running workloads?**

  TBD for beta.

* **What specific metrics should inform a rollback?**

  N/A

* **Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?**

  TBD for beta.

* **Is the rollout accompanied by any deprecations and/or removals of features, APIs,
fields of API types, flags, etc.?**

  TBD for beta.

### Monitoring Requirements

_This section must be completed when targeting beta graduation to a release._

* **How can an operator determine if the feature is in use by workloads?**

Operators can check for a kubelet config file passed into the `--image-sizing-provider-config`.

* **What are the SLIs (Service Level Indicators) an operator can use to determine
the health of the service?**
  - [X] Kubelet will fail to start if the sizing provider plugin fails with an error. An operator can look into kubelet logs to determine the failure error reported by sizing provider plugin.
 

* **What are the reasonable SLOs (Service Level Objectives) for the above SLIs?**

  If a sizing provider plugin is enabled and it fails to fetch the sizing values, the kubelet will abort the startup and log the error in kubelet logs. Since the user has explicitly requested the sizing values to be fetched using the exec plugin, kubelet should not default to any other values. Defaulting to any other values and bringing up the kubelet may result in an unpredictable behavior, such as node lock ups, as the default sizing values may be not optimal. 

  The SLO for successfully invoking sizing provider exec plugin should be based on the SLO for that plugin to fetch the sizing values.

* **Are there any missing metrics that would be useful to have to improve observability
of this feature?**

  No.

### Dependencies

_This section must be completed when targeting beta graduation to a release._

* **Does this feature depend on any specific services running in the cluster?**

This feature depends on the existence of a sizing provider plugin binary on the host and a configuration file for the plugin to be read by the kubelet.

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

  Use of the exec plugin may increase the startup time if the exec plugin invoked
by kubelet takes a long time.

* **Will enabling / using this feature result in non-negligible increase of
resource usage (CPU, RAM, disk, IO, ...) in any components?**

  Possibly, it depends on how long it takes for the exec plugin to return during kubelet startup.

## Implementation History


## Drawbacks

* exec plugins may be expensive to invoke by kubelet during the startup.
* a poorly implemented exec plugin may halt the startup for the kubelet.

## Alternatives

### Add a built-in sizing provider in-tree.
#### Pros
1. Since there is no need to execute an external binary this approach will be comparatively faster in fetching the sizing values. 
#### Cons
1. It's extremely hard to come up with an algorithm to predict optimal sizing values that could work across various deployments and cloud providers. When it comes to sizing values, there is no one size fits all solution. 

## Infrastructure Needed (Optional)

N/A