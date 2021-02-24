# KEP-2510: Request Bearer Token for Outgoing Webhook Requests

<!--
A table of contents is helpful for quickly jumping to sections of a KEP and for
highlighting any additional information provided beyond the standard KEP
template.

Ensure the TOC is wrapped with
  <code>&lt;!-- toc --&rt;&lt;!-- /toc --&rt;</code>
tags, and then generate with `hack/update-toc.sh`.
-->

<!-- toc -->
- [Release Signoff Checklist](#release-signoff-checklist)
- [Summary](#summary)
- [Motivation](#motivation)
  - [Background](#background)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Proposal](#proposal)
  - [User Stories](#user-stories)
    - [An aggreated apiserver developer can configure token authentication to webhook by setting a field in <code>MutatingWebhookConfiguration</code> or <code>ValidatingWebhookConfiguration</code>.](#an-aggreated-apiserver-developer-can-configure-token-authentication-to-webhook-by-setting-a-field-in--or-)
- [Design Details](#design-details)
  - [User-facing changes](#user-facing-changes)
  - [Internal changes](#internal-changes)
  - [Test Plan](#test-plan)
  - [Graduation Criteria](#graduation-criteria)
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

<!--
**ACTION REQUIRED:** In order to merge code into a release, there must be an
issue in [kubernetes/enhancements] referencing this KEP and targeting a release
milestone **before the [Enhancement Freeze](https://git.k8s.io/sig-release/releases)
of the targeted release**.

For enhancements that make changes to code or processes/procedures in core
Kubernetes—i.e., [kubernetes/kubernetes], we require the following Release
Signoff checklist to be completed.

Check these off as they are completed for the Release Team to track. These
checklist items _must_ be updated for the enhancement to be released.
-->

Items marked with (R) are required *prior to targeting to a milestone / release*.

- [ ] (R) Enhancement issue in release milestone, which links to KEP dir in [kubernetes/enhancements] (not the initial KEP PR)
- [ ] (R) KEP approvers have approved the KEP status as `implementable`
- [ ] (R) Design details are appropriately documented
- [ ] (R) Test plan is in place, giving consideration to SIG Architecture and SIG Testing input
- [ ] (R) Graduation criteria is in place
- [ ] (R) Production readiness review completed
- [ ] (R) Production readiness review approved
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

To add an option in WebhookClientConfig, so an apiserver's webhook client automatically requests a bearer token, and uses this token to authenticate to webhook servers. This improves developer experience by providing an out-of-box alternative to a complicated setup process.

## Motivation

### Background

Kubernetes developers can expand kube-apiserver with [aggregated apiservers](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/apiserver-aggregation/). Each aggregated apiserver can host a set of custom resources. Kubernetes developers can also configure these apiservers to use [admission webhooks](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#what-are-admission-webhooks). However, configuring aggregated apiservers to authenticate to webhooks can be a challenge. 

To configure an aggregated apiserver to authenticate to a webhook, a Kubernetes developer needs to:
1. Mint the apiserver's client certificate and key.
1. Mount the certificate and key into apiserver's container.
1. Mount webhook server's CA into apiserver's container.
1. Create a [kubeconfig file](https://kubernetes.io/docs/reference/access-authn-authz/webhook/#configuration-file-format), which specifies webhook server's URL and CA, apiserver's client and key. 
1. Mount the kubeconfig file into apiserver's container.
1. Create an [admission control configuration file](https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#authenticate-apiservers), and reference the kubeconfig files.
1. Mount the configuration file into apiserver's container.
1. Pass the kubeconfig file path into apiserver's flag `--authorization-webhook-config-file`.
1. Create a `MutatingWebhookConfiguration` or `ValidatingWebhookConfiguration` object in kube-apiserver. In the `WebhookClientConfig` child object, specify [webhook server's URL or in-cluster service reference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#webhookclientconfig-v1-admissionregistration-k8s-io).

This is a cumbersome and error-prone process to developers. This KEP intends to improve developer's experience by automatically adding a bearer token in the webhook request. This KEP stems from https://github.com/kubernetes/enhancements/pull/658, and focuses on the token request on apiserver side.

### Goals

* Allow users to easily configure apiservers to authenticate to admission webhooks with bearer token.
* Allow webhook clients to request tokens from kube-apiserver.
* The token can be used to authenticate the originated client, the originated apiserver.
* The token must be specific to an audience, the destined webhook server. 
* The token should co-exist with client certificate and key.

### Non-Goals

* Specify how the token is examined or reviewed by the webhook server.
* Specify whether the webhook server should use client cert or token for authentiation.
* Allow users to easily configure apiservers to authenticate to [audit webhook backends](AuditWebhookOptions) with bearer token. Audit backend does not have an API like WebhookClientConfig that allows more detailed config.

## Proposal

### User Stories

#### An aggreated apiserver developer can configure token authentication to webhook by setting a field in `MutatingWebhookConfiguration` or `ValidatingWebhookConfiguration`.
The developer does not need to mint the client certificate or key, and does not need to create a kubeconfig file. They can enable the automatic token option in `MutatingWebhookConfiguration` or `ValidatingWebhookConfiguration`. The corresponding webhook client will request a token and include it in the request.

## Design Details

<!--
This section should contain enough information that the specifics of your
change are understandable. This may include API specs (though not always
required) or even code snippets. If there's any ambiguity about HOW your
proposal will be implemented, this is the place to discuss them.
-->

### User-facing changes

A new field `authenticationSource` will be added to WebhookClientConfig. In this proposal, user can use `tokenRequest` source for requesting a bearer token. By doing this, users no longer have to mint token or ceritificate/key, or to create kubeconfig files.

The `authenticationSource` field is optional. If it is not set, there will be no change to webhook client behavior. This is to make sure such change is backward-compatible.

For example, this is an `ValidatingWebhookConfiguration` object created in Kubernetes. It defines an out-of-cluster webhook client that uses `tokenRequest` for adding a bearer token.
```yaml
apiVersion: admissionregistration.k8s.io/v1
kind: ValidatingWebhookConfiguration
metadata:
  name: "pod-policy.example.com"
webhooks:
- name: "pod-policy.example.com"
  rules:
  - apiGroups:   [""]
    apiVersions: ["v1"]
    operations:  ["CREATE"]
    resources:   ["pods"]
    scope:       "Namespaced"
  clientConfig:
    url: "https://my-webhook.example.com"
    caBundle: "Ci0tLS0tQk......tLS0K"
    authenticationSource:  # New addition.
      tokenRequest: {}  # New addition.
  admissionReviewVersions: ["v1", "v1beta1"]
  sideEffects: None
  timeoutSeconds: 5
```

In the case that a bearer token is already provided in the kubeconfig file, the bearer token generated in `tokenRequest` should be used instead.

For example, this is how webhook authentication is configured now, buy referring to a kubeconfig file in a `AdmissionConfiguration` plugin.
```yaml
apiVersion: apiserver.config.k8s.io/v1
kind: AdmissionConfiguration
plugins:
- name: ValidatingAdmissionWebhook
  configuration:
    apiVersion: apiserver.config.k8s.io/v1
    kind: WebhookAdmissionConfiguration
    kubeConfigFile: "<path-to-kubeconfig-file>"
```
The token defined here in a kubeconfig file will not be used, if the `tokenRequest` source is configured.
```yaml
apiVersion: v1
kind: Config
users:
- name: 'https://my-webhook.example.com'
  user:
    token: "239asdy93fs0...8sfd-0digfd===" # This token will be overridden.
```

However, if kubeconfig uses another authentication scheme that doesn't conflict with `authenticationSource`, both authentication methods will be used.

For example, the client certificate authentication defined in this kubeconfig file will be used together with `tokenRequest` source.
```yaml
apiVersion: v1
kind: Config
users:
- name: 'https://my-webhook.example.com'
  user:
    client-certificate: fake-cert-file
    client-key: fake-key-file
```

### Internal changes

Corresponding to the user-facing changes, two new structs will be added to the [webhook package](https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/util/webhook/BUILD):

```go
// AuthenticationSource specifies how a webhook client authenticates to a webhook server.
// If the AuthenticationSource conflicts with kubeconfig (e.g. bearer token in kubeconfig), the authentication source defined here will be used instead.
// If the AuthenticationSource doesn't conflicts with kubeconfig (e.g. client certificate in kubeconfig), both methods will be used.
// Only one source can be defined in AuthenticationSource.
type AuthenticationSource struct {
	TokenRequest TokenRequestAuthenticationSource
}

// TokenRequestAuthenticationSource requests a bearer token from TokenRequest API and attaches it as bearer token in webhook requests.
type TokenRequestAuthenticationSource struct {
}
```
ClientConfig struct will be updated:

```go
// ClientConfig defines parameters required for creating a hook client.
type ClientConfig struct {
	Name     string
	URL      string
	CABundle []byte
	Service  *ClientConfigService
	AnthenticationSource AuthenticationSource
}
```

A new type of `AuthenticationInfoResolver` will be added, which can request, cache, and refresh bearer token. 
```go
type tokenRequestAuthInfoResolver struct {
  // Internal implementation TBC.
}

func (*tokenRequestAuthInfoResolver) ClientConfigFor(hostPort string) (*rest.Config, error) {
  // Internal implementation TBC.
}

func (c *tokenRequestAuthInfoResolver) ClientConfigForService(serviceName, serviceNamespace string, servicePort int) (*rest.Config, error) {
  // Internal implementation TBC.
}
```

### Test Plan

An end-to-end test should include following general steps:
* Create a cluster.
* Deploy an in-cluster webhook server, which parses bearer tokens.
* Register the webhook as validation webhook for Pods with a `ValidatingWebhookConfiguration`, in which `tokenRequest` is used for `authenticationSource`.
* Create a Pod.
* Inspect the webhook server logs or metrics to see if bearer token parsing is correct.

### Graduation Criteria

TBC

### Upgrade / Downgrade Strategy

TBC

### Version Skew Strategy

This change should be compatible with different versions of other components. If the user doesn't define an `authenticationSource`, current authentication mechanism should still work.

## Production Readiness Review Questionnaire

TBC

### Feature Enablement and Rollback

TBC

###### How can this feature be enabled / disabled in a live cluster?

<!--
Pick one of these and delete the rest.
-->

- [ ] Feature gate (also fill in values in `kep.yaml`)
  - Feature gate name:
  - Components depending on the feature gate:
- [ ] Other
  - Describe the mechanism:
  - Will enabling / disabling the feature require downtime of the control
    plane?
  - Will enabling / disabling the feature require downtime or reprovisioning
    of a node? (Do not assume `Dynamic Kubelet Config` feature is enabled).

###### Does enabling the feature change any default behavior?

TBC

###### Can the feature be disabled once it has been enabled (i.e. can we roll back the enablement)?

<!--
Describe the consequences on existing workloads (e.g., if this is a runtime
feature, can it break the existing applications?).

NOTE: Also set `disable-supported` to `true` or `false` in `kep.yaml`.
-->

###### What happens if we reenable the feature if it was previously rolled back?

###### Are there any tests for feature enablement/disablement?

<!--
The e2e framework does not currently support enabling or disabling feature
gates. However, unit tests in each component dealing with managing data, created
with and without the feature, are necessary. At the very least, think about
conversion tests if API types are being modified.
-->

### Rollout, Upgrade and Rollback Planning

<!--
This section must be completed when targeting beta to a release.
-->

###### How can a rollout fail? Can it impact already running workloads?

<!--
Try to be as paranoid as possible - e.g., what if some components will restart
mid-rollout?
-->

###### What specific metrics should inform a rollback?

<!--
What signals should users be paying attention to when the feature is young
that might indicate a serious problem?
-->

###### Were upgrade and rollback tested? Was the upgrade->downgrade->upgrade path tested?

<!--
Describe manual testing that was done and the outcomes.
Longer term, we may want to require automated upgrade/rollback tests, but we
are missing a bunch of machinery and tooling and can't do that now.
-->

###### Is the rollout accompanied by any deprecations and/or removals of features, APIs, fields of API types, flags, etc.?

<!--
Even if applying deprecation policies, they may still surprise some users.
-->

### Monitoring Requirements

<!--
This section must be completed when targeting beta to a release.
-->

###### How can an operator determine if the feature is in use by workloads?

<!--
Ideally, this should be a metric. Operations against the Kubernetes API (e.g.,
checking if there are objects with field X set) may be a last resort. Avoid
logs or events for this purpose.
-->

###### What are the SLIs (Service Level Indicators) an operator can use to determine the health of the service?

<!--
Pick one more of these and delete the rest.
-->

- [ ] Metrics
  - Metric name:
  - [Optional] Aggregation method:
  - Components exposing the metric:
- [ ] Other (treat as last resort)
  - Details:

###### What are the reasonable SLOs (Service Level Objectives) for the above SLIs?

<!--
At a high level, this usually will be in the form of "high percentile of SLI
per day <= X". It's impossible to provide comprehensive guidance, but at the very
high level (needs more precise definitions) those may be things like:
  - per-day percentage of API calls finishing with 5XX errors <= 1%
  - 99% percentile over day of absolute value from (job creation time minus expected
    job creation time) for cron job <= 10%
  - 99,9% of /health requests per day finish with 200 code
-->

###### Are there any missing metrics that would be useful to have to improve observability of this feature?

<!--
Describe the metrics themselves and the reasons why they weren't added (e.g., cost,
implementation difficulties, etc.).
-->

### Dependencies

<!--
This section must be completed when targeting beta to a release.
-->

###### Does this feature depend on any specific services running in the cluster?

<!--
Think about both cluster-level services (e.g. metrics-server) as well
as node-level agents (e.g. specific version of CRI). Focus on external or
optional services that are needed. For example, if this feature depends on
a cloud provider API, or upon an external software-defined storage or network
control plane.

For each of these, fill in the following—thinking about running existing user workloads
and creating new ones, as well as about cluster-level services (e.g. DNS):
  - [Dependency name]
    - Usage description:
      - Impact of its outage on the feature:
      - Impact of its degraded performance or high-error rates on the feature:
-->

### Scalability

Consideration: latency in admission.

###### Will enabling / using this feature result in any new API calls?

<!--
Describe them, providing:
  - API call type (e.g. PATCH pods)
  - estimated throughput
  - originating component(s) (e.g. Kubelet, Feature-X-controller)
Focusing mostly on:
  - components listing and/or watching resources they didn't before
  - API calls that may be triggered by changes of some Kubernetes resources
    (e.g. update of object X triggers new updates of object Y)
  - periodic API calls to reconcile state (e.g. periodic fetching state,
    heartbeats, leader election, etc.)
-->
TokenRequest API call.

###### Will enabling / using this feature result in introducing new API types?

No

###### Will enabling / using this feature result in any new calls to the cloud provider?

No

###### Will enabling / using this feature result in increasing size or count of the existing API objects?

No, this feauture changes existing API objects.

###### Will enabling / using this feature result in increasing time taken by any operations covered by existing SLIs/SLOs?

<!--
Look at the [existing SLIs/SLOs].

Think about adding additional work or introducing new steps in between
(e.g. need to do X to start a container), etc. Please describe the details.

[existing SLIs/SLOs]: https://git.k8s.io/community/sig-scalability/slos/slos.md#kubernetes-slisslos
-->

###### Will enabling / using this feature result in non-negligible increase of resource usage (CPU, RAM, disk, IO, ...) in any components?

<!--
Things to keep in mind include: additional in-memory state, additional
non-trivial computations, excessive access to disks (including increased log
volume), significant amount of data sent and/or received over network, etc.
This through this both in small and large cases, again with respect to the
[supported limits].

[supported limits]: https://git.k8s.io/community//sig-scalability/configs-and-limits/thresholds.md
-->

### Troubleshooting

<!--
This section must be completed when targeting beta to a release.

The Troubleshooting section currently serves the `Playbook` role. We may consider
splitting it into a dedicated `Playbook` document (potentially with some monitoring
details). For now, we leave it here.
-->

###### How does this feature react if the API server and/or etcd is unavailable?

###### What are other known failure modes?

<!--
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
-->

###### What steps should be taken if SLOs are not being met to determine the problem?

## Implementation History

<!--
Major milestones in the lifecycle of a KEP should be tracked in this section.
Major milestones might include:
- the `Summary` and `Motivation` sections being merged, signaling SIG acceptance
- the `Proposal` section being merged, signaling agreement on a proposed design
- the date implementation started
- the first Kubernetes release where an initial version of the KEP was available
- the version of Kubernetes where the KEP graduated to general availability
- when the KEP was retired or superseded
-->

## Drawbacks

<!--
Why should this KEP _not_ be implemented?
-->

## Alternatives

<!--
What other approaches did you consider, and why did you rule them out? These do
not need to be as detailed as the proposal, but should include enough
information to express the idea and why it was not acceptable.
-->

## Infrastructure Needed (Optional)

<!--
Use this section if you need things from the project/SIG. Examples include a
new subproject, repos requested, or GitHub details. Listing these here allows a
SIG to get the process for these resources started right away.
-->
